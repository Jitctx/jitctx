# Code Review — PC01US-012 Legacy `has_annotation` Backward Compatibility

**Feature**: pc01-us-012-legacy-has-annotation
**Date**: 2026-04-30
**Reviewer**: @code-reviewer (acting via QA coordinator)
**Status**: CLEAN (0 BLOCKERs, 0 WARNINGs, 2 INFOs)

## Scope

(See security report — same file list.)

## Architectural Conformity

- **Layer placement**: Translation lives in `infrastructure/fsprofile/`,
  the only layer that ever speaks YAML. Domain (`internal/domain`) and
  application (`internal/application`) remain unaware of the legacy
  shortcut, which is correct per the project's DDD/Clean-Arch posture.
- **ISP**: No new port introduced. The translator is a package-private
  helper consumed by two existing loader entry points. No port surface
  bloat.
- **Composition root**: `internal/cli/wire.go` not touched. Correct —
  no new dependency to wire.
- **Domain purity**: Verified via grep for `Spring|Lombok|Mockito|JPA|@Service`
  across production scope — only the test file references "Autowired" as a
  generic string fixture (test value, not a Spring identifier coupling).
  Engine-neutrality (PC01RNF-001) preserved.

## Go Idioms & Naming

- Filename `legacyHasAnnotation.go` follows the project's camelCase
  convention (CLAUDE.md "Nomes de arquivos Go").
- Test filename `legacyHasAnnotation_test.go` correctly uses the
  toolchain-mandated `_test.go` suffix.
- Function name `translateLegacyHasAnnotation` is verb-led, lowercase
  (package-private — correct, since it has no caller outside `fsprofile`).
- Named return values `(effKind, effParams, translated)` are used to
  document the multi-return contract; the function is short enough that
  this is idiomatic rather than naked-return abuse.
- `maps.Copy` (Go 1.21 stdlib) replaces a hand-rolled loop — preferred
  per the project's "avoid mapsloop diagnostic" stance.

## Code-Smell Metrics

- **Cyclomatic complexity** of `translateLegacyHasAnnotation`: 4
  (well below the project's threshold).
- **Lines of code**: 30 production lines including comments. No
  duplication with existing fsprofile helpers.
- **Comment-to-code ratio**: high, but justified — the package-level
  doc comment encodes the FIRST-MATCH-WINS rule precedence which is
  load-bearing for downstream gates.
- The integration-test addition (~30 lines) follows the existing
  `auditIntegration_test.go` patterns (copyFixture + newAuditCmdFor +
  stdout substring assertions) verbatim.

## Test Consistency vs Requirements

Cross-checked against
`docs/propose-changes-01/quality-gate-evaluators.md` lines 514, 163, 173,
184:

- **PC01US-012 (line 514)**: AC1 "legacy `has_annotation: Service` rule
  on a class missing @Service flags exactly one violation with
  evidence equivalent to `required_annotations:[Service]`" — covered
  by `TestAuditCmd_Integration_PC01US012_LegacyHasAnnotation_FlagsMissing`
  via `Contains(stdout, "missing=[Service]")`,
  `Contains(stdout, "PlaceOrder.java")`, and
  `Contains(stdout, "[legacy-service-required]")`. AC1 satisfied.
- **PC01RF-012 (line 163)** "silent transition, no deprecation
  warning in this release" — verified by reading the translator: no
  `slog.Warn` is ever emitted from the translation path; only the
  pre-existing "unknown audit rule kind" WARN survives, which fires on
  unrecognised kinds, not on legacy translation. Documented inline at
  `legacyHasAnnotation.go:27-30`.
- **PC01RNF-001 (line 173, engine-neutrality)**: production files
  contain zero `Spring|Lombok|Mockito|Autowired|JPA` literals. The
  fixture is allowed to declare `has_annotation: Service` because
  fixtures are scope-exempt per project convention.
- **PC01RNF-003 (line 184, determinism)**: the translator returns a
  fresh `map[string]string` (`maps.Copy`), but maps are still
  iteration-order-unstable. Downstream serialisation (audit
  violation rendering) is the determinism boundary; the existing
  `TestAuditCmd_Integration_*_Determinism` suite (10 tests) was rerun
  and remained green, confirming no determinism regression introduced
  by the new pass-through path.

## Test Suite Status

- `go vet ./...` — clean.
- `gofmt -l .` — empty output.
- `go test ./...` — all packages pass.
- `go test ./... -run Integration -v` — all integration tests pass,
  including the 10 PC01RNF-003 determinism tests.
- `go test ./internal/infrastructure/fsprofile/ -run "LegacyHasAnnotation"`
  — 8 table cases + 2 standalone tests all PASS.

## Findings

### BLOCKERs

None.

### WARNINGs

None.

### INFOs

- **I-001** (`legacyHasAnnotation.go`): The doc-comment example uses
  the YAML value `Service` (a Spring identifier) for illustration. This
  is acceptable per the project's policy that doc-comments and fixtures
  may reference framework names as examples, but a future reviewer may
  want to swap it for a neutral name like `MyAnnotation` to harden
  PC01RNF-001 posture even in comments. **No action required.**
- **I-002** (`legacyHasAnnotation_test.go:66-68`): The test case
  `BothKindAndHasAnnotation_KindWins` uses `"Autowired"` as a generic
  annotation token. Same neutrality nit as I-001 — informational only.

## Verdict

CLEAN.
