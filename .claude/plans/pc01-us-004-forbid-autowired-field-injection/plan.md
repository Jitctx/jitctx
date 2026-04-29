# Plan — PC01US-004 Forbid @Autowired field injection except in test utilities

## Section 0 — Summary

- Feature: a profile author can declare a single `forbidden_annotations` audit
  rule (`id: no-field-injection`) that fires on any production field
  annotated with `@Autowired` (or any other configured annotation), with
  per-rule path exemptions so files under `**/testsupport/**` are skipped.
  Constructor-parameter `@Autowired` is NOT flagged because the rule
  scopes to fields only.
- Requirement IDs covered: **PC01US-004** (story); directly **PC01RF-002**
  (forbidden-annotations evaluator), **PC01RF-003** (field-scoped evaluation),
  **PC01RF-008** (per-rule path exemptions); cross-cutting **PC01RNF-001**
  (engine language-neutrality), **PC01RNF-003** (deterministic output),
  **PC01RNF-006** (real Tree-sitter on real `.java` fixtures), **PC01RF-009**
  (evidence-rich messages).
- Layers touched: **domain (model, service, errors), infrastructure
  (fsprofile mapper + treesitter fields adapter), tests (unit + integration),
  testdata fixtures**. No `internal/application/usecase`, no
  `internal/cli/command`, no `wire.go`, no `format/` changes. Audit pipeline
  already orchestrates the new evaluator transparently because the
  `EvaluateFile` dispatch is just one extra `case` arm.
- Tiers active: **1** (domain), **2** (infrastructure adapters), **6**
  (tests + fixtures). Tiers 3, 4, 5 collapse — see §5, §6, §7.
- Guidelines loaded:
  `.claude/guidelines/domain-layer-guidelines.yml`,
  `.claude/guidelines/infrastructure-layer-guidelines.yml`,
  `.claude/guidelines/unit-test-layer-guidelines.yml`,
  `.claude/guidelines/integration-test-layer-guidelines.yml`.
- Estimated file count: **2 new + 4 modified** in production source
  (`auditRule.go`, `javaFileSummary.go` modified; `auditRuleEvaluator.go`,
  `mapper.go`, `parser.go`, `fields.go` modified) plus **1 new unit-test
  file**, **1 new integration-test file**, **12 new fixture files**.

### Verified working-tree facts (2026-04-29)

1. `evalRequiredAnnotations` at `internal/domain/service/auditRuleEvaluator.go:287-325`
   already proves the dispatch shape — one new arm in the `switch rule.Kind`
   block at lines 32-48 plus a new private `evalForbiddenAnnotations`
   helper is the entire engine delta.
2. `model.JavaField` at `internal/domain/model/javaFileSummary.go:15-18`
   currently exposes only `Name string` and `Type string`. Field-level
   annotations and field-line numbers are NOT extracted today; they MUST be
   added in Tier 1 (model) and Tier 2 (parser), or the evaluator cannot
   inspect fields nor satisfy AC Scenario 1 ("violation reported on the
   field's line").
3. `internal/infrastructure/treesitter/fields.go` already enumerates direct
   `field_declaration` children of `class_body` (line 100-110) and expands
   multi-declarator fields. Adding `Annotations` and `Line` is purely
   additive — `extractFieldDeclaration` walks the node children and a new
   `extractAnnotations(modifiersChild, src)` invocation populates the new
   field. Line is `node.StartPoint().Row + 1` (same convention as
   `comments.go:79`).
4. `extractAnnotations` at `parser.go:138-155` already returns simple-name
   slices for any modifiers node (covers field modifiers identically to
   class modifiers). No new node type to query — we reuse the existing
   helper.
5. `auditRuleDTO` at `internal/infrastructure/fsprofile/dto.go:39-46`
   decodes `params: map[string]string`. The new param keys piggyback on
   the existing string-map mechanism — no new DTO field, no DTO
   migration. Whitelist update in `mapper.go:11-18` is one new entry.
6. The user-profile loader (`fsprofile.NewDetectorWithLogger`) requires
   the project's `pom.xml` to contain `org.springframework.boot` (per
   `testdata/pc01us002UsecaseImplStereotype/projectClean/pom.xml`).
   Fixtures MUST include this `pom.xml` verbatim or the user profile is
   silently ignored and the bundled profile (which lacks the new rule)
   is used instead.
7. `path/filepath.Match` (stdlib) does NOT support `**`. The repository
   has NO `doublestar` import (verified in `go.mod`). Glob handling for
   `exempt_paths` patterns like `**/testsupport/**` MUST be a small,
   in-domain matcher (§8 Q1 below). No new third-party dep.
8. Parameter annotations on constructor parameters are NOT currently
   modelled in `JavaDeclaration` (constructor params surface only as
   text inside `JavaMethod.Signature` — see `parser.go:323-369`).
   Scenario 3 ("constructor parameter `@Autowired` is allowed") is
   therefore satisfied **structurally**: the field-scope evaluator sees
   only `decl.Fields`; constructor-param annotations are simply not in
   the data model. We exploit this rather than fight it. (Risk R-006.)
9. Per `CLAUDE.md` "Nomes de arquivos Go", new Go file is camelCase
   (`forbiddenAnnotationsEvaluator_test.go` for the unit test;
   `forbidAutowiredFieldInjectionIntegration_test.go` for the
   integration test).
10. Forbidden-token gate (`internal/qualitygate/exemptions.go`) lists
    `JUnit`, `Mockito`, `javax`, `spring`, `@Entity`, `@RestController`,
    `port/in`, `application/`. Test files (`_test.go`) and `testdata/`
    paths are exempt by the gate's path filter
    (`javaReferencesGate_test.go:44, 173-174`). Production source MUST
    NOT contain `Autowired`/`Spring`/`Lombok`/`JPA`/`Mockito` literals
    (PC01RNF-001) — the new evaluator handles `@Autowired` only because
    the rule YAML names `Autowired` in `params.annotations`; the engine
    code never names it.

## Section 1 — File Set

| #  | File                                                                                                                             | Action  | Layer  | Tier | Group |
|----|----------------------------------------------------------------------------------------------------------------------------------|---------|--------|------|-------|
| 1  | `internal/domain/model/auditRule.go`                                                                                             | modify  | domain | 1    | T1-G1 |
| 2  | `internal/domain/model/javaFileSummary.go`                                                                                       | modify  | domain | 1    | T1-G1 |
| 3  | `internal/domain/service/auditRuleEvaluator.go`                                                                                  | modify  | domain | 1    | T1-G1 |
| 4  | `internal/infrastructure/fsprofile/mapper.go`                                                                                    | modify  | infra  | 2    | T2-G1 |
| 5  | `internal/infrastructure/treesitter/parser.go`                                                                                   | modify  | infra  | 2    | T2-G2 |
| 6  | `internal/infrastructure/treesitter/fields.go`                                                                                   | modify  | infra  | 2    | T2-G2 |
| 7  | `internal/domain/service/auditRuleEvaluator_test.go`                                                                             | modify  | tests  | 6    | T6-G4 |
| 8  | `testdata/pc01us004ForbidAutowiredFieldInjection/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml`                       | create  | tests  | 6    | T6-G1 |
| 9  | `testdata/pc01us004ForbidAutowiredFieldInjection/projectClean/pom.xml`                                                            | create  | tests  | 6    | T6-G1 |
| 10 | `testdata/pc01us004ForbidAutowiredFieldInjection/projectClean/project-state.yaml`                                                 | create  | tests  | 6    | T6-G1 |
| 11 | `testdata/pc01us004ForbidAutowiredFieldInjection/projectClean/src/main/java/com/acme/Foo.java`                                    | create  | tests  | 6    | T6-G1 |
| 12 | `testdata/pc01us004ForbidAutowiredFieldInjection/projectViolating/.jitctx/profiles/spring-boot-hexagonal.yaml`                    | create  | tests  | 6    | T6-G2 |
| 13 | `testdata/pc01us004ForbidAutowiredFieldInjection/projectViolating/pom.xml`                                                        | create  | tests  | 6    | T6-G2 |
| 14 | `testdata/pc01us004ForbidAutowiredFieldInjection/projectViolating/project-state.yaml`                                             | create  | tests  | 6    | T6-G2 |
| 15 | `testdata/pc01us004ForbidAutowiredFieldInjection/projectViolating/src/main/java/com/acme/Foo.java`                                | create  | tests  | 6    | T6-G2 |
| 16 | `testdata/pc01us004ForbidAutowiredFieldInjection/projectExempt/.jitctx/profiles/spring-boot-hexagonal.yaml`                       | create  | tests  | 6    | T6-G3 |
| 17 | `testdata/pc01us004ForbidAutowiredFieldInjection/projectExempt/pom.xml`                                                           | create  | tests  | 6    | T6-G3 |
| 18 | `testdata/pc01us004ForbidAutowiredFieldInjection/projectExempt/project-state.yaml`                                                | create  | tests  | 6    | T6-G3 |
| 19 | `testdata/pc01us004ForbidAutowiredFieldInjection/projectExempt/src/test/java/com/acme/testsupport/Helper.java`                    | create  | tests  | 6    | T6-G3 |
| 20 | `internal/cli/command/forbidAutowiredFieldInjectionIntegration_test.go`                                                           | create  | tests  | 6    | T6-G5 |

(File counts: **2 new production Go files in T1, 1 new infra file in T2-G2 if
the parser team prefers a separate file, otherwise edits to existing files;
this plan assumes edits**. **1 new unit test file (T6-G4 modifies the
existing `auditRuleEvaluator_test.go` in additive fashion**. **1 new
integration-test Go file**, **12 new testdata fixture files**.)

## Section 2 — Frozen Domain Contract

Every signature below is consumed by Tier 2 and Tier 6. Once approved, no
downstream group may rename a field, swap a type, or add a positional
parameter.

### 2.1 New `AuditRuleKind` enum value

In `internal/domain/model/auditRule.go`:

```go
const (
    AuditKindAnnotationPathMismatch  AuditRuleKind = "annotation_path_mismatch"
    AuditKindImplementsPathMismatch  AuditRuleKind = "implements_path_mismatch"
    AuditKindInterfaceNaming         AuditRuleKind = "interface_naming"
    AuditKindForbiddenImport         AuditRuleKind = "forbidden_import"
    AuditKindFieldTypeLayerViolation AuditRuleKind = "field_type_layer_violation"
    AuditKindRequiredAnnotations     AuditRuleKind = "required_annotations"

    // AuditKindForbiddenAnnotations enforces that NONE of the listed
    // annotation simple names are present on a target. The target scope
    // is selected by params["target"] ∈ {"class", "field"} (default
    // "class"). Per-rule path exemptions are honoured via
    // params["exempt_paths"] (comma-joined list of forward-slash globs).
    // PC01RF-002, PC01RF-003, PC01RF-008.
    AuditKindForbiddenAnnotations AuditRuleKind = "forbidden_annotations"
)
```

### 2.2 New `JavaField` fields (additive)

In `internal/domain/model/javaFileSummary.go`:

```go
// JavaField represents one field declared in a class body.
type JavaField struct {
    Name        string   // field identifier, e.g. "repository"
    Type        string   // raw type token as it appears in source
    Annotations []string // simple names, no leading @ (e.g. ["Autowired"]). NEW.
    Line        int      // 1-based line of the field_declaration node. NEW. 0 if unknown.
}
```

The two new fields are **append-only** at the end of the struct. Existing
callers that construct `JavaField{Name: ..., Type: ...}` keep compiling
unchanged because Go zero-values the new fields.

### 2.3 New evaluator dispatch arm

In `internal/domain/service/auditRuleEvaluator.go`, the switch in
`EvaluateFile` adds exactly one arm:

```go
case model.AuditKindForbiddenAnnotations:
    got = evalForbiddenAnnotations(moduleID, summary, rule)
```

### 2.4 New private evaluator function (frozen signature)

```go
// evalForbiddenAnnotations — params:
//
//   "path_scope":   substring restricting which files this rule applies to
//                   (e.g. "/src/main/java/"). REQUIRED.
//   "annotations":  comma-joined list of forbidden annotation simple names
//                   (without the leading "@"), e.g. "Autowired". The rule
//                   fires when ANY listed annotation is present on a
//                   matching target. REQUIRED, non-empty.
//   "target":       one of "class" | "field". Default "class".
//                   - "class"  → inspect decl.Annotations on every
//                                JavaDeclaration whose NodeType is in
//                                node_types (default class_declaration).
//                   - "field"  → inspect annotations on every
//                                JavaField inside every JavaDeclaration
//                                whose NodeType is in node_types.
//   "node_types":   optional comma-joined list of declaration node types.
//                   Default "class_declaration". "*" matches any.
//   "exempt_paths": optional comma-joined list of forward-slash globs.
//                   Each glob is matched against summary.Path with
//                   service.matchPathGlob (see §2.5). Any match exempts
//                   the file from THIS rule only.
//
// Substitution context:
//
//   {file}        — summary.Path
//   {name}        — declaration simple name (target=class) OR field name (target=field)
//   {forbidden}   — comma-joined params["annotations"] (verbatim, in order)
//   {found}       — "[A,B,...]" of the subset of forbidden annotations actually
//                   present on the offending target (deterministic, in the
//                   order the annotations were declared in params).
//
// Violation Line:
//
//   - target=class  → 0 (class line is not currently captured).
//   - target=field  → field.Line (1-based; PC01US-004 Scenario 1 asserts
//                     "violation reported on the field's line").
//
// PC01RF-002 / PC01RF-003 / PC01RF-008 / PC01RF-009.
func evalForbiddenAnnotations(
    moduleID string, summary model.JavaFileSummary, rule model.AuditRule,
) []auditvo.AuditViolation
```

### 2.5 New private path-glob helper (frozen signature)

```go
// matchPathGlob reports whether path matches the forward-slash glob
// pattern. Supported syntax:
//
//   "/literal/segment/"       — substring match (no glob meta-chars).
//   "*foo"  / "foo*" / "*foo*" — single-segment globs (filepath.Match style).
//   "**/seg/**" / "**/seg"     — "**" matches zero or more "/"-separated
//                                segments (including none).
//
// Implementation outline: split pattern and path on "/"; walk both
// concurrently with "**" consuming any number of path segments. No
// regex compilation; deterministic; allocation-free in the common case.
//
// Returns (matched bool). Never returns an error: a malformed pattern
// is treated as "no match" so a profile typo never panics the run; the
// loader is the layer that rejects malformed exempt_paths.
func matchPathGlob(pattern, path string) bool
```

This helper lives in `auditRuleEvaluator.go` as an unexported function
alongside `matchGlob` (which is restricted to single-segment trailing-`*`
matching today). `matchPathGlob` is the more general matcher; `matchGlob`
stays as-is for the existing `evalImplementsPathMismatch` path.

### 2.6 New helper: per-rule path exemption gate

```go
// pathExempt reports whether summary.Path matches any glob in
// rule.Params["exempt_paths"] (comma-joined). Empty/missing key returns
// false. Used by evalForbiddenAnnotations and reusable by future
// per-rule-exemption evaluators (PC01RF-008 is engine-wide).
func pathExempt(rule model.AuditRule, path string) bool
```

### 2.7 Profile-loader whitelist

`internal/infrastructure/fsprofile/mapper.go`:

```go
var knownAuditRuleKinds = map[model.AuditRuleKind]bool{
    model.AuditKindAnnotationPathMismatch:  true,
    model.AuditKindImplementsPathMismatch:  true,
    model.AuditKindInterfaceNaming:         true,
    model.AuditKindForbiddenImport:         true,
    model.AuditKindFieldTypeLayerViolation: true,
    model.AuditKindRequiredAnnotations:     true,
    model.AuditKindForbiddenAnnotations:    true, // NEW
}
```

### 2.8 New error sentinels

**None.** A malformed `forbidden_annotations` rule (empty `annotations`,
missing `path_scope`) is silently skipped by the evaluator (defensive
posture, mirroring `evalRequiredAnnotations:294-297`). PC01US-011 plan —
not this story — is responsible for adding profile-validate-time errors
on these conditions.

### 2.9 `Deps` struct

**Unchanged.** No new use case; no new port. The existing
`Deps.Audit audituc.UseCase` consumes the new rule transparently.

### 2.10 Reserved param-key registry

The following keys are now reserved in `AuditRule.Params` and MUST NOT
be reused for a different semantic in any future evaluator:

| Key            | Type              | Owner kinds                                           |
|----------------|-------------------|-------------------------------------------------------|
| `path_scope`   | substring         | `forbidden_import`, `field_type_layer_violation`, `required_annotations`, `forbidden_annotations` |
| `annotations`  | comma-joined list | `required_annotations`, `forbidden_annotations`       |
| `node_types`   | comma-joined list | `required_annotations`, `forbidden_annotations`       |
| `target`       | enum string       | `forbidden_annotations` (NEW; `required_annotations` continues to use `node_types`) |
| `exempt_paths` | comma-joined list | `forbidden_annotations` (NEW; PC01RF-008 cross-cutting) |

The `target` / `node_types` distinction is intentional: `target` is the
**conceptual scope** the profile author thinks in (class | field); the
engine maps `target=field` to "iterate `decl.Fields` on every declaration
whose `NodeType` matches `node_types`". For `target=class`, the engine
inspects `decl.Annotations` directly — `node_types` filters which
declarations are considered. Future stories (`target=method`,
`target=supertype`) extend this enum.

## Section 3 — Domain Layer Plan (Tier 1 — single group T1-G1)

### 3.1 `model/auditRule.go`

Append `AuditKindForbiddenAnnotations` at the bottom of the existing
`const ( ... )` block per §2.1. No other changes.

### 3.2 `model/javaFileSummary.go`

Append `Annotations []string` and `Line int` to `JavaField` per §2.2.
Update the field's doc comment to document the new fields. No other
changes.

### 3.3 `service/auditRuleEvaluator.go`

Three additive edits:

1. New `case model.AuditKindForbiddenAnnotations` arm in the
   `EvaluateFile` switch — single line plus the existing pattern.
2. New `evalForbiddenAnnotations` function per §2.4. Body shape:

   ```text
   if pathScope == "" or len(forbidden) == 0 → return nil  // defensive
   if !strings.Contains(summary.Path, pathScope) → return nil
   if pathExempt(rule, summary.Path) → return nil
   nodeTypes ← splitNonEmpty(params["node_types"]); default ["class_declaration"]
   target   ← params["target"]; default "class"
   for each decl in summary.Declarations:
     if !nodeTypeAllowed(decl.NodeType, nodeTypes) → continue
     switch target:
       case "class":
         found ← intersect(decl.Annotations, forbidden, preserveForbiddenOrder=true)
         if len(found) > 0:
           emit one violation with Line=0, name=decl.Name, found=[...]
       case "field":
         for each field in decl.Fields:
           found ← intersect(field.Annotations, forbidden, preserveForbiddenOrder=true)
           if len(found) > 0:
             emit one violation with Line=field.Line, name=field.Name, found=[...]
       default: continue   // unknown target — defensive
   return violations
   ```

3. New `matchPathGlob(pattern, path string) bool` and
   `pathExempt(rule model.AuditRule, path string) bool` helpers per §2.5
   and §2.6. `pathExempt` uses `splitNonEmpty(rule.Params["exempt_paths"])`
   then short-circuits on the first matching glob.

   `matchPathGlob` reuses the existing `splitNonEmpty` for the comma list
   produced by `pathExempt`. Within a single pattern, segment splitting
   is on `/`. The "**" segment consumes 0..N path segments; "*" inside a
   single segment defers to `path.Match` (stdlib) with the leading "/"
   stripped — `path.Match` is sufficient for the segment-local glob
   syntax PC01RF-008 actually uses (`**/testsupport/**`,
   `**/testsupport`, `*Helper.java`).

   Acceptance test for the matcher (lives in T6-G4):

   | pattern              | path                                                | match |
   |----------------------|------------------------------------------------------|-------|
   | `**/testsupport/**`  | `src/test/java/com/acme/testsupport/Helper.java`     | true  |
   | `**/testsupport/**`  | `src/main/java/com/acme/Foo.java`                    | false |
   | `**/testsupport`     | `src/test/java/com/acme/testsupport`                 | true  |
   | `**/testsupport`     | `src/test/java/com/acme/testsupport/Helper.java`     | false |
   | `**/Helper.java`     | `src/test/java/com/acme/testsupport/Helper.java`     | true  |
   | `**/foo/**`          | `src/test/java/com/acme/foo`                         | true  |
   | `**/foo/**`          | `src/main/java/foo.txt`                              | false |

### 3.4 No new domain ports

The evaluator is a pure function over `JavaFileSummary`. No new port.
PC01RF-010 (language-adapter abstraction) is satisfied because the
evaluator never names Java/Spring/Lombok/Autowired in its source.
**Verify by grep before merging T1-G1**:

```
grep -E '(?i)(autowired|spring|lombok|java(?!FileSummary|Field|Method|Declaration|Import|Identifier|TypeArg|Language)|mockito|jpa)' \
     internal/domain/service/auditRuleEvaluator.go
```

The expected match-set is empty (the existing comment exemption applies:
type names like `JavaFileSummary` and `JavaField` are domain identifiers
that pre-date this story and are documented under PC01RNF-001's "no
identifier names a specific language" — `Java*` is the parser-input type,
not a framework name).

## Section 4 — Infrastructure Layer Plan (Tier 2 — two groups, parallel)

### 4.1 T2-G1 — `fsprofile/mapper.go` whitelist

Single-line addition per §2.7. No DTO change. No mapper logic change
(the existing pass-through of `params: map[string]string` already
carries `target`, `exempt_paths`, etc. through unmodified).

### 4.2 T2-G2 — `treesitter/parser.go` + `treesitter/fields.go`

Goal: populate `JavaField.Annotations` and `JavaField.Line`.

**Edit 1 (`fields.go` `extractFieldDeclaration`):**

After determining `fieldType` (around current line 134), capture the
1-based line number once: `line := int(node.StartPoint().Row) + 1`.
This is the line of the `field_declaration` node — for multi-declarator
fields (`private int a, b;`) all expanded `JavaField` values share the
same line, which matches the AC ("violation reported on the field's
line" — there is one declaration line even with multiple declarators).

Walk the same `node.ChildCount()` loop a second time looking for a
`nodeModifiers` child; pass that to the existing `extractAnnotations`
helper from `parser.go`. The function returns `(simple, qualified
[]string)`; assign `simple` to a local `fieldAnnotations` and discard
`qualified` (we have no use for FQNs on fields in this story; future
stories can add a `QualifiedAnnotations` slice mirroring
`JavaDeclaration`).

When constructing each `model.JavaField{...}`, set
`Annotations: fieldAnnotations, Line: line`.

**Edit 2 (`fields.go` `ListJavaFields`):**

`ListJavaFields` reuses `extractClassFields` which calls
`extractFields`. Edit 1 above is in the shared call path; no separate
edit here. Existing tests in `fields_test.go` keep passing because the
new fields are zero-valued for any test that does not assert on them.

**Edit 3 (`parser.go` `extractClassDeclaration`):**

Already calls `extractFields(child, src)` at line 203 — propagates the
new values through `decl.Fields` automatically. No change.

**No `.scm` query work.** The Tree-sitter Java grammar already exposes
`field_declaration` and `modifiers`/`annotation` children directly via
the AST walk — the bundled `.scm` queries (`bundledqueries/`) are used
by the classifier, not by `parser.go` / `fields.go`. No
`bundledqueries/` edit is required for this story; the Critical
Context's note about "field-level annotations may need a query
addition" does not apply here (the parser walks the AST imperatively).

### 4.3 No `fsmanifest` change

The new audit rule produces `auditvo.AuditViolation` with `Line > 0`
when target=field. The existing renderer formats the line as part of
the violation report. `fsmanifest` does not persist violations
(violations are use-case output, not manifest content).

## Section 5 — Application Layer Plan

**N/A.** `internal/application/usecase/audituc/usecase.go` orchestrates:
(a) load manifest, (b) detect/resolve user profile, (c) walk Java
sources, (d) for each file, call `service.AuditEvaluator.EvaluateFile`
over every rule, (e) sort the violation union by file path / line /
rule ID. Step (d) iterates rules of any kind; the new
`AuditKindForbiddenAnnotations` is dispatched inside the evaluator's
own switch — application code does not name the kind.

PC01RNF-003 (deterministic output) is preserved: the existing sort key
already includes `Line`, so target=field violations on different
lines sort stably; target=field violations on the same line sort by
rule ID (already deterministic).

## Section 6 — Presentation Layer Plan

**N/A.** `auditCmd.go` is unchanged. `format/audit.go` already prints
`[ruleID] file:line  message` for any non-zero `Line`. Scenario 1's
"violation reported on the field's line" assertion is satisfied by
the formatter that has been on `main` since EP-03 — verified against
the output style produced by `usecaseImplStereotypeIntegration_test.go`
(class-line violations show line 0, hidden by the formatter; field-
line violations will show the captured line).

If the existing formatter currently elides `Line == 0` and renders
nothing for non-zero lines, that is fine: the integration test asserts
`require.Contains(t, out, "Foo.java:")` and `require.Contains(t, out,
":<expected-line>")` separately.

## Section 7 — Composition Root + Tests Plan

### 7.1 Wiring

**No edits** to `wire.go`, `root.go`, `execute.go`, `cmd/jitctx/main.go`,
`internal/config/`. The audit vertical slice is complete.

### 7.2 Unit tests — T6-G4

**File**: `internal/domain/service/auditRuleEvaluator_test.go` (additive
edits — new test functions appended; existing tests unchanged).

New table-driven test functions:

1. `TestAuditEvaluator_ForbiddenAnnotations_FieldScope_FlagsAutowired`
   — one declaration with one field annotated `["Autowired"]`,
   target=field, asserts one violation with `Line == field.Line` and
   evidence `found=[Autowired]`.
2. `TestAuditEvaluator_ForbiddenAnnotations_FieldScope_NoFlagWhenAnnotationAbsent`
   — same shape, field has annotations `["Inject"]`, asserts zero
   violations.
3. `TestAuditEvaluator_ForbiddenAnnotations_FieldScope_RespectsExemptPaths`
   — file path `src/test/java/com/acme/testsupport/Helper.java`, rule
   `exempt_paths: **/testsupport/**`, asserts zero violations even
   though the field carries the forbidden annotation.
4. `TestAuditEvaluator_ForbiddenAnnotations_OutsidePathScopeIsIgnored`
   — file path outside `path_scope`, asserts zero violations.
5. `TestAuditEvaluator_ForbiddenAnnotations_ClassScope_FlagsClassAnnotation`
   — target=class, declaration has `["Deprecated"]`, rule forbids
   `Deprecated`, asserts one violation with `Line == 0` and
   `name == decl.Name`.
6. `TestAuditEvaluator_ForbiddenAnnotations_MalformedRuleEmitsNothing`
   — empty `annotations`, missing `path_scope`, unknown `target` value
   each yield zero violations (defensive parity with
   `RequiredAnnotations_MalformedRuleEmitsNothing`).
7. `TestAuditEvaluator_ForbiddenAnnotations_MultipleFieldsOneOffending`
   — two fields, only one carries the forbidden annotation, asserts
   exactly one violation pointing to the offending field's line and
   name (NOT a per-class collapse).
8. `TestAuditEvaluator_MatchPathGlob` — exhaustive table for
   `matchPathGlob` per the §3.3 acceptance table.

These tests use `model.JavaFileSummary` directly — no parser, no
infra. Pure-function coverage.

### 7.3 Integration test — T6-G5

**File**: `internal/cli/command/forbidAutowiredFieldInjectionIntegration_test.go`.

**Helper**: replicate the `newAuditCmdFor*(t, workDir, manifestPath)`
pattern from `usecaseImplStereotypeIntegration_test.go:27-73` inline as
`newAuditCmdForForbidAutowiredFieldInjection`. Same constructor
arguments to `appaudituc.New(...)`. (Q3 resolution: no upstream DRY
refactor — the helper-extraction discussion is a follow-up PR.)

**Test functions** (each `t.Parallel()`):

1. `TestAuditCmd_Integration_ForbidAutowired_OnProductionFieldFlagsViolation`
   — copies `projectViolating` into `t.TempDir()`, runs
   `audit --dir <tmp> --manifest <tmp>/project-state.yaml`, asserts
   `require.NoError(t, err)`, stdout contains `[no-field-injection]`,
   stdout contains `Foo.java:<expected-line>` (the literal line the
   `@Autowired` field sits on in the fixture; the line number is
   pinned in the fixture to allow exact-match assertion via
   `strings.Contains` on `Foo.java:9` or whichever line the source
   places the field), stdout contains `found=[Autowired]`,
   `strings.Count(out, "[no-field-injection]") == 1`.
   **Backs PC01US-004 Scenario 1** (`PC01RF-002`, `PC01RF-003`,
   `PC01RF-009`).
2. `TestAuditCmd_Integration_ForbidAutowired_OnTestSupportFieldIsExempted`
   — copies `projectExempt` into `t.TempDir()` (the only Java file
   lives at
   `src/test/java/com/acme/testsupport/Helper.java`), runs the same
   command, asserts no error, stdout contains the no-violations line,
   stdout does NOT contain `[no-field-injection]`. **Backs PC01US-004
   Scenario 2** (`PC01RF-008`).
3. `TestAuditCmd_Integration_ForbidAutowired_OnConstructorParameterIsAllowed`
   — copies `projectClean` into `t.TempDir()` (Java file has a
   constructor whose parameter carries `@Autowired`, but no field
   does), runs the same command, asserts no error, stdout does NOT
   contain `[no-field-injection]`. **Backs PC01US-004 Scenario 3**
   (target=field discrimination).
4. `TestAuditCmd_Integration_ForbidAutowired_Determinism` — runs the
   `projectViolating` fixture twice in two separate `t.TempDir()`s,
   normalises temp-dir prefix, asserts byte-identical stdout.
   **Backs PC01RNF-003** for the new rule wiring.

Why no golden file: same rationale as PC01US-003 plan — `require.Contains`
on rule ID, evidence, and offending file path tracks the AC text exactly
without coupling to formatter cosmetics.

### 7.4 Fixture content

#### 7.4.1 Profile YAML (all three projects share an identical YAML body)

The profile head (`name`, `languages`, `query_lang`, `detect`,
`module_detection`, classification `rules`) is copied verbatim from
`testdata/pc01us002UsecaseImplStereotype/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml`
lines 1-81. The audit rule body is replaced with a single rule:

```yaml
audit_rules:
  - id: no-field-injection
    kind: forbidden_annotations
    severity: ERROR
    description: 'Field {name} must not carry forbidden annotations [{forbidden}]; found={found}'
    suggestion: 'Replace field injection on {name} with constructor injection'
    params:
      path_scope: /src/main/java/
      annotations: 'Autowired'
      target: field
      node_types: class_declaration
      exempt_paths: '**/testsupport/**'
```

`path_scope: /src/main/java/` is the broadest substring that excludes
test sources by default; the exempt-paths glob then pulls back any
test-support files that the user keeps under `src/main/java/`. For the
exempt-fixture, the file lives under `src/test/java/...` and would
already fall outside `path_scope`; we still keep `exempt_paths` so
that the AC literally exercises the `**/testsupport/**` glob the
feature specifies.

To make Scenario 2 unambiguously test the **glob** (not just the
`/src/main/java/` substring rejection), the fixture path for the
exempt project is **`src/main/java/com/acme/testsupport/Helper.java`**
(under `/src/main/java/` — passes `path_scope` — and under
`testsupport` — caught by `exempt_paths`). This ensures the test
fails if `pathExempt` is broken, even though the substring filter
would also have caught a `src/test/java/...` path.

> **Final fixture path decision**: the exempt project's Java file is
> `src/main/java/com/acme/testsupport/Helper.java`. (Updates the
> filename row #19 above — the row reads
> `src/test/java/com/acme/testsupport/Helper.java` for symmetry with
> the .feature wording, but the actual fixture lives under
> `src/main/java/...` so that the `**/testsupport/**` glob is what
> exempts it. The .feature path verbatim is preserved in the test
> assertion message.) Resolution: **fixture path is
> `src/main/java/com/acme/testsupport/Helper.java`**, see §8 Q3.

#### 7.4.2 Java fixtures

**`projectClean/src/main/java/com/acme/Foo.java`** — constructor
parameter `@Autowired`, NO field carries `@Autowired`. Expected: zero
violations. Sketch:

```java
package com.acme;

import org.springframework.beans.factory.annotation.Autowired;

public class Foo {

    private final UserRepo repo;

    public Foo(@Autowired UserRepo repo) {
        this.repo = repo;
    }
}

class UserRepo {}
```

**`projectViolating/src/main/java/com/acme/Foo.java`** — field
`@Autowired`. Expected: ONE violation. The field MUST sit on a known
line (the integration test asserts `Foo.java:<line>`), so we pin the
file format. Suggested layout (line numbers explicit so the test
assertion is reproducible):

```
1  package com.acme;
2
3  import org.springframework.beans.factory.annotation.Autowired;
4
5  public class Foo {
6
7      @Autowired
8      private UserRepo repo;
9  }
10
11 class UserRepo {}
```

**Field declaration node** sits on line 8 (the `private UserRepo
repo;` line). Tree-sitter's `field_declaration` start row is line 8
(the modifier `@Autowired` starts on line 7 but is a sibling
`modifiers` child, not the field-declaration node). The integration
test asserts `Foo.java:8`. (See §8 Q4 for the line-number pinning
rationale.)

**`projectExempt/src/main/java/com/acme/testsupport/Helper.java`** —
field `@Autowired` but path is exempted. Expected: zero violations.
Same shape as the violating fixture but at the exempt path:

```java
package com.acme.testsupport;

import org.springframework.beans.factory.annotation.Autowired;

public class Helper {

    @Autowired
    private UserRepo repo;
}

class UserRepo {}
```

#### 7.4.3 `pom.xml` (all three projects, identical)

Byte-for-byte copy of
`testdata/pc01us002UsecaseImplStereotype/projectClean/pom.xml`
(contains `org.springframework.boot` so the user-profile detector
loads the new rule).

#### 7.4.4 `project-state.yaml` (all three projects)

Minimal `schema_version: 2` manifest declaring one module rooted at
the file's package. Suggested shape for `projectClean`:

```yaml
schema_version: 2
generated_at: 2026-04-29T00:00:00Z
stack:
  languages:
    - java
  frameworks:
    - spring-boot-hexagonal
modules:
  - id: com.acme
    path: src/main/java/com/acme
    tags: []
    contracts:
      - name: Foo
        types:
          - service
        path: src/main/java/com/acme/Foo.java
        methods: []
    dependencies: []
contexts: []
```

`projectViolating` is identical; `projectExempt` declares the module
under `src/main/java/com/acme/testsupport` and the contract is
`Helper`.

## Section 8 — Open Questions & Risks

### Open questions (all resolved — no Blocking: Yes)

| # | Question | Blocking | Resolution |
|---|----------|----------|------------|
| Q1 | `path/filepath.Match` does NOT support `**`; the repo has no `doublestar` dep — how is `**/testsupport/**` matched? | No | Implement `service.matchPathGlob(pattern, path string) bool` per §2.5: split both on `/`; treat `**` as "consumes 0..N segments"; defer single-segment matching to `path.Match`. Eight assertions in T6-G4 lock the matcher's contract. **Rejected alternative**: add `github.com/bmatcuk/doublestar` — would expand the dep set for one rule. **Rejected alternative**: use `regexp` with a translator — `**/foo/**` → regex compilation per file is slower and less readable. |
| Q2 | `target: field` vs `node_types: field_declaration` — two ways to scope to fields. Which wins? | No | They are not interchangeable. `node_types` filters which **declarations** the rule visits (a field cannot be a `class_declaration` — `node_types` is the wrong knob). `target=field` is the explicit "iterate `decl.Fields` inside every visited declaration" mode. The plan freezes both keys with distinct semantics; PC01US-005 (`target=method`) reuses the same pattern. |
| Q3 | Should the exempt-project Java fixture sit under `src/main/java/...` or `src/test/java/...`? | No | **`src/main/java/com/acme/testsupport/Helper.java`** so the test exercises the `**/testsupport/**` glob without leaning on the `/src/main/java/` `path_scope` substring (which would also exclude any `src/test/java/...` path and make Scenario 2 a tautology). The .feature wording uses `src/test/java/...` as a *narrative example* — the AC asserts behaviour, not literal paths, and the `**/testsupport/**` glob clearly includes both. |
| Q4 | Is the asserted line number (`Foo.java:8`) brittle? | No | The fixture is committed verbatim with explicit line layout; Tree-sitter's `field_declaration` line is deterministic for a deterministic source file. The integration test asserts `strings.Contains(out, "Foo.java:8")` — if a future agent reformats the fixture, the test fails loudly with a clear "line N expected, got M" diff. The unit-test suite (T6-G4) further locks `field.Line` independently of any infra layer. |
| Q5 | How does target=class interact with PC01RF-008's exempt-paths? | No | `pathExempt` is checked at the top of `evalForbiddenAnnotations`, BEFORE the target switch. PC01RF-008 specifies that exempt-paths apply to "the rule" — i.e., the entire rule is skipped, regardless of class vs field scope. |
| Q6 | Should the bundled spring-boot-hexagonal profile gain the new rule? | No | Out of scope. The story is "profile author can author this rule"; rolling it into the bundled defaults is a separate decision tracked under PC01RNF-005's PROFILE_AUTHORING.md updates. The new rule lives only in the three new fixture profiles. |
| Q7 | Multi-declarator fields (`@Autowired private UserRepo a, b;`) — how many violations? | No | Two violations, one per declarator (`extractFieldDeclaration` already expands multi-declarator into one `JavaField` per name; each gets the same `Annotations` and `Line`). Out of AC scope but mention in the doc comment so the implementation is unambiguous. |
| Q8 | Do constructor-parameter annotations need to be modelled? | No | No. Scenario 3 is satisfied because constructor params live inside `JavaMethod.Signature` (text only) and never reach `decl.Fields`. The field-scope evaluator iterates `decl.Fields` exclusively. PC01US-005 / PC01US-006 may extend the model later; this story does not. |
| Q9 | What if an `exempt_paths` glob is malformed? | No | `matchPathGlob` returns `false` for a malformed pattern (no panic, no error). The profile-validate use case (PC01US-011) is the layer that should reject malformed globs at load time. |

### Risks

| ID | Risk | Probability | Impact | Mitigation |
|----|------|-------------|--------|------------|
| R-001 | Tree-sitter does not put annotation children under a `modifiers` node for `field_declaration` (different from `class_declaration`) | Low | High | `extractAnnotations` already accepts ANY node and walks its children for annotation nodes; field-declaration's `modifiers` child uses the same `nodeModifiers` constant. Verified by reading `parser.go:138-155` against the `field_declaration` grammar shape. T6-G2 unit test parses a real Java fixture with `@Autowired` on a field and asserts `field.Annotations == ["Autowired"]`. |
| R-002 | `JavaField.Annotations` addition breaks fixture-snapshot tests in `treesitter/fields_test.go` | Low | Medium | The two new fields are zero-valued for tests that build `JavaField` literals without naming them. Only tests that do equality on the whole struct (`require.Equal(t, expected, got)`) need updating; T2-G2 owners must run `fields_test.go` and patch any literal-equality assertions to include the new fields. |
| R-003 | `path_scope: /src/main/java/` is too broad; matches files outside the intended audit scope | Low | Low | The path_scope is a substring filter; the user controls it via the profile. The story's example (`forbidding @Autowired on fields`) is project-wide; broad scope is correct. |
| R-004 | Two consecutive runs produce different violation order due to map iteration | Low | High | `evalForbiddenAnnotations` walks `summary.Declarations` and `decl.Fields` in declaration order (slices, deterministic); `found` slice is built by walking `forbidden` in declaration order; substitution uses `strings.Join`. No map iteration in the hot path. T6-G5 Determinism test asserts byte equality. |
| R-005 | RNF-001 grep at PC01US-014 catches `Autowired` in the new test file | Low | Low | RNF-001 is scoped to `internal/domain`, `internal/application`, `internal/cli/*.go` (non-test). The new test files are `_test.go` — exempt by the same convention used in PC01US-002/PC01US-003. The fixture YAML and Java files live under `testdata/` which the gate filters out (`javaReferencesGate_test.go:44`). |
| R-006 | Constructor-parameter `@Autowired` reaches `decl.Annotations` because the parser confuses class-modifier annotations with method-parameter annotations | Very Low | High | `extractAnnotations` is invoked only on the class's outer `nodeModifiers` child (line 188) and on field/method modifiers via the same node type — not on `formal_parameter` children, which sit inside the method body. Verified by `parser.go:281-369` (parameter handling extracts text only via `extractFormalParams`; no annotation extraction). |
| R-007 | `**/testsupport/**` glob fails to match `**/testsupport` (no trailing path) | Medium | Low | The matcher per §3.3 explicitly defines `**` as "0..N segments" — `**/testsupport/**` matches both `.../testsupport/file` and `.../testsupport`. Locked by the matcher unit-test table (case 4 in §3.3). |
| R-008 | Integration test asserts `Foo.java:8` but fixture line drift breaks the test | Low | Low | Fixture content is committed verbatim and reviewed; the fixture file is small (11 lines). Mitigation: in the test, derive the expected line by reading the fixture source and finding the `private UserRepo repo;` line at runtime — but this couples the test to fixture content twice and is more brittle than asserting the literal. We assert the literal `Foo.java:8` and document the pinning. |
| R-009 | `node_types: class_declaration` filter accidentally excludes records / enums that the user expects to evaluate | Low | Low | The default is `class_declaration`; profile authors who want broader scope set `node_types: '*'` or list multiple kinds. Same default as `evalRequiredAnnotations`. |

## Section 9 — Parallel Execution Plan

```yaml
tiers:
  - id: 1
    name: Domain contract — kind enum, field model, evaluator dispatch, glob helper
    depends_on: []
    groups:
      - id: T1-G1
        scope:
          create: []
          modify:
            - internal/domain/model/auditRule.go
            - internal/domain/model/javaFileSummary.go
            - internal/domain/service/auditRuleEvaluator.go
        guidelines:
          - .claude/guidelines/domain-layer-guidelines.yml
        effort: M
        notes: >
          Single coordinated edit across three domain files. Adds
          AuditKindForbiddenAnnotations to the AuditRuleKind enum,
          appends Annotations and Line fields to JavaField, and adds
          three private helpers to auditRuleEvaluator.go:
          evalForbiddenAnnotations (the new dispatch arm),
          matchPathGlob (segment-based matcher supporting "**" and
          single-segment "*"), and pathExempt (per-rule exempt_paths
          gate). No new domain port. No new error sentinel. The
          evaluator file MUST stay free of any Java/Spring/Lombok
          identifier (PC01RNF-001) — verified by grep before merge.
          Param-key registry (Section 2.10) is the authoritative
          contract; no downstream tier may rename a key.

  - id: 2
    name: Infrastructure adapters (parallel)
    depends_on: [1]
    groups:
      - id: T2-G1
        scope:
          create: []
          modify:
            - internal/infrastructure/fsprofile/mapper.go
        guidelines:
          - .claude/guidelines/infrastructure-layer-guidelines.yml
        effort: S
        notes: >
          Single-line whitelist addition: knownAuditRuleKinds gains
          model.AuditKindForbiddenAnnotations. No DTO change; params
          map[string]string already round-trips arbitrary keys
          including target, exempt_paths, node_types. Independent of
          T2-G2.

      - id: T2-G2
        scope:
          create: []
          modify:
            - internal/infrastructure/treesitter/parser.go
            - internal/infrastructure/treesitter/fields.go
        guidelines:
          - .claude/guidelines/infrastructure-layer-guidelines.yml
        effort: M
        notes: >
          Populate JavaField.Annotations and JavaField.Line from the
          Tree-sitter AST. fields.go extractFieldDeclaration captures
          line = int(node.StartPoint().Row) + 1 once per
          field_declaration and walks the same children for a
          modifiers node, passing it to the existing
          extractAnnotations(node, src) helper from parser.go (which
          is already exported within the package). All expanded
          JavaField values from a multi-declarator field share the
          same Annotations and Line. parser.go is touched only to
          ensure extractAnnotations is reachable from fields.go (it
          already is — same package). No .scm query work; the
          bundled queries directory is untouched. Existing
          fields_test.go assertions that compare full JavaField
          structs may need their literals patched to include the new
          zero-valued fields — owners run go test ./... and patch
          any equality failures. Independent of T2-G1.

  - id: 6
    name: Tests + fixtures (parallel within tier)
    depends_on: [1, 2]
    groups:
      - id: T6-G1
        scope:
          create:
            - testdata/pc01us004ForbidAutowiredFieldInjection/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us004ForbidAutowiredFieldInjection/projectClean/pom.xml
            - testdata/pc01us004ForbidAutowiredFieldInjection/projectClean/project-state.yaml
            - testdata/pc01us004ForbidAutowiredFieldInjection/projectClean/src/main/java/com/acme/Foo.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Clean-state fixture for PC01US-004 Scenario 3 (constructor-
          parameter @Autowired allowed when target=field). pom.xml
          and profile head copied verbatim from
          testdata/pc01us002UsecaseImplStereotype/projectClean. The
          single audit rule is no-field-injection / kind
          forbidden_annotations / target field /
          exempt_paths **/testsupport/** / annotations Autowired /
          path_scope /src/main/java/. Foo.java has constructor-param
          @Autowired but NO field carries @Autowired so the field-
          scope evaluator emits zero violations. Independent of
          T6-G2 / T6-G3 / T6-G4 / T6-G5 — parallel-safe.

      - id: T6-G2
        scope:
          create:
            - testdata/pc01us004ForbidAutowiredFieldInjection/projectViolating/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us004ForbidAutowiredFieldInjection/projectViolating/pom.xml
            - testdata/pc01us004ForbidAutowiredFieldInjection/projectViolating/project-state.yaml
            - testdata/pc01us004ForbidAutowiredFieldInjection/projectViolating/src/main/java/com/acme/Foo.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Violating fixture for PC01US-004 Scenario 1. Profile YAML,
          pom.xml, project-state.yaml byte-identical to T6-G1.
          Foo.java is pinned to 11 lines so the field_declaration
          starts on line 8 — integration test asserts
          strings.Contains(out, "Foo.java:8") and
          strings.Contains(out, "found=[Autowired]"). Independent of
          T6-G1 / T6-G3 / T6-G4 / T6-G5.

      - id: T6-G3
        scope:
          create:
            - testdata/pc01us004ForbidAutowiredFieldInjection/projectExempt/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us004ForbidAutowiredFieldInjection/projectExempt/pom.xml
            - testdata/pc01us004ForbidAutowiredFieldInjection/projectExempt/project-state.yaml
            - testdata/pc01us004ForbidAutowiredFieldInjection/projectExempt/src/main/java/com/acme/testsupport/Helper.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Exempt fixture for PC01US-004 Scenario 2. Profile YAML
          identical to T6-G1/T6-G2. Helper.java has @Autowired on a
          field, but lives at
          src/main/java/com/acme/testsupport/Helper.java so the
          exempt_paths glob **/testsupport/** matches. Q3 resolution
          parks the file under src/main/java/... rather than
          src/test/java/... so the test exercises the glob, not the
          path_scope substring. Module manifest declares
          com.acme.testsupport as the only module. Independent of
          T6-G1 / T6-G2 / T6-G4 / T6-G5.

      - id: T6-G4
        scope:
          create: []
          modify:
            - internal/domain/service/auditRuleEvaluator_test.go
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: M
        notes: >
          Append eight new test functions per Section 7.2:
          - TestAuditEvaluator_ForbiddenAnnotations_FieldScope_FlagsAutowired
          - TestAuditEvaluator_ForbiddenAnnotations_FieldScope_NoFlagWhenAnnotationAbsent
          - TestAuditEvaluator_ForbiddenAnnotations_FieldScope_RespectsExemptPaths
          - TestAuditEvaluator_ForbiddenAnnotations_OutsidePathScopeIsIgnored
          - TestAuditEvaluator_ForbiddenAnnotations_ClassScope_FlagsClassAnnotation
          - TestAuditEvaluator_ForbiddenAnnotations_MalformedRuleEmitsNothing
          - TestAuditEvaluator_ForbiddenAnnotations_MultipleFieldsOneOffending
          - TestAuditEvaluator_MatchPathGlob (table-driven; eight
            cases per Section 3.3 acceptance table)
          Each test builds model.JavaFileSummary literals directly
          (no parser, no infra). All t.Parallel(). Independent of
          T6-G1 / T6-G2 / T6-G3 / T6-G5.

      - id: T6-G5
        scope:
          create:
            - internal/cli/command/forbidAutowiredFieldInjectionIntegration_test.go
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: M
        notes: >
          Integration test exercising audit end-to-end via real
          treesitter, fsprofile, fsmanifest, audituc adapters.
          Replicates the newAuditCmdFor*(t, workDir, manifestPath)
          helper inline (renamed
          newAuditCmdForForbidAutowiredFieldInjection) — same
          constructor parameters as
          usecaseImplStereotypeIntegration_test.go:47-61. Four test
          functions per Section 7.3:
          - TestAuditCmd_Integration_ForbidAutowired_OnProductionFieldFlagsViolation
          - TestAuditCmd_Integration_ForbidAutowired_OnTestSupportFieldIsExempted
          - TestAuditCmd_Integration_ForbidAutowired_OnConstructorParameterIsAllowed
          - TestAuditCmd_Integration_ForbidAutowired_Determinism
          Each uses t.TempDir() + t.Parallel(). Asserts
          string-contains rather than golden file. Depends on T6-G1
          / T6-G2 / T6-G3 fixture paths existing at runtime; no
          compile-time dependency on other Tier 6 groups.
```
