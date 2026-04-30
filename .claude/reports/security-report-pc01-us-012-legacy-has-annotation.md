# Security Report — PC01US-012 Legacy `has_annotation` Backward Compatibility

**Feature**: pc01-us-012-legacy-has-annotation
**Date**: 2026-04-30
**Auditor**: @security-gate (acting via QA coordinator)
**Status**: CLEAN

## Scope

Production:
- `internal/infrastructure/fsprofile/legacyHasAnnotation.go` (NEW)
- `internal/infrastructure/fsprofile/dto.go` (MODIFIED)
- `internal/infrastructure/fsprofile/bundleDto.go` (MODIFIED)
- `internal/infrastructure/fsprofile/auditLoader.go` (MODIFIED)
- `internal/infrastructure/fsprofile/bundleMapper.go` (MODIFIED)

Tests:
- `internal/infrastructure/fsprofile/legacyHasAnnotation_test.go` (NEW)
- `internal/cli/command/auditIntegration_test.go` (MODIFIED — appended)

Fixtures:
- `testdata/pc01us012LegacyHasAnnotation/projectMissingService/{project-state.yaml,pom.xml,.jitctx/profiles/spring-boot-hexagonal.yaml,src/main/java/com/acme/application/usecase/PlaceOrder.java}`

## Pillars

### 1. Dependency CVEs
No new imports. `legacyHasAnnotation.go` adds the stdlib `maps` package only
(Go 1.21+, already required by the toolchain). No `go.mod` change. **No findings.**

### 2. Filesystem Safety
The translator is a pure function: no `os` calls, no `path/filepath`, no
network. The two call sites (`auditLoader.go:61`, `bundleMapper.go:142`) live
inside loaders whose path-traversal posture is unchanged by this feature —
the pre-existing SEC-001 guard at `auditLoader.go:36-39`
(`strings.ContainsAny(profileName, "/\\")` + `..` rejection) is intact, and
`readProfileData` continues to enforce `strings.HasPrefix(candidate,
rootAbs+sep)` after `filepath.Clean`. **No findings.**

### 3. Hardcoded Secrets
`grep -nE "(password|secret|token|api_key|apikey|http://|https://|BEGIN PRIVATE)"`
across all new/modified files returned no matches. The fixture pom.xml hosts
only the public Spring artifact coordinates, which are public-by-design Maven
identifiers, not secrets. **No findings.**

### 4. Insecure Configuration
- `yaml.NewDecoder` + `dec.KnownFields(true)` is preserved on
  `auditLoader.go:46-47`. The new `HasAnnotation` field on `auditRuleDTO` /
  `bundleAuditRuleDTO` is explicitly tagged, so strict mode does not falsely
  reject legacy profiles that previously emitted `has_annotation:`.
- The translator never logs the raw rule body or params map (no PII / no
  unbounded user input echoed via `slog`).
- Fresh-map invariant (`maps.Copy` into a freshly allocated `effParams`)
  prevents downstream mutation from aliasing back into the parsed DTO,
  which is a small defence-in-depth win against accidental cross-rule
  state contamination.

**No findings.**

## Verdict

CLEAN — no auto-fixable findings, no manual findings.
