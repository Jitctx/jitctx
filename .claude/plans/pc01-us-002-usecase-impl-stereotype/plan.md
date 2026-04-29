# Plan — PC01US-002 Require Spring stereotype + constructor-injection on UseCaseImpl

## Section 0 — Summary

- Feature: a profile author can declare a single `required_annotations` audit rule (`id: usecase-impl-stereotype`) that flags any class in package `application.usecase` that does not declare both `@Service` and `@RequiredArgsConstructor`, with `missing=[...]` evidence on the violation message.
- Requirement IDs covered: **PC01US-002** (story); transitively **PC01RF-001** (multi-annotation all-of evaluator), **PC01RF-009** (evidence-rich messages), **PC01RNF-001** (engine language-neutrality), **PC01RNF-003** (deterministic output), **PC01RNF-006** (real-parser integration tests).
- Layers touched: **testdata fixtures + integration test only**. No Go source under `internal/domain`, `internal/application`, `internal/infrastructure`, or `internal/cli` is created or modified.
- Tiers active: **6** (Tier 6 only). Tiers 1-5 collapse because PC01RF-001 is already in production (see Section 8).
- Guidelines loaded: `.claude/guidelines/integration-test-layer-guidelines.yml`.
- Estimated file count: **7 new** (2 fixture trees × 3 files + 1 integration-test Go file), **0 modified**.

### Verified working-tree facts (2026-04-29)

1. `model.AuditKindRequiredAnnotations` sentinel exists at `/workspaces/jitctx/internal/domain/model/auditRule.go:21` with PC01RF-001 reference.
2. `evalRequiredAnnotations` at `/workspaces/jitctx/internal/domain/service/auditRuleEvaluator.go:287-325` substitutes `{missing}` with `"[" + strings.Join(missing, ",") + "]"`, producing the exact `missing=[RequiredArgsConstructor]` shape the Gherkin asserts. Order is preserved by `missingAnnotations` (lines 360-375).
3. `path_scope` is a `strings.Contains` substring filter (line 298).
4. Default `node_types` is `class_declaration` when omitted (lines 302-305) — interfaces are silently skipped.
5. Audit pipeline already wires the rule end-to-end: `/workspaces/jitctx/internal/cli/wire.go:189` builds the evaluator and passes it to `appaudituc.New(...)` (lines 232-246); `command.NewAuditCmd` registers the cobra command.
6. `auditRuleDTO` at `/workspaces/jitctx/internal/infrastructure/fsprofile/dto.go:39-46` decodes `params: map[string]string`, and `mapper.go:17` whitelists `AuditKindRequiredAnnotations` in `knownAuditRuleKinds`.
7. The `*UseCaseImpl` *name* selector is NOT yet in the evaluator — the fixture controls scope by placing exactly one class in `/application/usecase/` (a name-pattern selector belongs to PC01RF-004 / PC01US-005, out of scope here).
8. PC01RNF-001 forbidden-token gate (`/workspaces/jitctx/internal/qualitygate/exemptions.go:11-20`) does not list `Service`, `RequiredArgsConstructor`, or `UseCaseImpl`; testdata and `_test.go` files are structurally exempt anyway.

## Section 1 — File Set

| # | File | Action | Layer | Tier | Group |
|---|------|--------|-------|------|-------|
| 1 | `testdata/pc01us002UsecaseImplStereotype/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml` | create | testdata | 6 | T6-G1 |
| 2 | `testdata/pc01us002UsecaseImplStereotype/projectClean/project-state.yaml` | create | testdata | 6 | T6-G1 |
| 3 | `testdata/pc01us002UsecaseImplStereotype/projectClean/src/main/java/com/acme/application/usecase/FindUserUseCaseImpl.java` | create | testdata | 6 | T6-G1 |
| 4 | `testdata/pc01us002UsecaseImplStereotype/projectMissing/.jitctx/profiles/spring-boot-hexagonal.yaml` | create | testdata | 6 | T6-G2 |
| 5 | `testdata/pc01us002UsecaseImplStereotype/projectMissing/project-state.yaml` | create | testdata | 6 | T6-G2 |
| 6 | `testdata/pc01us002UsecaseImplStereotype/projectMissing/src/main/java/com/acme/application/usecase/FindUserUseCaseImpl.java` | create | testdata | 6 | T6-G2 |
| 7 | `internal/cli/command/usecaseImplStereotypeIntegration_test.go` | create | tests | 6 | T6-G3 |

## Section 2 — Frozen Domain Contract

No new Go signatures. The four touchpoints the test consumes (verbatim from `main`):

- `model.AuditRuleKind`, `AuditKindRequiredAnnotations`, `AuditSeverity*`, `AuditRule{ID, Kind, Severity, Description, Suggestion, Params}` (`internal/domain/model/auditRule.go`).
- `auditvo.AuditViolation{RuleID, Kind, Severity, ModuleID, FilePath, Line, Message, Suggestion}` (`internal/domain/vo/audit/auditViolation.go`).
- `audituc.UseCase.Execute(ctx, AuditProjectInput) (AuditProjectOutput, error)` (`internal/domain/usecase/audituc/usecase.go`).
- `command.NewAuditCmd(uc audituc.UseCase, _ *slog.Logger) *cobra.Command` (`internal/cli/command/auditCmd.go`).

`Deps` struct in `wire.go` is unchanged (`Deps.Audit audituc.UseCase` already wired). No new sentinel error.

## Section 3 — Domain Layer Plan

N/A. `evalRequiredAnnotations` already implements all-of semantics with `missing=[...]` evidence, locked by `auditRuleEvaluator_test.go:437-615` (PC01US-001 coverage exercises the same code path).

## Section 4 — Infrastructure Layer Plan

N/A. `treesitter/parser.go:138 extractAnnotations` already produces simple-name annotations (`["Service", "RequiredArgsConstructor"]` for the fixture). `fsprofile/auditLoader.go:32` already round-trips `required_annotations` rules via `dto.go:39 auditRuleDTO`. `fsmanifest` already parses `schema_version: 2` manifests with the module map needed.

## Section 5 — Application Layer Plan

N/A. `audituc.Impl.Execute` orchestration steps 5-9 already apply `required_annotations` rules through `service.AuditEvaluator.EvaluateFile` and produce sorted violations.

## Section 6 — Presentation Layer Plan

N/A. `auditCmd.go` and `format/audit.go` are consumed unchanged. The integration test asserts via `require.Contains` on stdout — no new golden, no formatter change.

## Section 7 — Composition Root + Tests Plan

### Wiring

No edits to wire.go / root.go / execute.go / main.go. Audit is a complete vertical slice; the new test reuses `appaudituc.New(...)` exactly as `auditIntegration_test.go:47-61` does.

### Unit tests

None. `auditRuleEvaluator_test.go` already locks the evaluator behaviour for the same code path (PC01US-001 coverage). A second unit test for the literal pair `{Service, RequiredArgsConstructor}` would be redundant — the evaluator is annotation-name-agnostic (finding 2).

### Integration test

**File**: `internal/cli/command/usecaseImplStereotypeIntegration_test.go`.

**Helper**: replicate `newAuditCmdFor(t, workDir, manifestPath)` from `auditIntegration_test.go:27-73` inline (Q3 below — no upstream refactor in this PR).

**Test functions** (each `t.Parallel()`):

1. `TestAuditCmd_Integration_UsecaseImplStereotype_BothPresentNoViolation` — copies `projectClean` into `t.TempDir()`, runs `audit --dir <tmp> --manifest <tmp>/project-state.yaml`, asserts no error, stdout contains `No sintatic violations detected` and does NOT contain `usecase-impl-stereotype`. Backs PC01US-002 Scenario 1.
2. `TestAuditCmd_Integration_UsecaseImplStereotype_MissingRequiredArgsConstructorReportsEvidence` — copies `projectMissing` into `t.TempDir()`, runs the same command, asserts no error, stdout contains `[usecase-impl-stereotype]`, `missing=[RequiredArgsConstructor]`, and the file path; asserts exactly one occurrence of `[usecase-impl-stereotype]`. Backs PC01US-002 Scenario 2.
3. (Optional) `TestAuditCmd_Integration_UsecaseImplStereotype_Determinism` — runs the failing fixture twice, asserts byte-identical stdout. Backs PC01RNF-003 for the new rule wiring.

### Fixture content

Profile YAML (shared between both fixtures):

```yaml
audit_rules:
  - id: usecase-impl-stereotype
    kind: required_annotations
    severity: ERROR
    description: 'UseCase implementation {name} must declare all of [{required}]; missing={missing}'
    suggestion: 'Add the missing annotation(s) to {name}: {missing}'
    params:
      path_scope: /application/usecase/
      annotations: 'Service,RequiredArgsConstructor'
```

The full profile head (`name`, `languages`, `query_lang`, `detect`, `module_detection`, classification `rules`) is copied verbatim from `testdata/auditClean/.../spring-boot-hexagonal.yaml` so the loader does not reject the file (R-001 mitigation).

`projectClean` Java fixture declares `@Service @RequiredArgsConstructor`; `projectMissing` declares `@Service` only.

`project-state.yaml`: minimal `schema_version: 2` manifest with one module covering `src/main/java/com/acme/application/usecase`.

### Why no golden report

A golden would couple the test to incidental rendering details (suggestion line, profile-name comment) outside the AC. Focused `require.Contains` on rule ID, evidence string, and file path tracks only the AC text.

## Section 8 — Open Questions & Risks

### PC01RF-001 status report

**PC01RF-001 IS COMPLETE on `main`.** Commit `414955c` (2026-04-29, "feat(audit): add required_annotations rule kind with all-of semantics") landed the sentinel, evaluator, fsprofile mapper entry, and 6 unit tests. Contract verification against PC01US-002 Gherkin is zero-mismatch:

| Scenario assertion | Implementation that satisfies it |
|---|---|
| "no violation reported" (Scenario 1) | `evalRequiredAnnotations` returns nil when `missingAnnotations` empty (line 313) |
| "evidence missing=[X]" (Scenario 2) | `ctx["missing"] = "[" + strings.Join(missing, ",") + "]"` then substituted into Description (lines 320, 405-410) |
| Rule selectable by id `usecase-impl-stereotype` | `auditLoader.go:73` round-trips arbitrary `id` strings |
| Java file in `application.usecase` | `path_scope: /application/usecase/` substring filter (finding 3) |
| Class declares both annotations | `parser.go:188` populates `Annotations` simple-names; exact-match in `missingAnnotations` |

**No engine delta required.** This plan is a tests-only ratification + fixture authoring plan, mirroring EP03US-007.

### Open questions

| # | Question | Blocking | Resolution |
|---|---|---|---|
| Q1 | Story says "on `*UseCaseImpl` classes" but evaluator does not consult `decl.Name`. Restrict by name pattern? | No | Fixture controls scope. Name-pattern selector belongs to PC01RF-004 / PC01US-005 (out of scope). |
| Q2 | Add the new rule to `profiles/spring-boot-hexagonal.yaml` and bundled-profile copies? | No | Story is "profile author can author this rule". Wider rollout would alter every existing audit golden — out of scope. Rule lives only in the two new fixture profiles. |
| Q3 | Lift `newAuditCmdFor` from `auditIntegration_test.go` into `helpers_test.go` first? | No | Refactor not bundled with AC. New file copies the helper; follow-up PR can DRY it. |
| Q4 | `description: '... [{required}]; missing={missing}'` brace-bracket shape acceptable? | No | Same shape used by PC01US-001 unit-test rule (`auditRuleEvaluator_test.go:448`). AC asserts only on `missing=[RequiredArgsConstructor]`. |

### Risks

| ID | Risk | Probability | Impact | Mitigation |
|---|---|---|---|---|
| R-001 | Fixture profile missing required `detect:` block, loader rejects | Low | Medium | Copy full profile head from `testdata/auditClean/...` verbatim; append the single new audit rule |
| R-002 | Tree-sitter extracts FQN annotations differently, test miscompares | Low | Low | `parser_test.go:141` proves simple-name extraction is stable; `auditClean/CreateUserService.java` already parses `@Service` correctly today |
| R-003 | Future refactor renames `{missing}` token, silently breaking Scenario 2 | Medium | Medium | Test asserts literal `missing=[RequiredArgsConstructor]` substring — any rename fails this test loud |
| R-004 | Forbidden-token gate flags new fixture/test for `spring`/`Spring` literals | Very Low | Low | Gate walks production .go only; testdata + `_test.go` exempt structurally |

## Section 9 — Parallel Execution Plan

```yaml
tiers:
  - id: 6
    name: Fixtures and integration test for usecase-impl-stereotype
    depends_on: []
    groups:
      - id: T6-G1
        scope:
          create:
            - testdata/pc01us002UsecaseImplStereotype/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us002UsecaseImplStereotype/projectClean/project-state.yaml
            - testdata/pc01us002UsecaseImplStereotype/projectClean/src/main/java/com/acme/application/usecase/FindUserUseCaseImpl.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Clean-state fixture for PC01US-002 Scenario 1. Profile declares one
          rule (id usecase-impl-stereotype, kind required_annotations,
          path_scope /application/usecase/, annotations
          Service,RequiredArgsConstructor). Java class declares both
          annotations so missingAnnotations returns []. Manifest is minimal
          schema_version 2 with one module rooted at the usecase package.

      - id: T6-G2
        scope:
          create:
            - testdata/pc01us002UsecaseImplStereotype/projectMissing/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us002UsecaseImplStereotype/projectMissing/project-state.yaml
            - testdata/pc01us002UsecaseImplStereotype/projectMissing/src/main/java/com/acme/application/usecase/FindUserUseCaseImpl.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Missing-annotation fixture for PC01US-002 Scenario 2. Profile YAML
          and project-state.yaml byte-identical to T6-G1. Java class declares
          @Service ONLY, so the evaluator emits exactly one violation with
          Message containing missing=[RequiredArgsConstructor]. Independent
          of T6-G1 — parallel-safe.

      - id: T6-G3
        scope:
          create:
            - internal/cli/command/usecaseImplStereotypeIntegration_test.go
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: M
        notes: >
          Integration test exercising audit end-to-end via real treesitter,
          fsprofile, fsmanifest, audituc adapters. Replicates the
          newAuditCmdFor(t, workDir, manifestPath) helper from
          auditIntegration_test.go:27. Three test functions
          (BothPresentNoViolation,
          MissingRequiredArgsConstructorReportsEvidence, optional
          Determinism) — all parallel-safe via t.TempDir + t.Parallel.
          Asserts string-contains rather than byte-for-byte golden. Depends
          only on T6-G1 + T6-G2 fixture paths existing.
```
