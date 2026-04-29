# Security Report — PC01US-004 (forbid @Autowired field injection)

Feature: `pc01-us-004-forbid-autowired-field-injection`
Auditor: @security-gate (executed by qa-coordinator)
Date: 2026-04-29

## Scope reviewed

- `internal/domain/model/auditRule.go`
- `internal/domain/model/javaFileSummary.go`
- `internal/domain/service/auditRuleEvaluator.go`
- `internal/domain/service/auditRuleEvaluator_test.go`
- `internal/infrastructure/fsprofile/mapper.go`
- `internal/infrastructure/treesitter/fields.go`
- `internal/cli/command/forbidAutowiredFieldInjectionIntegration_test.go`
- `testdata/pc01us004ForbidAutowiredFieldInjection/**`

## Pillars evaluated

### 1. Dependency CVEs
No new third-party dependencies introduced. `go.mod` not modified by this
feature. Pillar: **PASS**.

### 2. Filesystem safety
- `extractFieldDeclaration` reads only from the already-loaded `[]byte` source
  buffer; no path traversal surface.
- `matchPathGlob` operates on already-normalised forward-slash strings from
  `summary.Path`. It uses `path.Match` (not `filepath.Match`), so platform
  separators are not interpreted. A malformed glob never panics — `path.Match`
  returns an error which is treated as no-match.
- `pathExempt` reads `rule.Params["exempt_paths"]` — profile YAML, trusted
  at load time. No FS reads.
- Integration tests use `t.TempDir()`, never user-controlled paths.
Pillar: **PASS**.

### 3. Hardcoded secrets
None. Identifiers like `"Autowired"` in fixtures and a single doc-comment
example are public framework-API names, not credentials. Pillar: **PASS**.

### 4. Insecure configuration
- The new evaluator is permissive on malformed rules (returns nil rather
  than panicking) — this is a defensive-by-design choice, not a security
  concern.
- `regexp.MustCompile` in pre-existing `evalInterfaceNaming` is **out of
  scope** for this PR (not modified) and is fed by trusted profile YAML.
Pillar: **PASS**.

## Findings
None.

## Verdict
**CLEAN.** No security findings of any severity. Zero auto-fixable issues.
