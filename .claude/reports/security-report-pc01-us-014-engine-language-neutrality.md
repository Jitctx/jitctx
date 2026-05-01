# Security Audit — PC01US-014 Engine Language Neutrality

**Feature**: pc01-us-014-engine-language-neutrality
**Scope**: 47 files (domain renames, infra DTO/mapper plumbing, application/wire updates, qualitygate exemption, test renames, two new testdata fixtures, one new enforcement test)
**Auditor**: QA Coordinator (security pillar)
**Pillars**: dependency CVEs · filesystem safety · hardcoded secrets · insecure configuration

---

## Summary

| Severity | Count |
|---|---|
| CRITICAL | 0 |
| HIGH | 0 |
| MEDIUM | 0 |
| LOW | 0 |
| INFORMATIONAL | 1 |

**Verdict**: CLEAN. No exploitable findings. One INFORMATIONAL observation about silent error handling in the composition root.

---

## Pillar 1 — Dependency CVEs

`go.mod` and `go.sum` are **unchanged** by this feature. No new dependency surface, no version bumps, no CVE delta. Confirmed via `git diff HEAD go.mod go.sum`.

No findings.

---

## Pillar 2 — Filesystem Safety

Inspected:

- `internal/infrastructure/fsprofile/bundled.go` — `LoadBundled(ctx, name)` validates `name` against path traversal at line 43–45: rejects any `name` containing `/`, `\`, or `..`. The constructed path `"bundled/"+name` is then passed to `fs.Sub(bundledFS, ...)` over an embed-backed FS (no host-FS escape possible). Defence in depth is correct.
- `internal/infrastructure/fsprofile/bundled.go` — the new `DefaultProfileName` constant (`"spring-boot-hexagonal"`) is a hardcoded internal identifier; it has no user-controllable surface.
- `internal/infrastructure/mdspec/parser.go` — backward-compat alias for `"jpa-adapter"` is a static map entry; no dynamic input shapes the lookup.
- `internal/infrastructure/fsprofile/mapper.go` — alias rewrite (`"jpa-adapter"` → `model.ContractPersistenceAdapter`) is gated by exact `==` comparison on `ct == model.ContractType("jpa-adapter")`. No regex, no prefix match, no wildcard. Tight as required.

No findings.

---

## Pillar 3 — Hardcoded Secrets

Grep for `password|secret|api[_-]?key|token\s*=|bearer` across the modified surface returned zero matches. The only string literal of note is `"org.mockito.junit.jupiter.MockitoExtension"` in `bundled/spring-boot-hexagonal/profile.yaml` (a public Java FQN, not a secret).

No findings.

---

## Pillar 4 — Insecure Configuration

### SEC-001 — INFORMATIONAL — Silent failure when bundled profile cannot be loaded at wire time

**File**: `internal/cli/wire.go` lines 147–152
**Auto-fixable**: NO (design decision required)

```go
var scaffoldTestRunnerExtFQN string
if sb, loadErr := bundled.LoadBundled(context.Background(), fsprofile.DefaultProfileName); loadErr == nil {
    scaffoldTestRunnerExtFQN = sb.TestRunnerExtensionFQN
}
```

If `LoadBundled` returns an error (for example, the embedded FS is corrupted, the bundled directory is renamed without updating `DefaultProfileName`, or `domerr.ErrBundledProfileNotFound` fires after a future refactor), `scaffoldTestRunnerExtFQN` remains the empty string and the scaffold renderer silently omits the `@ExtendWith` line from generated tests. The user sees no warning.

This is not a security vulnerability per se — there is no escalation, and downstream code handles `""` cleanly (documented in `usecase.go:40`). It is, however, an **observability gap** with mild integrity implications: a user could run `jitctx scaffold` against a broken bundled FS and silently get under-annotated test classes for an indeterminate window.

**Recommendation** (not auto-applied): log the error at `slog.Warn` level so operators have a trail. A two-line change:

```go
if sb, loadErr := bundled.LoadBundled(context.Background(), fsprofile.DefaultProfileName); loadErr == nil {
    scaffoldTestRunnerExtFQN = sb.TestRunnerExtensionFQN
} else {
    logger.Warn("bundled profile load failed; @ExtendWith will be omitted",
        slog.String("profile", fsprofile.DefaultProfileName),
        slog.Any("err", loadErr))
}
```

Defer to the team — this may be intentional fail-soft behaviour.

---

## Confirmation of QA dispatch concerns

- **Backward-compat alias tightness**: VERIFIED. `internal/infrastructure/mdspec/parser.go:54-55` is a static map; `internal/infrastructure/fsprofile/mapper.go:47-49` uses exact-string equality. Neither uses regex/prefix/wildcard semantics. Adversarial input cannot widen the match.
- **No new secrets, no new dependencies, no new network/disk surface**.

---

## Conclusion

No fixable security findings. Proceed.
