# Security Report — PC01US-009 (Integration-Test Base Required Annotations)

Feature: pc01-us-009-it-base-required-annotations
Auditor: @security-gate (acting via QA coordinator)
Date: 2026-04-30

## Scope audited

- `internal/cli/command/integrationTestBaseRequiredAnnotationsIntegration_test.go` (new test file)
- `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectClean/**` (gitignored fixture)
- `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectWrongActiveProfile/**` (gitignored fixture)
- `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/projectMissingTestcontainers/**` (gitignored fixture)

## Pillars evaluated

### 1. Dependency CVEs
No dependencies added. The test imports only standard library plus already-vetted
internal packages (`appaudituc`, `command`, `service`, `fsconfig`, `fsmanifest`,
`fsprofile`, `treesitter`) and `github.com/stretchr/testify/require`.
**Finding count: 0.**

### 2. Filesystem safety
- `t.TempDir()` is used for all writable workspaces; `copyFixture` writes into
  that ephemeral directory only. No `os.RemoveAll` on attacker-controlled paths.
- Fixture files (`.java`, `.yaml`, `pom.xml`) live under
  `testdata/pc01us009IntegrationTestBaseRequiredAnnotations/` which is matched
  by the existing `testdata` entry in `.gitignore` (verified via
  `git check-ignore -v`). No secrets or large binaries committed.
- File modes used by `copyFixture` (`0o755`/`0o644`) are standard and the
  helper is shared/pre-existing — not introduced by this PR.
**Finding count: 0.**

### 3. Hardcoded secrets
- Strings present: `SpringBootTest`, `Testcontainers`, `ActiveProfiles`,
  `"test"`, `"prod"`, `com.acme`. None are credentials, tokens, API keys, or
  PII.
- `pom.xml` references public Maven coordinates only
  (`org.springframework.boot:spring-boot-starter-parent:3.2.0`).
**Finding count: 0.**

### 4. Insecure configuration
- No network, no HTTP server, no DB connection strings, no TLS configuration,
  no authentication code touched.
- Logger writes to `io.Discard` at `LevelError` — does not leak to disk.
- `cmd.SilenceUsage = true` is a UX setting, not a security boundary.
**Finding count: 0.**

## Summary

| Pillar                  | Findings | Severity   |
|-------------------------|----------|------------|
| Dependency CVEs         | 0        | —          |
| Filesystem safety       | 0        | —          |
| Hardcoded secrets       | 0        | —          |
| Insecure configuration  | 0        | —          |

**Verdict: CLEAN.** No fixable findings, nothing to remediate.
