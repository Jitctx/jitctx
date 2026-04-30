# Plan — PC01US-010 Require @Primary together with @Qualifier(*) on Transactional Decorator

## Section 0 — Summary

- Feature: **assert that a transactional-decorator bean class declares
  both `@Primary` AND `@Qualifier(...)` AND that the `@Qualifier`
  argument is non-empty**, e.g. profile rule `tx-decorator-contract`
  requires `[Primary, Qualifier]` AND additionally constrains
  `Qualifier`'s positional argument to be non-empty. The check is
  language-neutral: the rule lives in profile YAML, the engine
  references no Java/Spring identifier.
- User Story: **PC01US-010**.
- Requirement IDs covered:
  - **PC01RF-001** — all-of presence semantics (existing
    `required_annotations` evaluator). Re-asserted by AC1 / AC2.
  - **PC01RF-007** — annotation-with-arguments matching. Existing
    `expected_values` enforces EXACT-string equality only; AC3
    requires a "non-empty" matcher that exact equality cannot express
    (the AC's expected value is "any non-empty string", not a fixed
    literal). PC01US-010 introduces a new sibling parameter
    `non_empty_value_annotations` on the existing
    `AuditKindRequiredAnnotations` kind. The new param coexists with
    `expected_values`; both can be used on the same rule.
  - **PC01RF-009** — evidence-rich messages. The new evidence shape
    `annotation=Qualifier, value=empty, expected=non-empty` is
    surfaced by the new evaluator branch.
  - **PC01RNF-001** — engine language-neutrality. The strings
    `Primary`, `Qualifier`, `Spring`, `Transactional`, `Decorator`,
    `Tx`, and the rule id `tx-decorator-contract` ONLY appear in
    (a) profile YAML under `testdata/`, (b) `.java` fixture files,
    (c) integration-test literal-substring assertions, and (d) rule
    descriptions inside the YAML profile. Zero new occurrences inside
    `internal/domain` or `internal/application`.
  - **PC01RNF-003** — deterministic output. The new branch iterates
    `non_empty_value_annotations` in input-string order; emit order
    is `missing-violation` FIRST, then per-pair `expected_values`
    mismatches in input order, then per-name `non_empty_value_annotations`
    empty-value violations in input order.
  - **PC01RNF-006** — real Tree-sitter parse on real `.java` fixtures
    via integration tests.
- Acceptance scenarios mapped 1:1 in §7:
  - **AC1** (clean) — class `OrderServiceTxDecorator` declares
    `@Primary @Qualifier("txDecorator")` → zero violations.
  - **AC2** (missing `@Qualifier`) — class declares `@Primary` only →
    one violation whose evidence contains the literal substring
    `missing=[Qualifier]`.
  - **AC3** (empty `@Qualifier` value) — class declares
    `@Primary @Qualifier("")` → one violation whose evidence contains
    the literal substring `annotation=Qualifier, value=empty,
    expected=non-empty`.
- Layers touched: **domain (Tier 1), tests (Tier 6)**. No
  infrastructure adapter, no application use case, no presentation
  command, no wiring change. The `knownAuditRuleKinds` whitelist in
  `internal/infrastructure/fsprofile/mapper.go` already accepts
  `AuditKindRequiredAnnotations`; the loader passes `params:
  map[string]string` through verbatim, so the new key
  `non_empty_value_annotations` rides through unchanged.
- Tiers active: **1, 6**. Tiers 2, 3, 4, 5 are explicitly `N/A`.
  - Tier 1 (domain) — minimal extension to `evalRequiredAnnotations`
    in `internal/domain/service/auditRuleEvaluator.go` to recognise
    the new optional parameter `non_empty_value_annotations`. No new
    `AuditKind` constant; no new model field; no new error sentinel;
    no new VO. The change is additive within an existing function and
    keeps every previously-frozen substitution token intact.
  - Tier 2 omitted — no infrastructure adapter changes (the
    `params: map[string]string` round-trip is verbatim and the
    profile DTO already preserves unknown keys).
  - Tier 3 omitted — `appaudituc.Impl.Execute` already iterates parsed
    files and dispatches to the evaluator.
  - Tier 4 omitted — no new cobra command, no new formatter.
  - Tier 5 omitted — `internal/cli/{wire,root,execute}.go`,
    `cmd/jitctx/main.go`, `internal/config/**` unchanged.
  - Tier 6 — three integration-test scenarios + three Java fixture
    trees, plus targeted unit tests on the new branch in
    `auditRuleEvaluator_test.go`.
- Guidelines loaded:
  - `.claude/guidelines/domain-layer-guidelines.yml`
  - `.claude/guidelines/integration-test-layer-guidelines.yml`
  - `.claude/guidelines/unit-test-layer-guidelines.yml`
- Estimated file count: **14 new** (1 integration test + 12 fixture
  files across 3 fixture trees) and **2 modified**
  (`auditRuleEvaluator.go` + `auditRuleEvaluator_test.go`). The plan
  file itself is not counted in §1.

> **Discovery finding (load-bearing).**
>
> 1. **Existing-kind reuse check.** Reading
>    `evalRequiredAnnotations` lines 331–394 of
>    `internal/domain/service/auditRuleEvaluator.go`:
>    - **Scenario 1 (pass)** — covered by the existing all-of
>      presence path. With `annotations: 'Primary,Qualifier'` and no
>      argument constraint, the missing-set is empty and the function
>      emits zero violations. ✅ no change needed.
>    - **Scenario 2 (missing)** — covered by the existing missing-set
>      path. With `annotations: 'Primary,Qualifier'` and a class that
>      declares only `@Primary`, `missingAnnotations` returns
>      `["Qualifier"]` and the violation carries the literal evidence
>      `missing=[Qualifier]`. ✅ no change needed.
>    - **Scenario 3 (empty value)** — **NOT** covered by the existing
>      `expected_values` path. The current code path
>      (`evalRequiredAnnotations` lines 372–391) compares
>      `decl.AnnotationArgs[ann]` against a FIXED literal via exact
>      string equality. The AC requires "any non-empty value" — the
>      `@Qualifier("txDecorator")` clean fixture must pass AND the
>      `@Qualifier("")` violating fixture must fail. A single fixed
>      `expected_values` entry cannot satisfy both because it would
>      either reject `"txDecorator"` (when set to anything except
>      `"txDecorator"`) or accept `""` (when set to `""` or absent).
>      The gap is real.
>
>    **Resolution: introduce a new optional sibling parameter
>    `non_empty_value_annotations`** on the existing
>    `AuditKindRequiredAnnotations` kind. Comma-joined list of
>    annotation simple names (without the leading `@`); for each
>    listed annotation that IS present on a matching declaration,
>    the evaluator checks whether `decl.AnnotationArgs[ann]` is
>    empty (no positional argument captured). When empty, it emits
>    one violation per offending name with evidence
>    `annotation=<ann>, value=empty, expected=non-empty`.
>    - This is **additive**: no existing parameter changes, no
>      existing evidence shape changes, no existing test breaks.
>    - It is **distinct** from `expected_values`: the latter does
>      EXACT equality and emits `expected_value=…, actual=…`; the
>      former does NON-EMPTY presence and emits `value=empty,
>      expected=non-empty`.
>    - It is **deterministic** (PC01RNF-003): the new slice is
>      iterated in input-string order, after `missing-violation` and
>      `expected_values` violations.
>    - It is **language-neutral** (PC01RNF-001): no Java/Spring
>      identifier appears in the evaluator function.
>
> 2. **Why a new key, not a sentinel value on `expected_values`?**
>    Two design alternatives were weighed (see §8 Q1):
>    - **(A) Sentinel value**: `expected_values: 'Qualifier=<non-empty>'`.
>      Pro: zero new key. Con: literal `<non-empty>` could collide
>      with a real expected value (a profile author who wanted to
>      assert `Qualifier="<non-empty>"` literally would be surprised);
>      requires touching the well-tested
>      `parseExpectedValues`/iteration code path AND the
>      `expected_value=…, actual=…` evidence shape; the AC's evidence
>      `value=empty, expected=non-empty` does NOT match the
>      `expected_value=…, actual=…` template, so the path would have
>      to branch internally on the sentinel anyway.
>    - **(B) New sibling param** `non_empty_value_annotations`.
>      Pro: orthogonal — keeps `expected_values` semantics frozen;
>      distinct evidence shape matches the AC verbatim; standard
>      PC01 extension pattern (PC01US-005/006/007/008 each added new
>      keys). Con: one new reserved key (24 → 25 in §2.5).
>
>    **Picked (B)** — same posture PC01US-006 / PC01US-008 took when
>    cleanly extending an existing kind. The §2.5 reserved-keys
>    registry grows by one.
>
> 3. **Profile YAML expressibility.** The rule
>    `tx-decorator-contract` is expressible using the existing kind
>    `required_annotations` PLUS the new key
>    `non_empty_value_annotations`:
>
>    ```yaml
>    audit_rules:
>      - id: tx-decorator-contract
>        kind: required_annotations
>        severity: ERROR
>        description: '{name}: {evidence}'
>        suggestion: 'Annotate {name} with @Primary AND @Qualifier("...") with a non-empty value'
>        params:
>          path_scope: src/main/java/com/acme/application/decorator/
>          annotations: 'Primary,Qualifier'
>          non_empty_value_annotations: 'Qualifier'
>    ```
>
>    No `name_pattern`, no `target` change, no `node_types` override
>    (default `class_declaration` matches). Profile authors targeting
>    decorator beans by class-name suffix can either narrow
>    `path_scope` (preferred) or wait for a future `name_suffix` /
>    `name_regex` extension on `required_annotations` (out of
>    scope; tracked as Q4).

---

## Section 1 — File Set

| #  | File                                                                                                                                       | Action  | Layer    | Tier | Group  | Requirements |
|----|--------------------------------------------------------------------------------------------------------------------------------------------|---------|----------|------|--------|--------------|
| 1  | `internal/domain/service/auditRuleEvaluator.go`                                                                                            | modify  | domain   | 1    | T1-G1  | PC01US-010, PC01RF-007, PC01RF-009, PC01RNF-001, PC01RNF-003 |
| 2  | `internal/domain/service/auditRuleEvaluator_test.go`                                                                                       | modify  | tests    | 6    | T6-G1  | PC01US-010, PC01RF-007, PC01RF-009, PC01RNF-003 |
| 3  | `internal/cli/command/txDecoratorContractIntegration_test.go`                                                                              | create  | tests    | 6    | T6-G2  | PC01US-010, PC01RF-001, PC01RF-007, PC01RF-009, PC01RNF-006 |
| 4  | `testdata/pc01us010TxDecoratorContract/projectClean/pom.xml`                                                                               | create  | tests    | 6    | T6-G3  | PC01RNF-006 |
| 5  | `testdata/pc01us010TxDecoratorContract/projectClean/project-state.yaml`                                                                    | create  | tests    | 6    | T6-G3  | PC01RNF-006 |
| 6  | `testdata/pc01us010TxDecoratorContract/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml`                                           | create  | tests    | 6    | T6-G3  | PC01US-010, PC01RF-001, PC01RF-007 |
| 7  | `testdata/pc01us010TxDecoratorContract/projectClean/src/main/java/com/acme/application/decorator/OrderServiceTxDecorator.java`             | create  | tests    | 6    | T6-G3  | PC01US-010 |
| 8  | `testdata/pc01us010TxDecoratorContract/projectMissingQualifier/pom.xml`                                                                    | create  | tests    | 6    | T6-G4  | PC01RNF-006 |
| 9  | `testdata/pc01us010TxDecoratorContract/projectMissingQualifier/project-state.yaml`                                                         | create  | tests    | 6    | T6-G4  | PC01RNF-006 |
| 10 | `testdata/pc01us010TxDecoratorContract/projectMissingQualifier/.jitctx/profiles/spring-boot-hexagonal.yaml`                                | create  | tests    | 6    | T6-G4  | PC01US-010, PC01RF-001 |
| 11 | `testdata/pc01us010TxDecoratorContract/projectMissingQualifier/src/main/java/com/acme/application/decorator/OrderServiceTxDecorator.java` | create  | tests    | 6    | T6-G4  | PC01US-010, PC01RF-009 |
| 12 | `testdata/pc01us010TxDecoratorContract/projectEmptyQualifier/pom.xml`                                                                      | create  | tests    | 6    | T6-G5  | PC01RNF-006 |
| 13 | `testdata/pc01us010TxDecoratorContract/projectEmptyQualifier/project-state.yaml`                                                           | create  | tests    | 6    | T6-G5  | PC01RNF-006 |
| 14 | `testdata/pc01us010TxDecoratorContract/projectEmptyQualifier/.jitctx/profiles/spring-boot-hexagonal.yaml`                                  | create  | tests    | 6    | T6-G5  | PC01US-010, PC01RF-007 |
| 15 | `testdata/pc01us010TxDecoratorContract/projectEmptyQualifier/src/main/java/com/acme/application/decorator/OrderServiceTxDecorator.java`   | create  | tests    | 6    | T6-G5  | PC01US-010, PC01RF-009 |

Coverage notes:

- File #1 (`auditRuleEvaluator.go`) hosts the additive evaluator
  branch documented in §3.1. The change is additive within
  `evalRequiredAnnotations`; no other function is touched.
- File #2 (`auditRuleEvaluator_test.go`) gains four targeted unit
  cases documented in §7.2. The existing PC01US-006 cases are
  unchanged.
- File #3 (the integration test) loads each of the three project
  trees via `copyFixture` from `helpers_test.go`, runs `audit`
  against the temp workdir, and asserts on stdout. Each fixture tree
  has identical shape: `pom.xml` (Spring Boot detector trigger),
  `project-state.yaml` (schema_version 2, single module),
  `.jitctx/profiles/spring-boot-hexagonal.yaml` (canonical FULL
  profile shape — see §7.5), and ONE
  `OrderServiceTxDecorator.java` under
  `src/main/java/com/acme/application/decorator/`.
- Three fixture trees, each in its own group (T6-G3..G5), so the
  fixture-authoring work is parallelisable and conflict-free. The
  test file (T6-G2) is independent of the fixture content — it can
  be authored in parallel with the fixtures because it only
  references fixture paths, never their text content. The unit-test
  edit (T6-G1) is independent of the integration test (T6-G2) and
  the fixtures, but it DOES depend on Tier 1 publishing the new
  branch contract.

Requirement coverage trace (every ID in scope appears below):

| Requirement   | Where it lives in code                                                                                                              | Where this plan re-asserts it                       |
|---------------|--------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------------------|
| PC01US-010    | T1-G1 publishes the new `non_empty_value_annotations` branch in `evalRequiredAnnotations`                                            | T6-G1 unit cases + T6-G2 three integration scenarios |
| PC01RF-001    | `missingAnnotations` helper produces deterministic `missing=[...]` subset (existing — unchanged)                                     | T6-G3 (clean, both present) + T6-G4 (missing)        |
| PC01RF-007    | New non-empty-value branch covers the "any non-empty" matcher absent from `expected_values`'s exact-equality semantics               | T6-G3 (`txDecorator` accepted) + T6-G5 (empty fails) |
| PC01RF-009    | `makeViolation` substitution context emits the new literal `annotation=<ann>, value=empty, expected=non-empty` for the new branch    | T6-G2 stdout substring assertions + T6-G1 unit cases |
| PC01RNF-001   | grep audit (no Primary/Qualifier/Spring/Transactional literals in `internal/domain` or `internal/application`)                       | §7.6 grep gate (no test, no fixture)                 |
| PC01RNF-003   | New branch iterates `splitNonEmpty(params["non_empty_value_annotations"])` in input-string order, after missing + expected_values    | T6-G1 ordering case + T6-G2 fixed-fixture determinism|
| PC01RNF-006   | real Tree-sitter parse via existing `treesitter.Parser` adapter                                                                      | T6-G3/G4/G5 real `.java` fixtures                    |

---

## Section 2 — Frozen Domain Contract

This section freezes the surface that downstream tiers (T6-G1
unit tests, T6-G2 integration tests, T6-G3..G5 fixtures) consume.
PC01US-010 introduces NO new ports, NO new model types, NO new
use-case interfaces, NO new error sentinels, and NO new
`AuditRuleKind` constant. The single addition is one new
*parameter key* on the existing `AuditKindRequiredAnnotations`
kind, plus the corresponding evaluator branch.

### 2.1 `model.AuditKindRequiredAnnotations` (frozen, unchanged)

```go
// internal/domain/model/auditRule.go (existing — DO NOT MODIFY)

// AuditKindRequiredAnnotations enforces that every declaration in the
// rule's path_scope carries every annotation in params["annotations"].
// PC01US-002 introduced the kind; PC01US-006 added the
// expected_values parameter that produces argument-mismatch evidence
// per PC01RF-007. PC01US-010 adds the optional sibling parameter
// non_empty_value_annotations (consumed by the same evaluator
// function) that produces "annotation=<ann>, value=empty,
// expected=non-empty" evidence.
AuditKindRequiredAnnotations AuditRuleKind = "required_annotations"
```

The constant value is unchanged. The infrastructure mapper's
`knownAuditRuleKinds` whitelist already accepts it. No new constant
is added.

### 2.2 `evalRequiredAnnotations` parameter contract

The parameter keys consumed by `evalRequiredAnnotations` (in
`internal/domain/service/auditRuleEvaluator.go`) after PC01US-010:

| Key                            | Required? | Origin     | Semantics |
|--------------------------------|-----------|------------|-----------|
| `path_scope`                   | yes       | PC01US-002 | substring filter on `summary.Path` (e.g. `src/main/java/com/acme/application/decorator/`) |
| `annotations`                  | yes       | PC01US-002 | comma-joined simple names (no leading `@`) that must ALL be present on every matching declaration; order is preserved and used to derive deterministic `missing=[...]` evidence |
| `expected_values`              | no        | PC01US-006 | comma-joined `Annotation=Value` pairs; for each pair, when the annotation is present on a matching declaration, the evaluator compares `decl.AnnotationArgs[ann]` against the value via EXACT string equality. A mismatch emits ONE additional violation per pair with evidence `annotation=<ann>, expected_value=<expected>, actual=<actual>`. Determinism: pairs iterated in input-string order. |
| `non_empty_value_annotations`  | no        | PC01US-010 (NEW) | comma-joined annotation simple names (no leading `@`); for each listed annotation that IS present on a matching declaration, the evaluator checks whether `decl.AnnotationArgs[ann]` is the empty string. When empty, it emits ONE additional violation per offending name with evidence `annotation=<ann>, value=empty, expected=non-empty`. Determinism: names iterated in input-string order via `splitNonEmpty`. Names NOT also in `params["annotations"]` are silently ignored (the check is conditional on presence; a name not in `annotations` and not present on the declaration produces no violation, by design — the missing-set path covers that case). Names that ARE in `params["annotations"]` BUT absent from the declaration are short-circuited by the missing-violation path and the new branch never fires for them. |
| `node_types`                   | no        | PC01US-002 | comma-joined declaration node-type filter; default `class_declaration`; `*` wildcards |

### 2.3 New evaluator branch — frozen behaviour

The branch is appended to `evalRequiredAnnotations` AFTER the
existing `expected_values` loop. Order of emission per declaration:

1. `missing-violation` (if any required annotation is absent).
2. `expected_values` mismatch violations (for each pair where the
   annotation IS present and `decl.AnnotationArgs[ann] !=
   pair.Expected`), in `expected_values` input-string order.
3. **NEW** `non_empty_value_annotations` empty-value violations (for
   each listed name that IS present on the declaration AND
   `decl.AnnotationArgs[ann] == ""`), in
   `non_empty_value_annotations` input-string order.

A single declaration may produce multiple violations across the three
paths simultaneously; the order above is fixed by code order (no Go
map iteration anywhere). PC01RNF-003 holds.

### 2.4 Substitution context for the new branch

The new branch populates the same `makeViolation` substitution
tokens as the other paths. For the **non-empty-value violation**
path:

| Token        | Value (verbatim) |
|--------------|------------------|
| `{file}`     | `summary.Path`   |
| `{name}`     | declaration simple name |
| `{required}` | comma-joined `params["annotations"]` |
| `{evidence}` | literal `annotation=<ann>, value=empty, expected=non-empty` where `<ann>` is the offending annotation simple name |

The `{missing}` token is NOT populated for this path (only the
missing-violation path populates it). The renderer
(`internal/cli/format/audit.go`) leaves unknown tokens as-is, so a
profile that uses `{missing}` in the description while only the
non-empty-value path fires will render the literal `{missing}` token
— this is acceptable and matches the documented PC01US-006 posture.
Profile authors using the `tx-decorator-contract` rule should use
`{evidence}` (which is populated for ALL three paths) in the
description template.

### 2.5 Reserved param-keys registry after PC01US-010

| Key                              | Used by (kinds)                                                                                          | Origin     |
|----------------------------------|----------------------------------------------------------------------------------------------------------|------------|
| `path_scope`                     | forbidden_import, field_type_layer_violation, required_annotations, forbidden_annotations, method_naming, forbidden_field_type_pattern, required_parameterized_supertype | PC01US-002 |
| `annotations`                    | required_annotations, forbidden_annotations                                                              | PC01US-002 |
| `expected_values`                | required_annotations                                                                                     | PC01US-006 |
| `non_empty_value_annotations`    | required_annotations                                                                                     | **PC01US-010 (NEW)** |
| `node_types`                     | required_annotations, forbidden_annotations, method_naming, forbidden_field_type_pattern, required_parameterized_supertype | PC01US-002 |
| `target`                         | forbidden_annotations                                                                                    | PC01US-004 |
| `exempt_paths`                   | forbidden_annotations, method_naming, forbidden_field_type_pattern, required_parameterized_supertype     | PC01US-004 |
| `triggered_by`                   | method_naming                                                                                            | PC01US-005 |
| `name_pattern`                   | method_naming                                                                                            | PC01US-005 |
| `forbidden_type_patterns`        | forbidden_field_type_pattern                                                                             | PC01US-007 |
| `expected_supertype`             | required_parameterized_supertype                                                                         | PC01US-008 |
| `args`                           | required_parameterized_supertype                                                                         | PC01US-008 |
| `supertype_kind`                 | required_parameterized_supertype                                                                         | PC01US-008 |
| `path_required`                  | annotation_path_mismatch, interface_naming                                                               | EP-04 era  |
| `path_required_any`              | implements_path_mismatch                                                                                 | EP-04 era  |
| `name_suffix`                    | interface_naming                                                                                         | EP-04 era  |
| `name_regex`                     | interface_naming                                                                                         | EP-04 era  |
| `forbidden_type_suffix`          | field_type_layer_violation                                                                               | EP-04 era  |
| `forbidden_type_substring`       | field_type_layer_violation                                                                               | EP-04 era  |
| `import_prefix`                  | forbidden_import                                                                                         | EP-04 era  |
| `implements_glob`                | implements_path_mismatch                                                                                 | EP-04 era  |
| `annotation`                     | annotation_path_mismatch                                                                                 | EP-04 era  |

PC01US-010 adds **exactly one** new key:
`non_empty_value_annotations`. The §8 Q1 rationale for picking a new
key over a sentinel value is preserved here.

### 2.6 No changes to other contracts

- `model.JavaDeclaration`, `model.AuditRule`, `model.JavaField`,
  `model.JavaMethod`, `auditvo.AuditViolation` — all unchanged.
- `internal/cli/wire.go` `Deps` struct — unchanged. The audit use
  case already injects `service.AuditEvaluator{}`.
- No new error sentinels, no new typed errors.
- The bundled `spring-boot-hexagonal` profile is **NOT** modified by
  this story (same posture as
  PC01US-002/004/005/006/007/008/009). Profile content evolves
  under EP-04. Profile authors enable the rule by editing their own
  `.jitctx/profiles/*.yaml`.
- `internal/infrastructure/fsprofile/mapper.go` `knownAuditRuleKinds`
  — unchanged. `AuditKindRequiredAnnotations` is already
  whitelisted; `params: map[string]string` rides through verbatim
  including the new key.
- `internal/infrastructure/treesitter/parser.go` — unchanged. The
  parser already populates `decl.AnnotationArgs[ann] == ""` for
  marker annotations (proven by
  `TestParser_ClassWithMarkerAnnotation_AnnotationArgEmpty`). For
  `@Qualifier("")`, the captured text is the two characters `""`
  (the empty string literal, INCLUDING the surrounding double
  quotes — the parser preserves quotes verbatim per PC01US-006). To
  detect "empty value", the new branch tests against the literal
  Go strings `""` AND `"\"\""` (i.e. four characters: open quote,
  empty body, close quote — the canonical Java empty-string-literal
  capture). See §3.1 for the exact predicate; this is the load-
  bearing behaviour the unit tests in §7.2 lock in.

---

## Section 3 — Domain Layer Plan

### 3.1 Edit to `internal/domain/service/auditRuleEvaluator.go` (T1-G1)

**Function**: `evalRequiredAnnotations` — additive branch, no
changes to existing branches.

The new branch is appended after the existing `expected_values`
loop and before the function's final `return violations`. Pseudocode
in the style of the surrounding code:

```go
// (existing) missing-violation emission — UNCHANGED
// (existing) expected_values mismatch loop — UNCHANGED

// PC01US-010: non-empty-value branch.
// For every annotation listed in non_empty_value_annotations that
// IS present on this declaration, emit one violation if its
// captured argument text is empty.
//
// Determinism: input-string order via splitNonEmpty.
// Short-circuit: an annotation absent from decl.Annotations is
// covered by the missing-violation path; this branch only fires
// for annotations actually present whose captured arg text is
// considered "empty" by isEmptyAnnotationArg.
for _, ann := range splitNonEmpty(rule.Params["non_empty_value_annotations"]) {
    if !slices.Contains(decl.Annotations, ann) {
        continue
    }
    actual := decl.AnnotationArgs[ann]
    if !isEmptyAnnotationArg(actual) {
        continue
    }
    evidence := "annotation=" + ann + ", value=empty, expected=non-empty"
    ctxNe := map[string]string{
        "file":     summary.Path,
        "name":     decl.Name,
        "required": strings.Join(required, ","),
        "evidence": evidence,
    }
    violations = append(violations, makeViolation(moduleID, summary, rule, 0, ctxNe))
}
```

**New private helper** `isEmptyAnnotationArg` placed adjacent to
`parseExpectedValues` in the same file. It is the load-bearing
predicate; its contract is fixed by the parser's quoting behaviour
documented in §2.6 and Q3.

```go
// isEmptyAnnotationArg reports whether the captured annotation-
// argument text represents an "empty" value, per PC01RF-007. The
// Tree-sitter parser stores annotation arguments VERBATIM, including
// surrounding quotes for string literals. Therefore three forms map
// to "empty":
//
//   ""       — marker annotation (no argument list captured); the
//              parser leaves the entry as the empty string.
//   "\"\""   — explicit empty string literal @Ann("").
//   "''"     — explicit empty char/string literal @Ann('') (Java
//              char literal; defensive — most JVM toolchains reject
//              this at compile time, but the parser would still
//              capture it verbatim).
//
// All other captures (including whitespace-only such as "\" \"" or
// "0" or "false") are treated as NON-empty: the predicate is
// strictly about the parser-captured text, not about semantic
// emptiness. Profile authors needing semantic emptiness can layer
// expected_values on top.
//
// PC01RF-007 (annotation argument matching), PC01RNF-001 (no
// language identifier in this function — only verbatim parser
// output is matched), PC01RNF-003 (deterministic — pure string
// comparison).
func isEmptyAnnotationArg(captured string) bool {
    switch captured {
    case "", `""`, `''`:
        return true
    }
    return false
}
```

**Doc-comment update** on `evalRequiredAnnotations`: extend the
existing parameter list to document `non_empty_value_annotations`
verbatim (mirror the table in §2.2). Update the
"Substitution context" block to document the third evidence shape.
Update the "Determinism (PC01RNF-003)" block to document the
per-declaration emit order: missing → expected_values → non-empty.

**No other domain file changes.** No new model types, no new ports,
no new VOs, no new use cases, no new errors. The `JavaDeclaration`
already has the `AnnotationArgs map[string]string` field needed
(populated by PC01US-006).

### 3.2 Engine-neutrality posture (PC01RNF-001)

The new code in §3.1 references zero Java/Spring identifiers. The
strings `Primary`, `Qualifier`, `Transactional`, `Decorator`, `Tx`,
`Spring`, and the rule id `tx-decorator-contract` MUST NOT appear
inside `internal/domain/service/auditRuleEvaluator.go` after this
change. The §7.6 grep gate enforces this.

---

## Section 4 — Infrastructure Layer Plan

**N/A.** No infrastructure adapter is added or modified.

- `internal/infrastructure/fsprofile/mapper.go`
  `knownAuditRuleKinds` already whitelists
  `AuditKindRequiredAnnotations`.
- `internal/infrastructure/fsprofile/auditRulesDTO` (or its
  equivalent — the DTO carrying `params: map[string]string`) passes
  unknown keys through verbatim, so `non_empty_value_annotations`
  rides through unchanged. No DTO field addition.
- `internal/infrastructure/treesitter/parser.go` is unchanged. The
  parser's existing behaviour for annotation argument capture
  (verbatim text including quotes; empty string for marker
  annotations) is the load-bearing input to the new branch — see
  §2.6 / §3.1.

---

## Section 5 — Application Layer Plan

**N/A.** `appaudituc.Impl.Execute` already iterates parsed files,
calls `AuditEvaluator.EvaluateFile`, sorts the union via the
existing `AuditRuleFilter`, and emits the deterministic
`AuditProjectOutput`. No edit required. The new evaluator branch is
internal to the existing function `evalRequiredAnnotations`; it
emits additional `auditvo.AuditViolation` values via the same
`makeViolation` helper.

---

## Section 6 — Presentation Layer Plan

**N/A.** No new cobra command, no formatter change. The `audit`
command already prints violations via the existing renderer
(`internal/cli/format/audit.go`), which substitutes `{file}`,
`{name}`, `{required}`, and `{evidence}` tokens through
`makeViolation` → `substituteSuggestion`. The stdout/stderr contract
is unchanged: violations render under `## Sintatic Violations`; the
rule ID is emitted as the literal `[tx-decorator-contract]` token in
each violation line; the integration tests assert on those
substrings.

---

## Section 7 — Composition Root + Tests Plan

### 7.1 Composition root

**N/A.** `internal/cli/wire.go`, `root.go`, `execute.go`,
`cmd/jitctx/main.go`, and `internal/config/**` are all unchanged.
The `Deps` struct is unchanged.

### 7.2 Unit tests (T6-G1 — `internal/domain/service/auditRuleEvaluator_test.go`)

Four new table-style cases appended to the existing
`required_annotations` block (lines 1187+ in current file). Each
case uses the existing `newEvaluator()` helper, the existing
`testModuleID` constant, and constructs the rule + summary inline
(matching the surrounding style — see lines 1195–1230 for
template).

The new cases reference the rule via a local helper
`txDecoratorContractRule()` (camelCase, file-scoped) modelled on
`unitTestClassContractRule` at line 1195. The helper returns:

```go
func txDecoratorContractRule() model.AuditRule {
    return model.AuditRule{
        ID:          "tx-decorator-contract",
        Kind:        model.AuditKindRequiredAnnotations,
        Severity:    model.AuditSeverityError,
        Description: "{name}: {evidence}",
        Suggestion:  "Apply the contract to {name}",
        Params: map[string]string{
            "path_scope":                  "src/main/java/",
            "annotations":                 "Primary,Qualifier",
            "non_empty_value_annotations": "Qualifier",
            "node_types":                  "class_declaration",
        },
    }
}
```

This helper carries the strings `Primary`, `Qualifier`, and
`tx-decorator-contract` — but it lives in
`internal/domain/service/auditRuleEvaluator_test.go`, which is a
TEST file inside the domain package. PC01RNF-001 explicitly excludes
test files and `testdata/**` from the proscribed-string set (the
gate scope is "engine-neutral domain CODE", not domain TEST code —
same posture PC01US-006 took for `Mockito` / `ExtendWith` literals
in the same file).

#### Case 1 — `TestAuditEvaluator_RequiredAnnotations_TxDecorator_PrimaryAndQualifierWithNonEmptyValuePass`

- Rule: `txDecoratorContractRule()`.
- Summary: a `class_declaration` named `OrderServiceTxDecorator` at
  path `src/main/java/com/acme/application/decorator/OrderServiceTxDecorator.java`,
  annotations `["Primary", "Qualifier"]`, `AnnotationArgs:
  {"Primary": "", "Qualifier": "\"txDecorator\""}`.
- Expectation: `require.Empty(t, got)` — zero violations. Maps to AC1.

#### Case 2 — `TestAuditEvaluator_RequiredAnnotations_TxDecorator_PrimaryOnlyEmitsMissingQualifier`

- Rule: `txDecoratorContractRule()`.
- Summary: same path/name as case 1, annotations `["Primary"]`,
  `AnnotationArgs: {"Primary": ""}`.
- Expectation: `require.Len(t, got, 1)` and
  `require.Contains(t, got[0].Message, "missing=[Qualifier]")`.
  Maps to AC2.

#### Case 3 — `TestAuditEvaluator_RequiredAnnotations_TxDecorator_EmptyQualifierValueEmitsNonEmptyEvidence`

- Rule: `txDecoratorContractRule()`.
- Summary: same path/name as case 1, annotations
  `["Primary", "Qualifier"]`, `AnnotationArgs: {"Primary": "",
  "Qualifier": "\"\""}` (the four-character capture `""` —
  open-quote, empty body, close-quote).
- Expectation: `require.Len(t, got, 1)` and
  `require.Contains(t, got[0].Message, "annotation=Qualifier,
  value=empty, expected=non-empty")`. Maps to AC3.

#### Case 4 — `TestAuditEvaluator_RequiredAnnotations_TxDecorator_OrderingMissingThenExpectedValuesThenNonEmpty`

- This case locks in the §2.3 emit order. Rule has BOTH
  `expected_values: 'Primary=somevalue'` AND
  `non_empty_value_annotations: 'Qualifier'`, plus
  `annotations: 'Primary,Qualifier,Other'`.
- Summary declares `["Primary", "Qualifier"]` (so `Other` is
  missing), `AnnotationArgs: {"Primary": "wrongvalue", "Qualifier":
  "\"\""}`.
- Expectation: `require.Len(t, got, 3)`.
  - `got[0].Message` contains `missing=[Other]`.
  - `got[1].Message` contains `expected_value=somevalue,
    actual=wrongvalue`.
  - `got[2].Message` contains `annotation=Qualifier, value=empty,
    expected=non-empty`.
  - Maps to PC01RNF-003 ordering.

A separate sub-test or nested helper covers
`isEmptyAnnotationArg` directly with the three accepted empty
forms (`""`, `"\"\""`, `"''"`) and three rejected non-empty forms
(`"x"`, `"\"x\""`, `"\" \""`). Inline this as a `t.Run` block in
case 3 OR as a stand-alone case 5 — author's choice (no plan-level
constraint).

### 7.3 Parser unit tests

**N/A.** `internal/infrastructure/treesitter/parser_test.go`
already proves:

- `decl.AnnotationArgs[ann]` is `""` for marker annotations
  (`TestParser_ClassWithMarkerAnnotation_AnnotationArgEmpty`,
  line 346 of the current file).
- `decl.AnnotationArgs[ann]` includes surrounding quotes for string
  literals (`TestParser_ClassWithExtendWithArg_PopulatesAnnotationArgs`,
  line 318; asserts `"User service tests"` for
  `@DisplayName("User service tests")`).

These two existing test cases together pin the input-side contract
that PC01US-010's `isEmptyAnnotationArg` predicate consumes. No new
parser test is needed.

### 7.4 Integration tests (T6-G2 — `internal/cli/command/txDecoratorContractIntegration_test.go`)

Three test functions, each:

- `t.Parallel()`.
- Builds a real `audit` cobra command via a local helper modelled on
  `newAuditCmdForIntegrationTestBaseRequiredAnnotations` (Q-DRY: a
  local copy is acceptable per the no-upstream-refactor rule
  established by PC01US-007 / PC01US-008 / PC01US-009). The helper
  wires:
  - `fsprofile.NewDetectorWithLogger(profilesDir, logger)`
  - `fsprofile.NewAuditRulesLoader(profilesDir, logger)`
  - `fsmanifest.New(manifestPath)`
  - `treesitter.New()` for both `JavaParser` injection points
  - `treesitter.NewWalker()`
  - `service.NewAuditEvaluator()`
  - `fsconfig.New(logger)`
  - `service.NewAuditRuleFilter()`
  - `fsprofile.NewBundleAuditRulesAdapter()`,
    `fsprofile.NewBundled()`,
    `fsprofile.NewBundleLoader(logger, nil)`,
    `fsprofile.NewResolver(...)`
- Uses `t.TempDir()` + `copyFixture` from `helpers_test.go`.
- Asserts on `stdout` via `require.Contains` and
  `strings.Count(stdout, "[tx-decorator-contract]")`.

The shared local helper is `newAuditCmdForTxDecoratorContract`
(camelCase per project filename convention).

#### Test 1 — `TestAuditCmd_Integration_TxDecoratorContract_PrimaryAndQualifierWithNonEmptyValuePass` (AC1)

- Fixture: `pc01us010TxDecoratorContract/projectClean`.
- Manifest path: `<tempDir>/project-state.yaml`.
- Command: `audit`.
- Expectations:
  - `err == nil` (the audit command exits zero on a clean project per
    EP03US-002 contract).
  - `stdout` does NOT contain `[tx-decorator-contract]`.
  - `stdout` does NOT contain `missing=`.
  - `stdout` does NOT contain `value=empty`.

Maps to AC1.

#### Test 2 — `TestAuditCmd_Integration_TxDecoratorContract_PrimaryOnlyEmitsMissingQualifier` (AC2)

- Fixture: `pc01us010TxDecoratorContract/projectMissingQualifier`.
- Expectations:
  - `stdout` contains `[tx-decorator-contract]`.
  - `strings.Count(stdout, "[tx-decorator-contract]") == 1`.
  - `stdout` contains the literal substring `missing=[Qualifier]`.
  - `stdout` does NOT contain `value=empty` (the new branch is
    short-circuited because `Qualifier` is absent — covered by the
    missing-violation path).

Maps to AC2.

#### Test 3 — `TestAuditCmd_Integration_TxDecoratorContract_EmptyQualifierValueEmitsNonEmptyEvidence` (AC3)

- Fixture: `pc01us010TxDecoratorContract/projectEmptyQualifier`.
- Expectations:
  - `stdout` contains `[tx-decorator-contract]`.
  - `strings.Count(stdout, "[tx-decorator-contract]") == 1`.
  - `stdout` contains the literal substring
    `annotation=Qualifier, value=empty, expected=non-empty`.
  - `stdout` does NOT contain `missing=` (both required
    annotations are present).

Maps to AC3.

### 7.5 Fixtures (T6-G3, T6-G4, T6-G5 — three trees under `testdata/pc01us010TxDecoratorContract/`)

Naming convention follows PC01US-007 / PC01US-008 / PC01US-009:
lower-camelCase project root segments matching
`pc01us010TxDecoratorContract`. testdata is gitignored (project
convention); the integration test author force-adds with `git add
-f` when committing.

Each project tree has the same shape:

```
projectXxx/
├── pom.xml                         # contains org.springframework.boot for module detection
├── project-state.yaml              # schema_version: 2; one module; one file
├── .jitctx/
│   └── profiles/
│       └── spring-boot-hexagonal.yaml   # ONE audit rule: tx-decorator-contract (PLUS the FULL canonical profile)
└── src/
    └── main/
        └── java/
            └── com/acme/application/decorator/
                └── OrderServiceTxDecorator.java
```

**Critical: the profile YAML MUST carry the FULL canonical
`spring-boot-hexagonal` shape** — `name`, `languages`,
`query_lang`, `detect`, `module_detection`, `rules` (the
classification rules), AND `audit_rules`. A profile with ONLY
`audit_rules` will not be activated by the auto-detector — the
detector reads `detect.files` to decide whether the profile applies
to the project, and the resolver requires `name`, `languages`, and
`query_lang`. This is the same fixture-content requirement that
PC01US-007 / PC01US-008 / PC01US-009 documented; copy the canonical
shape from `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml`
verbatim, replacing only the `audit_rules` block.

The full canonical profile, with the PC01US-010 `audit_rules`
payload, is identical across the three fixtures. The canonical YAML
body the fixtures share:

```yaml
name: spring-boot-hexagonal
languages: [java]
query_lang: java

# Auto-detection.
detect:
  files:
    - name: pom.xml
      contains: "org.springframework.boot"
    - name: build.gradle
      contains: "org.springframework.boot"
    - name: build.gradle.kts
      contains: "org.springframework.boot"

module_detection:
  strategy: hexagonal
  roots:
    - src/main/java/**/domain
    - src/main/java/**
  markers:
    - kind: path_contains
      value: /port/in/
    - kind: path_contains
      value: /port/out/
    - kind: annotation
      value: Entity

rules:
  - match:
      node_type: interface_declaration
      path_contains: /port/in/
    classify_as: input-port
  - match:
      node_type: interface_declaration
      path_contains: /port/out/
    classify_as: output-port
  - match:
      node_type: class_declaration
      has_annotation: Entity
    classify_as: entity
  - match:
      node_type: class_declaration
      has_annotation: RestController
    classify_as: rest-adapter
  - match:
      node_type: class_declaration
      has_annotation: Repository
    classify_as: jpa-adapter
  - match:
      node_type: class_declaration
      implements: "*UseCase"
      path_contains: /service/
    classify_as: service
  - match:
      node_type: class_declaration
      implements: "*UseCase"
      path_contains: /application/
    classify_as: service
  - match:
      node_type: class_declaration
      has_annotation: Service
    classify_as: service

audit_rules:
  - id: tx-decorator-contract
    kind: required_annotations
    severity: ERROR
    description: '{name}: {evidence}'
    suggestion: 'Annotate {name} with @Primary AND @Qualifier("...") with a non-empty value'
    params:
      path_scope: src/main/java/com/acme/application/decorator/
      annotations: 'Primary,Qualifier'
      non_empty_value_annotations: 'Qualifier'
```

Notes on the audit rule:

- `path_scope: src/main/java/com/acme/application/decorator/` —
  substring filter that matches the ONE decorator-class file in
  each fixture.
- `annotations: 'Primary,Qualifier'` — comma-joined required-set in
  declaration order. The all-of presence path uses this directly.
- `non_empty_value_annotations: 'Qualifier'` — a single annotation
  name; the new branch fires when `Qualifier` IS present on the
  declaration AND its captured argument text is empty.
- `description` template uses `{name}` and `{evidence}`. The
  `{evidence}` token resolves to one of the three shapes documented
  in §2.3 / §2.4 depending on which path emits the violation.

`pom.xml` content (minimum to satisfy module-detection — copy the
shape used by PC01US-009 fixtures):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.acme</groupId>
  <artifactId>pc01us010</artifactId>
  <version>0.0.1-SNAPSHOT</version>
  <parent>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-parent</artifactId>
    <version>3.2.0</version>
  </parent>
</project>
```

`project-state.yaml` skeleton (`schema_version: 2`, one module, one
file — copy shape from `pc01us009IntegrationTestBaseRequiredAnnotations`
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
  - id: com.acme.application.decorator
    path: src/main/java/com/acme/application/decorator
    tags: []
    contracts:
      - name: OrderServiceTxDecorator
        types:
          - service
        path: src/main/java/com/acme/application/decorator/OrderServiceTxDecorator.java
        methods: []
    dependencies: []
contexts: []
```

The `contracts[].types: [service]` value is a placeholder used to
make the manifest pass schema validation; the audit use case does
NOT use the `types` field for `required_annotations` evaluation
(it iterates parsed files via the walker). Same trick used in
PC01US-007 / PC01US-008 / PC01US-009 fixtures.

#### `.java` content per fixture

**projectClean / OrderServiceTxDecorator.java** (AC1 — passing):

```java
package com.acme.application.decorator;

import org.springframework.context.annotation.Primary;
import org.springframework.beans.factory.annotation.Qualifier;

@Primary
@Qualifier("txDecorator")
public class OrderServiceTxDecorator {
}
```

The class has no body methods or fields — the rule scope is the
class declaration's annotations.

**projectMissingQualifier / OrderServiceTxDecorator.java** (AC2 — missing annotation):

```java
package com.acme.application.decorator;

import org.springframework.context.annotation.Primary;

@Primary
public class OrderServiceTxDecorator {
}
```

`@Qualifier` is intentionally omitted (and its import). The
missing-violation path emits exactly one violation with evidence
`missing=[Qualifier]`. The non-empty-value path is short-circuited
because `Qualifier` is absent. Total violations on this fixture:
exactly one.

**projectEmptyQualifier / OrderServiceTxDecorator.java** (AC3 — empty value):

```java
package com.acme.application.decorator;

import org.springframework.context.annotation.Primary;
import org.springframework.beans.factory.annotation.Qualifier;

@Primary
@Qualifier("")
public class OrderServiceTxDecorator {
}
```

Both required annotations are present; `Qualifier`'s argument is
the empty string literal `""` (parser captures the verbatim text
`""`, which `isEmptyAnnotationArg` treats as empty). The missing-
violation path is short-circuited (nothing missing); the new
non-empty-value path emits exactly one violation with evidence
`annotation=Qualifier, value=empty, expected=non-empty`. Total
violations on this fixture: exactly one.

The `path_scope` substring `src/main/java/com/acme/application/decorator/`
matches all three fixtures' `.java` paths. The walker emits forward-
slash paths already (proven by EP-01 / EP-03 integration tests).

### 7.6 Engine-neutrality grep gate (PC01RNF-001)

Before declaring the story done, run the cumulative grep gate from
PC01RNF-001 with PC01US-010's added literals:

```bash
grep -rE "Primary|Qualifier|Transactional|TxDecorator|Decorator|Spring" \
    internal/domain/service/auditRuleEvaluator.go \
    internal/domain/service/*.go \
    internal/domain/model/*.go \
    internal/domain/vo \
    internal/domain/port \
    internal/domain/usecase \
    internal/domain/errors \
    internal/application
```

This MUST return zero matches in the listed paths. The strings are
confined to:

- `testdata/pc01us010TxDecoratorContract/**` (the three fixture
  trees);
- `internal/cli/command/txDecoratorContractIntegration_test.go`
  (the integration-test literal-substring assertions);
- `internal/domain/service/auditRuleEvaluator_test.go` (the unit-
  test rule helper — PC01RNF-001 explicitly excludes test files
  from the proscribed-string set; same posture PC01US-006 took for
  `Mockito` / `ExtendWith`);
- the rule `description` / `suggestion` templates inside fixture
  YAML (which the renderer substitutes at runtime).

This is the same posture PC01US-002 / PC01US-005 / PC01US-006 /
PC01US-009 adopted for `Lombok` / `Mockito` / `JUnit` /
`Testcontainers`. No `internal/domain/service/auditRuleEvaluator.go`
edit gains a new framework literal — `isEmptyAnnotationArg` is a
pure-string predicate; `evalRequiredAnnotations`'s new branch
references only the parameter key and the captured-string text.

---

## Section 8 — Open Questions & Risks

All questions were pre-resolved during discovery — none are
blocking.

- **Q1 — Sentinel value (`expected_values: 'Qualifier=<non-empty>'`)
  vs new sibling param (`non_empty_value_annotations`)?**
  Pre-resolved: **new sibling param**. The sentinel approach would
  (a) overload `expected_values`'s exact-equality semantics with a
  second matcher, (b) require branching the evidence shape inside
  the same loop (the AC's evidence `value=empty, expected=non-empty`
  does NOT match the existing `expected_value=…, actual=…` template
  emitted by `expected_values`), (c) collide with a hypothetical
  profile author who genuinely wants to assert `Qualifier="<non-
  empty>"` literally, and (d) push the special-case sentinel string
  into the engine's domain code (PC01RNF-001 violation: a literal
  marker like `<non-empty>` is a domain-level decision encoded
  inside the engine). The new-key approach is orthogonal,
  language-neutral, and matches the standard PC01 extension
  pattern. Cost: one new reserved key (24 → 25). Blocking: No.

- **Q2 — Why not introduce a new `AuditRuleKind` constant
  `tx_decorator_contract` (or `non_empty_annotation_value`)?**
  Pre-resolved: same posture as PC01US-006 (which extended
  `required_annotations` rather than minting a new kind for
  `expected_values`). The all-of presence semantic is shared with
  AC1/AC2; adding a separate kind would force profile authors to
  declare the same rule twice (one for missing, one for empty).
  PC01RF-007 explicitly frames `expected_values` and the new
  `non_empty_value_annotations` as parameters of the
  `required_annotations` kind. Blocking: No.

- **Q3 — How does the parser capture `@Qualifier("")` versus
  `@Qualifier("txDecorator")`?** Pre-resolved: the Tree-sitter
  parser captures string-literal annotation arguments VERBATIM,
  including surrounding double quotes. So
  `@Qualifier("txDecorator")` produces
  `decl.AnnotationArgs["Qualifier"] == "\"txDecorator\""` (15
  characters: open-quote, txDecorator, close-quote) and
  `@Qualifier("")` produces `decl.AnnotationArgs["Qualifier"] ==
  "\"\""` (2 characters: open-quote, close-quote). The empty
  marker form `@Qualifier` (no argument list) produces
  `decl.AnnotationArgs["Qualifier"] == ""` (0 characters). The
  `isEmptyAnnotationArg` predicate in §3.1 treats `""` AND `"\"\""`
  AND `"''"` as empty — locking these three forms in via Case 5
  (or Case 3's sub-tests) of §7.2. The integration test for AC3
  exercises the `"\"\""` form specifically because it is the form
  the AC's `@Qualifier(\"\")` Java syntax produces. Reference:
  `TestParser_ClassWithExtendWithArg_PopulatesAnnotationArgs`
  (line 318 of `parser_test.go`) and
  `TestParser_ClassWithMarkerAnnotation_AnnotationArgEmpty`
  (line 346). Blocking: No.

- **Q4 — Does the rule need a `name_pattern` parameter to scope to
  decorator beans?** Pre-resolved: **No.** The fixture trees place
  ONLY `OrderServiceTxDecorator.java` under
  `src/main/java/com/acme/application/decorator/`, so
  `path_scope: src/main/java/com/acme/application/decorator/`
  naturally targets the decorator bean. Real-world adopters who
  keep multiple classes under the same path can EITHER (a) narrow
  `path_scope` further or (b) use the filename in the path. Adding
  a `name_pattern` parameter to `required_annotations` is OUT of
  scope; if a future requirement needs name-based scoping, it lands
  as a separate story (cumulative key registry growth). Blocking:
  No.

- **Q5 — Should the bundled `spring-boot-hexagonal` profile gain
  this rule?** Pre-resolved: **No**, same posture as
  PC01US-002/004/005/006/007/008/009. The bundled profile evolves
  separately under EP-04. This story ships fixtures + integration
  tests + a domain branch; profile authors adopting PC01US-010
  enable the rule by editing their own `.jitctx/profiles/*.yaml`.
  Blocking: No.

- **Q6 — Walker scope.** Pre-resolved: fixtures live under
  `src/main/java/com/acme/application/decorator/...`; `path_scope:
  src/main/java/com/acme/application/decorator/` is the substring
  filter the integration tests rely on. The walker emits paths
  with forward slashes (proven). Blocking: No.

- **Q7 — Determinism when a single declaration triggers all three
  paths simultaneously.** Pre-resolved by §2.3 contract: emit
  order is missing → expected_values → non-empty, with each path
  iterating its parameter slice in input-string order. The unit
  test Case 4 (§7.2) locks this in. Blocking: No.

- **Q8 — Profile YAML fixture content depth.** Pre-resolved: the
  profile YAML carries the FULL canonical `spring-boot-hexagonal`
  shape (name, languages, query_lang, detect, module_detection,
  rules, audit_rules). Audit_rules-only YAML would not be detected
  by the auto-detector. This is the same fixture-content
  requirement PC01US-007 / PC01US-008 / PC01US-009 documented.
  Documented in §7.5. Blocking: No.

- **Q9 — Profile loader reaction to the unknown key
  `non_empty_value_annotations` BEFORE T1-G1 ships.** Pre-
  resolved: irrelevant to this plan because T1-G1 ships in the
  same PR as T6-G2..G5. The loader's `params: map[string]string`
  pass-through preserves unknown keys (forward-compatible per the
  `model.AuditRule` doc-comment line 80–81). After T1-G1 the
  evaluator recognises the key. Before T1-G1 the key would be
  silently ignored. The integration tests in T6-G2 will fail until
  T1-G1 lands — that is the intended dependency edge in §9.
  Blocking: No.

- **Q10 — Multiple violations on the same fixture.** Pre-resolved:
  - AC1 fixture: zero violations (both required annotations
    present, `Qualifier` carries non-empty value).
  - AC2 fixture: exactly one violation (missing-violation path);
    the non-empty-value path is short-circuited because `Qualifier`
    is absent.
  - AC3 fixture: exactly one violation (non-empty-value path); the
    missing-violation path is short-circuited because both
    annotations are present.
  - The integration tests' `strings.Count(stdout,
    "[tx-decorator-contract]") == 1` assertion catches any
    regression that doubles the violation count. Blocking: No.

- **Q11 — Engine-neutrality grep gate scope expansion.** Pre-
  resolved: PC01US-010 expands the de-facto banlist for in-
  codebase verification with `Primary`, `Qualifier`,
  `Transactional`, `Decorator`, `TxDecorator` — all framework-
  specific identifiers that must not bleed into
  `internal/domain/service/auditRuleEvaluator.go` (the only domain
  file edited by this story) or `internal/application`. The grep
  gate documented in §7.6 covers this. The strings are confined to
  `testdata/`, the integration test, the unit-test rule helper
  (per the test-file exemption), and rule descriptions. Blocking:
  No.

- **Risk R1 — Tree-sitter capture of `@Qualifier("")`.** Already
  proven by PC01US-006's parser tests. The parser preserves
  surrounding quotes for string literals; the empty literal `""`
  is captured as the two-character string `""`. The
  `isEmptyAnnotationArg` predicate explicitly handles this form.
  Mitigation: §7.2 Case 3 asserts on the `"\"\""` capture
  directly; the integration test for AC3 exercises the same
  capture path through the real Tree-sitter parser.

- **Risk R2 — Profile authors using `expected_values` AND
  `non_empty_value_annotations` for the same annotation
  simultaneously.** Behaviour: both branches run independently. If
  `expected_values: 'Qualifier="x"'` is set AND `Qualifier="y"` is
  declared on the class AND `non_empty_value_annotations:
  'Qualifier'`, two violations emit (mismatch + non-empty-violated
  if `y == ""`). If `Qualifier="x"` matches AND is non-empty, zero
  extra violations. This is intentional — the two parameters are
  composable; profile authors who want EITHER but not BOTH should
  configure only one. Documented in the doc-comment update (§3.1).
  Blocking: No.

No `Blocking: Yes` entries. Discovery proceeds to implementation.

---

## Section 9 — Parallel Execution Plan (authoritative for `@agent-manager`)

```yaml
tiers:
  - id: 1
    name: Domain — non_empty_value_annotations branch on evalRequiredAnnotations
    depends_on: []
    groups:
      - id: T1-G1
        scope:
          create: []
          modify:
            - internal/domain/service/auditRuleEvaluator.go
        guidelines:
          - .claude/guidelines/domain-layer-guidelines.yml
        effort: S
        notes: >
          Additive branch in evalRequiredAnnotations consuming the new
          optional parameter non_empty_value_annotations
          (comma-joined annotation simple names). New private helper
          isEmptyAnnotationArg co-located with parseExpectedValues
          treats "", "\"\"", and "''" as empty (matches the parser's
          verbatim capture for marker annotations and empty string
          literals — see §3.1 and Q3). Determinism: emit order is
          missing → expected_values → non-empty, each iterated in
          input-string order. Doc-comment update extends the existing
          parameter list and substitution-context block to document
          the new branch and evidence shape
          "annotation=<ann>, value=empty, expected=non-empty".
          Engine-neutrality grep gate (§7.6) MUST pass before this
          group is declared done — no Primary / Qualifier /
          Transactional / Decorator / TxDecorator / Spring literal
          inside internal/domain/service/auditRuleEvaluator.go.

  - id: 6
    name: Tests + fixtures (parallel) — unit, integration, and three fixture trees
    depends_on: [1]
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
          Four new t.Parallel() cases on evalRequiredAnnotations,
          modelled on the PC01US-006 unitTestClassContractRule cases.
          Local helper txDecoratorContractRule() carries the strings
          Primary / Qualifier / tx-decorator-contract — allowed in
          test files per PC01RNF-001 (same posture PC01US-006 took
          for Mockito/ExtendWith). Cases:
          (1) AC1 — Primary + Qualifier("txDecorator") → zero violations;
          (2) AC2 — Primary only → one missing=[Qualifier] violation;
          (3) AC3 — Primary + Qualifier("") → one annotation=Qualifier,
              value=empty, expected=non-empty violation;
          (4) PC01RNF-003 ordering — declaration with missing + wrong
              expected_values + empty-value all together emits exactly
              three violations in the documented order (missing,
              expected_values, non-empty).
          A nested t.Run or stand-alone case 5 covers
          isEmptyAnnotationArg directly with the three accepted empty
          forms ("", "\"\"", "''") and three rejected non-empty forms
          ("x", "\"x\"", "\" \"").

      - id: T6-G2
        scope:
          create:
            - internal/cli/command/txDecoratorContractIntegration_test.go
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: M
        notes: >
          Three test functions (AC1 clean, AC2 missing-Qualifier, AC3
          empty-Qualifier-value), each t.Parallel(). Local helper
          newAuditCmdForTxDecoratorContract modelled on the PC01US-009
          newAuditCmdForIntegrationTestBaseRequiredAnnotations helper
          (no upstream DRY refactor). Loads each fixture via
          copyFixture, runs `audit` against the temp workdir, asserts
          on stdout. AC1 asserted via absence of [tx-decorator-contract]
          AND missing= AND value=empty in stdout. AC2 asserted via the
          verbatim substring missing=[Qualifier]. AC3 asserted via
          the verbatim substring annotation=Qualifier, value=empty,
          expected=non-empty.
          Engine-neutrality grep gate (§7.6) cross-checks that
          internal/domain/service/auditRuleEvaluator.go contains no
          Primary/Qualifier literal after the T1-G1 edit lands.

      - id: T6-G3
        scope:
          create:
            - testdata/pc01us010TxDecoratorContract/projectClean/pom.xml
            - testdata/pc01us010TxDecoratorContract/projectClean/project-state.yaml
            - testdata/pc01us010TxDecoratorContract/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us010TxDecoratorContract/projectClean/src/main/java/com/acme/application/decorator/OrderServiceTxDecorator.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Clean fixture for AC1. OrderServiceTxDecorator declares both
          @Primary and @Qualifier("txDecorator") under
          src/main/java/com/acme/application/decorator/. Profile
          contains the full canonical spring-boot-hexagonal shape
          (name, languages, query_lang, detect, module_detection,
          rules) PLUS the single tx-decorator-contract audit rule
          with annotations 'Primary,Qualifier' and
          non_empty_value_annotations 'Qualifier'. testdata is
          gitignored — author force-adds when committing.

      - id: T6-G4
        scope:
          create:
            - testdata/pc01us010TxDecoratorContract/projectMissingQualifier/pom.xml
            - testdata/pc01us010TxDecoratorContract/projectMissingQualifier/project-state.yaml
            - testdata/pc01us010TxDecoratorContract/projectMissingQualifier/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us010TxDecoratorContract/projectMissingQualifier/src/main/java/com/acme/application/decorator/OrderServiceTxDecorator.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Violating fixture for AC2 — OrderServiceTxDecorator declares
          only @Primary; @Qualifier is intentionally omitted (and its
          import). The missing-violation path of the evaluator fires;
          the integration test asserts the verbatim substring
          missing=[Qualifier] in stdout. The non-empty-value path is
          short-circuited because Qualifier is absent. Profile YAML
          identical to projectClean's profile.

      - id: T6-G5
        scope:
          create:
            - testdata/pc01us010TxDecoratorContract/projectEmptyQualifier/pom.xml
            - testdata/pc01us010TxDecoratorContract/projectEmptyQualifier/project-state.yaml
            - testdata/pc01us010TxDecoratorContract/projectEmptyQualifier/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us010TxDecoratorContract/projectEmptyQualifier/src/main/java/com/acme/application/decorator/OrderServiceTxDecorator.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Violating fixture for AC3 — OrderServiceTxDecorator declares
          @Primary and @Qualifier("") (empty string literal). Both
          required annotations present, so the missing-violation path
          is short-circuited; the new non-empty-value path fires; the
          integration test asserts the verbatim substring
          annotation=Qualifier, value=empty, expected=non-empty in
          stdout. Profile YAML identical to projectClean's profile.
```

---

## Self-Validation Checklist

**File-set coverage**
- Every file in §1 appears exactly once across §9 groups
  (cross-checked: T1-G1 has 1, T6-G1 has 1, T6-G2 has 1, T6-G3 has
  4, T6-G4 has 4, T6-G5 has 4 — total 15, matching §1's 15 rows).
- Every requirement ID (PC01US-010, PC01RF-001, PC01RF-007,
  PC01RF-009, PC01RNF-001, PC01RNF-003, PC01RNF-006) appears in at
  least one §1 row AND in the §1 traceability matrix.
- No file path appears in two groups.

**Frozen contract**
- No new ports, model types, use-case interfaces, or error sentinels
  are introduced. The only new SYMBOL inside the domain layer is the
  private helper `isEmptyAnnotationArg` co-located with
  `parseExpectedValues` (NOT exported, NOT a port). The frozen
  contract in §2 documents the existing surface plus the new
  parameter key and evidence shape; every other symbol is unchanged.
- `Deps` struct in `internal/cli/wire.go` is unchanged — explicitly
  noted in §2.6 and §7.1.
- No fields marked `TODO` or `{placeholder}` in the frozen contract.

**DAG**
- `depends_on` edges: T1-G1 → ∅; T6-G1..G5 → [1]. Acyclic.
- Tier 1 present because `internal/domain/service/auditRuleEvaluator.go`
  is modified by T1-G1 (R1 of the classification rules).
- Tier 2 omitted because no `internal/infrastructure/**` file
  appears in §1.
- Tier 3 omitted because no `internal/application/**` file appears
  in §1.
- Tier 4 omitted because no `internal/cli/command/*Cmd.go` or
  `internal/cli/format/*.go` file appears in §1.
- Tier 5 omitted because no wiring file (`wire.go`, `root.go`,
  `execute.go`, `main.go`, `internal/config/**`) appears in §1.
- Tier 6 present because two `_test.go` files and three fixture
  trees appear in §1.
- All `guidelines[]` paths exist under
  `/workspaces/jitctx/.claude/guidelines/` (verified:
  `domain-layer-guidelines.yml`,
  `integration-test-layer-guidelines.yml`,
  `unit-test-layer-guidelines.yml`).

**Open questions**
- Zero `Blocking: Yes` entries. Discovery is unblocked.
