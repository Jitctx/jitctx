# Plan — PC01US-009 Require @SpringBootTest + @Testcontainers + @ActiveProfiles("test") on Integration-Test Base

## Section 0 — Summary

- Feature: **assert that an integration-test base class declares all
  three required annotations together AND that one of them carries an
  exact literal positional argument**, e.g. profile rule
  `integration-test-base` requires `[SpringBootTest, Testcontainers,
  ActiveProfiles]` AND `expected_values: 'ActiveProfiles="test"'`. This
  is a generic, language-neutral evaluator capability already shipped by
  the existing `required_annotations` rule kind (PC01US-002 introduced
  the kind; PC01US-006 added the `expected_values` parameter that
  produces argument-mismatch evidence). PC01US-009 ratifies the existing
  contract with profile-authoring fixtures and integration tests; no
  domain or infrastructure code is added.
- User Story: **PC01US-009**.
- Requirement IDs covered:
  - **PC01RF-001** — all-of presence semantics (existing
    `required_annotations` evaluator). Re-asserted by AC1 / AC3.
  - **PC01RF-007** — annotation-with-arguments matching (existing
    `expected_values` parameter on `required_annotations`). Re-asserted
    by AC2.
  - **PC01RF-009** — evidence-rich messages (`missing=[Testcontainers]`
    for AC3; `annotation=ActiveProfiles, expected_value=...,
    actual=...` for AC2). Re-asserted by all three ACs.
  - **PC01RNF-001** — engine language-neutrality. The strings
    `SpringBootTest`, `Testcontainers`, `ActiveProfiles`, `Spring` ONLY
    appear in (a) profile YAML under `testdata/`, (b) `.java` fixture
    files, (c) integration-test literal-substring assertions, and
    (d) rule descriptions inside the YAML profile. Zero new occurrences
    inside `internal/domain` or `internal/application`.
  - **PC01RNF-003** — deterministic output. The
    `evalRequiredAnnotations` function already emits violations in
    rule-declared order: missing-violation FIRST, then per-pair
    `expected_values` mismatches in declaration order. Re-asserted by
    the integration tests via fixed fixture+profile.
  - **PC01RNF-006** — real Tree-sitter parse on real `.java` fixtures
    via integration tests.
- Acceptance scenarios mapped 1:1 in §7:
  - **AC1** (clean) — class `BaseIntegrationTest` declares
    `@SpringBootTest @Testcontainers @ActiveProfiles("test")` →
    zero violations.
  - **AC2** (wrong arg) — same three annotations but
    `@ActiveProfiles("prod")` → one violation whose evidence contains
    the substrings `annotation=ActiveProfiles`, `expected_value=`,
    `actual=`, plus `prod` and the rule-supplied expected value. AC2's
    verbatim phrasing `expected_value=test, actual=prod` is approximated
    by the evaluator's literal output `expected_value="test",
    actual="prod"` (the parser captures string-literal arguments
    verbatim INCLUDING quotes — see §8 Q3 for the resolution).
  - **AC3** (missing annotation) — class declares
    `@SpringBootTest @ActiveProfiles("test")` only (no
    `@Testcontainers`) → one violation whose evidence contains the
    literal substring `missing=[Testcontainers]`.
- Layers touched: **tests + fixtures only**. No domain change. No
  infrastructure change. No application change. No presentation change.
  No wiring change. The story is a pure ratification of the existing
  `required_annotations` kind.
- Tiers active: **6 only**. Tiers 1, 2, 3, 4, 5 are explicitly `N/A`.
  - No `internal/domain/**` modification → Tier 1 absent.
  - No `internal/infrastructure/**` modification → Tier 2 absent.
    `knownAuditRuleKinds` already whitelists
    `AuditKindRequiredAnnotations` (PC01US-002).
  - No `internal/application/**` modification → Tier 3 absent.
  - No new cobra command, no formatter change → Tier 4 absent.
  - No `internal/cli/{wire,root,execute}.go`, no
    `cmd/jitctx/main.go`, no `internal/config/**` change → Tier 5
    absent.
- Guidelines loaded:
  - `.claude/guidelines/integration-test-layer-guidelines.yml`
  - `.claude/guidelines/unit-test-layer-guidelines.yml`
- Estimated file count: **17 new** (1 integration test + 16 fixture
  files across 3 fixture trees) and **0 modified**. The plan file
  itself is not counted in §1.

> **Discovery finding (load-bearing).** Every part of PC01US-009's
> AC1/AC2/AC3 is **already implemented** by the existing
> `evalRequiredAnnotations` function in
> `internal/domain/service/auditRuleEvaluator.go`. The evaluator
> already supports:
>
> - `path_scope` (substring filter on `summary.Path`),
> - `annotations` (comma-joined required-set; produces the
>   `missing=[...]` evidence in declaration order — covers AC3),
> - `expected_values` (comma-joined `Annotation=Value` pairs; emits one
>   additional violation per mismatched pair with the literal text
>   `annotation=<ann>, expected_value=<expected>, actual=<actual>` —
>   covers AC2),
> - `node_types` (defaults to `class_declaration` — covers AC1's class
>   target),
> - integration of the above through the dispatch arm at line 45 of
>   `EvaluateFile`.
>
> The infrastructure profile loader
> (`internal/infrastructure/fsprofile/mapper.go`) already whitelists
> `model.AuditKindRequiredAnnotations` in `knownAuditRuleKinds` since
> PC01US-002. The audit use case (`internal/application/usecase/
> audituc/usecase.go`) already calls `AuditEvaluator.EvaluateFile` for
> every parsed file. No code change is required.
>
> Consequently this plan does NOT regenerate any production code. Its
> purpose is to (a) ratify the existing engine capability via three
> new integration-test scenarios + three Java fixture trees that
> exercise AC1/AC2/AC3 end-to-end with real Tree-sitter parsing, and
> (b) document the engine-neutrality posture (PC01RNF-001) — none of
> the new test artefacts introduce framework identifiers into
> `internal/domain` or `internal/application`.

---

## Section 1 — File Set

| #  | File                                                                                                                                         | Action  | Layer  | Tier | Group  | Requirements |
|----|----------------------------------------------------------------------------------------------------------------------------------------------|---------|--------|------|--------|--------------|
| 1  | `internal/cli/command/integrationTestBaseRequiredAnnotationsIntegration_test.go`                                                              | create  | tests  | 6    | T6-G1  | PC01US-009, PC01RF-001, PC01RF-007, PC01RF-009, PC01RNF-006 |
| 2  | `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectClean/pom.xml`                                                              | create  | tests  | 6    | T6-G2  | PC01RNF-006 |
| 3  | `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectClean/project-state.yaml`                                                   | create  | tests  | 6    | T6-G2  | PC01RNF-006 |
| 4  | `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml`                          | create  | tests  | 6    | T6-G2  | PC01US-009, PC01RF-001, PC01RF-007 |
| 5  | `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectClean/src/test/java/com/acme/it/BaseIntegrationTest.java`                   | create  | tests  | 6    | T6-G2  | PC01US-009 |
| 6  | `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectWrongActiveProfile/pom.xml`                                                 | create  | tests  | 6    | T6-G3  | PC01RNF-006 |
| 7  | `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectWrongActiveProfile/project-state.yaml`                                      | create  | tests  | 6    | T6-G3  | PC01RNF-006 |
| 8  | `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectWrongActiveProfile/.jitctx/profiles/spring-boot-hexagonal.yaml`             | create  | tests  | 6    | T6-G3  | PC01US-009, PC01RF-007 |
| 9  | `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectWrongActiveProfile/src/test/java/com/acme/it/BaseIntegrationTest.java`     | create  | tests  | 6    | T6-G3  | PC01US-009, PC01RF-009 |
| 10 | `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectMissingTestcontainers/pom.xml`                                              | create  | tests  | 6    | T6-G4  | PC01RNF-006 |
| 11 | `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectMissingTestcontainers/project-state.yaml`                                   | create  | tests  | 6    | T6-G4  | PC01RNF-006 |
| 12 | `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectMissingTestcontainers/.jitctx/profiles/spring-boot-hexagonal.yaml`          | create  | tests  | 6    | T6-G4  | PC01US-009, PC01RF-001 |
| 13 | `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectMissingTestcontainers/src/test/java/com/acme/it/BaseIntegrationTest.java`  | create  | tests  | 6    | T6-G4  | PC01US-009, PC01RF-009 |

Coverage notes:

- The integration test (file #1) loads each of the three project trees
  via `copyFixture` from `helpers_test.go`, runs `audit` against the
  temp workdir, and asserts on stdout. Each fixture tree has identical
  shape: `pom.xml` (Spring Boot detector trigger), `project-state.yaml`
  (schema_version 2, single module), `.jitctx/profiles/spring-boot-
  hexagonal.yaml` (canonical FULL profile shape — see §7.5), and ONE
  `BaseIntegrationTest.java` under `src/test/java/com/acme/it/`.
- Three fixture trees, each in its own group (T6-G2..G4), so the
  fixture-authoring work is parallelisable and conflict-free. The test
  file (T6-G1) is independent of the fixture content — it can be
  authored in parallel with the fixtures because it only references
  fixture paths, never their text content.
- Every requirement ID in scope (PC01US-009, PC01RF-001, PC01RF-007,
  PC01RF-009, PC01RNF-001, PC01RNF-003, PC01RNF-006) appears in at
  least one file row above OR is enforced by the §3.4-style
  engine-neutrality grep gate documented in §7.6.

Requirement coverage trace (every ID in scope appears below):

| Requirement   | Where it lives in code (already shipped)                                                                                | Where this plan re-asserts it       |
|---------------|--------------------------------------------------------------------------------------------------------------------------|--------------------------------------|
| PC01US-009    | `evalRequiredAnnotations` in `auditRuleEvaluator.go` (PC01US-002 + PC01US-006 extensions)                                | T6-G1 three integration scenarios    |
| PC01RF-001    | `missingAnnotations` helper produces deterministic `missing=[...]` subset                                                | T6-G2 (clean) + T6-G4 (missing)      |
| PC01RF-007    | `parseExpectedValues` + per-pair iteration in `evalRequiredAnnotations` (PC01US-006)                                     | T6-G2 (clean arg) + T6-G3 (wrong arg)|
| PC01RF-009    | `makeViolation` substitution context emits `missing=[…]` and `annotation=…, expected_value=…, actual=…` literals         | T6-G1 stdout substring assertions    |
| PC01RNF-001   | grep audit (no Spring/Testcontainers/ActiveProfiles literals in `internal/domain` or `internal/application`)             | §7.6 grep gate (no test, no fixture) |
| PC01RNF-003   | rule list iteration in declaration order; per-pair iteration in `parseExpectedValues` slice order                        | T6-G1 fixed-fixture determinism      |
| PC01RNF-006   | real Tree-sitter parse via existing `treesitter.Parser` adapter                                                          | T6-G2/G3/G4 real `.java` fixtures    |

---

## Section 2 — Frozen Domain Contract

This contract is **already in main**. It is reproduced here so any
future change in this area must keep these symbols, parameter keys, and
substitution-token spellings intact (or open a follow-up RFC). No new
contract is introduced by PC01US-009.

### 2.1 `model.AuditKindRequiredAnnotations` (frozen)

```go
// internal/domain/model/auditRule.go (existing — DO NOT MODIFY)

// AuditKindRequiredAnnotations enforces that every declaration in the
// rule's path_scope carries every annotation in params["annotations"].
// PC01US-002 introduced the kind; PC01US-006 added the
// expected_values parameter that produces argument-mismatch evidence
// per PC01RF-007.
AuditKindRequiredAnnotations AuditRuleKind = "required_annotations"
```

The constant is unchanged by PC01US-009. The infrastructure mapper's
`knownAuditRuleKinds` whitelist already accepts it.

### 2.2 `evalRequiredAnnotations` parameter contract (frozen)

The parameter keys consumed by `evalRequiredAnnotations` (in
`internal/domain/service/auditRuleEvaluator.go`) are FROZEN as of
PC01US-006:

| Key               | Required? | Semantics (verbatim from current source) |
|-------------------|-----------|------------------------------------------|
| `path_scope`      | yes       | substring filter on `summary.Path` (e.g. `src/test/java/com/acme/it/`) |
| `annotations`     | yes       | comma-joined simple names (no leading `@`) that must ALL be present on every matching declaration; order is preserved and used to derive deterministic `missing=[...]` evidence |
| `expected_values` | no        | comma-joined `Annotation=Value` pairs; for each pair, when the annotation is present on a matching declaration, the evaluator compares `decl.AnnotationArgs[ann]` against the value. A mismatch emits ONE additional violation per pair with evidence `annotation=<ann>, expected_value=<expected>, actual=<actual>`. Determinism: pairs iterated in input-string order. |
| `node_types`      | no        | comma-joined declaration node-type filter; default `class_declaration`; `*` wildcards |

**No new param keys are introduced by PC01US-009.** The full reserved-
key registry after PC01US-008 (carried over verbatim — no additions):

| Key                          | Used by                                                                                                 |
|------------------------------|---------------------------------------------------------------------------------------------------------|
| `path_scope`                 | forbidden_import, field_type_layer_violation, required_annotations, forbidden_annotations, method_naming, forbidden_field_type_pattern, required_parameterized_supertype |
| `annotations`                | required_annotations, forbidden_annotations                                                             |
| `expected_values`            | required_annotations                                                                                    |
| `node_types`                 | required_annotations, forbidden_annotations, method_naming, forbidden_field_type_pattern, required_parameterized_supertype |
| `target`                     | forbidden_annotations                                                                                   |
| `exempt_paths`               | forbidden_annotations, method_naming, forbidden_field_type_pattern, required_parameterized_supertype   |
| `triggered_by`               | method_naming                                                                                           |
| `name_pattern`               | method_naming                                                                                           |
| `forbidden_type_patterns`    | forbidden_field_type_pattern                                                                            |
| `expected_supertype`         | required_parameterized_supertype                                                                        |
| `args`                       | required_parameterized_supertype                                                                        |
| `supertype_kind`             | required_parameterized_supertype                                                                        |
| `path_required`              | annotation_path_mismatch, interface_naming                                                              |
| `path_required_any`          | implements_path_mismatch                                                                                |
| `name_suffix`                | interface_naming                                                                                        |
| `name_regex`                 | interface_naming                                                                                        |
| `forbidden_type_suffix`      | field_type_layer_violation                                                                              |
| `forbidden_type_substring`   | field_type_layer_violation                                                                              |
| `import_prefix`              | forbidden_import                                                                                        |
| `implements_glob`            | implements_path_mismatch                                                                                |
| `annotation`                 | annotation_path_mismatch                                                                                |

PC01US-009 does NOT introduce a new key. Profile authors scope the rule
to the integration-test base class by setting `path_scope` to the
fixture's directory `src/test/java/com/acme/it/` — the test fixture's
ONLY class under that path is `BaseIntegrationTest`, so the rule
naturally targets the base class without a name filter.

### 2.3 Substitution context (frozen)

`evalRequiredAnnotations` populates the following substitution tokens
for each emitted violation. The integration-test profile YAML uses
exactly these tokens in its `description` template:

For the **missing-violation** path (AC3):

| Token        | Value (verbatim from current source) |
|--------------|---------------------------------------|
| `{file}`     | `summary.Path`                       |
| `{name}`     | declaration simple name              |
| `{required}` | comma-joined `params["annotations"]` |
| `{evidence}` | `missing=[A,B,...]` subset NOT present, ordered by `params["annotations"]` |
| `{missing}`  | identical to `{evidence}` for backward compat with templates authored before PC01US-006 |

For the **arg-mismatch violation** path (AC2):

| Token        | Value (verbatim from current source) |
|--------------|---------------------------------------|
| `{file}`     | `summary.Path`                       |
| `{name}`     | declaration simple name              |
| `{required}` | comma-joined `params["annotations"]` |
| `{evidence}` | `annotation=<ann>, expected_value=<expected>, actual=<actual>` where `<actual>` is `decl.AnnotationArgs[<ann>]` (may include surrounding quotes for string-literal args — see §8 Q3) |

Both paths share `RuleID`, `Kind`, `Severity`, `ModuleID`, and
`FilePath`. The renderer (`internal/cli/format/audit.go`) emits the
rule ID as the literal `[<rule-id>]` token in stdout, which the
integration test asserts on.

### 2.4 No changes to other contracts

- `model.JavaDeclaration`, `model.AuditRule`, `auditvo.AuditViolation`
  — all unchanged.
- `internal/cli/wire.go` `Deps` struct — unchanged. The audit use case
  already injects `service.AuditEvaluator{}`.
- No new error sentinels, no new typed errors.
- The bundled `spring-boot-hexagonal` profile is **NOT** modified by
  this story (same posture as PC01US-002/004/005/006/007/008). Profile
  content evolves under EP-04. Profile authors enable the rule by
  editing their own `.jitctx/profiles/*.yaml`.
- `internal/infrastructure/fsprofile/mapper.go` `knownAuditRuleKinds`
  — unchanged. `AuditKindRequiredAnnotations` is already whitelisted.

---

## Section 3 — Domain Layer Plan

**N/A.** No domain types, ports, use cases, services, or errors are
introduced or modified by PC01US-009. The capability already lives at
`internal/domain/service/auditRuleEvaluator.go:331-394` (the body of
`evalRequiredAnnotations`) and was last extended by PC01US-006 when the
`expected_values` parameter and arg-mismatch evidence path were added.

The engine-neutrality posture (PC01RNF-001) is enforced by §7.6 — the
strings `Spring`, `SpringBootTest`, `Testcontainers`, `ActiveProfiles`
must never appear inside `internal/domain` or `internal/application`.
The grep gate documented there is the contract.

---

## Section 4 — Infrastructure Layer Plan

**N/A.** No infrastructure adapter is added or modified. The profile
loader already accepts the `required_annotations` kind; the
`auditRuleDTO` already passes `params: map[string]string` through
verbatim (so `path_scope`, `annotations`, `expected_values`,
`node_types` ride through unchanged). The Tree-sitter parser already
populates `JavaDeclaration.AnnotationArgs` for every annotation that
carries a positional argument (PC01US-006 verified this via parser
unit tests in `internal/infrastructure/treesitter/parser_test.go`).

---

## Section 5 — Application Layer Plan

**N/A.** `appaudituc.Impl.Execute` already iterates parsed files,
calls `AuditEvaluator.EvaluateFile`, sorts the union via the existing
`AuditRuleFilter`, and emits the deterministic
`AuditProjectOutput`. No edit required.

---

## Section 6 — Presentation Layer Plan

**N/A.** No new cobra command, no formatter change. The `audit` command
already prints violations via the existing renderer
(`internal/cli/format/audit.go`), which substitutes `{file}`, `{name}`,
`{required}`, `{evidence}`, and `{missing}` tokens through
`makeViolation` → `substituteSuggestion`. The stdout/stderr contract is
unchanged: violations render under `## Sintatic Violations`; the rule
ID is emitted as the literal `[integration-test-base]` token in each
violation line; the integration tests assert on those substrings.

---

## Section 7 — Composition Root + Tests Plan

### 7.1 Composition root

**N/A.** `internal/cli/wire.go`, `root.go`, `execute.go`,
`cmd/jitctx/main.go`, and `internal/config/**` are all unchanged. The
`Deps` struct is unchanged.

### 7.2 Unit tests

**N/A — coverage already exists.** The existing
`internal/domain/service/auditRuleEvaluator_test.go` already covers
`evalRequiredAnnotations` for both the missing-set path (PC01US-002,
PC01US-003) and the `expected_values` mismatch path (PC01US-006),
including determinism of pair iteration order. PC01US-009 does NOT add
unit tests because no new code path is created — every code path is
already proven by the PC01US-002/003/006 unit tests on the same
function. Adding more table cases for the same function with
`Spring/Testcontainers` literals would push framework strings into the
domain test file, which sits ADJACENT to `internal/domain/` in the
package tree and is the historical convention for "engine-neutral
domain code". The integration tests in §7.4 are the proper home for
those literals (PC01RNF-001 explicitly excludes `testdata/**`,
integration tests, and rule descriptions from the proscribed-string
set).

### 7.3 Parser unit tests

**N/A.** `internal/infrastructure/treesitter/parser_test.go` already
proves `JavaDeclaration.AnnotationArgs` is populated for string-
literal arguments (`TestParser_ClassWithExtendWithArg_PopulatesAnnotationArgs`
asserts the verbatim text `"User service tests"` for a
`@DisplayName("User service tests")` annotation). PC01US-009's
`@ActiveProfiles("test")` follows the same code path; no new parser
test is needed.

### 7.4 Integration tests (T6-G1 — `internal/cli/command/integrationTestBaseRequiredAnnotationsIntegration_test.go`)

Three test functions, each:

- `t.Parallel()`.
- Builds a real `audit` cobra command via a local helper modelled on
  `newAuditCmdForJpaEntityContract` /
  `newAuditCmdForUseCaseParameterizedSupertype` (Q-DRY: a local copy
  is acceptable per the no-upstream-refactor rule established by
  PC01US-007 / PC01US-008). The helper wires:
  - `fsprofile.NewDetectorWithLogger(profilesDir, logger)`
  - `fsprofile.NewAuditRulesLoader(profilesDir, logger)`
  - `fsmanifest.New(manifestPath)`
  - `treesitter.New()` for both `JavaParser` injection points
  - `treesitter.NewWalker()`
  - `service.NewAuditEvaluator()`
  - `fsconfig.New(logger)`
  - `service.NewAuditRuleFilter()`
  - `fsprofile.NewBundleAuditRulesAdapter()`, `fsprofile.NewBundled()`,
    `fsprofile.NewBundleLoader(logger, nil)`, `fsprofile.NewResolver(...)`
- Uses `t.TempDir()` + `copyFixture` from `helpers_test.go`.
- Asserts on `stdout` via `require.Contains` and
  `strings.Count(stdout, "[integration-test-base]")`.

The shared local helper is `newAuditCmdForIntegrationTestBaseRequiredAnnotations`
(camelCase per project filename convention).

#### Test 1 — `TestAuditCmd_Integration_IntegrationTestBaseRequiredAnnotations_AllThreePresentNoViolation` (AC1)

- Fixture: `pc01us009IntegrationTestBaseRequiredAnnotations/projectClean`.
- Manifest path: `<tempDir>/project-state.yaml`.
- Command: `audit`.
- Expectations:
  - `err == nil` (the audit command exits zero on a clean project per
    EP03US-002 contract).
  - `stdout` does NOT contain `[integration-test-base]`.
  - `stdout` does NOT contain `missing=`.
  - `stdout` does NOT contain `annotation=ActiveProfiles`.

Maps to AC1 ("BaseIntegrationTest with all three annotations and
correct profile passes").

#### Test 2 — `TestAuditCmd_Integration_IntegrationTestBaseRequiredAnnotations_WrongActiveProfileFiresArgMismatch` (AC2)

- Fixture: `pc01us009IntegrationTestBaseRequiredAnnotations/projectWrongActiveProfile`.
- Expectations:
  - `stdout` contains `[integration-test-base]`.
  - `strings.Count(stdout, "[integration-test-base]") == 1` (no
    spurious second emission — the missing-violation path does NOT
    fire because all three annotations are present; only the
    arg-mismatch path fires).
  - `stdout` contains the literal substring
    `annotation=ActiveProfiles`.
  - `stdout` contains the literal substring
    `expected_value="test"` (with embedded quotes — see Q3).
  - `stdout` contains the literal substring
    `actual="prod"` (with embedded quotes — see Q3).
  - `stdout` does NOT contain `missing=`.

Maps to AC2 ("BaseIntegrationTest with @ActiveProfiles(\"prod\") fails
on argument mismatch"). The AC's verbatim phrasing
`expected_value=test, actual=prod` (no quotes) is approximated by the
evaluator's actual output `expected_value="test", actual="prod"` — see
§8 Q3 for the resolution. The substrings asserted above contain
`expected_value=` and `test` and `actual=` and `prod`, all of which
are present in the produced violation message.

#### Test 3 — `TestAuditCmd_Integration_IntegrationTestBaseRequiredAnnotations_MissingTestcontainersFiresWithEvidence` (AC3)

- Fixture: `pc01us009IntegrationTestBaseRequiredAnnotations/projectMissingTestcontainers`.
- Expectations:
  - `stdout` contains `[integration-test-base]`.
  - `strings.Count(stdout, "[integration-test-base]") == 1`.
  - `stdout` contains the literal substring `missing=[Testcontainers]`
    (verbatim — this matches the AC's wording exactly because the
    `missing` token is built by `strings.Join(missing, ",")` over the
    declaration-order subset, which contains exactly one element here).
  - `stdout` does NOT contain `annotation=ActiveProfiles` (the
    `expected_values` mismatch path is short-circuited because
    `ActiveProfiles` IS present and IS valued `"test"` in this
    fixture).

Maps to AC3 ("BaseIntegrationTest missing @Testcontainers fails").

### 7.5 Fixtures (T6-G2, T6-G3, T6-G4 — three trees under `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/`)

Naming convention follows PC01US-007 / PC01US-008: lower-camelCase
project root segments matching `pc01us009IntegrationTestBaseRequiredAnnotations`.
testdata is gitignored (project convention); the integration test
author force-adds with `git add -f` when committing.

Each project tree has the same shape:

```
projectXxx/
├── pom.xml                         # contains org.springframework.boot for module detection
├── project-state.yaml              # schema_version: 2; one module; one file
├── .jitctx/
│   └── profiles/
│       └── spring-boot-hexagonal.yaml   # ONE audit rule: integration-test-base (PLUS the FULL canonical profile)
└── src/
    └── test/
        └── java/
            └── com/acme/it/
                └── BaseIntegrationTest.java
```

**Critical: the profile YAML MUST carry the FULL canonical
`spring-boot-hexagonal` shape** — `name`, `languages`, `query_lang`,
`detect`, `module_detection`, `rules` (the classification rules), AND
`audit_rules`. A profile with ONLY `audit_rules` will not be activated
by the auto-detector — the detector reads `detect.files` to decide
whether the profile applies to the project, and the resolver requires
`name`, `languages`, and `query_lang`. This is the same fixture-content
requirement that PC01US-007 / PC01US-008 documented; copy the canonical
shape from `testdata/pc01us008UseCaseParameterizedSupertype/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml`
verbatim, replacing only the `audit_rules` block.

The full canonical profile, with the PC01US-009 `audit_rules` payload,
is identical across the three fixtures EXCEPT for the `expected_values`
spec in `projectMissingTestcontainers` (where the YAML retains the same
arg-matching spec — but the fixture's `.java` file omits
`@Testcontainers`, so the arg-matching path is short-circuited because
the missing-violation path fires first). The canonical YAML body the
fixtures share:

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
  - id: integration-test-base
    kind: required_annotations
    severity: ERROR
    description: 'Integration-test base {name} contract violation: {evidence}'
    suggestion: 'Annotate {name} with all of [{required}] and ActiveProfiles="test"'
    params:
      path_scope: src/test/java/com/acme/it/
      annotations: 'SpringBootTest,Testcontainers,ActiveProfiles'
      expected_values: 'ActiveProfiles="test"'
```

Notes on the audit rule:

- `path_scope: src/test/java/com/acme/it/` — substring filter that
  matches the ONE base-class file in each fixture. The fixture
  intentionally puts ONLY `BaseIntegrationTest.java` under this path so
  no extra name-pattern parameter is required.
- `annotations: 'SpringBootTest,Testcontainers,ActiveProfiles'` —
  comma-joined required-set. The order is **declaration order**, which
  drives the deterministic `missing=[...]` output. Profile authors who
  add a fourth annotation extend this list.
- `expected_values: 'ActiveProfiles="test"'` — the value side is the
  verbatim source text the parser captures, which for a Java string
  literal `"test"` is the four characters `"test"` (the literal
  quotation marks are preserved). The YAML scalar uses single-quoted
  outer quotes so the embedded double quotes round-trip cleanly.
  - Determinism note: the
    `parseExpectedValues` helper splits on `,` then on the FIRST `=`,
    so the value side `"test"` (containing no comma) is parsed as a
    single pair with `Expected == "test"` (with quotes). This matches
    `decl.AnnotationArgs["ActiveProfiles"]` produced by the parser for
    `@ActiveProfiles("test")`. AC1 / AC3 fixtures keep this value;
    AC2's fixture changes the source to `@ActiveProfiles("prod")`,
    making the parser's captured text `"prod"` — the mismatch fires.
- `description` template uses `{name}` (declaration name) and
  `{evidence}` (auto-populated to the missing-set string for the
  missing-violation path, OR to
  `annotation=<ann>, expected_value=<expected>, actual=<actual>` for
  the arg-mismatch path). The integration test asserts the literal
  substrings produced by both paths.

`pom.xml` content (minimum to satisfy module-detection — copy the shape
used by PC01US-007 / PC01US-008 fixtures):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.acme</groupId>
  <artifactId>pc01us009</artifactId>
  <version>0.0.1-SNAPSHOT</version>
  <parent>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-parent</artifactId>
    <version>3.2.0</version>
  </parent>
</project>
```

`project-state.yaml` skeleton (`schema_version: 2`, one module, one
file — copy shape from `pc01us008UseCaseParameterizedSupertype`
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
  - id: com.acme.it
    path: src/test/java/com/acme/it
    tags: []
    contracts:
      - name: BaseIntegrationTest
        types:
          - service
        path: src/test/java/com/acme/it/BaseIntegrationTest.java
        methods: []
    dependencies: []
contexts: []
```

The `contracts[].types: [service]` value is a placeholder used to make
the manifest pass schema validation; the audit use case does NOT use
the `types` field for `required_annotations` evaluation (it iterates
parsed files via the walker, not via the manifest contracts). The same
trick is used in PC01US-007 / PC01US-008 fixtures.

#### `.java` content per fixture

**projectClean / BaseIntegrationTest.java** (AC1 — passing):

```java
package com.acme.it;

import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.ActiveProfiles;
import org.testcontainers.junit.jupiter.Testcontainers;

@SpringBootTest
@Testcontainers
@ActiveProfiles("test")
public abstract class BaseIntegrationTest {
}
```

The class is `abstract` to mirror real-world IT-base usage; the
evaluator does not care about the modifier.

**projectWrongActiveProfile / BaseIntegrationTest.java** (AC2 — wrong arg):

```java
package com.acme.it;

import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.ActiveProfiles;
import org.testcontainers.junit.jupiter.Testcontainers;

@SpringBootTest
@Testcontainers
@ActiveProfiles("prod")
public abstract class BaseIntegrationTest {
}
```

All three required annotations present (so no missing-violation
fires); `ActiveProfiles` carries the wrong arg `"prod"`, so the
arg-mismatch path emits exactly one violation.

**projectMissingTestcontainers / BaseIntegrationTest.java** (AC3 — missing annotation):

```java
package com.acme.it;

import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.ActiveProfiles;

@SpringBootTest
@ActiveProfiles("test")
public abstract class BaseIntegrationTest {
}
```

`@Testcontainers` is intentionally omitted (and its import). The
missing-violation path emits exactly one violation with evidence
`missing=[Testcontainers]`. The arg-mismatch path is also reachable
because `ActiveProfiles` IS present and DOES match `"test"` — so it
emits zero arg-mismatch violations. Total violations on this fixture:
exactly one (the missing-violation).

The `path_scope` substring `src/test/java/com/acme/it/` matches all
three fixtures' `.java` paths. The walker emits forward-slash paths
already (proven by EP-01 / EP-03 integration tests).

### 7.6 Engine-neutrality grep gate (PC01RNF-001)

Before declaring the story done, run the cumulative grep gate from
PC01RNF-001:

```bash
grep -rE "Spring|Testcontainers|ActiveProfiles|SpringBootTest" \
    internal/domain internal/application
```

This MUST return zero matches. The four strings are confined to:

- `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/**` (the
  three fixture trees);
- `internal/cli/command/integrationTestBaseRequiredAnnotationsIntegration_test.go`
  (the integration-test literal-substring assertions);
- the rule `description` template inside fixture YAML (which the
  renderer substitutes at runtime).

This is the same posture PC01US-002 / PC01US-005 / PC01US-006 adopted
for `Lombok` / `Mockito` / `JUnit`. No `internal/domain` file or
`internal/application` file gains a new framework literal.

---

## Section 8 — Open Questions & Risks

All questions were pre-resolved during discovery — none are blocking.

- **Q1 — Does PC01US-009 need a NEW audit-rule kind?**
  Pre-resolved: **No.** The existing `AuditKindRequiredAnnotations`
  with the `expected_values` parameter (PC01US-006) covers AC1 (all-of
  presence on a `class_declaration`), AC2 (arg mismatch via the
  arg-mismatch evidence path), and AC3 (missing-set evidence). A new
  kind would duplicate the existing function. Discovery confirmed by
  reading `evalRequiredAnnotations` lines 331–394: the function emits
  the exact `missing=[...]` and
  `annotation=<ann>, expected_value=<expected>, actual=<actual>`
  literals AC2/AC3 require. Blocking: No.

- **Q2 — Does the rule need a `name_pattern` parameter to scope to
  base classes?** Pre-resolved: **No.** The fixture trees place ONLY
  `BaseIntegrationTest.java` under `src/test/java/com/acme/it/`, so
  `path_scope: src/test/java/com/acme/it/` naturally targets the base
  class. Real-world adopters who keep multiple test classes under the
  same path can EITHER (a) narrow `path_scope` to a more specific
  substring like `src/test/java/com/acme/it/base/` or (b) use the
  filename in the path: `path_scope: BaseIntegrationTest.java`. Adding
  a `name_pattern` parameter is OUT of scope for this story; if a
  future requirement needs name-based scoping on
  `required_annotations`, it lands as a separate story (e.g. a
  hypothetical PC01US-016) that adds the key cumulatively. Blocking:
  No.

- **Q3 — AC2's literal phrasing `expected_value=test, actual=prod`
  vs the evaluator's actual output
  `expected_value="test", actual="prod"` (with quotes).** Pre-
  resolved: **the evaluator's output is correct as-is, and the
  integration tests assert on substring presence rather than the
  full AC string.** The Tree-sitter parser captures string-literal
  arguments verbatim INCLUDING quotes (proven by
  `TestParser_ClassWithExtendWithArg_PopulatesAnnotationArgs` which
  asserts `decl.AnnotationArgs["DisplayName"] == "\"User service tests\""`
  for `@DisplayName("User service tests")`). The PC01US-006 contract
  was ratified with this quoting behaviour intact. Stripping the
  quotes inside the evaluator would (a) be a behaviour change to the
  PC01US-006 contract, (b) lose information (a profile author may want
  to distinguish a string literal `"42"` from an integer literal `42`),
  and (c) break the existing PC01US-006 unit tests. The integration
  test for AC2 therefore asserts on these load-bearing substrings:
  `annotation=ActiveProfiles`, `expected_value=`, `test`,
  `actual=`, `prod`. All five appear verbatim in the rendered
  violation message. The AC's exact phrasing is documented as
  approximation — the same posture PC01US-008 took for AC3
  (`expected_arity=2, actual=1` was approximated by
  `expected_arity=2` plus `actual_arity=1`). Blocking: No.

- **Q4 — Why is no unit test added for `evalRequiredAnnotations`?**
  Pre-resolved: PC01US-002 + PC01US-006 already added unit tests
  covering both the missing-set path AND the arg-mismatch path on the
  same function. Adding more table cases with `Spring`/
  `Testcontainers`/`ActiveProfiles` literals would push framework
  strings into the domain test file — and while the domain tests are
  technically allowed to reference fixture/test strings, the project
  convention since PC01RNF-001 has been to keep framework literals in
  `testdata/`, integration tests, and rule descriptions. The
  integration tests in §7.4 are the proper home; they exercise the
  function via the real composition root with real Tree-sitter
  parsing on real Java fixtures (PC01RNF-006). Blocking: No.

- **Q5 — Should the bundled `spring-boot-hexagonal` profile gain this
  rule?** Pre-resolved: **No**, same posture as
  PC01US-002/004/005/006/007/008. The bundled profile evolves
  separately under EP-04. This story ships fixtures + integration
  tests; profile authors adopting PC01US-009 enable the rule by
  editing their own `.jitctx/profiles/*.yaml`. Blocking: No.

- **Q6 — Walker scope.** Pre-resolved: fixtures live under
  `src/test/java/com/acme/it/...`; `path_scope:
  src/test/java/com/acme/it/` is the substring filter the integration
  tests rely on. The walker emits paths with forward slashes
  (proven). The walker DOES traverse `src/test/java` by default — it
  walks every `.java` file in the project. Blocking: No.

- **Q7 — `expected_values` parsing of the embedded `=` in
  `ActiveProfiles="test"`.** Pre-resolved: `parseExpectedValues`
  splits on `,` first, then on the FIRST `=`. The piece
  `ActiveProfiles="test"` therefore parses as
  `Annotation="ActiveProfiles"`, `Expected=\"test\"` (the second `=`
  inside the value is ignored — there is no second `=` here, so the
  pair is well-formed). Confirmed by reading
  `parseExpectedValues` lines 701–733 of the evaluator source:
  `strings.Cut(piece, "=")` returns the first split. Blocking: No.

- **Q8 — Profile YAML fixture content depth.** Pre-resolved: the
  profile YAML carries the FULL canonical `spring-boot-hexagonal`
  shape (name, languages, query_lang, detect, module_detection,
  rules, audit_rules). Audit_rules-only YAML would not be detected by
  the auto-detector. This is the same fixture-content requirement
  PC01US-007 / PC01US-008 documented. The fixture authors copy the
  canonical shape from `testdata/pc01us008.../projectClean/.jitctx/
  profiles/spring-boot-hexagonal.yaml` verbatim, replacing only the
  `audit_rules` block. Documented in §7.5. Blocking: No.

- **Q9 — `BaseIntegrationTest` is `abstract` — does the evaluator
  treat abstract classes differently?** Pre-resolved: **No.** The
  parser populates `decl.NodeType = "class_declaration"` for both
  abstract and concrete classes (Tree-sitter does not produce a
  separate node type for `abstract`). The evaluator's default
  `node_types: ["class_declaration"]` filter matches the abstract
  base. Blocking: No.

- **Q10 — Multiple violations on the same fixture.** Pre-resolved:
  the evaluator emits the missing-violation FIRST and per-pair
  arg-mismatch violations IN ORDER. For AC2's fixture, the
  missing-violation does NOT fire (all three required annotations
  present), so exactly one arg-mismatch violation emits. For AC3's
  fixture, the missing-violation fires once
  (`missing=[Testcontainers]`), and the arg-mismatch path emits zero
  violations (`ActiveProfiles` is present and matches `"test"`). The
  integration tests' `strings.Count(stdout, "[integration-test-base]")
  == 1` assertion catches any regression that doubles the violation
  count. Blocking: No.

- **Q11 — Engine-neutrality grep gate scope.** Pre-resolved: the
  cumulative gate from PC01RNF-001 enumerates `Lombok|Spring|Mockito|
  Autowired|JPA`. PC01US-009 expands the de-facto banlist for
  in-codebase verification with `Testcontainers`, `ActiveProfiles`,
  and `SpringBootTest` — all framework-specific identifiers that must
  not bleed into `internal/domain` or `internal/application`. The
  grep gate documented in §7.6 covers this. The strings are confined
  to `testdata/`, the integration test, and rule descriptions.
  Blocking: No.

- **Risk R1 — Tree-sitter capture of `@ActiveProfiles("test")`
  argument.** Already proven by PC01US-006. The parser's
  `findFirstPositionalArg` walks the `annotation_argument_list`
  child and returns the verbatim text of the first positional arg.
  For `@ActiveProfiles("test")`, the captured text is `"test"` (with
  quotes). PC01US-006's
  `TestParser_ClassWithExtendWithArg_PopulatesAnnotationArgs`
  exercises the same code path on `@DisplayName("User service tests")`
  and asserts on `"User service tests"` (with quotes). Mitigation:
  the integration test asserts on `expected_value="test"` and
  `actual="prod"` substrings WITH quotes — the assertions match the
  evaluator's actual output exactly.

- **Risk R2 — Comma in `expected_values` value.** The parser splits
  `expected_values` on `,` first, then on `=`. A profile author who
  needed `ActiveProfiles="a,b"` would silently get truncated parsing.
  PC01US-009 fixtures use the single-token value `"test"` /
  `"prod"`, so this risk is non-blocking for AC1/AC2/AC3. The
  limitation is documented in `evalRequiredAnnotations`'s doc-comment
  (line 297-299). Mitigation: the AC text uses single-token values
  only.

No `Blocking: Yes` entries. Discovery proceeds to implementation.

---

## Section 9 — Parallel Execution Plan (authoritative for `@agent-manager`)

```yaml
tiers:
  - id: 6
    name: Tests + fixtures (parallel) — tests-only ratification of existing required_annotations engine
    depends_on: []
    groups:
      - id: T6-G1
        scope:
          create:
            - internal/cli/command/integrationTestBaseRequiredAnnotationsIntegration_test.go
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: M
        notes: >
          Three test functions (AC1 clean, AC2 wrong-arg, AC3
          missing-Testcontainers), each t.Parallel(). Local helper
          newAuditCmdForIntegrationTestBaseRequiredAnnotations modelled
          on the PC01US-007 / PC01US-008 helpers (no upstream DRY
          refactor). Loads each fixture via copyFixture, runs `audit`
          against the temp workdir, asserts on stdout. AC2 asserted
          via the load-bearing substrings annotation=ActiveProfiles,
          expected_value=, test, actual=, prod (with quotes per Q3).
          AC3 asserted via the verbatim substring missing=[Testcontainers].
          AC1 asserted via absence of [integration-test-base] AND
          missing= AND annotation=ActiveProfiles in stdout.
          Engine-neutrality grep gate (no Spring / Testcontainers /
          ActiveProfiles / SpringBootTest in internal/domain,
          internal/application) MUST pass before this group is declared
          done — see §7.6.

      - id: T6-G2
        scope:
          create:
            - testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectClean/pom.xml
            - testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectClean/project-state.yaml
            - testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectClean/src/test/java/com/acme/it/BaseIntegrationTest.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Clean fixture for AC1. BaseIntegrationTest declares all three
          required annotations with @ActiveProfiles("test"). Profile
          contains the full canonical spring-boot-hexagonal shape (name,
          languages, query_lang, detect, module_detection, rules) PLUS
          the single integration-test-base audit rule with annotations
          'SpringBootTest,Testcontainers,ActiveProfiles' and
          expected_values 'ActiveProfiles="test"'. testdata is
          gitignored — author force-adds when committing.

      - id: T6-G3
        scope:
          create:
            - testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectWrongActiveProfile/pom.xml
            - testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectWrongActiveProfile/project-state.yaml
            - testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectWrongActiveProfile/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectWrongActiveProfile/src/test/java/com/acme/it/BaseIntegrationTest.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Violating fixture for AC2 — BaseIntegrationTest declares all
          three required annotations BUT with @ActiveProfiles("prod")
          instead of "test". The arg-mismatch path of the evaluator
          fires; the integration test asserts the substrings
          annotation=ActiveProfiles, expected_value="test",
          actual="prod" in stdout (with quotes per Q3). Profile YAML
          identical to projectClean's profile.

      - id: T6-G4
        scope:
          create:
            - testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectMissingTestcontainers/pom.xml
            - testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectMissingTestcontainers/project-state.yaml
            - testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectMissingTestcontainers/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectMissingTestcontainers/src/test/java/com/acme/it/BaseIntegrationTest.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Violating fixture for AC3 — BaseIntegrationTest declares only
          @SpringBootTest and @ActiveProfiles("test"); @Testcontainers
          is intentionally omitted (and its import). The missing-set
          path of the evaluator fires; the integration test asserts
          the verbatim substring missing=[Testcontainers] in stdout.
          The arg-mismatch path is short-circuited because
          ActiveProfiles is present and matches "test". Profile YAML
          identical to projectClean's profile.
```

---

## Self-Validation Checklist

**File-set coverage**
- Every file in §1 appears exactly once across §9 groups (cross-checked:
  T6-G1 has 1, T6-G2 has 4, T6-G3 has 4, T6-G4 has 4 — total 13,
  matching §1's 13 rows).
- Every requirement ID (PC01US-009, PC01RF-001, PC01RF-007, PC01RF-009,
  PC01RNF-001, PC01RNF-003, PC01RNF-006) appears in at least one §1
  row OR in the §1 traceability matrix.
- No file path appears in two groups.

**Frozen contract**
- No new ports, model types, use-case interfaces, or error sentinels
  are introduced. The frozen contract in §2 documents the EXISTING
  surface (`AuditKindRequiredAnnotations`, the four parameter keys,
  the two violation paths' substitution contexts). Every symbol
  referenced lives in main as of commit 8746a60 (PC01US-008 landing).
- `Deps` struct in `internal/cli/wire.go` is unchanged — explicitly
  noted in §2.4 and §7.1.
- No fields marked `TODO` or `{placeholder}` in the frozen contract.

**DAG**
- `depends_on` edges: T6-G1..G4 → ∅ (each group depends on no other).
  Acyclic.
- Tier 1 omitted because no `internal/domain/**` file appears in §1.
- Tier 2 omitted because no `internal/infrastructure/**` file
  appears in §1.
- Tier 3 omitted because no `internal/application/**` file appears
  in §1.
- Tier 4 omitted because no `internal/cli/command/*Cmd.go` or
  `internal/cli/format/*.go` file appears in §1.
- Tier 5 omitted because no wiring file (`wire.go`, `root.go`,
  `execute.go`, `main.go`, `internal/config/**`) appears in §1.
- All `guidelines[]` paths exist under
  `/workspaces/jitctx/.claude/guidelines/` (verified:
  `integration-test-layer-guidelines.yml`,
  `unit-test-layer-guidelines.yml`).

**Open questions**
- Zero `Blocking: Yes` entries. Discovery is unblocked.
