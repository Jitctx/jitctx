# Security Report — pc01-us-003-jpa-entity-contract

**Scope:** 9 files (1 Go test file + 8 testdata fixture files under
`testdata/pc01us003JpaEntityContract/`).

**Verdict:** CLEAN — no findings.

## Pillar 1 — Dependency CVEs

No new dependencies introduced. The Go test file only imports existing
internal packages and `github.com/stretchr/testify/require`, both already
vendored and used across the codebase. The Maven `pom.xml` files inside
testdata are inert fixture artifacts (never built or executed) and exist
solely so `fsprofile`'s `detect.files` predicate can identify the project
as Spring Boot — `spring-boot-starter-parent:3.2.0` is referenced as a
text marker, not consumed.

## Pillar 2 — Filesystem Safety

The integration test uses `t.TempDir()` for both fixtures and
`copyFixture` (a pre-existing helper used by sibling integration tests).
Manifest path is built via `filepath.Join` on a temp directory — no path
traversal vector. No `os.OpenFile` with mode `0666`, no symlink creation,
no archive extraction (zip slip / tar slip not applicable).

## Pillar 3 — Hardcoded Secrets

Grep over the entire scope for password, secret, token, api_key, and
private-key headers returned zero matches. Java fixture contains only
the literal `"orders"` table name — not a credential.

## Pillar 4 — Insecure Configuration

No configuration changes. Profile YAML files reuse the
`spring-boot-hexagonal` schema established by PC01US-002; the new
`audit_rules` entry uses the already-merged `required_annotations` rule
kind (commit 414955c). Severity is `ERROR`, params are well-scoped via
`path_scope: /infrastructure/persistence/entity/`, so the rule cannot
match outside the JPA layer.

## Auto-Fix Summary

No auto-fixable findings. No findings of any severity.
