# Security Audit Report — EP04US-007 Profile Validation Command

**Feature:** ep04us007
**Auditor:** @security-gate
**Date:** 2026-04-27
**Pillars audited:** Dependency CVEs · Filesystem safety · Hardcoded secrets · Insecure configuration

## Scope

Modified production:
- `internal/cli/command/profileCmd.go`
- `internal/cli/format/errors.go`
- `internal/cli/root.go`
- `internal/cli/wire.go`
- `internal/domain/errors/errors.go`

Created production:
- `internal/application/usecase/profilevalidateuc/usecase.go`
- `internal/cli/command/profileValidateCmd.go`
- `internal/domain/usecase/profilevalidateuc/port.go`
- `internal/domain/vo/profile/validateProfileInput.go`
- `internal/domain/vo/profile/validateProfileOutput.go`

Created tests / fixtures: as listed in scope.

---

## Pillar 1 — Dependency CVEs

| Module | Version | Status |
|---|---|---|
| `gopkg.in/yaml.v3` | v3.0.1 | Latest. v3.0.1 was the fix release for CVE-2022-28948 (panic on malformed YAML). No newer 3.x exists. CLEAN. |
| `github.com/spf13/cobra` | v1.8.1 | No known CVEs. CLEAN. |
| `github.com/stretchr/testify` | v1.9.0 | Test-only. No known CVEs. CLEAN. |

No new third-party transitive dependencies introduced by this feature. `go.mod` and `go.sum` were not modified.

**Verdict:** PASS.

---

## Pillar 2 — Filesystem Safety

The validate use case touches the user-supplied `<path>` argument and reads `<path>/profile.yaml`. Findings:

### SEC-001 — User-supplied path is not lexically cleaned before joining (INFORMATIONAL)

**File:** `internal/application/usecase/profilevalidateuc/usecase.go:74-89, 164, 219, 261`

`filepath.Abs(in.Path)` is called once at the top of `Execute` and the result `abs` is reused as the directory base for three `os.ReadFile(filepath.Join(dir, "profile.yaml"))` calls inside the helpers. Behaviour:

1. `filepath.Abs` already calls `filepath.Clean`, so `abs` is canonicalised. Path traversal via `../` cannot escape the user's intended directory because the CLI is single-tenant and the user supplied the path themselves.
2. The helpers each receive `abs` (not `in.Path`) so traversal sequences in the original input are eliminated before any read.
3. There is no symlink dereferencing concern relevant to a CLI tool the user runs themselves on their own filesystem.

**Risk:** Low. jitctx is a developer-machine CLI; the operator IS the trust boundary. No multi-tenant context, no unprivileged caller. This finding is recorded for completeness.

**Auto-fixable:** N/A.

### SEC-002 — Information disclosure in error message via `%q` formatting (INFORMATIONAL)

**File:** `internal/application/usecase/profilevalidateuc/usecase.go:76, 80, 91`

```go
return out, fmt.Errorf("validate profile: resolve path %q: %w", in.Path, err)
msg := fmt.Sprintf("profile path %q does not exist", in.Path)
msg := fmt.Sprintf("profile path %q is not a directory", in.Path)
```

`%q` echoes the user-supplied path back to stderr. For a developer CLI this is intentional and helpful — the user wants to see what they typed. No secrets pass through this surface (the path is a profile directory chosen by the user, not a credential).

**Risk:** None for this threat model.
**Auto-fixable:** N/A.

### SEC-003 — Test-fixture file modes (INFORMATIONAL)

**File:** `internal/application/usecase/profilevalidateuc/usecase_test.go:52, 71-72`

Tests write fixtures with mode `0o644` and create directories with `0o755`. These are conventional and identical to every other test in the repository (verified via grep across `internal/cli/command/*Integration_test.go`). They live under `t.TempDir()` and are deleted automatically. No risk.

**Auto-fixable:** N/A.

**Verdict:** PASS.

---

## Pillar 3 — Hardcoded Secrets

Manual scan of all in-scope production and test files for credentials, API keys, tokens, connection strings, private-key markers, or vendor-specific secret formats.

- `knownClassificationKeys` in usecase.go:31-37 contains only public schema keys (`kind`, `implements_all`, `implements_none`, `has_annotation`, `path_contains`) — these are sourced verbatim from the public `bundleClassificationDTO` in `internal/infrastructure/fsprofile/bundleDto.go:89-95`. Not secrets.
- Test fixtures (`testdata/ep04us007/*/profile.yaml`) contain only sample type names (`x.y.Service`, `service.java.tmpl`). Not secrets.
- Integration test logger writes to `nopWriter` discard sink — no log exfiltration even in test mode.

No secrets found.

**Verdict:** PASS.

---

## Pillar 4 — Insecure Configuration

### Cobra command surface

`profileValidateCmd.go` declares `cobra.ExactArgs(1)` for the path argument and consumes only `cmd.Context()`, `cmd.OutOrStdout()`, and `cmd.ErrOrStderr()`. No flags, no env vars, no shell-out, no template rendering of user input. The output `"Profile valid"` is a fixed literal.

### Logger handling

Both `New(loader, logger)` (use case) and `NewProfileValidateCmd(uc, logger)` (cobra) tolerate `nil` and fall back to `slog.Default()` (which is configured in `internal/config` to write to stderr). No accidental stdout pollution.

### YAML parsing safety

`yaml.Unmarshal(data, &root)` decodes into `*yaml.Node` (DOM). yaml.v3 v3.0.1 has had its known billion-laughs / panic-on-malformed-input issues addressed. `yaml.Unmarshal` does NOT execute arbitrary code.

### Stdout/stderr discipline

- `"Profile valid"` → stdout via `cmd.OutOrStdout()`.
- Warning lines (`unknown classification field 'X'`) → stderr via `cmd.ErrOrStderr()`.
- Fatal errors → propagated via `RunE` return, rendered by cobra to stderr.

Compliant with the project rule "stdout = output, stderr = logs/diagnostics".

**Verdict:** PASS.

---

## Summary

| Pillar | Findings | Severity |
|---|---|---|
| Dependency CVEs | 0 | — |
| Filesystem safety | 3 informational | INFO |
| Hardcoded secrets | 0 | — |
| Insecure configuration | 0 | — |

**Overall verdict:** CLEAN — no fixable findings (all observations are informational and auto-fixable: N/A).

No remediation required.
