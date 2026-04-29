# Plan — PC01US-003 Require full JPA + Lombok stack on entities

## Section 0 — Summary

- Feature: a profile author can declare a single `required_annotations` audit rule (`id: jpa-entity-contract`) that flags any class under the `infrastructure.persistence.entity` package that does not declare ALL of `@Entity`, `@Table`, `@Getter`, `@Setter`, `@NoArgsConstructor`, and `@AllArgsConstructor`, with `missing=[...]` evidence on the violation message.
- Requirement IDs covered: **PC01US-003** (story); transitively **PC01RF-001** (multi-annotation all-of evaluator), **PC01RF-009** (evidence-rich messages), **PC01RNF-001** (engine language-neutrality), **PC01RNF-003** (deterministic output), **PC01RNF-006** (real-parser integration tests).
- Layers touched: **testdata fixtures + integration test only**. No Go source under `internal/domain`, `internal/application`, `internal/infrastructure`, or `internal/cli` is created or modified.
- Tiers active: **6** (Tier 6 only). Tiers 1-5 collapse because PC01RF-001 is already in production (see Section 8).
- Guidelines loaded: `.claude/guidelines/integration-test-layer-guidelines.yml`.
- Estimated file count: **9 new** (2 fixture trees × 4 files + 1 integration-test Go file), **0 modified**.

### Verified working-tree facts (2026-04-29)

1. `model.AuditKindRequiredAnnotations = "required_annotations"` sentinel exists at `/workspaces/jitctx/internal/domain/model/auditRule.go:21` and is dispatched at `/workspaces/jitctx/internal/domain/service/auditRuleEvaluator.go:43-44`.
2. `evalRequiredAnnotations` at `/workspaces/jitctx/internal/domain/service/auditRuleEvaluator.go:287-325` substitutes `{missing}` with `"[" + strings.Join(missing, ",") + "]"`, producing the exact `missing=[Setter]` shape Scenario 2 asserts. Order is preserved by `missingAnnotations` (lines 360-375) — it walks `required` in declaration order and emits absent entries.
3. `path_scope` is a `strings.Contains` substring filter (line 298): `/infrastructure/persistence/entity/` will match any Java file under that path segment.
4. Default `node_types` is `class_declaration` when omitted (lines 302-305) — interfaces are silently skipped, which matches the AC ("entity classes").
5. `auditRuleDTO` at `/workspaces/jitctx/internal/infrastructure/fsprofile/dto.go:39-46` decodes `params: map[string]string`, and `mapper.go:17` whitelists `AuditKindRequiredAnnotations` in `knownAuditRuleKinds` — round-trip from YAML works today (PC01US-002 exercises the same path).
6. `treesitter/parser.go` `extractAnnotations` produces simple-name annotations (`["Entity", "Table", "Getter", "Setter", "NoArgsConstructor", "AllArgsConstructor"]`) for the JPA fixture; `auditClean/Order.java` already proves multi-annotation extraction.
7. The audit pipeline already runs end-to-end: `appaudituc.New(...)` in `/workspaces/jitctx/internal/cli/wire.go` is consumed verbatim by the PC01US-002 integration test (`usecaseImplStereotypeIntegration_test.go:47-61`); the new test reuses the identical helper shape.
8. Forbidden-token gate (`/workspaces/jitctx/internal/qualitygate/exemptions.go`) lists `@Entity`, `JUnit`, `Mockito`, `javax`, `spring` (case-sensitive). The companion test at `/workspaces/jitctx/internal/qualitygate/javaReferencesGate_test.go:173-174` skips ALL `_test.go` files outright (plan §8 Q1 exemption from EP04US-009), and `testdata/` segments are filtered out at line 44. So neither the new `jpaEntityContractIntegration_test.go` file nor the new `testdata/pc01us003JpaEntityContract/...` tree is subject to the gate, even when they contain the literals `@Entity`, `@Table`, `@Setter`, etc.
9. Per `CLAUDE.md` "Nomes de arquivos Go", new test file uses camelCase + the toolchain-mandated `_test.go` suffix: `jpaEntityContractIntegration_test.go`.

## Section 1 — File Set

| # | File | Action | Layer | Tier | Group |
|---|------|--------|-------|------|-------|
| 1 | `testdata/pc01us003JpaEntityContract/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml` | create | testdata | 6 | T6-G1 |
| 2 | `testdata/pc01us003JpaEntityContract/projectClean/pom.xml` | create | testdata | 6 | T6-G1 |
| 3 | `testdata/pc01us003JpaEntityContract/projectClean/project-state.yaml` | create | testdata | 6 | T6-G1 |
| 4 | `testdata/pc01us003JpaEntityContract/projectClean/src/main/java/com/acme/infrastructure/persistence/entity/OrderEntity.java` | create | testdata | 6 | T6-G1 |
| 5 | `testdata/pc01us003JpaEntityContract/projectMissing/.jitctx/profiles/spring-boot-hexagonal.yaml` | create | testdata | 6 | T6-G2 |
| 6 | `testdata/pc01us003JpaEntityContract/projectMissing/pom.xml` | create | testdata | 6 | T6-G2 |
| 7 | `testdata/pc01us003JpaEntityContract/projectMissing/project-state.yaml` | create | testdata | 6 | T6-G2 |
| 8 | `testdata/pc01us003JpaEntityContract/projectMissing/src/main/java/com/acme/infrastructure/persistence/entity/OrderEntity.java` | create | testdata | 6 | T6-G2 |
| 9 | `internal/cli/command/jpaEntityContractIntegration_test.go` | create | tests | 6 | T6-G3 |

## Section 2 — Frozen Domain Contract

**No new Go signatures.** The new integration test consumes only types and constructors that are already on `main`:

- `model.AuditRuleKind`, `model.AuditKindRequiredAnnotations`, `AuditSeverity*`, `model.AuditRule{ID, Kind, Severity, Description, Suggestion, Params}` (`internal/domain/model/auditRule.go`).
- `auditvo.AuditViolation{RuleID, Kind, Severity, ModuleID, FilePath, Line, Message, Suggestion}` (`internal/domain/vo/audit/auditViolation.go`).
- `audituc.UseCase.Execute(ctx, AuditProjectInput) (AuditProjectOutput, error)` (`internal/domain/usecase/audituc/usecase.go`).
- `appaudituc.New(...)` constructor (`internal/application/usecase/audituc/usecase.go`) — the parameter list is locked by `auditIntegration_test.go:47-61` and `usecaseImplStereotypeIntegration_test.go:47-61`.
- `command.NewAuditCmd(uc audituc.UseCase, _ *slog.Logger) *cobra.Command` (`internal/cli/command/auditCmd.go`).
- Shared test helpers `copyFixture(t, src, dst)` and `fixtureDir(t, parts ...string) string` from `internal/cli/command/helpers_test.go`.

`Deps` struct in `internal/cli/wire.go` is unchanged (`Deps.Audit audituc.UseCase` is already wired and exercised by `auditCmd`). No new sentinel error.

## Section 3 — Domain Layer Plan

N/A. `evalRequiredAnnotations` already implements all-of semantics with `missing=[...]` evidence. The same code path is locked by `auditRuleEvaluator_test.go` (PC01US-001 unit-test coverage) and exercised end-to-end by the PC01US-002 integration test that already lives on `main`. No domain-layer change is required for this story.

## Section 4 — Infrastructure Layer Plan

N/A. `treesitter/parser.go` `extractAnnotations` already produces simple-name annotations, so a six-annotation entity yields `["Entity", "Table", "Getter", "Setter", "NoArgsConstructor", "AllArgsConstructor"]` (or the same minus `Setter` for the missing variant). `fsprofile/auditLoader.go` already round-trips `required_annotations` rules through `auditRuleDTO`. `fsmanifest` already parses `schema_version: 2` manifests with the modules block we need.

## Section 5 — Application Layer Plan

N/A. `audituc.Impl.Execute` orchestration steps already (a) load the manifest, (b) detect/resolve the user profile, (c) walk Java sources, (d) run `service.AuditEvaluator.EvaluateFile` over every rule including `required_annotations`, and (e) emit violations sorted by file path / line / rule ID (PC01RNF-003).

## Section 6 — Presentation Layer Plan

N/A. `auditCmd.go` and `format/audit.go` are consumed unchanged. The integration test asserts via `require.Contains` on stdout (rule ID, evidence string, source file name) and via `strings.Count` on the rule-ID bracket — no golden file, no formatter change.

## Section 7 — Composition Root + Tests Plan

### Wiring

No edits to `wire.go` / `root.go` / `execute.go` / `main.go`. Audit is a complete vertical slice; the new test reuses `appaudituc.New(...)` exactly as `usecaseImplStereotypeIntegration_test.go:47-61` does (which itself mirrors `auditIntegration_test.go:27-73`).

### Unit tests

None. `auditRuleEvaluator_test.go` already locks the evaluator behaviour for the same code path (PC01US-001 coverage). A second unit test for the literal six-element set `{Entity, Table, Getter, Setter, NoArgsConstructor, AllArgsConstructor}` would be redundant — the evaluator is annotation-name-agnostic; it iterates the comma-joined `required` slice with no special-casing.

### Integration test

**File**: `internal/cli/command/jpaEntityContractIntegration_test.go`.

**Helper**: replicate the `newAuditCmdFor*` helper from `usecaseImplStereotypeIntegration_test.go:27-73` inline as `newAuditCmdForJpaEntityContract(t, workDir, manifestPath)`. Mirror the same parameter list and the same constructor call to `appaudituc.New(...)`. (Parallel-safe: each test gets its own `t.TempDir()`.) No upstream refactor — DRYing the helper across audit integration tests is a separate non-AC follow-up.

**Test functions** (each `t.Parallel()`):

1. `TestAuditCmd_Integration_JpaEntityContract_AllSixPresentNoViolation` — copies `projectClean` into `t.TempDir()`, runs `audit --dir <tmp> --manifest <tmp>/project-state.yaml`, asserts no error, stdout contains the no-violations message and does NOT contain `jpa-entity-contract`. **Backs PC01US-003 Scenario 1.**
2. `TestAuditCmd_Integration_JpaEntityContract_MissingSetterReportsEvidence` — copies `projectMissing` into `t.TempDir()`, runs the same command, asserts no error, stdout contains `[jpa-entity-contract]`, `missing=[Setter]`, and `OrderEntity.java`; asserts exactly one occurrence of `[jpa-entity-contract]` (Setter is the ONLY missing annotation, so the bracket evidence is exactly `[Setter]`). **Backs PC01US-003 Scenario 2.**
3. `TestAuditCmd_Integration_JpaEntityContract_Determinism` — runs the failing fixture twice (in two separate `t.TempDir()`s), normalises temp-dir prefix, asserts byte-identical stdout. **Backs PC01RNF-003 for the new rule wiring.**

**Why `missing=[Setter]` is byte-stable**: `evalRequiredAnnotations` builds `missing` by iterating `required` in profile-declared order (`auditRuleEvaluator.go:368-374`). The fixture profile declares annotations as `'Entity,Table,Getter,Setter,NoArgsConstructor,AllArgsConstructor'`; with all six declared on the clean fixture and exactly `Setter` removed on the missing fixture, the missing slice is a single element and renders to `[Setter]`.

### Fixture content

**Profile YAML** (the `audit_rules:` block — head comes verbatim from `testdata/pc01us002UsecaseImplStereotype/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml`, the only audit rule appended is the new one):

```yaml
audit_rules:
  - id: jpa-entity-contract
    kind: required_annotations
    severity: ERROR
    description: 'JPA entity {name} must declare all of [{required}]; missing={missing}'
    suggestion: 'Add the missing annotation(s) to {name}: {missing}'
    params:
      path_scope: /infrastructure/persistence/entity/
      annotations: 'Entity,Table,Getter,Setter,NoArgsConstructor,AllArgsConstructor'
```

**Profile head** (the `name`, `languages`, `query_lang`, `detect`, `module_detection`, classification `rules`): copied verbatim from `testdata/pc01us002UsecaseImplStereotype/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml` lines 1-81 so the loader does not reject the file (R-001 mitigation). The PC01US-002 head already includes `detect: org.springframework.boot` so the `pom.xml` auto-detect path resolves and the user profile loads.

**`pom.xml`** (both projects): identical to `testdata/pc01us002UsecaseImplStereotype/projectClean/pom.xml`. Required so `fsprofile` detects the user profile via the `org.springframework.boot` literal in the parent block (without it, `profileMatches` falls back to bundled defaults that do not carry the new rule).

**`projectClean` Java fixture** (`OrderEntity.java`):

```java
package com.acme.infrastructure.persistence.entity;

import jakarta.persistence.Entity;
import jakarta.persistence.Table;
import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

@Entity
@Table(name = "orders")
@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
public class OrderEntity {

    private Long id;
    private String description;
}
```

**`projectMissing` Java fixture** (`OrderEntity.java`): byte-identical to the clean variant **except** the `@Setter` annotation line and its corresponding `import lombok.Setter;` line are removed. All five remaining annotations (`@Entity @Table @Getter @NoArgsConstructor @AllArgsConstructor`) stay so that `missing` evaluates to exactly `[Setter]`.

**`project-state.yaml`** (both projects): minimal `schema_version: 2` manifest with one module rooted at the entity package. Suggested module shape:

```yaml
schema_version: 2
generated_at: 2026-04-29T00:00:00Z
stack:
  languages:
    - java
  frameworks:
    - spring-boot-hexagonal
modules:
  - id: infrastructure.persistence.entity
    path: src/main/java/com/acme/infrastructure/persistence/entity
    tags: []
    contracts:
      - name: OrderEntity
        types:
          - entity
        path: src/main/java/com/acme/infrastructure/persistence/entity/OrderEntity.java
        methods: []
    dependencies: []
contexts: []
```

The two fixture manifests are byte-identical (no need to vary the module ID across clean/missing — the failing fixture in PC01US-002 used `application` rather than `application.usecase` only because of an earlier authoring drift, not a contract requirement; here we keep both manifests identical to minimise diff noise).

### Why no golden report

A golden would couple the test to incidental rendering details (suggestion line, profile-name comment, ASCII tree art) outside the AC. Focused `require.Contains` assertions on rule ID, evidence string, and offending file path track the AC text exactly and survive cosmetic formatter changes.

## Section 8 — Open Questions & Risks

### PC01RF-001 status report

**PC01RF-001 IS COMPLETE on `main`.** Commit `414955c` (2026-04-29, "feat(audit): add required_annotations rule kind with all-of semantics") landed the sentinel, evaluator, fsprofile mapper entry, and 6 unit tests; commit `b8c6589` (PC01US-002) landed the first end-to-end ratification. Contract verification against PC01US-003 Gherkin is zero-mismatch:

| Scenario assertion | Implementation that satisfies it |
|---|---|
| "no violation reported" (Scenario 1) | `evalRequiredAnnotations` returns nil when `missingAnnotations` empty (`auditRuleEvaluator.go:313-314`) |
| "evidence missing=[Setter]" (Scenario 2) | `ctx["missing"] = "[" + strings.Join(missing, ",") + "]"` then substituted into Description (lines 320, plus template substitution in `makeViolation`) |
| Rule selectable by id `jpa-entity-contract` | `auditLoader.go` round-trips arbitrary `id` strings through `auditRuleDTO.ID` |
| Java file in `infrastructure.persistence.entity` package | `path_scope: /infrastructure/persistence/entity/` substring filter (`auditRuleEvaluator.go:298`) |
| Class declares all six annotations | `parser.go` `extractAnnotations` populates `Annotations` simple-names; `missingAnnotations` does exact-match set difference |

**No engine delta required.** This plan is a tests-only ratification + fixture authoring plan, mirroring the PC01US-002 pattern.

### Open questions

| # | Question | Blocking | Resolution |
|---|---|---|---|
| Q1 | Story says "on JPA entity classes" but evaluator does not consult `decl.Name` or supertype. Restrict by name pattern (`*Entity`)? | No | Fixture controls scope by placing exactly one class under `/infrastructure/persistence/entity/`. A name-pattern selector belongs to PC01RF-004 / PC01US-005 (out of scope here). |
| Q2 | Add the new rule to `profiles/spring-boot-hexagonal.yaml` and bundled-profile copies? | No | Story is "profile author can author this rule". Wider rollout would alter every existing audit golden — out of scope. Rule lives only in the two new fixture profiles. |
| Q3 | Lift `newAuditCmdFor*` from PC01US-002 + main audit integration test into a single `helpers_test.go` helper? | No | Refactor not bundled with this AC. New file copies the helper; follow-up PR can DRY across the three audit integration test files. |
| Q4 | `description: '... [{required}]; missing={missing}'` brace-bracket shape acceptable? | No | Same shape used by PC01US-001 unit-test rule and PC01US-002 integration fixture (`auditRuleEvaluator_test.go:448`). AC asserts only on `missing=[Setter]`. |
| Q5 | `@Table(name = "orders")` carries an argument — does the evaluator misclassify it? | No | `extractAnnotations` returns the simple name `Table` regardless of argument shape (verified by parser tests on annotated classes in `auditClean/`). The PC01RF-007 annotation-with-arguments capability is OUT of scope for this story. |

### Risks

| ID | Risk | Probability | Impact | Mitigation |
|---|---|---|---|---|
| R-001 | Fixture profile missing required `detect:` block, loader rejects | Low | Medium | Copy full profile head from `testdata/pc01us002UsecaseImplStereotype/projectClean/...` verbatim; append the single new audit rule. |
| R-002 | Tree-sitter extracts FQN annotations differently for `@jakarta.persistence.Entity` form, miscompares | Low | Low | Fixture uses simple `@Entity` form on the class plus `import jakarta.persistence.Entity;` — same pattern proven by PC01US-002 fixture (`@Service` + `import org.springframework.stereotype.Service;`). |
| R-003 | Future refactor renames `{missing}` token, silently breaking Scenario 2 | Medium | Medium | Test asserts literal `missing=[Setter]` substring — any rename fails this test loud. |
| R-004 | Forbidden-token gate flags new fixture for `@Entity`, `javax`, `spring` literals | Very Low | Low | `javaReferencesGate_test.go:44` filters `testdata/` segments and `:173-174` exempts every `_test.go` file. Both new artefacts are structurally exempt. The Java fixture uses `jakarta.persistence.*`, not `javax.persistence.*`, so even the `javax` token does not appear. |
| R-005 | RNF-001 grep at PC01US-014 catches `JPA`, `Lombok`, `Entity`, `Setter`, `NoArgsConstructor` in the new test file | Low | Medium | RNF-001 is scoped to `internal/domain`, `internal/application`, `internal/cli/*.go` (non-test). The new test file is `_test.go` and is therefore exempt by the same convention used in PC01US-002 (whose test file is allowed to mention `Service` and `RequiredArgsConstructor`). The test file SHOULD use the literal annotation names in `require.Contains` assertions — that is the AC. |
| R-006 | If the evaluator's `missing` ordering ever changes from "required-declaration order" to something else, Scenario 2 still passes (single-element list) but Determinism could regress | Low | Low | `TestAuditCmd_Integration_JpaEntityContract_Determinism` runs the same fixture twice and asserts byte equality, catching any non-determinism in the evidence assembly. |

## Section 9 — Parallel Execution Plan

```yaml
tiers:
  - id: 6
    name: Fixtures and integration test for jpa-entity-contract
    depends_on: []
    groups:
      - id: T6-G1
        scope:
          create:
            - testdata/pc01us003JpaEntityContract/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us003JpaEntityContract/projectClean/pom.xml
            - testdata/pc01us003JpaEntityContract/projectClean/project-state.yaml
            - testdata/pc01us003JpaEntityContract/projectClean/src/main/java/com/acme/infrastructure/persistence/entity/OrderEntity.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Clean-state fixture for PC01US-003 Scenario 1. Profile head copied
          verbatim from testdata/pc01us002UsecaseImplStereotype/projectClean
          (so the org.springframework.boot detector loads the user profile);
          appended audit rule has id jpa-entity-contract, kind
          required_annotations, path_scope /infrastructure/persistence/entity/,
          annotations 'Entity,Table,Getter,Setter,NoArgsConstructor,AllArgsConstructor'.
          OrderEntity.java declares all six annotations, so
          missingAnnotations returns [] and no violation fires. pom.xml is
          byte-identical to the PC01US-002 reference. Manifest is minimal
          schema_version 2 with one module rooted at the entity package.
          Independent of T6-G2 and T6-G3 — parallel-safe.

      - id: T6-G2
        scope:
          create:
            - testdata/pc01us003JpaEntityContract/projectMissing/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us003JpaEntityContract/projectMissing/pom.xml
            - testdata/pc01us003JpaEntityContract/projectMissing/project-state.yaml
            - testdata/pc01us003JpaEntityContract/projectMissing/src/main/java/com/acme/infrastructure/persistence/entity/OrderEntity.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Missing-Setter fixture for PC01US-003 Scenario 2. Profile YAML,
          pom.xml, and project-state.yaml are byte-identical to T6-G1.
          OrderEntity.java is identical to the clean variant EXCEPT the
          '@Setter' annotation line and the 'import lombok.Setter;' line are
          removed; all five other annotations remain so the evaluator emits
          exactly one violation with Message containing 'missing=[Setter]'.
          Independent of T6-G1 and T6-G3 — parallel-safe.

      - id: T6-G3
        scope:
          create:
            - internal/cli/command/jpaEntityContractIntegration_test.go
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: M
        notes: >
          Integration test exercising audit end-to-end via real treesitter,
          fsprofile, fsmanifest, audituc adapters. Replicates the
          newAuditCmdFor*(t, workDir, manifestPath) helper inline (renamed
          newAuditCmdForJpaEntityContract) — same constructor parameters as
          usecaseImplStereotypeIntegration_test.go:47-61. Three test
          functions (AllSixPresentNoViolation, MissingSetterReportsEvidence,
          Determinism) — all parallel-safe via t.TempDir + t.Parallel.
          Asserts string-contains rather than byte-for-byte golden. Depends
          only on T6-G1 + T6-G2 fixture paths existing at runtime; no
          compile-time dependency on either group.
```
