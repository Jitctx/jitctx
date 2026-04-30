# Plan — PC01US-012 Preserve backward compatibility with legacy `has_annotation`

## Section 0 — Summary

- Feature: **make audit-rule profiles that declare a top-level
  `has_annotation: <Name>` shortcut continue to work unchanged**, by
  translating that legacy YAML shape into the modern
  `kind: required_annotations` rule shape with
  `params.annotations: <Name>` at class scope, **at the YAML parsing
  boundary inside `internal/infrastructure/fsprofile/`**. The legacy
  key emits no warning in this release (PC01RF-012). The translation
  happens BEFORE the kind-whitelist check and BEFORE
  `validateAuditRuleParams` runs (PC01US-011), so a translated rule
  satisfies all downstream gates without special-casing them.
- User Story: **PC01US-012**.
- Requirement IDs covered:
  - **PC01RF-012** — backward compatibility with legacy
    `has_annotation`: "`has_annotation: X` is semantically equivalent
    to `required_annotations: [X]` at class scope. The legacy key may
    be removed in a future release but emits no warning in this one."
  - **PC01RNF-001** — engine language-neutrality. The translation
    helper lives in `internal/infrastructure/fsprofile/`; no
    Java/Spring/Lombok/Mockito/Autowired/JPA literal is introduced.
    The new identifiers are neutral profile-schema words
    (`has_annotation`, `required_annotations`, `class`).
  - **PC01RNF-003** — determinism. The translation produces the
    same `model.AuditRule` byte-for-byte across runs (no map
    iteration, single string assignment to `params["annotations"]`).
    Existing PC01RNF-003 tests remain green.
- Acceptance scenario mapped 1:1 in §7:
  - **AC1** (Gherkin lines 202–206 of
    `quality-gate-evaluators.feature`) — a profile declares an audit
    rule using the legacy `has_annotation: Service` shortcut on
    package `application.usecase`. A class in that package WITHOUT
    `@Service` produces ONE violation with evidence equivalent to
    `required_annotations:[Service]` (i.e. evidence substring
    `missing=[Service]`).
- Layers touched: **infrastructure (Tier 2), tests + fixtures
  (Tier 6)**. No domain change. No application change. No
  presentation change. No wiring change.
- Tiers active: **2, 6**. Tiers 1, 3, 4, 5 are explicitly `N/A`.
  - Tier 1 absent — no new `AuditRuleKind` constant, no new model
    field, no new VO field, no new port, no new use-case interface,
    no new error sentinel. The legacy key is translated INTO the
    existing `model.AuditKindRequiredAnnotations` constant, reusing
    `evalRequiredAnnotations`.
  - Tier 3 absent — `appaudituc.Impl.Execute` already iterates
    parsed files and dispatches to the evaluator. The translated
    rule reaches the use case as a normal `model.AuditRule` and is
    indistinguishable from a hand-authored
    `kind: required_annotations` rule.
  - Tier 4 absent — `auditCmd` and the report formatter render the
    translated rule's violations without modification.
  - Tier 5 absent — no `internal/cli/{wire,root,execute}.go`,
    `cmd/jitctx/main.go`, or `internal/config/**` change.
- Guidelines loaded:
  - `.claude/guidelines/infrastructure-layer-guidelines.yml`
  - `.claude/guidelines/integration-test-layer-guidelines.yml`
  - `.claude/guidelines/unit-test-layer-guidelines.yml`
- Estimated file count: **6 new** (1 unit-test file + 1 integration
  test + 4 fixture files for one fixture tree) and **4 modified**
  (`bundleDto.go`, `dto.go` for the new optional YAML field +
  `bundleMapper.go`, `auditLoader.go` for the translation hook).
  The plan file itself is not counted in §1.

> **Discovery finding (load-bearing).**
>
> 1. **There is no `has_annotation` audit-rule shape in the codebase
>    today.** A repo-wide grep for `has_annotation` shows occurrences
>    in TWO unrelated places:
>    - **Classification rules** (`bundleMatchDTO.HasAnnotation` in
>      `bundleDto.go:63`, `matchDTO.HasAnnotation` in `dto.go:51`,
>      `model.ProfileMatch.HasAnnotation` in `frameworkProfile.go:71`,
>      `model.ClassificationRule.HasAnnotation` in
>      `classificationRule.go:33`). These belong to the EP-03/EP-04
>      classifier (`profileClassifier.go`, `declarativeClassifier.go`)
>      that maps source files to architectural contract types
>      (entity, rest-adapter, jpa-adapter, …). They are NOT audit
>      rules and do NOT produce violations.
>    - **Bundled spring-boot-hexagonal profile**
>      (`internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal/profile.yaml`,
>      lines 49, 54, 59, 76, 101, 122, 129) — these are USES of the
>      classifier `has_annotation` key, not audit rules.
>    The `auditRuleDTO` (`dto.go:39-46`) and `bundleAuditRuleDTO`
>    (`bundleDto.go:76-83`) currently expose ONLY the modern
>    `kind` / `severity` / `description` / `suggestion` / `params`
>    fields. There is no audit-rule kind named `has_annotation`,
>    no `auditRuleDTO.HasAnnotation` field, and no historical commit
>    reference to a legacy audit-rule `has_annotation`.
>
>    **Implication:** PC01RF-012 describes a backward-compat contract
>    for a legacy audit-rule shape that was envisioned in the
>    proposal-changes-01 design but never previously shipped in the
>    audit-rule DTOs. PC01US-012 introduces the legacy YAML field for
>    the FIRST time in the audit-rule DTOs AND the translation that
>    folds it into the modern shape, in the same story. The result
>    is forward-compatible (new profiles use `kind:
>    required_annotations`) AND backward-compatible (older
>    profile-author intent expressed via the shortcut keeps working
>    once they upgrade to a jitctx version with this story landed).
>
> 2. **Two YAML loaders consume audit rules** in
>    `internal/infrastructure/fsprofile/`:
>    - `auditLoader.go::LoadAuditRules` (legacy single-file profiles
>      under `~/.jitctx/profiles/<name>.yaml`) — uses
>      `KnownFields(true)`, drops unknown kinds with a WARN log,
>      fails fatally on unknown severity, runs
>      `validateAuditRuleParams` (PC01US-011) before constructing
>      `model.AuditRule`.
>    - `bundleMapper.go::toBundleDomain` (EP-04 directory profiles
>      under `<dir>/profile.yaml`) — uses `KnownFields(false)`,
>      drops unknown kinds with a WARN log, fails fatally on
>      unknown severity, runs `validateAuditRuleParams` before
>      appending to `RawAuditRules`.
>    `KnownFields(true)` is the load-bearing constraint. If the
>    legacy `has_annotation` field is added to ONLY one of the two
>    DTOs, profiles using the shortcut will fail to decode in the
>    other path with a `field has_annotation not found in type` error.
>    Both DTOs must declare the new optional field. Both loaders
>    must call the same translation helper at the same point in the
>    pipeline.
>
> 3. **Translation must run BEFORE `validateAuditRuleParams` and
>    BEFORE the `knownAuditRuleKinds` check.** The validator
>    requires that a `required_annotations` rule has a non-empty
>    `params["annotations"]` (PC01US-011 message M1). A legacy rule
>    declared as `has_annotation: Service` carries an empty `kind`
>    (the YAML doesn't set `kind:`); without translation, the
>    `knownAuditRuleKinds` whitelist would drop it (`kind` ==
>    `""` is not in the map → unknown-kind WARN drop). With
>    translation FIRST, the rule's effective `kind` becomes
>    `required_annotations` and `params["annotations"]` becomes
>    `Service`, satisfying both the kind whitelist and the
>    PC01US-011 validator.
>
>    **Order in both call sites (after this story):**
>    1. Translate legacy → modern (NEW — see §4.1 helper).
>    2. Existing `knownAuditRuleKinds` whitelist (drop unknown,
>       WARN).
>    3. Existing `knownAuditSeverities` whitelist (fatal on
>       unknown).
>    4. Existing `validateAuditRuleParams` (PC01US-011 — fatal on
>       schema violation).
>    5. Append `model.AuditRule` to result slice.
>
> 4. **Severity defaulting on the translated rule.** A legacy
>    profile written today as `has_annotation: Service` has no
>    `kind:` line — but it MAY still declare `severity:`,
>    `description:`, `suggestion:`, and `params:` (since these are
>    common to every audit-rule entry). The translation MUST
>    preserve these top-level fields verbatim. When `severity:` is
>    absent, the rule cannot survive the existing
>    `knownAuditSeverities` whitelist (which fails fatally on
>    unknown severity, including the empty string). The Gherkin's
>    "an existing profile with `has_annotation: Service`" wording
>    is silent on severity — but PC01US-012's AC requires that the
>    rule produces a violation, so the fixture MUST set
>    `severity: ERROR` (or `WARNING`/`INFO`). The translation
>    helper does NOT default severity — it leaves it to the
>    fixture and to the existing fatal-on-unknown-severity gate.
>    This stays consistent with how every other audit rule shape
>    has always required an explicit `severity:`.
>
>    Decision recorded in §8 Q2 below: severity is NOT defaulted by
>    the translation; profile authors keep declaring it explicitly.
>
> 5. **Path scope from "package" wording in the Gherkin.** The
>    Gherkin says "on package application.usecase". The audit
>    evaluator (`evalRequiredAnnotations`, line 350 of
>    `auditRuleEvaluator.go`) only knows `params["path_scope"]`
>    (substring of `summary.Path`). There is NO `package:` param
>    today. The legacy YAML may carry `path_scope:
>    application/usecase/` directly inside `params:` — that key is
>    forward-compatible (preserved verbatim by the translation
>    because `params:` is round-tripped). The legacy shortcut
>    therefore looks like:
>
>    ```yaml
>    audit_rules:
>      - id: legacy-service-required
>        has_annotation: Service           # NEW shortcut field
>        severity: ERROR
>        description: 'classes under application/usecase/ require @Service'
>        suggestion: 'add @{required} to {file}'
>        params:
>          path_scope: application/usecase/
>    ```
>
>    The translation produces the equivalent modern rule:
>
>    ```yaml
>    # equivalent post-translation
>    - id: legacy-service-required
>      kind: required_annotations
>      severity: ERROR
>      description: 'classes under application/usecase/ require @Service'
>      suggestion: 'add @{required} to {file}'
>      params:
>        path_scope: application/usecase/
>        annotations: Service
>    ```
>
>    The Gherkin substring "violation reported equivalent to
>    `required_annotations:[Service]`" is satisfied by the missing
>    evidence `missing=[Service]` produced by `evalRequiredAnnotations`
>    when the class lacks `@Service`. The integration test asserts on
>    that substring (see §7.4 Test 1).
>
> 6. **What if a profile declares BOTH `kind:` AND `has_annotation:`
>    on the same rule?** This is ambiguous (proposal-changes-01.md
>    does not pin the resolution). Decision recorded in §8 Q3:
>    when BOTH are set, the translation helper prefers the explicit
>    `kind:` value and IGNORES `has_annotation:` (modern wins;
>    legacy is treated as a no-op extra field). This keeps
>    forward-compatible profiles unaffected when an author migrates
>    a rule by adding `kind:` while transitionally keeping the
>    legacy field. Documented in the helper's doc-comment and
>    locked by unit-test case `BothKindAndHasAnnotation_KindWins`.
>
> 7. **List form — `has_annotation: [A, B]`?** The Gherkin shows the
>    SCALAR form `has_annotation: Service`. PC01RF-012 says
>    "`has_annotation: X` is semantically equivalent to
>    `required_annotations: [X]`" — singular. The list form
>    `[A, B]` would be ambiguous (does it mean all-of? first-match?
>    multiple separate rules?). Decision recorded in §8 Q4: only
>    the SCALAR form is supported in this story. A YAML sequence
>    under `has_annotation:` decodes as the empty string into a
>    `string`-typed field (or fails to decode under
>    `KnownFields(true)` for the legacy single-file path); either
>    way, the helper treats it as "no legacy translation" and lets
>    downstream gates handle it.
>
> 8. **Reuse of the existing evaluator (NO domain change).** Since
>    the translation produces a `kind: required_annotations` rule
>    with `params["annotations"]` populated, the existing
>    `evalRequiredAnnotations` (auditRuleEvaluator.go lines 347+)
>    handles it without modification. PC01RF-012's "at class scope"
>    matches `evalRequiredAnnotations`'s default `node_types =
>    ["class_declaration"]` (set when `params["node_types"]` is
>    empty/absent — line 363-365). The translation does NOT need
>    to set `params["target"] = "class"` because:
>    - `evalRequiredAnnotations` does not consult `params["target"]`
>      (it consults `params["node_types"]`).
>    - The PC01US-011 validator (`validateAuditRuleParams`) only
>      rejects `target` when it is non-empty AND not in the closed
>      enum. Leaving `target` absent passes the validator (M2 does
>      not fire on absent targets).
>    The minimal contract therefore is:
>    `params["annotations"] = <legacy value>` ONLY. No other params
>    keys are inserted by the translation.
>
> 9. **Engine-neutrality grep gate.** The new code introduces
>    only the neutral identifiers `has_annotation` and
>    `required_annotations` (existing kind constant). It does NOT
>    introduce `Service`, `Spring`, `Java`, `Lombok`, `Mockito`,
>    `Autowired`, or `JPA` anywhere in the engine. The literal
>    `Service` only appears in YAML fixtures, integration-test
>    substring assertions, and the Gherkin scenario itself.

---

## Section 1 — File Set

| #  | File                                                                                                                  | Action | Layer | Tier | Group | Requirements |
|----|-----------------------------------------------------------------------------------------------------------------------|--------|-------|------|-------|--------------|
| 1  | `internal/infrastructure/fsprofile/legacyHasAnnotation.go`                                                            | create | infra | 2    | T2-G1 | PC01US-012, PC01RF-012 |
| 2  | `internal/infrastructure/fsprofile/dto.go`                                                                            | modify | infra | 2    | T2-G1 | PC01US-012, PC01RF-012 |
| 3  | `internal/infrastructure/fsprofile/bundleDto.go`                                                                      | modify | infra | 2    | T2-G1 | PC01US-012, PC01RF-012 |
| 4  | `internal/infrastructure/fsprofile/bundleMapper.go`                                                                   | modify | infra | 2    | T2-G1 | PC01US-012, PC01RF-012 |
| 5  | `internal/infrastructure/fsprofile/auditLoader.go`                                                                    | modify | infra | 2    | T2-G1 | PC01US-012, PC01RF-012 |
| 6  | `internal/infrastructure/fsprofile/legacyHasAnnotation_test.go`                                                       | create | tests | 6    | T6-G1 | PC01US-012, PC01RF-012 |
| 7  | `internal/cli/command/auditIntegration_test.go`                                                                       | modify | tests | 6    | T6-G2 | PC01US-012 |
| 8  | `testdata/pc01us012LegacyHasAnnotation/projectMissingService/.jitctx/profiles/spring-boot-hexagonal/profile.yaml`     | create | tests | 6    | T6-G3 | PC01US-012, PC01RF-012 |
| 9  | `testdata/pc01us012LegacyHasAnnotation/projectMissingService/.jitctx/manifest.yaml`                                   | create | tests | 6    | T6-G3 | PC01US-012 |
| 10 | `testdata/pc01us012LegacyHasAnnotation/projectMissingService/src/main/java/com/acme/application/usecase/PlaceOrder.java` | create | tests | 6    | T6-G3 | PC01US-012 |
| 11 | `testdata/pc01us012LegacyHasAnnotation/projectMissingService/.jitctx/profiles/spring-boot-hexagonal/templates/.gitkeep` | create | tests | 6    | T6-G3 | PC01US-012 |

Coverage notes:

- File #1 (`legacyHasAnnotation.go`) hosts the new pure helper
  `translateLegacyHasAnnotation(s legacyAuditRuleShape) (effectiveKind, effectiveParams, translated)`
  documented in §4.1. The helper has no model, no port, no
  error-sentinel dependency. It is unit-tested in T6-G1 directly
  with 8+ table cases.
- File #2 (`dto.go`) gains ONE optional field on `auditRuleDTO`:
  `HasAnnotation string \`yaml:"has_annotation"\``. This satisfies
  the legacy single-file loader (`KnownFields(true)`), so a profile
  YAML carrying `has_annotation:` under an audit-rule entry no
  longer fails to decode. The field has no semantic meaning beyond
  the translation step; it is not surfaced in `model.AuditRule`.
- File #3 (`bundleDto.go`) gains the same optional field on
  `bundleAuditRuleDTO`. The bundle loader uses `KnownFields(false)`
  so absence would not error, but adding the field explicitly makes
  the legacy intent typed and discoverable in the codebase.
- File #4 (`bundleMapper.go`) gains a single call-site insertion
  inside the audit-rules loop in `toBundleDomain`, BEFORE the
  `knownAuditRuleKinds` check (so legacy rules with empty `kind`
  pass the whitelist after translation).
- File #5 (`auditLoader.go`) gains the same single insertion inside
  `LoadAuditRules`, BEFORE the `knownAuditRuleKinds` check.
- File #6 (the unit-test file) is a white-box `t.Parallel()` table
  test exercising the helper directly; no fixture files, no I/O.
- File #7 (the integration test) gains ONE new function alongside
  the existing audit cases.
- Files #8 / #9 / #10 / #11 form the single fixture tree
  `pc01us012LegacyHasAnnotation/projectMissingService/`. The
  `.gitkeep` keeps the empty `templates/` directory tracked. The
  Java fixture declares a class WITHOUT `@Service` so the legacy
  rule fires, producing the expected `missing=[Service]` evidence.

Requirement coverage trace (every ID in scope appears below):

| Requirement | Where it lives in code                                                                                                 | Where this plan re-asserts it                                                  |
|-------------|------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------|
| PC01US-012  | T2-G1 publishes `translateLegacyHasAnnotation` and wires it into both audit-rule loaders                               | T6-G1 unit cases + T6-G2 integration scenario                                   |
| PC01RF-012  | New helper folds `has_annotation: X` into `kind: required_annotations` + `params.annotations: X`; no warning emitted   | T6-G1 unit cases + T6-G2 AC1                                                    |
| PC01RNF-001 | grep audit (no Java/Spring/Lombok literals introduced inside `legacyHasAnnotation.go`); helper uses only neutral identifiers | §7.6 grep gate (no test, no fixture)                                            |
| PC01RNF-003 | Translation is deterministic (no map iteration); existing PC01RNF-003 byte-equality tests remain green                 | No new test (PC01US-013 owns the byte-equality assertion; PC01US-012 piggy-backs) |

---

## Section 2 — Frozen Domain Contract

PC01US-012 introduces NO new ports, NO new model types, NO new
use-case interfaces, NO new error sentinels, and NO new
`AuditRuleKind` constant. The contract below is the **reuse contract**
plus the **legacy YAML schema extension** at the infrastructure
boundary.

### 2.1 `model.AuditRule` (frozen, unchanged)

```go
// internal/domain/model/auditRule.go (existing — DO NOT MODIFY)

type AuditRule struct {
    ID          string
    Kind        AuditRuleKind
    Severity    AuditSeverity
    Description string
    Suggestion  string
    Params      map[string]string
}
```

### 2.2 `AuditRuleKind` constants (frozen, unchanged)

```go
// internal/domain/model/auditRule.go (existing — DO NOT MODIFY).
// PC01US-012 reuses AuditKindRequiredAnnotations; no new constant.

const (
    AuditKindRequiredAnnotations AuditRuleKind = "required_annotations"
    // … other constants unchanged …
)
```

### 2.3 DTO field additions (legacy YAML schema extension)

```go
// internal/infrastructure/fsprofile/dto.go (modify — T2-G1).
//
// auditRuleDTO gains ONE optional field for the legacy
// has_annotation shortcut. The field is read by the translation
// helper in legacyHasAnnotation.go and has no other consumer.

type auditRuleDTO struct {
    ID            string            `yaml:"id"`
    Kind          string            `yaml:"kind"`
    Severity      string            `yaml:"severity"`
    Description   string            `yaml:"description"`
    Suggestion    string            `yaml:"suggestion"`
    Params        map[string]string `yaml:"params"`
    // PC01US-012: legacy backward-compat shortcut. When non-empty,
    // translateLegacyHasAnnotation rewrites Kind to
    // "required_annotations" and Params["annotations"] to this
    // value. See legacyHasAnnotation.go.
    HasAnnotation string            `yaml:"has_annotation"`
}
```

```go
// internal/infrastructure/fsprofile/bundleDto.go (modify — T2-G1).
//
// bundleAuditRuleDTO gains the same field for the EP-04 directory
// profile shape.

type bundleAuditRuleDTO struct {
    ID            string            `yaml:"id"`
    Kind          string            `yaml:"kind"`
    Severity      string            `yaml:"severity"`
    Description   string            `yaml:"description"`
    Suggestion    string            `yaml:"suggestion"`
    Params        map[string]string `yaml:"params"`
    // PC01US-012: legacy backward-compat shortcut. See dto.go.
    HasAnnotation string            `yaml:"has_annotation"`
}
```

The two DTOs remain structurally identical.

### 2.4 New helper — `translateLegacyHasAnnotation` (frozen contract)

```go
// internal/infrastructure/fsprofile/legacyHasAnnotation.go (new — T2-G1).
//
// translateLegacyHasAnnotation folds the legacy `has_annotation: X`
// audit-rule shortcut into the modern
// `kind: required_annotations, params.annotations: X` shape.
//
// Inputs:
//   - kind: the rule's declared kind (may be empty for legacy
//     shortcut rules).
//   - hasAnnotation: the legacy `has_annotation:` field value (may
//     be empty when not used).
//   - params: the rule's params map (may be nil — caller passes
//     d.Params directly).
//
// Outputs:
//   - effKind: the effective kind after translation. Equal to kind
//     when no translation occurred; equal to "required_annotations"
//     when translation occurred.
//   - effParams: the effective params map after translation. The
//     returned map is ALWAYS a fresh allocation (never aliases the
//     caller's map) to keep call-site mutation safe. When no
//     translation occurred, it is a shallow copy of params.
//   - translated: true iff translation was applied.
//
// Translation rules (FIRST match wins):
//   1. hasAnnotation == "" → no translation; return (kind, copy(params), false).
//   2. kind != "" AND hasAnnotation != "" → modern kind wins; the
//      legacy field is ignored. Returns (kind, copy(params), false).
//      This locks Q3 (§8): a profile mid-migration that declares
//      both retains the modern semantics; the legacy field becomes
//      a no-op extra. No error, no warning — PC01RF-012 says the
//      legacy key emits no warning in this release.
//   3. kind == "" AND hasAnnotation != "" → translation applies.
//      Returns ("required_annotations", paramsWithAnnotations, true)
//      where paramsWithAnnotations is a fresh map containing every
//      key from params PLUS params["annotations"] = hasAnnotation.
//      If params["annotations"] was already set, the helper PREFERS
//      the existing params value (params wins over has_annotation
//      when both supply the annotation list — locks the more-
//      specific intent). Locked by unit-test case
//      `LegacyAndExplicitAnnotationsParam_ParamsWins`.
func translateLegacyHasAnnotation(
    kind, hasAnnotation string, params map[string]string,
) (effKind string, effParams map[string]string, translated bool)
```

**Pure function.** No filesystem, no logger, no model dependency,
no error return. The helper preserves PC01RNF-001 neutrality —
it references only the literal string `"required_annotations"`
(the existing kind constant value) and never names any framework
identifier.

### 2.5 Call-site contract (both loaders)

Both `bundleMapper.toBundleDomain` and `auditLoader.LoadAuditRules`
gain the SAME translation hook at the SAME pipeline position:

```go
for _, d := range dto.AuditRules {
    // PC01US-012 — legacy shortcut translation. Runs FIRST so the
    // translated rule satisfies all downstream gates uniformly.
    effKind, effParams, _ := translateLegacyHasAnnotation(
        d.Kind, d.HasAnnotation, d.Params,
    )

    // … existing logic, but reading effKind / effParams instead of
    //     d.Kind / d.Params …
    kind := model.AuditRuleKind(effKind)
    if !knownAuditRuleKinds[kind] {
        // unknown-kind WARN drop (existing)
        continue
    }
    sev := model.AuditSeverity(d.Severity)
    if !knownAuditSeverities[sev] {
        // unknown-severity fatal (existing)
        return nil, fmt.Errorf("…: %w", domerr.ErrProfileInvalid)
    }
    // PC01US-011 schema validation (existing).
    if err := validateAuditRuleParams(auditRuleSchema{
        ID: d.ID, Kind: effKind, Params: effParams,
    }); err != nil {
        return nil, fmt.Errorf("…: %w: %w", err, domerr.ErrProfileInvalid)
    }
    rules = append(rules, model.AuditRule{
        ID:          d.ID,
        Kind:        kind,
        Severity:    sev,
        Description: d.Description,
        Suggestion:  d.Suggestion,
        Params:      effParams,
    })
}
```

### 2.6 Reserved param keys after PC01US-012

PC01US-012 adds NO new reserved param keys. The closed set after
PC01US-011 (24 keys) is unchanged: `path_scope`, `annotations`,
`expected_values`, `node_types`, `target`, `exempt_paths`,
`triggered_by`, `name_pattern`, `forbidden_type_patterns`,
`expected_supertype`, `args`, `supertype_kind`, `path_required`,
`path_required_any`, `name_suffix`, `name_regex`,
`forbidden_type_suffix`, `forbidden_type_substring`,
`import_prefix`, `implements_glob`, `annotation`,
`non_empty_value_annotations`.

The helper writes ONLY into the existing `annotations` key;
it does not introduce a new param key.

### 2.7 `Deps` struct in `internal/cli/wire.go`

**Unchanged.** No new dependency is wired. The translation runs
inside the existing infrastructure adapters; their constructors
are unchanged.

### 2.8 New error sentinels

**None.** The translation is infallible — any combination of
`kind` and `has_annotation` produces a well-defined output. The
only errors that can fire AFTER translation are the existing
ones (unknown kind, unknown severity, schema validation),
unchanged by this story.

---

## Section 3 — Domain Layer Plan

**N/A.** No `internal/domain/**` file is created or modified by
PC01US-012. The legacy translation is an infrastructure-layer
concern by guideline rule R2 (path
`internal/infrastructure/fsprofile/**` → Tier 2). The contract
that profile authors see at the YAML level (the legacy shortcut
key) is documented in this plan, the helper's doc-comment, and
the unit-test cases in T6-G1. It does not need to land as a
domain type because:

- No new `AuditRuleKind` constant is introduced (the legacy
  value is folded into the existing `required_annotations` kind).
- No new `model.AuditRule` field is needed (the translation
  populates the existing `Params["annotations"]` slot).
- No new error sentinel is needed (the translation is infallible).
- No new VO is needed (the translation operates on the DTO before
  the VO/model conversion).

---

## Section 4 — Infrastructure Layer Plan

### 4.1 New file — `internal/infrastructure/fsprofile/legacyHasAnnotation.go` (T2-G1)

```go
// Package fsprofile (legacyHasAnnotation.go).
//
// PC01US-012 — backward compatibility for the legacy `has_annotation: X`
// audit-rule shortcut. Profiles authored before the modern
// kind/params shape was introduced may declare an audit rule as:
//
//	audit_rules:
//	  - id: legacy-rule
//	    has_annotation: Service
//	    severity: ERROR
//	    description: '...'
//	    suggestion: '...'
//	    params:
//	      path_scope: application/usecase/
//
// The translation helper folds this into the modern equivalent:
//
//	- id: legacy-rule
//	  kind: required_annotations
//	  severity: ERROR
//	  description: '...'
//	  suggestion: '...'
//	  params:
//	    path_scope: application/usecase/
//	    annotations: Service
//
// Per PC01RF-012, the legacy key keeps working with no deprecation
// warning in this release. Per PC01RNF-001, the helper introduces
// no framework-specific identifier — only the neutral schema words
// `has_annotation` and `required_annotations`.
//
// Translation rules (FIRST match wins):
//   1. has_annotation absent  → no translation.
//   2. kind explicitly set    → modern kind wins; has_annotation is
//                                ignored (no error, no warning).
//   3. kind absent + has_annotation present → translate to
//      kind=required_annotations,
//      params.annotations = has_annotation
//      (UNLESS params.annotations is already set — in which case
//       the existing params value wins).
//
// Both audit-rule loaders (auditLoader.go::LoadAuditRules and
// bundleMapper.go::toBundleDomain) call the helper FIRST in the
// per-rule pipeline, BEFORE the kind whitelist and BEFORE the
// PC01US-011 schema validator. Running translation first allows
// downstream gates to treat the rule uniformly with hand-authored
// modern rules.

package fsprofile

// translateLegacyHasAnnotation folds the legacy `has_annotation: X`
// audit-rule shortcut into the modern kind/params shape. Pure
// function — no filesystem, no logger, no model import, no error
// return.
//
// The returned effParams is ALWAYS a fresh allocation (never aliases
// params), so callers can safely mutate or pass it through to the
// validator and the model.AuditRule conversion without worrying
// about shared state.
func translateLegacyHasAnnotation(
    kind, hasAnnotation string, params map[string]string,
) (effKind string, effParams map[string]string, translated bool) {
    // Always return a fresh copy of params so callers don't share
    // state with the DTO map.
    effParams = make(map[string]string, len(params)+1)
    for k, v := range params {
        effParams[k] = v
    }

    // Rule 1 — no legacy field set: pass-through.
    if hasAnnotation == "" {
        return kind, effParams, false
    }

    // Rule 2 — modern kind wins when both are set.
    if kind != "" {
        return kind, effParams, false
    }

    // Rule 3 — translate. Existing params.annotations wins over
    // has_annotation when both supply the list (more-specific
    // intent). When params.annotations is empty/absent, fold the
    // legacy scalar value into it.
    if _, hasParam := effParams["annotations"]; !hasParam {
        effParams["annotations"] = hasAnnotation
    } else if effParams["annotations"] == "" {
        effParams["annotations"] = hasAnnotation
    }
    // Else: params.annotations is already non-empty — preserve it.
    // The legacy field becomes informational metadata.

    return "required_annotations", effParams, true
}
```

**Why a fresh map copy?** The DTO's `Params` field is decoded once
per profile load. If the helper mutated it in-place, a re-load via
the same DTO instance (or any future test that round-trips the DTO)
would see the mutation. Copying is cheap (audit profiles have at
most ~20 rules with ~5 params each) and locks isolation.

**Why hard-code the literal `"required_annotations"` instead of
importing `model.AuditKindRequiredAnnotations`?** The helper does
not import `model` — it operates on plain strings. The validator
in §4.2 of PC01US-011 already imports `model` for the kind switch.
Either approach is valid; hard-coding the literal here keeps the
helper a single-file, single-import unit and matches the precedent
of `auditRuleValidator.go::splitNonEmpty` (which also avoids
importing the domain helper). A unit-test case
`TranslatedKindMatchesModelConstant` asserts string equality with
`string(model.AuditKindRequiredAnnotations)` so the literal cannot
drift silently.

### 4.2 Modified file — `internal/infrastructure/fsprofile/dto.go` (T2-G1)

Add ONE field at the END of `auditRuleDTO`:

```go
type auditRuleDTO struct {
    ID            string            `yaml:"id"`
    Kind          string            `yaml:"kind"`
    Severity      string            `yaml:"severity"`
    Description   string            `yaml:"description"`
    Suggestion    string            `yaml:"suggestion"`
    Params        map[string]string `yaml:"params"`
    HasAnnotation string            `yaml:"has_annotation"` // PC01US-012
}
```

The field placement matters because `KnownFields(true)` (auditLoader)
otherwise rejects YAML using `has_annotation:` on an audit-rule entry.

### 4.3 Modified file — `internal/infrastructure/fsprofile/bundleDto.go` (T2-G1)

Add the SAME field on `bundleAuditRuleDTO`:

```go
type bundleAuditRuleDTO struct {
    ID            string            `yaml:"id"`
    Kind          string            `yaml:"kind"`
    Severity      string            `yaml:"severity"`
    Description   string            `yaml:"description"`
    Suggestion    string            `yaml:"suggestion"`
    Params        map[string]string `yaml:"params"`
    HasAnnotation string            `yaml:"has_annotation"` // PC01US-012
}
```

The bundle loader currently uses `KnownFields(false)`, so the
field is not strictly required for decode. It is added for parity
with `auditRuleDTO`, for IDE autocompletion in tests, and so the
translation helper has a typed source.

### 4.4 Modified file — `internal/infrastructure/fsprofile/bundleMapper.go` (T2-G1)

Inside `toBundleDomain`, replace lines 137–168 (the audit-rules
loop) with the translation-aware version. The diff is small —
ONE call to `translateLegacyHasAnnotation` near the top of the
loop and substitution of `effKind` / `effParams` for `d.Kind` /
`d.Params` in the downstream lines:

```go
for _, d := range dto.AuditRules {
    // PC01US-012 — legacy shortcut translation. Runs FIRST so a
    // legacy `has_annotation: X` rule reaches the kind whitelist
    // already wearing kind=required_annotations and
    // params.annotations=X.
    effKind, effParams, _ := translateLegacyHasAnnotation(
        d.Kind, d.HasAnnotation, d.Params,
    )

    kind := model.AuditRuleKind(effKind)
    if !knownAuditRuleKinds[kind] {
        logger.Warn("unknown audit rule kind in bundle profile, dropping rule",
            slog.String("kind", effKind),
            slog.String("rule_id", d.ID),
            slog.String("profile", dto.Name),
        )
        continue
    }
    sev := model.AuditSeverity(d.Severity)
    if !knownAuditSeverities[sev] {
        return nil, fmt.Errorf("bundle profile %q: audit rule %q: unknown severity %q: %w",
            dto.Name, d.ID, d.Severity, domerr.ErrProfileInvalid)
    }
    // PC01US-011: per-kind structural validation. Note: now driven
    // by effKind/effParams so the validator sees the post-translation
    // shape.
    if err := validateAuditRuleParams(auditRuleSchema{
        ID: d.ID, Kind: effKind, Params: effParams,
    }); err != nil {
        return nil, fmt.Errorf("bundle profile %q: audit rule %q: %w: %w",
            dto.Name, d.ID, err, domerr.ErrProfileInvalid)
    }
    rawAuditRules = append(rawAuditRules, model.AuditRule{
        ID:          d.ID,
        Kind:        kind,
        Severity:    sev,
        Description: d.Description,
        Suggestion:  d.Suggestion,
        Params:      effParams,
    })
}
```

Behaviour for non-legacy profiles is unchanged: when `d.HasAnnotation`
is empty, `translateLegacyHasAnnotation` returns
`(d.Kind, copy(d.Params), false)` — `effKind == d.Kind` and
`effParams` is a deep-equal copy. The only observable difference is
the fresh `Params` map inside `model.AuditRule` (was: a reference
to the DTO's map; now: a fresh allocation). No existing test asserts
on map identity (every assertion uses `require.Equal` or substring
checks on rendered output), so no existing test breaks.

### 4.5 Modified file — `internal/infrastructure/fsprofile/auditLoader.go` (T2-G1)

Inside `LoadAuditRules`, the same edit pattern:

```go
for _, d := range dto.AuditRules {
    // PC01US-012 — legacy shortcut translation.
    effKind, effParams, _ := translateLegacyHasAnnotation(
        d.Kind, d.HasAnnotation, d.Params,
    )

    kind := model.AuditRuleKind(effKind)
    if !knownAuditRuleKinds[kind] {
        l.logger.Warn("unknown audit rule kind in profile, dropping rule",
            slog.String("kind", effKind),
            slog.String("rule_id", d.ID),
            slog.String("profile", profileName),
        )
        continue
    }
    sev := model.AuditSeverity(d.Severity)
    if !knownAuditSeverities[sev] {
        return nil, fmt.Errorf("profile %q: audit rule %q: unknown severity %q: %w",
            profileName, d.ID, d.Severity, domerr.ErrProfileInvalid)
    }
    if err := validateAuditRuleParams(auditRuleSchema{
        ID: d.ID, Kind: effKind, Params: effParams,
    }); err != nil {
        return nil, fmt.Errorf("profile %q: audit rule %q: %w: %w",
            profileName, d.ID, err, domerr.ErrProfileInvalid)
    }
    rules = append(rules, model.AuditRule{
        ID:          d.ID,
        Kind:        kind,
        Severity:    sev,
        Description: d.Description,
        Suggestion:  d.Suggestion,
        Params:      effParams,
    })
}
```

### 4.6 Other adapters

**N/A.** `treesitter`, `fsmanifest`, `token`, `fsprofile/bundled.go`,
`fsprofile/detector.go`, `fsprofile/extractor.go`,
`fsprofile/resolver.go`, `fsprofile/bundleAuditRulesAdapter.go`,
`fsprofile/loader.go::Load` (legacy classification loader),
`fsprofile/mapper.go` — none of these touch audit-rule DTOs at the
YAML boundary. No edits.

The classifier-side `has_annotation` (in `bundleMatchDTO` and
`matchDTO`) is UNRELATED to this story and is NOT touched. The
two `HasAnnotation` keys live in disjoint pipelines: one feeds
classification (for `module_detection` / `rules:` /
`types[].classification:`), the other feeds audit rules. Both
keep working independently.

---

## Section 5 — Application Layer Plan

**N/A.** `internal/application/usecase/audituc/usecase.go` is
unchanged. The translated `model.AuditRule` reaches the use case
identical in shape to a hand-authored
`kind: required_annotations` rule. The existing dispatch
(`appaudituc.Impl.Execute` → `evaluator.EvaluateRules`) routes
to `evalRequiredAnnotations` (auditRuleEvaluator.go:347+) which
emits the expected `missing=[Service]` violation when the class
lacks `@Service`.

---

## Section 6 — Presentation Layer Plan

**N/A.** `internal/cli/command/auditCmd.go` is unchanged.
`internal/cli/format/auditReport.go` (or equivalent renderer) is
unchanged. The translated rule's violations carry the same
evidence shape and rule ID as a modern rule, so the rendered
report is bit-identical to "hand-author the equivalent modern
rule" — which is exactly what PC01RF-012 demands.

---

## Section 7 — Composition Root + Tests Plan

### 7.1 Composition root

**N/A.** `internal/cli/wire.go`, `internal/cli/root.go`,
`internal/cli/execute.go`, `cmd/jitctx/main.go`,
`internal/config/**` are unchanged.

### 7.2 Unit tests — `internal/infrastructure/fsprofile/legacyHasAnnotation_test.go` (T6-G1)

A single white-box test file (package `fsprofile`) with a
`t.Parallel()` table-driven test exercising the helper directly.

```go
package fsprofile

import (
    "testing"

    "github.com/stretchr/testify/require"

    "github.com/jitctx/jitctx/internal/domain/model"
)

func TestTranslateLegacyHasAnnotation_Table(t *testing.T) {
    t.Parallel()
    cases := []struct {
        name              string
        kind              string
        hasAnnotation     string
        params            map[string]string

        wantKind          string
        wantParams        map[string]string
        wantTranslated    bool
    }{
        {
            name:           "NoLegacyField_PassThrough",
            kind:           "required_annotations",
            hasAnnotation:  "",
            params:         map[string]string{"annotations": "Foo"},
            wantKind:       "required_annotations",
            wantParams:     map[string]string{"annotations": "Foo"},
            wantTranslated: false,
        },
        {
            name:           "NoLegacyField_NoKind_PassThrough",
            kind:           "",
            hasAnnotation:  "",
            params:         map[string]string{},
            wantKind:       "",
            wantParams:     map[string]string{},
            wantTranslated: false,
        },
        {
            name:           "LegacyOnly_TranslatesToRequiredAnnotations",
            kind:           "",
            hasAnnotation:  "Service",
            params:         map[string]string{"path_scope": "application/usecase/"},
            wantKind:       "required_annotations",
            wantParams: map[string]string{
                "path_scope":  "application/usecase/",
                "annotations": "Service",
            },
            wantTranslated: true,
        },
        {
            name:           "LegacyOnly_NilParams_TranslatesAndAllocates",
            kind:           "",
            hasAnnotation:  "Service",
            params:         nil,
            wantKind:       "required_annotations",
            wantParams:     map[string]string{"annotations": "Service"},
            wantTranslated: true,
        },
        {
            name:           "BothKindAndHasAnnotation_KindWins",
            kind:           "forbidden_annotations",
            hasAnnotation:  "Service",
            params:         map[string]string{"annotations": "Autowired"},
            wantKind:       "forbidden_annotations",
            wantParams:     map[string]string{"annotations": "Autowired"},
            wantTranslated: false,
        },
        {
            name:           "LegacyAndExplicitAnnotationsParam_ParamsWins",
            kind:           "",
            hasAnnotation:  "Service",
            params:         map[string]string{"annotations": "Repository"},
            wantKind:       "required_annotations",
            wantParams:     map[string]string{"annotations": "Repository"},
            wantTranslated: true,
        },
        {
            name:           "LegacyAndEmptyAnnotationsParam_LegacyWins",
            kind:           "",
            hasAnnotation:  "Service",
            params:         map[string]string{"annotations": ""},
            wantKind:       "required_annotations",
            wantParams:     map[string]string{"annotations": "Service"},
            wantTranslated: true,
        },
        {
            name:           "ExplicitRequiredAnnotations_NoLegacy_PassThrough",
            kind:           "required_annotations",
            hasAnnotation:  "",
            params: map[string]string{
                "path_scope":  "src/",
                "annotations": "Foo,Bar",
            },
            wantKind: "required_annotations",
            wantParams: map[string]string{
                "path_scope":  "src/",
                "annotations": "Foo,Bar",
            },
            wantTranslated: false,
        },
    }
    for _, tc := range cases {
        tc := tc
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            gotKind, gotParams, gotTranslated :=
                translateLegacyHasAnnotation(tc.kind, tc.hasAnnotation, tc.params)
            require.Equal(t, tc.wantKind, gotKind)
            require.Equal(t, tc.wantParams, gotParams)
            require.Equal(t, tc.wantTranslated, gotTranslated)
        })
    }
}

// TranslatedKindMatchesModelConstant locks the literal
// "required_annotations" emitted by translateLegacyHasAnnotation
// to model.AuditKindRequiredAnnotations. If a future maintainer
// renames the model constant, this test breaks.
func TestTranslatedKindMatchesModelConstant(t *testing.T) {
    t.Parallel()
    gotKind, _, gotTranslated := translateLegacyHasAnnotation(
        "", "Service", nil,
    )
    require.True(t, gotTranslated)
    require.Equal(t, string(model.AuditKindRequiredAnnotations), gotKind)
}

// FreshMapAllocation locks the doc-comment guarantee that the
// returned effParams is never the caller's input map.
func TestTranslateLegacyHasAnnotation_ReturnsFreshMap(t *testing.T) {
    t.Parallel()
    in := map[string]string{"path_scope": "src/"}
    _, out, _ := translateLegacyHasAnnotation("", "Service", in)
    out["path_scope"] = "MUTATED"
    require.Equal(t, "src/", in["path_scope"],
        "translation must not alias the caller's params map")
}
```

### 7.3 Parser unit tests

**N/A.** PC01US-012 does not touch the parser.

### 7.4 Integration tests — `internal/cli/command/auditIntegration_test.go` (T6-G2)

Append ONE new function to the existing file. It follows the
same shape as `TestAuditCmd_Integration_AuditViolationsGoldenMatch`
(see `auditIntegration_test.go:106-128` for the template). The
new test does NOT use the byte-identical golden assertion —
instead it asserts on substring containment of the missing-set
evidence, since the report formatting is shared with every other
`required_annotations` rule and is already locked by other
tests.

```go
// TestAuditCmd_Integration_PC01US012_LegacyHasAnnotation_FlagsMissing
// is the AC1 fixture for PC01US-012. The profile uses the legacy
// `has_annotation: Service` shortcut on a class that does NOT
// declare @Service; the audit must flag exactly one missing-Service
// violation.
func TestAuditCmd_Integration_PC01US012_LegacyHasAnnotation_FlagsMissing(t *testing.T) {
    t.Parallel()

    workDir := t.TempDir()
    copyFixture(t, fixtureDir(t,
        "pc01us012LegacyHasAnnotation", "projectMissingService"), workDir)

    manifestPath := filepath.Join(workDir, ".jitctx", "manifest.yaml")
    stdout, _, run := newAuditCmdFor(t, workDir, manifestPath)

    err := run("--manifest", manifestPath)
    // The audit command exits non-zero ONLY when there are ERROR
    // severity violations. The legacy rule fixture sets
    // severity: ERROR, so the audit exits non-zero. Existing
    // auditViolations test asserts the same posture.
    require.Error(t, err, "ERROR-severity violation must surface as non-zero exit")

    out := stdout.String()
    require.Contains(t, out, "missing=[Service]",
        "evidence must show Service is the missing annotation")
    require.Contains(t, out, "PlaceOrder.java",
        "violation must reference the offending class file")
}
```

The test reuses `newAuditCmdFor` (lines 27-73), `fixtureDir`
(`helpers_test.go:55`), and `copyFixture` (`helpers_test.go:19`) —
no new helper plumbing needed.

### 7.5 Fixtures (T6-G3)

Naming convention follows PC01US-007 / PC01US-008 / PC01US-009 /
PC01US-010 / PC01US-011: lower-camelCase project root segments
matching `pc01us012LegacyHasAnnotation`. testdata is gitignored;
integration-test author force-adds with `git add -f` when committing.

The fixture tree is a regular Java project (single class) plus a
profile directory carrying the legacy shortcut rule:

```
testdata/pc01us012LegacyHasAnnotation/projectMissingService/
├── .jitctx/
│   ├── manifest.yaml
│   └── profiles/
│       └── spring-boot-hexagonal/
│           ├── profile.yaml          # legacy has_annotation rule
│           └── templates/
│               └── .gitkeep
└── src/
    └── main/
        └── java/
            └── com/
                └── acme/
                    └── application/
                        └── usecase/
                            └── PlaceOrder.java
```

#### File `profile.yaml` (the legacy-shortcut rule)

```yaml
name: pc01us012-legacy-has-annotation
language: java
types: []

audit_rules:
  # Legacy shortcut form — has_annotation: Service equivalent to
  # required_annotations: [Service] at class scope (PC01RF-012).
  - id: legacy-service-required
    has_annotation: Service
    severity: ERROR
    description: 'classes under application/usecase/ require @Service'
    suggestion: 'add @{required} to {file}'
    params:
      path_scope: application/usecase/
```

The `has_annotation: Service` line is the legacy shortcut. After
translation it is equivalent to:

```yaml
- id: legacy-service-required
  kind: required_annotations
  severity: ERROR
  description: '...'
  suggestion: '...'
  params:
    path_scope: application/usecase/
    annotations: Service
```

Note the absence of `kind:` — that is what triggers the
translation.

#### File `manifest.yaml`

```yaml
schema: 1
profile: spring-boot-hexagonal
modules: []
files: []
```

A minimal empty manifest. The audit command does NOT require the
manifest to enumerate scanned files for this fixture; the
parser walks `src/` directly via the existing scanner. This
matches the pattern PC01US-007 / PC01US-009 / PC01US-010 used
for their "rule fires" fixtures.

#### File `PlaceOrder.java` (the violating class)

```java
package com.acme.application.usecase;

// PC01US-012 fixture: deliberately MISSING @Service so the legacy
// has_annotation rule fires with missing=[Service] evidence.
public class PlaceOrder {
    public PlaceOrder() {
    }

    public void execute() {
    }
}
```

The class lives in `src/main/java/com/acme/application/usecase/`
so its path contains the substring `application/usecase/`,
matching the rule's `path_scope`. It declares NO `@Service`
annotation, so `evalRequiredAnnotations` produces one violation
with evidence `missing=[Service]`.

#### File `templates/.gitkeep`

Zero-byte placeholder so the empty `templates/` directory tracks
in git. Same posture every PC01 fixture uses.

### 7.6 Engine-neutrality grep gate (PC01RNF-001)

Before declaring T2-G1 done, run the cumulative grep gate from
PC01RNF-001:

```bash
grep -rE "(Lombok|Spring|Mockito|Autowired|JPA)" \
    internal/infrastructure/fsprofile/legacyHasAnnotation.go \
    internal/infrastructure/fsprofile/dto.go \
    internal/infrastructure/fsprofile/bundleDto.go \
    internal/infrastructure/fsprofile/bundleMapper.go \
    internal/infrastructure/fsprofile/auditLoader.go \
    internal/domain \
    internal/application \
    internal/cli
```

This MUST return zero new matches in the listed paths. PC01US-012
adds NO Java/Spring literal anywhere in the engine — the helper
only uses the neutral identifiers `has_annotation` and
`required_annotations`.

The framework-specific identifiers in the AC1 fixture (`Service`,
`com.acme.application.usecase`, `PlaceOrder`) live ONLY in the
YAML fixture under `testdata/`, in the `.java` fixture file, and
in the integration-test's literal-substring assertions. The
testdata directory is excluded from the grep gate.

---

## Section 8 — Open Questions & Risks

All questions were pre-resolved during discovery — none are blocking.

- **Q1 — Should the translation default `severity:` when absent?**
  Pre-resolved: **No.** The Gherkin scenario is silent on
  severity, but every audit rule in the codebase has always
  required an explicit `severity:` to survive
  `knownAuditSeverities`. Defaulting severity in the legacy
  translation would diverge from modern-rule behaviour and
  invent a contract not pinned by the requirement. Profile
  authors keep declaring `severity:` on legacy-shortcut rules
  exactly as on modern rules. The fixture in §7.5 sets
  `severity: ERROR` to satisfy AC1's "violation reported"
  expectation. Blocking: No.

- **Q2 — Should the translation default `params.path_scope:`
  when absent?** Pre-resolved: **No.** The evaluator
  `evalRequiredAnnotations` (line 350-357) requires `path_scope`
  to be non-empty; absence yields zero violations (defensive
  silent no-op). The legacy-shortcut form likely has
  `path_scope` declared in `params:` already (the Gherkin's
  "on package application.usecase" implies a scope is in the
  profile). If the profile author omits it, the rule is
  silently a no-op — same posture as a modern
  `kind: required_annotations` rule with absent `path_scope`.
  Defaulting is NOT in scope here. Blocking: No.

- **Q3 — Resolution when BOTH `kind:` and `has_annotation:` are
  declared on the same rule.** Pre-resolved: **modern kind
  wins, legacy field is silently ignored.** Rationale:
  PC01RF-012 says the legacy key emits no warning in this
  release; mid-migration profiles will commonly carry both as
  the author transitions. Preferring the modern kind preserves
  the more-specific intent and matches the directional flow of
  the migration. The legacy field in this case is a no-op
  extra; future stories may emit a deprecation warning when
  PC01RF-012 graduates from "no warning" to "removed". Locked
  by unit-test case `BothKindAndHasAnnotation_KindWins`.
  Blocking: No.

- **Q4 — Does `has_annotation:` accept a YAML sequence
  `[A, B]`?** Pre-resolved: **No, scalar only.** Rationale:
  PC01RF-012 phrases the equivalence as
  "`has_annotation: X` ≡ `required_annotations: [X]`"
  (singular). A YAML sequence cannot decode into a `string`-typed
  field; under `KnownFields(true)` the auditLoader path returns
  a decode error (`cannot unmarshal !!seq into string`), under
  `KnownFields(false)` the bundle path silently leaves the
  field empty (no translation). Profile authors needing
  multi-annotation all-of semantics declare a modern
  `kind: required_annotations` rule directly; the legacy
  shortcut is intentionally narrow. Blocking: No.

- **Q5 — Does the helper interact with `params["target"]`?**
  Pre-resolved: **No.** `evalRequiredAnnotations` reads
  `params["node_types"]` (default
  `["class_declaration"]`), NOT `params["target"]`. Setting
  `target: class` in the translated output would be a no-op
  for the evaluator AND would unnecessarily expand the helper's
  behaviour beyond what PC01RF-012 specifies. The PC01US-011
  validator does NOT require `target` to be present; it only
  rejects `target` values OUTSIDE the closed enum. Leaving
  `target` absent is the minimal correct behaviour. Blocking:
  No.

- **Q6 — Does the helper interact with `params["annotations"]`
  when the legacy field AND `params.annotations` are both
  set?** Pre-resolved: **`params.annotations` wins** when
  non-empty (the more-specific intent), the legacy field
  fills it ONLY when absent or empty. Locked by unit-test
  cases `LegacyAndExplicitAnnotationsParam_ParamsWins` and
  `LegacyAndEmptyAnnotationsParam_LegacyWins`. Blocking: No.

- **Q7 — Why not place the legacy-shortcut field on the
  classifier-side `bundleMatchDTO` / `matchDTO`?** Pre-resolved:
  those are UNRELATED. The classifier-side `has_annotation`
  matches a single annotation in the EP-03 module-classification
  pipeline (see `profileClassifier.go:35`,
  `declarativeClassifier.go:62`). It is a long-standing field
  with its own semantics (annotation-name matching for type
  classification, NOT audit violations). PC01US-012 introduces
  a NEW `has_annotation` field on the `auditRuleDTO` /
  `bundleAuditRuleDTO` structs ONLY. The two fields share a
  YAML key name but live on disjoint DTO types and feed
  disjoint pipelines. Future maintainers must take care not
  to confuse them — both unit-test files include an inline
  comment pointing at this distinction. Blocking: No.

- **Q8 — Backward-compat for `auditRuleDTO`'s
  `KnownFields(true)` decoder.** Pre-resolved: by adding the
  `HasAnnotation string yaml:"has_annotation"` field to
  `auditRuleDTO`, profiles using the shortcut decode cleanly
  under the strict-fields path. Profiles NOT using the
  shortcut continue to decode without change (the field
  defaults to ""). Tested via the unit-test
  `NoLegacyField_PassThrough` and via every existing
  `auditLoader_test.go` assertion (none of which sets
  `has_annotation:` — they continue to pass). Blocking: No.

- **Q9 — Why duplicate the call-site code rather than extract
  it into a single helper that both loaders call?** Pre-resolved:
  the two loaders have DIFFERENT error-prefix wrappers
  (`"profile %q: audit rule %q: ..."` vs
  `"bundle profile %q: audit rule %q: ..."`) and DIFFERENT
  logger handles (`l.logger` vs the parameter `logger`).
  Extracting both into one helper would require passing the
  logger AND the prefix, AND would obscure the per-loader
  call site that other PC01 stories already touched (PC01US-011
  added the validator hook in both places). The duplication
  is intentional and matches the precedent set by PC01US-011.
  Blocking: No.

- **Risk R1 — A profile that already declares
  `kind: required_annotations` AND uses `has_annotation:`
  (mid-migration profile).** Pre-resolved: covered by Q3 —
  modern kind wins, legacy field becomes a no-op extra. No
  warning, no error. Locked by unit test
  `BothKindAndHasAnnotation_KindWins`. Blocking: No.

- **Risk R2 — A profile authored with `has_annotation:` AND a
  `params.annotations:` value that disagrees with the legacy
  scalar.** Pre-resolved: `params.annotations` wins (Q6).
  This is the rare case but locked by unit test
  `LegacyAndExplicitAnnotationsParam_ParamsWins`. Profile
  authors who intentionally want the disagreement would be
  expressing modern semantics anyway. Blocking: No.

- **Risk R3 — `KnownFields(true)` rejects the new field for
  some reason (e.g. yaml.v3 quirk).** Pre-resolved: the new
  field is a normal `string` with a `yaml:"has_annotation"`
  tag, identical in shape to the existing
  `Kind / Severity / Description / Suggestion` fields. The
  yaml.v3 decoder has no quirks with such fields; existing
  classifier-side `bundleMatchDTO.HasAnnotation` (also a
  `string` with the same YAML tag, also `yaml:"has_annotation"`)
  decodes cleanly under strict-fields in
  `BundleLoader.LoadBundle`. The pattern is proven. Blocking:
  No.

- **Risk R4 — Existing profile YAML files in `bundled/` and
  `testdata/` use `has_annotation:` inside `rules[].match:`
  (classifier).** Pre-resolved: `rules[].match.has_annotation`
  decodes into `bundleMatchDTO`, NOT `bundleAuditRuleDTO`.
  Adding the new field on `bundleAuditRuleDTO` does NOT shadow
  the classifier-side field. The YAML hierarchy
  (`rules[].match.has_annotation` vs
  `audit_rules[].has_annotation`) keeps them disjoint. No
  existing fixture needs modification. Blocking: No.

No `Blocking: Yes` entries. Discovery proceeds to implementation.

---

## Section 9 — Parallel Execution Plan (authoritative for `@agent-manager`)

```yaml
tiers:
  - id: 2
    name: Infrastructure — legacy has_annotation translator + DTO field + loader call sites
    depends_on: []
    groups:
      - id: T2-G1
        scope:
          create:
            - internal/infrastructure/fsprofile/legacyHasAnnotation.go
          modify:
            - internal/infrastructure/fsprofile/dto.go
            - internal/infrastructure/fsprofile/bundleDto.go
            - internal/infrastructure/fsprofile/bundleMapper.go
            - internal/infrastructure/fsprofile/auditLoader.go
        guidelines:
          - .claude/guidelines/infrastructure-layer-guidelines.yml
        effort: M
        notes: >
          Pure helper translateLegacyHasAnnotation in
          legacyHasAnnotation.go folds legacy
          `has_annotation: X` audit-rule shortcuts into the modern
          `kind: required_annotations, params.annotations: X`
          shape. Adds a `HasAnnotation string yaml:"has_annotation"`
          field on auditRuleDTO and bundleAuditRuleDTO so
          KnownFields(true) decoders accept the legacy shortcut.
          Both call sites (bundleMapper.go::toBundleDomain and
          auditLoader.go::LoadAuditRules) invoke the helper FIRST
          in the per-rule pipeline — BEFORE the
          knownAuditRuleKinds whitelist and BEFORE the PC01US-011
          validateAuditRuleParams hook — so the translated rule
          satisfies all downstream gates uniformly. Translation
          rules (FIRST match wins) — (1) no legacy field →
          pass-through, (2) explicit kind set → modern wins
          (legacy ignored, no warning per PC01RF-012),
          (3) legacy only → translate to required_annotations
          with params.annotations populated unless
          params.annotations is already non-empty (params wins).
          Returned effParams is ALWAYS a fresh allocation
          (never aliases the DTO's params map). Engine-neutrality
          grep gate (Section 7.6) MUST pass before this group is
          declared done — no
          Lombok/Spring/Mockito/Autowired/JPA literal inside any
          of the five files.

  - id: 6
    name: Tests + fixtures (parallel) — unit, integration, and one fixture tree
    depends_on: [2]
    groups:
      - id: T6-G1
        scope:
          create:
            - internal/infrastructure/fsprofile/legacyHasAnnotation_test.go
          modify: []
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: M
        notes: >
          White-box t.Parallel() table test in package fsprofile
          covering every translation rule plus the
          model-constant-equality lock and the fresh-map
          allocation guarantee. Eight table cases covering
          NoLegacyField_PassThrough,
          NoLegacyField_NoKind_PassThrough,
          LegacyOnly_TranslatesToRequiredAnnotations,
          LegacyOnly_NilParams_TranslatesAndAllocates,
          BothKindAndHasAnnotation_KindWins,
          LegacyAndExplicitAnnotationsParam_ParamsWins,
          LegacyAndEmptyAnnotationsParam_LegacyWins,
          ExplicitRequiredAnnotations_NoLegacy_PassThrough.
          Plus two standalone tests:
          TestTranslatedKindMatchesModelConstant (locks the
          literal "required_annotations" against
          model.AuditKindRequiredAnnotations) and
          TestTranslateLegacyHasAnnotation_ReturnsFreshMap
          (locks the no-aliasing guarantee).

      - id: T6-G2
        scope:
          create: []
          modify:
            - internal/cli/command/auditIntegration_test.go
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          One new t.Parallel() function appended to the existing
          file — TestAuditCmd_Integration_PC01US012_LegacyHasAnnotation_FlagsMissing.
          The test copies fixture
          pc01us012LegacyHasAnnotation/projectMissingService into
          a tempdir, runs `jitctx audit`, and asserts the
          rendered stdout contains the substring
          "missing=[Service]" AND "PlaceOrder.java". Reuses
          newAuditCmdFor, fixtureDir, and copyFixture helpers —
          no new helper plumbing.

      - id: T6-G3
        scope:
          create:
            - testdata/pc01us012LegacyHasAnnotation/projectMissingService/.jitctx/profiles/spring-boot-hexagonal/profile.yaml
            - testdata/pc01us012LegacyHasAnnotation/projectMissingService/.jitctx/profiles/spring-boot-hexagonal/templates/.gitkeep
            - testdata/pc01us012LegacyHasAnnotation/projectMissingService/.jitctx/manifest.yaml
            - testdata/pc01us012LegacyHasAnnotation/projectMissingService/src/main/java/com/acme/application/usecase/PlaceOrder.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          AC1 fixture. profile.yaml declares ONE audit rule using
          the legacy shortcut (id: legacy-service-required,
          has_annotation: Service, severity: ERROR, params:
          {path_scope: application/usecase/}). The Java fixture
          PlaceOrder.java lives under
          src/main/java/com/acme/application/usecase/ (matching
          the path_scope) and deliberately OMITS the @Service
          annotation, so the translated rule
          (kind=required_annotations,
          params.annotations=Service) fires with evidence
          missing=[Service]. manifest.yaml is the minimal
          schema-1 stub. templates/.gitkeep keeps the empty
          templates/ directory tracked (testdata is gitignored —
          force-add with git add -f when committing).
```
