# Security Audit — pc01-us-002-usecase-impl-stereotype

Date: 2026-04-29
Auditor: @security-gate (executed by qa-coordinator)
Feature: PC01US-002 — usecase-impl-stereotype audit rule integration tests

## Scope (9 files, all newly added; nothing modified)

- internal/cli/command/usecaseImplStereotypeIntegration_test.go
- testdata/pc01us002UsecaseImplStereotype/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml
- testdata/pc01us002UsecaseImplStereotype/projectClean/project-state.yaml
- testdata/pc01us002UsecaseImplStereotype/projectClean/pom.xml
- testdata/pc01us002UsecaseImplStereotype/projectClean/src/main/java/com/acme/application/usecase/FindUserUseCaseImpl.java
- testdata/pc01us002UsecaseImplStereotype/projectMissing/.jitctx/profiles/spring-boot-hexagonal.yaml
- testdata/pc01us002UsecaseImplStereotype/projectMissing/project-state.yaml
- testdata/pc01us002UsecaseImplStereotype/projectMissing/pom.xml
- testdata/pc01us002UsecaseImplStereotype/projectMissing/src/main/java/com/acme/application/usecase/FindUserUseCaseImpl.java

## Pillar 1 — Dependency CVEs

No new module dependencies introduced. The integration test imports only
packages already vendored in the repository (`testify/require`, `log/slog`,
`bytes`, `context`, `io`, `path/filepath`, `strings`, `testing`, plus
internal/* packages). `go.mod` was not touched. **No findings.**

## Pillar 2 — Filesystem safety

The test exclusively uses `t.TempDir()` for write paths and resolves fixture
sources through the existing `fixtureDir(t, parts...)` helper, which is
known-safe (rooted at `testdata/` and validated in helpers_test.go). The
manifest path is built with `filepath.Join` against the per-test temp dir.
No untrusted input flows into a path; no symlink-following calls; no
`os.RemoveAll` on caller-controlled directories. **No findings.**

## Pillar 3 — Hardcoded secrets

The four fixture Java/YAML/XML files contain only the literals
`com.acme.*`, `audit-violations`, `0.0.1-SNAPSHOT`, `User not found: ...`,
and Spring Boot parent coordinates `org.springframework.boot:spring-boot-
starter-parent:3.2.0`. None are credentials, tokens, API keys, or PII.
The `pom.xml` Spring Boot version (3.2.0) is a public Maven coordinate and
not consumed at build time — `pom.xml` is only inspected as a marker file
by the framework profile detector (`contains: "org.springframework.boot"`).
**No findings.**

## Pillar 4 — Insecure configuration

The two `spring-boot-hexagonal.yaml` profile fixtures and the two
`project-state.yaml` manifest fixtures declare audit rules and module
classifications with no network endpoints, no credentials, no permissive
glob expansions reaching outside `src/main/java/**`. The audit rule
`usecase-impl-stereotype` is parameterised with `path_scope:
/application/usecase/` and a comma-separated annotation list — no regex
catastrophic-backtracking surface. `t.Parallel()` is used safely because
each test allocates its own `t.TempDir()`. **No findings.**

## Summary

| Severity | Count |
| --- | --- |
| CRITICAL | 0 |
| HIGH | 0 |
| MEDIUM | 0 |
| LOW | 0 |
| INFORMATIONAL | 0 |

No fixable findings. This is a tests-only feature whose engine code (the
`required_annotations` rule kind) was already merged on `main` in commit
`414955c` and was not modified by PC01US-002. The PC01RNF-001 forbidden-
token gate (`internal/qualitygate/`) structurally exempts both `_test.go`
files and `testdata/` paths, so the Java identifiers `Service`,
`RequiredArgsConstructor`, etc., introduced in fixtures do not regress the
gate (verified: `go test ./internal/qualitygate/... -count=1` PASS).

Verdict: **CLEAN**.
