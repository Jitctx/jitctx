# Code Review Report — ep01us-005 (Spring Boot Hexagonal Profile)

**Reviewer:** @code-reviewer (executed inline by QA Coordinator)
**Date:** 2026-04-17
**Requirements:** /workspaces/jitctx/docs/ep01/requirements.md
**Feature file:** docs/ep01/ep01.feature lines 168-212
**Scope:**
- internal/application/usecase/scanuc/usecase.go (modified)
- internal/cli/command/scanIntegration_test.go (modified)
- internal/domain/model/frameworkProfile.go (modified)
- internal/domain/service/profileClassifier_test.go (modified)
- internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal.yaml (modified)
- internal/infrastructure/fsprofile/detector.go (modified)
- internal/infrastructure/fsprofile/detector_test.go (modified)
- internal/infrastructure/fsprofile/loader.go (modified)
- testdata/springBootMinimal/expected/project-state.yaml (modified)
- testdata/springBootMinimal/project/src/main/java/com/app/user_management/service/CreateUserServiceImpl.java (added)

## Architectural Conformity

- **Domain purity (CLAUDE.md "Sem tags YAML/JSON no domínio").**
  `frameworkProfile.go` has zero struct tags — serialization lives in
  `fsprofile` DTOs. New `ProfileSource` enum (bundled/custom) is a pure
  domain type; its values are set by infrastructure adapters, never
  mentioned in the YAML contract. COMPLIANT.
- **ISP on ports.** `DetectProfilePort` (single method `Detect`) is
  satisfied by `fsprofile.Detector`; `LoadProfilePort` / `ListProfilesPort`
  by `fsprofile.Loader`. No port merges methods. COMPLIANT.
- **Infrastructure isolation.** `internal/application/usecase/scanuc`
  imports only domain packages (`domain/model`, `domain/port/...`,
  `domain/service`, `domain/vo`, `domain/errors`) and stdlib. No
  infrastructure import. COMPLIANT.
- **Composition-root manual wiring.** Integration test
  `buildScanFactoryWithLogger` manually injects all adapters (no DI
  container, no reflection). COMPLIANT.
- **context.Context first parameter** on every port call
  (`Detect(ctx, fsys)`, `Load(ctx, name)`, `List(ctx)`, use-case
  `Execute(ctx, input)`). COMPLIANT.
- **Errors over panic.** Sentinels (`domerr.ErrNoProfileMatch`,
  `domerr.ErrProfileInvalid`, `domerr.ErrPartialParse`,
  `domerr.ErrManifestWrite`) wrapped via `fmt.Errorf("…: %w", err)`.
  No panics introduced. COMPLIANT.
- **stdout vs stderr.** Use case logs only via `slog` (stderr); scan
  command stdout is limited to the summary line. Integration tests
  split the two via separate buffers. COMPLIANT.
- **Classification rule ordering (EP01RF-012 §Business Rule).** YAML
  rules are ordered first-match-wins with @Entity before @Service and
  path-based service matching before @Service catch-all. Rule order is
  locked by `TestClassifyDeclaration_BundledProfile`. COMPLIANT.

## Go Idioms & Naming

- **Filenames camelCase** — `frameworkProfile.go`, `profileClassifier_test.go`,
  `scanIntegration_test.go`, `detector.go`, `detector_test.go`, `loader.go`.
  COMPLIANT with CLAUDE.md.
- **Package names** — `fsprofile`, `scanuc`, `service`, `model` — short,
  lowercase, no underscores. COMPLIANT.
- **No `Impl` suffix at domain port level.** Implementations named
  `Detector`, `Loader`, `Impl` (the latter in `scanuc` per existing
  convention from prior cycles). COMPLIANT.
- **Constructors** — `NewDetector`, `NewDetectorWithLogger`, `New`,
  `NewWithLogger`. Pair mirrors the pattern used for `fsmanifest`.
  COMPLIANT.
- **Embedded fs** — `//go:embed bundled/*.yaml` is declared once in
  `loader.go` and reused by `detector.go` via the package-level
  `embeddedProfiles` variable. Idiomatic.
- **Error wrapping** — every error leaving a boundary is annotated
  (`"walk java: %w"`, `"parse %s: %w"`, `"discover contexts: %w"`,
  `"profile %q not found: %w"`). COMPLIANT.

## Code-Smell Metrics

- `scanuc.Impl.Execute` — 128 lines, cyclomatic ~11. On the higher end
  but acceptable for a use-case orchestrator; each step is commented
  with its contract §5.4 number. No duplicated blocks. Not a blocker.
- `Detector.loadCustomProfiles` — 32 lines, cyclomatic 6. OK.
- `Detector.loadBundledProfiles` — 24 lines, cyclomatic 5. OK.
- `Loader.Load` — 44 lines, cyclomatic 8. Guard logic (traversal
  rejection + userDir try + bundled fallback) justifies the branching.
  OK.
- `decodeProfile` — 8 lines, cyclomatic 2. OK.
- `profileMatches` — 11 lines, cyclomatic 3. OK.
- Integration test file — 10 scenarios, each self-contained. Some
  duplication in factory setup (`buildScanFactoryWithLogger` +
  manifestPath boilerplate), but already extracted to the helper. OK.
- Bundled YAML — 82 lines with explicit scenario-to-line comments.
  Readable and self-documenting.

## Test Consistency

- **Feature line 171-175** — `TestScanCmd_Integration_ProfileLog`
  asserts the exact substring `"Profile: spring-boot-hexagonal"` in the
  log buffer and the `source=bundled` structured attribute. COVERED.
- **Feature line 177-180** — `TestScanCmd_Integration_AutoDetectGradle`
  and `TestScanCmd_Integration_AutoDetectGradleKts` cover both Groovy
  and Kotlin DSL detection. COVERED.
- **Feature line 182-185** (input port) — `TestClassifyDeclaration`
  subtest `"input port by path and node type"`. COVERED.
- **Feature line 187-190** (output port) — `TestClassifyDeclaration`
  subtest `"output port by path"`. COVERED.
- **Feature line 192-195** (entity) — `TestClassifyDeclaration`
  subtest `"entity by annotation"`. COVERED.
- **Feature line 197-200** (REST adapter) — `TestClassifyDeclaration`
  subtest `"rest adapter by annotation"`. COVERED.
- **Feature line 202-206** (service by UseCase impl) —
  `TestClassifyDeclaration` subtest `"service by implements and path"`
  and integration `TestScanCmd_Integration_ServiceByImplementsUseCase`
  which asserts `CreateUserServiceImpl` is classified as `type: service`
  in the manifest. COVERED.
- **Feature line 208-212** (custom overrides bundled) —
  `TestDetector_CustomOverridesBundled` (unit) and
  `TestScanCmd_Integration_CustomOverrideAutoDetect` (integration).
  COVERED.
- **EP01RF-012 §Business Rule** (@Repository → jpa-adapter, not in
  Gherkin) — `TestClassifyDeclaration_BundledProfile` sub-case pins
  this rule. COVERED and locks the bundled YAML to the requirement.
- **Profile-source provenance** — `TestDetector_BundledSourceStamp` and
  `TestDetector_CustomSourceStamp` lock the `ProfileSource` field.
  COVERED.
- **Malformed custom profile fallback** (EP01RF-012 §Exceptions) —
  `TestScanCmd_Integration_MalformedCustomProfileFallback` asserts warn
  log and fallback to bundled. COVERED.
- All tests use `t.Parallel()` and `require` (fail-fast). COMPLIANT
  with unit-test-layer-guidelines.
- Fixture-based integration tests use `stripGeneratedAt` +
  `normalizeYAML` to keep assertions deterministic. COMPLIANT.

## Findings

### W-001 (WARNING) — `scanuc.Impl.Execute` drops the underlying Save error

**File:** internal/application/usecase/scanuc/usecase.go:169-171
```go
if err := u.manifest.Save(ctx, state); err != nil {
    return scanvo.ScanProjectOutput{}, fmt.Errorf("save manifest: %w", domerr.ErrManifestWrite)
}
```
The concrete error returned by `u.manifest.Save` is swallowed in favour
of the sentinel `ErrManifestWrite`. While this preserves the typed
error for caller classification, it loses diagnostic detail (syscall
string, path, etc.) that would help debugging manifest-write failures.

**Suggested remediation (non-blocking):** wrap both — e.g.
`fmt.Errorf("save manifest: %w: %v", domerr.ErrManifestWrite, err)`
so `errors.Is` still works while the underlying cause reaches logs.
Not required for this cycle; pattern is unchanged from earlier US-001
cycles.

### W-002 (WARNING) — Lingering design-decision comment in usecase.go

**File:** internal/application/usecase/scanuc/usecase.go:156
```go
// Derive framework name: strip "-hexagonal" suffix for clarity? No — use prof.Name as-is per plan.
```
This reads like a review-time note rather than durable documentation.
The resolved decision ("use prof.Name as-is") belongs in the plan/ADR;
the code comment should explain *what* the line does, not *why the
author chose not to do something else*.

**Suggested remediation (non-blocking):** remove or rephrase as
`// Frameworks list mirrors the detected profile name verbatim; renaming
happens at the profile layer, not here.`

### I-001 (INFO) — `CreateUserServiceImpl.java` fixture references `User` without import

**File:** testdata/springBootMinimal/project/src/main/java/com/app/user_management/service/CreateUserServiceImpl.java
```java
public class CreateUserServiceImpl implements CreateUserUseCase {
    public User execute(String name, String email) { return null; }
}
```
`User` lives in `com.app.user_management.domain` but is referenced
without an `import com.app.user_management.domain.User;`. Tree-sitter
parses the file fine (syntax-only), and the integration test only
cares about the AST-level classification, so this does not affect
behaviour. Real Java compilation would fail, which is acceptable for a
parser fixture.

**No action required.** If this fixture ever powers a compile-level
test the import must be added.

### I-002 (INFO) — Detector silently swallows non-ENOENT `ReadDir` errors

**File:** internal/infrastructure/fsprofile/detector.go:59-62
```go
entries, err := os.ReadDir(d.userDir)
if err != nil {
    return nil // directory may not exist
}
```
All errors from `os.ReadDir` are coerced to "no custom profiles". ENOENT
is expected and correct, but EACCES / EIO surface as the same no-op.
Low priority because the caller has no actionable recovery, but worth
a warn-log for operator visibility.

**No action required** for this cycle.

## Summary

| Severity | Count |
|----------|-------|
| BLOCKER  | 0 |
| WARNING  | 2 |
| INFO     | 2 |

**Verdict: PASS WITH WARNINGS** (no BLOCKERs; WARNINGs are advisory).
