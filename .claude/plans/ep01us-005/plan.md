# Plan — EP01US-005: Spring Boot Hexagonal Profile

## Section 0 — Summary

- Feature: formalize the `spring-boot-hexagonal` profile as a declarative
  YAML asset bundled into the binary, and harden the profile-detection /
  profile-loading logic that already feeds `jitctx scan`. Acceptance
  focuses on auto-detection (Maven + Gradle + Gradle Kotlin DSL),
  path-based classification (`port/in/`, `port/out/`), annotation-based
  classification (`@Entity`, `@RestController`, `@Repository`, `@Service`),
  the `implements *UseCase` + `service|application` path heuristic, and
  **custom-over-bundled** precedence when a profile in
  `.jitctx/profiles/` also matches.
- Requirement IDs: **EP01US-005**, **EP01RF-012**; carries partial
  responsibility for **EP01RNF-002** (deterministic output — profile
  selection must be deterministic under ties) and **EP01RNF-004** (single
  binary — bundled profile must be `go:embed`-ed, not read from disk at
  runtime).
- Layers touched: **domain** (one tiny port addition, one typed error),
  **infrastructure** (`fsprofile` hardening + one new YAML rule and
  clearer iteration), **application** (one log-format correction in
  `scanuc`), **tests** (unit + integration coverage for every Gherkin
  scenario on feature lines 168-212). Presentation, wire, and composition
  root are **unchanged**.
- Tiers active: **1, 2, 3, 6** (Tier 4 collapsed — no cobra command /
  formatter change; Tier 5 collapsed — `wire.go`, `root.go`, `execute.go`,
  `main.go`, and `internal/config/**` are untouched because both
  `DetectProfilePort` and `LoadProfilePort` are already wired via
  `fsprofile.NewDetectorWithLogger` and `fsprofile.New`).
- Guidelines loaded:
  - `.claude/guidelines/domain-layer-guidelines.yml`
  - `.claude/guidelines/infrastructure-layer-guidelines.yml`
  - `.claude/guidelines/application-layer-guidelines.yml`
  - `.claude/guidelines/unit-test-layer-guidelines.yml`
  - `.claude/guidelines/integration-test-layer-guidelines.yml`
- Estimated file count: **1 new, 7 modified** (8 files total, plus 2
  testdata fixtures).

### What prior user stories already give us and must NOT be duplicated

Verified by reading the live source on `main` (commits merged up to and
including `3d64613` / EP01US-001-004):

| Capability | Location | Reuse status |
|---|---|---|
| `model.FrameworkProfile` with `Detect`, `ModuleDetection`, `Rules`, `Languages`, `QueryLang` | `internal/domain/model/frameworkProfile.go` | **Reused verbatim.** Schema fields cover every classification kind required by EP01US-005 (node_type + path_contains + has_annotation + implements). |
| `port/profile/{detectProfilePort,loadProfilePort,listProfilesPort}.go` ISP interfaces | `internal/domain/port/profile/` | **Reused verbatim** for `LoadProfilePort` and `ListProfilesPort`. `DetectProfilePort.Detect` also reused — but a tiny typed error addition lets callers tell *how* a profile matched (bundled vs custom), needed by the "custom overrides bundled" scenario log. |
| `fsprofile.Detector` / `fsprofile.Loader` with `//go:embed bundled/*.yaml`, `KnownFields(true)`, path-traversal rejection | `internal/infrastructure/fsprofile/` | **Reused.** Modified to: (a) guarantee deterministic iteration across both custom and bundled directories, (b) emit a `Source` (`"custom"` / `"bundled"`) on the returned profile so callers can log the provenance the Gherkin asks for. |
| `service.ClassifyDeclaration` with `NodeType | PathContains | HasAnnotation | Implements` matching and `*Glob` wildcard on `Implements` | `internal/domain/service/profileClassifier.go` | **Reused verbatim.** All six Gherkin classification scenarios on lines 182-206 are already satisfied by the existing matcher; this story only extends **test coverage** for the rows that have no explicit unit or integration test today. |
| `service.BuildModules` classifies + groups into modules via the profile | `internal/domain/service/moduleBuilder.go` | **Reused verbatim.** No change. |
| `bundled/spring-boot-hexagonal.yaml` with detect rules for `pom.xml`, `build.gradle`, `build.gradle.kts` and nine classification rules | `internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal.yaml` | **Modified** — reorganized so `implements: "*UseCase"` rules appear *before* the catch-all `has_annotation: Service` rule (first-match-wins semantics; see §4.1). Same rule surface, explicit comments tying each rule to a Gherkin line. |
| `scanuc.Impl` which drives the full scan (detect → walk → parse → classify → build modules → contexts → manifest) | `internal/application/usecase/scanuc/usecase.go` | **Modified** — the info log is currently `u.logger.Info("profile selected", "profile", prof.Name)` which slog renders as `msg="profile selected" profile=spring-boot-hexagonal`. Feature line 175 requires `"Profile: spring-boot-hexagonal"` *as a substring of the scan log*. Change the log message to `Profile: %s` using a formatted value so the assertion holds verbatim, regardless of slog handler (`TextHandler` or `JSONHandler`). |
| Integration test `TestScanCmd_Integration_ProfileLog` asserting `"profile=spring-boot-hexagonal"` | `internal/cli/command/scanIntegration_test.go` lines 230-253 | **Modified** — rewrite the assertion to match the new Gherkin-literal substring `"Profile: spring-boot-hexagonal"`. The old assertion was a slog-handler artefact, not a spec requirement. |
| Integration tests for custom override (`TestScanCmd_Integration_CustomProfileOverride`) and gradle auto-detect (`TestScanCmd_Integration_AutoDetectGradle`) | `internal/cli/command/scanIntegration_test.go` | **Reused** as regression coverage. Two *new* integration tests are added for the scenarios not yet exercised: (a) `build.gradle.kts` auto-detect (feature line 177 says "build.gradle" which already has a test — we extend coverage to the Kotlin DSL which the bundled profile also claims to support), and (b) the **tie-breaking** case where both a bundled and a custom profile match without `--profile` being passed (feature lines 208-212). |

### In-scope behaviour (delta on top of EP01US-001)

1. **Precedence contract is locked.** When both a custom profile in
   `.jitctx/profiles/*.yaml` and a bundled profile match the project,
   the custom one wins — deterministically. Today the detector already
   loops custom-first, then bundled, and `os.ReadDir`/`fs.ReadDir`
   return sorted entries, but this is an undocumented implementation
   detail that future maintainers might regress. Promote it to contract:
   (a) add `profileSource` as an enum-typed field on the returned
   profile, (b) sort inside the detector explicitly (so the order does
   not silently change if Go stdlib changes its mind), (c) cover with a
   dedicated integration test.

2. **Profile selection log format is aligned with the Gherkin.** The
   scan log must contain the exact substring `"Profile: spring-boot-hexagonal"`.
   `scanuc.Impl.Execute` is the only place this log line is emitted; the
   call becomes
   `u.logger.Info(fmt.Sprintf("Profile: %s", prof.Name), "source", prof.Source)`.
   `Source` (`"bundled"` | `"custom"`) is a structured attribute; the
   human-readable message carries the profile name with the
   capital-P colon format the spec mandates.

3. **Bundled profile rule order is stabilised.** Currently the bundled
   YAML has the `@Service` annotation rule *before* the two
   `implements: "*UseCase"` rules. Because the classifier is
   first-match-wins, a class annotated `@Service` that also implements
   `CreateUserUseCase` and lives under `/service/` is classified
   `service` by either rule — net effect identical today. But the
   ordering matters the moment a user extends the bundled profile. The
   feature scenarios enumerate the annotation rules before the
   implements rule (feature lines 192-206), so we mirror that order in
   the bundled YAML and add explanatory comments mapping each rule to
   its scenario line. No semantic change for today's fixtures.

4. **Malformed-custom-profile fallback is retained, but now verified.**
   Feature line 208-212 covers the happy path (custom overrides
   bundled). The requirements text EP01RF-012 §*Exceptions* adds: "If a
   custom profile in `.jitctx/profiles/` has syntax errors, fall back to
   the bundled profile and log a warning." The detector already logs
   the parse error and skips the file — but there is no integration
   test for the *detector* path (there is one for the loader). A new
   integration test asserts that scan still succeeds with a broken
   custom profile and still emits `Profile: spring-boot-hexagonal`.

5. **`ProfileSource` becomes part of the domain contract.** A new
   `type ProfileSource string` with constants `ProfileSourceBundled`
   and `ProfileSourceCustom` is introduced on `model.FrameworkProfile`.
   This is one new field on an existing struct, not a new model file.

### Out of scope (deferred)

- No new cobra command (EP01US-005 only runs through the existing
  `jitctx scan`).
- No new formatter output (stdout stays the existing "scanned:" summary;
  the feature only references *stderr logs*, not stdout).
- No change to `ListProfilesPort` — already used only by the (still
  stubbed) `jitctx list` command which is out of scope for EP01.
- No change to `module_detection` strategies beyond `hexagonal`
  (explicitly stated as out-of-scope in EP01 §1.5).
- No extra Gradle flavours (`.kts` is already in the bundled YAML; we
  add test coverage, not new rules).
- No change to the `DetectProfilePort.Detect` signature; `Source` is
  filled in *before* the profile leaves the detector, so existing
  consumers (scanuc) keep compiling.
- No changes to the YAML DTO schema on the wire — `profileDTO` stays;
  the Source enum is computed by the infrastructure adapter at load
  time, not declared in the YAML file.

---

## Section 1 — File Set

| # | File | Action | Layer | Tier | Group |
|---|------|--------|-------|------|-------|
| 1 | `internal/domain/model/frameworkProfile.go` | modify | domain | 1 | T1-G1 |
| 2 | `internal/infrastructure/fsprofile/detector.go` | modify | infra | 2 | T2-G1 |
| 3 | `internal/infrastructure/fsprofile/loader.go` | modify | infra | 2 | T2-G1 |
| 4 | `internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal.yaml` | modify | infra | 2 | T2-G1 |
| 5 | `internal/application/usecase/scanuc/usecase.go` | modify | app | 3 | T3-G1 |
| 6 | `internal/infrastructure/fsprofile/detector_test.go` | modify | tests | 6 | T6-G1 |
| 7 | `internal/domain/service/profileClassifier_test.go` | modify | tests | 6 | T6-G2 |
| 8 | `internal/cli/command/scanIntegration_test.go` | modify | tests | 6 | T6-G3 |
| 9 | `testdata/springBootMinimal/project/src/main/java/com/app/usermanagement/service/CreateUserServiceImpl.java` | create | tests | 6 | T6-G3 |
| 10 | `testdata/springBootMinimal/expected/project-state.yaml` | modify | tests | 6 | T6-G3 |

Requirement coverage:

- **EP01US-005 / EP01RF-012** — files 1-10 (end-to-end).
- **EP01RNF-002 (determinism under profile-tie)** — files 2 (sorted
  iteration), 8 (integration regression).
- **EP01RNF-004 (single binary)** — files 3, 4 (bundled YAML stays
  `go:embed`-ed; loader never reads a runtime-installed profile file).

---

## Section 2 — Frozen Domain Contract

Everything below is the **verbatim Go shape** that Tier 2 and Tier 3
must consume without modification. If implementation discovers that a
signature needs to change, stop and escalate — do not silently drift.

### 2.1 `model.FrameworkProfile` (field addition only)

```go
// internal/domain/model/frameworkProfile.go
package model

// ProfileSource identifies where a profile was loaded from. Populated
// by the infrastructure adapter at load/detect time; the domain never
// constructs ProfileSource values itself.
type ProfileSource string

const (
    ProfileSourceBundled ProfileSource = "bundled"
    ProfileSourceCustom  ProfileSource = "custom"
)

type FrameworkProfile struct {
    Name            string
    Source          ProfileSource // NEW — populated by fsprofile; zero value means "unknown" and is only valid in tests
    Detect          ProfileDetect
    ModuleDetection ModuleDetection
    Rules           []ProfileRule
    QueryLang       string
    Languages       []string
}

// (ProfileDetect, ProfileFileMatcher, ModuleDetection, ModuleMarker,
//  ProfileRule, ProfileMatch — UNCHANGED from main.)
```

The zero value (empty string) is intentionally accepted so the many
existing unit-test fixtures (e.g. in `profileClassifier_test.go`) that
build `&model.FrameworkProfile{Rules: ...}` inline continue to compile
without change. No validation is added on `Source` in the model
constructor because there is no `NewFrameworkProfile` constructor today
and we are not introducing one for this story.

### 2.2 Ports — no interface changes

No new port files. No method added to `DetectProfilePort`,
`LoadProfilePort`, or `ListProfilesPort`. The `Detect` and `Load`
implementations fill in `Source` on the returned `*model.FrameworkProfile`
before returning to the caller. Signature for reference:

```go
// internal/domain/port/profile/detectProfilePort.go — UNCHANGED
type DetectProfilePort interface {
    Detect(ctx context.Context, fsys fs.FS) (*model.FrameworkProfile, error)
}

// internal/domain/port/profile/loadProfilePort.go — UNCHANGED
type LoadProfilePort interface {
    Load(ctx context.Context, name string) (*model.FrameworkProfile, error)
}

// internal/domain/port/profile/listProfilesPort.go — UNCHANGED
type ListProfilesPort interface {
    List(ctx context.Context) ([]string, error)
}
```

### 2.3 Use-case signatures — no changes

```go
// internal/domain/usecase/scanuc/port.go — UNCHANGED
type UseCase interface {
    Execute(ctx context.Context, input scanvo.ScanProjectInput) (scanvo.ScanProjectOutput, error)
}
```

`scanvo.ScanProjectInput` / `ScanProjectOutput` are unchanged — the
log-format correction happens entirely inside `scanuc.Impl.Execute`
without new fields.

### 2.4 Errors — no new sentinels

`domerr.ErrNoProfileMatch` and `domerr.ErrProfileInvalid` already cover
every failure mode EP01US-005 describes. No new sentinel, no new typed
error. `internal/domain/errors/errors.go` is **not** in the file set.

### 2.5 `Deps` struct — unchanged

```go
// internal/cli/wire.go — UNCHANGED; included here for completeness
type Deps struct {
    ScanFactory command.ScanUseCaseFactory
    Query       queryuc.UseCase
    Plan        planuc.UseCase
    Contracts   contractsuc.UseCase
    Logger      *slog.Logger
}
```

### 2.6 Profile YAML schema (frozen)

The on-disk / embedded schema for a profile YAML file is the one
already enforced by `fsprofile.profileDTO` with `KnownFields(true)`.
EP01US-005 introduces **zero schema changes**. The bundled
`spring-boot-hexagonal.yaml` stays within that schema; the
`spring-boot-hexagonal` profile after this story looks like:

```yaml
# internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal.yaml
name: spring-boot-hexagonal
languages: [java]
query_lang: java

# Auto-detection — feature lines 171-181.
detect:
  files:
    - name: pom.xml                    # Maven     (line 172)
      contains: "org.springframework.boot"
    - name: build.gradle               # Gradle    (line 178)
      contains: "org.springframework.boot"
    - name: build.gradle.kts           # Gradle KTS (inferred from same)
      contains: "org.springframework.boot"

module_detection:
  strategy: hexagonal
  roots:
    - src/main/java/**/domain
    - src/main/java/**
  markers:
    - kind: path_contains
      value: /port/in/
    - kind: path_contains
      value: /port/out/
    - kind: annotation
      value: Entity

# Classification — order matters (first match wins). Each rule carries
# an inline comment pointing back to the Gherkin scenario it satisfies.
rules:
  # Scenario: Classify input port by path convention (line 182)
  - match:
      node_type: interface_declaration
      path_contains: /port/in/
    classify_as: input-port

  # Scenario: Classify output port by path convention (line 187)
  - match:
      node_type: interface_declaration
      path_contains: /port/out/
    classify_as: output-port

  # Scenario: Classify entity by @Entity annotation (line 192)
  - match:
      node_type: class_declaration
      has_annotation: Entity
    classify_as: entity

  # Scenario: Classify REST adapter by @RestController annotation (line 197)
  - match:
      node_type: class_declaration
      has_annotation: RestController
    classify_as: rest-adapter

  # @Repository is not in the Gherkin but is required by EP01RF-012 §Business Rule.
  - match:
      node_type: class_declaration
      has_annotation: Repository
    classify_as: jpa-adapter

  # Scenario: Classify service by UseCase implementation (line 202)
  # Implements rules FIRST so that a class matching both *UseCase and @Service
  # stays deterministic if the user extends classify_as later.
  - match:
      node_type: class_declaration
      implements: "*UseCase"
      path_contains: /service/
    classify_as: service

  - match:
      node_type: class_declaration
      implements: "*UseCase"
      path_contains: /application/
    classify_as: service

  # @Service catch-all (covers classes that don't implement *UseCase but are
  # annotated). Not asserted by any Gherkin — retained from the merged MVP.
  - match:
      node_type: class_declaration
      has_annotation: Service
    classify_as: service
```

### 2.7 Classification rule shape (frozen)

Each rule evaluates as an AND across the non-empty fields on its
`match` block; evaluation order across rules is the YAML array order
(first match wins). This is already how `service.ClassifyDeclaration`
behaves; this story does not change the algorithm, only the rule order
in the bundled YAML and the test coverage.

### 2.8 Detection port contract (frozen)

- `Detect(ctx, fsys)` iterates **custom profiles in
  `<userDir>` sorted by filename ascending, THEN bundled profiles
  sorted by filename ascending**. The first profile whose `Detect.Files`
  matches the target `fsys` (case-insensitive substring match on the
  file content) wins.
- The returned `*model.FrameworkProfile` has `Source =
  model.ProfileSourceCustom` when it came from `<userDir>`, else
  `model.ProfileSourceBundled`.
- On zero matches, returns `(nil, domerr.ErrNoProfileMatch)`.

---

## Section 3 — Domain Layer Plan

**Tier 1 — one group only (mandated: domain is always one group).**

### 3.1 `internal/domain/model/frameworkProfile.go` — modify

- Add the `ProfileSource` named-string type and its two constants
  (`ProfileSourceBundled`, `ProfileSourceCustom`) at the top of the
  file, above the existing `FrameworkProfile` struct.
- Add a single field `Source ProfileSource` to `FrameworkProfile`.
  Placement: immediately after `Name` to keep identification fields
  together. No struct tags, no validation function (per §2.1 rationale).
- Do NOT touch `ProfileDetect`, `ProfileFileMatcher`, `ModuleDetection`,
  `ModuleMarker`, `ProfileRule`, `ProfileMatch`. Do NOT rename the
  file. Do NOT introduce a constructor.

No other domain file changes. `internal/domain/errors/errors.go`,
`internal/domain/service/profileClassifier.go`, and
`internal/domain/port/profile/*.go` stay untouched.

---

## Section 4 — Infrastructure Layer Plan

**Tier 2 — one group** (`T2-G1`). All three infra files live under
`internal/infrastructure/fsprofile/` — per the Step-6 rule "one group
per `infrastructure/{collaborator}/` subdirectory". The Detector, the
Loader, and the bundled YAML form one coherent collaborator and are
edited as a unit.

### 4.1 `internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal.yaml` — modify

See §2.6 for the exact target content. Net diff: (a) reorder rules so
`implements: "*UseCase"` blocks come before `has_annotation: Service`,
(b) add comments mapping each rule to a Gherkin scenario line, (c) no
rule additions, deletions, or changes in `classify_as`. The YAML stays
in-tree under `bundled/` and keeps being picked up by the existing
`//go:embed bundled/*.yaml` directive.

### 4.2 `internal/infrastructure/fsprofile/detector.go` — modify

1. **Stamp `Source` on every returned profile.**
   - In `loadCustomProfiles`, after `decodeProfile` succeeds, set
     `prof.Source = model.ProfileSourceCustom`.
   - In `loadBundledProfiles`, same thing with
     `model.ProfileSourceBundled`.
2. **Make iteration order explicit.** `os.ReadDir` already returns
   sorted entries on every supported OS, and `fs.ReadDir` on an
   `embed.FS` also sorts — but the current code relies on it
   implicitly. Add an explicit `sort.Slice(entries, func(i, j int)
   bool { return entries[i].Name() < entries[j].Name() })` in both
   loaders (no behaviour change — ensures a future Go or Windows
   regression cannot break the precedence contract).
3. **Keep the custom-first / bundled-second loop** in
   `Detect`. No API change on the port.
4. **Preserve existing path-traversal rejection and malformed-YAML
   fallback** (a custom profile that fails `decodeProfile` is skipped,
   warning logged via `d.logger.Warn`).
5. **No change to `profileMatches`.** Case-insensitive substring match
   is already correct for feature lines 172 and 178.

### 4.3 `internal/infrastructure/fsprofile/loader.go` — modify

Mirror the Source stamping so the `Load` path (used by the `jitctx
scan --profile <name>` flag) agrees with the `Detect` path.

1. When the user-dir path wins, set `prof.Source =
   model.ProfileSourceCustom` on the returned value.
2. When falling through to `embeddedProfiles`, set `prof.Source =
   model.ProfileSourceBundled`.
3. Keep `List` untouched (it only returns names, no provenance).
4. Keep `decodeProfile(data, true)` strict mode untouched
   (`KnownFields(true)` remains the first line of defence against
   typos in user profiles, per infrastructure-layer guideline 4.x).

No change to `dto.go` or `mapper.go` — `Source` is set *after* the
mapper runs, not plumbed through the YAML DTO.

### 4.4 DTO and atomic-write checkpoints

- `profileDTO` (`dto.go`) is unchanged; the YAML wire format is
  untouched.
- The loader does not write files (profiles are read-only), so the
  "temp file + os.Rename" pattern from `fsmanifest` does not apply.
- `embed.FS` is the single source of truth for bundled profiles, per
  `EP01RNF-004` (no runtime file system lookups for bundled content).

---

## Section 5 — Application Layer Plan

**Tier 3 — one group** (`T3-G1`). Only `scanuc.Impl.Execute` changes.

### 5.1 `internal/application/usecase/scanuc/usecase.go` — modify

Single-line change in the body of `Execute` (current line 87):

```go
// BEFORE
u.logger.Info("profile selected", "profile", prof.Name)

// AFTER
u.logger.Info(
    fmt.Sprintf("Profile: %s", prof.Name),
    "source", string(prof.Source),
)
```

Rationale:

- `fmt.Sprintf("Profile: %s", prof.Name)` puts the exact substring
  `"Profile: spring-boot-hexagonal"` into the log message body,
  regardless of whether the handler is `slog.TextHandler` (which would
  produce `msg="Profile: spring-boot-hexagonal"`) or
  `slog.JSONHandler` (which would produce
  `"msg":"Profile: spring-boot-hexagonal"`). Both contain the asserted
  substring.
- `source` stays as a structured attribute so operators grepping for
  `source=custom` / `source=bundled` still work.
- No change to the `Port` / `UseCase` signature, no change to
  `ScanProjectInput` / `ScanProjectOutput`.

Nothing else in `scanuc` changes: the `Detect(ctx, fsys)` call, the
`input.ProfileName` guard, the walking / classifying / module-building
/ context-discovery / manifest-write steps all stay as-is. No new
`fmt` import is required — `fmt` is already imported on line 5 of the
current file (used by the existing `fmt.Errorf` calls).

### 5.2 Error-wrapping policy

Unchanged from today. `Detect` failures still bubble up as
`domerr.ErrNoProfileMatch`, translated to the user message
`"no matching framework profile found; create a custom profile..."`
by `internal/cli/format/errors.go::TranslateError`.

---

## Section 6 — Presentation Layer Plan

**Tier 4 — N/A.** No cobra command file, no formatter file, no flag
change. `scanCmd.go` already accepts `--profile <name>` and passes it
to `ScanProjectInput.ProfileName` which the use case already validates
against the detected profile. EP01US-005 does not change the CLI
surface.

### 6.1 stdout / stderr contract

- stdout: still the existing "scanned: X modules, Y contexts" summary
  printed by `format.WriteScanReport`. No change.
- stderr (slog): the profile selection log line changes from
  `msg="profile selected" profile=spring-boot-hexagonal` to
  `msg="Profile: spring-boot-hexagonal" source=bundled` (or
  `source=custom`). This is the only observable contract change, and
  it aligns stderr with the Gherkin assertion — that is the whole
  point of the story.

---

## Section 7 — Composition Root + Tests Plan

### 7.1 Composition root — N/A

**Tier 5 not used.** `internal/cli/wire.go`, `internal/cli/root.go`,
`internal/cli/execute.go`, `cmd/jitctx/main.go`, and
`internal/config/**` are not in the file set. `wire.go` already builds
`fsprofile.NewDetectorWithLogger(cfg.ProfilesDir, logger)` and passes
it into the scan factory; nothing about that wiring changes.
`cfg.ProfilesDir` (default `.jitctx/profiles`) is already the custom
profile directory.

### 7.2 Unit tests

**Tier 6 — groups T6-G1 and T6-G2 run in parallel.** Each group owns a
single test file so the group definition is test-file-scoped.

#### T6-G1 — `internal/infrastructure/fsprofile/detector_test.go` (modify)

Add three new tests alongside the existing three; keep the existing
three as-is (they validate the pom / gradle / no-match paths).

1. `TestDetector_BuildGradleKts` — feature line 178 is about
   `build.gradle`; EP01RF-012 references Gradle generally and the
   bundled YAML already declares `build.gradle.kts`. Cover it with an
   `fstest.MapFS` containing a Kotlin DSL buildscript. Assert
   `prof.Name == "spring-boot-hexagonal"` and
   `prof.Source == model.ProfileSourceBundled`.
2. `TestDetector_CustomOverridesBundled` — feature lines 208-212. Set
   up: write a custom YAML named `my-spring.yaml` under `t.TempDir()`
   whose `detect.files` also matches on `pom.xml`. Use an `fstest.MapFS`
   with a Spring-Boot pom. Assert `prof.Name == "my-spring"` and
   `prof.Source == model.ProfileSourceCustom`.
3. `TestDetector_CustomSourceStamp` and
   `TestDetector_BundledSourceStamp` — sanity that `Source` is
   populated on both branches. Keep small and deterministic.
4. Extend `TestDetector_PomXML` and `TestDetector_BuildGradle` with an
   additional assertion `require.Equal(t, model.ProfileSourceBundled,
   prof.Source)` so the existing two tests also lock the stamp.

No new mocks, no `testdata/` fixtures — `fstest.MapFS` + `t.TempDir()`
cover both custom and bundled cases.

#### T6-G2 — `internal/domain/service/profileClassifier_test.go` (modify)

The existing six sub-cases already cover all Gherkin classification
scenarios *abstractly* (with synthetic profiles). Add two sub-cases to
lock the **bundled-profile** behaviour end-to-end:

1. `entity rule beats service rule for @Entity-annotated class that
   also has @Service` — sanity check on rule ordering.
2. `@Repository classifies as jpa-adapter` — the EP01RF-012 Business
   Rule mentions `@Repository` explicitly but no Gherkin currently
   pins it; pin it with a unit test so the bundled YAML rule is
   load-bearing, not dead code.

Both sub-cases use `prof := loadBundledForTest(t)` — a new tiny helper
at the top of the existing `_test.go` that calls
`fsprofile.New(t.TempDir()).Load(ctx, "spring-boot-hexagonal")`. This
ties the domain-layer test to the bundled YAML file so that reordering
rules in the YAML will fail this test if the semantics drift.

### 7.3 Integration tests

#### T6-G3 — `internal/cli/command/scanIntegration_test.go` (modify)

Edits + additions (all in one file so it is one group):

1. **Fix `TestScanCmd_Integration_ProfileLog` (line 230).** Change the
   assertion from `"profile=spring-boot-hexagonal"` to
   `"Profile: spring-boot-hexagonal"`. Add a second assertion
   `require.Contains(t, logBuf.String(), "source=bundled")` so the
   structured attribute is also verified.

2. **Add `TestScanCmd_Integration_AutoDetectGradleKts`.** Mirror the
   existing `AutoDetectGradle` test: copy the `springBootMinimal`
   fixture, replace `pom.xml` with a `build.gradle.kts` containing
   `plugins { id("org.springframework.boot") version "3.2.0" }`, run
   scan, assert `Profile: spring-boot-hexagonal` in the log buffer and
   `manifestPath` exists.

3. **Add `TestScanCmd_Integration_ServiceByImplementsUseCase`.** Covers
   Gherkin line 202 — the "implements *UseCase + path contains
   service|application" heuristic. Uses a new fixture file (see step
   5). After scan, read `project-state.yaml` and assert that the
   module's contracts include a contract named `CreateUserServiceImpl`
   with `type: service`. Rely on the existing
   `springBootMinimal/expected/project-state.yaml` which now also
   contains the service contract.

4. **Add `TestScanCmd_Integration_CustomOverrideAutoDetect`.** Covers
   Gherkin lines 208-212 from the *CLI* angle (the existing
   `TestScanCmd_Integration_CustomProfileOverride` passes
   `--profile my-spring` explicitly; the Gherkin does NOT pass a
   flag). Fixture: copy `springBootMinimal`, drop a `my-spring.yaml`
   into `.jitctx/profiles/` whose `detect.files` also matches on
   `pom.xml`. Run scan *without* `--profile`. Assert log contains
   `Profile: my-spring` and `source=custom`, and the generated
   manifest lists the stack framework as `my-spring`.

5. **Add `TestScanCmd_Integration_MalformedCustomProfileFallback`.**
   Covers EP01RF-012 §Exceptions. Copy fixture, write an intentionally
   broken YAML (`name: bad\n  invalid: : :`) into
   `.jitctx/profiles/`. Run scan with no `--profile`. Assert exit code
   is 0 (no error returned by `cmd.ExecuteContext`), the manifest is
   written, and the log contains `Profile: spring-boot-hexagonal`
   (fallback to bundled). Capture `stderr`/log-buffer and assert
   presence of `"custom profile parse error"` (already emitted by
   `detector.loadCustomProfiles`).

#### T6-G3 also owns the fixture updates

5. **Create `testdata/springBootMinimal/project/src/main/java/com/app/usermanagement/service/CreateUserServiceImpl.java`.**
   Content: a class implementing `CreateUserUseCase` under a
   `service/` subpackage so the `implements *UseCase + /service/`
   rule fires. Approximate shape:

   ```java
   package com.app.usermanagement.service;

   import com.app.usermanagement.port.in.CreateUserUseCase;

   public class CreateUserServiceImpl implements CreateUserUseCase {
       public UserResponse execute(CreateUserCommand cmd) {
           return null;
       }
   }
   ```

   File path: `testdata/springBootMinimal/project/src/main/java/com/app/usermanagement/service/CreateUserServiceImpl.java`.

6. **Update `testdata/springBootMinimal/expected/project-state.yaml`.**
   Add the new `CreateUserServiceImpl` contract under the
   `user-management` module with `type: service`. Keep everything else
   byte-identical so the existing
   `TestScanCmd_Integration_HappyPath` comparison stays green (the
   helper `stripGeneratedAt` + `normalizeYAML` handles whitespace).
   If the manifest sort order would put the new contract in a
   different position than alphabetically expected, rely on
   `service.SortProjectState` to determine the canonical position; do
   not hand-sort.

### 7.4 What is NOT added

- No new test in `internal/infrastructure/fsprofile/loader_test.go` —
  the existing three cover `Load`, `CustomOverride`, and
  `RejectsTraversal`. The Source-stamping path of `Load` is covered by
  the changes to `detector_test.go` and the integration tests; adding
  another unit test just to assert `prof.Source` would be
  overspecification.
- No `testscript` integration — the story has no process-boundary
  assertion that the in-process integration tests cannot satisfy.

### 7.5 Determinism sanity

`TestScanCmd_Integration_Deterministic` (existing) already scans twice
and compares manifests byte-for-byte after stripping `generated_at`.
After the fixture update (step 5 above), it continues to pass because
the contract list is sorted by `service.SortProjectState`, not by
iteration order.

---

## Section 8 — Open Questions & Risks

| # | Question / Risk | Blocking? | Resolution taken in plan |
|---|-----------------|-----------|--------------------------|
| 1 | Feature line 175 asserts `"Profile: spring-boot-hexagonal"` but the existing integration test asserts `"profile=spring-boot-hexagonal"`. Who wins? | No | Gherkin is the product specification; the test is an implementation artefact. Align to Gherkin: change the log message and the assertion (§5.1, §7.3 step 1). |
| 2 | Should `ProfileSource` be a VO in `internal/domain/vo/` or a named string on the model? | No | Named string on the model. Argument: it is pure identification metadata (no constructor, no invariant), and creating a VO folder for one enum would churn five import paths without benefit. Pattern mirrors `model.ContractType`. |
| 3 | Does `build.gradle.kts` need an explicit Gherkin scenario of its own? | No | The bundled YAML already claims `.kts` support (line 11 of the on-disk file today). We add a unit test + integration test without altering the on-disk YAML contract. If a future epic removes `.kts` coverage, both tests fail loudly. |
| 4 | Should `@Repository` classification have a dedicated Gherkin scenario? | No | EP01RF-012 §Business Rule explicitly lists `@Repository`. Gherkin on lines 168-212 skips it. Solution: retain the bundled rule, pin with a unit test (§7.2 T6-G2 sub-case 2). Non-blocking — the rule is load-bearing on the existing `jpa-adapter` contract type. |
| 5 | The current detector iterates `os.ReadDir` order, which Go docs describe as sorted — is the explicit `sort.Slice` redundant? | No | Redundant today on Linux/macOS/Windows. Keeping the explicit sort is a cheap determinism guard; the extra three lines of code prevent future stdlib drift from silently breaking EP01RNF-002. Non-blocking. |
| 6 | When `Detect` returns a custom profile, should `scanuc` still honour the `input.ProfileName != prof.Name` guard (line 81 of `usecase.go`)? | No | Yes — existing behaviour is correct. If the user passes `--profile spring-boot-hexagonal` and the custom `my-spring` profile happens to match first, the guard returns `ErrNoProfileMatch`. This is desirable (user asked for a specific profile and did not get it) and is already unit-tested via the wider scan integration suite. No change. |
| 7 | Should `Source` be exposed in `project-state.yaml`? | No | No. The manifest serializes `Stack.Frameworks = [prof.Name]` only. Leaking `Source` to the on-disk manifest would force manifest consumers to handle a new field for cosmetic value. Keep `Source` as a runtime/log-only concept. |
| 8 | Does the new fixture file (`CreateUserServiceImpl.java`) break `TestScanCmd_Integration_HappyPath` (existing)? | No | The happy-path test compares generated output to `testdata/springBootMinimal/expected/project-state.yaml`, which we are updating in the same commit/group. Both files change together under T6-G3. |

No blocking questions. Proceed to implementation.

---

## Section 9 — Parallel Execution Plan (authoritative for @agent-manager)

```yaml
tiers:
  - id: 1
    name: Domain contract
    depends_on: []
    groups:
      - id: T1-G1
        scope:
          create: []
          modify:
            - internal/domain/model/frameworkProfile.go
        guidelines:
          - .claude/guidelines/domain-layer-guidelines.yml
        effort: S
        notes: >
          Add type ProfileSource (named string) with constants
          ProfileSourceBundled and ProfileSourceCustom above the
          existing FrameworkProfile struct, and add a single field
          Source ProfileSource to FrameworkProfile (placed
          immediately after Name). Do not touch any other struct.
          Do not add a constructor. No struct tags. No imports change.

  - id: 2
    name: Infrastructure (fsprofile)
    depends_on: [1]
    groups:
      - id: T2-G1
        scope:
          create: []
          modify:
            - internal/infrastructure/fsprofile/detector.go
            - internal/infrastructure/fsprofile/loader.go
            - internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal.yaml
        guidelines:
          - .claude/guidelines/infrastructure-layer-guidelines.yml
        effort: M
        notes: >
          detector.go — stamp prof.Source = model.ProfileSourceCustom
          in loadCustomProfiles after decodeProfile succeeds, and
          prof.Source = model.ProfileSourceBundled in
          loadBundledProfiles. Add an explicit sort.Slice by filename
          ascending in both loaders (import "sort"). Do NOT change the
          Detect method signature or the custom-first / bundled-second
          loop. Keep the existing Warn log on malformed YAML.
          loader.go — mirror: stamp Source on the returned profile in
          both the user-dir hit branch and the embeddedProfiles
          fallback branch. Do NOT change path-traversal rejection,
          KnownFields(true), or the List method. dto.go and mapper.go
          are NOT in scope — Source is set after toDomain runs.
          spring-boot-hexagonal.yaml — reorder the rules so the two
          implements:"*UseCase" rules appear before has_annotation:
          Service, and add a brief inline comment on each rule
          pointing to the Gherkin scenario line. The declared
          classify_as values and the detect.files list are unchanged
          (pom.xml, build.gradle, build.gradle.kts all remain).

  - id: 3
    name: Application (scanuc log format)
    depends_on: [1]
    groups:
      - id: T3-G1
        scope:
          create: []
          modify:
            - internal/application/usecase/scanuc/usecase.go
        guidelines:
          - .claude/guidelines/application-layer-guidelines.yml
        effort: S
        notes: >
          Replace the single line
          u.logger.Info("profile selected", "profile", prof.Name)
          with
          u.logger.Info(fmt.Sprintf("Profile: %s", prof.Name),
          "source", string(prof.Source)).
          fmt is already imported. No other change to Execute. No
          change to port / VO / Input / Output / error-wrapping.

  - id: 6
    name: Tests (parallel)
    depends_on: [2, 3]
    groups:
      - id: T6-G1
        scope:
          create: []
          modify:
            - internal/infrastructure/fsprofile/detector_test.go
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: M
        notes: >
          Add TestDetector_BuildGradleKts (fstest.MapFS with a Kotlin
          DSL buildscript — assert Name == spring-boot-hexagonal and
          Source == ProfileSourceBundled),
          TestDetector_CustomOverridesBundled (write custom
          my-spring.yaml into t.TempDir() with matching detect.files
          on pom.xml; assert Name == my-spring and Source ==
          ProfileSourceCustom), TestDetector_CustomSourceStamp and
          TestDetector_BundledSourceStamp sanity tests. Extend the
          two existing tests (PomXML and BuildGradle) with an
          assertion on Source == ProfileSourceBundled. All tests use
          t.Parallel() and fstest.MapFS / t.TempDir() only (no
          testdata/).

      - id: T6-G2
        scope:
          create: []
          modify:
            - internal/domain/service/profileClassifier_test.go
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: S
        notes: >
          Add two sub-cases to the existing table-driven test: one
          locking the @Repository -> jpa-adapter rule, one locking
          that @Entity wins over @Service for a class annotated with
          both. Introduce a small top-of-file helper
          loadBundledForTest(t) that calls fsprofile.New(t.TempDir())
          .Load(ctx, "spring-boot-hexagonal") so the two new cases run
          against the actual bundled YAML (the rest of the table
          keeps using inline profiles). All sub-cases use t.Parallel().

      - id: T6-G3
        scope:
          create:
            - testdata/springBootMinimal/project/src/main/java/com/app/usermanagement/service/CreateUserServiceImpl.java
          modify:
            - internal/cli/command/scanIntegration_test.go
            - testdata/springBootMinimal/expected/project-state.yaml
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: L
        notes: >
          scanIntegration_test.go — (1) change the
          TestScanCmd_Integration_ProfileLog assertion from
          "profile=spring-boot-hexagonal" to
          "Profile: spring-boot-hexagonal" and add
          require.Contains(t, logBuf.String(), "source=bundled"); (2)
          add TestScanCmd_Integration_AutoDetectGradleKts mirroring
          the existing AutoDetectGradle but writing build.gradle.kts;
          (3) add TestScanCmd_Integration_ServiceByImplementsUseCase
          asserting the new fixture file is classified as service;
          (4) add TestScanCmd_Integration_CustomOverrideAutoDetect
          running scan WITHOUT --profile against a tree that contains
          both a matching custom profile and the matching bundled
          profile, asserting Profile: my-spring + source=custom in
          logs and framework == my-spring in the manifest; (5) add
          TestScanCmd_Integration_MalformedCustomProfileFallback that
          drops an invalid YAML into .jitctx/profiles/ and asserts the
          scan still succeeds and falls back to
          spring-boot-hexagonal. All new tests use t.Parallel() and
          t.TempDir(). Reuse discardLogger / buildScanFactoryWithLogger
          / copyFixture / fixtureDir from helpers_test.go.
          CreateUserServiceImpl.java — package
          com.app.usermanagement.service, implements
          CreateUserUseCase (imported from the existing
          port/in/CreateUserUseCase.java), body returns null. The
          class sits under /service/ so the bundled "implements
          *UseCase + /service/" rule classifies it. No new imports
          unavailable in the fixture tree.
          expected/project-state.yaml — add a new contract entry
          under the user-management module with name
          CreateUserServiceImpl and type service. Let
          service.SortProjectState decide the in-array position
          (alphabetical); do not hand-place. Keep every other line
          byte-for-byte identical so the existing HappyPath test
          stays green.
```

---

## Self-validation checklist

- [x] Every file in Section 1 appears in exactly one group in Section 9
      (10 rows → 10 distinct entries across T1-G1, T2-G1, T3-G1, T6-G1,
      T6-G2, T6-G3).
- [x] Every requirement ID is covered by at least one Section 1 row:
      EP01US-005 + EP01RF-012 — all rows; EP01RNF-002 — rows 2, 8;
      EP01RNF-004 — rows 3, 4.
- [x] No file path appears in two groups.
- [x] Every port referenced in Section 2 exists in the codebase today
      (`DetectProfilePort`, `LoadProfilePort`, `ListProfilesPort` all
      present under `internal/domain/port/profile/`; verified by
      directory listing). Zero new ports.
- [x] Use-case `Execute` signature unchanged — reconfirmed against
      `scanuc.UseCase`.
- [x] `Deps` struct: no new fields — reconfirmed in `wire.go`.
- [x] No `TODO` / `{placeholder}` in the plan.
- [x] DAG is acyclic: T1 → T2, T1 → T3, T2 + T3 → T6. No cycles.
- [x] Tier 1 exists because `internal/domain/model/frameworkProfile.go`
      is in the file set.
- [x] Tier 5 is NOT introduced because no wiring file appears in the
      file set; listed as N/A in Section 7.1.
- [x] Every `guidelines[]` path in Section 9 exists (verified:
      `domain-layer-guidelines.yml`, `infrastructure-layer-guidelines.yml`,
      `application-layer-guidelines.yml`, `unit-test-layer-guidelines.yml`,
      `integration-test-layer-guidelines.yml` all present under
      `.claude/guidelines/`).
- [x] No `Blocking: Yes` open question — proceed to implementation.
