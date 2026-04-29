# Code Review — pc01-us-006-unit-test-class-contract

**Scope**: changes for PC01US-006 (Require @ExtendWith and @DisplayName on unit-test classes).
**Requirements**: docs/propose-changes-01/quality-gate-evaluators.feature
**Plan**: .claude/plans/pc01-us-006-unit-test-class-contract/plan.md

## Architectural Conformity

- `JavaDeclaration.AnnotationArgs` lives in `internal/domain/model/` — pure struct, no framework deps. Compliant with domain-layer guideline.
- `evalRequiredAnnotations` extension stays in `internal/domain/service/auditRuleEvaluator.go` — domain service, no I/O. Compliant.
- Tree-sitter parser changes confined to `internal/infrastructure/treesitter/`. The infrastructure does not leak into application/domain. Compliant.
- No new ports were introduced; the existing `ParseJavaFilePort` signature is preserved (the additional `args` slice is an internal-package detail). Compliant with ISP.
- No `internal/infrastructure` imports inside `internal/application` or `internal/domain` — verified.

## Go Idioms & Naming

- Filenames: `unitTestClassContractIntegration_test.go` follows camelCase project convention. Compliant.
- `expectedValuePair` is unexported — appropriate (local helper type).
- `parseExpectedValues` returns `nil` for empty input rather than `[]expectedValuePair{}` — idiomatic.
- Use of `strings.Cut` in place of `strings.Index(..., "=")` is the modern idiom (Go 1.18+).
- Three named-return slices `(simple, qualified, args []string)` — the docstring justifies the named returns and the invariant `len(simple)==len(qualified)==len(args)`. Acceptable; readability is good.
- No panics; defensive returns of `nil`/`""` for malformed inputs match the surrounding evaluator style.

## Code-Smell Metrics

- `evalRequiredAnnotations` complexity grew modestly (one extra loop over `expected`). Function still ~40 lines. Acceptable.
- `findFirstPositionalArg` is one new ~30-line helper with clear single responsibility.
- No duplication: `annotationArgsMap` centralises the simple→arg mapping used by class/interface/enum/record extractors.
- `parser.go` four extractor sites pay a small repetition cost (`var args []string; …; args` then `annotationArgsMap`). This is a pre-existing pattern (mirrors how Annotations/QualifiedAnnotations are wired). Acceptable.

## Test Consistency vs Requirements

Acceptance Criteria from the .feature file are honoured:

- **AC1 (clean class passes)**: covered by `TestAuditEvaluator_RequiredAnnotations_UnitTestClassWithBothAnnotationsAndCorrectArgPasses` and `TestAuditCmd_Integration_UnitTestClassContract_CleanFixture_NoViolation`.
- **AC2 (wrong ExtendWith arg)**: literal evidence `annotation=ExtendWith, expected_value=MockitoExtension.class, actual=SpringExtension.class` (note spaces after commas) asserted both in the unit test and the integration test. Verbatim verified against the evaluator emit code (`auditRuleEvaluator.go:376-378`).
- **AC3 (missing @DisplayName)**: literal evidence `missing=[DisplayName]` asserted in unit + integration tests.
- **PC01RNF-003 (deterministic output)**: `TestAuditEvaluator_RequiredAnnotations_BothMissingAndMismatchEmitTwoOrderedViolations` locks ordering invariant (missing first, then expected_values-order mismatches). `TestAuditCmd_Integration_UnitTestClassContract_Determinism` runs the same fixture twice with byte-identical output after temp-dir normalisation.
- **PC01RNF-001 (engine language-neutrality)**: grep confirmed `MockitoExtension`, `SpringExtension`, `ExtendWith`, `DisplayName` appear in this PR's engine source files (`internal/domain/...`, `internal/infrastructure/treesitter/...`) **only inside doc-comments** — not as engine identifiers. Pre-existing references in `internal/application/usecase/scaffolduc/` are out of scope.
- **PC01RF-007 / Q2 (verbatim arg capture)**: `TestParser_ClassWithExtendWithArg_PopulatesAnnotationArgs` locks both class-literal `.class` suffix and string-literal surrounding quotes.
- **Backward compatibility**: `TestAuditEvaluator_RequiredAnnotations_NoExpectedValues_BehavesLikeBefore` and pre-existing PC01US-001/002/003 tests continue to pass — verified via `go test ./... -count=1` (all green).
- **R-002 (additive `JavaDeclaration` field)**: zero existing parser/evaluator tests broken.

Edge cases also covered: duplicate `expected_values` keys (last wins), malformed pieces (silently skipped), marker annotations producing empty arg, classes with no annotations producing nil/empty `AnnotationArgs`.

## BLOCKERs

None.

## WARNINGs

None material. Two minor observations (not blockers):

- W-001 (style): The four extractor functions in `parser.go` (class/interface/enum/record) each repeat the three-slice unpacking. A small refactor could centralise this, but it duplicates an existing pattern and would inflate the PR. Defer.
- W-002 (docs): The doc-comment on `expected_values` (auditRuleEvaluator.go:275-294) is dense — long parameter docstrings are a project-wide style and consistent with surrounding helpers. No change required.

## INFOs

- I-001: Three fixture profiles are byte-identical (verified via `diff`). The orchestrator's late fix to `projectMissingDisplayName/spring-boot-hexagonal.yaml` is sound.
- I-002: `parseExpectedValues` correctly preserves first-appearance order while letting last value win for duplicate keys — slightly subtle. The unit test `…ExpectedValuesParsing_DuplicateKeyLastWins` locks this invariant.
- I-003: `findFirstPositionalArg` accepts BOTH `argument_list` and `annotation_argument_list` per Plan §2.3 deviation note — forward-compatible.

## Verdict

**CLEAN — no BLOCKERs, no WARNINGs that block merge.**
