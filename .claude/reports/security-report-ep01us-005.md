# Security Report — ep01us-005 (Spring Boot Hexagonal Profile)

**Auditor:** @security-gate (executed inline by QA Coordinator)
**Date:** 2026-04-17
**Scope:**
- internal/application/usecase/scanuc/usecase.go
- internal/cli/command/scanIntegration_test.go
- internal/domain/model/frameworkProfile.go
- internal/domain/service/profileClassifier_test.go
- internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal.yaml
- internal/infrastructure/fsprofile/detector.go
- internal/infrastructure/fsprofile/detector_test.go
- internal/infrastructure/fsprofile/loader.go
- testdata/springBootMinimal/expected/project-state.yaml
- testdata/springBootMinimal/project/src/main/java/com/app/user_management/service/CreateUserServiceImpl.java

## Pillar 1 — Dependency CVEs

- No new third-party imports introduced. The feature reuses
  `gopkg.in/yaml.v3 v3.0.1` (already vetted in ep01us-004) and the stdlib
  (`embed`, `io/fs`, `os`, `context`, `log/slog`, `path/filepath`,
  `sort`, `strings`, `bytes`).
- `github.com/stretchr/testify/require` is test-only and unchanged.

**Status:** CLEAN.

## Pillar 2 — Filesystem Safety

- `fsprofile/loader.go` — explicit path-traversal guard on `Load(name)`:
  rejects `/`, `\`, `..` in the name (lines 45-48), and re-validates after
  `filepath.Join`+`filepath.Clean` that the candidate remains under
  `rootAbs+separator` (lines 53-58). Matches the SEC-001 remediation from
  earlier cycles.
- `fsprofile/detector.go` — enumerates entries via `os.ReadDir(userDir)`
  and `fs.ReadDir(embeddedProfiles, "bundled")`. No user-controlled paths
  are concatenated into the read target beyond `filepath.Join(userDir,
  e.Name())`, where `e.Name()` is the directory entry name (safe; no
  traversal).
- `profileMatches` (detector.go:118) calls `fs.ReadFile(fsys,
  matcher.Name)`. `matcher.Name` can come from a user-authored custom
  profile YAML. Because `fsys` is an `os.DirFS`, `fs.ValidPath` rejects
  names containing `..` or leading `/`, so a malicious custom profile
  cannot escape the project root via this path.
- `scanuc/usecase.go` constructs `os.DirFS(input.WorkDir)` from the CLI
  `--path` flag. This is end-user input; no server boundary is crossed.
  Safe.
- Test fixtures are written under `t.TempDir()` with modes `0o755`
  (dirs) / `0o644` (files). Bounded to the test sandbox; no TOCTOU,
  symlink, or traversal surface.
- `testdata/springBootMinimal/...` is a checked-in fixture tree. No
  runtime writes into it.

**Status:** CLEAN.

## Pillar 3 — Hardcoded Secrets

- No credentials, tokens, API keys, or connection strings in any scope
  file.
- Java fixture contains only placeholder identifiers
  (`CreateUserServiceImpl`, `User`). No PII, no secrets.
- Expected manifest YAML contains no sensitive content.

**Status:** CLEAN.

## Pillar 4 — Insecure Configuration

- YAML decoding uses `dec.KnownFields(true)` (loader.go:127) — strict
  mode rejects unknown fields, preventing silent acceptance of
  malicious/unexpected keys in custom profile YAML.
- Detector matches profile file contents via
  `strings.Contains(strings.ToLower(...), strings.ToLower(...))`
  (detector.go:124). No regex compilation of user input, so no ReDoS.
- `frameworkProfile.go` — domain model has zero serialization tags
  (compliant with CLAUDE.md's "Sem tags YAML/JSON no domínio"). DTOs in
  `fsprofile` handle marshalling.
- Bundled profile `spring-boot-hexagonal.yaml` ships inside the binary
  via `//go:embed bundled/*.yaml`. Read-only at runtime.
- Custom profile parse failures are logged at WARN and the offending
  profile is skipped (detector.go:80-84; loader.go:64-69) — fail-open
  for classification but without silent data corruption. Acceptable per
  EP01RF-012 §Exceptions.
- Integration test `TestScanCmd_Integration_ManifestTraversalRejected`
  asserts the manifest path is guarded against escaping the project
  directory.

**Status:** CLEAN.

## Findings

None. Zero auto-fixable items.

## Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| HIGH     | 0 |
| MEDIUM   | 0 |
| LOW      | 0 |
| INFO     | 0 |

**Verdict: CLEAN**
