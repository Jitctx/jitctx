# Code Review Report — PC01US-005 (Enforce test method naming convention)

**Feature:** `pc01-us-005-test-method-naming`
**Date:** 2026-04-29
**Requirements:** `docs/propose-changes-01/quality-gate-evaluators.feature`

## Summary

| Severity | Count |
|----------|-------|
| BLOCKER  | 0     |
| WARNING  | 0     |
| INFO     | 2     |

**Verdict: PASS (CLEAN).**

All special-focus areas verified; tests + lint clean.

## Architectural conformity

- `model.JavaMethod` additions (`Name`, `Annotations`, `Line`) are append-only
  pure data. No framework deps in `internal/domain/model/`.
- `evalMethodNaming` lives in `internal/domain/service/`, no I/O, no goroutines,
  accepts only domain types. Pure function semantics preserved.
- New `extractMethodName` helper in `internal/infrastructure/treesitter/parser.go`
  is correctly placed in infrastructure, returning a domain-side string.
- `internal/cli/command/testMethodNamingIntegration_test.go` wires real adapters
  via `appaudituc.New(...)`, mirroring sibling integration tests. No facade
  introduced between cobra and the use case.

## PC01RNF-001 — language-neutral engine

Verified by grep over the five production files in scope. No literal occurrences
of `JUnit`, `Mockito`, `Spring`, `Lombok`, `Autowired`, or `JPA` introduced by
this PR (`ContractJPAAdapter` in `mapper.go` is a pre-existing enum, not added).
The trigger annotation is read exclusively from `rule.Params["triggered_by"]`;
`evalMethodNaming` source contains no literal `"Test"`.

## PC01RNF-003 — deterministic output

`evalMethodNaming` iterates `summary.Declarations` (slice) and `decl.Methods`
(slice); annotation lookup uses `slices.Contains` (linear scan). No map
iteration. The substitution context map keys are disjoint
(`{file}` / `{name}` / `{expected_pattern}` / `{triggered_by}`), so
`substituteSuggestion`'s iteration order is irrelevant to output.
Confirmed end-to-end by `TestAuditCmd_Integration_TestMethodNaming_Determinism`
which asserts byte-identical normalized output across two temp dirs.

## PC01RF-004 — evidence substring

YAML rule description:
`'Test method violates naming convention: name={name}, expected_pattern={expected_pattern}'`
Substitution context populates `{name}=testFindUser` and
`{expected_pattern}=^should[A-Z].*_when[A-Z].*$`, producing the literal AC2
substring. Asserted verbatim by
`TestAuditCmd_Integration_TestMethodNaming_NonCompliantFlagsViolation`.

## Defensive `regexp.Compile`

`evalMethodNaming` uses `regexp.Compile` (not `MustCompile`) and returns `nil`
on compile error. Covered by the `malformed-regex` subtest in
`TestAuditEvaluator_MethodNaming_MalformedRuleEmitsNothing`. No panic path.

## R-002 — additive `JavaMethod` fields

Existing parser tests still pass. New fields are zero-valued by default; no
test that constructs `JavaMethod{Signature: ...}` literals had to change.

## Go idioms & naming

- Filenames camelCase: `auditRule.go`, `javaFileSummary.go`,
  `testMethodNamingIntegration_test.go`. Compliant with project convention.
- `extractMethodName` private, documented, single responsibility.
- Use-case package imports (`appaudituc`, `command`, `service`,
  `fsconfig`, `fsmanifest`, `fsprofile`, `treesitter`) follow established
  alias style in sibling integration tests.

## Test consistency vs. `quality-gate-evaluators.feature`

| Acceptance Criterion | Backing test | Status |
|----------------------|--------------|--------|
| AC1 (compliant `should*_when*` produces no violation) | `TestAuditCmd_Integration_TestMethodNaming_CompliantNoViolation` + `TestAuditEvaluator_MethodNaming_Compliant_NoViolation` | PASS |
| AC2 (`testFindUser` flagged with literal evidence) | `TestAuditCmd_Integration_TestMethodNaming_NonCompliantFlagsViolation` + `TestAuditEvaluator_MethodNaming_NonCompliant_FlagsWithEvidence` | PASS |
| PC01RNF-003 determinism | `TestAuditCmd_Integration_TestMethodNaming_Determinism` | PASS |
| PC01RF-008 exempt_paths | `TestAuditEvaluator_MethodNaming_RespectsExemptPaths` | PASS |

## INFO findings (no action required)

### INFO-001 — Fixture under `src/main/java/` instead of `src/test/java/`

Documented deviation from plan §7.4.2. The walker (`walker.go:21-67`) only
scans `src/main/java/`, mirroring the precedent set by PC01US-004's `Helper.java`.
Path-scope semantics for `src/test/java/` are still exercised by unit tests
that build `JavaFileSummary` literals directly. Acceptable.

### INFO-002 — Local copy of audit-cmd wiring helper

`newAuditCmdForTestMethodNaming` duplicates wiring already present in sibling
integration tests. Per the Q3 resolution noted in the test file, no upstream
DRY refactor is in scope for this PR; consolidation is a future cleanup.

## Lint / test status

- `go vet ./...` — clean
- `gofmt -l .` — clean
- `go test ./internal/domain/service/ ./internal/infrastructure/treesitter/ ./internal/infrastructure/fsprofile/ ./internal/cli/command/ -count=1` — all PASS
- `go test ./internal/cli/command/ -run TestAuditCmd_Integration_TestMethodNaming -v -count=1` — 3/3 PASS

## BLOCKERs

None.
