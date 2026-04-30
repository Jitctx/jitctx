# Plan — PC01US-007 Forbid Collections of Entities in Domain Models

## Section 0 — Summary

- Feature: **Forbid parameterized-type collections of entity types on fields inside domain models**, e.g. `private List<OrderEntity> orders;` and `private Set<UserEntity> users;`. The rule is profile-driven and matches a configurable list of `Outer<*Inner>` patterns; the engine itself ships zero domain-language identifiers (PC01RNF-001).
- User Story: **PC01US-007**.
- Requirement IDs covered:
  - **PC01RF-005** — parameterized type-argument matching (NEW capability landed by this story).
  - **PC01RF-009** — evidence-rich messages (literal substrings: `type=java.util.List<OrderEntity>`, `matched_pattern=List<*Entity>`).
  - **PC01RNF-001** — engine language-neutrality (no `Entity`/`List`/`Set` literals in `internal/domain` or `internal/application`).
  - **PC01RNF-003** — deterministic output (ordered patterns, ordered emit).
  - **PC01RNF-006** — real Tree-sitter parse on real `.java` fixtures via the integration tests.
- Acceptance scenarios mapped 1:1 in §6/§7:
  - **AC1** (clean) — `private List<String> tags;` under a `domain-no-entity-collection` rule whose patterns are `List<*Entity>,Set<*Entity>` → zero violations (the inner type does not match `*Entity`).
  - **AC2** (violation, FQN evidence) — `private List<OrderEntity> orders;` with `import java.util.List;` → message contains literal substring `type=java.util.List<OrderEntity>, matched_pattern=List<*Entity>`.
  - **AC3** (line evidence) — `private Set<UserEntity> users;` → violation reported on the field's `JavaField.Line` (1-based row of the `field_declaration` node).
- Layers touched: **domain** (new audit-rule kind + evaluator + helpers), **infrastructure** (`fsprofile/mapper.go` whitelist), **tests** (unit tests appended to evaluator suite, three integration tests, three fixture trees).
- Tiers active: **1, 2, 6**. Tiers 3, 4, 5 are `N/A` — no use-case orchestration change (`AuditEvaluator.EvaluateFile` already dispatches by `rule.Kind`), no new cobra command, no wiring change in `internal/cli/wire.go`.
- Guidelines loaded:
  - `.claude/guidelines/domain-layer-guidelines.yml`
  - `.claude/guidelines/infrastructure-layer-guidelines.yml`
  - `.claude/guidelines/unit-test-layer-guidelines.yml`
  - `.claude/guidelines/integration-test-layer-guidelines.yml`
- Estimated file count: **15 new** (1 plan, 1 integration test, 12 fixture files spread across 3 fixture trees, plus 1 nothing-else) and **3 modified** (`auditRule.go`, `auditRuleEvaluator.go`, `auditRuleEvaluator_test.go`, `fsprofile/mapper.go`). The plan file itself is this document and is not counted in §1.

---

## Section 1 — File Set

| #  | File                                                                                                                                                          | Action  | Layer  | Tier | Group  | Requirements |
|----|---------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|--------|------|--------|--------------|
| 1  | `internal/domain/model/auditRule.go`                                                                                                                          | modify  | domain | 1    | T1-G1  | PC01RF-005 |
| 2  | `internal/domain/service/auditRuleEvaluator.go`                                                                                                               | modify  | domain | 1    | T1-G1  | PC01RF-005, PC01RF-009, PC01RNF-001, PC01RNF-003 |
| 3  | `internal/infrastructure/fsprofile/mapper.go`                                                                                                                 | modify  | infra  | 2    | T2-G1  | PC01RF-005 |
| 4  | `internal/domain/service/auditRuleEvaluator_test.go`                                                                                                          | modify  | domain | 6    | T6-G1  | PC01RF-005, PC01RF-009, PC01RNF-001, PC01RNF-003 |
| 5  | `internal/cli/command/domainNoEntityCollectionIntegration_test.go`                                                                                            | create  | tests  | 6    | T6-G2  | PC01RF-005, PC01RF-009, PC01RNF-006 |
| 6  | `testdata/pc01us007DomainNoEntityCollection/projectClean/pom.xml`                                                                                             | create  | tests  | 6    | T6-G3  | PC01RNF-006 |
| 7  | `testdata/pc01us007DomainNoEntityCollection/projectClean/project-state.yaml`                                                                                  | create  | tests  | 6    | T6-G3  | PC01RNF-006 |
| 8  | `testdata/pc01us007DomainNoEntityCollection/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml`                                                         | create  | tests  | 6    | T6-G3  | PC01RF-005 |
| 9  | `testdata/pc01us007DomainNoEntityCollection/projectClean/src/main/java/com/acme/domain/Tag.java`                                                              | create  | tests  | 6    | T6-G3  | PC01RF-005 |
| 10 | `testdata/pc01us007DomainNoEntityCollection/projectListEntity/pom.xml`                                                                                        | create  | tests  | 6    | T6-G4  | PC01RNF-006 |
| 11 | `testdata/pc01us007DomainNoEntityCollection/projectListEntity/project-state.yaml`                                                                             | create  | tests  | 6    | T6-G4  | PC01RNF-006 |
| 12 | `testdata/pc01us007DomainNoEntityCollection/projectListEntity/.jitctx/profiles/spring-boot-hexagonal.yaml`                                                    | create  | tests  | 6    | T6-G4  | PC01RF-005 |
| 13 | `testdata/pc01us007DomainNoEntityCollection/projectListEntity/src/main/java/com/acme/domain/Order.java`                                                       | create  | tests  | 6    | T6-G4  | PC01RF-005, PC01RF-009 |
| 14 | `testdata/pc01us007DomainNoEntityCollection/projectSetEntity/pom.xml`                                                                                         | create  | tests  | 6    | T6-G5  | PC01RNF-006 |
| 15 | `testdata/pc01us007DomainNoEntityCollection/projectSetEntity/project-state.yaml`                                                                              | create  | tests  | 6    | T6-G5  | PC01RNF-006 |
| 16 | `testdata/pc01us007DomainNoEntityCollection/projectSetEntity/.jitctx/profiles/spring-boot-hexagonal.yaml`                                                     | create  | tests  | 6    | T6-G5  | PC01RF-005 |
| 17 | `testdata/pc01us007DomainNoEntityCollection/projectSetEntity/src/main/java/com/acme/domain/User.java`                                                         | create  | tests  | 6    | T6-G5  | PC01RF-005, PC01RF-009 |

Coverage notes:
- File 1 (kind constant) and File 2 (new evaluator function + two helpers) ride together in one Tier 1 group because the evaluator immediately consumes the new constant. Splitting them would force a transitive stub in the constant file.
- File 3 (mapper whitelist) is a one-liner — separate Tier 2 group because the infrastructure adapter logically depends on the published Tier 1 contract.
- All test files are Tier 6, partitioned per file (unit tests in T6-G1, integration test in T6-G2, three fixture trees in T6-G3..G5). No file appears in more than one group.

---

## Section 2 — Frozen Domain Contract

The contract below is **frozen**: downstream tiers consume it verbatim. No tier may rename a constant, change a parameter key, or alter a substitution-token spelling.

### 2.1 New `AuditRuleKind` constant

```go
// internal/domain/model/auditRule.go (added to existing const block)

// AuditKindForbiddenFieldTypePattern enforces that no field declared in a
// matching file has a parameterized type matching any of the configured
// "Outer<Inner>" patterns. The outer name is matched by exact equality;
// the inner name is matched by a single-segment glob (leading-`*` suffix
// match, trailing-`*` prefix match, bare `*` wildcard, or exact). Fields
// whose declared type is non-parameterized (no `<>` brackets) are silently
// skipped — this kind ONLY fires on parameterized declarations.
//
// The evaluator resolves the field's outer type to a fully-qualified name
// using the file's import list; the resolved FQN is surfaced under the
// {type} substitution token for evidence-rich messages (PC01RF-009).
//
// PC01RF-005 (parameterized type-argument matching).
AuditKindForbiddenFieldTypePattern AuditRuleKind = "forbidden_field_type_pattern"
```

The seven existing constants (`AuditKindAnnotationPathMismatch` … `AuditKindMethodNaming`) stay verbatim. The `AuditKindFieldTypeLayerViolation` kind is **NOT** removed (Q9 — backward compatibility for the older `forbidden_type_suffix` / `forbidden_type_substring` knobs).

### 2.2 New evaluator function — `evalForbiddenFieldTypePattern`

```go
// internal/domain/service/auditRuleEvaluator.go (new function appended
// after evalMethodNaming; switch arm added to EvaluateFile).

// evalForbiddenFieldTypePattern — params:
//
//   "path_scope":               substring restricting which files this rule
//                                applies to (e.g. "src/main/java/"). REQUIRED.
//   "forbidden_type_patterns":  comma-joined list of "Outer<Inner>" patterns,
//                                where Inner may contain a single `*` glob:
//                                  - "*"        wildcard, matches any inner
//                                  - "*Suffix"  matches if inner ends with Suffix
//                                  - "Prefix*"  matches if inner starts with Prefix
//                                  - "Exact"    matches only the exact inner name
//                                Outer names are always EXACT-matched (case-
//                                sensitive). Patterns without `<` and `>` are
//                                rejected silently (defensive — the loader
//                                should validate at load time). REQUIRED,
//                                non-empty.
//   "node_types":               optional comma-joined list of declaration
//                                node types the rule iterates fields under.
//                                Default "class_declaration". "*" matches any.
//   "exempt_paths":             optional comma-joined list of forward-slash
//                                globs (matchPathGlob); a hit exempts the
//                                file from this rule only.
//
// Substitution context (per emitted violation):
//
//   {file}            — summary.Path
//   {name}            — declaration simple name (the enclosing class/record/etc.)
//   {field_name}      — JavaField.Name
//   {type}            — fully-qualified parameterized type, e.g.
//                       "java.util.List<OrderEntity>". The outer simple name
//                       is resolved against summary.Imports via resolveFQN
//                       (see §2.4). When no FQN resolves, the outer name is
//                       emitted as-is (no java.lang. synthesis — Q3).
//   {matched_pattern} — the verbatim pattern that fired, e.g. "List<*Entity>"
//                       (the FIRST matching pattern in the order they appear
//                       in params["forbidden_type_patterns"]; deterministic
//                       per PC01RNF-003).
//
// Violation Line: field.Line (1-based row of the field_declaration node;
// AC3 asserts "violation reported on the field's line"). 0 if the parser
// could not determine a row.
//
// Determinism (PC01RNF-003):
//   - The patterns are iterated in the order they appear in
//     params["forbidden_type_patterns"].
//   - Per field, the FIRST matching pattern wins; subsequent patterns are
//     not evaluated against that field. This guarantees one violation per
//     offending field even if multiple patterns would match.
//   - Fields are iterated in JavaField slice order, which the parser
//     produces in source-declaration order.
//
// PC01RF-005, PC01RF-009, PC01RNF-001 (no Java/Spring identifier in this
// function — `Entity`, `List`, `Set` come exclusively from rule.Params).
func evalForbiddenFieldTypePattern(
    moduleID string, summary model.JavaFileSummary, rule model.AuditRule,
) []auditvo.AuditViolation
```

The dispatch arm added to `AuditEvaluator.EvaluateFile`:

```go
case model.AuditKindForbiddenFieldTypePattern:
    got = evalForbiddenFieldTypePattern(moduleID, summary, rule)
```

### 2.3 New private helper — `matchTypePattern`

```go
// matchTypePattern reports whether the field type token `fieldType`
// matches the pattern token `pattern`. Both are split on the FIRST `<`
// and the LAST `>` to extract outer + inner; outer is exact-matched and
// inner uses a single-segment glob:
//
//   pattern="*"        → match any inner
//   pattern starts "*" → suffix-match inner against pattern[1:]
//   pattern ends   "*" → prefix-match inner against pattern[:len-1]
//   otherwise          → exact match
//
// The function returns (outer string, inner string, matched bool). The
// extracted outer/inner from `fieldType` are returned regardless of the
// match result so callers can build evidence strings without re-parsing.
// When `fieldType` is non-parameterized (no `<` or `>`) or when the
// pattern itself is non-parameterized, the function returns (fieldType,
// "", false).
//
// Whitespace inside the field type (e.g. "List< OrderEntity >") is
// trimmed from both outer and inner before comparison.
func matchTypePattern(fieldType, pattern string) (outer, inner string, matched bool)
```

Edge cases the helper MUST handle deterministically:
- `fieldType="List"`, `pattern="List<*Entity>"` → returns `("List", "", false)`. Non-parameterized field types do not match parameterized patterns.
- `fieldType="List<OrderEntity>"`, `pattern="List<*Entity>"` → returns `("List", "OrderEntity", true)`.
- `fieldType="List<String>"`, `pattern="List<*Entity>"` → returns `("List", "String", false)`.
- `fieldType="Set<UserEntity>"`, `pattern="List<*Entity>"` → outer mismatch → returns `("Set", "UserEntity", false)`.
- `fieldType="List<List<Entity>>"`: split on the FIRST `<` and the LAST `>` yields outer=`List`, inner=`List<Entity>`. The current matcher treats inner as a single token; nested generics are out of scope (Q8). The inner glob `*Entity` would match `List<Entity>` because it ends with `Entity` — documented as a known limitation in §8 Q8.
- `fieldType="Map<String,UserEntity>"`: comma inside `<>` is preserved verbatim in the inner token. Multi-parameter outer patterns (e.g. `Map<*,*Entity>`) are explicitly **out of scope** (Q1).

### 2.4 New private helper — `resolveFQN`

```go
// resolveFQN returns a fully-qualified name for the simple type name
// `simple` by scanning `imports` for an entry whose terminal segment
// (substring after the LAST `.`) equals `simple`. The first matching
// import wins (imports preserve source order). When no import matches,
// `simple` is returned verbatim — there is NO java.lang.* fallback
// because no acceptance criterion exercises it (Q3).
//
// Example: resolveFQN("List", []string{"java.util.List", "java.util.Set"})
// returns "java.util.List".
func resolveFQN(simple string, imports []string) string
```

### 2.5 Reserved param keys (additive — registry update)

The full reserved-key table after PC01US-007:

| Key                          | Used by                                           |
|------------------------------|---------------------------------------------------|
| `path_scope`                 | forbidden_import, field_type_layer_violation, required_annotations, forbidden_annotations, method_naming, **forbidden_field_type_pattern (NEW)** |
| `annotations`                | required_annotations, forbidden_annotations       |
| `expected_values`            | required_annotations                              |
| `node_types`                 | required_annotations, forbidden_annotations, method_naming, **forbidden_field_type_pattern (NEW)** |
| `target`                     | forbidden_annotations                             |
| `exempt_paths`               | forbidden_annotations, method_naming, **forbidden_field_type_pattern (NEW)** |
| `triggered_by`               | method_naming                                     |
| `name_pattern`               | method_naming                                     |
| `path_required`              | annotation_path_mismatch, interface_naming        |
| `path_required_any`          | implements_path_mismatch                          |
| `name_suffix`                | interface_naming                                  |
| `name_regex`                 | interface_naming                                  |
| `forbidden_type_suffix`      | field_type_layer_violation                        |
| `forbidden_type_substring`   | field_type_layer_violation                        |
| `forbidden_type_patterns`    | **forbidden_field_type_pattern (NEW)**            |
| `import_prefix`              | forbidden_import                                  |
| `implements_glob`            | implements_path_mismatch                          |
| `annotation`                 | annotation_path_mismatch                          |

### 2.6 No changes to other contracts

- `model.JavaField`, `model.JavaDeclaration`, `model.JavaFileSummary`, `model.AuditRule`, `auditvo.AuditViolation` — all unchanged. The existing `JavaField.Line`, `JavaField.Name`, `JavaField.Type`, and `JavaFileSummary.Imports` already carry every datum the new evaluator needs.
- `internal/cli/wire.go` `Deps` struct — unchanged. The audit use case already injects `service.AuditEvaluator`; adding a new switch arm changes no construction code.
- No new error sentinels and no new typed errors. The defensive paths in `evalForbiddenFieldTypePattern` follow the established pattern (return `nil` violations on missing required params, malformed patterns, or path-scope miss).

---

## Section 3 — Domain Layer Plan (Tier 1, T1-G1)

### 3.1 `internal/domain/model/auditRule.go` (modify)

- Append the `AuditKindForbiddenFieldTypePattern` constant to the existing `const ( … )` block, preserving the alphabetical-ish grouping used by PC01US-002..006 (each new kind is added at the bottom with its own doc-comment).
- The doc-comment must reference PC01RF-005 and explicitly state the "non-parameterized field types are silently skipped" rule (§2 Q4).
- No other changes to this file.

### 3.2 `internal/domain/service/auditRuleEvaluator.go` (modify)

Three additions, in this order:

1. **Switch arm in `AuditEvaluator.EvaluateFile`** — add the `case model.AuditKindForbiddenFieldTypePattern:` branch immediately above `case model.AuditKindMethodNaming:` (alphabetical by constant suffix). This keeps the dispatch table grouped by data shape (path-scoped field iterators next to other path-scoped iterators).

2. **`evalForbiddenFieldTypePattern` function** — implementation contract:
   - Read `path_scope` and `forbidden_type_patterns`. Return `nil` if either is empty (defensive).
   - Apply `strings.Contains(summary.Path, pathScope)` filter; return `nil` on miss.
   - Apply `pathExempt(rule, summary.Path)` and return `nil` on hit.
   - Resolve `node_types` defaulting to `["class_declaration"]` via `splitNonEmpty` + the existing default.
   - Split `forbidden_type_patterns` via `splitNonEmpty` (the existing helper).
   - For each declaration whose `NodeType` passes `nodeTypeAllowed`, iterate `decl.Fields`. For each field:
     - For each pattern (in order), call `matchTypePattern(field.Type, pattern)`. On the FIRST `matched=true` hit:
       - Build `fqnOuter := resolveFQN(outer, summary.Imports)`.
       - Build `typeStr := fqnOuter + "<" + inner + ">"`.
       - Build the substitution context with `{file}`, `{name}` (decl name), `{field_name}`, `{type}`, `{matched_pattern}` (verbatim trimmed pattern token).
       - Append `makeViolation(moduleID, summary, rule, field.Line, ctx)`.
       - **Break** the per-field pattern loop (one violation per offending field, even if multiple patterns would match).
   - Return the accumulated slice.

3. **`matchTypePattern` and `resolveFQN` helpers** — appended below the existing `matchPathGlob` helper. Both are pure, allocation-light, and free of any framework / language identifier.

Engine-neutrality audit (PC01RNF-001) — verify before declaring this group done:

```bash
grep -nE "Entity|Mockito|Spring|JUnit|Lombok|Autowired|JPA|DisplayName|ExtendWith" \
    internal/domain/model/auditRule.go \
    internal/domain/service/auditRuleEvaluator.go
```

This grep MUST return zero hits inside the new code. Tokens like `Outer`, `Inner`, `pattern`, `field`, `imports` are the only domain-meaningful identifiers used. The string `Entity` appears only inside `testdata/**` fixtures and the integration test's literal-substring assertions — never inside `internal/`.

---

## Section 4 — Infrastructure Layer Plan (Tier 2, T2-G1)

### 4.1 `internal/infrastructure/fsprofile/mapper.go` (modify)

Single additive change to `knownAuditRuleKinds`:

```go
var knownAuditRuleKinds = map[model.AuditRuleKind]bool{
    model.AuditKindAnnotationPathMismatch:     true,
    model.AuditKindImplementsPathMismatch:     true,
    model.AuditKindInterfaceNaming:            true,
    model.AuditKindForbiddenImport:            true,
    model.AuditKindFieldTypeLayerViolation:    true,
    model.AuditKindRequiredAnnotations:        true,
    model.AuditKindForbiddenAnnotations:       true,
    model.AuditKindMethodNaming:               true,
    model.AuditKindForbiddenFieldTypePattern:  true, // PC01US-007
}
```

No DTO change. The existing `auditRuleDTO` already accepts arbitrary `params: map[string]string`, so `forbidden_type_patterns`, `node_types`, and `exempt_paths` ride through verbatim. No atomic-write checkpoint impact (the loader is read-only).

The infrastructure adapter satisfies no new domain port — `fsprofile` already supplies `AuditRulesLoader` (one ISP method per port) and is unchanged in shape.

---

## Section 5 — Application Layer Plan

**Status: N/A.** No use case changes. `appaudituc.Impl` already calls `AuditEvaluator.EvaluateFile` once per parsed file; the new switch arm inside `EvaluateFile` is invisible to the use case. No `Input`/`Output` VO changes, no new port calls, no new error wrapping.

---

## Section 6 — Presentation Layer Plan

**Status: N/A.** No new cobra command, no formatter change. The `audit` command already prints violations via the existing renderer, which substitutes `{type}` and `{matched_pattern}` tokens through `makeViolation` → `substituteSuggestion`. Verify at integration time (T6-G2): the literal AC2 substring `type=java.util.List<OrderEntity>, matched_pattern=List<*Entity>` must appear in `stdout` of `audit` against `projectListEntity`.

---

## Section 7 — Composition Root + Tests Plan

### 7.1 Composition root

**Status: N/A.** `internal/cli/wire.go`, `root.go`, `execute.go`, `cmd/jitctx/main.go`, and `internal/config/**` are unchanged. The `Deps` struct is unchanged. The audit use case already injects `service.AuditEvaluator{}`, which gains a new method-arm purely by source edit.

### 7.2 Unit tests (T6-G1 — `internal/domain/service/auditRuleEvaluator_test.go`)

Append three table-driven test functions. Each must use `testify/require` and follow the existing camelCase-of-package convention used by the file's other `evalXxx` tests.

1. `TestEvaluateFile_ForbiddenFieldTypePattern_NonEntityCollectionPasses`
   - Inputs: `summary` with one `class_declaration` containing `JavaField{Name:"tags", Type:"List<String>", Line: 7}`. Imports: `["java.util.List"]`. Path: `src/main/java/com/acme/domain/Tag.java`.
   - Rule: `Kind: forbidden_field_type_pattern`, `path_scope: "src/main/java/"`, `forbidden_type_patterns: "List<*Entity>,Set<*Entity>"`.
   - Expectation: zero violations. Maps to AC1.

2. `TestEvaluateFile_ForbiddenFieldTypePattern_ListEntityFiresWithFQNEvidence`
   - Inputs: `summary` with one `class_declaration` containing `JavaField{Name:"orders", Type:"List<OrderEntity>", Line: 9}`. Imports: `["java.util.List"]`. Path: `src/main/java/com/acme/domain/Order.java`.
   - Rule: same as above; description template `"Domain model field {field_name} carries forbidden collection: type={type}, matched_pattern={matched_pattern}"`.
   - Expectations:
     - Exactly ONE violation.
     - `violation.Line == 9`.
     - `violation.Message` contains the literal substring `type=java.util.List<OrderEntity>, matched_pattern=List<*Entity>` (AC2).
     - `violation.Kind == model.AuditKindForbiddenFieldTypePattern`.
   - Maps to AC2.

3. `TestEvaluateFile_ForbiddenFieldTypePattern_SetEntityFiresOnFieldLine`
   - Inputs: `summary` with one `class_declaration` containing `JavaField{Name:"users", Type:"Set<UserEntity>", Line: 11}`. Imports: `["java.util.Set"]`. Path: `src/main/java/com/acme/domain/User.java`.
   - Rule: same as above.
   - Expectations: exactly one violation, `violation.Line == 11`, message contains `type=java.util.Set<UserEntity>` and `matched_pattern=Set<*Entity>`.
   - Maps to AC3.

Plus three precision/edge tests (mandatory — protect against regressions):

4. `TestEvaluateFile_ForbiddenFieldTypePattern_NonParameterizedFieldIsIgnored`
   - Field `Type:"OrderEntity"` (no `<>`). Pattern `List<*Entity>`. Expect zero violations (Q4).

5. `TestEvaluateFile_ForbiddenFieldTypePattern_PathScopeMiss`
   - Path `src/test/java/...`, scope `src/main/java/`. Expect zero violations.

6. `TestEvaluateFile_ForbiddenFieldTypePattern_FQNFallbackWhenImportMissing`
   - Imports: `[]`. Field `Type:"List<UserEntity>"`. Pattern `List<*Entity>`. Expect violation with `type=List<UserEntity>` (no synthesized prefix — Q3).

The unit tests directly assert the literal substrings AC2 / AC3 require (PC01RF-009).

### 7.3 Integration tests (T6-G2 — `internal/cli/command/domainNoEntityCollectionIntegration_test.go`)

Three test functions, each:
- `t.Parallel()`.
- Builds a real `audit` cobra command via a local helper modelled on `newAuditCmdForForbidAutowiredFieldInjection` (Q-DRY: a local copy is acceptable per PC01US-004 precedent; no upstream refactor in this PR).
- Uses `t.TempDir()` + `copyFixture` (defined in `helpers_test.go`).
- Asserts on `stdout`.

Functions:

1. `TestAuditCmd_Integration_DomainNoEntityCollection_NonEntityCollectionPasses`
   - Fixture: `pc01us007DomainNoEntityCollection/projectClean` (`Tag.java` with `List<String>`).
   - Expectation: `stdout` does NOT contain `[domain-no-entity-collection]`. (`Tag.java` has no entity-collection field; the rule must not fire.)

2. `TestAuditCmd_Integration_DomainNoEntityCollection_ListEntityFiresWithFQNEvidence`
   - Fixture: `pc01us007DomainNoEntityCollection/projectListEntity` (`Order.java` with `private List<OrderEntity> orders;` on line 9).
   - Expectations:
     - `stdout` contains `[domain-no-entity-collection]`.
     - `stdout` contains the literal substring `type=java.util.List<OrderEntity>, matched_pattern=List<*Entity>` (AC2 — PC01RF-009 evidence requirement).
     - `stdout` contains `Order.java:9` (AC3 — line evidence).
     - `strings.Count(stdout, "[domain-no-entity-collection]") == 1`.

3. `TestAuditCmd_Integration_DomainNoEntityCollection_SetEntityFiresOnFieldLine`
   - Fixture: `pc01us007DomainNoEntityCollection/projectSetEntity` (`User.java` with `private Set<UserEntity> users;` on line 11).
   - Expectations:
     - `stdout` contains `[domain-no-entity-collection]`.
     - `stdout` contains `User.java:11`.
     - `stdout` contains `type=java.util.Set<UserEntity>, matched_pattern=Set<*Entity>`.

### 7.4 Fixtures (T6-G3, T6-G4, T6-G5 — three trees under `testdata/pc01us007DomainNoEntityCollection/`)

Naming convention follows PC01US-006: lower-camelCase project root segments matching `pc01us007DomainNoEntityCollection`. testdata is gitignored (project convention); the integration test author force-adds with `git add -f` when committing.

Each project tree has the same shape:

```
projectXxx/
├── pom.xml                         # contains org.springframework.boot for module detection
├── project-state.yaml              # schema_version: 2; one module; one file
├── .jitctx/
│   └── profiles/
│       └── spring-boot-hexagonal.yaml   # ONE rule: domain-no-entity-collection
└── src/
    └── main/
        └── java/
            └── com/acme/domain/
                └── XXX.java
```

Profile rule (identical across the three projects — exact YAML the integration tests load):

```yaml
audit_rules:
  - id: domain-no-entity-collection
    kind: forbidden_field_type_pattern
    severity: ERROR
    description: 'Domain model field {field_name} carries forbidden collection: type={type}, matched_pattern={matched_pattern}'
    suggestion: 'Replace {type} with a non-entity collection or a domain VO'
    params:
      path_scope: src/main/java/
      forbidden_type_patterns: 'List<*Entity>,Set<*Entity>'
```

`pom.xml` content (minimum to satisfy module-detection — copy the shape used by PC01US-006 fixtures):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.acme</groupId>
  <artifactId>pc01us007</artifactId>
  <version>0.0.1-SNAPSHOT</version>
  <parent>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-parent</artifactId>
    <version>3.2.0</version>
  </parent>
</project>
```

`project-state.yaml` skeleton (`schema_version: 2`, one module, one file — copy shape from `pc01us006UnitTestClassContract` fixtures):

```yaml
schema_version: 2
project:
  root: .
  language: java
modules:
  - id: app
    path: .
    files:
      - <relative path to the .java file>
```

`.java` content per fixture (line numbers asserted by tests are explicit; do not reformat):

**projectClean / Tag.java** (AC1):

```java
// 1: package com.acme.domain;
// 2:
// 3: import java.util.List;
// 4:
// 5: public class Tag {
// 6:
// 7:     private List<String> tags;
// 8:
// 9: }
package com.acme.domain;

import java.util.List;

public class Tag {

    private List<String> tags;

}
```

**projectListEntity / Order.java** (AC2 — line 9 asserted):

```java
// 1: package com.acme.domain;
// 2:
// 3: import java.util.List;
// 4:
// 5: import com.acme.domain.OrderEntity;
// 6:
// 7: public class Order {
// 8:
// 9:     private List<OrderEntity> orders;
// 10:
// 11: }
package com.acme.domain;

import java.util.List;

import com.acme.domain.OrderEntity;

public class Order {

    private List<OrderEntity> orders;

}
```

**projectSetEntity / User.java** (AC3 — line 11 asserted):

```java
// 1:  package com.acme.domain;
// 2:
// 3:  import java.util.Set;
// 4:
// 5:  import com.acme.domain.UserEntity;
// 6:
// 7:  public class User {
// 8:
// 9:     // owns a small set of related users
// 10:
// 11:    private Set<UserEntity> users;
// 12:
// 13: }
package com.acme.domain;

import java.util.Set;

import com.acme.domain.UserEntity;

public class User {

    // owns a small set of related users

    private Set<UserEntity> users;

}
```

The line numbers in the comments above are the values the integration tests assert on. The fixture authors must verify with Tree-sitter that `field_declaration.StartPoint().Row + 1` matches the asserted line (Q7). The PC01US-006 fixtures already exercise this row capture against unannotated fields, so behavior is already proven.

---

## Section 8 — Open Questions & Risks

All questions were pre-resolved during discovery — none are blocking.

- **Q1 — Pattern syntax.** Pre-resolved: support **only** `Outer<*InnerGlob>` patterns with a single inner type-parameter (`List<*Entity>`, `Set<*Entity>`, `List<OrderEntity>`, `List<*>`). Multi-parameter outer patterns (`Map<*, *Entity>`) are explicitly out of scope; the matcher splits on the FIRST `<` and LAST `>`, so `Map<*, *Entity>` is parseable as `outer=Map`, `inner=*, *Entity`, but the inner-glob matcher will not handle the comma — declared as a known limitation. Blocking: No.
- **Q2 — FQN resolution algorithm.** Pre-resolved: imports-only. Scan `summary.Imports` for an FQN whose terminal segment (after the last `.`) matches the field's outer simple name; first match wins; on miss return the simple name unchanged. Blocking: No.
- **Q3 — Empty/missing imports.** Pre-resolved: emit the type as-is. No `java.lang.` synthesis for `String`/`Integer`/etc., because no AC exercises that path; profile authors writing rules that need it can use `forbidden_type_substring` or wait for a future-extension key. Blocking: No.
- **Q4 — Non-parameterized field types.** Pre-resolved: silently skip. The matcher returns `(_, _, false)` when `fieldType` lacks `<>`, even if the pattern would otherwise match. Documented in §2.3 and unit-tested. Blocking: No.
- **Q5 — Should the bundled `spring-boot-hexagonal` profile gain this rule?** Pre-resolved: **No**, same posture as PC01US-004/005/006. The bundled profile evolves separately under EP-04; this story ships the engine capability, not the profile content. Profile authors enable the rule by editing their own `.jitctx/profiles/*.yaml`. Blocking: No.
- **Q6 — Walker scope.** Pre-resolved: fixtures live under `src/main/java/com/acme/domain/...`; `path_scope: "src/main/java/"` is the substring filter the integration tests rely on. The walker emits paths with forward slashes, so the substring match works on Windows too (the existing scan integration tests already prove this). Blocking: No.
- **Q7 — Line number for unannotated fields.** Pre-resolved: Tree-sitter `field_declaration.StartPoint().Row + 1` already returns the line of the `private` keyword for unannotated fields, as proven by PC01US-004 fixtures (`Foo.java:7`). The new fixtures follow the same shape. Blocking: No.
- **Q8 — Comma inside `forbidden_type_patterns`.** Pre-resolved: AC patterns contain no commas inside `<>`. The split-on-`,` parser treats every comma as a top-level separator; profile authors writing `Map<*,*Entity>` will get two malformed patterns (`Map<*` and `*Entity>`) which the matcher rejects via the missing-`<>` guard. Documented as a known limitation. Future work: a pattern-language upgrade. Blocking: No.
- **Q9 — Backward compatibility with `field_type_layer_violation`.** Pre-resolved: the older kind stays. The new kind is purely additive. The `forbidden_type_suffix` and `forbidden_type_substring` params remain bound to the older kind only. Blocking: No.
- **Q10 — Multiple matching patterns per field.** Pre-resolved: emit ONE violation per field, using the FIRST matching pattern in source order (PC01RNF-003 determinism). The unit tests assert this implicitly by counting violations. Blocking: No.
- **Q11 — Engine language-neutrality enforcement.** Pre-resolved: the §3.2 grep gate runs in the dev workflow before commit. The mention of `Entity` / `List` / `Set` is confined to fixtures, integration-test literal-substring assertions, and rule descriptions — never inside `internal/domain` or `internal/application`. Blocking: No.
- **Risk R1 — Tree-sitter inner-type capture.** The parser already populates `JavaField.Type` with the raw type token verbatim, including the `<...>` segment (verified against PC01US-002/003 fixtures that test `UserRepositoryJpa` and similar). No parser change is required. Mitigation: the unit tests in §7.2 are pure — they synthesize `JavaFileSummary` directly and prove the evaluator semantics independently of Tree-sitter. The integration tests in §7.3 then prove the end-to-end pipeline.
- **Risk R2 — `List<List<Entity>>` (nested generics).** The matcher's split on FIRST `<` / LAST `>` treats inner as `List<Entity>`. The glob `*Entity` matches that string. This is documented as a known false-positive vector under Q8; no AC exercises it; no fixture exercises it.

No `Blocking: Yes` entries. Discovery proceeds to implementation.

---

## Section 9 — Parallel Execution Plan (authoritative for `@agent-manager`)

```yaml
tiers:
  - id: 1
    name: Domain contract — new audit-rule kind, evaluator, helpers
    depends_on: []
    groups:
      - id: T1-G1
        scope:
          create: []
          modify:
            - internal/domain/model/auditRule.go
            - internal/domain/service/auditRuleEvaluator.go
        guidelines:
          - .claude/guidelines/domain-layer-guidelines.yml
        effort: M
        notes: >
          Adds AuditKindForbiddenFieldTypePattern, the
          evalForbiddenFieldTypePattern function, and the matchTypePattern +
          resolveFQN helpers. Switch arm appended to
          AuditEvaluator.EvaluateFile. Engine-neutrality grep gate
          (no Entity/List/Set/Spring/JUnit literals in modified files)
          MUST pass before this group is declared done.

  - id: 2
    name: Infrastructure — profile-loader kind whitelist
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
          One-line addition to knownAuditRuleKinds. No DTO change.
          No new port satisfied. Strictly mechanical follow-on to T1-G1.

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
          directly. Asserts literal substrings from AC2/AC3
          (type=java.util.List<OrderEntity>, matched_pattern=List<*Entity>).

      - id: T6-G2
        scope:
          create:
            - internal/cli/command/domainNoEntityCollectionIntegration_test.go
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: M
        notes: >
          Three test functions, t.Parallel(). Local helper
          newAuditCmdForDomainNoEntityCollection modelled on the
          PC01US-004 helper (no upstream DRY refactor). Loads each
          fixture via copyFixture, runs `audit` against the temp
          workdir, asserts on stdout.

      - id: T6-G3
        scope:
          create:
            - testdata/pc01us007DomainNoEntityCollection/projectClean/pom.xml
            - testdata/pc01us007DomainNoEntityCollection/projectClean/project-state.yaml
            - testdata/pc01us007DomainNoEntityCollection/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us007DomainNoEntityCollection/projectClean/src/main/java/com/acme/domain/Tag.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Clean fixture for AC1. Field `private List<String> tags;`.
          Profile contains the single domain-no-entity-collection rule
          with patterns "List<*Entity>,Set<*Entity>". testdata is
          gitignored — author force-adds when committing.

      - id: T6-G4
        scope:
          create:
            - testdata/pc01us007DomainNoEntityCollection/projectListEntity/pom.xml
            - testdata/pc01us007DomainNoEntityCollection/projectListEntity/project-state.yaml
            - testdata/pc01us007DomainNoEntityCollection/projectListEntity/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us007DomainNoEntityCollection/projectListEntity/src/main/java/com/acme/domain/Order.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Violating fixture for AC2 — line 9 holds
          `private List<OrderEntity> orders;`. The fixture must
          import `java.util.List` so resolveFQN produces
          "java.util.List".

      - id: T6-G5
        scope:
          create:
            - testdata/pc01us007DomainNoEntityCollection/projectSetEntity/pom.xml
            - testdata/pc01us007DomainNoEntityCollection/projectSetEntity/project-state.yaml
            - testdata/pc01us007DomainNoEntityCollection/projectSetEntity/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us007DomainNoEntityCollection/projectSetEntity/src/main/java/com/acme/domain/User.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Violating fixture for AC3 — line 11 holds
          `private Set<UserEntity> users;`. Imports `java.util.Set`.
```

---

## Self-Validation Checklist

**File-set coverage**
- Every file in §1 appears exactly once across §9 groups (cross-checked: T1-G1 has 2, T2-G1 has 1, T6-G1 has 1, T6-G2 has 1, T6-G3 has 4, T6-G4 has 4, T6-G5 has 4 — total 17, matching §1's 17 rows).
- Every requirement ID (PC01RF-005, PC01RF-009, PC01RNF-001, PC01RNF-003, PC01RNF-006) appears in at least one §1 row.
- No file path appears in two groups.

**Frozen contract**
- `AuditKindForbiddenFieldTypePattern` is scheduled in T1-G1 (modify auditRule.go).
- `evalForbiddenFieldTypePattern`, `matchTypePattern`, `resolveFQN` signatures match the §5/§7 narrative verbatim.
- `Deps` struct in `internal/cli/wire.go` is unchanged — explicitly noted.
- No fields marked `TODO` or `{placeholder}`.

**DAG**
- `depends_on` edges: T1-G1 → ∅; T2-G1 → [1]; T6-G1..G5 → [2]. Acyclic.
- Tier 1 exists because §1 has `internal/domain/**` modifications.
- Tier 5 omitted because no wiring file appears in §1.
- All `guidelines[]` paths exist under `/workspaces/jitctx/.claude/guidelines/`.

**Open questions**
- Zero `Blocking: Yes` entries. Discovery is unblocked.
