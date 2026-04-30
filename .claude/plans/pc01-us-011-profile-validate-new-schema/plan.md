# Plan — PC01US-011 Validate new YAML schema in `jitctx profile validate`

## Section 0 — Summary

- Feature: **make `jitctx profile validate` reject malformed audit-rule
  declarations that use the new PC01 YAML schema** — concretely, an
  audit rule whose `kind` is `required_annotations` with an empty
  `params.annotations` list, OR an audit rule whose `params.target`
  carries a value outside the supported set
  `[class, field, method, supertype]`. A profile that uses every
  PC01-introduced kind correctly must continue to pass with exit 0.
- User Story: **PC01US-011**.
- Requirement IDs covered:
  - **PC01RF-011** — profile-schema validation for new keys: empty
    `required_annotations` and unknown `target` are the two
    Gherkin-pinned fatals; the third example in the requirement
    ("missing `kind` on `supertype_rules`") is NOT pinned by an
    acceptance scenario and is parked in §8 (Q-OPTIONAL — out of scope
    for this story; revisit in a follow-up if a future scenario asserts
    it).
  - **PC01RNF-001** — engine language-neutrality. The validator lives
    in `internal/infrastructure/fsprofile/`; no Java/Spring/Lombok
    literal is introduced in the validator. The error-message catalogue
    is restricted to: parameter key names (`required_annotations`,
    `target`), enum values (`class`, `field`, `method`, `supertype`),
    and `rule '<id>'` quoting. None of these are framework-specific.
- Acceptance scenarios mapped 1:1 in §7:
  - **AC1** (empty required-annotations) — profile `pc01us011/emptyRequiredAnnotations`
    declares ONE audit rule with `id: X`, `kind: required_annotations`,
    `params: {annotations: ""}` →
    `jitctx profile validate <dir>` exits non-zero AND stderr contains
    the literal substring
    `rule 'X': required_annotations must declare at least one annotation`.
  - **AC2** (unknown target) — profile `pc01us011/unknownTarget`
    declares ONE audit rule with `id: X`, `kind: forbidden_annotations`,
    `params: {annotations: "Some", target: "foo"}` →
    `jitctx profile validate <dir>` exits non-zero AND stderr contains
    the literal substring
    `rule 'X': target must be one of [class, field, method, supertype]`.
  - **AC3** (valid full-schema profile) — profile `pc01us011/validFullSchema`
    declares ONE audit rule per shipped kind exercising every
    PC01-introduced shape mentioned in scenario 3
    (`required_annotations`, `forbidden_annotations`,
    `forbidden_field_type_pattern` for `forbidden_field_types`,
    `method_naming` for `method_rules`, `required_parameterized_supertype`
    for `supertype_rules`) → `jitctx profile validate <dir>` exits zero.
- Layers touched: **infrastructure (Tier 2), tests + fixtures (Tier 6)**.
  No domain change. No application change. No presentation change.
  No wiring change.
- Tiers active: **2, 6**. Tiers 1, 3, 4, 5 are explicitly `N/A`.
  - Tier 1 absent — no new model field, no new VO field, no new port,
    no new use-case interface, no new error sentinel. The validation
    error reuses the existing `domerr.ErrProfileInvalid` sentinel
    (already wrapped by the `*domerr.ProfileValidationError` carrier
    constructed in the application use case from any `loadErr` value).
  - Tier 3 absent — `profilevalidateuc.Impl.Execute` already routes
    `LoadBundle` errors through `humanizeLoadErr(loadErr)` into the
    output's `Errors` slice; the new validator surfaces its messages
    through that path verbatim.
  - Tier 4 absent — `profileValidateCmd` and `format.TranslateError`
    already render `*ProfileValidationError` with `\n  - <msg>` per
    fatal; the new substrings appear after the bullet without further
    formatting work.
  - Tier 5 absent — no `internal/cli/{wire,root,execute}.go`,
    `cmd/jitctx/main.go`, or `internal/config/**` change.
- Guidelines loaded:
  - `.claude/guidelines/infrastructure-layer-guidelines.yml`
  - `.claude/guidelines/integration-test-layer-guidelines.yml`
  - `.claude/guidelines/unit-test-layer-guidelines.yml`
- Estimated file count: **15 new** (1 unit-test file + 1 integration test
  + 13 fixture files across 3 fixture trees) and **2 modified**
  (`bundleMapper.go` for the new validator hook + `auditLoader.go` for
  the same hook applied on the legacy `Load`/`LoadAuditRules` flow).
  The plan file itself is not counted in §1.

> **Discovery finding (load-bearing).**
>
> 1. **Where validation lives today.** Two YAML loaders consume audit
>    rules in `internal/infrastructure/fsprofile/`:
>    - `auditLoader.go::LoadAuditRules` (legacy single-file profiles
>      under `~/.jitctx/profiles/<name>.yaml`) — uses `KnownFields(true)`,
>      drops unknown kinds with a WARN log, fails fatally on unknown
>      severity, and **does NOT validate `params` content**.
>    - `bundleMapper.go::toBundleDomain` (EP-04 directory profiles
>      under `<dir>/profile.yaml`) — uses `KnownFields(false)`, drops
>      unknown kinds with a WARN log, fails fatally on unknown severity,
>      and **does NOT validate `params` content**.
>    The integration test for `jitctx profile validate` (file
>    `internal/cli/command/profileValidateIntegration_test.go`) wires
>    `BundleLoader`, so the bundle path is the load-bearing one. The
>    legacy single-file path ALSO needs the validator hook so that the
>    `audit` and `scan` commands surface the same schema errors at load
>    time (defence in depth — but the Gherkin scenarios only assert
>    against `profile validate`).
>
> 2. **What a "rule" looks like in YAML.** PC01 introduced new
>    `AuditRuleKind` constants (`required_annotations`,
>    `forbidden_annotations`, `method_naming`,
>    `forbidden_field_type_pattern`,
>    `required_parameterized_supertype` — see
>    `internal/domain/model/auditRule.go:9-65`). Each rule is one
>    sequence entry under the top-level `audit_rules:` key:
>
>    ```yaml
>    audit_rules:
>      - id: X
>        kind: required_annotations
>        severity: ERROR
>        description: '...'
>        suggestion: '...'
>        params:
>          path_scope: src/main/java/
>          annotations: 'A,B,C'   # comma-joined list (or '' for empty)
>          target: class           # one of: class | field | method | supertype
>    ```
>
>    The Gherkin phrase "a profile YAML with rule 'X' declaring
>    `required_annotations: []`" maps to the YAML shape
>    `kind: required_annotations` plus `params.annotations: ""` (or the
>    `annotations` key absent altogether). The phrase
>    "scenario 3 — using `required_annotations`, `forbidden_annotations`,
>    `field_rules`, `method_rules`, `supertype_rules`,
>    `forbidden_field_types`" is the SUPERSET of evaluator capabilities
>    available after PC01US-002…PC01US-008 and corresponds to the kind
>    constants enumerated above. `field_rules` is encoded as
>    `kind: forbidden_annotations` with `params.target: field`;
>    `method_rules` is encoded as `kind: method_naming`;
>    `supertype_rules` is encoded as `kind: required_parameterized_supertype`;
>    `forbidden_field_types` is encoded as
>    `kind: forbidden_field_type_pattern`. There are NO new top-level
>    YAML keys to add to `bundleDTO`/`profileDTO`.
>
> 3. **The two gaps the validator must close.**
>    - **Gap A** — empty `annotations` on a `required_annotations` rule
>      is currently silently tolerated by the loader: the
>      `params: map[string]string` round-trips an empty string verbatim,
>      and the evaluator's `splitNonEmpty(rule.Params["annotations"])`
>      (auditRuleEvaluator.go:351) returns `nil`, producing zero
>      violations. The rule is effectively a no-op. PC01RF-011 declares
>      this case a **profile-validation error** (see
>      `quality-gate-evaluators.md:67-68`: "Empty list is a profile
>      validation error, not a runtime violation").
>    - **Gap B** — `params.target` accepts any string. The evaluator
>      (auditRuleEvaluator.go:593-596 for `forbidden_annotations`,
>      :608-635 for the `target=class|field` switch) has a defensive
>      "unknown target — defensive, no violations" branch (line 631).
>      An unknown `target` is therefore silently a no-op, identical to
>      Gap A. PC01RF-011 demands that this be a fatal at load-time.
>
> 4. **Rendering path for stderr.** When the bundle loader returns an
>    error wrapping `ErrProfileInvalid`, the use case (file
>    `internal/application/usecase/profilevalidateuc/usecase.go`,
>    lines 110-116) calls `humanizeLoadErr(loadErr)` which returns
>    `err.Error()` verbatim for non-`ErrProfileYamlMissing` errors. The
>    use case appends a `ValidationIssue{Code: "profile_invalid",
>    Message: <verbatim string>}` to `out.Errors`. After Step 5 it
>    constructs a `*ProfileValidationError{..., Errors: [...]}`. The
>    presentation translator
>    (`internal/cli/format/errors.go:91-99`) renders the carrier as
>    `profile "<path>": N error(s)\n  - <msg1>\n  - <msg2>...`. The
>    cobra command (`profileValidateCmd.go:30-31`) returns this
>    rendered error to cobra, which prints it on stderr.
>    Therefore: any error string the validator emits at the
>    `bundleMapper`/`auditLoader` boundary appears verbatim on stderr.
>    The two pinned substrings
>    `rule 'X': required_annotations must declare at least one annotation`
>    and
>    `rule 'X': target must be one of [class, field, method, supertype]`
>    must therefore be the EXACT `Error()` text the validator
>    constructs.
>
> 5. **Line/column reporting (PC01RF-011).** The requirement asks for
>    "schema errors with line/column", but the Gherkin scenarios for
>    PC01US-011 do NOT assert on line/column substrings — they assert
>    only on the `rule '<id>': <message>` substring. The current
>    `bundleDTO`/`auditRuleDTO` decode path uses
>    `yaml.NewDecoder(...).Decode(&dto)` which discards `*yaml.Node`
>    positions. Adding line/column would require a two-pass decode
>    (Node tree first, locate each rule's `params` mapping, then
>    targeted validation). This is a sizeable refactor (touches every
>    audit-rule DTO consumer), is NOT pinned by the acceptance
>    scenarios, and would dilute the focus of PC01US-011. It is parked
>    as **Q-OPTIONAL** in §8 — defer to a dedicated story when a
>    Gherkin scenario actually pins line/column substrings. Same
>    posture PC01US-009 took for ratifying-only stories. Blocking: No.
>
> 6. **Why both `auditLoader.go` and `bundleMapper.go`?**
>    `profile validate` consumes `BundleLoader.LoadBundle` →
>    `bundleMapper.toBundleDomain`. That is the load-bearing path for
>    the Gherkin scenarios. The legacy single-file path
>    (`Loader.LoadAuditRules`) is consumed by `audit` and `scan`. Not
>    applying the same validation there would mean a profile that
>    `audit` accepts (silently no-op rules) `profile validate` rejects,
>    a confusing regression in defence-in-depth. The fix is to extract
>    the validator into a small pure helper
>    `validateAuditRuleParams(d auditRuleDTO) error` (or its bundle
>    twin) and call it from both loaders. The helper depends only on
>    the DTO shape — no model, no service, no port. Cost: one new
>    private function plus two small call-site edits. Benefit: any
>    command that loads audit rules now produces the same fatal at
>    load time. The integration tests in §7 only exercise
>    `profile validate`; the legacy-loader call site is covered by the
>    unit tests in §7.2.
>
> 7. **Reuse of the bundled fixture shape.** PC01US-007 / PC01US-008 /
>    PC01US-009 / PC01US-010 each shipped a "FULL canonical
>    spring-boot-hexagonal" profile under
>    `testdata/pc01usNNN<Camel>/projectXxx/.jitctx/profiles/spring-boot-hexagonal.yaml`.
>    PC01US-011 fixtures are different: they are profile DIRECTORIES
>    (the `<dir>` argument to `profile validate <dir>`), NOT project
>    workspaces. They live under
>    `testdata/pc01us011ProfileValidateNewSchema/<scenario>/profile.yaml`
>    plus a `templates/` subdirectory carrying any template referenced
>    from `types[].template`. This matches the EP04US-007 fixture
>    layout (`testdata/ep04us007/cleanProfile/...`). The scenario-3
>    valid profile MUST also satisfy the EP-04 directory shape: a
>    `name:`, a `language: java`, no missing template, no duplicate
>    type id. The simplest viable shape is `name: validFullSchema`,
>    `language: java`, `types: []` (no type declarations needed for
>    audit-rule validation), and the `audit_rules:` block exercising
>    each kind.

---

## Section 1 — File Set

| #  | File                                                                                                                            | Action  | Layer    | Tier | Group  | Requirements |
|----|---------------------------------------------------------------------------------------------------------------------------------|---------|----------|------|--------|--------------|
| 1  | `internal/infrastructure/fsprofile/auditRuleValidator.go`                                                                       | create  | infra    | 2    | T2-G1  | PC01US-011, PC01RF-011 |
| 2  | `internal/infrastructure/fsprofile/auditRuleValidator_test.go`                                                                  | create  | tests    | 6    | T6-G1  | PC01US-011, PC01RF-011 |
| 3  | `internal/infrastructure/fsprofile/bundleMapper.go`                                                                             | modify  | infra    | 2    | T2-G1  | PC01US-011, PC01RF-011 |
| 4  | `internal/infrastructure/fsprofile/auditLoader.go`                                                                              | modify  | infra    | 2    | T2-G1  | PC01US-011, PC01RF-011 |
| 5  | `internal/cli/command/profileValidateIntegration_test.go`                                                                       | modify  | tests    | 6    | T6-G2  | PC01US-011 |
| 6  | `testdata/pc01us011ProfileValidateNewSchema/emptyRequiredAnnotations/profile.yaml`                                              | create  | tests    | 6    | T6-G3  | PC01US-011, PC01RF-011 |
| 7  | `testdata/pc01us011ProfileValidateNewSchema/emptyRequiredAnnotations/templates/.gitkeep`                                        | create  | tests    | 6    | T6-G3  | PC01US-011 |
| 8  | `testdata/pc01us011ProfileValidateNewSchema/unknownTarget/profile.yaml`                                                         | create  | tests    | 6    | T6-G4  | PC01US-011, PC01RF-011 |
| 9  | `testdata/pc01us011ProfileValidateNewSchema/unknownTarget/templates/.gitkeep`                                                   | create  | tests    | 6    | T6-G4  | PC01US-011 |
| 10 | `testdata/pc01us011ProfileValidateNewSchema/validFullSchema/profile.yaml`                                                       | create  | tests    | 6    | T6-G5  | PC01US-011, PC01RF-011 |
| 11 | `testdata/pc01us011ProfileValidateNewSchema/validFullSchema/templates/.gitkeep`                                                 | create  | tests    | 6    | T6-G5  | PC01US-011 |

Coverage notes:

- File #1 (`auditRuleValidator.go`) hosts the new pure helper
  `validateAuditRuleParams(d auditRuleDTO) error` documented in §4.1.
  The helper has no model, no port, no error-sentinel dependency
  beyond the existing `domerr.ErrProfileInvalid`. It is unit-tested
  in T6-G1 directly without spinning up a full `BundleLoader`.
- File #3 (`bundleMapper.go`) gains a single call-site insertion
  inside the audit-rules loop in `toBundleDomain` (between the
  severity check and the `model.AuditRule` append): when
  `validateAuditRuleParams(d)` returns a non-nil error, the function
  returns it wrapped via `fmt.Errorf("bundle profile %q: audit rule
  %q: %w: %w", dto.Name, d.ID, err, domerr.ErrProfileInvalid)`. The
  resulting error string carries the literal `rule '<id>': ...`
  message produced by the validator (see §4.1 for the message
  catalogue). The wrapping prefix
  `bundle profile "<name>": audit rule "<id>":` is NOT pinned by
  any Gherkin substring assertion, so it is free to live in front of
  the literal substring; `require.Contains` matches the substring
  regardless of the prefix.
- File #4 (`auditLoader.go`) gains the same single insertion inside
  `LoadAuditRules`'s loop, wrapped via `fmt.Errorf("profile %q: audit
  rule %q: %w: %w", profileName, d.ID, err, domerr.ErrProfileInvalid)`.
  This call site is exercised only by the unit test in T6-G1 (no
  Gherkin scenario asserts on the legacy single-file path).
- File #5 (the integration test) gains three new test functions
  alongside the five existing `TestProfileValidate_*` cases.
  Authoring is independent of the fixtures (T6-G3..G5) — the test
  only references fixture paths.
- Files #6 / #8 / #10 are profile YAML files. The other fixture
  files are `.gitkeep` placeholders so the empty `templates/`
  directory tracks in git (testdata is gitignored; force-add policy
  applies — same posture PC01US-007/008/009/010 followed). The
  scenario-3 valid profile uses `types: []`, so no real template is
  needed.
- Three fixture trees, each in its own group (T6-G3..G5). The
  fixture-authoring work is parallelisable and conflict-free.

Requirement coverage trace (every ID in scope appears below):

| Requirement | Where it lives in code                                                                                                                                                                              | Where this plan re-asserts it                                |
|-------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------|
| PC01US-011  | T2-G1 publishes the new `validateAuditRuleParams` helper and wires it into both audit-rule loaders                                                                                                  | T6-G1 unit cases + T6-G2 three integration scenarios          |
| PC01RF-011  | New helper enforces per-kind structural validation (empty annotations, unknown target); message catalogue in §4.1                                                                                    | T6-G1 unit cases + T6-G2 AC1 + T6-G2 AC2                      |
| PC01RNF-001 | grep audit (no Java/Spring/Lombok literals introduced inside `internal/infrastructure/fsprofile/auditRuleValidator.go`); message catalogue uses only neutral identifiers                              | §7.6 grep gate (no test, no fixture)                          |

---

## Section 2 — Frozen Domain Contract

PC01US-011 introduces NO new ports, NO new model types, NO new
use-case interfaces, NO new error sentinels, and NO new
`AuditRuleKind` constant. The single addition is one new
**infrastructure-private helper function**
(`validateAuditRuleParams`) plus its call sites in two existing
loaders. The contract below is therefore the reuse contract — what
the validator consumes and what its output must look like to satisfy
the downstream test assertions.

### 2.1 `model.AuditRule` (frozen, unchanged)

```go
// internal/domain/model/auditRule.go (existing — DO NOT MODIFY)

// AuditRule is one declarative rule loaded from the active profile YAML.
// The Params map carries the kind-specific knobs; the evaluator selects
// the keys it needs. Unknown keys are tolerated (forward-compatible).
type AuditRule struct {
    ID          string
    Kind        AuditRuleKind
    Severity    AuditSeverity
    Description string
    Suggestion  string
    Params      map[string]string
}
```

### 2.2 `auditRuleDTO` / `bundleAuditRuleDTO` (frozen, unchanged)

```go
// internal/infrastructure/fsprofile/dto.go::auditRuleDTO and
// internal/infrastructure/fsprofile/bundleDto.go::bundleAuditRuleDTO
// (existing — DO NOT MODIFY).

type auditRuleDTO struct {
    ID          string            `yaml:"id"`
    Kind        string            `yaml:"kind"`
    Severity    string            `yaml:"severity"`
    Description string            `yaml:"description"`
    Suggestion  string            `yaml:"suggestion"`
    Params      map[string]string `yaml:"params"`
}

type bundleAuditRuleDTO struct {
    ID          string            `yaml:"id"`
    Kind        string            `yaml:"kind"`
    Severity    string            `yaml:"severity"`
    Description string            `yaml:"description"`
    Suggestion  string            `yaml:"suggestion"`
    Params      map[string]string `yaml:"params"`
}
```

The two DTOs are structurally identical. The validator helper accepts
the structural subset — see §2.3.

### 2.3 New helper — `validateAuditRuleParams` (frozen contract)

```go
// internal/infrastructure/fsprofile/auditRuleValidator.go (new — T2-G1).
//
// validateAuditRuleParams enforces per-kind structural validation on a
// single audit-rule DTO. It is invoked from both auditLoader.go::
// LoadAuditRules and bundleMapper.go::toBundleDomain after the kind
// is recognised and the severity validated. Return value:
//   - nil when the DTO satisfies all per-kind structural rules;
//   - a non-nil error whose Error() string starts with
//     "rule '<id>': <reason>" when a rule fails validation. The
//     caller is expected to wrap the error with the profile-context
//     prefix and the ErrProfileInvalid sentinel.
//
// The helper is pure — it does not touch the filesystem, does not
// log, and does not depend on model.AuditRule (it consumes the DTO
// directly so it can be called BEFORE the model conversion).
func validateAuditRuleParams(d auditRuleDTO) error
```

**Note:** Go forbids overloading; the helper must accept ONE DTO type.
Because `auditRuleDTO` and `bundleAuditRuleDTO` are structurally
identical, T2-G1 unifies the validator entry point by introducing a
**param-only** signature that takes the fields the validator actually
inspects:

```go
// auditRuleSchema is the structural subset of auditRuleDTO /
// bundleAuditRuleDTO that validateAuditRuleParams inspects. It avoids
// importing one DTO into the other's file and lets both loaders call
// the same helper with a tiny adapter.
type auditRuleSchema struct {
    ID     string
    Kind   string
    Params map[string]string
}

func validateAuditRuleParams(s auditRuleSchema) error
```

Each call site converts inline: `validateAuditRuleParams(auditRuleSchema{ID: d.ID, Kind: d.Kind, Params: d.Params})`.
This stays loyal to the discovery rule "every file in Section 1
appears in exactly one group across Section 9" because the helper
itself ships in T2-G1 and the call-site edits ALSO ship in T2-G1
(file 3 + file 4 in §1).

### 2.4 Error message catalogue (frozen, exact substrings)

The validator emits exactly the following English messages. Each
message starts with `rule '<id>': ` where `<id>` is `auditRuleSchema.ID`
verbatim. The message catalogue is closed — only these four messages
are produced by PC01US-011's validator. Future PC01 stories may
extend the catalogue; this story freezes the initial set.

| ID    | Trigger condition                                                                                                              | Verbatim message                                                                                |
|-------|--------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------|
| M1    | Kind is `required_annotations` AND `splitNonEmpty(Params["annotations"])` is empty (key absent OR empty string OR whitespace)  | `rule '<id>': required_annotations must declare at least one annotation`                          |
| M2    | `Params["target"]` is non-empty AND not in `{"class", "field", "method", "supertype"}`                                         | `rule '<id>': target must be one of [class, field, method, supertype]`                            |
| M3    | (defensive) Kind is `forbidden_annotations` AND `splitNonEmpty(Params["annotations"])` is empty                                | `rule '<id>': forbidden_annotations must declare at least one annotation`                         |
| M4    | (defensive) Kind is `forbidden_field_type_pattern` AND `splitNonEmpty(Params["forbidden_type_patterns"])` is empty             | `rule '<id>': forbidden_field_type_pattern must declare at least one pattern`                     |

Messages M3 and M4 are defensive completeness for kinds that share
the "must declare at least one X" shape with M1; they are NOT pinned
by Gherkin substrings but ARE unit-tested in T6-G1 to lock behaviour.
Adding M3/M4 keeps the helper's logic uniform across kinds with
list-shaped required parameters and avoids the "scenario 3 fixture
silently passes a no-op rule" hazard. Profile authors writing
empty-list rules of any kind get a consistent fatal.

The four messages MUST start with `rule '<id>': ` (single-quoted ID,
trailing colon-space) so the Gherkin substring assertions match
verbatim. Using double quotes or omitting the leading prefix breaks
the .feature contract.

### 2.5 Reserved param keys after PC01US-011

PC01US-011 adds NO new reserved param keys. The closed set after
PC01US-010 (24 keys) is unchanged: `path_scope`, `annotations`,
`expected_values`, `node_types`, `target`, `exempt_paths`,
`triggered_by`, `name_pattern`, `forbidden_type_patterns`,
`expected_supertype`, `args`, `supertype_kind`, `path_required`,
`path_required_any`, `name_suffix`, `name_regex`,
`forbidden_type_suffix`, `forbidden_type_substring`,
`import_prefix`, `implements_glob`, `annotation`,
`non_empty_value_annotations`, plus the kind-implied keys.

### 2.6 `Deps` struct in `internal/cli/wire.go`

**Unchanged.** No new dependency is wired. The
`profilevalidateuc.UseCase` field already exists.

### 2.7 New error sentinels

**None.** The validator wraps the existing
`domerr.ErrProfileInvalid` sentinel via `fmt.Errorf("...: %w: %w",
err, domerr.ErrProfileInvalid)`. This preserves
`errors.Is(err, ErrProfileInvalid)` semantics and continues to flow
through the existing `*domerr.ProfileValidationError` carrier
constructed by `profilevalidateuc.Impl.Execute`.

---

## Section 3 — Domain Layer Plan

**N/A.** No `internal/domain/**` file is created or modified by
PC01US-011. The validator is an infrastructure-layer concern by
guideline rule R2 (path
`internal/infrastructure/fsprofile/**` → Tier 2). The contract that
profile authors see at the YAML level (the four reserved messages of
§2.4) is documented in this plan and re-asserted by the unit tests
in T6-G1; it does NOT need to land as a domain type because the
helper consumes a DTO subset and emits a plain `error`.

---

## Section 4 — Infrastructure Layer Plan

### 4.1 New file — `internal/infrastructure/fsprofile/auditRuleValidator.go` (T2-G1)

```go
// Package fsprofile (auditRuleValidator.go).
//
// Per-kind structural validation of audit-rule DTOs. PC01US-011.
//
// Both audit-rule loaders (auditLoader.go::LoadAuditRules for the legacy
// single-file shape and bundleMapper.go::toBundleDomain for the EP-04
// directory shape) call validateAuditRuleParams immediately after the
// kind whitelist + severity whitelist checks, before constructing the
// model.AuditRule. The helper is pure: no filesystem, no logger, no
// model dependency. It consumes the structural subset of either DTO via
// auditRuleSchema.
//
// Message catalogue (closed for PC01US-011):
//   M1: "rule '<id>': required_annotations must declare at least one annotation"
//   M2: "rule '<id>': target must be one of [class, field, method, supertype]"
//   M3: "rule '<id>': forbidden_annotations must declare at least one annotation"
//   M4: "rule '<id>': forbidden_field_type_pattern must declare at least one pattern"
//
// All four are emitted as plain errors.New(...) values. The caller (each
// loader call site) wraps the error with the profile-context prefix and
// the domerr.ErrProfileInvalid sentinel via fmt.Errorf("%w: %w", err,
// domerr.ErrProfileInvalid).

package fsprofile

import (
    "errors"
    "fmt"
    "strings"

    "github.com/jitctx/jitctx/internal/domain/model"
)

// auditRuleSchema is the structural subset of auditRuleDTO and
// bundleAuditRuleDTO that the validator inspects. Both loader call
// sites convert inline.
type auditRuleSchema struct {
    ID     string
    Kind   string
    Params map[string]string
}

// validAuditRuleTargets is the closed enum for params["target"]. The
// list and ORDER are pinned to match the verbatim substring asserted
// by PC01US-011 AC2 ("[class, field, method, supertype]" — comma-space
// joined, square-bracketed).
var validAuditRuleTargets = []string{"class", "field", "method", "supertype"}

// validateAuditRuleParams enforces per-kind structural validation on
// the given audit-rule descriptor. Returns nil on success.
//
// Per-kind checks (executed in this order — the FIRST failing check
// short-circuits and returns):
//
//   1. params["target"]: when non-empty, must be in
//      validAuditRuleTargets. Applies to ALL kinds (the param is a
//      cross-kind selector — present on required_annotations,
//      forbidden_annotations, method_naming).
//   2. kind == required_annotations: splitNonEmpty(params["annotations"])
//      must be non-empty.
//   3. kind == forbidden_annotations: splitNonEmpty(params["annotations"])
//      must be non-empty.
//   4. kind == forbidden_field_type_pattern:
//      splitNonEmpty(params["forbidden_type_patterns"]) must be
//      non-empty.
//
// Other kinds (interface_naming, forbidden_import,
// field_type_layer_violation, method_naming,
// required_parameterized_supertype, annotation_path_mismatch,
// implements_path_mismatch) currently have NO PC01US-011 schema
// constraints — the helper returns nil for them. Future stories may
// extend the catalogue.
func validateAuditRuleParams(s auditRuleSchema) error {
    if t := strings.TrimSpace(s.Params["target"]); t != "" {
        if !targetIsValid(t) {
            return fmt.Errorf("rule '%s': target must be one of [%s]",
                s.ID, strings.Join(validAuditRuleTargets, ", "))
        }
    }
    switch model.AuditRuleKind(s.Kind) {
    case model.AuditKindRequiredAnnotations:
        if len(splitNonEmpty(s.Params["annotations"])) == 0 {
            return fmt.Errorf(
                "rule '%s': required_annotations must declare at least one annotation",
                s.ID)
        }
    case model.AuditKindForbiddenAnnotations:
        if len(splitNonEmpty(s.Params["annotations"])) == 0 {
            return fmt.Errorf(
                "rule '%s': forbidden_annotations must declare at least one annotation",
                s.ID)
        }
    case model.AuditKindForbiddenFieldTypePattern:
        if len(splitNonEmpty(s.Params["forbidden_type_patterns"])) == 0 {
            return fmt.Errorf(
                "rule '%s': forbidden_field_type_pattern must declare at least one pattern",
                s.ID)
        }
    }
    return nil
}

// targetIsValid checks t against the closed enum; helper extracted
// to keep the message-construction site readable.
func targetIsValid(t string) bool {
    for _, ok := range validAuditRuleTargets {
        if t == ok {
            return true
        }
    }
    return false
}

// splitNonEmpty splits s on commas, trims whitespace from each
// element, and drops empties. Mirrors the function of the same name
// in internal/domain/service/auditRuleEvaluator.go (kept private to
// each package per Go style — duplication is intentional, the two
// functions live in different layers and may diverge in future
// stories). PC01RNF-001 neutrality is preserved: no Java/Spring
// literal is introduced.
func splitNonEmpty(s string) []string {
    if strings.TrimSpace(s) == "" {
        return nil
    }
    parts := strings.Split(s, ",")
    out := make([]string, 0, len(parts))
    for _, p := range parts {
        p = strings.TrimSpace(p)
        if p == "" {
            continue
        }
        out = append(out, p)
    }
    return out
}

// Compile-time assertion: a future maintainer renaming
// model.AuditKindRequiredAnnotations breaks this file (kind switch
// no longer matches), surfacing the dependency at build time.
var _ = errors.New // keep errors import live if a future revision
                   // switches to errors.New for static messages.
```

The file relies on `model` only for the kind-constant compare. There
is NO reverse dependency from the validator into the evaluator —
`splitNonEmpty` is duplicated locally to keep the helper free of any
domain-service import (per `infrastructure-layer-guidelines.yml` —
infrastructure may import `internal/domain/model` and
`internal/domain/errors`, never `internal/domain/service`).

### 4.2 Modified file — `internal/infrastructure/fsprofile/bundleMapper.go` (T2-G1)

Insert one block inside the audit-rules loop (line range 137–160 in
the current file) between the severity check and the
`model.AuditRule` append:

```go
// PC01US-011: per-kind structural validation. Validation runs AFTER
// the kind whitelist and severity whitelist (so fatals fire only on
// recognised kinds) and BEFORE the model.AuditRule conversion.
if err := validateAuditRuleParams(auditRuleSchema{
    ID: d.ID, Kind: d.Kind, Params: d.Params,
}); err != nil {
    return nil, fmt.Errorf("bundle profile %q: audit rule %q: %w: %w",
        dto.Name, d.ID, err, domerr.ErrProfileInvalid)
}
```

The wrapping prefix `bundle profile "<name>": audit rule "<id>": `
appears BEFORE the validator's literal `rule '<id>': ...` substring.
The Gherkin assertions use `Contains`, so the prefix is fine.

### 4.3 Modified file — `internal/infrastructure/fsprofile/auditLoader.go` (T2-G1)

Insert one block inside `LoadAuditRules` (line range 58–81 in the
current file) between the severity check and the
`model.AuditRule` append:

```go
// PC01US-011: per-kind structural validation. Same helper used by
// bundleMapper.toBundleDomain (defence in depth — the legacy
// single-file profile load path should produce the same fatals as
// the EP-04 directory path).
if err := validateAuditRuleParams(auditRuleSchema{
    ID: d.ID, Kind: d.Kind, Params: d.Params,
}); err != nil {
    return nil, fmt.Errorf("profile %q: audit rule %q: %w: %w",
        profileName, d.ID, err, domerr.ErrProfileInvalid)
}
```

### 4.4 Other adapters

**N/A.** `treesitter`, `fsmanifest`, `token`, `fsprofile/bundled.go`,
`fsprofile/detector.go`, `fsprofile/extractor.go`,
`fsprofile/resolver.go`, `fsprofile/bundleAuditRulesAdapter.go`,
`fsprofile/loader.go::Load` (legacy classification loader) — none of
these touch audit-rule params. No edits.

---

## Section 5 — Application Layer Plan

**N/A.** `internal/application/usecase/profilevalidateuc/usecase.go`
is unchanged. The existing flow:

1. Step 3 calls `u.loader.LoadBundle(...)`.
2. When `LoadBundle` returns the new wrapped error, the use case
   appends a `ValidationIssue{Code: classifyLoadErr(loadErr), Message:
   humanizeLoadErr(loadErr)}` to `out.Errors`.
3. `humanizeLoadErr` returns `err.Error()` verbatim for non-
   `ErrProfileYamlMissing` errors, so the message includes the
   `bundle profile "<name>": audit rule "<id>": rule '<id>': <reason>`
   substring chain.
4. `classifyLoadErr` returns `"profile_invalid"` for the wrapped
   `ErrProfileInvalid` (the new fatal `errors.Is`-matches the
   sentinel).
5. Step 5 aggregates the issues into a `*ProfileValidationError` and
   returns it to cobra.

The existing test `TestProfileValidate_MissingTemplate_ExitsOne`
(profileValidateIntegration_test.go:87-104) already proves this
routing works for non-yaml-missing, non-name-missing fatals. The
new fatal rides the same path.

---

## Section 6 — Presentation Layer Plan

**N/A.** `profileValidateCmd.go` is unchanged.
`internal/cli/format/errors.go::TranslateError` is unchanged. The
existing `*ProfileValidationError` rendering branch (lines 87-99)
emits:

```
profile "<path>": <N> error(s)
  - <msg1>
  - <msg2>
  ...
```

For PC01US-011 AC1, `<msg1>` is the verbatim
`bundle profile "<bundleName>": audit rule "X": rule 'X': required_annotations must declare at least one annotation: profile is invalid`
chain. The integration test asserts on the substring
`rule 'X': required_annotations must declare at least one annotation`,
which appears intact within that line.

---

## Section 7 — Composition Root + Tests Plan

### 7.1 Composition root

**N/A.** `internal/cli/wire.go`, `internal/cli/root.go`,
`internal/cli/execute.go`, `cmd/jitctx/main.go`,
`internal/config/**` are unchanged.

### 7.2 Unit tests — `internal/infrastructure/fsprofile/auditRuleValidator_test.go` (T6-G1)

A single test file with table-driven `t.Parallel()` cases exercising
`validateAuditRuleParams` directly. The helper is pure, so no
fixture files and no I/O.

```go
package fsprofile

import (
    "testing"

    "github.com/stretchr/testify/require"
)

func TestValidateAuditRuleParams_Table(t *testing.T) {
    t.Parallel()
    cases := []struct {
        name      string
        in        auditRuleSchema
        wantErr   bool
        wantMsg   string // substring match when wantErr is true
    }{
        // M1 — required_annotations / empty annotations.
        {
            name: "M1_RequiredAnnotations_EmptyAnnotations",
            in: auditRuleSchema{
                ID: "X", Kind: "required_annotations",
                Params: map[string]string{"annotations": ""},
            },
            wantErr: true,
            wantMsg: "rule 'X': required_annotations must declare at least one annotation",
        },
        {
            name: "M1_RequiredAnnotations_AbsentAnnotations",
            in: auditRuleSchema{
                ID: "X", Kind: "required_annotations",
                Params: map[string]string{},
            },
            wantErr: true,
            wantMsg: "rule 'X': required_annotations must declare at least one annotation",
        },
        {
            name: "M1_RequiredAnnotations_WhitespaceAnnotations",
            in: auditRuleSchema{
                ID: "X", Kind: "required_annotations",
                Params: map[string]string{"annotations": "  ,  "},
            },
            wantErr: true,
            wantMsg: "rule 'X': required_annotations must declare at least one annotation",
        },
        {
            name: "M1_RequiredAnnotations_NonEmpty_Pass",
            in: auditRuleSchema{
                ID: "ok", Kind: "required_annotations",
                Params: map[string]string{"annotations": "A"},
            },
            wantErr: false,
        },
        // M2 — target enum.
        {
            name: "M2_UnknownTarget_Foo",
            in: auditRuleSchema{
                ID: "X", Kind: "forbidden_annotations",
                Params: map[string]string{
                    "annotations": "Some",
                    "target":      "foo",
                },
            },
            wantErr: true,
            wantMsg: "rule 'X': target must be one of [class, field, method, supertype]",
        },
        {
            name: "M2_UnknownTarget_FiresBeforeKindCheck",
            // Even with a kind-specific failure pending, target check
            // runs first by §4.1 ordering.
            in: auditRuleSchema{
                ID: "X", Kind: "required_annotations",
                Params: map[string]string{
                    "annotations": "",
                    "target":      "bogus",
                },
            },
            wantErr: true,
            wantMsg: "rule 'X': target must be one of [class, field, method, supertype]",
        },
        {
            name: "M2_KnownTarget_Class_Pass",
            in: auditRuleSchema{
                ID: "ok", Kind: "forbidden_annotations",
                Params: map[string]string{
                    "annotations": "Some",
                    "target":      "class",
                },
            },
            wantErr: false,
        },
        {
            name: "M2_KnownTarget_Field_Pass",
            in: auditRuleSchema{
                ID: "ok", Kind: "forbidden_annotations",
                Params: map[string]string{
                    "annotations": "Some",
                    "target":      "field",
                },
            },
            wantErr: false,
        },
        {
            name: "M2_KnownTarget_Method_Pass",
            in: auditRuleSchema{
                ID: "ok", Kind: "method_naming",
                Params: map[string]string{
                    "triggered_by": "Test",
                    "name_pattern": "^should.*",
                    "target":       "method",
                },
            },
            wantErr: false,
        },
        {
            name: "M2_KnownTarget_Supertype_Pass",
            in: auditRuleSchema{
                ID: "ok", Kind: "required_parameterized_supertype",
                Params: map[string]string{
                    "implements_glob": "*UseCase",
                    "args":            "*,*",
                    "target":          "supertype",
                },
            },
            wantErr: false,
        },
        {
            name: "M2_TargetAbsent_Pass",
            in: auditRuleSchema{
                ID: "ok", Kind: "required_annotations",
                Params: map[string]string{"annotations": "A"},
            },
            wantErr: false,
        },
        // M3 — forbidden_annotations / empty annotations.
        {
            name: "M3_ForbiddenAnnotations_EmptyAnnotations",
            in: auditRuleSchema{
                ID: "X", Kind: "forbidden_annotations",
                Params: map[string]string{"annotations": ""},
            },
            wantErr: true,
            wantMsg: "rule 'X': forbidden_annotations must declare at least one annotation",
        },
        // M4 — forbidden_field_type_pattern / empty patterns.
        {
            name: "M4_ForbiddenFieldTypePattern_EmptyPatterns",
            in: auditRuleSchema{
                ID: "X", Kind: "forbidden_field_type_pattern",
                Params: map[string]string{"forbidden_type_patterns": ""},
            },
            wantErr: true,
            wantMsg: "rule 'X': forbidden_field_type_pattern must declare at least one pattern",
        },
        // Other kinds — currently NO PC01US-011 constraints.
        {
            name: "OtherKind_MethodNaming_NoConstraints_Pass",
            in: auditRuleSchema{
                ID: "ok", Kind: "method_naming",
                Params: map[string]string{
                    "triggered_by": "Test",
                    "name_pattern": "^should.*",
                },
            },
            wantErr: false,
        },
        {
            name: "OtherKind_RequiredParameterizedSupertype_NoConstraints_Pass",
            in: auditRuleSchema{
                ID: "ok", Kind: "required_parameterized_supertype",
                Params: map[string]string{"implements_glob": "*UseCase"},
            },
            wantErr: false,
        },
        {
            name: "OtherKind_InterfaceNaming_NoConstraints_Pass",
            in: auditRuleSchema{
                ID:     "ok",
                Kind:   "interface_naming",
                Params: map[string]string{},
            },
            wantErr: false,
        },
    }
    for _, tc := range cases {
        tc := tc
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            err := validateAuditRuleParams(tc.in)
            if tc.wantErr {
                require.Error(t, err)
                require.Contains(t, err.Error(), tc.wantMsg)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

The test file lives in package `fsprofile` (white-box) so it can
exercise the unexported helper directly. This is the same posture as
`auditLoader_test.go`, `bundleMapper_test.go`, and other existing
fsprofile tests.

### 7.3 Parser unit tests

**N/A.** PC01US-011 does not touch the parser.

### 7.4 Integration tests — `internal/cli/command/profileValidateIntegration_test.go` (T6-G2)

Append three new functions to the existing test file. Each follows
the same shape as the existing `TestProfileValidate_*` cases (see
profileValidateIntegration_test.go:40-147 for the template). Each
test:

- Calls `t.Parallel()`.
- Builds the cobra command via the existing `buildProfileValidateCmd`
  helper (lines 20-30 of the existing file).
- Uses `fixtureDir(t, "pc01us011ProfileValidateNewSchema",
  "<scenario>")` — `fixtureDir` already supports variadic path parts.
- Sets `cmd.SetArgs([]string{fixture})`.
- Calls `cmd.ExecuteContext(context.Background())`.
- Asserts on `err` / `stdout` / `stderr`.

#### Test 1 — `TestProfileValidate_PC01US011_EmptyRequiredAnnotations_ExitsOne` (AC1)

```go
fixture := fixtureDir(t, "pc01us011ProfileValidateNewSchema",
    "emptyRequiredAnnotations")
// build cmd, capture stdout/stderr, execute
err := cmd.ExecuteContext(context.Background())
require.Error(t, err)
require.Contains(t, err.Error(),
    "rule 'X': required_annotations must declare at least one annotation")
```

**Important:** the existing tests use `err.Error()` because
`format.TranslateError` wraps the error into `errors.New(string)`
before cobra prints it (see profileValidateIntegration_test.go:78-80
for the documented rationale). The substring assertion targets the
returned cobra error, NOT the captured `stderr` buffer (cobra prints
the error to its `errOrStderr` after `RunE` returns; the `bytes.Buffer`
captured via `cmd.SetErr` may or may not contain it depending on
cobra version, so existing tests assert on `err.Error()` — same
posture here). Maps to AC1.

#### Test 2 — `TestProfileValidate_PC01US011_UnknownTarget_ExitsOne` (AC2)

```go
fixture := fixtureDir(t, "pc01us011ProfileValidateNewSchema",
    "unknownTarget")
err := cmd.ExecuteContext(context.Background())
require.Error(t, err)
require.Contains(t, err.Error(),
    "rule 'X': target must be one of [class, field, method, supertype]")
```

Maps to AC2.

#### Test 3 — `TestProfileValidate_PC01US011_ValidFullSchema_ExitsZero` (AC3)

```go
fixture := fixtureDir(t, "pc01us011ProfileValidateNewSchema",
    "validFullSchema")
err := cmd.ExecuteContext(context.Background())
require.NoError(t, err, "valid full-schema profile must validate cleanly")
require.Contains(t, stdout.String(), "Profile valid")
```

Maps to AC3.

### 7.5 Fixtures (T6-G3, T6-G4, T6-G5)

Naming convention follows PC01US-007 / PC01US-008 / PC01US-009 /
PC01US-010: lower-camelCase project root segments matching
`pc01us011ProfileValidateNewSchema`. testdata is gitignored;
integration-test author force-adds with `git add -f` when committing.

Each scenario tree has shape:

```
<scenario>/
├── profile.yaml
└── templates/
    └── .gitkeep        # placeholder so the empty dir tracks
```

The `templates/` directory is required by the EP-04 bundle loader
(`loadFromFS` calls `fs.ReadDir(fsys, "templates")`; missing dir is
tolerated, but a present-empty dir is the simplest way to keep the
fixture deterministic across platforms). The `.gitkeep` is a
zero-byte file.

#### `emptyRequiredAnnotations/profile.yaml` (T6-G3 — AC1)

```yaml
name: pc01us011-empty-required-annotations
language: java
types: []
audit_rules:
  - id: X
    kind: required_annotations
    severity: ERROR
    description: 'PC01US-011 AC1 fixture'
    suggestion: 'declare at least one annotation'
    params:
      path_scope: src/
      annotations: ''
```

The `id: X` matches the Gherkin's literal `'X'`. The `kind:
required_annotations` is whitelisted; the empty `annotations` value
triggers M1.

#### `unknownTarget/profile.yaml` (T6-G4 — AC2)

```yaml
name: pc01us011-unknown-target
language: java
types: []
audit_rules:
  - id: X
    kind: forbidden_annotations
    severity: ERROR
    description: 'PC01US-011 AC2 fixture'
    suggestion: 'use a known target value'
    params:
      path_scope: src/
      annotations: 'Some'
      target: foo
```

The `target: foo` triggers M2; the rule passes the kind whitelist
and severity whitelist, and would otherwise be valid (annotations is
non-empty), so M2's "target check fires first" ordering is the only
reason this fixture fails. This locks the §4.1 ordering by integration
test (in addition to the unit test case
`M2_UnknownTarget_FiresBeforeKindCheck`).

#### `validFullSchema/profile.yaml` (T6-G5 — AC3)

```yaml
name: pc01us011-valid-full-schema
language: java
types: []
audit_rules:
  # required_annotations
  - id: ra-1
    kind: required_annotations
    severity: ERROR
    description: 'all-of presence rule'
    suggestion: 'declare {required}'
    params:
      path_scope: src/main/java/com/acme/application/
      annotations: 'A,B'

  # forbidden_annotations (target=field encoding for "field_rules")
  - id: fa-1
    kind: forbidden_annotations
    severity: ERROR
    description: 'forbidden field annotation'
    suggestion: 'remove {forbidden}'
    params:
      path_scope: src/main/java/
      annotations: 'C'
      target: field

  # method_naming (encoding for "method_rules")
  - id: mn-1
    kind: method_naming
    severity: WARNING
    description: 'naming pattern'
    suggestion: 'rename method to match pattern'
    params:
      path_scope: src/test/java/
      triggered_by: 'D'
      name_pattern: '^should[A-Z].*'

  # required_parameterized_supertype (encoding for "supertype_rules")
  - id: rps-1
    kind: required_parameterized_supertype
    severity: ERROR
    description: 'parameterized supertype'
    suggestion: 'extend the parameterized base'
    params:
      path_scope: src/main/java/com/acme/application/usecase/
      supertype_kind: implements
      implements_glob: '*UseCase'
      args: '*,*'

  # forbidden_field_type_pattern (encoding for "forbidden_field_types")
  - id: ffp-1
    kind: forbidden_field_type_pattern
    severity: ERROR
    description: 'forbidden parameterized field type'
    suggestion: 'replace with a domain-pure type'
    params:
      path_scope: src/main/java/com/acme/domain/
      forbidden_type_patterns: 'List<*Entity>'
```

This profile uses every PC01-introduced kind. Each rule has a
well-formed `params` block (`annotations` non-empty for M1/M3, no
`target` mismatches, `forbidden_type_patterns` non-empty for M4).
The `language: java` line is required by EP04US-005's loader, and
`types: []` keeps the fixture minimal (no template files needed).

The Gherkin scenario 3 asks for "required_annotations,
forbidden_annotations, field_rules, method_rules, supertype_rules,
forbidden_field_types". Per discovery finding #2 in §0,
`field_rules` is encoded as `forbidden_annotations` with
`target: field`; that mapping is documented inline in the fixture's
comments and re-asserted by the unit test
`M2_KnownTarget_Field_Pass`.

### 7.6 Engine-neutrality grep gate (PC01RNF-001)

Before declaring T2-G1 done, run the cumulative grep gate from
PC01RNF-001:

```bash
grep -rE "(Lombok|Spring|Mockito|Autowired|JPA)" \
    internal/infrastructure/fsprofile/auditRuleValidator.go \
    internal/infrastructure/fsprofile/bundleMapper.go \
    internal/infrastructure/fsprofile/auditLoader.go \
    internal/domain \
    internal/application \
    internal/cli
```

This MUST return zero new matches in the listed paths. PC01US-011
adds NO Java/Spring literal anywhere in the engine — the validator
only emits neutral identifiers (`required_annotations`, `target`,
`class`, `field`, `method`, `supertype`).

The framework-specific identifiers in the AC3 fixture
(`com.acme.application`, `com.acme.domain`, `*UseCase`, `*Entity`)
live ONLY in the YAML fixture under `testdata/`, which is excluded
from the grep gate (testdata is the canonical place for
framework-specific examples — same posture every PC01 story took).

---

## Section 8 — Open Questions & Risks

All questions were pre-resolved during discovery — none are blocking.

- **Q1 — Should the validator also check `params["annotations"]`
  when the kind is `required_annotations` AND `params["target"] !=
  "class"`?** Pre-resolved: **No.** The Gherkin pins only the
  empty-list case for M1; the cross-kind `target` enum check (M2)
  fires for ANY rule, regardless of kind. Adding extra cross-cutting
  checks risks rejecting profiles that were previously silently
  tolerated. Stay minimal — only ship checks pinned by acceptance
  scenarios plus the defensive M3/M4 completeness twins. Blocking:
  No.

- **Q2 — Should line/column be surfaced in the error messages?**
  Pre-resolved: **deferred (Q-OPTIONAL).** PC01RF-011 mentions
  "schema errors with line/column" as a business rule, but the
  Gherkin scenarios for PC01US-011 do NOT pin line/column substrings.
  Adding line/column would require a two-pass YAML decode (yaml.Node
  tree → targeted `Line`/`Column` lookup at the offending mapping).
  This is sizeable and out of scope for PC01US-011. A follow-up story
  can introduce a `params:` mapping-node visitor and re-emit fatals
  with line/column anchors. Blocking: No.

- **Q3 — Should the validator reject UNKNOWN param keys (typo
  detection)?** Pre-resolved: **No.** The current loader is
  forward-compatible by design (`model.AuditRule.Params` doc-comment
  lines 80-81: "Unknown keys are tolerated"). Adding a closed param
  whitelist would break older custom profiles that carry future-
  schema keys. Typo detection on `params` is tracked as a separate
  concern; it is NOT in PC01US-011's acceptance scope. Blocking:
  No.

- **Q4 — Does the `target: foo` fixture risk being rejected for an
  earlier reason (e.g. unrecognised `kind` in the EP-03 single-file
  loader)?** Pre-resolved: **No.** The `forbidden_annotations`
  kind is in `knownAuditRuleKinds` (mapper.go:18) — recognised since
  PC01US-004. The severity `ERROR` is in `knownAuditSeverities`. The
  `params.annotations: 'Some'` value is well-formed (M3 doesn't fire).
  So the FIRST validation gate the rule fails is M2, exactly as AC2
  pins. Locked in by integration test 2 + unit-test case
  `M2_UnknownTarget_FiresBeforeKindCheck`. Blocking: No.

- **Q5 — Why not extend the bundled `spring-boot-hexagonal` profile
  with a PC01US-011 rule?** Pre-resolved: **same posture as
  PC01US-002 / PC01US-004 / PC01US-005 / PC01US-006 / PC01US-007 /
  PC01US-008 / PC01US-009 / PC01US-010.** PC01US-011 is a validator
  story, not a rule story. It does not introduce a new
  `AuditRuleKind` and does not author a new evaluator rule. The
  bundled profile is unchanged. Blocking: No.

- **Q6 — What about the third PC01RF-011 example "missing kind on
  supertype_rules"?** Pre-resolved: **out of scope (Q-OPTIONAL).**
  The Gherkin scenarios only pin M1 (empty required_annotations)
  and M2 (unknown target). The "missing kind" case is structurally
  enforced by the EP-04 directory loader's `KnownFields(false)` +
  the existing `knownAuditRuleKinds` whitelist (which already drops
  unknown kinds with a WARN log). PC01RF-011's third example would
  require additional Gherkin coverage; revisit when such coverage
  ships. Blocking: No.

- **Q7 — Why the `auditRuleSchema` adapter struct?** Pre-resolved:
  Go does not support overloading, and `auditRuleDTO` /
  `bundleAuditRuleDTO` are structurally identical but distinct named
  types. Refactoring one to alias the other risks breaking
  consumers (the YAML field tags must survive). The lightweight
  adapter struct keeps the validator decoupled from either DTO and
  documented as a structural subset. The conversion is a one-line
  literal at each call site — zero runtime cost. Blocking: No.

- **Q8 — Why duplicate `splitNonEmpty` instead of importing from the
  domain service?** Pre-resolved: per
  `infrastructure-layer-guidelines.yml`, infrastructure adapters
  may import `internal/domain/model` and `internal/domain/errors`
  but NOT `internal/domain/service`. The guideline's rationale is
  that services are the layer above infra; importing them creates a
  cyclical dependency hazard. `splitNonEmpty` is a 12-line pure
  helper; duplication is the correct trade. The two functions may
  diverge in future stories (e.g. infrastructure may want to keep a
  trailing whitespace, domain may not). Blocking: No.

- **Q9 — What error code does the use case attach for the new
  fatals?** Pre-resolved: `classifyLoadErr(err)` (usecase.go:274-285)
  falls through to the default branch and returns
  `"profile_invalid"` for any error that wraps `ErrProfileInvalid`
  but is neither `ErrProfileYamlMissing`, `*TemplateMissingError`,
  nor `ErrLanguageUnsupported`. The new validator wraps
  `ErrProfileInvalid` directly, so the code is `"profile_invalid"`.
  This is the same code already used for unknown-severity, missing-
  type-id, and other generic profile-invalid fatals. No new code
  needed. Blocking: No.

- **Risk R1 — The integration test relies on `err.Error()` rather
  than the captured stderr buffer.** Pre-resolved: existing tests
  (e.g. `TestProfileValidate_MissingTemplate_ExitsOne`,
  profileValidateIntegration_test.go:87-104) document this rationale
  in a comment (line 78-80): "format.TranslateError wraps
  ProfileValidationError into errors.New(string), so errors.As
  cannot find the typed error at this point. Assert on the rendered
  message string which carries the canonical literal." The new tests
  follow the same pattern. The Gherkin says "stderr contains ..." —
  the rendered error IS what cobra writes to stderr; asserting on
  `err.Error()` is functionally equivalent under the existing
  `format.TranslateError` shape. Blocking: No.

- **Risk R2 — A profile with multiple bad rules.** Pre-resolved:
  the bundle mapper returns on the FIRST validation error
  (auditLoader returns the same way). For PC01US-011's three
  scenarios this is sufficient — each fixture has at most one bad
  rule. A follow-up story may aggregate per-rule fatals into a
  multi-issue carrier; out of scope here.

No `Blocking: Yes` entries. Discovery proceeds to implementation.

---

## Section 9 — Parallel Execution Plan (authoritative for `@agent-manager`)

```yaml
tiers:
  - id: 2
    name: Infrastructure — audit-rule schema validator + loader call sites
    depends_on: []
    groups:
      - id: T2-G1
        scope:
          create:
            - internal/infrastructure/fsprofile/auditRuleValidator.go
          modify:
            - internal/infrastructure/fsprofile/bundleMapper.go
            - internal/infrastructure/fsprofile/auditLoader.go
        guidelines:
          - .claude/guidelines/infrastructure-layer-guidelines.yml
        effort: M
        notes: >
          Pure helper validateAuditRuleParams in auditRuleValidator.go
          consuming the structural subset auditRuleSchema (id, kind,
          params). Closed message catalogue M1-M4 per Section 2.4.
          Order of checks (FIRST failing returns): (1) target enum,
          (2) kind-specific empty-list checks. Both call sites (one
          in bundleMapper.go::toBundleDomain, one in
          auditLoader.go::LoadAuditRules) wrap the helper error with
          a profile-context prefix and the domerr.ErrProfileInvalid
          sentinel via fmt.Errorf("%w: %w", err,
          domerr.ErrProfileInvalid). The helper duplicates
          splitNonEmpty locally (no domain-service import — per
          infrastructure-layer-guidelines.yml). Engine-neutrality
          grep gate (Section 7.6) MUST pass before this group is
          declared done — no Lombok/Spring/Mockito/Autowired/JPA
          literal inside any of the three files.

  - id: 6
    name: Tests + fixtures (parallel) — unit, integration, and three fixture trees
    depends_on: [2]
    groups:
      - id: T6-G1
        scope:
          create:
            - internal/infrastructure/fsprofile/auditRuleValidator_test.go
          modify: []
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: M
        notes: >
          White-box t.Parallel() table test in package fsprofile
          covering every message in the catalogue plus passing
          variants. Fifteen subcases total (see Section 7.2). The
          M2_UnknownTarget_FiresBeforeKindCheck case locks the
          target-first ordering. The OtherKind_* cases lock the
          "no constraints on this kind" non-failure path. The
          helper is pure — no fixtures, no I/O, no logger.

      - id: T6-G2
        scope:
          create: []
          modify:
            - internal/cli/command/profileValidateIntegration_test.go
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: M
        notes: >
          Three new t.Parallel() functions appended to the existing
          file:
          (1) TestProfileValidate_PC01US011_EmptyRequiredAnnotations_ExitsOne
              — fixture pc01us011ProfileValidateNewSchema/emptyRequiredAnnotations,
              asserts err.Error() contains
              "rule 'X': required_annotations must declare at least one annotation".
          (2) TestProfileValidate_PC01US011_UnknownTarget_ExitsOne
              — fixture pc01us011ProfileValidateNewSchema/unknownTarget,
              asserts err.Error() contains
              "rule 'X': target must be one of [class, field, method, supertype]".
          (3) TestProfileValidate_PC01US011_ValidFullSchema_ExitsZero
              — fixture pc01us011ProfileValidateNewSchema/validFullSchema,
              asserts NoError and stdout contains "Profile valid".
          Reuses the existing buildProfileValidateCmd and fixtureDir
          helpers; no new helper plumbing needed.

      - id: T6-G3
        scope:
          create:
            - testdata/pc01us011ProfileValidateNewSchema/emptyRequiredAnnotations/profile.yaml
            - testdata/pc01us011ProfileValidateNewSchema/emptyRequiredAnnotations/templates/.gitkeep
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          AC1 fixture. profile.yaml declares a single audit rule
          (id: X, kind: required_annotations, severity: ERROR,
          params: {path_scope: src/, annotations: ''}). The empty
          annotations triggers validator M1. templates/.gitkeep is
          a zero-byte placeholder so the empty templates/ directory
          tracks in git (testdata is gitignored — force-add with
          git add -f when committing).

      - id: T6-G4
        scope:
          create:
            - testdata/pc01us011ProfileValidateNewSchema/unknownTarget/profile.yaml
            - testdata/pc01us011ProfileValidateNewSchema/unknownTarget/templates/.gitkeep
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          AC2 fixture. profile.yaml declares a single audit rule
          (id: X, kind: forbidden_annotations, severity: ERROR,
          params: {path_scope: src/, annotations: 'Some',
          target: foo}). The unknown target triggers validator M2.
          Note: annotations is non-empty so M3 does NOT fire — the
          target check fires first per the Section 4.1 ordering.

      - id: T6-G5
        scope:
          create:
            - testdata/pc01us011ProfileValidateNewSchema/validFullSchema/profile.yaml
            - testdata/pc01us011ProfileValidateNewSchema/validFullSchema/templates/.gitkeep
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          AC3 fixture. profile.yaml declares one well-formed rule
          per PC01-introduced kind (required_annotations,
          forbidden_annotations target=field, method_naming,
          required_parameterized_supertype, forbidden_field_type_pattern).
          Encoding mapping: field_rules → forbidden_annotations
          target=field; method_rules → method_naming;
          supertype_rules → required_parameterized_supertype;
          forbidden_field_types → forbidden_field_type_pattern.
          types: [] keeps the fixture minimal (no template files
          required). All audit rules have non-empty annotations /
          patterns so M1/M3/M4 do not fire; no rule sets a target
          outside the {class, field, method, supertype} enum so M2
          does not fire. Expected outcome: exit 0, stdout contains
          "Profile valid".
```
