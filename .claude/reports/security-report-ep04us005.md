# Security audit — EP04US-005 (Tree-sitter Queries Bundled by Language)

Auditor: QA Coordinator (security pillar)
Date: 2026-04-27
Scope: 22 files (13 production + 8 tests + 1 .scm fixture)

## Pillars

| Pillar                       | Verdict | Notes                                                                             |
|------------------------------|---------|-----------------------------------------------------------------------------------|
| Dependency CVEs              | PASS    | No new third-party deps. `go.mod` unchanged. Stdlib only (`io/fs`, `embed`, `slices`, `sync`). |
| Filesystem safety            | PASS    | Path-traversal guards present in both bundled-name validation paths. embed.FS is read-only. |
| Hardcoded secrets            | PASS    | No credentials, tokens, or URLs introduced. The `.scm` file is benign Tree-sitter syntax. |
| Insecure configuration       | PASS    | No network, no IPC, no exec. The mutex protecting the registry cache is correctly held.|

## Detailed findings

### SEC-001 — Path traversal in bundled-name validation (covered)
**Severity:** INFORMATIONAL (already mitigated)
**Files:** `internal/infrastructure/fsprofile/bundleLoader.go:96`, `internal/infrastructure/fsprofile/bundled.go:35`
**Auto-fixable:** N/A
**Finding:** Both code paths validate the bundled profile name with
`strings.ContainsAny(name, "/\\") || strings.Contains(name, "..")` before
joining it to the embed path. This catches `../etc/passwd` and absolute
paths. The `bundled_test.go:55` test (`b.LoadBundled(ctx, "../../etc/passwd")`)
exercises the rejection. No action required.

### SEC-002 — Registry language id validation
**Severity:** INFORMATIONAL (already mitigated)
**File:** `internal/infrastructure/treesitter/bundledQueries/registry.go:87`
**Auto-fixable:** N/A
**Finding:** `LoadLanguageQueries` rejects ids containing `/`, `\`, or `..`
before calling `fs.ReadDir(bundledFS, dir)`. This is defence-in-depth — even
though `vo.Language` ids are constants, an attacker controlling profile YAML
could in principle pass an attacker-chosen string. The `LanguageUnsupportedError`
return is consistent with the unrecognised-id path; no information leak.

### SEC-003 — Concurrent cache access
**Severity:** INFORMATIONAL (already mitigated)
**File:** `internal/infrastructure/treesitter/bundledQueries/registry.go:31`
**Auto-fixable:** N/A
**Finding:** `Registry.cache` is guarded by `sync.Mutex` held across the
read-and-populate critical section, including the ReadDir/ReadFile calls.
This is heavier than necessary (a sync.Once-per-language pattern would be
cheaper) but not a vulnerability — it cannot deadlock and cannot leak
inconsistent state. WARNING-level performance concern, not a security one.
Tracked in the code-review report.

### SEC-004 — embed.FS read-only guarantee
**Severity:** INFORMATIONAL
**File:** `internal/infrastructure/treesitter/bundledQueries/bundled.go:14`
**Auto-fixable:** N/A
**Finding:** `//go:embed all:java` is a compile-time embed; the resulting
`embed.FS` is read-only by language design. No runtime path can mutate the
bundled queries. The `all:` prefix preserves dotfiles (e.g. `.gitkeep`) — no
security impact, only relevant for empty-tree scenarios.

### SEC-005 — Test fixtures committed
**Severity:** INFORMATIONAL
**Files:** `testdata/ep04us005/profile-{a,b,cobol}/profile.yaml`
**Finding:** Fixtures contain only `name:` and `language:` scalars. No
secrets, no external URLs, no executable content. `testdata/` is gitignored
per repo convention; the `.gitkeep` files preserve the templates/
subdirectory shape.

## Summary

- Total findings: 5 (all INFORMATIONAL)
- Auto-fixable findings requiring remediation: **0**
- BLOCKERs: **0**

**Verdict: CLEAN.**
