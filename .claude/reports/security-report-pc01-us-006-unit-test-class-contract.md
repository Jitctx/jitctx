# Security Audit — pc01-us-006-unit-test-class-contract

**Scope**: changes for PC01US-006 (Require @ExtendWith and @DisplayName on unit-test classes).

## Files Audited

Modified Go:
- internal/domain/model/javaFileSummary.go
- internal/domain/service/auditRuleEvaluator.go
- internal/domain/service/auditRuleEvaluator_test.go
- internal/infrastructure/treesitter/parser.go
- internal/infrastructure/treesitter/parser_test.go
- internal/infrastructure/treesitter/queries.go
- internal/infrastructure/treesitter/fields.go

New Go:
- internal/cli/command/unitTestClassContractIntegration_test.go

New testdata fixtures (Java + YAML + pom.xml under testdata/pc01us006UnitTestClassContract/{projectClean,projectWrongArg,projectMissingDisplayName}).

## Pillar Findings

### 1. Dependency CVEs
No new third-party imports. The PR uses only `strings`, `slices`, `regexp`, `path` (stdlib) and pre-existing internal packages plus the already-vendored `go-tree-sitter`. **No findings.**

### 2. Filesystem Safety
- `parseExpectedValues` operates on in-memory strings only — no filesystem I/O.
- `extractAnnotations` / `findFirstPositionalArg` traverse Tree-sitter nodes already produced by the existing parser; no new file reads or path manipulation.
- The integration test uses `t.TempDir()` and the existing `copyFixture` helper; no path concatenation outside the temp root.

**No findings.**

### 3. Hardcoded Secrets
None. Strings introduced (`"missing="`, `"annotation="`, `"expected_value="`, `"actual="`, plus literals `"("`, `")"`, `","`, `"="`) are formatting tokens, not secrets. Fixture YAML/Java carry only test identifiers.

### 4. Insecure Configuration
- The new optional `expected_values` parameter is parsed defensively: malformed pieces (no `=`, empty key) are dropped silently rather than panicking.
- `findFirstPositionalArg` returns the verbatim source slice between known node boundaries — no formatting / shell-out / unsafe deserialisation.
- Regex compilation paths in `auditRuleEvaluator.go` are unchanged by this PR; the new code uses `strings.Cut` only.
- Map iteration (a non-determinism risk, not a security risk per se) is avoided in the emit path: `expected` is iterated as an ordered slice (PC01RNF-003).

**No findings.**

## Verdict

**CLEAN — no security findings (no severity > INFORMATIONAL).**
