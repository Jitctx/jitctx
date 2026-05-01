# Security Report — PC01US-013 Deterministic Violation Output

**Feature:** pc01-us-013-deterministic-violation-output
**Cycle:** 1
**Auditor:** @security-gate (coordinated by qa-coordinator)

## Scope (5 files, all NEW, tests-only ratification)

- `internal/cli/command/deterministicViolationOutputIntegration_test.go`
- `testdata/pc01us013DeterministicOutput/projectFixed/pom.xml`
- `testdata/pc01us013DeterministicOutput/projectFixed/project-state.yaml`
- `testdata/pc01us013DeterministicOutput/projectFixed/.jitctx/profiles/spring-boot-hexagonal.yaml`
- `testdata/pc01us013DeterministicOutput/projectFixed/src/main/java/com/acme/application/usecase/PlaceOrder.java`

## Pillar 1 — Dependency CVEs

No new dependencies introduced. Test imports are all already-vetted internal
packages plus `bytes`, `context`, `io`, `log/slog`, `path/filepath`,
`strings`, `testing`, and the existing `github.com/stretchr/testify/require`.
**No findings.**

## Pillar 2 — Filesystem Safety

`t.TempDir()` is used as the working directory; the test never writes outside
that sandbox. `filepath.Join` is used everywhere, no string concatenation of
paths. Fixtures are read via the established `copyFixture` helper. **No
findings.**

## Pillar 3 — Hardcoded Secrets

Scanned test file and all four fixture files for `password|secret|api_key|
AWS_|PRIVATE_KEY|token=` patterns — zero matches. The fixture pom.xml
declares only the public `org.springframework.boot:spring-boot-starter:3.2.0`
coordinate. **No findings.**

## Pillar 4 — Insecure Configuration

The fixture YAML profile uses `severity: ERROR` (intended), `path_scope`
restricted to `src/main/java/com/acme/application/usecase/`, and no network,
auth, or crypto configuration. **No findings.**

## Verdict

**CLEAN** — zero security findings, zero auto-fixable items.
