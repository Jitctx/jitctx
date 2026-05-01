# Code Review — PC01US-014 Engine Language Neutrality

**Feature**: pc01-us-014-engine-language-neutrality
**Reviewer**: QA Coordinator (code-review pillar)
**Concerns**: architectural conformity · Go idioms & naming · code-smell metrics · test consistency
**Requirements**: `docs/propose-changes-01/quality-gate-evaluators.md` (PC01US-014 line 547, PC01RNF-001 line 173, PC01RF-010 line 145, R-005 line 665)

---

## Summary

| Severity | Count |
|---|---|
| BLOCKER | 1 |
| WARNING | 4 |
| INFO | 2 |

`go test ./...`, `go vet ./...`, `gofmt -l .` — all clean. The `TestEngineLanguageNeutrality_NoFrameworkIdentifiers_PC01US014` enforcement test passes. The `TestJavaReferencesGate` (EP04US-009 quality gate) reports `files_visited=198 exemptions_hit=6 violations=0`. All 10 PC01RNF-003 determinism tests pass.

The single BLOCKER is a build-hygiene issue: five new testdata artefacts that the integration tests depend on are gitignored and untracked, so a clean clone cannot run the suite.

---

## BLOCKERS

### B-001 — New testdata fixtures are gitignored and untracked

**Files**:
- `internal/cli/command/testdata/forbiddenAnnotations.txt` (consumed by `persistenceEntityContractIntegration_test.go:84` and `scaffoldIntegration_test.go:827`)
- `internal/cli/format/testdata/forbiddenAnnotations.txt` (no consumer — see W-002)
- `internal/domain/service/testdata/forbiddenAnnotations.txt` (consumed by `auditRuleEvaluator_test.go:27`)
- `internal/application/usecase/scaffolduc/testdata/forbiddenAnnotations.txt` (no consumer — see W-002)
- `testdata/pc01us004ForbidFieldInjection/**` (consumed by `forbidFieldInjectionIntegration_test.go:82,108,128,149,156`)

**Evidence**:
```
$ git check-ignore -v internal/cli/command/testdata/forbiddenAnnotations.txt
.gitignore:19:testdata    internal/cli/command/testdata/forbiddenAnnotations.txt

$ git status --ignored --short testdata/ | grep ForbidField
!! testdata/pc01us004ForbidFieldInjection/

$ git ls-files testdata/pc01us004ForbidFieldInjection/ | wc -l
0
```

The repository has `testdata` in `.gitignore` (root, line 19) but force-adds the fixtures that tests genuinely depend on — 262 testdata paths are tracked despite the ignore rule. The PC01US-014 fix loop forgot to apply that pattern to the new fixtures, so they exist on the working tree but are absent from version control.

**Impact**: Tests pass on this developer's machine because the directories exist locally. On a clean clone, in CI, or after `git clean -fdx`, three integration tests will fail with `read testdata/forbiddenAnnotations.txt: no such file or directory` and the `forbidFieldInjectionIntegration_test.go` block will fail at `copyFixture`.

**Resolution** (mechanical):
```
git add -f internal/cli/command/testdata/forbiddenAnnotations.txt
git add -f internal/cli/format/testdata/forbiddenAnnotations.txt
git add -f internal/domain/service/testdata/forbiddenAnnotations.txt
git add -f internal/application/usecase/scaffolduc/testdata/forbiddenAnnotations.txt
git add -f testdata/pc01us004ForbidFieldInjection/
```

After staging, re-run `git status` to confirm the fixtures are tracked, then a fresh `go test ./...` to confirm no regression. The two orphan files (W-002) should either be force-added (so the project is self-consistent until tests are written) or deleted; pick one.

---

## WARNINGS

### W-001 — Stale comment in `javaScaffoldConstants.go` claims an exemption was removed

**File**: `internal/application/usecase/scaffolduc/javaScaffoldConstants.go` lines 6–11

The header comment reads:

> PC01US-014 has removed the prior qualitygate exemption for this file:
> every identifier and comment is now language-neutral.

But `internal/qualitygate/exemptions.go:55` still lists this file under `ExemptFiles`, **correctly** — the file still contains `@Entity` (×2) and `@RestController` (×2) string literals, both of which are in `ForbiddenTokens`. The `TestJavaReferencesGate_HonoursExemptions` subtest for `javaScaffoldConstants.go` passes, confirming the exemption is in active use.

The comment is misleading and would deceive a future maintainer who relies on it. Either:

- (a) update the comment to describe the *current* state ("PC01US-014 trimmed the framework-name comments while preserving the @Entity / @RestController string literals; the qualitygate exemption is retained because those literals remain"); or
- (b) if the intent was to remove the exemption, do that work and replace the literals with non-token-bearing forms (the literals are emitted into rendered Java, so this is non-trivial).

(a) is the lower-cost fix and matches what actually shipped.

### W-002 — Orphan testdata files with no test consumer

**Files**:
- `internal/cli/format/testdata/forbiddenAnnotations.txt`
- `internal/application/usecase/scaffolduc/testdata/forbiddenAnnotations.txt`

Both files exist (36 bytes, identical content: `Lombok\nSpring\nMockito\nAutowired\nJPA\n`) but no `*_test.go` in those packages references `forbiddenAnnotations`. Confirmed via `grep -rn "forbiddenAnnotations" internal/cli/format internal/application` (zero matches).

This is dead testdata. Either:

- (a) wire them up to the tests they were intended for (presumably mirrored from `auditRuleEvaluator_test.go`'s `loadForbidden` helper), or
- (b) delete them.

(b) is acceptable until the tests get written. Either decision should be paired with B-001's `git add -f` choice — leaving them on disk *and* ungitted is the worst of both worlds.

### W-003 — Stale `jpa-adapter` references in scaffolduc/usecase.go comments

**File**: `internal/application/usecase/scaffolduc/usecase.go`
- Line 273: `// consistency with service / jpa-adapter (US-001 acceptance scenario`
- Line 280: `// Dependencies: service / rest-adapter / jpa-adapter.`
- Line 338: `// Intentionally non-testable (input-port, output-port, jpa-adapter).`
- Line 550: `// service / jpa-adapter / rest-adapter template, which precedes it with`

The contract type was renamed `ContractJPAAdapter → ContractPersistenceAdapter` and the value `"jpa-adapter" → "persistence-adapter"`. These comments still describe the old vocabulary. They are lowercase `jpa-adapter` strings, so they do **not** trigger the case-sensitive `JPA` token in `TestEngineLanguageNeutrality_NoFrameworkIdentifiers_PC01US014`, but they are stylistically inconsistent with the rest of the rename and confusing to a future reader.

Replace `jpa-adapter` with `persistence-adapter` in those four comments.

### W-004 — Comment on JavaField in `javaFileSummary.go` references `UserRepositoryJpa` example

**File**: `internal/domain/model/javaFileSummary.go` line 46

```go
Type        string   // raw type token as it appears in source, e.g. "UserRepositoryJpa" or "List<User>"
```

`UserRepositoryJpa` is a fictitious type name embedded only in a doc comment, so the case-sensitive `JPA` token does not appear (`Jpa` ≠ `JPA`). The enforcement test correctly does not flag it. Still, the example is now off-message — the engine layer's documentation is signalling a persistence-framework-specific convention while the rest of the work pushes toward neutrality.

Replace with a neutral example, e.g. `"UserRecord"` or `"List<User>"`.

---

## INFOs

### I-001 — Template basename `jpaAdapter` in templateRegistry.go map value

**File**: `internal/infrastructure/fsscaffold/templateRegistry.go` line 90

```go
"persistence-adapter": "jpaAdapter",
```

The template *file* is `internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal/templates/jpaAdapter.tmpl`. Both live in the infrastructure layer, which is **out of scope** for PC01RNF-001's engine-neutrality metric, so this is not a blocker.

That said, it is the last visible thread connecting the contract type to its old framework. A future story could rename `jpaAdapter.tmpl → persistenceAdapter.tmpl`, update this map, update the four template-registry tests that reference `"jpaAdapter"`, and update the bundled profile. Out of scope for PC01US-014; logging it for the backlog.

### I-002 — Two backward-compat aliases for the same legacy literal

The legacy value `"jpa-adapter"` is rewritten to `model.ContractPersistenceAdapter` in **two** places:

- `internal/infrastructure/mdspec/parser.go:54-55` (`knownContractTypes` map entry — for spec authors using the old token in markdown specs)
- `internal/infrastructure/fsprofile/mapper.go:47-49` (`if ct == model.ContractType("jpa-adapter")` — for profile authors using the old `classify_as` value in YAML)

This is intentional per the user's Q-BLOCKING-2 ratification ("YES, both"). Each entry has a `PC01US-014` comment pointing at the other. Both are tight (exact-match, no fuzzy semantics). Acceptable as designed.

When the deprecation window closes (e.g. EP05), both aliases should be deleted in the same PR with an entry in the migration guide. Logging for the backlog.

---

## Architectural conformity

- ISP rigidity preserved — the new `TestRunnerExtensionFQN` field on `model.ProfileBundle` is a value, not a port; no port shape changed.
- No new `infrastructure → application` imports introduced.
- `wire.go` continues to be the single composition root; the new `LoadBundled` call at startup is colocated with the existing wiring (line 150).
- `IDFieldAnnotator` follows the same value-struct + constructor pattern as the other domain services (`ContractPathMapper`, `JavaImportResolver`, etc.).
- Filename casing follows project convention (`idFieldAnnotator.go`, camelCase).

No architectural BLOCKERs.

---

## Test-consistency check (vs requirements)

| Requirement | Status |
|---|---|
| PC01US-014 — engine no longer references a specific persistence framework | PASS — enforcement test green; case-sensitive scan returns zero hits in engine layers (excluding self-exempt test file). |
| PC01RNF-001 — engine language neutrality | PASS — `TestEngineLanguageNeutrality_NoFrameworkIdentifiers_PC01US014` passes; the `forbiddenEngineTokens` slice in the test file is the only place the literals appear in scope; self-exemption via `filepath.Base(p) == selfBasename` works as designed. |
| PC01RF-010 — language-adapter abstraction | PASS — `TestRunnerExtensionFQN` is now sourced from `ProfileBundle` rather than hardcoded in the application layer. |
| R-005 — mitigation for framework leakage | PASS — backward-compat aliases preserve compatibility for legacy specs and profiles per Q-BLOCKING-2 ratification. |
| PC01RNF-003 — determinism | PASS — all 10 determinism tests green (`go test ./... -run "Determini|PC01RNF-003"`). |

---

## Verdict

One BLOCKER (B-001 — untracked testdata fixtures), four WARNINGs (one stale claim in a comment, one orphan testdata pair, two stale `jpa-adapter` mentions in comments), two INFO items for the backlog. The Go code itself is clean — `go test`, `go vet`, `gofmt`, the engine-neutrality enforcement test, the `TestJavaReferencesGate` quality gate, and the determinism tests are all green.

Proceed to fix loop for B-001 (and W-001 / W-002 / W-003 / W-004 if cycle budget allows).
