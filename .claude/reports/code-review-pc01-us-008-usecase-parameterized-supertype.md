# Code Review — PC01US-008 (Parameterized UseCase<I,O> supertype rule)

Feature: pc01-us-008-usecase-parameterized-supertype
Reviewer: @code-reviewer
Requirements: /workspaces/jitctx/docs/propose-changes-01/quality-gate-evaluators.md (PC01US-008, PC01RF-006)
Plan: /workspaces/jitctx/.claude/plans/pc01-us-008-usecase-parameterized-supertype/plan.md

## Summary

The PR adds a new `required_parameterized_supertype` audit rule kind plus
the supporting domain model and Tree-sitter extraction. The implementation
is focused, layered correctly, and fully covered by unit + integration
tests (all green). No BLOCKERs.

## Architectural conformity

- New domain types `SupertypeKind` and `ParameterizedSupertype` are pure
  structs in `internal/domain/model/javaFileSummary.go` — no framework
  imports, no YAML tags. PASS.
- New evaluator `evalRequiredParameterizedSupertype` lives in
  `internal/domain/service` as a pure function, no I/O, no goroutines.
  PASS.
- Tree-sitter changes are confined to `internal/infrastructure/treesitter`
  and the JavaDeclaration is populated through the existing extraction
  paths (class/interface/enum/record). PASS.
- Mapper registration in `internal/infrastructure/fsprofile/mapper.go`
  updates `knownAuditRuleKinds` only — no domain leakage. PASS.
- No `internal/infrastructure` import from `internal/application` (none
  added). PASS.
- ISP and "errors not panic" maintained — defensive returns on malformed
  input rather than panic.

## Engine-neutrality (PC01RNF-001)

`internal/domain/service/auditRuleEvaluator.go` and the new domain model
contain no Java/Spring/Lombok identifiers. All
`UseCase`/`Spring`/`Lombok`/`@Autowired` strings live in:
- profile YAML params (which is the entire point of the parameterized
  rule kind),
- unit-test fixtures and integration testdata,
- `internal/application/usecase/scaffolduc/javaScaffoldConstants.go`
  (pre-existing scaffolder template constants — out of scope for this PR).

The `UseCase` Go interface name in `internal/domain/usecase/*` and
`internal/application/usecase/*` is jitctx's own port abstraction, not a
reference to the Java framework type. PASS.

## Determinism (PC01RNF-003)

- Source-order iteration over `summary.Declarations`,
  `decl.ParameterizedSupertypes`, and `argGlobs`. PASS.
- "First-match wins" via `c := outerMatched[0]` is documented in the
  doc-comment. PASS.
- `parseExpectedValues` and `parseSupertypePattern` use ordered slices;
  no Go-map iteration drives violation order. PASS.
- `splitGenericArgs` (parser) and `splitTopLevel` (evaluator) are both
  deterministic depth-aware splitters. PASS.

## Go idioms & naming

- All new files in camelCase: `auditRule.go`, `javaFileSummary.go`,
  `auditRuleEvaluator.go`, `parser.go`, `mapper.go`,
  `usecaseParameterizedSupertypeIntegration_test.go`. PASS.
- Doc comments on every exported identifier; package-level conventions
  upheld. PASS.
- Errors wrapped with `fmt.Errorf("ctx: %w", err)` in
  `parser.ParseJavaFile`. PASS.
- `context.Context` first parameter on the parser entry point (no new
  ports added). PASS.

## Reuse audit

Confirmed — `globMatch` in `internal/domain/service/profileClassifier.go`
is reused by both `matchOuterGlob` and `matchInnerGlob` wrappers. No
re-declaration. PASS.

## Test consistency

Unit tests for the new evaluator (auditRuleEvaluator_test.go):
- AC1 matching-supertype-passes
- AC2 actual=none with literal-evidence assertion
- AC3 wrong-arity (two sub-tests covering both evidence templates)
- Q1 non-parameterized-supertype ignored
- Path-scope miss (out of scope)
- Q4 scoped FQN outer-glob match

Parser tests (parser_test.go):
- Parameterized implements (Kind, Outer, TypeArgs)
- Parameterized extends (class)
- Nested generic preserves depth-zero-only commas

Integration tests (usecaseParameterizedSupertypeIntegration_test.go):
- AC1 clean project — no `[usecase-supertype]` violation
- AC2 missing supertype — `expected_supertype=UseCase<*,*>, actual=none`
- AC3 wrong arity — `expected_arity=2` and `actual_arity=1` evidence,
  exactly one violation

`go test ./... -count=1`, `go vet ./...`, `gofmt -l .` all clean. PASS.

## Findings

### BLOCKER — none

### WARNING — 1

- **W-001** `evalRequiredParameterizedSupertype` is ~125 lines (function
  body lines 1021-1147 in auditRuleEvaluator.go). The function is
  linearly structured into five well-commented steps and is no harder to
  follow than the neighbouring evaluators (`evalRequiredAnnotations` is
  ~63 lines body; `evalForbiddenFieldTypePattern` is ~46), but it is
  large enough to merit a future refactor (e.g. extract
  `selectCandidate` and `buildViolationContext` helpers). Not a blocker;
  surfaced for awareness.

### INFO — 2

- **I-001** `matchOuterGlob` and `matchInnerGlob` are thin wrappers over
  `globMatch` with identical semantics. The doc-comment justifies the
  duplication ("kept distinct for readability at call sites"). Defensible
  given the call-site readability gain; future readers may collapse to a
  single helper if signatures of the two roles ever diverge.
- **I-002** `splitTopLevel` (domain/service/auditRuleEvaluator.go ~line
  968) and `splitGenericArgs` (infrastructure/treesitter/parser.go ~line
  411) both implement depth-aware comma splitting on angle brackets.
  Their signatures and call contexts differ enough (slots-only vs
  (outer, args, ok)) and they live in different architectural layers
  (domain vs infrastructure), so extracting a shared util would force a
  cross-layer dependency. Acceptable as-is.

## Verdict

CLEAN — no BLOCKERs. One WARNING and two INFOs are advisory only and do
not block the pipeline.
