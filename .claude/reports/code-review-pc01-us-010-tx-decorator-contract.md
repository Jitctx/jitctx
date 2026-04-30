# Code Review — pc01-us-010-tx-decorator-contract

**Reviewer**: @code-reviewer (executed inline by @qa-coordinator due to no Task subagent dispatch in this session)
**Date**: 2026-04-30
**Scope**: see security report.
**Requirements**:
- `docs/propose-changes-01/quality-gate-evaluators.feature` lines 161–177
- `docs/propose-changes-01/quality-gate-evaluators.md` PC01US-010 section (lines 460–482)

## 1. Architectural Conformity

- The new evaluator logic lives entirely in `internal/domain/service/auditRuleEvaluator.go` (domain/service layer). It depends only on `internal/domain/model` and `internal/domain/vo/audit`, respecting the dependency direction. No imports from `internal/infrastructure` or `internal/application`.
- Integration test wiring in `txDecoratorContractIntegration_test.go` lives in `internal/cli/command/` (presentation) under the `_test` package and only consumes infrastructure adapters at construction time, mirroring the established pattern (`integrationTestBaseRequiredAnnotationsIntegration_test.go`, `forbidAutowiredFieldInjectionIntegration_test.go`).
- `context.Context` is plumbed through the cobra command; the use case is invoked via `cmd.ExecuteContext(context.Background())`. Consistent with the codebase.
- No new ports, no new sentinels needed — the change is purely additive on an existing evaluator branch.

**Result**: PASS.

## 2. Go Idioms & Naming

- File names: all camelCase (`auditRuleEvaluator.go`, `txDecoratorContractIntegration_test.go`) per CLAUDE.md.
- New helper `isEmptyAnnotationArg` is unexported, lowerCamelCase, single-responsibility. Doc comment leads with the function name as required by Go convention.
- New rule parameter key `non_empty_value_annotations` follows the existing snake_case convention used for other params (`path_scope`, `expected_values`, `node_types`).
- Switch statement in `isEmptyAnnotationArg` uses raw-string literals for the quote forms — clearer than escaping. Good idiom.
- Determinism contract is documented in the function-level comment and enforced by iterating `splitNonEmpty(...)` instead of a Go map.

**Result**: PASS.

## 3. Code-Smell Metrics

- `evalRequiredAnnotations` grew from ~70 to ~95 lines after this change. Still readable; the three branches (missing, expected_values, non_empty_value) are sequential and well-commented. No nested complexity escalation.
- No duplicated logic: the non-empty branch reuses `splitNonEmpty`, `slices.Contains`, and `makeViolation`.
- `isEmptyAnnotationArg` is 7 LOC. Cyclomatic complexity 1.
- The integration-test helper `newAuditCmdForTxDecoratorContract` is a near-duplicate of `newAuditCmdForIntegrationTestBaseRequiredAnnotations`. The Q-DRY note at line 25 explicitly acknowledges this and defers consolidation. Acceptable as a WARNING-level observation.

**Result**: PASS WITH WARNINGS.

## 4. Test Consistency

Acceptance criteria mapping:

| Spec scenario (lines 164–177 of .feature) | Unit test | Integration test |
|---|---|---|
| AC1: Decorator with @Primary + @Qualifier("txDecorator") passes | `…_PrimaryAndQualifierWithNonEmptyValuePass` | `TestAuditCmd_Integration_TxDecoratorContract_PrimaryAndQualifierWithNonEmptyValuePass` |
| AC2: @Primary only → `missing=[Qualifier]` | `…_PrimaryOnlyEmitsMissingQualifier` | `TestAuditCmd_Integration_TxDecoratorContract_PrimaryOnlyEmitsMissingQualifier` |
| AC3: @Primary + @Qualifier("") → `annotation=Qualifier, value=empty, expected=non-empty` | `…_EmptyQualifierValueEmitsNonEmptyEvidence` | `TestAuditCmd_Integration_TxDecoratorContract_EmptyQualifierValueEmitsNonEmptyEvidence` |
| PC01RNF-003 emit ordering | `…_OrderingMissingThenExpectedValuesThenNonEmpty` | covered transitively |

All assertions check the canonical evidence substrings exactly as the spec mandates. The unit-test layer additionally exercises the three empty forms (`""`, `"\""`, `"''"`) and three non-empty forms (`"x"`, `"\"x\""`, `"\" \""`) of `isEmptyAnnotationArg` via subtests, satisfying PC01RNF-001's requirement that the helper is dispatched purely on parser-captured text.

`go test ./...` passes (verified). `go vet ./...` clean. `gofmt -l` clean.

**Result**: PASS.

## Findings

### BLOCKERs
None.

### WARNINGs
- **W-001** — `internal/cli/command/txDecoratorContractIntegration_test.go:27`: helper `newAuditCmdForTxDecoratorContract` duplicates the wiring of `newAuditCmdForIntegrationTestBaseRequiredAnnotations` and `newAuditCmdForForbidAutowiredFieldInjection`. The author flagged this in the inline comment ("Q-DRY resolution: local copy acceptable; no upstream refactor in this PR"). Suggestion: a future small refactor extracts a single `newAuditCmdForFixtureProject(t, workDir, manifestPath)` helper into a shared `_test.go` package file, reducing N copies. Out of scope for this PR.

### INFOs
- **I-001** — `internal/domain/service/auditRuleEvaluator.go:418`: each call iterates `splitNonEmpty(rule.Params["non_empty_value_annotations"])` per declaration. For typical rule profiles the list has ≤3 entries and declarations per file are bounded; no measurable cost. Could be hoisted out of the per-declaration loop alongside `parseExpectedValues(...)` for symmetry, but this is purely stylistic.

## Summary

| Severity | Count |
|---|---|
| BLOCKER | 0 |
| WARNING | 1 |
| INFO | 1 |

**Verdict**: PASS WITH WARNINGS. No fixable BLOCKERs.
