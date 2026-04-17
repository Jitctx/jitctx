# Security Report — EP01US-003 (Filter Query by Type and Tags)

**Date**: 2026-04-17
**Scope**: 5 files (modified + created for EP01US-003)
**Auditor**: @security-gate
**Verdict**: CLEAN

---

## Scope

Modified:
- `internal/cli/command/queryCmd.go`
- `internal/cli/format/markdown.go`
- `internal/cli/format/markdown_test.go`

Created:
- `internal/cli/command/queryFiltersIntegration_test.go`
- `internal/cli/command/queryTypeParser_test.go`

---

## Summary

| Pillar                       | Findings |
| ---------------------------- | -------- |
| Dependency CVEs              | 0        |
| Filesystem safety            | 0        |
| Hardcoded secrets            | 0        |
| Insecure configuration       | 0        |

No actionable security findings. All findings below are INFORMATIONAL.

---

## Pillar 1 — Dependency CVEs

US-003 introduces **no new third-party dependencies**. The modified and
created files import only:

- stdlib: `bytes`, `context`, `encoding/json`, `fmt`, `io`, `log/slog`,
  `os`, `path/filepath`, `regexp`, `strings`, `testing`, `time`.
- pre-existing first-party modules under
  `github.com/jitctx/jitctx/internal/...`.
- already-vetted third parties:
  - `github.com/spf13/cobra` v1.8.1 — no known CVEs.
  - `github.com/stretchr/testify/require` v1.9.0 — test-only.
  - `gopkg.in/yaml.v3` v3.0.1 — current, CVE-2022-28948 only affects < v3.0.0.

`go.mod` / `go.sum` are unchanged by the US-003 commit. **Finding count: 0**.

---

## Pillar 2 — Filesystem Safety

### Production code (`queryCmd.go`, `markdown.go`)

Neither file performs direct filesystem I/O.
- `queryCmd.go` only invokes the use case via the injected
  `queryuc.UseCase` interface; all path handling is delegated to the
  infrastructure adapters vetted under US-002 (where `SEC-INFO-001`
  documented the `os.DirFS` containment behaviour).
- `markdown.go` writes to an `io.Writer` supplied by cobra
  (`cmd.OutOrStdout()`); no file descriptor is opened.

The new `parseArtifactTypes` helper only performs string `TrimSpace` on
flag values before handing them to `vo.ArtifactType.Validate()`. Unknown
values are dropped with a `slog.Warn` record. No path, no shell, no
exec — pure string classification.

### Test code

`queryFiltersIntegration_test.go` writes exclusively under `t.TempDir()`
(Go-managed, auto-cleaned). Directories are created with mode `0o755` and
files with `0o644`, matching the existing `queryIntegration_test.go`
convention. `markdown_test.go` and `queryTypeParser_test.go` do not touch
the filesystem.

### SEC-INFO-001 — `.jitctx/<type>/<id>.md` path construction in tests

**Severity**: INFORMATIONAL
**Location**: `internal/cli/command/queryFiltersIntegration_test.go:41-48`
**Auto-fixable**: N/A

Test fixtures assemble paths via `filepath.Join(tmpDir, ".jitctx", ctx.ctxType)`
where `ctx.ctxType` is a test-controlled literal (`"guidelines"`, `"scenarios"`,
`"requirements"`). No untrusted input reaches `filepath.Join`, and
`t.TempDir()` roots the write so any hypothetical traversal is contained
within the per-test sandbox. Noted for parity with the US-002 manifest-traversal
review; no action required.

**Finding count: 0**.

---

## Pillar 3 — Hardcoded Secrets

Grep across scope for secret-like tokens (`password`, `secret`, `api[_-]?key`,
`token`, `bearer`, `AKIA`, private-key headers) returned only:

- `vo.TokenBudget` references (token *budget*, not authentication token).
- `github.com/.../internal/infrastructure/token` (the token-estimator
  adapter, also a budget concept).
- `slog` structured-logging attribute names.

No credentials, API keys, or bearer tokens are embedded. Fixture tags
(`"security"`, `"auth"`) are scenario labels matching the Gherkin feature
file, not secrets. **Finding count: 0**.

---

## Pillar 4 — Insecure Configuration

### Logging

The unknown-`--type` warning is routed through the cobra-injected
`*slog.Logger` and only logs:

```go
logger.Warn("ignoring unknown --type value",
    "value", trimmed,
    "accepted", "guidelines|requirements|scenarios|contracts",
)
```

The `value` attribute echoes user-supplied flag input — this is the
expected diagnostic payload and does not leak secrets because the flag
only accepts artifact-type identifiers (validated against a closed enum).
Per `config/logger` convention, the logger writes to **stderr**, keeping
stdout clean for machine output (also a correctness property, not only
security).

### Cobra configuration

- `Args: cobra.NoArgs` — rejects surprise positional input.
- `MarkFlagRequired("module")` — prevents a no-filter broadcast query.
- `StringSliceVar` for `--tags` and `--type` applies cobra's standard
  CSV parsing; no shell evaluation occurs.
- `cmd.SilenceUsage = true` is set in tests; production wiring lives in
  `internal/cli/wire.go` (out of scope for this diff).

### Output format selection

`WriteQueryResult` dispatches on `format` with a default branch falling
through to markdown — unknown `--output` values silently render as
markdown. This is a UX choice, not a security issue (no injection
surface); noted for code-review awareness.

**Finding count: 0**.

---

## Verdict

**CLEAN** — zero auto-fixable security findings. US-003 is strictly
additive over US-002's already-vetted surface (same adapters, same
filesystem contract, no new deps, no new write paths outside
`t.TempDir()`). Safe to merge from a security standpoint.
