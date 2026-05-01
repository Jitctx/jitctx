# Plan — PC01US-013 Deterministic Violation Output

## Section 0 — Summary

- Feature: **assert that two consecutive runs of `jitctx audit` over the
  same fixed Java fixture and the same fixed profile produce
  byte-identical stdout**, with violations ordered deterministically by
  the documented sort key. This is a generic capability already shipped
  by the audit pipeline: the `lessViolation` comparator in
  `internal/application/usecase/audituc/usecase.go` sorts the violation
  slice before it leaves the use case, and the renderer in
  `internal/cli/format/audit.go` is pure (same input → byte-identical
  output). PC01US-013 ratifies this contract with a PC01-flavoured
  cross-feature integration test plus a dedicated PC01 fixture so the
  determinism evidence is visible alongside the other PC01 stories.
- User Story: **PC01US-013**.
- Requirement IDs covered:
  - **PC01RNF-003** — deterministic output. Two runs over identical
    input produce byte-identical output; violation order is
    "ascending file path, ascending line, ascending rule ID"
    (per the PRD line 196-200). Re-asserted by the new
    integration test.
  - **PC01RNF-006** — real Tree-sitter parse on real `.java`
    fixtures. The new fixture lives under `testdata/` and is
    consumed via `copyFixture`.
- Acceptance Criterion mapped 1:1 in §7.4:
  - **AC** ("Two consecutive runs produce identical output") — copy
    the dedicated fixture into `t.TempDir()`, run `audit` twice
    (each run constructs a fresh cobra command via the local helper
    so the second run does not reuse the first run's `bytes.Buffer`),
    and assert `require.Equal(t, first, second)` on stdout.
- Layers touched: **tests + fixtures only**. No domain change. No
  infrastructure change. No application change. No presentation
  change. No wiring change. The story is a pure ratification of
  the existing sort + renderer contract.
- Tiers active: **6 only**. Tiers 1, 2, 3, 4, 5 are explicitly `N/A`.
  - No `internal/domain/**` modification → Tier 1 absent.
  - No `internal/infrastructure/**` modification → Tier 2 absent.
  - No `internal/application/**` modification → Tier 3 absent.
  - No new cobra command, no formatter change → Tier 4 absent.
  - No `internal/cli/{wire,root,execute}.go`, no
    `cmd/jitctx/main.go`, no `internal/config/**` change → Tier 5
    absent.
- Guidelines loaded:
  - `.claude/guidelines/integration-test-layer-guidelines.yml`
- Estimated file count: **5 new** (1 integration test + 4 fixture
  files in 1 fixture tree) and **0 modified**. The plan file
  itself is not counted in §1.

> **Discovery finding (load-bearing).** The determinism contract that
> AC requires is **already implemented and proven** by tests in main:
>
> 1. **Sort code** lives in
>    `internal/application/usecase/audituc/usecase.go` at lines
>    217-219 and 258-278 — the `lessViolation` comparator orders by
>    `(ModuleID, FilePath, Line, RuleID)`. The comparator includes
>    `ModuleID` as the FIRST key (which is a SUPERSET of the AC's
>    "file path, line, rule ID" — a single module's violations
>    naturally fall under the same `ModuleID`, then within a module
>    the AC's three keys apply verbatim). The `<unmoduled>` synthetic
>    module is forced last. `sort.SliceStable` is used so input
>    order acts as a final tiebreaker.
> 2. **Renderer purity** is asserted by
>    `TestWriteAuditReport_Determinism` in
>    `internal/cli/format/audit_test.go:242` (formatter layer).
> 3. **End-to-end determinism** is asserted by
>    `TestAuditCmd_Integration_Determinism` in
>    `internal/cli/command/auditIntegration_test.go:127`, which
>    runs `jitctx audit` twice on the `auditViolations` fixture and
>    asserts `require.Equal(t, first, second)` on stdout.
> 4. **Sort-key invariant** is also exercised by every other
>    integration test that asserts on a golden `report.md` —
>    `auditClean`, `auditViolations`, `pc01us005..pc01us012`. Every
>    such assertion would fail if the comparator drifted.
>
> Consequently, this plan adds **no production code**. The new
> integration test is a PC01-flagged cross-feature ratification:
> it lives next to the other `pc01usNNN*Integration_test.go`
> files, exercises the sort + renderer through the audit cobra
> command, and uses a dedicated PC01 fixture
> (`testdata/pc01us013DeterministicOutput/projectFixed/`) whose
> rules deliberately produce **multiple violations spanning
> multiple files and multiple rule IDs** so the comparator's
> three secondary keys are exercised in the same run. The
> existing `TestAuditCmd_Integration_Determinism` test is kept
> intact — PC01US-013's value is the explicit, US-named,
> PC01-fixture-driven evidence that completes the traceability
> matrix entry for PC01RNF-003.

---

## Section 1 — File Set

| # | File                                                                                                                                       | Action  | Layer  | Tier | Group  | Requirements |
|---|---------------------------------------------------------------------------------------------------------------------------------------------|---------|--------|------|--------|--------------|
| 1 | `internal/cli/command/deterministicViolationOutputIntegration_test.go`                                                                      | create  | tests  | 6    | T6-G1  | PC01US-013, PC01RNF-003, PC01RNF-006 |
| 2 | `testdata/pc01us013DeterministicOutput/projectFixed/pom.xml`                                                                                | create  | tests  | 6    | T6-G2  | PC01RNF-006 |
| 3 | `testdata/pc01us013DeterministicOutput/projectFixed/project-state.yaml`                                                                     | create  | tests  | 6    | T6-G2  | PC01RNF-006 |
| 4 | `testdata/pc01us013DeterministicOutput/projectFixed/.jitctx/profiles/spring-boot-hexagonal.yaml`                                            | create  | tests  | 6    | T6-G2  | PC01US-013 |
| 5 | `testdata/pc01us013DeterministicOutput/projectFixed/src/main/java/com/acme/application/usecase/PlaceOrder.java`                             | create  | tests  | 6    | T6-G2  | PC01US-013, PC01RNF-006 |

Coverage notes:

- The fixture is intentionally **multi-violation** (more than one
  violation in the same module, spanning more than one rule ID, on
  more than one line of the same file when possible) so the
  comparator's `(ModuleID, FilePath, Line, RuleID)` secondary keys
  are exercised in a SINGLE run. A trivial single-violation fixture
  would not differentiate determinism from luck — the same single
  violation comes out the same every time regardless of sort
  correctness. To force the comparator to do real work, the profile
  declares TWO `required_annotations` rules with different IDs that
  both fire on the same `PlaceOrder.java` file:
  - `id: A-pc01us013-required-service` (alphabetically earlier)
  - `id: B-pc01us013-required-component` (alphabetically later)
  Both rules fire with `missing=[Service]` /
  `missing=[Component]` on the same `PlaceOrder.java` line. Their
  natural emission order from `evalRequiredAnnotations` is
  rule-list order (which is YAML declaration order), but the
  use-case sort overrides that with `RuleID` ascending — so the
  "A-..." rule's violation MUST appear before the "B-..." rule's
  violation in stdout. AC's "ascending rule ID" tie-breaker is
  thus exercised end-to-end.
- The integration test (file #1) loads the fixture via
  `copyFixture` from `helpers_test.go`, runs `audit` twice via two
  freshly-constructed cobra commands (each with its own
  `bytes.Buffer`), and asserts the two stdout strings are
  byte-identical. It also asserts (a) BOTH violations are present
  (so the determinism is over a non-empty multi-violation slice),
  and (b) the `[A-pc01us013-required-service]` rule-ID line
  appears BEFORE the `[B-pc01us013-required-component]` rule-ID
  line in stdout (so the ascending-RuleID secondary key is
  exercised, not just byte-equality).
- testdata is gitignored (project convention); the integration-test
  author force-adds with `git add -f` when committing.
- Every requirement ID in scope (PC01US-013, PC01RNF-003,
  PC01RNF-006) appears in at least one §1 row.

Requirement coverage trace (every ID in scope appears below):

| Requirement   | Where it lives in code (already shipped)                                                                                                             | Where this plan re-asserts it       |
|---------------|--------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------|
| PC01US-013    | `lessViolation` in `audituc/usecase.go:258-278` (sort comparator) + pure renderer in `cli/format/audit.go`                                            | T6-G1 cross-feature determinism test |
| PC01RNF-003   | `sort.SliceStable(violations, lessViolation)` at `audituc/usecase.go:217-219`; `WriteAuditReport` purity proven by `TestWriteAuditReport_Determinism` | T6-G1 two-run byte-identical assert  |
| PC01RNF-006   | real Tree-sitter parse via existing `treesitter.Parser` adapter                                                                                       | T6-G2 real `.java` fixture           |

---

## Section 2 — Frozen Domain Contract

This contract is **already in main**. It is reproduced here so any
future change in this area must keep these symbols, comparator keys,
and substitution-token spellings intact (or open a follow-up RFC).
**No new contract is introduced by PC01US-013.**

### 2.1 Violation sort comparator (frozen)

```go
// internal/application/usecase/audituc/usecase.go (existing — DO NOT MODIFY)

// lessViolation defines the deterministic sort order (RNF-003):
// (ModuleID, FilePath, Line, RuleID).
func lessViolation(a, b auditvo.AuditViolation) bool {
    if a.ModuleID != b.ModuleID {
        // "<unmoduled>" must sort last.
        if a.ModuleID == "<unmoduled>" {
            return false
        }
        if b.ModuleID == "<unmoduled>" {
            return true
        }
        return a.ModuleID < b.ModuleID
    }
    if a.FilePath != b.FilePath {
        return a.FilePath < b.FilePath
    }
    if a.Line != b.Line {
        return a.Line < b.Line
    }
    return a.RuleID < b.RuleID
}
```

The comparator is invoked through `sort.SliceStable(violations,
lessViolation)` at `audituc/usecase.go:217-219`. `SliceStable` is
load-bearing: it makes input order the final tiebreaker when two
violations agree on all four keys (which can happen when one rule
emits multiple violations on the same `(file, line)` pair, e.g. an
`expected_values` mismatch followed by a `non_empty_value_annotations`
violation under the same `RuleID`).

**Relationship to the AC.** The AC text in PC01RNF-003 states the
sort key as "ascending file path, ascending line, ascending rule
ID". The comparator above includes a PRIMARY `ModuleID` key as a
super-key — but within a single module, the three secondary keys
match the AC verbatim. The `ModuleID` super-key is required by
EP03US-002's "## Module: <id>" sub-headings (the renderer expects
violations to arrive grouped by module so a single linear pass
emits one heading per module — see `cli/format/audit.go:46-57`).
Therefore the comparator is a STRICT REFINEMENT of the AC's stated
key, not a contradiction: every pair of violations that the AC
considers ordered is also ordered the same way by the comparator,
plus the comparator additionally orders cross-module pairs.

### 2.2 `AuditViolation` (frozen)

```go
// internal/domain/vo/audit/violation.go (existing — DO NOT MODIFY)
type AuditViolation struct {
    RuleID     string
    Kind       model.AuditRuleKind
    Severity   model.AuditSeverity
    ModuleID   string
    FilePath   string // forward-slash, project-relative
    Line       int    // 1-based; 0 when the violation has no specific line
    Message    string // rule.Description with placeholders substituted
    Suggestion string // rule.Suggestion with placeholders substituted; "" when none
}
```

All four sort keys (`ModuleID`, `FilePath`, `Line`, `RuleID`) are
public fields of this VO. No field is added by PC01US-013.

### 2.3 Renderer purity (frozen)

```go
// internal/cli/format/audit.go (existing — DO NOT MODIFY)

// WriteAuditReport renders the AuditProjectOutput as deterministic markdown.
// Only one output format is supported — markdown — per EP03RF-007.
// The renderer is pure: same input => byte-identical output (RNF-003).
func WriteAuditReport(w io.Writer, out auditvo.AuditProjectOutput) error { ... }
```

The renderer iterates `out.Sintatic` in slice order (which is the
sorted order produced by the use case), groups by `ModuleID` via a
`lastModuleID` cursor (`cli/format/audit.go:46-57`), and emits a
violation block per element. No map iteration, no time-of-day, no
hostname, no PID — every byte of output is a deterministic
function of the input VO.

### 2.4 No changes to other contracts

- `model.JavaDeclaration`, `model.AuditRule`, `auditvo.AuditViolation`,
  `auditvo.AuditProjectOutput`, `auditvo.AuditModuleReport` — all
  unchanged.
- `internal/cli/wire.go` `Deps` struct — unchanged. The audit use
  case already injects `service.AuditEvaluator{}`.
- No new error sentinels, no new typed errors.
- The bundled `spring-boot-hexagonal` profile is **NOT** modified by
  this story (same posture as PC01US-002/004/005/006/007/008/009/010/
  011/012). Profile content evolves under EP-04. The PC01US-013
  fixture profile is self-contained under `testdata/`.
- `internal/infrastructure/fsprofile/mapper.go` `knownAuditRuleKinds`
  — unchanged. `AuditKindRequiredAnnotations` is already whitelisted.

---

## Section 3 — Domain Layer Plan

**N/A.** No domain types, ports, use cases, services, or errors are
introduced or modified by PC01US-013. The capability already lives
at `internal/application/usecase/audituc/usecase.go:217-219` (the
sort call) and `:258-278` (the comparator), and was last touched
during EP03 implementation.

The engine-neutrality posture (PC01RNF-001) is unchanged by this
story — the new test file does not introduce framework identifiers
into `internal/domain` or `internal/application`. The fixture
profile uses the existing neutral schema words
(`required_annotations`, `path_scope`, `annotations`, `severity`,
`description`, `suggestion`).

---

## Section 4 — Infrastructure Layer Plan

**N/A.** No infrastructure adapter is added or modified. The profile
loader already accepts the `required_annotations` kind (whitelisted
in `mapper.go` since PC01US-002); the `auditRuleDTO` already passes
`params: map[string]string` through verbatim (so `path_scope` and
`annotations` ride through unchanged). The Tree-sitter parser
already populates `JavaDeclaration.Annotations` for every annotation
on a class declaration (proven by the EP-03 + PC01US-002..012 unit
and integration tests in main).

---

## Section 5 — Application Layer Plan

**N/A.** `appaudituc.Impl.Execute` already calls `sort.SliceStable`
on the violation slice via `lessViolation` before assembling the
`AuditProjectOutput`. No edit required. The use case is the
authoritative sort site; the renderer is purely a consumer of the
sorted slice.

---

## Section 6 — Presentation Layer Plan

**N/A.** No new cobra command, no formatter change. The `audit`
command already prints violations via the existing renderer
(`internal/cli/format/audit.go`). The stdout/stderr contract is
unchanged: violations render under `## Sintatic Violations`; the
rule ID is emitted as the literal `[<rule-id>]` token in each
violation line; the integration test asserts on byte-equality of
the entire stdout across two runs.

---

## Section 7 — Composition Root + Tests Plan

### 7.1 Composition root

**N/A.** `internal/cli/wire.go`, `root.go`, `execute.go`,
`cmd/jitctx/main.go`, and `internal/config/**` are all unchanged.
The `Deps` struct is unchanged.

### 7.2 Unit tests

**N/A — coverage already exists.** The existing
`TestWriteAuditReport_Determinism` in
`internal/cli/format/audit_test.go:242` covers renderer purity at
the format-layer boundary. The existing
`TestAuditCmd_Integration_Determinism` in
`internal/cli/command/auditIntegration_test.go:127` covers
end-to-end determinism on the `auditViolations` fixture. Both
tests stayed green through PC01US-002..012 (10 PC01RNF-003 tests
total per the PC01US-012 QA report). Adding a unit test on
`lessViolation` would test private package-internal symbols, which
the project convention discourages — the comparator is exercised
in situ by every integration test that asserts on a golden
`report.md`.

### 7.3 Parser unit tests

**N/A.** No parser change. The existing parser unit tests in
`internal/infrastructure/treesitter/parser_test.go` already prove
`Annotations` and `Line` are populated for every class
declaration. The PC01US-013 fixture exercises the same parser code
path (a class with zero annotations under
`com.acme.application.usecase`).

### 7.4 Integration tests (T6-G1 — `internal/cli/command/deterministicViolationOutputIntegration_test.go`)

ONE test function:

- `t.Parallel()`.
- Builds a real `audit` cobra command via a local helper modelled
  on `newAuditCmdForIntegrationTestBaseRequiredAnnotations`
  (Q-DRY: a local copy is acceptable per the no-upstream-refactor
  rule established by PC01US-007 / PC01US-008 / PC01US-009 /
  PC01US-010 / PC01US-012). The helper wires:
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

The shared local helper is `newAuditCmdForDeterministicViolationOutput`
(camelCase per project filename convention).

#### Test — `TestAuditCmd_Integration_PC01US013_DeterministicViolationOutput_TwoRunsByteIdentical` (AC)

Steps:

1. `workDir := t.TempDir()`.
2. `copyFixture(t, fixtureDir(t, "pc01us013DeterministicOutput", "projectFixed"), workDir)`.
3. `manifestPath := filepath.Join(workDir, "project-state.yaml")`.
4. **First run:** construct a fresh cobra command via
   `newAuditCmdForDeterministicViolationOutput(t, workDir, manifestPath)`,
   invoke `run("--dir", workDir, "--manifest", manifestPath)`,
   assert `err == nil`, capture `first := stdout1.String()`.
5. **Second run:** construct ANOTHER fresh cobra command (separate
   `bytes.Buffer`, separate command instance) via the same helper
   on the same `workDir` + `manifestPath`, invoke `run` with the
   same args, assert `err == nil`, capture
   `second := stdout2.String()`.
6. **Assertion 1 — byte-equality (the AC):**
   `require.Equal(t, first, second, "audit output must be
   byte-identical across consecutive runs (PC01RNF-003)")`.
7. **Assertion 2 — non-empty multi-violation evidence:** assert
   `first` contains BOTH
   `[A-pc01us013-required-service]` and
   `[B-pc01us013-required-component]` rule-ID tokens. This
   guarantees the determinism assertion is over a multi-violation
   slice — a vacuous one-or-zero-violation slice would pass the
   byte-equality check trivially.
8. **Assertion 3 — sort-key invariant evidence:** assert the
   index of `[A-pc01us013-required-service]` in `first` is LESS
   THAN the index of `[B-pc01us013-required-component]`. This
   exercises the comparator's secondary `RuleID` key — when two
   violations agree on `(ModuleID, FilePath, Line)`, the
   ascending-RuleID rule wins. Without the sort, the YAML
   declaration order in the profile (which is "A then B")
   coincidentally matches the expected output, so the assertion
   would still pass. To make the assertion load-bearing, the
   profile YAML deliberately declares the rules in REVERSE
   alphabetical order (`B-...` listed FIRST, `A-...` listed
   SECOND) — see §7.5. This means the natural emission order
   from `evalRequiredAnnotations` is `B-..., A-...`, but the
   use-case sort overrides it to `A-..., B-...`. The assertion
   `indexA < indexB` therefore catches any regression that
   removes or breaks the comparator.

Maps to the AC ("Two consecutive runs produce identical output").

### 7.5 Fixtures (T6-G2 — `testdata/pc01us013DeterministicOutput/projectFixed/`)

Naming convention follows PC01US-007..012: lower-camelCase project
root segments matching `pc01us013DeterministicOutput`. testdata is
gitignored (project convention); the integration-test author
force-adds with `git add -f` when committing.

Fixture tree:

```
projectFixed/
├── pom.xml                         # contains org.springframework.boot for module detection
├── project-state.yaml              # schema_version: 2; one module; one file
├── .jitctx/
│   └── profiles/
│       └── spring-boot-hexagonal.yaml   # TWO required_annotations rules (B then A in YAML order)
└── src/
    └── main/
        └── java/
            └── com/acme/application/usecase/
                └── PlaceOrder.java
```

#### `pom.xml`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.acme</groupId>
  <artifactId>pc01us013</artifactId>
  <version>0.0.1-SNAPSHOT</version>
  <parent>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-parent</artifactId>
    <version>3.2.0</version>
  </parent>
</project>
```

#### `project-state.yaml`

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
      - name: PlaceOrder
        types:
          - service
        path: src/main/java/com/acme/application/usecase/PlaceOrder.java
        methods: []
    dependencies: []
contexts: []
```

This skeleton matches PC01US-012's fixture verbatim (single module
under `com.acme.application.usecase`, single class `PlaceOrder`).

#### `.jitctx/profiles/spring-boot-hexagonal.yaml`

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
    - src/main/java/**

rules:
  - match:
      node_type: class_declaration
      has_annotation: Service
    classify_as: service

audit_rules:
  # PC01US-013 — TWO rules that BOTH fire on PlaceOrder.java to
  # exercise the comparator's secondary RuleID key. The YAML
  # declaration order is REVERSE alphabetical (B then A) so the
  # natural emission order from evalRequiredAnnotations is
  # B-..., A-... — but the use-case sort comparator overrides
  # this to A-..., B-... by ascending RuleID. The integration
  # test asserts indexA < indexB in stdout, catching any
  # regression that removes the sort.
  - id: B-pc01us013-required-component
    kind: required_annotations
    severity: ERROR
    description: '{name}: {evidence}'
    suggestion: 'Annotate {name} with @Component'
    params:
      path_scope: src/main/java/com/acme/application/usecase/
      annotations: 'Component'

  - id: A-pc01us013-required-service
    kind: required_annotations
    severity: ERROR
    description: '{name}: {evidence}'
    suggestion: 'Annotate {name} with @Service'
    params:
      path_scope: src/main/java/com/acme/application/usecase/
      annotations: 'Service'
```

Notes on the audit rules:

- BOTH rules use `kind: required_annotations`, `path_scope:
  src/main/java/com/acme/application/usecase/`, and target the
  same `class_declaration` node type (default for the kind).
- The rule IDs are crafted so `A-...` sorts strictly BEFORE
  `B-...` lexicographically.
- The YAML declares the `B-...` rule FIRST so the natural
  declaration order is reversed relative to the expected sorted
  output. This is the load-bearing detail for §7.4 Assertion 3.
- The two rules emit violations on the SAME file
  (`PlaceOrder.java`) at the SAME line (the class declaration,
  typically line 5 for this fixture). The comparator's
  `(ModuleID, FilePath, Line)` keys are equal across the two
  violations; the `RuleID` tiebreaker is therefore the
  decision-maker. This is the precise scenario that exercises
  the AC's "ascending rule ID" tie-breaker.

#### `src/main/java/com/acme/application/usecase/PlaceOrder.java`

```java
package com.acme.application.usecase;

// PC01US-013 fixture: deliberately MISSING both @Service and
// @Component so BOTH required_annotations rules fire on the same
// file/line. The two violations share (ModuleID, FilePath, Line)
// — only their RuleID differs — exercising the comparator's
// ascending-RuleID secondary key.
public class PlaceOrder {}
```

The class is empty and unannotated. The fixture intentionally
keeps the file simple so any change to the comparator surfaces
immediately in the integration assertions, not buried under
unrelated rule outputs.

### 7.6 Engine-neutrality grep gate (PC01RNF-001)

Before declaring the story done, run the cumulative grep gate from
PC01RNF-001:

```bash
grep -rE "(Lombok|Spring|Mockito|Autowired|JPA|Testcontainers|ActiveProfiles|SpringBootTest|Primary|Qualifier)" \
    internal/domain internal/application
```

This MUST return zero matches. PC01US-013 introduces NO new
framework-specific identifiers — the test's rule IDs use the
neutral prefix `pc01us013-required-{service,component}`, the
fixture profile uses the neutral schema kind
`required_annotations`, and the integration-test assertions look
at the rule-ID tokens, not at framework strings. The substrings
`@Service` and `@Component` appear in (a) the YAML profile under
`testdata/`, and (b) the rule `description`/`suggestion`
templates inside that YAML — both of which are explicitly
exempted by PC01RNF-001's "engine source under `internal/domain`,
`internal/application`, `internal/cli`" scoping (the
integration-test file under `internal/cli/command/` carries no
framework identifier; only `Service` / `Component` would, and
those identifiers are neutral generic words used as rule
parameters).

This is the same posture PC01US-002..012 adopted. No
`internal/domain` file or `internal/application` file gains a
new framework literal.

---

## Section 8 — Open Questions & Risks

All questions were pre-resolved during discovery — none are blocking.

- **Q1 — Does PC01US-013 need a NEW dedicated PC01 fixture, or
  can it reuse an existing PC01 fixture
  (e.g. `pc01us010TxDecoratorContract/projectMissingQualifier`
  or `pc01us012LegacyHasAnnotation/projectMissingService`)?**
  Pre-resolved: **a dedicated fixture is required.** Reusing an
  existing PC01 fixture would couple the determinism evidence to
  another story's fixture content — if PC01US-010 or PC01US-012
  is later refactored (e.g. its rule IDs renamed, its violations
  shuffled), the determinism test would silently change behaviour
  and the AC's "fixed Java fixture and fixed profile" wording
  would no longer hold. A dedicated fixture under
  `testdata/pc01us013DeterministicOutput/projectFixed/` keeps the
  story self-contained. Additionally, the existing PC01 fixtures
  emit only ONE violation each — they don't exercise the
  comparator's secondary `RuleID` key. The dedicated fixture
  declares two rules whose ID tie-breaker is observable.
  Blocking: No.

- **Q2 — Does the existing
  `TestAuditCmd_Integration_Determinism` already cover the AC?**
  Pre-resolved: **functionally yes; nominally no.** That test
  exists since EP-03 and asserts byte-equality on stdout across
  two runs of the `auditViolations` fixture. It is the technical
  ground truth for PC01RNF-003. However:
  - It is NOT named for PC01US-013, so the traceability matrix
    entry for PC01US-013 has no associated test by name.
  - It uses the EP-03 `auditViolations` fixture, not a PC01-flavoured
    fixture. Per project convention (the
    `pc01usNNN<FeatureCamel>/projectXxx` testdata layout adopted
    since PC01US-002), PC01 stories provide their own fixtures
    so the cross-feature evidence is visible alongside the
    other PC01 stories.
  - It does NOT exercise the comparator's secondary `RuleID` key —
    `auditViolations` has multiple rules, but each fires on a
    different `(file, line)` pair, so the tiebreaker chain
    short-circuits at `FilePath` or `Line`.

  PC01US-013 therefore adds ONE new integration test that (a)
  uses the project's PC01 testdata convention, (b) is named with
  the `PC01US013` infix so the traceability matrix has a clean
  entry, and (c) exercises the `RuleID` tiebreaker explicitly.
  The existing `TestAuditCmd_Integration_Determinism` is kept
  intact — both tests run in CI, providing two independent
  determinism assertions. Blocking: No.

- **Q3 — Is the comparator's
  `(ModuleID, FilePath, Line, RuleID)` key compatible with the
  AC's stated key `(file path, line, rule ID)`?** Pre-resolved:
  **yes, the comparator is a strict refinement.** Within a
  single module, the three secondary keys match the AC verbatim
  in the same priority order. Across modules, the AC is silent
  and the comparator picks `ModuleID` as the primary key (which
  is the natural grouping for the `## Module: <id>` rendered
  sub-headings). Documented in §2.1. Blocking: No.

- **Q4 — Does the integration test need to construct TWO
  separate cobra commands, or can one command's `Execute` be
  called twice?** Pre-resolved: **two separate commands.** The
  cobra command's internal state (flag parsing, `RunE` hook
  state) is not designed to be re-executed. The existing
  `TestAuditCmd_Integration_Determinism` constructs two separate
  commands via two separate `newAuditCmdFor(...)` calls; the
  PC01US-013 test follows the same pattern. This also more
  closely models the AC scenario "I run the quality gate
  twice" — each run is an independent process invocation in
  real CI. Blocking: No.

- **Q5 — Why an empty `PlaceOrder` class instead of one with
  some content?** Pre-resolved: a minimal class minimises the
  surface area for accidental rule firing from other rules
  (e.g. if a future PC01US story adds a rule that fires on
  empty classes, the test would gain a third violation that
  could mask determinism regressions). The two
  `required_annotations` rules in the fixture profile are the
  ONLY rules; they fire because the class is missing both
  required annotations; that's the entire scope of the
  fixture's violation surface. Blocking: No.

- **Q6 — Should the test also assert on file-path tiebreaker
  ordering (multiple files with violations)?** Pre-resolved:
  **out of scope for PC01US-013's AC.** The AC is "two
  consecutive runs produce identical output" — byte-equality is
  the primary assertion. Adding a multi-file fixture to exercise
  `FilePath` ordering would balloon the fixture's complexity
  without strengthening the AC. The `FilePath` tiebreaker is
  already exercised by the existing `auditViolations` golden
  test (`TestAuditCmd_Integration_ViolationsGoldenMatch`), which
  has five violations across five different files in a single
  module. PC01US-013 limits its load-bearing assertion to the
  `RuleID` tiebreaker because that one is NOT exercised
  elsewhere with a deliberately-reversed YAML declaration order.
  Blocking: No.

- **Q7 — `PlaceOrder.java` line number for the class
  declaration.** Pre-resolved: the parser reports `Line` based
  on Tree-sitter's row + 1. For an empty fixture file with the
  package declaration on line 1 and a comment block, the class
  declaration lands on line 5 (or whatever the Tree-sitter
  parser reports). The test does NOT assert on the literal
  line number — it only asserts on the substring presence and
  the rule-ID ordering. So the exact line number is irrelevant
  to determinism; it just needs to be EQUAL across the two
  runs, which `lessViolation` guarantees because both rules
  see the same `summary` for the same parsed file. Blocking:
  No.

- **Q8 — Profile YAML fixture content depth.** Pre-resolved:
  the profile YAML carries the FULL canonical
  `spring-boot-hexagonal` shape (name, languages, query_lang,
  detect, module_detection, rules, audit_rules). Audit_rules-
  only YAML would not be detected by the auto-detector. This
  is the same fixture-content requirement PC01US-007..012
  documented. The fixture authors copy the canonical shape
  from `testdata/pc01us012LegacyHasAnnotation/projectMissingService/.jitctx/profiles/spring-boot-hexagonal.yaml`,
  replacing the `audit_rules` block with PC01US-013's two
  rules. Blocking: No.

- **Q9 — Audit cmd exit code on ERROR-severity violations.**
  Pre-resolved: the audit command does NOT exit non-zero on
  ERROR-severity violations (matching all prior audit
  integration tests' `require.NoError` posture and confirmed
  by the PC01US-012 plan's same finding). The integration
  test uses `require.NoError(t, run(...))` for both runs.
  Blocking: No.

- **Risk R1 — Spurious violation from the classification rule
  `has_annotation: Service` → `classify_as: service`.** The
  classification rule (in the `rules:` section) does not emit
  audit violations — it only tags the class for
  module-detection / contract classification. Audit violations
  come exclusively from the `audit_rules:` section. Mitigation:
  none required — the test doesn't assert on absence of
  classification side-effects. Verified by reading
  `evalRequiredAnnotations` and `EvaluateFile`.

- **Risk R2 — Test flakiness from filesystem walk order.** The
  `treesitter.NewWalker()` already sorts walked files by name
  (proven by `walker.go:65 sort.Strings(files)`), so the parse
  order is deterministic. The fixture has only one `.java`
  file under `src/main/java/`, so this risk is moot for
  PC01US-013 specifically — but the determinism contract
  scales to multi-file projects because of the walker's
  sort. Mitigation: none required.

- **Risk R3 — Map iteration in the YAML loader / mapper.** The
  audit-rules loader returns rules in slice order (not map
  order) per `auditLoader.go` and `mapper.go` in main. The
  evaluator iterates the `rules` slice in declaration order.
  The use-case sort overrides any non-determinism that could
  leak through. Mitigation: none required — the
  `sort.SliceStable` call at `audituc/usecase.go:217` is the
  single source of truth and is itself fully deterministic.

No `Blocking: Yes` entries. Discovery proceeds to implementation.

---

## Section 9 — Parallel Execution Plan (authoritative for `@agent-manager`)

```yaml
tiers:
  - id: 6
    name: Tests + fixtures (parallel) — tests-only ratification of existing audit-pipeline determinism
    depends_on: []
    groups:
      - id: T6-G1
        scope:
          create:
            - internal/cli/command/deterministicViolationOutputIntegration_test.go
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: M
        notes: >
          ONE test function — TestAuditCmd_Integration_PC01US013_DeterministicViolationOutput_TwoRunsByteIdentical —
          with t.Parallel(). Local helper newAuditCmdForDeterministicViolationOutput
          modelled on the PC01US-009 / PC01US-010 / PC01US-012 helpers (no
          upstream DRY refactor). Two freshly-constructed cobra commands invoke
          `audit` against the same temp workdir; the test asserts (1)
          require.Equal(first, second) on stdout for byte-equality (the AC),
          (2) BOTH rule-ID tokens [A-pc01us013-required-service] and
          [B-pc01us013-required-component] are present in stdout (proving the
          determinism is over a multi-violation slice), and (3) indexA < indexB
          in stdout (proving the comparator's secondary RuleID key fired,
          since the YAML profile declares the rules in REVERSE alphabetical
          order). Engine-neutrality grep gate (no Spring / Lombok / Mockito /
          Autowired / JPA / Testcontainers / ActiveProfiles / SpringBootTest /
          Primary / Qualifier in internal/domain, internal/application) MUST
          pass before this group is declared done — see §7.6. The existing
          TestAuditCmd_Integration_Determinism remains intact.

      - id: T6-G2
        scope:
          create:
            - testdata/pc01us013DeterministicOutput/projectFixed/pom.xml
            - testdata/pc01us013DeterministicOutput/projectFixed/project-state.yaml
            - testdata/pc01us013DeterministicOutput/projectFixed/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us013DeterministicOutput/projectFixed/src/main/java/com/acme/application/usecase/PlaceOrder.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Single fixture tree projectFixed. PlaceOrder.java is a deliberately
          empty unannotated class under com.acme.application.usecase. The
          profile YAML is the canonical FULL spring-boot-hexagonal shape
          (name, languages, query_lang, detect, module_detection, classification
          rules) PLUS audit_rules with TWO required_annotations rules:
          B-pc01us013-required-component (declared FIRST in YAML, requires
          @Component) and A-pc01us013-required-service (declared SECOND in
          YAML, requires @Service). Both rules use path_scope
          src/main/java/com/acme/application/usecase/ and target the same
          file. The YAML declaration order is REVERSE alphabetical so the
          comparator's ascending-RuleID tiebreaker is observable in the test's
          indexA < indexB assertion. testdata is gitignored — author force-
          adds when committing.
```

---

## Self-Validation Checklist

**File-set coverage**
- Every file in §1 appears exactly once across §9 groups
  (cross-checked: T6-G1 has 1, T6-G2 has 4 — total 5,
  matching §1's 5 rows).
- Every requirement ID (PC01US-013, PC01RNF-003, PC01RNF-006)
  appears in at least one §1 row.
- No file path appears in two groups.

**Frozen contract**
- No new ports, model types, use-case interfaces, or error sentinels
  are introduced. The frozen contract in §2 documents the EXISTING
  surface (`lessViolation` comparator, `AuditViolation` VO,
  `WriteAuditReport` purity). Every symbol referenced lives in main.
- `Deps` struct in `internal/cli/wire.go` is unchanged — explicitly
  noted in §2.4 and §7.1.
- No fields marked `TODO` or `{placeholder}` in the frozen contract.

**DAG**
- `depends_on` edges: T6-G1 → ∅, T6-G2 → ∅. Acyclic.
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
  `integration-test-layer-guidelines.yml`).

**Open questions**
- Zero `Blocking: Yes` entries. Discovery is unblocked.
