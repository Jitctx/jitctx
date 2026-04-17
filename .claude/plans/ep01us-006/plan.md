# Plan — EP01US-006: Tree-sitter Java Integration

## Section 0 — Summary

- **Feature**: formalize and harden the Tree-sitter Java extraction that
  already powers `jitctx scan`. Six Gherkin scenarios (feature lines
  214-248) define the contract: extract class declarations with
  annotations, interface declarations with method signatures, import
  statements, generic types in signatures, **multiple** annotations on a
  single class, and partial extraction from a file containing syntax
  errors (the valid portion must still yield results and the parse tree
  must expose at least one ERROR node). The extraction is the surface
  `service.BuildModules` consumes to classify declarations per EP01RF-003.
- **Requirement IDs**: **EP01US-006**, **EP01RF-001** (Tree-sitter is the
  mandated parser — no regex fallback), **EP01RF-003** (contract extraction
  feeds on this layer). Also carries partial responsibility for
  **EP01RNF-002** (deterministic output — parser must produce the same
  `JavaFileSummary` for the same bytes on every run) and **EP01RNF-004**
  (single-binary — the Tree-sitter grammar is linked via
  `github.com/smacker/go-tree-sitter/java` with no runtime dependency; any
  `.scm` query sets must be embedded via `go:embed`).
- **Layers touched**: **domain** (one field addition on
  `JavaDeclaration`, documentation sweep on the port interface — no new
  ports or errors), **infrastructure** (`treesitter` adapter hardening:
  multi-annotation handling, generic-type preservation in signatures,
  ERROR-node surfacing, qualified-annotation names, `populate queries.go`
  placeholder), **tests** (unit coverage for every Gherkin scenario,
  integration test for the broken-file tolerance path already covered by
  `TestScanCmd_Integration_UnparseableTolerant`). Application,
  presentation, wire, and composition root are **unchanged**.
- **Tiers active**: **1, 2, 6**. Tier 3 collapsed (no change in
  `scanuc.Impl.Execute` — it already consumes `JavaFileSummary` and
  already tolerates `ErrPartialParse`). Tier 4 collapsed (no cobra /
  formatter change). Tier 5 collapsed (no wiring change — the `Parser` /
  `Walker` constructors are already called in `internal/cli/wire.go`).
- **Guidelines loaded**:
  - `.claude/guidelines/domain-layer-guidelines.yml`
  - `.claude/guidelines/infrastructure-layer-guidelines.yml`
  - `.claude/guidelines/unit-test-layer-guidelines.yml`
  - `.claude/guidelines/integration-test-layer-guidelines.yml`
- **Estimated file count**: **1 new, 5 modified** (6 files total, plus 1
  new testdata fixture for the multi-annotation + generics regression).

### What prior user stories already give us and must NOT be duplicated

Verified by reading `main` at commit `3d64613`:

| Capability | Location | Reuse status |
|---|---|---|
| `parser.ParseJavaFilePort` — `ParseJavaFile(ctx, fsys, path) (JavaFileSummary, error)` ISP port | `internal/domain/port/parser/parseJavaFilePort.go` | **Reused verbatim.** Signature covers every Gherkin scenario — no method addition, no new port file. |
| `parser.WalkJavaFilesPort` — `WalkJavaFiles(ctx, fsys) ([]string, error)` ISP port | `internal/domain/port/parser/walkJavaFilesPort.go` | **Reused verbatim.** Already scoped to `src/main/java/**`. Not in scope for this story. |
| `model.JavaFileSummary` / `model.JavaDeclaration` / `model.JavaMethod` — extraction VOs | `internal/domain/model/javaFileSummary.go` | **Partially reused.** `JavaDeclaration` gains ONE new field (`QualifiedAnnotations []string`) so that `extractAnnotations` can preserve dotted annotation names for future rules while `Annotations` stays short (backwards compatible with existing `profileClassifier.has_annotation` matching). See §2.1 for the rationale. |
| `treesitter.Parser.ParseJavaFile` — concrete Tree-sitter adapter using `sitter.NewParser` + `JavaLanguage()` with explicit top-level traversal (package / import / class / interface / enum / record) | `internal/infrastructure/treesitter/parser.go` | **Modified.** Five targeted edits: (a) multi-annotation extraction (today `extractAnnotations` walks a `modifiers` node, but only collects the *last* annotation's identifier because `findChildByTypes` returns the first identifier of each annotation child — this works for simple `@Entity` but fails `@Table(name="users")` when the `identifier` is nested under a `scoped_identifier`); (b) surface qualified annotation names (`org.springframework.stereotype.Service` → `Service` for `Annotations` **and** `org.springframework.stereotype.Service` for `QualifiedAnnotations`); (c) generics in method return types — already correct because `buildMethodSignature` matches `generic_type`, but needs a unit test pinning `Optional<User> findByEmail(String email)`; (d) in `extractSingleParam`, accept `scoped_type_identifier` so `java.util.Optional<User>` as a parameter also extracts cleanly; (e) inside `extractImport`, collect `static` prefix information but continue returning the scoped identifier — **no behaviour change unless asserted**, verified with a unit test for static imports. |
| `treesitter.Walker.WalkJavaFiles` — filesystem walker limited to `src/main/java/**` | `internal/infrastructure/treesitter/walker.go` | **Reused verbatim.** Not in scope. |
| `containsErrors(root)` recursive ERROR-node probe in `parser.go` | `internal/infrastructure/treesitter/parser.go` lines 88-98 | **Reused verbatim.** Covers Gherkin line 248 "parse tree contains at least one ERROR node" via `summary.HasErrors = true`. |
| `domerr.ErrPartialParse` sentinel (wrapped on partial parse) and `domerr.ErrParseFailure` (wrapped on hard failure) | `internal/domain/errors/errors.go` lines 17-18 | **Reused verbatim.** Covers the "partial extraction" scenario: `scanuc` already checks `errors.Is(err, domerr.ErrPartialParse)` and keeps declarations from `s.Declarations` when present. |
| `queries.go` — empty placeholder package file reserved for `.scm` query sets | `internal/infrastructure/treesitter/queries.go` | **Modified.** Currently a `package treesitter` stub with no content. This story **does NOT introduce `.scm` queries** — the existing go-tree-sitter traversal is sufficient for all six Gherkin scenarios (see §4.4 Open Question on `.scm` vs. AST-walk). The file is repurposed as the home of a small query-constants block (Tree-sitter node-type strings as named constants: `nodeClassDecl`, `nodeInterfaceDecl`, `nodeAnnotation`, `nodeMarkerAnnotation`, `nodeNormalAnnotation`, `nodeModifiers`, `nodeScopedIdentifier`, `nodeIdentifier`, `nodeMethodDecl`, `nodeFormalParameters`, `nodeGenericType`, `nodeTypeIdentifier`, `nodeScopedTypeIdentifier`, `nodeVoidType`, `nodeFormalParameter`, `nodeSpreadParameter`, `nodeVariableDeclaratorID`, `nodePackageDecl`, `nodeImportDecl`, `nodeEnumDecl`, `nodeRecordDecl`, `nodeSuperclass`, `nodeSuperInterfaces`, `nodeExtendsInterfaces`, `nodeTypeList`, `nodeInterfaceTypeList`, `nodeClassBody`, `nodeInterfaceBody`). This consolidates the magic strings scattered across `parser.go` into one place — a refactor step, not a behaviour change. See §4.1 for the rationale and the `.scm` Open Question in §8. |
| `scanuc.Impl.Execute` partial-parse handling — if `errors.Is(err, ErrPartialParse)` keep the summary when `len(s.Declarations) > 0`, else skip | `internal/application/usecase/scanuc/usecase.go` lines 102-120 | **Reused verbatim.** This is exactly what Gherkin line 247 asks for ("valid portion still yields results"). Application layer is not in the file set. |
| `TestScanCmd_Integration_UnparseableTolerant` — existing CLI-level test that drops `Broken.java` and asserts scan still completes | `internal/cli/command/scanIntegration_test.go` (verified present from EP01US-001 work) | **Reused.** Covers the partial-extraction path from the outside; this story adds a finer-grained unit assertion that `summary.HasErrors == true` for the same input. |

### In-scope behaviour (delta on top of EP01US-001 / EP01US-005)

1. **Multi-annotation extraction is locked.** Today `extractAnnotations`
   iterates children of the `modifiers` node and, for each
   `annotation | marker_annotation | normal_annotation`, calls
   `findChildByTypes(child, src, "identifier")`. For `@Entity @Table(name="users") public class User`, this produces two annotation children; the current implementation returns `["Entity"]` for a `marker_annotation` (correct) but may return a **wrong or empty string** for a `normal_annotation` whose name is a `scoped_identifier` (e.g. `@jakarta.persistence.Table(...)`) because `findChildByTypes` only checks the first-level `identifier` child. Fix: walk the annotation child's first named child and extract `type_identifier`, `identifier`, **or** the terminal segment of a `scoped_identifier`. Assert with a new unit test `TestParser_MultipleAnnotationsWithArguments`.

2. **Qualified annotation names are preserved.** Add a second slice
   `QualifiedAnnotations []string` on `JavaDeclaration`. Populated when
   the annotation name was a `scoped_identifier`
   (`org.springframework.stereotype.Service` → `Service` in `Annotations`
   **and** `org.springframework.stereotype.Service` in
   `QualifiedAnnotations`). Marker annotations that use a simple
   identifier produce only `Annotations` (the qualified slice stays the
   same length at the corresponding index, filled with the simple name).
   This is a superset of today's behaviour — no existing consumer breaks,
   and future profile rules can match on qualified names if they want to
   disambiguate `javax.persistence.Entity` from
   `jakarta.persistence.Entity`. Field is optional from the serializer's
   perspective (infrastructure mapping decides whether to emit).

3. **Generics in method return types are pinned with unit tests.** The
   current `buildMethodSignature` already matches `generic_type` and
   includes its raw text, so `Optional<User> findByEmail(String email)`
   is produced correctly today — but there is no test locking this
   behaviour. Add `TestParser_MethodReturnGeneric` and
   `TestParser_MethodParamGeneric` so a future refactor that strips
   generics (as `extractSimpleName` does for the `Implements` list)
   cannot silently regress the signature contract.

4. **Generics in method parameters.** `extractSingleParam` matches
   `generic_type` on the type side but does NOT match
   `scoped_type_identifier` (the Tree-sitter node produced for
   qualified generic type parameters like `java.util.Optional<User>`).
   Add `scoped_type_identifier` to the `case` list so the qualified case
   works. Test: `TestParser_MethodParamScopedGeneric` with
   `void apply(java.util.Optional<User> maybeUser)`.

5. **Import-statement variants.** `extractImport` currently returns the
   first `scoped_identifier` / `identifier` child of an
   `import_declaration`. This drops the `static` keyword and the
   wildcard suffix. For EP01US-006 Gherkin line 230-232, only the
   scoped name matters (the scenario asks for the `import path` itself,
   not the static-ness) — **current behaviour is correct**. Lock it with
   two new tests: `TestParser_ImportStatic` and
   `TestParser_ImportWildcard`. No production change.

6. **Partial parse surfaces ERROR node explicitly.** Today
   `summary.HasErrors = containsErrors(root)` and `ErrPartialParse` is
   wrapped when `hasErrors == true`. Gherkin line 248 asks the parse
   tree to contain at least one ERROR node — this is already how
   `containsErrors` determines `HasErrors`. Add
   `TestParser_PartialParseSurfacesErrorAndKeepsValidDeclarations`
   which:
   - feeds a Java file with a valid `public class Good { }` followed by
     an unclosed method `public void doBad(`;
   - asserts `errors.Is(err, domerr.ErrPartialParse)`;
   - asserts `summary.HasErrors == true`;
   - asserts the valid `Good` declaration is present in
     `summary.Declarations`.

   This test pins the contract on lines 247-248 end-to-end.

7. **Node-type magic strings are consolidated** into `queries.go` as
   package-private constants. Zero semantic change; keeps `parser.go`
   readable and gives future `.scm`-based migrations a single file to
   rewrite. No new public API.

### Out of scope (deferred)

- **No `.scm` query files.** The go-tree-sitter Go binding does not
  require `.scm` files for AST walking — the existing
  `parser.ParseCtx` + `root.Child(i)` traversal covers every
  EP01US-006 scenario. Introducing a `.scm`-based query layer is a
  future-epic concern (would enable e.g. `tree-sitter query
  --predicate` based matching) but adds a runtime dependency (an
  embedded query string + `sitter.NewQuery(lang, source)`) that does
  not earn its weight on the six in-scope scenarios. See §8 Q4.
- **No change to `WalkJavaFilesPort`** (walker not in scope).
- **No change to `scanuc.Impl.Execute`** — the use case already consumes
  the existing `JavaFileSummary` shape and already tolerates
  `ErrPartialParse`. Adding `QualifiedAnnotations` is additive and the
  use case does not currently read it; future stories (e.g. fully-qualified
  profile rules) will start consuming it.
- **No change to `service.ClassifyDeclaration`** — today it reads
  `JavaDeclaration.Annotations` only. It can continue to match on simple
  names after this story lands. A future story that needs qualified
  matching will flip the classifier to also read `QualifiedAnnotations`;
  no changes required for that in EP01US-006.
- **No change to the `ProjectState` / `project-state.yaml` schema.**
  `QualifiedAnnotations` is an in-memory value, not plumbed through
  the manifest DTO. The `fsmanifest` DTOs / mappers stay untouched.
- **No new infrastructure adapter**, no new port, no new use case.
- **No change to token estimation, profile loading, manifest writing,
  context discovery, or any downstream layer.**

---

## Section 1 — File Set

| # | File | Action | Layer | Tier | Group |
|---|------|--------|-------|------|-------|
| 1 | `internal/domain/model/javaFileSummary.go` | modify | domain | 1 | T1-G1 |
| 2 | `internal/infrastructure/treesitter/parser.go` | modify | infra | 2 | T2-G1 |
| 3 | `internal/infrastructure/treesitter/queries.go` | modify | infra | 2 | T2-G1 |
| 4 | `internal/infrastructure/treesitter/parser_test.go` | modify | tests | 6 | T6-G1 |
| 5 | `testdata/springBootMinimal/project/src/main/java/com/app/user_management/domain/UserWithTableAnnotation.java` | create | tests | 6 | T6-G2 |
| 6 | `internal/cli/command/scanIntegration_test.go` | modify | tests | 6 | T6-G2 |

### Requirement coverage

- **EP01US-006 / EP01RF-001 / EP01RF-003** — rows 1-6 (end-to-end). Each
  Gherkin scenario maps to at least one row:
  - Extract class + annotation (line 217-221) → rows 2, 4.
  - Extract interface + method (line 223-227) → rows 2, 4.
  - Extract import (line 229-232) → rows 2, 4.
  - Generics in signatures (line 234-237) → rows 2, 4.
  - Multiple annotations on a class (line 239-242) → rows 1, 2, 4, 5, 6.
  - Partial extraction with ERROR node (line 244-248) → rows 2, 4, 6
    (integration regression via existing `Broken.java`).
- **EP01RNF-002** (deterministic output) — rows 2, 4 (Tree-sitter traversal
  order is deterministic by node index; tests lock the expected ordering).
- **EP01RNF-004** (single binary) — no new runtime asset is introduced; row
  3 keeps the `queries.go` file a compile-time constants block, still part
  of `package treesitter`.

---

## Section 2 — Frozen Domain Contract

Everything below is the **verbatim Go shape** Tier 2 and Tier 6 consume
without modification. If implementation discovers that a signature needs
to change, stop and escalate — do not silently drift.

### 2.1 `model.JavaDeclaration` — one field addition

```go
// internal/domain/model/javaFileSummary.go
package model

// JavaFileSummary is the structured output produced by parsing a single
// Java file. UNCHANGED in this story; included verbatim for context.
type JavaFileSummary struct {
    Path         string   // forward-slash path
    Package      string   // "com.app.user.port.in"
    Imports      []string // fully-qualified types
    Declarations []JavaDeclaration
    HasErrors    bool // true if Tree-sitter reported ERROR nodes
}

// JavaDeclaration represents a top-level type declaration in a Java file.
type JavaDeclaration struct {
    NodeType              string       // "class_declaration" | "interface_declaration" | "enum_declaration" | "record_declaration"
    Name                  string       // simple name
    Annotations           []string     // simple names, no leading @ (e.g. ["Entity", "Table"])
    QualifiedAnnotations  []string     // NEW — same length and order as Annotations; entries are fully-qualified when the source wrote `@a.b.C`, else equal to the simple name
    Implements            []string     // simple and/or qualified interface names
    Extends               []string     // superclass simple or qualified name (length 0 or 1)
    Methods               []JavaMethod // class/interface-owned methods
}

// JavaMethod represents a method extracted from a Java declaration.
// UNCHANGED.
type JavaMethod struct {
    Signature string // "ReturnType name(ParamType0 name0, ParamType1 name1, ...)"; generics preserved as-written
}
```

Invariant on `QualifiedAnnotations`:

- `len(d.QualifiedAnnotations) == len(d.Annotations)` in every case
  where `Annotations` is populated. When the source spelled
  `@Entity`, both slices carry `"Entity"` at the same index. When the
  source spelled `@jakarta.persistence.Entity`, `Annotations[i] ==
  "Entity"` and `QualifiedAnnotations[i] ==
  "jakarta.persistence.Entity"`.
- The new slice is additive; no existing consumer reads it today, so
  zero downstream code changes.
- Zero value (nil slice) is legal when `Annotations` is also nil.

### 2.2 Ports — no interface changes

No new port files. No method added to `ParseJavaFilePort` or
`WalkJavaFilesPort`. Signatures for reference:

```go
// internal/domain/port/parser/parseJavaFilePort.go — UNCHANGED
package parser

import (
    "context"
    "io/fs"

    "github.com/jitctx/jitctx/internal/domain/model"
)

// ParseJavaFilePort parses one Java file and returns a structured summary.
type ParseJavaFilePort interface {
    // ParseJavaFile parses one Java file and returns a structured summary.
    // Syntactic errors produce a non-nil summary AND a wrapped ErrPartialParse
    // that the caller may choose to skip (errors.Is(err, ErrPartialParse)).
    ParseJavaFile(ctx context.Context, fsys fs.FS, path string) (model.JavaFileSummary, error)
}

// internal/domain/port/parser/walkJavaFilesPort.go — UNCHANGED
type WalkJavaFilesPort interface {
    WalkJavaFiles(ctx context.Context, fsys fs.FS) ([]string, error)
}
```

### 2.3 Use-case signatures — no changes

```go
// internal/domain/usecase/scanuc/port.go — UNCHANGED
type UseCase interface {
    Execute(ctx context.Context, input scanvo.ScanProjectInput) (scanvo.ScanProjectOutput, error)
}
```

`scanvo.ScanProjectInput` / `ScanProjectOutput` unchanged. The new
`QualifiedAnnotations` field is not surfaced in `ScanProjectOutput`;
it lives only inside the in-memory `[]JavaFileSummary` that
`scanuc.Impl.Execute` forwards to `service.BuildModules`.

### 2.4 Errors — no new sentinels

`domerr.ErrParseFailure` and `domerr.ErrPartialParse` already cover
every failure mode EP01US-006 describes. No new sentinel, no new typed
error. `internal/domain/errors/errors.go` is **not** in the file set.

### 2.5 `Deps` struct — unchanged

```go
// internal/cli/wire.go — UNCHANGED; included here for completeness
type Deps struct {
    ScanFactory command.ScanUseCaseFactory
    Query       queryuc.UseCase
    Plan        planuc.UseCase
    Contracts   contractsuc.UseCase
    Logger      *slog.Logger
}
```

### 2.6 Parser adapter signature (frozen)

```go
// internal/infrastructure/treesitter/parser.go — SIGNATURE UNCHANGED
type Parser struct{}

func New() *Parser

// Implements parser.ParseJavaFilePort.
func (p *Parser) ParseJavaFile(
    ctx context.Context, fsys fs.FS, path string,
) (model.JavaFileSummary, error)
```

Contract locked by this story:

1. Returns `(JavaFileSummary{}, ctx.Err())` on a pre-cancelled context.
2. Returns `(JavaFileSummary{}, fmt.Errorf("read file %s: %w", path, domerr.ErrParseFailure))` on `fs.ReadFile` failure.
3. Returns `(JavaFileSummary{}, fmt.Errorf("parse tree error for %s: %w", path, domerr.ErrParseFailure))` on `sitter.Parser.ParseCtx` failure or nil tree.
4. On successful parse:
   - Populates `Path`, `Package`, `Imports`, `Declarations`, `HasErrors`.
   - Each `JavaDeclaration` carries `NodeType`, `Name`, `Annotations`,
     `QualifiedAnnotations`, `Implements`, `Extends`, `Methods`.
   - Ordering is **file order** (top-to-bottom). The current code walks
     `root.Child(i)` from `i=0` to `i=ChildCount()-1`; this is
     deterministic and preserved.
5. If `containsErrors(root) == true`, `HasErrors = true` and returns
   `(summary, fmt.Errorf("partial parse %s: %w", path, domerr.ErrPartialParse))` — summary still populated with whatever could be extracted from the valid portions.

### 2.7 Annotation-name extraction contract (frozen)

For every `annotation | marker_annotation | normal_annotation` child
under a `modifiers` node, the extractor:

1. Finds the **name-bearing child** of the annotation node. Acceptable
   types, in priority order: `type_identifier`, `identifier`,
   `scoped_identifier`.
2. If the name-bearing child is a `scoped_identifier`, its full text
   (e.g. `jakarta.persistence.Table`) is appended to
   `QualifiedAnnotations` and the terminal `.`-separated segment
   (`Table`) is appended to `Annotations`.
3. Otherwise, the child's text is appended to both slices.
4. Annotations whose name cannot be extracted (malformed AST) are
   **silently skipped** — they do not produce `""` entries in either
   slice. The two slices stay length-equal at all times.

### 2.8 Method-signature contract (frozen)

`buildMethodSignature(node, src)` returns:

```
"<ReturnType> <name>(<ParamType0> <name0>, <ParamType1> <name1>, ...)"
```

Rules:

- `ReturnType` is the raw source text of whichever return-type child
  appeared first. Accepted node types: `void_type`, `type_identifier`,
  `integral_type`, `floating_point_type`, `boolean_type`, `array_type`,
  `generic_type`, `scoped_type_identifier` (NEW in this story).
- `name` is the first `identifier` child of the `method_declaration`
  node (never the method-body scope's identifiers).
- `(params)` is produced by `extractFormalParams`, which joins per-param
  `ExtractSingleParam` results with `", "`. Each per-param entry is
  `"<Type> <name>"` or just `"<Type>"` when the variable-declarator-id
  could not be resolved. For `spread_parameter` (`String...`), the raw
  source text is used.
- Empty return type OR empty name ⇒ `buildMethodSignature` returns
  `""` and the method is NOT added to `decl.Methods`. Unchanged.

Generics are preserved verbatim — `Optional<User>` stays
`Optional<User>`, `java.util.Optional<User>` stays
`java.util.Optional<User>`. `extractSimpleName` (which strips generics)
is used ONLY for the `Implements` / `Extends` lists where simple
matching semantics matter, never for method signatures.

---

## Section 3 — Domain Layer Plan

**Tier 1 — one group only (mandated: domain is always one group).**

### 3.1 `internal/domain/model/javaFileSummary.go` — modify

- Add ONE exported field `QualifiedAnnotations []string` to
  `JavaDeclaration`, placed immediately after `Annotations`. Keep
  godoc style consistent with existing fields.
- No new type, no constructor, no validation method, no change to
  `JavaFileSummary` or `JavaMethod`.
- No new imports.
- No change to `internal/domain/port/parser/*.go`.
- No change to `internal/domain/errors/errors.go`.
- No change to `internal/domain/service/*.go` — `ClassifyDeclaration`
  does not read `QualifiedAnnotations` today and this story does not
  teach it to.

---

## Section 4 — Infrastructure Layer Plan

**Tier 2 — one group** (`T2-G1`). Both infra files live under
`internal/infrastructure/treesitter/` — per the Step-6 rule "one group
per `infrastructure/{collaborator}/` subdirectory". `parser.go` and
`queries.go` are edited as a unit.

### 4.1 `internal/infrastructure/treesitter/queries.go` — modify

Today: `package treesitter` with an empty body. Replace with a block of
package-private constants naming every Tree-sitter Java node type
consumed by `parser.go`. No new imports. Approximate shape (final exact
list must mirror what `parser.go` actually references — this is a
refactor step, not a behaviour change):

```go
// internal/infrastructure/treesitter/queries.go
package treesitter

// Tree-sitter Java grammar node type names consumed by parser.go.
// Centralized here so a future migration to a .scm-based query layer
// has a single seam to replace.
const (
    nodePackageDecl          = "package_declaration"
    nodeImportDecl           = "import_declaration"
    nodeClassDecl            = "class_declaration"
    nodeInterfaceDecl        = "interface_declaration"
    nodeEnumDecl             = "enum_declaration"
    nodeRecordDecl           = "record_declaration"

    nodeModifiers            = "modifiers"
    nodeAnnotation           = "annotation"
    nodeMarkerAnnotation     = "marker_annotation"
    nodeNormalAnnotation     = "normal_annotation"

    nodeIdentifier           = "identifier"
    nodeTypeIdentifier       = "type_identifier"
    nodeScopedIdentifier     = "scoped_identifier"
    nodeScopedTypeIdentifier = "scoped_type_identifier"

    nodeSuperclass           = "superclass"
    nodeSuperInterfaces      = "super_interfaces"
    nodeExtendsInterfaces    = "extends_interfaces"
    nodeTypeList             = "type_list"
    nodeInterfaceTypeList    = "interface_type_list"
    nodeClassBody            = "class_body"
    nodeInterfaceBody        = "interface_body"

    nodeMethodDecl           = "method_declaration"
    nodeFormalParameters     = "formal_parameters"
    nodeFormalParameter      = "formal_parameter"
    nodeSpreadParameter      = "spread_parameter"
    nodeVariableDeclaratorID = "variable_declarator_id"

    nodeVoidType             = "void_type"
    nodeIntegralType         = "integral_type"
    nodeFloatingPointType    = "floating_point_type"
    nodeBooleanType          = "boolean_type"
    nodeArrayType            = "array_type"
    nodeGenericType          = "generic_type"
)
```

These constants replace the string literals inline in `parser.go` but
do NOT change any matching logic.

### 4.2 `internal/infrastructure/treesitter/parser.go` — modify

Targeted edits (each maps to a Gherkin scenario):

1. **`extractAnnotations` — multi-annotation + scoped names.** Replace
   the body so it handles `scoped_identifier` as the name-bearing child
   of a `normal_annotation` or `annotation` node, and populates the
   new `QualifiedAnnotations` slice at the same index as `Annotations`.
   Pseudocode:

   ```go
   func extractAnnotations(node *sitter.Node, src []byte) (simple, qualified []string) {
       if node == nil { return nil, nil }
       for i := 0; i < int(node.ChildCount()); i++ {
           child := node.Child(i)
           switch child.Type() {
           case nodeAnnotation, nodeMarkerAnnotation, nodeNormalAnnotation:
               raw := findAnnotationNameChild(child, src)
               if raw == "" { continue } // malformed — skip, keep invariant len(simple) == len(qualified)
               simple = append(simple, terminalSegment(raw))
               qualified = append(qualified, raw)
           }
       }
       return
   }

   // findAnnotationNameChild returns the source text of the first
   // type_identifier | identifier | scoped_identifier child of an
   // annotation node, or "" when none found.
   func findAnnotationNameChild(ann *sitter.Node, src []byte) string {
       for j := 0; j < int(ann.ChildCount()); j++ {
           c := ann.Child(j)
           switch c.Type() {
           case nodeTypeIdentifier, nodeIdentifier, nodeScopedIdentifier:
               return nodeText(c, src)
           }
       }
       return ""
   }

   // terminalSegment returns the trailing ".xxx" segment of a dotted
   // name, or the name itself if no dot is present.
   func terminalSegment(name string) string {
       if idx := strings.LastIndex(name, "."); idx >= 0 {
           return name[idx+1:]
       }
       return name
   }
   ```

   Call sites in `extractClassDeclaration`,
   `extractInterfaceDeclaration`, `extractEnumDeclaration`,
   `extractRecordDeclaration` are updated to bind both return values
   into `decl.Annotations` and `decl.QualifiedAnnotations` respectively.

2. **`extractSingleParam` — accept `scoped_type_identifier`.** Extend
   the `case` list on the type side:

   ```go
   case nodeTypeIdentifier, nodeIntegralType, nodeFloatingPointType,
        nodeBooleanType, nodeArrayType, nodeGenericType, nodeVoidType,
        nodeScopedTypeIdentifier: // NEW
       paramType = nodeText(child, src)
   ```

   Same addition in `buildMethodSignature` for the return-type side.

3. **Replace every node-type string literal with its constant from
   `queries.go`.** Pure refactor — no behavioural change. Runs alongside
   edits 1 and 2 under the same group to avoid merge conflicts.

4. **Guard rails kept unchanged**: `containsErrors` recursion,
   `ctx.Err()` entry check, `defer tree.Close()`, partial-parse error
   wrapping, `ErrPartialParse` / `ErrParseFailure` semantics, the
   `domerr`-wrapped `fmt.Errorf` messages. None of these change.

5. **No new exported symbols.** `Parser` is already the sole exported
   type; `New`, `JavaLanguage` (in `languages.go`), and `NewWalker` (in
   `walker.go`) continue to be the only exported constructors.

6. **No new imports in `parser.go`.** `strings` is already imported
   (used by `extractSimpleName`). The new `terminalSegment` helper
   reuses `strings.LastIndex`.

### 4.3 DTO / atomic-write / go:embed checkpoints

- The Tree-sitter adapter does not write files; atomic-rename patterns
  do not apply.
- `go:embed` not used in this story — the Tree-sitter grammar is linked
  statically via `github.com/smacker/go-tree-sitter/java`, and no `.scm`
  file is introduced (see §8 Q4). `EP01RNF-004` (single binary) stays
  satisfied by construction.
- `queries.go` stays compile-time constants — not an embedded asset.

### 4.4 `.scm` query updates — explicit decision

**No `.scm` files are added or modified in this story.** The rationale,
spelled out so future maintainers know the call was deliberate:

1. The go-tree-sitter API (`sitter.NewParser`, `root.Child(i)`,
   `node.Type()`) is sufficient to implement every Gherkin scenario on
   feature lines 217-248.
2. Introducing a `.scm` file would require: (a) `//go:embed
   queries/java.scm` with a fresh query string, (b) `sitter.NewQuery`
   at startup, (c) a new execution path alongside the existing AST
   walk. The added complexity is not justified by any acceptance
   criterion on EP01US-006.
3. When a future epic needs `.scm` (e.g. for cross-cutting queries
   such as "find every class whose superclass ends in `Repository`"),
   `queries.go` becomes the natural home: its constants get replaced
   with embedded `.scm` strings, and the AST walk in `parser.go`
   switches to `sitter.NewQueryCursor().Exec(query, root)`. The seam
   is pre-placed by this story's refactor.

This is explicitly listed as `Blocking: No` in §8 so the planner's
self-validation does not stall on the absence of `.scm` work.

---

## Section 5 — Application Layer Plan

**Tier 3 — N/A.** `scanuc.Impl.Execute` already consumes
`JavaFileSummary` by value, forwards the whole slice to
`service.BuildModules`, and already tolerates `ErrPartialParse`. No
change is required for EP01US-006. The new `QualifiedAnnotations`
field is additive and unused today.

---

## Section 6 — Presentation Layer Plan

**Tier 4 — N/A.** No cobra command change, no formatter change, no
flag addition, no stdout / stderr shape change.

### 6.1 stdout / stderr contract

- `stdout`: unchanged (the scan report from `format.WriteScanReport`).
- `stderr` (slog): unchanged beyond what EP01US-005 already established.
  The parser emits no log line of its own — it returns errors / summaries
  to `scanuc`, which is the one that decides whether to `Warn` on
  `ErrPartialParse`.

---

## Section 7 — Composition Root + Tests Plan

### 7.1 Composition root — N/A

**Tier 5 not used.** `internal/cli/wire.go`, `internal/cli/root.go`,
`internal/cli/execute.go`, `cmd/jitctx/main.go`, and
`internal/config/**` are not in the file set. `wire.go` already builds
`treesitter.New()` and `treesitter.NewWalker()` and injects them into
`scanuc.New`; no re-wiring required.

### 7.2 Unit tests

**Tier 6 — group T6-G1.** One file, added as a unit-test pillar:
`internal/infrastructure/treesitter/parser_test.go`. The existing five
tests stay (`TestParser_ClassWithAnnotation`,
`TestParser_InterfaceWithMethods`, `TestParser_Imports`,
`TestParser_PartialParse`, `TestParser_ImplementsInterface`). Add the
following tests so every Gherkin scenario on feature lines 217-248 has
an explicit assertion.

#### T6-G1 — new tests in `parser_test.go`

1. **`TestParser_MultipleAnnotationsWithArguments`** — pins feature
   lines 239-242.

   ```java
   package com.app.user.domain;

   import jakarta.persistence.Entity;
   import jakarta.persistence.Table;

   @Entity
   @Table(name = "users")
   public class User {}
   ```

   Assertions:
   - `require.NoError(t, err)`.
   - `require.Len(t, summary.Declarations, 1)`.
   - `decl := summary.Declarations[0]`.
   - `require.Equal(t, []string{"Entity", "Table"}, decl.Annotations)`.
   - `require.Equal(t, []string{"Entity", "Table"},
     decl.QualifiedAnnotations)` (both written as simple names).

2. **`TestParser_QualifiedAnnotation`** — pins §2.7 qualified-name
   contract.

   ```java
   package com.app.user.domain;

   @jakarta.persistence.Entity
   public class User {}
   ```

   Assertions:
   - `decl.Annotations == ["Entity"]`.
   - `decl.QualifiedAnnotations == ["jakarta.persistence.Entity"]`.

3. **`TestParser_MethodReturnGeneric`** — pins feature lines 234-237.

   ```java
   package com.app.user.port.out;

   import java.util.Optional;

   public interface UserRepository {
       Optional<User> findByEmail(String email);
   }
   ```

   Assertions:
   - `decl := summary.Declarations[0]`.
   - `require.Len(t, decl.Methods, 1)`.
   - `require.Equal(t, "Optional<User> findByEmail(String email)",
     decl.Methods[0].Signature)`.

4. **`TestParser_MethodParamGeneric`** — complements scenario 4.

   ```java
   package com.app.user.port.in;

   public interface CreateBatch {
       void apply(java.util.List<String> names);
   }
   ```

   Assertions:
   - `decl.Methods[0].Signature == "void apply(java.util.List<String> names)"`.
   - Proves the `scoped_type_identifier` case in `extractSingleParam`.

5. **`TestParser_ImportStatic`** — locks the current behaviour on
   `import static`. Feature line 229-232 does not mention static
   imports, but the existing test suite does not pin this case and
   static imports are common in Spring code.

   ```java
   package com.app.util;

   import static java.util.Collections.emptyList;
   import com.app.notification.port.in.SendNotificationUseCase;

   public class Helper {}
   ```

   Assertions:
   - `require.Contains(t, summary.Imports,
     "com.app.notification.port.in.SendNotificationUseCase")`.
   - `require.Contains(t, summary.Imports,
     "java.util.Collections.emptyList")` — the current extractor
     returns the first `scoped_identifier`, which for a static import
     includes the member name.

6. **`TestParser_ImportWildcard`** — locks the wildcard behaviour.

   ```java
   package com.app.util;

   import java.util.*;

   public class Helper {}
   ```

   Assertions:
   - `require.Contains(t, summary.Imports, "java.util")` — matches
     today's behaviour (the `*` is not part of any
     `scoped_identifier`).

7. **`TestParser_PartialParseSurfacesErrorAndKeepsValidDeclarations`** —
   pins feature lines 244-248 at the unit level. (The existing
   `TestParser_PartialParse` asserts the error and `HasErrors` but does
   not verify that the valid declarations before the error are kept.)

   ```java
   package com.app.mixed;

   public class Good {
       public void doGood() { }
   }

   public class Bad {
       public void doBad(
   ```

   Assertions:
   - `require.True(t, errors.Is(err, domerr.ErrPartialParse))`.
   - `require.True(t, summary.HasErrors)`.
   - `require.NotEmpty(t, summary.Declarations)` — at minimum, `Good`
     with `NodeType == "class_declaration"` and `Name == "Good"` must
     be present.

All tests use `t.Parallel()`, `fstest.MapFS`, and the existing
`treesitter.New()` constructor. No new helpers, no `testdata/`
dependencies for unit tests.

### 7.3 Integration tests

**Tier 6 — group T6-G2.** One file change plus one new testdata fixture.

#### T6-G2 — `internal/cli/command/scanIntegration_test.go` (modify)

Add one test that exercises multi-annotation extraction end-to-end
(scan → classify → manifest) so the unit-level fix actually surfaces in
the `project-state.yaml`:

1. **`TestScanCmd_Integration_MultiAnnotationClass`** — drops a new
   Java file (`UserWithTableAnnotation.java`, see step 2 below) into a
   copy of the `springBootMinimal` fixture, runs `jitctx scan`, reads
   the emitted `project-state.yaml`, and asserts:
   - The `user-management` module contains a contract named
     `UserWithTableAnnotation`.
   - Its `type` is `entity` (because the bundled profile classifies on
     `has_annotation: Entity` — which must still match when the class
     is also annotated with `@Table(...)`).
   - A debug/info log line containing both annotation names is emitted
     (optional assertion — only if the log buffer contains it; the
     existing `scanuc` logs declarations at `Debug` level, so this is
     a soft check).

   Reuse the existing `copyFixture` / `buildScanFactoryWithLogger` /
   `discardLogger` helpers from `helpers_test.go`. `t.Parallel()` and
   `t.TempDir()`.

**Existing integration test preservation:**

- `TestScanCmd_Integration_UnparseableTolerant` (existing, covers
  `Broken.java`) stays green — this story does not change the scan's
  tolerance policy; it only strengthens what the parser extracts. The
  test already feeds `Broken.java` (which is in the fixture tree
  verified at §6 above) and asserts the scan completes with the broken
  file skipped. This is the CLI-side proof of feature lines 244-248.
- `TestScanCmd_Integration_HappyPath` stays green because
  `expected/project-state.yaml` is NOT modified — the new fixture file
  is introduced through the new `_MultiAnnotationClass` test, which
  uses a *separate* `t.TempDir()` copy of the fixture (not the canonical
  `expected/` tree).

#### Testdata fixture

2. **Create
   `testdata/springBootMinimal/project/src/main/java/com/app/user_management/domain/UserWithTableAnnotation.java`.**

   ```java
   package com.app.user_management.domain;

   @Entity
   @Table(name = "users")
   public class UserWithTableAnnotation {
       private Long id;
       private String email;
   }
   ```

   - Package path mirrors the existing `User.java` sibling so the
     module catalog detects it under the same `user-management`
     module.
   - No import lines needed — `Entity` and `Table` annotations resolve
     at parse time to simple names, and Tree-sitter does not care
     about unresolved imports (classification operates on
     `Annotations[]`, not on the import graph).
   - File is NOT added to the canonical `expected/project-state.yaml`
     — the new integration test uses its own copy of the fixture, not
     the golden file. This preserves `TestScanCmd_Integration_HappyPath`.

### 7.4 What is NOT added

- No new unit test in `internal/domain/service/profileClassifier_test.go` —
  the classifier still reads `Annotations[]` only; the new
  `QualifiedAnnotations` slice is not yet consumed. A future story
  that adds qualified-match rules will extend that test suite then.
- No new test in `internal/infrastructure/treesitter/walker_test.go` —
  `Walker` is not in scope.
- No `testscript`-based integration — every scenario is satisfied by
  the in-process tests.

### 7.5 Determinism sanity

The Tree-sitter parser traverses `root.Child(i)` from `0` to
`ChildCount()-1`. This order is deterministic by AST node index and
does not change between runs on the same input bytes. All new unit
tests assert specific slice contents and positions, which would fail
loudly if the ordering changed. EP01RNF-002 remains satisfied at the
parser level.

---

## Section 8 — Open Questions & Risks

| # | Question / Risk | Blocking? | Resolution taken in plan |
|---|-----------------|-----------|--------------------------|
| 1 | Should `QualifiedAnnotations` be added as a field on `JavaDeclaration` or as a separate map `map[string]string` (simple → qualified)? | No | Use a parallel slice. Rationale: slice preserves ordering, is cheaper to allocate, and matches the existing `Annotations []string` shape. Invariant `len == len` is stated explicitly in §2.1 and is checked by the new unit tests. Pattern mirrors `model.JavaDeclaration.Implements` / `Extends` (both flat slices). |
| 2 | Should `extractAnnotations` also resolve the annotation's fully-qualified name by looking up the corresponding `import` in the same file? | No | No — the parser is a syntactic layer, not a semantic one. `QualifiedAnnotations` carries ONLY what the source spelled literally. Semantic resolution (matching `@Entity` against `import jakarta.persistence.Entity;`) is a future-epic concern. |
| 3 | The Gherkin on lines 240-242 says `annotations = ["Entity", "Table"]` — should the classifier match when both annotations are present, or is single-annotation matching still enough? | No | Single-annotation matching is enough — the bundled profile has a rule `has_annotation: Entity` → `classify_as: entity` which wins on the first matching rule. The `@Table(...)` annotation is extracted (goal of this story) but does not participate in classification today. A future story can add a `has_annotations: [Entity, Table]` variant to the profile schema if needed. |
| 4 | Should this story introduce `.scm` Tree-sitter query files? | No | No. See §4.4 for the rationale — the AST-walk implementation is sufficient for every Gherkin scenario, and adding `.scm` adds complexity without acceptance-criterion justification. `queries.go` becomes the seam where that migration would happen in a future story. |
| 5 | Feature line 244 says "valid class ‘User’ followed by an unclosed method" but the current `Broken.java` fixture has an unclosed method on a class **without** a preceding fully-valid class. Does the existing fixture satisfy scenario 6? | No | The existing fixture satisfies half the scenario (the "parse tree contains ERROR node" half). The "valid portion still yields results" half is satisfied by the new unit test `TestParser_PartialParseSurfacesErrorAndKeepsValidDeclarations` (§7.2 test 7), which uses a purpose-built in-memory fixture with one complete class plus one broken class. No change to `Broken.java` is needed. |
| 6 | Should `QualifiedAnnotations` be surfaced in `project-state.yaml`? | No | No. The manifest DTO stays untouched. `QualifiedAnnotations` is an in-memory value consumed by future profile rules; surfacing it to disk would churn the manifest schema without a consumer. |
| 7 | The new integration test `TestScanCmd_Integration_MultiAnnotationClass` drops a new Java file into the fixture — does this break `TestScanCmd_Integration_HappyPath` which compares against a frozen `expected/project-state.yaml`? | No | No. The new test uses `copyFixture` to stage the tree into `t.TempDir()` and adds the new file there; the canonical `expected/project-state.yaml` is untouched (see §7.3 step 2). |
| 8 | `extractImport` currently returns the raw scoped identifier which for `import static java.util.Collections.emptyList;` becomes `"java.util.Collections.emptyList"`. Is that the right contract? | No | Yes — consistent with the existing EP01US-001 behaviour and not contradicted by EP01US-006 Gherkin. Pinned with `TestParser_ImportStatic` so a future change must be deliberate. |
| 9 | The existing `TestParser_PartialParse` fixture is a class with an unclosed method body — does the recursive `containsErrors` correctly flag this as `HasErrors`? | No | Yes — verified by reading the existing test (lines 82-98 of `parser_test.go`). The recursion walks down into the `method_declaration` subtree and finds a MISSING `}` node, which `sitter.Node.IsMissing()` reports. The new unit test `TestParser_PartialParseSurfacesErrorAndKeepsValidDeclarations` strengthens this with a mixed valid/invalid input. |

**No blocking questions. Proceed to implementation.**

---

## Section 9 — Parallel Execution Plan (authoritative for @agent-manager)

```yaml
tiers:
  - id: 1
    name: Domain contract
    depends_on: []
    groups:
      - id: T1-G1
        scope:
          create: []
          modify:
            - internal/domain/model/javaFileSummary.go
        guidelines:
          - .claude/guidelines/domain-layer-guidelines.yml
        effort: S
        notes: >
          Add a single exported field QualifiedAnnotations []string to
          the JavaDeclaration struct, placed immediately after
          Annotations. Keep godoc style consistent with the existing
          fields. Do not add constructors, validators, or new types.
          Do not touch JavaFileSummary or JavaMethod. No new imports.
          No changes to any file under internal/domain/port/parser/,
          internal/domain/errors/, internal/domain/service/, or
          internal/domain/usecase/.

  - id: 2
    name: Infrastructure (treesitter)
    depends_on: [1]
    groups:
      - id: T2-G1
        scope:
          create: []
          modify:
            - internal/infrastructure/treesitter/parser.go
            - internal/infrastructure/treesitter/queries.go
        guidelines:
          - .claude/guidelines/infrastructure-layer-guidelines.yml
        effort: M
        notes: >
          queries.go — replace the empty package body with a block of
          package-private string constants naming every Tree-sitter
          Java node type consumed by parser.go (nodeClassDecl,
          nodeInterfaceDecl, nodeAnnotation, nodeMarkerAnnotation,
          nodeNormalAnnotation, nodeModifiers, nodeIdentifier,
          nodeScopedIdentifier, nodeTypeIdentifier,
          nodeScopedTypeIdentifier, nodeMethodDecl,
          nodeFormalParameters, nodeGenericType, nodePackageDecl,
          nodeImportDecl, nodeVoidType, nodeIntegralType,
          nodeFloatingPointType, nodeBooleanType, nodeArrayType,
          nodeEnumDecl, nodeRecordDecl, nodeSuperclass,
          nodeSuperInterfaces, nodeExtendsInterfaces, nodeTypeList,
          nodeInterfaceTypeList, nodeClassBody, nodeInterfaceBody,
          nodeFormalParameter, nodeSpreadParameter,
          nodeVariableDeclaratorID). No new imports. No exported
          symbols.
          parser.go — three categories of edit, all under one group to
          avoid merge conflicts: (1) rewrite extractAnnotations so it
          returns two slices (simple, qualified) of equal length;
          handle scoped_identifier as the name-bearing child of a
          normal_annotation or annotation node; skip malformed
          annotations entirely so the length invariant holds;
          terminalSegment(rawName) yields the simple name. Update
          extractClassDeclaration, extractInterfaceDeclaration,
          extractEnumDeclaration, extractRecordDeclaration to bind
          both return values into decl.Annotations and
          decl.QualifiedAnnotations. (2) Add nodeScopedTypeIdentifier
          to the type-match case lists in buildMethodSignature and
          extractSingleParam so qualified generics like
          java.util.Optional<User> resolve cleanly. (3) Replace every
          string literal for Tree-sitter node types with its constant
          from queries.go — pure refactor, no behavioural change. Do
          NOT change containsErrors, the ctx.Err() entry check, the
          defer tree.Close(), the error-wrapping around
          ErrPartialParse / ErrParseFailure, or the iteration order
          inside the top-level root.Child(i) loop. Do NOT introduce
          .scm files. Do NOT add new exported symbols. strings is
          already imported.

  - id: 6
    name: Tests (parallel)
    depends_on: [2]
    groups:
      - id: T6-G1
        scope:
          create: []
          modify:
            - internal/infrastructure/treesitter/parser_test.go
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: M
        notes: >
          Add seven new tests alongside the existing five, keeping all
          existing tests unchanged: (1)
          TestParser_MultipleAnnotationsWithArguments — feeds @Entity
          @Table(name="users") public class User, asserts Annotations
          == ["Entity","Table"] and QualifiedAnnotations ==
          ["Entity","Table"]. (2) TestParser_QualifiedAnnotation —
          feeds @jakarta.persistence.Entity public class User, asserts
          Annotations == ["Entity"] and QualifiedAnnotations ==
          ["jakarta.persistence.Entity"]. (3)
          TestParser_MethodReturnGeneric — interface with Optional<User>
          findByEmail(String email), asserts signature ==
          "Optional<User> findByEmail(String email)". (4)
          TestParser_MethodParamGeneric — method void
          apply(java.util.List<String> names), asserts signature ==
          "void apply(java.util.List<String> names)" — locks the
          scoped_type_identifier case added in T2-G1. (5)
          TestParser_ImportStatic — file with import static
          java.util.Collections.emptyList and a normal scoped import;
          asserts both strings appear in summary.Imports. (6)
          TestParser_ImportWildcard — file with import java.util.*;
          asserts summary.Imports contains "java.util". (7)
          TestParser_PartialParseSurfacesErrorAndKeepsValidDeclarations
          — mixed source with a valid Good class followed by an
          unclosed Bad class; asserts errors.Is(err,
          domerr.ErrPartialParse), summary.HasErrors == true, and
          summary.Declarations contains a "class_declaration" named
          "Good". All tests use t.Parallel() and fstest.MapFS. No new
          helpers, no testdata/ dependency.

      - id: T6-G2
        scope:
          create:
            - testdata/springBootMinimal/project/src/main/java/com/app/user_management/domain/UserWithTableAnnotation.java
          modify:
            - internal/cli/command/scanIntegration_test.go
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: M
        notes: >
          UserWithTableAnnotation.java — package
          com.app.user_management.domain; class annotated with both
          @Entity and @Table(name = "users"); two private fields (id,
          email) to make it a non-trivial class. No imports needed —
          annotations resolve at parse time to simple names, and the
          bundled profile rule "has_annotation: Entity" ->
          "classify_as: entity" fires on the simple name.
          scanIntegration_test.go — add ONE new test
          TestScanCmd_Integration_MultiAnnotationClass. It calls
          copyFixture to stage the springBootMinimal tree into
          t.TempDir(), runs the scan factory, reads the emitted
          project-state.yaml, and asserts: (a) user-management module
          exists; (b) its contracts include a contract named
          "UserWithTableAnnotation" with type "entity". Use
          t.Parallel() and t.TempDir(). Reuse discardLogger,
          buildScanFactoryWithLogger, and copyFixture from
          helpers_test.go. Do NOT modify
          testdata/springBootMinimal/expected/project-state.yaml —
          the canonical golden file stays untouched so
          TestScanCmd_Integration_HappyPath keeps passing.
```

---

## Self-validation checklist

- [x] Every file in Section 1 appears in exactly one group in Section 9
      (6 rows → 6 distinct entries across T1-G1, T2-G1, T6-G1, T6-G2).
- [x] Every requirement ID is covered by at least one Section 1 row:
      EP01US-006 + EP01RF-001 + EP01RF-003 — rows 1-6; EP01RNF-002 —
      rows 2, 4 (deterministic traversal pinned by unit tests);
      EP01RNF-004 — row 3 (queries.go stays compile-time constants, no
      runtime asset).
- [x] No file path appears in two groups.
- [x] Every port referenced in Section 2 exists in the codebase today
      (`ParseJavaFilePort`, `WalkJavaFilesPort` both present under
      `internal/domain/port/parser/`). Zero new ports.
- [x] Use-case `Execute` signature unchanged — reconfirmed against
      `scanuc.UseCase`.
- [x] `Deps` struct: no new fields — reconfirmed in `wire.go`.
- [x] No `TODO` / `{placeholder}` in the plan.
- [x] DAG is acyclic: T1 → T2 → T6. No cycles.
- [x] Tier 1 exists because `internal/domain/model/javaFileSummary.go`
      is in the file set.
- [x] Tier 5 is NOT introduced because no wiring file appears in the
      file set; listed as N/A in Section 7.1.
- [x] Every `guidelines[]` path in Section 9 exists (verified:
      `domain-layer-guidelines.yml`,
      `infrastructure-layer-guidelines.yml`,
      `unit-test-layer-guidelines.yml`,
      `integration-test-layer-guidelines.yml` all present under
      `.claude/guidelines/`).
- [x] `.scm` query decision is documented (§4.4, §8 Q4) and non-blocking.
- [x] No `Blocking: Yes` open question — proceed to implementation.
