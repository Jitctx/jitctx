# Security Report — pc01-us-010-tx-decorator-contract

**Auditor**: @security-gate (executed inline by @qa-coordinator due to no Task subagent dispatch in this session)
**Date**: 2026-04-30
**Scope**:
- `internal/domain/service/auditRuleEvaluator.go` (modified)
- `internal/domain/service/auditRuleEvaluator_test.go` (modified)
- `internal/cli/command/txDecoratorContractIntegration_test.go` (new)
- `testdata/pc01us010TxDecoratorContract/projectClean/**` (new)
- `testdata/pc01us010TxDecoratorContract/projectMissingQualifier/**` (new)
- `testdata/pc01us010TxDecoratorContract/projectEmptyQualifier/**` (new)

## Pillar 1 — Dependency CVEs

No new Go dependencies introduced. `go.mod` untouched by this feature. The fixture `pom.xml` files reference `spring-boot-starter-parent:3.2.0` but they are static test data parsed only as text by the framework-detection heuristic (`fsprofile.Detector`); they are never resolved, downloaded, or executed. No supply-chain surface.

**Findings**: none.

## Pillar 2 — Filesystem Safety

The new evaluator branch (`evalRequiredAnnotations` PC01US-010 block) and helper (`isEmptyAnnotationArg`) operate exclusively on in-memory `model.JavaFileSummary` structs. No `os.Open`, `filepath.Join` from user input, no path traversal surface introduced. The integration test uses `t.TempDir()` and the project's existing `copyFixture` helper, identical to the pattern in `forbidAutowiredFieldInjectionIntegration_test.go` and `integrationTestBaseRequiredAnnotationsIntegration_test.go`.

**Findings**: none.

## Pillar 3 — Hardcoded Secrets

Grep across the SCOPE for `password|secret|api[_-]?key|token|bearer|aws_|AKIA` returned only innocuous matches: the word "token" in code comments (`"*"` wildcard token, parser tokens), and the field name `AnnotationArgs`. Test fixtures contain no credentials, no `.env`, no keys.

**Findings**: none.

## Pillar 4 — Insecure Configuration

- The new YAML rule fixture (`spring-boot-hexagonal.yaml` per project) is well-scoped (`path_scope: src/main/java/com/acme/application/decorator/`) and does not relax security defaults.
- `isEmptyAnnotationArg` switch is exhaustive on three string forms (`""`, `\"\"`, `''`) — no panic surface, no regex injection, no unbounded compilation.
- `splitNonEmpty` over `non_empty_value_annotations` reuses the existing helper; behaviour is bounded by the rule profile (loaded from disk by infrastructure under loader validation) and per-declaration iteration is O(annotations × required) with no exponential blowup.
- Engine-neutrality (PC01RNF-001) verified: `grep -E "Primary|Qualifier|Transactional|Decorator|TxDecorator|Spring"` against `auditRuleEvaluator.go` returns only the two pre-existing meta-comments (lines 345 and 1091) flagged as acceptable in the SCOPE preamble. No new occurrences in production code.

**Findings**: none.

## Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| HIGH | 0 |
| MEDIUM | 0 |
| LOW | 0 |
| INFORMATIONAL | 0 |

**Verdict**: CLEAN. No fixable findings.
