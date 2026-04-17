# Security Report — ep01us-006 (Tree-sitter Java Integration)

**Date:** 2026-04-17
**Auditor:** @security-gate (inline by @qa-coordinator)
**Scope:**
- internal/cli/command/scanIntegration_test.go
- internal/domain/model/javaFileSummary.go
- internal/infrastructure/treesitter/parser.go
- internal/infrastructure/treesitter/parser_test.go
- internal/infrastructure/treesitter/queries.go
- testdata/springBootMinimal/fixtures/UserWithTableAnnotation.java

---

## Summary

| Severity         | Count |
|------------------|-------|
| CRITICAL         | 0     |
| HIGH             | 0     |
| MEDIUM           | 0     |
| LOW              | 0     |
| INFORMATIONAL    | 2     |

**Verdict:** CLEAN. No auto-fixable findings.

---

## Pillar 1 — Dependency CVEs

No new third-party dependencies introduced in scope. The parser uses the
already-vendored `github.com/smacker/go-tree-sitter` via `sitter.NewParser`
and `parser.ParseCtx` — no version bump in this feature's scope.

No findings.

## Pillar 2 — Filesystem Safety

The parser reads through `fs.FS` (`fs.ReadFile(fsys, path)`), so all path
handling is delegated to the injected filesystem abstraction. Scan uses
`fscontext.New()` which roots the FS at `--path` and the integration test
`TestScanCmd_Integration_ManifestTraversalRejected` confirms path-traversal
rejection on `--manifest ../escape.yaml`. No raw `os.Open`/`filepath.Join`
with user-controlled prefixes in the parser itself.

Test writes use `t.TempDir()` exclusively; no writes outside the per-test
sandbox.

No findings.

## Pillar 3 — Hardcoded Secrets

Grep of scope for likely secret patterns (`password`, `secret`, `token=`,
`api_key`, `BEGIN PRIVATE`): no matches. Test fixtures contain only domain
content (`User`, `email` field name without a value) and a harmless
`@Table(name = "users")` JPA annotation.

No findings.

## Pillar 4 — Insecure Configuration

- Tree-sitter parsing is CPU-bounded and driven by `context.Context` via
  `ParseCtx`, so long-running parses honor cancellation. No resource limits
  are applied, but jitctx is a local CLI invoked on user-owned code —
  threat model does not include adversarial inputs large enough to exhaust
  memory. **INFORMATIONAL** only.
- Test file writes use mode `0o644` and directory mode `0o755` — standard
  and appropriate for temp artifacts.
- The parser does not `panic` on nil trees or malformed nodes; errors are
  wrapped with domain sentinels (`ErrParseFailure`, `ErrPartialParse`).

### SEC-001 (INFORMATIONAL) — No explicit byte-size cap on parser input

- **File:** internal/infrastructure/treesitter/parser.go:30
- **Description:** `fs.ReadFile` reads the full Java source into memory
  before handing it to Tree-sitter. A pathological 1 GB `.java` file would
  be fully resident during parse. Acceptable for the current local-CLI
  threat model.
- **Auto-fixable:** NO
- **Recommendation:** Document the assumption in a follow-up; no action
  required now.

### SEC-002 (INFORMATIONAL) — Recursive `containsErrors` is unbounded

- **File:** internal/infrastructure/treesitter/parser.go:87
- **Description:** `containsErrors` recurses the full AST. Tree-sitter's
  AST depth is bounded by the grammar, but an adversarial input might
  still push it deep. Same threat-model consideration as SEC-001.
- **Auto-fixable:** NO
- **Recommendation:** None for this feature.

---

## Auto-fixable Findings

None.
