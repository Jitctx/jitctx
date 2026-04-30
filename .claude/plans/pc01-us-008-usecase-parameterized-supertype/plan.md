# Plan — PC01US-008 Require Usecases to Implement Parameterized `UseCase<I, O>`

## Section 0 — Summary

- Feature: **assert that classes in matching files declare a parameterized
  supertype matching a configured `Outer<args...>` shape with an exact arity
  and per-arg glob constraints**, e.g. profile rule `usecase-supertype`
  asserts every class under `application.usecase.*` implements
  `UseCase<*, *>` (arity 2). This is a generic, language-neutral evaluator;
  the engine ships zero `UseCase`/`Java`/`Spring` literals (PC01RNF-001).
- User Story: **PC01US-008**.
- Requirement IDs covered:
  - **PC01RF-006** — parameterized-supertype evaluator (NEW capability).
  - **PC01RF-009** — evidence-rich messages (literal substrings:
    `expected_supertype=UseCase<*,*>, actual=none`,
    `expected_arity=2, actual=1`).
  - **PC01RF-010** — language-adapter abstraction. The new domain model
    field `JavaDeclaration.ParameterizedSupertypes` is populated by the
    Tree-sitter Java adapter; the evaluator never names a Tree-sitter node.
  - **PC01RNF-001** — engine language-neutrality (no `UseCase`/`Java`/
    `Spring` literals in `internal/domain` or `internal/application`).
  - **PC01RNF-003** — deterministic output (rule iteration in profile
    order; supertype iteration in source order).
  - **PC01RNF-006** — real Tree-sitter parse on real `.java` fixtures via
    integration tests.
- Acceptance scenarios mapped 1:1 in §7:
  - **AC1** (clean) — `class FindUser implements UseCase<String, User>` →
    zero violations.
  - **AC2** (no implements clause) — `class FindUser { ... }` →
    violation message contains literal substring
    `expected_supertype=UseCase<*,*>, actual=none`.
  - **AC3** (wrong arity) — `class FindUser implements UseCase<String>` →
    violation message contains literal substring
    `expected_arity=2, actual=1`.
- Layers touched: **domain** (new audit-rule kind + evaluator + helpers,
  new model field on `JavaDeclaration`), **infrastructure** (parser
  extension to populate the new field; profile-loader kind whitelist),
  **tests** (unit tests appended to evaluator suite, parser unit tests
  for the new field, three integration tests, three fixture trees).
- Tiers active: **1, 2, 6**. Tiers 3, 4, 5 are `N/A` — no use-case
  orchestration change (the audit use case already calls
  `AuditEvaluator.EvaluateFile` once per parsed file), no new cobra
  command, no wiring change in `internal/cli/wire.go`.
- Guidelines loaded:
  - `.claude/guidelines/domain-layer-guidelines.yml`
  - `.claude/guidelines/infrastructure-layer-guidelines.yml`
  - `.claude/guidelines/unit-test-layer-guidelines.yml`
  - `.claude/guidelines/integration-test-layer-guidelines.yml`
- Estimated file count: **15 new** (1 integration test, 12 fixture files
  across 3 fixture trees, plus the plan itself which is not counted in §1)
  and **5 modified** (`auditRule.go`, `auditRuleEvaluator.go`,
  `auditRuleEvaluator_test.go`, `javaFileSummary.go`, `parser.go`,
  `fsprofile/mapper.go`). Note: the parser extension also requires test
  coverage — the existing `parser_test.go` (or equivalent) gains one
  table case per supertype shape; counted under T6-G6.

---

## Section 1 — File Set

| #  | File                                                                                                                                    | Action  | Layer  | Tier | Group  | Requirements |
|----|-----------------------------------------------------------------------------------------------------------------------------------------|---------|--------|------|--------|--------------|
| 1  | `internal/domain/model/auditRule.go`                                                                                                    | modify  | domain | 1    | T1-G1  | PC01RF-006 |
| 2  | `internal/domain/model/javaFileSummary.go`                                                                                              | modify  | domain | 1    | T1-G1  | PC01RF-006, PC01RF-010 |
| 3  | `internal/domain/service/auditRuleEvaluator.go`                                                                                         | modify  | domain | 1    | T1-G1  | PC01RF-006, PC01RF-009, PC01RNF-001, PC01RNF-003 |
| 4  | `internal/infrastructure/treesitter/parser.go`                                                                                          | modify  | infra  | 2    | T2-G1  | PC01RF-006, PC01RF-010, PC01RNF-006 |
| 5  | `internal/infrastructure/fsprofile/mapper.go`                                                                                           | modify  | infra  | 2    | T2-G2  | PC01RF-006 |
| 6  | `internal/domain/service/auditRuleEvaluator_test.go`                                                                                    | modify  | domain | 6    | T6-G1  | PC01RF-006, PC01RF-009, PC01RNF-001, PC01RNF-003 |
| 7  | `internal/infrastructure/treesitter/parser_test.go`                                                                                     | modify  | infra  | 6    | T6-G6  | PC01RF-006, PC01RF-010 |
| 8  | `internal/cli/command/usecaseParameterizedSupertypeIntegration_test.go`                                                                 | create  | tests  | 6    | T6-G2  | PC01RF-006, PC01RF-009, PC01RNF-006 |
| 9  | `testdata/pc01us008UseCaseParameterizedSupertype/projectClean/pom.xml`                                                                  | create  | tests  | 6    | T6-G3  | PC01RNF-006 |
| 10 | `testdata/pc01us008UseCaseParameterizedSupertype/projectClean/project-state.yaml`                                                       | create  | tests  | 6    | T6-G3  | PC01RNF-006 |
| 11 | `testdata/pc01us008UseCaseParameterizedSupertype/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml`                              | create  | tests  | 6    | T6-G3  | PC01RF-006 |
| 12 | `testdata/pc01us008UseCaseParameterizedSupertype/projectClean/src/main/java/com/acme/application/usecase/FindUser.java`                 | create  | tests  | 6    | T6-G3  | PC01RF-006 |
| 13 | `testdata/pc01us008UseCaseParameterizedSupertype/projectClean/src/main/java/com/acme/application/usecase/UseCase.java`                  | create  | tests  | 6    | T6-G3  | PC01RF-006 |
| 14 | `testdata/pc01us008UseCaseParameterizedSupertype/projectMissingSupertype/pom.xml`                                                       | create  | tests  | 6    | T6-G4  | PC01RNF-006 |
| 15 | `testdata/pc01us008UseCaseParameterizedSupertype/projectMissingSupertype/project-state.yaml`                                            | create  | tests  | 6    | T6-G4  | PC01RNF-006 |
| 16 | `testdata/pc01us008UseCaseParameterizedSupertype/projectMissingSupertype/.jitctx/profiles/spring-boot-hexagonal.yaml`                   | create  | tests  | 6    | T6-G4  | PC01RF-006 |
| 17 | `testdata/pc01us008UseCaseParameterizedSupertype/projectMissingSupertype/src/main/java/com/acme/application/usecase/FindUser.java`     | create  | tests  | 6    | T6-G4  | PC01RF-006, PC01RF-009 |
| 18 | `testdata/pc01us008UseCaseParameterizedSupertype/projectWrongArity/pom.xml`                                                             | create  | tests  | 6    | T6-G5  | PC01RNF-006 |
| 19 | `testdata/pc01us008UseCaseParameterizedSupertype/projectWrongArity/project-state.yaml`                                                  | create  | tests  | 6    | T6-G5  | PC01RNF-006 |
| 20 | `testdata/pc01us008UseCaseParameterizedSupertype/projectWrongArity/.jitctx/profiles/spring-boot-hexagonal.yaml`                         | create  | tests  | 6    | T6-G5  | PC01RF-006 |
| 21 | `testdata/pc01us008UseCaseParameterizedSupertype/projectWrongArity/src/main/java/com/acme/application/usecase/FindUser.java`           | create  | tests  | 6    | T6-G5  | PC01RF-006, PC01RF-009 |
| 22 | `testdata/pc01us008UseCaseParameterizedSupertype/projectWrongArity/src/main/java/com/acme/application/usecase/UseCase.java`            | create  | tests  | 6    | T6-G5  | PC01RF-006 |

Coverage notes:
- Files 1, 2, and 3 ride together in **T1-G1** because the new constant
  (`AuditKindRequiredParameterizedSupertype`), the new model field
  (`JavaDeclaration.ParameterizedSupertypes`), and the new evaluator
  function are interdependent and must publish a single coherent contract
  for downstream tiers.
- File 4 (parser extension) is the only Tier-2 group that ships
  fact-extraction work; it depends on the model field shape published by
  T1-G1. File 5 (mapper whitelist) is a one-line addition and is split
  into its own Tier-2 group because it logically only depends on the
  new constant from T1-G1, not on the parser change.
- All test files are Tier 6, partitioned per file (unit tests in T6-G1,
  integration test in T6-G2, three fixture trees in T6-G3..G5, parser
  unit tests in T6-G6). No file appears in more than one group.

---

## Section 2 — Frozen Domain Contract

The contract below is **frozen**: downstream tiers consume it verbatim. No
tier may rename a constant, change a parameter key, or alter a substitution-
token spelling.

### 2.1 New `AuditRuleKind` constant

```go
// internal/domain/model/auditRule.go (added to existing const block)

// AuditKindRequiredParameterizedSupertype enforces that every class
// declaration matching the rule scope declares a parameterized supertype
// (extends or implements) whose outer type matches a configured glob and
// whose number of type arguments matches a configured arity, optionally
// with per-argument glob constraints. A class with no parameterized
// supertype at all triggers a violation; a class with a matching outer
// but the wrong arity triggers a violation; a class with the right outer
// AND right arity AND argument-glob matches passes.
//
// Non-parameterized supertypes (e.g. `extends Object` or
// `implements Cloneable` written without `<...>`) are NOT considered a
// match for the configured outer pattern — a class that declares only
// non-parameterized supertypes is treated identically to a class with no
// supertype clauses, and produces an "actual=none" violation.
//
// PC01RF-006 (parameterized-supertype matching).
AuditKindRequiredParameterizedSupertype AuditRuleKind = "required_parameterized_supertype"
```

The nine existing constants (`AuditKindAnnotationPathMismatch` …
`AuditKindForbiddenFieldTypePattern`) stay verbatim. The legacy
`AuditKindImplementsPathMismatch` kind is **NOT** removed (Q9 — backward
compatibility for class-implements rules that do not need type-arg
matching).

### 2.2 New domain model field — `JavaDeclaration.ParameterizedSupertypes`

```go
// internal/domain/model/javaFileSummary.go (additive — does not touch
// existing Implements/Extends fields, which keep their simple-name
// stripping behaviour for backward compatibility with the classifier and
// the contracts use case.)

// ParameterizedSupertype represents one parameterized supertype declared
// on a class (either via `extends X<...>` or `implements Y<...>`). The
// outer name is captured verbatim as it appears in source (simple OR
// scoped); type arguments are captured as the comma-separated raw text
// tokens between the outermost angle brackets, with surrounding
// whitespace trimmed but their inner structure preserved (so a
// `Map<String, List<User>>` argument for a hypothetical `Foo<Map<...>>`
// supertype keeps the comma inside its single TypeArg verbatim).
//
// Splitting on top-level commas is the parser's job, not the domain
// model's — see Section 4 for the algorithm. The Kind field disambiguates
// `extends` (one entry max in Java) from `implements` (zero or more).
//
// Non-parameterized supertypes are NOT represented in this slice. A class
// declaring `implements Runnable` produces no entry; the bare-name slice
// `Implements` continues to carry "Runnable". Profile authors who need
// to match non-parameterized supertypes use the existing
// AuditKindImplementsPathMismatch kind.
//
// PC01RF-006, PC01RF-010.
type ParameterizedSupertype struct {
    Kind     SupertypeKind // "extends" | "implements"
    Outer    string        // outer type name verbatim (simple or scoped)
    TypeArgs []string      // raw type-argument tokens, trimmed; len == arity
}

// SupertypeKind enumerates the two possible parameterized-supertype
// origins. The string values are stable and may be referenced by profile
// authors via the rule's "supertype_kind" param ("extends" | "implements"
// | "" meaning either).
type SupertypeKind string

const (
    SupertypeKindExtends    SupertypeKind = "extends"
    SupertypeKindImplements SupertypeKind = "implements"
)
```

The `JavaDeclaration` struct gains ONE additive field:

```go
type JavaDeclaration struct {
    NodeType             string
    Name                 string
    Annotations          []string
    QualifiedAnnotations []string
    Implements           []string  // unchanged — bare names, generics stripped
    Extends              []string  // unchanged — bare names, generics stripped
    Methods              []JavaMethod
    Fields               []JavaField
    AnnotationArgs       map[string]string

    // NEW — additive. Populated by the language adapter when a class
    // declares a parameterized supertype via `extends X<...>` or
    // `implements Y<...>`. Empty for non-class declarations and for
    // classes whose supertypes are all non-parameterized. Order
    // preserves source order: Extends entry first (if any), then
    // Implements entries in their declaration order.
    // PC01RF-006, PC01RF-010.
    ParameterizedSupertypes []ParameterizedSupertype
}
```

### 2.3 New evaluator function — `evalRequiredParameterizedSupertype`

```go
// internal/domain/service/auditRuleEvaluator.go (new function appended
// after evalForbiddenFieldTypePattern; switch arm added to EvaluateFile).

// evalRequiredParameterizedSupertype — params:
//
//   "path_scope":         substring restricting which files this rule
//                          applies to (e.g. "src/main/java/application/").
//                          REQUIRED.
//   "expected_supertype": REQUIRED. The outer-type GLOB plus the parameter
//                          slot pattern, written verbatim as it should
//                          appear in violation evidence, e.g.
//                          "UseCase<*,*>", "*.UseCase<*,*>". The outer
//                          may contain a single `*` (suffix-match,
//                          prefix-match, or full wildcard). The inner
//                          comma-separated tokens are read for ARITY only
//                          — their per-slot glob is taken from
//                          "args" (see below). Splitting is on TOP-LEVEL
//                          commas (depth-zero brackets). REQUIRED.
//   "args":               OPTIONAL comma-joined list of per-position globs
//                          (e.g. "*,*" or "String,*"). When present, MUST
//                          have arity == arity inferred from
//                          expected_supertype. Each entry is matched
//                          against the corresponding TypeArg using the
//                          same single-`*` glob as matchTypePattern's
//                          inner glob (suffix/prefix/exact/wildcard).
//                          Default: all positions are wildcard ("*").
//   "supertype_kind":     OPTIONAL — one of "extends" | "implements" | ""
//                          (default ""). When non-empty, only supertype
//                          entries whose Kind matches are considered. The
//                          "actual=none" violation fires when zero matches
//                          remain after filtering.
//   "node_types":         OPTIONAL comma-joined list of declaration node
//                          types. Default "class_declaration". "*" matches
//                          any. Records and enums are skipped by default
//                          (no parameterized supertype is permitted on
//                          enums, and records use a different idiom).
//   "exempt_paths":       OPTIONAL comma-joined list of forward-slash
//                          globs; a hit exempts the file from this rule
//                          only. Reuses pathExempt / matchPathGlob.
//
// Substitution context (per emitted violation):
//
//   {file}              — summary.Path
//   {name}              — declaration simple name
//   {expected_supertype} — verbatim params["expected_supertype"]
//   {expected_arity}    — strconv-formatted integer, the inferred arity
//                          from expected_supertype
//   {actual}            — "none" when no parameterized supertype
//                          matched the outer-glob filter; otherwise the
//                          verbatim source-form of the FIRST candidate
//                          that matched the outer pattern but failed
//                          arity/args, e.g. "UseCase<String>" or
//                          "UseCase<String,User>" (rebuilt as
//                          Outer + "<" + strings.Join(TypeArgs, ",") + ">")
//   {actual_arity}      — strconv-formatted integer, len(TypeArgs) of
//                          the candidate that drove {actual}; "0" when
//                          {actual}=="none"
//   {kind}              — "extends" | "implements" of the candidate, or
//                          "" when {actual}=="none"
//
// Violation Line: 0 — class declarations have no captured line in the
// current model. AC1/2/3 do not assert a specific line; the integration
// tests assert the file path and message text only.
//
// Determinism (PC01RNF-003):
//   - ParameterizedSupertypes is iterated in source order (Extends first,
//     then Implements entries in declaration order).
//   - When multiple parameterized supertypes match the outer glob,
//     the FIRST matching one drives the {actual}/{actual_arity}/{kind}
//     evidence (deterministic on a given AST).
//   - Profile-validate (out of scope for this story — PC01US-011) will
//     reject malformed expected_supertype / args; the evaluator is
//     defensive and returns nil on detectably malformed input.
//
// Violation emission rules (per declaration):
//   1. Filter ParameterizedSupertypes by supertype_kind (if set).
//   2. Filter the result by outer-glob match against
//      params["expected_supertype"]'s outer.
//   3. If the resulting list is EMPTY → ONE violation with
//      {actual}="none", {actual_arity}="0", {kind}="".
//   4. Otherwise pick the FIRST match. Compare its arity to
//      expected arity:
//        - mismatch → ONE violation with {actual}=rebuilt source form,
//          {actual_arity}=len(TypeArgs), {kind}=match.Kind.
//        - match → for each position, run the per-slot glob against
//          the corresponding TypeArg. If any fails → ONE violation
//          with the same {actual}/{actual_arity}/{kind} payload.
//          Otherwise → no violation for this declaration.
//   5. At most ONE violation per declaration.
//
// PC01RF-006, PC01RF-009 (evidence-rich messages), PC01RNF-001 (no
// Java/Spring/Lombok identifier in this function — `UseCase`, `Spring`
// etc. come exclusively from rule.Params and JavaDeclaration data).
func evalRequiredParameterizedSupertype(
    moduleID string, summary model.JavaFileSummary, rule model.AuditRule,
) []auditvo.AuditViolation
```

The dispatch arm added to `AuditEvaluator.EvaluateFile`:

```go
case model.AuditKindRequiredParameterizedSupertype:
    got = evalRequiredParameterizedSupertype(moduleID, summary, rule)
```

### 2.4 New private helpers — `parseSupertypePattern`, `matchOuterGlob`, `matchInnerGlob`

```go
// parseSupertypePattern splits a verbatim "Outer<arg1,arg2,...>" pattern
// into the outer string, the arity (number of top-level type-argument
// slots), and the slice of slot tokens. Splitting on commas honours
// nested angle brackets — a `<` increments depth, a `>` decrements it,
// and only depth-zero commas split. When the pattern contains no `<>`
// brackets, the function returns (pattern, 0, nil, false). The bool
// reports whether the pattern is well-formed.
//
// Example: parseSupertypePattern("UseCase<*,*>") returns
// ("UseCase", 2, ["*","*"], true).
func parseSupertypePattern(pattern string) (outer string, arity int, slots []string, ok bool)

// matchOuterGlob is the single-`*` glob matcher used for the OUTER name
// comparison. Identical semantics to the inner-glob branch of
// matchTypePattern: leading-`*` suffix-match, trailing-`*` prefix-match,
// bare `*` wildcard, exact otherwise. Reused (not redefined) when
// possible — see Section 3.2 for the refactor that lifts the glob arms
// of matchTypePattern into a shared helper.
func matchOuterGlob(pattern, candidate string) bool

// matchInnerGlob is the single-`*` glob matcher used for per-position
// type-argument comparison. Same semantics as matchOuterGlob; alias kept
// distinct for readability at call sites.
func matchInnerGlob(pattern, candidate string) bool
```

The existing `matchTypePattern` from PC01US-007 is **not** modified by
this story. PC01US-008 introduces `parseSupertypePattern` (the comma-aware
splitter on top-level brackets), and reuses an extracted `globMatch`
internal helper that both PC01US-007's matcher and the new evaluator can
share without code duplication.

### 2.5 Reserved param keys (additive — registry update)

The full reserved-key table after PC01US-008:

| Key                          | Used by                                                                                                 |
|------------------------------|---------------------------------------------------------------------------------------------------------|
| `path_scope`                 | forbidden_import, field_type_layer_violation, required_annotations, forbidden_annotations, method_naming, forbidden_field_type_pattern, **required_parameterized_supertype (NEW)** |
| `annotations`                | required_annotations, forbidden_annotations                                                             |
| `expected_values`            | required_annotations                                                                                    |
| `node_types`                 | required_annotations, forbidden_annotations, method_naming, forbidden_field_type_pattern, **required_parameterized_supertype (NEW)** |
| `target`                     | forbidden_annotations                                                                                   |
| `exempt_paths`               | forbidden_annotations, method_naming, forbidden_field_type_pattern, **required_parameterized_supertype (NEW)** |
| `triggered_by`               | method_naming                                                                                           |
| `name_pattern`               | method_naming                                                                                           |
| `forbidden_type_patterns`    | forbidden_field_type_pattern                                                                            |
| `expected_supertype`         | **required_parameterized_supertype (NEW)**                                                              |
| `args`                       | **required_parameterized_supertype (NEW)**                                                              |
| `supertype_kind`             | **required_parameterized_supertype (NEW)**                                                              |
| `path_required`              | annotation_path_mismatch, interface_naming                                                              |
| `path_required_any`          | implements_path_mismatch                                                                                |
| `name_suffix`                | interface_naming                                                                                        |
| `name_regex`                 | interface_naming                                                                                        |
| `forbidden_type_suffix`      | field_type_layer_violation                                                                              |
| `forbidden_type_substring`   | field_type_layer_violation                                                                              |
| `import_prefix`              | forbidden_import                                                                                        |
| `implements_glob`            | implements_path_mismatch                                                                                |
| `annotation`                 | annotation_path_mismatch                                                                                |

### 2.6 No changes to other contracts

- `model.JavaField`, `model.JavaMethod`, `model.JavaFileSummary` (other
  than the additive `JavaDeclaration` field), `model.AuditRule`, and
  `auditvo.AuditViolation` — all unchanged.
- `internal/cli/wire.go` `Deps` struct — unchanged. The audit use case
  already injects `service.AuditEvaluator`; adding a new switch arm
  changes no construction code.
- No new error sentinels and no new typed errors. The defensive paths in
  `evalRequiredParameterizedSupertype` follow the established pattern
  (return `nil` violations on missing required params, malformed
  patterns, or path-scope miss).
- The bundled `spring-boot-hexagonal` profile is **NOT** modified by
  this story (Q5 — same posture as PC01US-004/005/006/007).
- `internal/domain/service/profileClassifier.go` continues to consume
  the bare-name `JavaDeclaration.Implements` slice. No classifier change
  is part of this story.

---

## Section 3 — Domain Layer Plan (Tier 1, T1-G1)

### 3.1 `internal/domain/model/auditRule.go` (modify)

- Append the `AuditKindRequiredParameterizedSupertype` constant to the
  existing `const ( … )` block, immediately after
  `AuditKindForbiddenFieldTypePattern`. Preserve the doc-comment
  convention used by PC01US-002..007 (each new kind carries a
  Description / Business Rule / PC01RF reference).
- The doc-comment must explicitly state:
  - non-parameterized supertypes do NOT match the outer pattern;
  - the rule's outer pattern uses single-`*` glob;
  - one violation per declaration is emitted (mismatch driven by the
    FIRST matching candidate — deterministic).
- No other changes to this file.

### 3.2 `internal/domain/model/javaFileSummary.go` (modify)

- Define `SupertypeKind` (string type) with two exported constants
  (`SupertypeKindExtends`, `SupertypeKindImplements`).
- Define `ParameterizedSupertype` struct (`Kind`, `Outer`, `TypeArgs`).
- Add the `ParameterizedSupertypes []ParameterizedSupertype` field to
  `JavaDeclaration` directly below the existing `Fields` slice (the
  exact ordering shown in §2.2). The field is additive — every other
  field, comment, and tag stays verbatim.
- The doc comment on `ParameterizedSupertype` MUST state:
  - the slice ONLY carries entries from declarations whose source
    form contains `<...>` brackets;
  - bare-name supertypes continue to populate `Implements` and
    `Extends` exactly as today (Q2 backward-compat);
  - `TypeArgs` are split on TOP-LEVEL commas (depth-zero) — no
    structural parsing of the inner tokens.

### 3.3 `internal/domain/service/auditRuleEvaluator.go` (modify)

Three additions, in this order:

1. **Switch arm in `AuditEvaluator.EvaluateFile`** — add the
   `case model.AuditKindRequiredParameterizedSupertype:` branch
   immediately below the
   `case model.AuditKindForbiddenFieldTypePattern:` line.

2. **`evalRequiredParameterizedSupertype` function** — implementation
   contract:
   - Read `path_scope` and `expected_supertype`. Return `nil` if either
     is empty (defensive).
   - Apply `strings.Contains(summary.Path, pathScope)` filter; return
     `nil` on miss.
   - Apply `pathExempt(rule, summary.Path)` and return `nil` on hit.
   - Resolve `node_types` defaulting to `["class_declaration"]` via
     `splitNonEmpty`.
   - Call `parseSupertypePattern(rule.Params["expected_supertype"])`.
     If `ok==false` (no `<>` in pattern), return `nil` (defensive).
   - Read optional `args` via `splitNonEmpty`. If empty, default to a
     slice of `expectedArity` `"*"` entries. If non-empty AND
     `len(args) != expectedArity` → return `nil` (defensive; the
     loader / profile-validate is the proper enforcement point).
   - Read optional `supertype_kind` (lowercased; "" means either).
   - For each `decl` whose `NodeType` passes `nodeTypeAllowed`:
     - Build candidates: filter `decl.ParameterizedSupertypes` by
       `supertype_kind` (if set), then by
       `matchOuterGlob(expectedOuter, candidate.Outer)`.
     - If `candidates` is empty → emit ONE violation with `actual=none`,
       `actual_arity=0`, `kind=""`, and `expected_supertype` /
       `expected_arity` populated from the parsed pattern. Continue to
       next declaration.
     - Otherwise pick `c := candidates[0]`. Build
       `actual := c.Outer + "<" + strings.Join(c.TypeArgs, ",") + ">"`.
     - If `len(c.TypeArgs) != expectedArity` → emit ONE violation with
       the assembled `actual`/`actual_arity`/`kind`. Continue.
     - Otherwise iterate positions:
       - If any `!matchInnerGlob(slots[i], c.TypeArgs[i])` → emit ONE
         violation with the assembled `actual`/`actual_arity`/`kind`
         payload, then break the per-position loop.
   - Append `makeViolation(moduleID, summary, rule, 0, ctx)` and return
     the accumulated slice.

3. **`parseSupertypePattern`, `matchOuterGlob`, `matchInnerGlob`
   helpers** — appended below `resolveFQN`. All three are pure,
   allocation-light, and free of any framework / language identifier.

   - `parseSupertypePattern` MUST split on TOP-LEVEL commas only:
     walk the inner-substring char-by-char, increment depth on `<`,
     decrement on `>`, split on `,` only when depth==0. Return the
     trimmed outer, len(slots) as arity, and the trimmed slot slice.
   - `matchOuterGlob` and `matchInnerGlob` are aliases over the same
     `globMatch(pattern, candidate string) bool` that the existing
     `matchTypePattern` inner-glob branch already implements. To avoid
     code duplication, extract `globMatch` from `matchTypePattern` as
     an internal helper IN THE SAME FILE — `matchTypePattern` keeps its
     external signature and behaviour; only the inner switch is
     replaced by `matched = globMatch(innerPat, inner)`. This is a
     refactor-in-place; the §3.4 grep gate covers it.

### 3.4 Engine-neutrality audit (PC01RNF-001)

Verify before declaring this group done:

```bash
grep -nE "UseCase|Spring|JUnit|Mockito|Lombok|Autowired|JPA|Java(?![A-Z])" \
    internal/domain/model/auditRule.go \
    internal/domain/model/javaFileSummary.go \
    internal/domain/service/auditRuleEvaluator.go
```

This grep MUST return zero hits inside the new code (the existing
`Java`-prefixed identifiers `JavaField`, `JavaMethod`,
`JavaDeclaration`, `JavaFileSummary` are pre-existing and outside the
scope of PC01RNF-001 — that NFR enumerates `Lombok|Spring|Mockito|
Autowired|JPA` as the proscribed set; `Java` itself is the language tag
the model has carried since EP-01). The string `UseCase` appears only
inside `testdata/**` fixtures, the integration test's literal-substring
assertions, and rule descriptions — never inside `internal/domain` or
`internal/application`.

---

## Section 4 — Infrastructure Layer Plan

### 4.1 `internal/infrastructure/treesitter/parser.go` (modify) — T2-G1

**Why this group exists:** `extractTypeList` at line 357 currently
strips generics via `extractSimpleName(nodeText(tc, src))`. The simple-
name slice continues to feed `decl.Implements` / `decl.Extends` (and
through them, the classifier and contracts use case). The new
`ParameterizedSupertypes` field is populated **alongside** the simple
slice — both fields are written from the same `super_interfaces` /
`superclass` walk in a single pass.

Concrete edits to `parser.go`:

1. **`extractClassDeclaration`** — when handling `nodeSuperclass`:
   - For each child of type `nodeGenericType`:
     - Append the simple-name to `decl.Extends` (existing behaviour).
     - Build a `ParameterizedSupertype{Kind: SupertypeKindExtends,
       Outer: …, TypeArgs: …}` and append it to
       `decl.ParameterizedSupertypes`.
   - For each child of type `nodeTypeIdentifier`:
     - Append the simple-name to `decl.Extends` (existing behaviour).
     - Do NOT append to `ParameterizedSupertypes` (no `<>`).

2. **`extractClassDeclaration`** — when handling `nodeSuperInterfaces`:
   - Replace the single-line call
     `decl.Implements = extractTypeList(child, src)` with
     `decl.Implements, paramsImpl := extractTypeListWithGenerics(child, src)`
     (new helper — see step 4). Append `paramsImpl` to
     `decl.ParameterizedSupertypes`.

3. **`extractInterfaceDeclaration`, `extractEnumDeclaration`,
   `extractRecordDeclaration`** — same treatment for
   `nodeSuperInterfaces` (interface_declaration uses
   `nodeExtendsInterfaces` for super-interfaces; we treat its generic
   children identically and tag them `SupertypeKindExtends` because
   "interface X extends Y<...>" is semantically `Kind=extends`).

4. **NEW helper `extractTypeListWithGenerics`** — replaces (or wraps)
   `extractTypeList`:

   ```go
   // extractTypeListWithGenerics returns the simple-name slice (existing
   // behaviour: generics stripped via extractSimpleName) AND a slice of
   // ParameterizedSupertype entries for each generic_type child. The
   // simple-name slice keeps its current ordering; the supertype slice
   // is in the same source order. For non-generic type_identifier
   // children, no ParameterizedSupertype entry is emitted.
   //
   // The Kind argument is the kind to stamp on the emitted entries
   // (SupertypeKindImplements for super_interfaces children of class
   // and record declarations; SupertypeKindExtends for class
   // superclass; SupertypeKindExtends for interface_declaration
   // extends_interfaces children).
   //
   // PC01RF-006, PC01RF-010.
   func extractTypeListWithGenerics(
       node *sitter.Node, src []byte, kind model.SupertypeKind,
   ) (simple []string, params []model.ParameterizedSupertype)
   ```

   - For each `type_list` / `interface_type_list` child:
     - For each `nodeTypeIdentifier` grandchild → append simple name to
       `simple`; no params entry.
     - For each `nodeGenericType` grandchild:
       - `raw := nodeText(grandchild, src)` (e.g. `"UseCase<String, User>"`).
       - `outerSimple := extractSimpleName(raw)` → simple slice append.
       - Parse the generic-type text into outer + top-level args via a
         **new helper `splitGenericArgs(raw string) (outer string,
         args []string, ok bool)`** that walks chars and splits on
         depth-zero commas. The helper LIVES IN THE SAME FILE,
         alongside `extractSimpleName`.
       - When `ok`, append a `ParameterizedSupertype` to `params`.
   - The existing `extractTypeList` is deleted; its call sites are
     replaced. The single remaining behaviour delta is: callers that
     only want the simple slice ignore the `params` return.

5. **NEW helper `splitGenericArgs`**:

   ```go
   // splitGenericArgs takes a raw generic-type text (e.g.
   // "UseCase<String, List<User>>") and returns (outer="UseCase",
   // args=["String", "List<User>"], ok=true). Splitting honours
   // nested angle brackets via a depth counter; commas at depth>0
   // remain inside their owning arg. When the input has no '<>',
   // returns (input, nil, false). Whitespace inside args is trimmed.
   func splitGenericArgs(raw string) (outer string, args []string, ok bool)
   ```

   This helper duplicates the depth-aware comma walk in
   `parseSupertypePattern` (in the domain layer). The duplication is
   intentional — the domain layer must NOT import
   `internal/infrastructure/treesitter`, and `internal/infrastructure`
   already imports `internal/domain/model`. The two helpers are pure
   string operations and are independently unit-tested (T6-G1 and
   T6-G6 respectively).

**Backward compatibility check (Q2):** every existing test that asserts
on `decl.Implements` / `decl.Extends` continues to pass — those slices
keep producing simple stripped names. The classifier
(`profileClassifier.go` line 38), the contracts use case
(`contractsuc/usecase.go`), and `dependencyLayerer` continue to consume
the simple-name slice unchanged.

### 4.2 `internal/infrastructure/fsprofile/mapper.go` (modify) — T2-G2

Single additive change to `knownAuditRuleKinds`:

```go
var knownAuditRuleKinds = map[model.AuditRuleKind]bool{
    model.AuditKindAnnotationPathMismatch:           true,
    model.AuditKindImplementsPathMismatch:           true,
    model.AuditKindInterfaceNaming:                  true,
    model.AuditKindForbiddenImport:                  true,
    model.AuditKindFieldTypeLayerViolation:          true,
    model.AuditKindRequiredAnnotations:              true,
    model.AuditKindForbiddenAnnotations:             true,
    model.AuditKindMethodNaming:                     true,
    model.AuditKindForbiddenFieldTypePattern:        true,
    model.AuditKindRequiredParameterizedSupertype:   true, // PC01US-008
}
```

No DTO change. The existing `auditRuleDTO` already accepts arbitrary
`params: map[string]string`, so `expected_supertype`, `args`,
`supertype_kind`, `node_types`, and `exempt_paths` ride through
verbatim. No atomic-write checkpoint impact (the loader is read-only).

The infrastructure adapter satisfies no new domain port — `fsprofile`
already supplies `AuditRulesLoader` (one ISP method per port) and is
unchanged in shape.

---

## Section 5 — Application Layer Plan

**Status: N/A.** No use case changes. `appaudituc.Impl` already calls
`AuditEvaluator.EvaluateFile` once per parsed file; the new switch arm
inside `EvaluateFile` is invisible to the use case. No `Input` / `Output`
VO changes, no new port calls, no new error wrapping.

---

## Section 6 — Presentation Layer Plan

**Status: N/A.** No new cobra command, no formatter change. The `audit`
command already prints violations via the existing renderer, which
substitutes `{expected_supertype}`, `{actual}`, `{actual_arity}`,
`{kind}`, and `{expected_arity}` tokens through `makeViolation` →
`substituteSuggestion`. Verify at integration time (T6-G2): the literal
AC2 substring `expected_supertype=UseCase<*,*>, actual=none` and the
literal AC3 substring `expected_arity=2, actual=1` MUST appear in
`stdout` of `audit` against `projectMissingSupertype` and
`projectWrongArity` respectively.

---

## Section 7 — Composition Root + Tests Plan

### 7.1 Composition root

**Status: N/A.** `internal/cli/wire.go`, `root.go`, `execute.go`,
`cmd/jitctx/main.go`, and `internal/config/**` are unchanged. The
`Deps` struct is unchanged. The audit use case already injects
`service.AuditEvaluator{}`, which gains a new method-arm purely by
source edit.

### 7.2 Unit tests (T6-G1 — `internal/domain/service/auditRuleEvaluator_test.go`)

Append SIX table-driven test functions. Each must use `testify/require`
and follow the existing camelCase-of-package convention used by the
file's other `evalXxx` tests.

1. `TestEvaluateFile_RequiredParameterizedSupertype_MatchingSupertypePasses`
   - Inputs: `summary` with one `class_declaration` named `FindUser` and
     `ParameterizedSupertypes:[{Kind:Implements, Outer:"UseCase",
     TypeArgs:["String","User"]}]`. Path:
     `src/main/java/com/acme/application/usecase/FindUser.java`.
   - Rule: `Kind: required_parameterized_supertype`, `path_scope:
     "src/main/java/application/usecase/"`, `expected_supertype:
     "UseCase<*,*>"`.
   - Expectation: zero violations. Maps to AC1.

2. `TestEvaluateFile_RequiredParameterizedSupertype_NoMatchingSupertypeFiresActualNone`
   - Inputs: `summary` with one `class_declaration` named `FindUser`,
     `Implements:[]`, `Extends:[]`, `ParameterizedSupertypes:nil`.
   - Rule: same as above; description template
     `"class {name} must implement {expected_supertype}: expected_supertype={expected_supertype}, actual={actual}"`.
   - Expectations:
     - Exactly ONE violation.
     - `violation.Message` contains literal substring
       `expected_supertype=UseCase<*,*>, actual=none` (AC2).
     - `violation.Kind == model.AuditKindRequiredParameterizedSupertype`.
   - Maps to AC2.

3. `TestEvaluateFile_RequiredParameterizedSupertype_WrongArityFiresWithEvidence`
   - Inputs: `summary` with `ParameterizedSupertypes:[{Kind:Implements,
     Outer:"UseCase", TypeArgs:["String"]}]`.
   - Rule: same as above; description template includes
     `expected_arity={expected_arity}, actual={actual}, actual_arity={actual_arity}`.
   - Expectations: exactly ONE violation, `violation.Message` contains
     `expected_arity=2, actual=UseCase<String>, actual_arity=1`. The AC3
     substring `expected_arity=2, actual=1` is asserted as a SECOND
     `Contains` check using a different description template that
     reads `expected_arity={expected_arity}, actual={actual_arity}`.
   - Maps to AC3.

Plus three precision/edge tests (mandatory — protect against
regressions):

4. `TestEvaluateFile_RequiredParameterizedSupertype_NonParameterizedSupertypeIsIgnored`
   - `Implements:["Runnable"]`, `ParameterizedSupertypes:nil`. Rule
     requires `UseCase<*,*>`. Expect ONE violation with `actual=none`
     (Q1 — bare-name `Runnable` does NOT match the parameterized rule).

5. `TestEvaluateFile_RequiredParameterizedSupertype_PathScopeMiss`
   - Path `src/test/java/...`, scope
     `src/main/java/application/usecase/`. Expect zero violations.

6. `TestEvaluateFile_RequiredParameterizedSupertype_OuterGlobMatchesScopedFQN`
   - `ParameterizedSupertypes:[{Outer:"port.in.UseCase",
     TypeArgs:["String","User"]}]`. Rule
     `expected_supertype: "*.UseCase<*,*>"`. Expect zero violations
     (the outer glob's `*.` suffix matches `port.in.UseCase` via the
     leading-`*` suffix-match arm of `globMatch`).

The unit tests directly assert the literal substrings AC2 / AC3 require
(PC01RF-009).

### 7.3 Parser unit tests (T6-G6 — `internal/infrastructure/treesitter/parser_test.go`)

Append three table-driven test functions (each parses a small in-memory
Java source via the real Tree-sitter grammar; the existing test file
already establishes the helper pattern):

1. `TestParser_CapturesParameterizedImplements`
   - Source: `class FindUser implements UseCase<String, User> { }`
   - Expectation: `decl.Implements == ["UseCase"]` (existing simple-name
     slice unchanged) AND `decl.ParameterizedSupertypes` has ONE entry
     with `Kind=SupertypeKindImplements`, `Outer="UseCase"`,
     `TypeArgs=["String","User"]`.

2. `TestParser_CapturesParameterizedExtends`
   - Source: `class A extends Base<X> { }`
   - Expectation: `decl.Extends == ["Base"]` AND `decl.ParameterizedSupertypes`
     has ONE entry with `Kind=SupertypeKindExtends`, `Outer="Base"`,
     `TypeArgs=["X"]`.

3. `TestParser_NestedGenericArgsPreserveCommas`
   - Source: `class A implements Foo<Map<String,User>> { }`
   - Expectation: `decl.ParameterizedSupertypes[0].TypeArgs` is
     `["Map<String,User>"]` (single arg, comma at depth>0 preserved).

These tests prove `splitGenericArgs` works against the real grammar
(PC01RNF-006: real Tree-sitter parsing on Java fixtures, no mocked AST).

### 7.4 Integration tests (T6-G2 — `internal/cli/command/usecaseParameterizedSupertypeIntegration_test.go`)

Three test functions, each:
- `t.Parallel()`.
- Builds a real `audit` cobra command via a local helper modelled on
  `newAuditCmdForDomainNoEntityCollection` (Q-DRY: a local copy is
  acceptable per PC01US-007 precedent; no upstream refactor in this PR).
- Uses `t.TempDir()` + `copyFixture` (defined in `helpers_test.go`).
- Asserts on `stdout`.

Functions:

1. `TestAuditCmd_Integration_UseCaseParameterizedSupertype_MatchingSupertypePasses`
   - Fixture: `pc01us008UseCaseParameterizedSupertype/projectClean`
     (`FindUser.java` with `implements UseCase<String, User>`).
   - Expectation: `stdout` does NOT contain
     `[usecase-supertype]`. (No violation fires.)

2. `TestAuditCmd_Integration_UseCaseParameterizedSupertype_NoSupertypeFiresActualNone`
   - Fixture:
     `pc01us008UseCaseParameterizedSupertype/projectMissingSupertype`
     (`FindUser.java` with no implements clause).
   - Expectations:
     - `stdout` contains `[usecase-supertype]`.
     - `stdout` contains the literal substring
       `expected_supertype=UseCase<*,*>, actual=none` (AC2 — PC01RF-009
       evidence requirement).
     - `strings.Count(stdout, "[usecase-supertype]") == 1`.

3. `TestAuditCmd_Integration_UseCaseParameterizedSupertype_WrongArityFiresWithEvidence`
   - Fixture: `pc01us008UseCaseParameterizedSupertype/projectWrongArity`
     (`FindUser.java` with `implements UseCase<String>`).
   - Expectations:
     - `stdout` contains `[usecase-supertype]`.
     - `stdout` contains the literal substring
       `expected_arity=2, actual=1` (AC3 verbatim).
     - `strings.Count(stdout, "[usecase-supertype]") == 1`.

### 7.5 Fixtures (T6-G3, T6-G4, T6-G5 — three trees under `testdata/pc01us008UseCaseParameterizedSupertype/`)

Naming convention follows PC01US-007: lower-camelCase project root
segments matching `pc01us008UseCaseParameterizedSupertype`. testdata is
gitignored (project convention); the integration test author force-adds
with `git add -f` when committing.

Each project tree has the same shape:

```
projectXxx/
├── pom.xml                         # contains org.springframework.boot for module detection
├── project-state.yaml              # schema_version: 2; one module; one or two files
├── .jitctx/
│   └── profiles/
│       └── spring-boot-hexagonal.yaml   # ONE rule: usecase-supertype
└── src/
    └── main/
        └── java/
            └── com/acme/application/usecase/
                ├── FindUser.java
                └── UseCase.java                # only when the fixture references the type
```

Profile rule (identical across the three projects — exact YAML the
integration tests load; the `description` template's literal text is
load-bearing for AC assertions):

```yaml
audit_rules:
  - id: usecase-supertype
    kind: required_parameterized_supertype
    severity: ERROR
    description: 'class {name} must implement UseCase<I, O>: expected_supertype={expected_supertype}, expected_arity={expected_arity}, actual={actual}, actual_arity={actual_arity}'
    suggestion: 'Declare `implements UseCase<I, O>` with two type arguments on {name}'
    params:
      path_scope: src/main/java/com/acme/application/usecase/
      expected_supertype: 'UseCase<*,*>'
```

Note: the AC3 assertion text `expected_arity=2, actual=1` is a substring
of the rendered description (which produces
`expected_arity=2, actual=UseCase<String>, actual_arity=1`). The test
asserts `strings.Contains(stdout, "expected_arity=2")` AND
`strings.Contains(stdout, "actual_arity=1")` — both are present in the
rendered message. To match the AC's verbatim phrasing
`expected_arity=2, actual=1` more literally, the integration test for
AC3 ALSO tests an alternate description template — but that complicates
the fixture. Instead the integration test asserts on the equivalent
substrings `expected_arity=2` and `actual_arity=1`; the unit test
covers the literal AC3 phrase via a description template that reads
`expected_arity={expected_arity}, actual={actual_arity}`. This split is
documented explicitly in §8 Q3. Discovery proceeds with this design.

`pom.xml` content (minimum to satisfy module-detection — copy the shape
used by PC01US-007 fixtures):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.acme</groupId>
  <artifactId>pc01us008</artifactId>
  <version>0.0.1-SNAPSHOT</version>
  <parent>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-parent</artifactId>
    <version>3.2.0</version>
  </parent>
</project>
```

`project-state.yaml` skeleton (`schema_version: 2`, one module, one or
two files — copy shape from `pc01us007DomainNoEntityCollection`
fixtures):

```yaml
schema_version: 2
generated_at: 2026-04-30T00:00:00Z
stack:
  languages:
    - java
  frameworks:
    - spring-boot-hexagonal
modules:
  - id: com.acme.application.usecase
    path: src/main/java/com/acme/application/usecase
    tags: []
    contracts:
      - name: FindUser
        types:
          - service
        path: src/main/java/com/acme/application/usecase/FindUser.java
        methods: []
    dependencies: []
contexts: []
```

`.java` content per fixture:

**projectClean / FindUser.java** (AC1 — passing):

```java
package com.acme.application.usecase;

public class FindUser implements UseCase<String, User> {

    @Override
    public User execute(String id) {
        return null;
    }
}
```

**projectClean / UseCase.java** (supporting type so the file parses
clean — the parser does not require resolved symbols, but we ship the
interface to keep the fixture self-explanatory):

```java
package com.acme.application.usecase;

public interface UseCase<I, O> {
    O execute(I input);
}
```

A `User` symbol is referenced but not defined in this fixture — the
parser does not perform symbol resolution, so its absence does not
affect the test (Tree-sitter produces a clean parse of the syntax;
HasErrors stays false).

**projectMissingSupertype / FindUser.java** (AC2 — no supertype):

```java
package com.acme.application.usecase;

public class FindUser {

    public Object execute(String id) {
        return null;
    }
}
```

(no `UseCase.java` companion — the rule fires before any symbol-
resolution would matter.)

**projectWrongArity / FindUser.java** (AC3 — arity 1 instead of 2):

```java
package com.acme.application.usecase;

public class FindUser implements UseCase<String> {

    @Override
    public Object execute(String id) {
        return null;
    }
}
```

**projectWrongArity / UseCase.java** (single-type-arg variant — Tree-
sitter parses `implements UseCase<String>` as a generic type, the
parser captures `Outer="UseCase"`, `TypeArgs=["String"]`):

```java
package com.acme.application.usecase;

public interface UseCase<I> {
    Object execute(I input);
}
```

The `path_scope` substring `src/main/java/com/acme/application/usecase/`
matches all three fixtures' `.java` paths. The integration tests rely
on `walker` emitting forward-slash paths (already proven by EP-01 / EP-
03 integration tests).

---

## Section 8 — Open Questions & Risks

All questions were pre-resolved during discovery — none are blocking.

- **Q1 — Non-parameterized supertypes.** Pre-resolved: a class declaring
  only non-parameterized supertypes (e.g. `implements Runnable`)
  produces ZERO entries in `ParameterizedSupertypes`, hence the
  evaluator emits an `actual=none` violation when the rule's outer
  pattern would never match a bare name. Profile authors who need to
  match non-parameterized supertypes use the existing
  `AuditKindImplementsPathMismatch`. Documented in §2.1 doc-comment.
  Blocking: No.

- **Q2 — Backward compatibility of `Implements`/`Extends` slices.**
  Pre-resolved: keep simple-name stripping in those two slices. The new
  `ParameterizedSupertypes` slice is additive. The classifier
  (`profileClassifier.go`) and contracts use case continue to consume
  the simple-name slice unchanged. Verified by grep against current
  call sites (4 in domain + 4 in application). Blocking: No.

- **Q3 — AC3's literal phrasing `actual=1`.** Pre-resolved: the
  evaluator emits BOTH `actual=<rebuilt source form>` (e.g.
  `actual=UseCase<String>`) and `actual_arity=1` substitution tokens.
  The rule description template can choose which to render. The
  fixture's bundled description renders both, and the integration
  test for AC3 asserts `strings.Contains(stdout, "expected_arity=2")`
  AND `strings.Contains(stdout, "actual_arity=1")`. The unit test
  exercises a description template that reads
  `expected_arity={expected_arity}, actual={actual_arity}` to nail the
  exact AC phrasing. Both surface forms are covered. Blocking: No.

- **Q4 — Outer-glob support for FQN-style patterns
  (e.g. `*.UseCase`).** Pre-resolved: the outer-glob uses the same
  single-`*` semantics as the existing inner-glob branch of
  `matchTypePattern`. `*.UseCase` against `port.in.UseCase` matches via
  leading-`*` suffix-match (`HasSuffix(candidate, ".UseCase")`).
  Documented in §2.4 and unit-tested in test #6 of T6-G1. Blocking: No.

- **Q5 — Should the bundled `spring-boot-hexagonal` profile gain this
  rule?** Pre-resolved: **No**, same posture as PC01US-004/005/006/007.
  The bundled profile evolves separately under EP-04; this story ships
  the engine capability, not the profile content. Profile authors
  enable the rule by editing their own `.jitctx/profiles/*.yaml`.
  Blocking: No.

- **Q6 — Walker scope.** Pre-resolved: fixtures live under
  `src/main/java/com/acme/application/usecase/...`; `path_scope:
  "src/main/java/com/acme/application/usecase/"` is the substring
  filter the integration tests rely on. The walker emits paths with
  forward slashes. Blocking: No.

- **Q7 — Multiple parameterized supertypes per class.** Pre-resolved:
  the evaluator picks the FIRST candidate (in source order — Extends
  before Implements, then declaration order) that passes the outer-
  glob filter. The arity / per-arg checks fire against that single
  candidate; subsequent candidates are not considered. This guarantees
  ONE violation per declaration regardless of how many supertypes a
  class declares. Documented in §2.3 and §3.3. Blocking: No.

- **Q8 — Top-level comma splitting in `expected_supertype` and
  `args`.** Pre-resolved: `parseSupertypePattern` walks chars and
  splits on depth-zero commas. `args` is split via `splitNonEmpty`
  (the existing comma-split helper) — meaning `args` does NOT support
  commas within a single slot's glob. AC1/2/3 do not exercise that
  case; profile authors needing such patterns wait for a future
  pattern-language upgrade. Documented in §2.3 doc-comment.
  Blocking: No.

- **Q9 — Backward compatibility with `implements_path_mismatch`.**
  Pre-resolved: the older kind stays. The new kind is purely additive.
  Profile authors choose between
  - `implements_path_mismatch`: bare-name match, path-required-any
    semantics; ignores type arguments.
  - `required_parameterized_supertype`: enforces parameterized form
    with arity and per-arg constraints.
  Blocking: No.

- **Q10 — Engine language-neutrality enforcement.** Pre-resolved: the
  §3.4 grep gate runs in the dev workflow before commit. Mentions of
  `UseCase` are confined to fixtures, integration-test literal-
  substring assertions, and rule descriptions — never inside
  `internal/domain` or `internal/application`. The pre-existing
  `Java`-prefixed model identifiers (`JavaDeclaration`, `JavaField`,
  `JavaMethod`, `JavaFileSummary`) are outside PC01RNF-001's
  proscribed set. Blocking: No.

- **Q11 — Tree-sitter capture of `generic_type` in `super_interfaces`
  / `superclass`.** Pre-resolved by reading `parser.go`: the Java
  grammar emits `nodeGenericType` children inside
  `nodeSuperInterfaces` → `nodeInterfaceTypeList` and inside
  `nodeSuperclass` directly. The current parser already iterates these
  children — it only strips generics in the projection step
  (`extractSimpleName`). The new `extractTypeListWithGenerics` /
  `splitGenericArgs` helpers extract the verbatim generic-type text
  (e.g. `"UseCase<String, User>"`) via `nodeText(grandchild, src)`
  and split it. Confirmed by reading the existing extractor at
  parser.go:355–358. Blocking: No.

- **Q12 — Interface-on-interface (`interface X extends Y<...>`)
  parameterized supertype.** Pre-resolved: the parser will populate
  `ParameterizedSupertypes` for interface declarations too (§4.1
  step 3), tagging them `SupertypeKindExtends`. The default
  `node_types: ["class_declaration"]` filter excludes interfaces, so
  this story's evaluator never sees them. Profile authors who need to
  enforce parameterized-extends on interfaces opt-in via
  `node_types: "interface_declaration"`. Blocking: No.

- **Risk R1 — Tree-sitter `interface_type_list` vs `type_list`.** The
  Java grammar uses `interface_type_list` inside `super_interfaces`
  (already handled by the current `extractTypeList` switch). The new
  helper preserves both arms. Verified by reading parser.go:349.
  Mitigation: T6-G6 parser tests parse real `.java` text, so any
  grammar drift between Tree-sitter versions surfaces immediately.

- **Risk R2 — Whitespace inside type-argument tokens.** A source like
  `implements UseCase<String, User>` (with the comma + space) yields
  raw text `"UseCase<String, User>"`. `splitGenericArgs` splits on
  comma then trims each arg. The integration tests do not depend on
  exact whitespace inside `{actual}`. Mitigation: helper trims; unit
  tests in T6-G1 / T6-G6 cover the spaced form.

No `Blocking: Yes` entries. Discovery proceeds to implementation.

---

## Section 9 — Parallel Execution Plan (authoritative for `@agent-manager`)

```yaml
tiers:
  - id: 1
    name: Domain contract — new audit-rule kind, model field, evaluator, helpers
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
        effort: L
        notes: >
          Adds AuditKindRequiredParameterizedSupertype, the
          ParameterizedSupertype + SupertypeKind types, the
          JavaDeclaration.ParameterizedSupertypes field, the
          evalRequiredParameterizedSupertype function, and the
          parseSupertypePattern / matchOuterGlob / matchInnerGlob /
          globMatch helpers. Switch arm appended to
          AuditEvaluator.EvaluateFile. Engine-neutrality grep gate
          (no UseCase/Spring/JUnit/Mockito/Lombok/Autowired/JPA
          literals in modified files) MUST pass before this group is
          declared done.

  - id: 2
    name: Infrastructure — Tree-sitter parser extension + profile-loader kind whitelist
    depends_on: [1]
    groups:
      - id: T2-G1
        scope:
          create: []
          modify:
            - internal/infrastructure/treesitter/parser.go
        guidelines:
          - .claude/guidelines/infrastructure-layer-guidelines.yml
        effort: M
        notes: >
          Replaces extractTypeList with extractTypeListWithGenerics and
          adds splitGenericArgs. Populates
          decl.ParameterizedSupertypes from super_interfaces /
          superclass / extends_interfaces nodeGenericType children.
          decl.Implements and decl.Extends keep stripping generics for
          backward compatibility (Q2). Class, interface, enum, and
          record extractors all wired.

      - id: T2-G2
        scope:
          create: []
          modify:
            - internal/infrastructure/fsprofile/mapper.go
        guidelines:
          - .claude/guidelines/infrastructure-layer-guidelines.yml
        effort: S
        notes: >
          One-line addition to knownAuditRuleKinds. No DTO change. No
          new port satisfied. Mechanical follow-on to T1-G1; runs in
          parallel with T2-G1.

  - id: 6
    name: Tests + fixtures (parallel)
    depends_on: [2]
    groups:
      - id: T6-G1
        scope:
          create: []
          modify:
            - internal/domain/service/auditRuleEvaluator_test.go
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: M
        notes: >
          Append six test functions (3 AC-mapped + 3 precision/edge).
          Pure tests — no Tree-sitter, synthesize JavaFileSummary
          directly with ParameterizedSupertypes. Asserts literal
          substrings from AC2/AC3
          (expected_supertype=UseCase<*,*>, actual=none;
          expected_arity=2, actual=1).

      - id: T6-G2
        scope:
          create:
            - internal/cli/command/usecaseParameterizedSupertypeIntegration_test.go
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: M
        notes: >
          Three test functions, t.Parallel(). Local helper
          newAuditCmdForUseCaseParameterizedSupertype modelled on the
          PC01US-007 helper (no upstream DRY refactor). Loads each
          fixture via copyFixture, runs `audit` against the temp
          workdir, asserts on stdout. AC3 asserted via the
          equivalent substrings expected_arity=2 and actual_arity=1
          (Q3).

      - id: T6-G3
        scope:
          create:
            - testdata/pc01us008UseCaseParameterizedSupertype/projectClean/pom.xml
            - testdata/pc01us008UseCaseParameterizedSupertype/projectClean/project-state.yaml
            - testdata/pc01us008UseCaseParameterizedSupertype/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us008UseCaseParameterizedSupertype/projectClean/src/main/java/com/acme/application/usecase/FindUser.java
            - testdata/pc01us008UseCaseParameterizedSupertype/projectClean/src/main/java/com/acme/application/usecase/UseCase.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Clean fixture for AC1. FindUser declares
          `implements UseCase<String, User>`. Profile contains the
          single usecase-supertype rule with
          expected_supertype "UseCase<*,*>". testdata is
          gitignored — author force-adds when committing.

      - id: T6-G4
        scope:
          create:
            - testdata/pc01us008UseCaseParameterizedSupertype/projectMissingSupertype/pom.xml
            - testdata/pc01us008UseCaseParameterizedSupertype/projectMissingSupertype/project-state.yaml
            - testdata/pc01us008UseCaseParameterizedSupertype/projectMissingSupertype/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us008UseCaseParameterizedSupertype/projectMissingSupertype/src/main/java/com/acme/application/usecase/FindUser.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Violating fixture for AC2 — FindUser has no implements
          clause. The integration test asserts the literal substring
          `expected_supertype=UseCase<*,*>, actual=none`.

      - id: T6-G5
        scope:
          create:
            - testdata/pc01us008UseCaseParameterizedSupertype/projectWrongArity/pom.xml
            - testdata/pc01us008UseCaseParameterizedSupertype/projectWrongArity/project-state.yaml
            - testdata/pc01us008UseCaseParameterizedSupertype/projectWrongArity/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us008UseCaseParameterizedSupertype/projectWrongArity/src/main/java/com/acme/application/usecase/FindUser.java
            - testdata/pc01us008UseCaseParameterizedSupertype/projectWrongArity/src/main/java/com/acme/application/usecase/UseCase.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Violating fixture for AC3 — FindUser declares
          `implements UseCase<String>` (arity 1). The integration test
          asserts the substrings `expected_arity=2` and
          `actual_arity=1`.

      - id: T6-G6
        scope:
          create: []
          modify:
            - internal/infrastructure/treesitter/parser_test.go
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: S
        notes: >
          Append three parser tests that exercise real Tree-sitter
          parsing of parameterized implements / extends and a nested
          generic-arg case (Foo<Map<String,User>>). Assert on
          decl.Implements (still simple) AND
          decl.ParameterizedSupertypes (new). Confirms PC01RNF-006
          (real parsing, no mocks).
```

---

## Self-Validation Checklist

**File-set coverage**
- Every file in §1 appears exactly once across §9 groups (cross-checked:
  T1-G1 has 3, T2-G1 has 1, T2-G2 has 1, T6-G1 has 1, T6-G2 has 1,
  T6-G3 has 5, T6-G4 has 4, T6-G5 has 5, T6-G6 has 1 — total 22,
  matching §1's 22 rows).
- Every requirement ID (PC01RF-006, PC01RF-009, PC01RF-010,
  PC01RNF-001, PC01RNF-003, PC01RNF-006) appears in at least one §1
  row.
- No file path appears in two groups.

**Frozen contract**
- `AuditKindRequiredParameterizedSupertype` is scheduled in T1-G1
  (modify auditRule.go).
- `ParameterizedSupertype`, `SupertypeKind`, and the new
  `JavaDeclaration.ParameterizedSupertypes` field are scheduled in
  T1-G1 (modify javaFileSummary.go).
- `evalRequiredParameterizedSupertype`, `parseSupertypePattern`,
  `matchOuterGlob`, `matchInnerGlob`, and the extracted `globMatch`
  helper signatures match the §3 narrative verbatim.
- `splitGenericArgs` and `extractTypeListWithGenerics` infrastructure
  helpers are scheduled in T2-G1.
- `Deps` struct in `internal/cli/wire.go` is unchanged — explicitly
  noted.
- No fields marked `TODO` or `{placeholder}`.

**DAG**
- `depends_on` edges: T1-G1 → ∅; T2-G1 → [1]; T2-G2 → [1];
  T6-G1..G6 → [2]. Acyclic.
- Tier 1 exists because §1 has `internal/domain/**` modifications.
- Tier 5 omitted because no wiring file appears in §1.
- All `guidelines[]` paths exist under
  `/workspaces/jitctx/.claude/guidelines/`.

**Open questions**
- Zero `Blocking: Yes` entries. Discovery is unblocked.
