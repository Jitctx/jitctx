# Security Report — PC01US-008 (Parameterized UseCase<I,O> supertype rule)

Feature: pc01-us-008-usecase-parameterized-supertype
Auditor: @security-gate
Scope:
- internal/domain/model/auditRule.go
- internal/domain/model/javaFileSummary.go
- internal/domain/service/auditRuleEvaluator.go
- internal/domain/service/auditRuleEvaluator_test.go
- internal/infrastructure/fsprofile/mapper.go
- internal/infrastructure/treesitter/parser.go
- internal/infrastructure/treesitter/parser_test.go
- internal/cli/command/usecaseParameterizedSupertypeIntegration_test.go
- testdata/pc01us008UseCaseParameterizedSupertype/** (fixtures)

## Pillar 1 — Dependency CVEs

No new third-party dependencies introduced. Imports added by this change:
- `strconv` (stdlib)
- `path` (stdlib, already present)
- existing `github.com/smacker/go-tree-sitter` (already in go.mod, no version bump)

No CVE exposure delta.

## Pillar 2 — Filesystem safety

- `parser.ParseJavaFile` reads via `fs.ReadFile(fsys, path)`. The caller passes
  a rooted `fs.FS` — no path traversal vector introduced by this PR.
- No new `os.Open`, `os.ReadFile`, `filepath.Join` with attacker-influenced
  path segments in production code paths.
- The integration test uses `t.TempDir()` and `filepath.Join` with hard-coded
  fixture names — test-only, no exposure.

## Pillar 3 — Hardcoded secrets

None found. The literal strings introduced (`"UseCase<*,*>"`,
`"application/usecase/"`, `"usecase-supertype"`) are rule identifiers and
glob patterns, not credentials.

## Pillar 4 — Insecure configuration

- The new evaluator `evalRequiredParameterizedSupertype` is defensive on
  malformed `expected_supertype` (returns nil on missing brackets) and on
  arity mismatches between `args` and `expected_supertype` (returns nil
  rather than panicking). No regex compilation in this evaluator (no ReDoS
  surface).
- `splitTopLevel` and `splitGenericArgs` walk the input byte-by-byte with a
  bounded depth counter; no recursion, no unbounded backtracking, no
  catastrophic-input vector.

## Findings

None. Severity: INFORMATIONAL aggregate.

## Verdict

CLEAN — no auto-fixable security findings; no manual-review findings of
severity > INFORMATIONAL.
