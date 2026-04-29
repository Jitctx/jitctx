# Code Review Report — PC01US-004 (forbid @Autowired field injection)

Feature: `pc01-us-004-forbid-autowired-field-injection`
Auditor: @code-reviewer (executed by qa-coordinator)
Requirements: `docs/propose-changes-01/quality-gate-evaluators.feature`
Date: 2026-04-29

## Scope

7 modified/added Go files plus 12 testdata fixture files (3 small Spring-Boot
projects under `testdata/pc01us004ForbidAutowiredFieldInjection/`).

## Architectural conformity

| Concern | Result |
|---|---|
| Domain stays free of infra imports | PASS — `auditRuleEvaluator.go` imports only stdlib + `domain/model` + `domain/vo/audit`. |
| ISP — one method per port | PASS — no port surface modified; `ListJavaFieldsPort` continues to host one method satisfied by the same `*Parser`. |
| Composition root unchanged | PASS — `internal/cli/wire.go` not modified; the new evaluator kind is reached via the existing `EvaluateFile` switch. |
| stdout/stderr discipline | PASS — evaluator does no I/O. Integration test uses `slog.NewTextHandler(io.Discard, ...)`. |
| No YAML tags on domain | PASS — `JavaField` additions (`Annotations`, `Line`) carry no struct tags. |
| PC01RNF-001 language-neutrality | PASS with note — the only `Autowired` reference in the engine source is in a doc-comment example explicitly flagged in the special-focus brief as content-of-data illustration. No engine logic branches on the literal. |

## Go idioms & naming

- Filenames: `forbidAutowiredFieldInjectionIntegration_test.go` follows the
  project's camelCase convention (`_test.go` suffix is the toolchain
  exception). PASS.
- Identifiers: all English, idiomatic (`evalForbiddenAnnotations`,
  `intersectAnnotations`, `matchPathGlob`, `pathExempt`,
  `extractFieldDeclaration`). PASS.
- Error wrapping: not introduced by this PR; existing wrapping in
  `fields.go` is unchanged. PASS.
- `context.Context` first parameter: preserved in
  `ListJavaFields(ctx, fsys, path)`. PASS.
- Unused imports / dead code: none.

## Code-smell metrics

| File | Notes |
|---|---|
| `auditRuleEvaluator.go` | `evalForbiddenAnnotations` is ~60 LOC with one switch; cyclomatic ~6 — within budget. `matchSegments` recursion bounded by `len(pattern) * len(path)`; safe for filesystem-realistic inputs. |
| `fields.go` | `extractFieldDeclaration` ~50 LOC, two clear loops (type pass, declarator pass). Below threshold. |
| Test files | Each `Test*` is single-purpose; table-driven where appropriate (`TestAuditEvaluator_MatchPathGlob` covers 7 cases). |

No duplication concerns. The drive-by lint fix on
`evalAnnotationPathMismatch` (replacing inner-loop with `slices.Contains`) is
a small improvement.

## Test consistency vs. requirements

Cross-checked against `quality-gate-evaluators.feature` and PC01US-004
acceptance criteria embedded in the integration test docstrings:

- Scenario 1 (production field flagged, line surfaced) — covered by
  `TestAuditCmd_Integration_ForbidAutowired_OnProductionFieldFlagsViolation`
  and unit `..._FieldScope_FlagsAutowired`. Both assert
  `Line == field.Line` and `found=[Autowired]` evidence.
- Scenario 2 (testsupport exempt) — covered by
  `..._OnTestSupportFieldIsExempted` and `..._FieldScope_RespectsExemptPaths`.
- Scenario 3 (constructor parameter allowed) — covered by
  `..._OnConstructorParameterIsAllowed`. The fixture's constructor parameter
  carries `@Autowired` but is correctly NOT a `field_declaration`, so
  `extractFields` skips it. Confirmed: `extractFields` only iterates
  `class_body` direct children and filters on `nodeFieldDecl`.
- PC01RNF-003 (deterministic output) — covered by
  `..._Determinism` (byte-identical normalised stdout across two temp dirs)
  AND structurally by `intersectAnnotations` preserving the order declared
  in `params["annotations"]`.
- PC01RF-008 path-exemption matrix — `TestAuditEvaluator_MatchPathGlob` covers
  the 7 cases listed in plan §3.3 including the zero-segment trailing `**`
  case (`**/foo/**` matches `.../foo`). I traced this case manually:
  patSegs `["**","foo","**"]` against pathSegs `[..."acme","foo"]` —
  the inner loop tries `i=5` placing `"foo"` against pat `"foo"`, then
  trailing `"**"` returns true on empty pathSegs. Matches plan expectation.

## Param-key registry §2.10 alignment

YAML profile (`testdata/.../spring-boot-hexagonal.yaml`) declares:
```
path_scope: src/main/java/
annotations: 'Autowired'
target: field
node_types: class_declaration
exempt_paths: '**/testsupport/**'
```
All keys match `evalForbiddenAnnotations` reads. The leading-slash defect
called out in the brief (`/src/main/java/` vs `src/main/java/`) is correctly
resolved — the walker yields paths without leading slash, and
`strings.Contains(summary.Path, pathScope)` works correctly with
`src/main/java/` as substring.

## R-002 risk verification

`grep` for existing `JavaField{...}` literals confirms the only consumers
are tests in `auditRuleEvaluator_test.go` (which were updated alongside the
struct change) and the producer in `fields.go`. No `treesitter/fields_test.go`
exists. Additive struct fields in Go are backward-compatible — no broken
call sites detected. `go test ./... -count=1` is green.

## Findings
None.

## Verdict
**CLEAN.** Zero BLOCKERs. Zero WARNINGs. Zero INFOs.
