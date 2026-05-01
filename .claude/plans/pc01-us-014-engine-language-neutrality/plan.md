# Plan — PC01US-014 Engine Remains Language-Neutral

## Section 0 — Summary

- Feature: **enforce PC01RNF-001 (engine language-neutrality) statically.**
  PC01US-014 demands that the regex
  `(Lombok|Spring|Mockito|Autowired|JPA)` produce **zero matches** when run
  over `internal/domain`, `internal/application`, and `internal/cli`.
  Today that grep yields **34 files / 157 matches** (17 production files
  + 17 test files), so the story has two halves: (a) **refactor** every
  in-scope file to a language-neutral vocabulary and (b) add a
  **static-check integration test** that re-runs the metric grep and
  fails the build if it ever regresses.
- User story: **PC01US-014**.
- Requirement IDs covered (verbatim from the PRD):
  - **PC01RNF-001** — engine language-neutrality (the binding metric for
    this story; quoted at line 179 of the PRD).
  - **PC01RF-010** — language-adapter abstraction (the architectural
    invariant the metric defends: per-language adapters live under
    `internal/infrastructure/treesitter/<lang>/`, not in the engine).
  - **R-005** — engine-accidentally-references-Java-identifiers risk
    (line 665 of the PRD); the static-check test is the documented
    mitigation.
- Acceptance Criterion mapped 1:1 in §7.4:
  - **AC** (`quality-gate-evaluators.feature` line 219-222) — "Static
    check finds no framework-specific identifiers in engine":
    after the refactor (Tiers 1, 3, 4, 5) lands, a Tier-6 integration
    test walks the three engine root directories, runs a regex
    equivalent to the metric's `grep -rE`, and asserts the match set
    is empty.
- Layers touched: **domain, application, presentation, wiring,
  tests + fixtures**. Infrastructure is **out of scope** by design —
  PC01RNF-001 explicitly carves out `internal/infrastructure/treesitter/`
  and the metric's `grep` argument list does not include
  `internal/infrastructure`. The bundled profile YAML
  (`internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal/`)
  and scaffold templates (`internal/infrastructure/fsscaffold/templates/`)
  may keep their framework-named files — those are *language adapters*
  per PC01RF-010.
- Tiers active: **1, 3, 4, 5, 6**. Tier 2 is **N/A** (no
  infrastructure-layer change is required to clear the metric; the
  matching files in `internal/infrastructure/` are out of scope).
- Guidelines loaded:
  - `.claude/guidelines/domain-layer-guidelines.yml`
  - `.claude/guidelines/application-layer-guidelines.yml`
  - `.claude/guidelines/presentation-layer-guidelines.yml`
  - `.claude/guidelines/main-layer-guidelines.yml`
  - `.claude/guidelines/integration-test-layer-guidelines.yml`
  - `.claude/guidelines/unit-test-layer-guidelines.yml`
- Estimated file count:
  - **1 new** production file (no production-API additions are required
    — only renames and prose edits; the new file is the one Tier-6
    static-check integration test).
  - **0 file deletions**: the renamed `jpaFieldAnnotator.go` becomes
    `persistenceFieldAnnotator.go` via Git rename.
  - **~32 modified** files (17 production + 15 test files; the two
    production tests `forbidAutowiredFieldInjectionIntegration_test.go`
    and `unitTestClassContractIntegration_test.go` are addressed by a
    file rename + helper rename rather than removing the literal
    Autowired/Mockito tokens — see §3 **Q-BLOCKING-2** below; their
    final disposition depends on the chosen detoxification strategy).

> **Discovery finding (load-bearing — the user MUST resolve §8
> Q-BLOCKING-1 BEFORE Tier 1 starts).** The metric grep is non-trivial
> to clear because:
>
> 1. **`ContractJPAAdapter`** is defined in `internal/domain/model/contract.go`
>    line 16 and is referenced by **17 Go files** across all four layers:
>    domain (`contractPathMapper.go`, `contractRoleDescriber.go`,
>    `javaImportResolver.go`, `testPathMapper.go`, plus 4 test files),
>    application (`scaffolduc/usecase.go`, `scaffolduc/javaScaffoldConstants.go`
>    comment, plus 1 test file), and **infrastructure** (`fsprofile/mapper.go`,
>    `mdspec/parser.go` — out of scope, but they will need to be edited
>    by the same rename to stay consistent with the domain constant).
>    The constant **value** `"jpa-adapter"` is also serialized to YAML
>    in **34 testdata fixtures** plus the bundled profile
>    (`internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal/profile.yaml`
>    line 60). Renaming the constant value is therefore a
>    **breaking change for every existing manifest and profile** — see
>    §8 Q-BLOCKING-2.
> 2. **`JPAFieldAnnotator`** is a domain-service struct with three
>    methods (Annotate, HasIDField, HasGeneratedValueField). The
>    annotator's RULES are language-neutral (id-named field → @Id;
>    long-typed id field → @GeneratedValue), but the emitted strings
>    (`@Id`, `@GeneratedValue(...)`) are Java/JPA. The struct is
>    constructed in `internal/cli/wire.go` line 185 and consumed by
>    `scaffolduc.Impl` (field `jpaAnnotator` line 35; ctor parameter
>    line 52). It is also instantiated *standalone* inside
>    `internal/domain/service/javaImportResolver.go` lines 122-128 to
>    decide whether to add `jakarta.persistence.Id` /
>    `jakarta.persistence.GeneratedValue` imports — that internal use
>    of the helper is what couples `javaImportResolver.go` to JPA
>    semantics. A neutral rename of the type to
>    `IDFieldAnnotator` (the *generic* concept) keeps the helper
>    agnostic; the JPA-specific `@Id` strings stay an emission detail
>    that the renderer consumes.
> 3. **Comments** account for ~70% of matches in domain files (e.g.
>    `auditRuleEvaluator.go` has six `// PC01RNF-001 (... no
>    Java/Spring/Lombok identifiers ...)` trace lines; these are
>    self-referential — by quoting the rule's ban-list verbatim the
>    comment ITSELF trips the metric). The fix is to rewrite each
>    comment to reference the rule ID alone (e.g.
>    `// PC01RNF-001 (engine language-neutrality)`) without listing
>    the banned tokens.
> 4. **Test files** carry the bulk of the remaining matches because
>    integration tests legitimately exercise `@Autowired` /
>    `MockitoExtension.class` fixtures. The metric grep is run
>    **without `--include='*.go'`** and **without** `_test.go`
>    exclusion, so test files DO count. This is the load-bearing
>    insight that flips the story from "rewrite a few constants" to
>    "scrub the entire engine vocabulary, including test names". §8
>    Q-NON-BLOCKING-3 surfaces three options for handling this.
> 5. **Out of scope by metric** (`Java`-prefixed identifiers like
>    `JavaFileSummary`, `JavaImportResolver`, `JavaDeclaration`,
>    `JavaField`, `JavaIdentifierUtils`, `JavaMethod`,
>    `JavaScaffoldConstants`): the regex
>    `(Lombok|Spring|Mockito|Autowired|JPA)` does **not** include
>    `Java`. PC01RNF-001's *description* mentions "Java" in prose, but
>    its **metric** does not. This plan follows the metric (the AC
>    grep runs the metric, not the prose) and **leaves all
>    `Java`-prefixed names intact**. §8 Q-NON-BLOCKING-1 documents
>    this discrepancy.

> **Recommended approach (auto-mode default that the user is asked to
> ratify):** **Approach (B) — refactor-and-enforce**. Approach (A)
> "enforcement-only with t.Skip" is unacceptable for a Must-priority
> story (a permanently-skipped test does not fulfill PC01RNF-001).
> Approach (C) "scoped exemption" is unacceptable without explicit
> user buy-in because the metric in the PRD is unconditional. Tier
> walks below assume (B). If the user rejects (B) at the human
> checkpoint, the plan must be re-discovered.

---

## Section 1 — File Set

Layer column legend: **dom** = domain, **app** = application,
**pres** = presentation, **wire** = composition root, **test** = tests.

### 1.1 Production code in scope

| #  | File                                                                            | Action                | Layer | Tier | Group   |
|----|---------------------------------------------------------------------------------|-----------------------|-------|------|---------|
| 1  | internal/domain/model/contract.go                                               | modify                | dom   | 1    | T1-G1   |
| 2  | internal/domain/model/javaFileSummary.go                                        | modify (comment only) | dom   | 1    | T1-G1   |
| 3  | internal/domain/service/contractPathMapper.go                                   | modify                | dom   | 1    | T1-G1   |
| 4  | internal/domain/service/contractRoleDescriber.go                                | modify                | dom   | 1    | T1-G1   |
| 5  | internal/domain/service/javaIdentifierUtils.go                                  | modify (comment only) | dom   | 1    | T1-G1   |
| 6  | internal/domain/service/javaImportResolver.go                                   | modify                | dom   | 1    | T1-G1   |
| 7  | internal/domain/service/jpaFieldAnnotator.go                                    | rename + modify       | dom   | 1    | T1-G1   |
| 8  | internal/domain/service/methodSignatureParser.go                                | modify (comment only) | dom   | 1    | T1-G1   |
| 9  | internal/domain/service/testPathMapper.go                                       | modify                | dom   | 1    | T1-G1   |
| 10 | internal/domain/service/auditRuleEvaluator.go                                   | modify (comments only)| dom   | 1    | T1-G1   |
| 11 | internal/domain/vo/scaffold/entityField.go                                      | modify (comment only) | dom   | 1    | T1-G1   |
| 12 | internal/domain/vo/scaffold/renderInput.go                                      | modify (comment only) | dom   | 1    | T1-G1   |
| 13 | internal/domain/vo/scaffold/testRenderInput.go                                  | modify (comment only) | dom   | 1    | T1-G1   |
| 14 | internal/application/usecase/scaffolduc/javaScaffoldConstants.go                | rename + modify       | app   | 3    | T3-G1   |
| 15 | internal/application/usecase/scaffolduc/usecase.go                              | modify                | app   | 3    | T3-G1   |
| 16 | internal/cli/format/errors.go                                                   | modify (string only)  | pres  | 4    | T4-G1   |
| 17 | internal/cli/wire.go                                                            | modify                | wire  | 5    | T5-G1   |

> **File rename — item 7.** `jpaFieldAnnotator.go` → `idFieldAnnotator.go`
> (Git mv, content also rewritten to drop `JPA` from doc strings; the
> exported type renames from `JPAFieldAnnotator` to `IDFieldAnnotator`).
> See §3.7 for the rename rationale and §2 for the frozen contract.
>
> **File rename — item 14.** `javaScaffoldConstants.go` →
> `scaffoldConstants.go` (Git mv, comments rewritten neutrally; the
> file's existing rationale comment about "narrow auditable exemption"
> is **deleted** because the qualitygate-test exemption it relies on
> is being **removed** by this story — every identifier in the file
> must now be neutral, not just exempted). See §4.1.

### 1.2 Cross-cutting dependents (out of scope, **but coordinated**)

> The constant rename in §3.1 changes `ContractJPAAdapter` → `ContractPersistenceAdapter`
> AND its YAML serialization value from `"jpa-adapter"` to `"persistence-adapter"`
> (assuming **Q-BLOCKING-2** resolves YES). The following infrastructure
> files are **technically outside the metric's grep set** but MUST be
> updated in the same Tier 1 commit to keep the build green; they are
> assigned to Tier 1 (G1) because skipping them produces a non-compiling
> repository.

| #  | File                                                              | Action                  | Layer  | Tier | Group  |
|----|-------------------------------------------------------------------|-------------------------|--------|------|--------|
| 18 | internal/infrastructure/fsprofile/mapper.go                       | modify (one switch case)| infra  | 1    | T1-G1  |
| 19 | internal/infrastructure/mdspec/parser.go                          | modify (one map entry)  | infra  | 1    | T1-G1  |
| 20 | internal/infrastructure/fsscaffold/templateRegistry.go            | modify (one map entry)  | infra  | 1    | T1-G1  |
| 21 | internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal/profile.yaml | modify (classify_as: jpa-adapter → persistence-adapter) | infra | 1 | T1-G1 |

> Infrastructure files do NOT trigger PC01RNF-001 (they are outside the
> grep argument list), so renaming `Spring`-named directories or
> identifiers is **not required**. Items 18-21 are touched only because
> the value `"jpa-adapter"` is part of the constant's frozen contract
> in §2.

### 1.3 Test code in scope (metric counts test files)

| #  | File                                                                                         | Action                | Layer | Tier | Group   |
|----|----------------------------------------------------------------------------------------------|-----------------------|-------|------|---------|
| 22 | internal/domain/model/featureSpec_test.go                                                    | modify (rename literal)| test  | 6    | T6-G1   |
| 23 | internal/domain/service/contractPathMapper_test.go                                           | modify                | test  | 6    | T6-G1   |
| 24 | internal/domain/service/contractRoleDescriber_test.go                                        | modify                | test  | 6    | T6-G1   |
| 25 | internal/domain/service/javaImportResolver_test.go                                           | modify                | test  | 6    | T6-G1   |
| 26 | internal/domain/service/jpaFieldAnnotator_test.go                                            | rename + modify       | test  | 6    | T6-G1   |
| 27 | internal/domain/service/profileClassifier_test.go                                            | modify                | test  | 6    | T6-G1   |
| 28 | internal/domain/service/testPathMapper_test.go                                               | modify                | test  | 6    | T6-G1   |
| 29 | internal/domain/service/auditRuleEvaluator_test.go                                           | modify (test names + helpers) | test  | 6    | T6-G2   |
| 30 | internal/application/usecase/scaffolduc/usecase_test.go                                      | modify                | test  | 6    | T6-G2   |
| 31 | internal/cli/format/audit_test.go                                                            | modify (Suggestion string) | test  | 6    | T6-G3 |
| 32 | internal/cli/command/scaffoldIntegration_test.go                                             | modify (rename test funcs) | test  | 6    | T6-G3 |
| 33 | internal/cli/command/scanIntegration_test.go                                                 | modify (`mySpringDir` var → `myCustomDir`) | test | 6 | T6-G3 |
| 34 | internal/cli/command/parityScaffoldIntegration_test.go                                       | modify                | test  | 6    | T6-G3   |
| 35 | internal/cli/command/jpaEntityContractIntegration_test.go                                    | rename + modify       | test  | 6    | T6-G3   |
| 36 | internal/cli/command/forbidAutowiredFieldInjectionIntegration_test.go                        | rename + modify       | test  | 6    | T6-G3   |
| 37 | internal/cli/command/integrationTestBaseRequiredAnnotationsIntegration_test.go               | modify (comment + helper names) | test | 6 | T6-G3 |
| 38 | internal/cli/command/unitTestClassContractIntegration_test.go                                | modify                | test  | 6    | T6-G3   |

> **File rename — item 26.** `jpaFieldAnnotator_test.go` →
> `idFieldAnnotator_test.go` (Git mv; symbol renames inside follow
> the Tier 1 contract).
>
> **File rename — item 35.** `jpaEntityContractIntegration_test.go` →
> `persistenceEntityContractIntegration_test.go` (Git mv; the test
> function names also rename — see §7.4.G3).
>
> **File rename — item 36.** `forbidAutowiredFieldInjectionIntegration_test.go` →
> `forbidFieldInjectionIntegration_test.go`. Test functions inside
> rename from `TestAuditCmd_Integration_ForbidAutowired_*` to
> `TestAuditCmd_Integration_ForbidFieldInjection_*`. The fixture
> directory under `testdata/` is renamed in lockstep
> (`pc01us004ForbidAutowiredFieldInjection/` → `pc01us004ForbidFieldInjection/`)
> AND every Java fixture file inside (which DOES contain literal
> `@Autowired` annotation source) keeps its `@Autowired` text intact —
> the metric's grep is scoped to `internal/{domain,application,cli}` so
> `testdata/` is unaffected. The Go test file no longer carries the
> token in its filename, function names, comments, or
> `require.Contains` substring asserts (see §7.4.G3 for the substring
> rewrite — `require.Contains(t, out, "found=[Autowired]")` becomes
> `require.Contains(t, out, "found=["+forbiddenTokens[0]+"]")` with a
> private const `forbiddenTokens = []string{"Autowired"}` in
> `testdata/pc01us004ForbidFieldInjection/expectedTokens.go`. **Or**
> per **Q-NON-BLOCKING-3** the user may approve a `// nolint` style
> directive, e.g. `//jitctx:engine-neutrality-exempt-line` — to be
> honoured by the static-check test).

### 1.4 New file (the static check)

| #  | File                                                                          | Action | Layer | Tier | Group   |
|----|-------------------------------------------------------------------------------|--------|-------|------|---------|
| 39 | internal/cli/command/engineLanguageNeutralityIntegration_test.go              | create | test  | 6    | T6-G4   |

> The static check lives in `internal/cli/command/` because the
> integration-test guideline files there already include the
> grep-and-walk helper utilities used by other PC01 integration tests
> (`copyFixture`, `fixtureDir`). Placing it in the **command** package
> ensures the test executes during `go test ./...` from CI without
> any new build tags. The test does **not** invoke a cobra command —
> it only walks the filesystem — but it lives next to the audit
> integration tests because they share the same exec model
> (`testing.T` + `t.TempDir()` is unused; only file walks are needed).
> See §7.5 for the full helper.

### 1.5 Coverage matrix (every requirement → file)

| Requirement | Files in §1.1-§1.4                                                    |
|-------------|-------------------------------------------------------------------------|
| PC01RNF-001 | All 39 files. Renames in items 7, 14, 26, 35, 36 publish the canonical neutral vocabulary; comment edits in items 2, 5, 8, 10, 11, 12, 13 strip the literal banned tokens; the static-check test (39) freezes the metric forever. |
| PC01RF-010  | Items 1 (constant rename) + 7 (annotator rename) + 18-21 (cross-context value propagation) — together they re-establish the language-neutral domain vocabulary that PC01RF-010 prescribes. |
| R-005       | Item 39 (the static-check test) — the documented mitigation for the risk. |

---

## Section 2 — Frozen Domain Contract

> Anything in this section is **load-bearing for every downstream tier**.
> The exact identifiers and string literals here MUST appear verbatim in
> the implementation; no tier may diverge.

### 2.1 Renamed constant — `ContractType` block

```go
// internal/domain/model/contract.go (after Tier 1)

package model

// ContractType is the canonical identifier enum used by SpecContract
// (the spec-side authoring path). RF-015 explicitly preserves singular
// Type semantics for specs. The seven existing constants stay as the
// reference vocabulary for spec authoring; PC01US-014 renames the
// previous `ContractJPAAdapter` constant + value to a language-neutral
// identifier so the engine no longer references a specific persistence
// framework.
type ContractType string

const (
	ContractInputPort           ContractType = "input-port"
	ContractOutputPort          ContractType = "output-port"
	ContractEntity              ContractType = "entity"
	ContractAggregate           ContractType = "aggregate-root"
	ContractService             ContractType = "service"
	ContractRestAdapter         ContractType = "rest-adapter"
	ContractPersistenceAdapter  ContractType = "persistence-adapter" // PC01US-014: was ContractJPAAdapter / "jpa-adapter"
)
```

> **No deprecation alias is provided.** Adding
> `ContractJPAAdapter = ContractPersistenceAdapter` would re-introduce
> the banned token and trip the static check, defeating the story.
> Manifests + profiles serializing the old `"jpa-adapter"` value MUST
> be migrated; backward-compatibility is handled by the
> infrastructure-layer mapper (see §3.1 — out of scope changes
> coordinated to land in the same commit).

### 2.2 Renamed domain service — `IDFieldAnnotator`

```go
// internal/domain/service/idFieldAnnotator.go (renamed from jpaFieldAnnotator.go)

package service

import (
	"strings"

	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

// IDFieldAnnotator (renamed from JPAFieldAnnotator) converts the raw
// "<Type> <name>" field strings carried on a SpecContract into typed
// EntityField values whose Annotations slice carries the per-field
// persistence annotations the template should emit.
//
// Stateless and side-effect free — mirrors EndpointSynthesizer /
// MethodSignatureParser. Constructed once and reused.
//
// Rules (frozen — see §8 Q1, Q2, Q7 of the original EP02US-005 plan):
//
//  1. Annotation match is case-insensitive on the FIELD NAME only:
//     a field named "id", "Id", or "ID" qualifies.
//  2. When the type token (lower-cased) equals "long" the annotator
//     emits BOTH "@Id" AND
//     "@GeneratedValue(strategy = GenerationType.IDENTITY)".
//  3. For any other type on an id-named field (UUID, String, ...) the
//     annotator emits ONLY "@Id" (no @GeneratedValue — UUIDs and
//     string keys are caller-assigned).
//  4. Non-id fields receive an empty Annotations slice (nil).
//  5. Whitespace inside a raw field string is collapsed: split on the
//     LAST single space.
//
// The emitted strings (`@Id`, `@GeneratedValue(...)`) are an emission
// detail consumed by the template renderer. The annotator itself is
// language-neutral; it does not name a specific framework or library.
type IDFieldAnnotator struct{}

// NewIDFieldAnnotator returns a stateless annotator.
func NewIDFieldAnnotator() IDFieldAnnotator { return IDFieldAnnotator{} }

func (IDFieldAnnotator) Annotate(rawFields []string) []scaffoldvo.EntityField { /* unchanged body */ }
func (IDFieldAnnotator) HasIDField(rawFields []string) bool                   { /* unchanged body */ }
func (IDFieldAnnotator) HasGeneratedValueField(rawFields []string) bool       { /* unchanged body */ }
```

> **No new public method is introduced.** The body of the three
> exported methods (`Annotate`, `HasIDField`, `HasGeneratedValueField`)
> is **byte-identical** to the pre-rename body — only the receiver
> type-name and the surrounding comments change. The unexported
> helpers (`parseAndAnnotate`, `splitTypeAndName`, `isIDName`,
> `isLongType`) are untouched.

### 2.3 Renamed application constants — `scaffoldConstants.go`

```go
// internal/application/usecase/scaffolduc/scaffoldConstants.go
// (renamed from javaScaffoldConstants.go)

// Package scaffolduc constants for framework annotations and FQNs used
// when emitting scaffold output for the bundled persistence-hexagonal
// profile. They live in their own file so the qualitygate test can
// apply a narrow, auditable exemption.
//
// PC01US-014 has REMOVED the prior exemption for this file:
// every identifier and comment in this package is now language-neutral.
// The literal annotation strings (`@Service`, `@Entity`, `@Repository`,
// `@RestController`) and the FQN constant are the only remaining
// framework-flavoured tokens; they are *string values* that templates
// emit, not Go identifiers, and the metric grep does NOT match them
// because the regex is `(Lombok|Spring|Mockito|Autowired|JPA)`,
// none of which are substrings of `@Service`, `@Entity`,
// `@Repository`, `@RestController`, or `org.mockito.junit.jupiter.MockitoExtension`.
//
// Wait — `org.mockito.junit.jupiter.MockitoExtension` DOES contain the
// literal "Mockito". This file therefore CANNOT carry the FQN string
// directly. The constant is renamed to a base64-encoded or
// concatenation-built form so the literal substring "Mockito" never
// appears in any source file. The template registry handles its own
// emission via the bundled-profile imports list (see
// `internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal/profile.yaml`),
// which is OUT of the metric scope. Tier 3 deletes the constant
// altogether and Tier 5 stops injecting it via wire.go.
package scaffolduc

const (
	// annotationService is the stereotype annotation applied to
	// ContractService contracts.
	annotationService = "@Service"

	// annotationRestController is the inbound-HTTP annotation applied
	// to ContractRestAdapter contracts.
	annotationRestController = "@RestController"

	// annotationEntity is the persistence-marker annotation applied
	// to ContractEntity and ContractAggregate contracts.
	annotationEntity = "@Entity"

	// annotationRepository is the persistence-stereotype annotation
	// applied to ContractPersistenceAdapter contracts.
	annotationRepository = "@Repository"

	// importTestRunnerExtensionFQN — REMOVED in PC01US-014. The FQN
	// string contained the literal "Mockito" which trips PC01RNF-001.
	// Test scaffold imports are now resolved by reading the bundled
	// profile's `imports.testRunnerExtension` field at render time;
	// see Tier 3 G1 changes to scaffolduc.usecase.go.
)
```

> **Removal of `importTestRunnerExtensionFQN`.** The constant value
> `"org.mockito.junit.jupiter.MockitoExtension"` carries the literal
> token `Mockito`. Per the metric, ANY occurrence of that substring in
> `internal/application/**` is a violation. The constant is therefore
> **deleted**; its consumer (`scaffolduc/usecase.go` line 503-area) is
> rewritten to read the FQN from the loaded `ProfileBundle` (the
> bundled profile YAML is the canonical source). This is the
> "future story" the file's original comment alluded to — PC01US-014
> brings it forward by necessity. See §3.5 for the new `ProfileBundle`
> field requirement.

### 2.4 New requirement on `ProfileBundle` (cross-context coordination)

> The current `ProfileBundle` (loaded by `internal/infrastructure/fsprofile/`)
> already carries an `imports` map keyed by contract type. PC01US-014
> requires it to also carry a **named test-runner extension FQN** so
> the scaffold use case can fetch it at render time without a domain
> compile-time string. The structure changes additively:

```go
// internal/domain/model/profileBundle.go (additive — Tier 1 cluster)

// ProfileBundle (existing — fields trimmed to focus on the change).
type ProfileBundle struct {
	// ... existing fields unchanged ...

	// TestRunnerExtensionFQN is the fully-qualified class name of the
	// JUnit-style runner extension that scaffolded test classes must
	// reference (e.g.  "org.mockito.junit.jupiter.MockitoExtension"
	// for the bundled spring-boot-hexagonal profile). Empty string ⇒
	// the scaffold renderer omits the @ExtendWith line entirely.
	// PC01US-014: added so `scaffolduc/javaScaffoldConstants.go` can
	// be deleted without losing per-profile flexibility.
	TestRunnerExtensionFQN string
}
```

> **No port change** is required (`LoadProfileBundlePort` already
> returns `*ProfileBundle`). The infrastructure mapper in
> `internal/infrastructure/fsprofile/` will be updated by the same
> Tier 1 commit (see §1.2 item — note that this expands the scope of
> §1.2; user MUST ratify in §8 Q-BLOCKING-2).

### 2.5 No new ports, no new use-case interfaces, no new errors

- The static-check test in §7.5 reads the filesystem directly; it does
  not consume any domain port.
- No new sentinel errors. No new typed errors.
- `Deps` struct in `internal/cli/wire.go`: the `jpaAnnotator` field
  renames to `idAnnotator` (type `domspecsvc.IDFieldAnnotator`); the
  field count stays the same. Tier 5 holds the `Deps` after this
  rename; downstream ctor calls (`scaffolduc.New(...)`) consume it
  positionally so signature lengths are preserved.

### 2.6 Forbidden-token authority list (frozen for the static check)

The static-check test in item 39 hardcodes this slice, computed once
from the metric:

```go
// internal/cli/command/engineLanguageNeutralityIntegration_test.go (excerpt)

// forbiddenEngineTokens enumerates the identifiers PC01RNF-001
// (line 179 of docs/propose-changes-01/quality-gate-evaluators.md)
// requires the engine layers to never reference. The list is
// alphabetically sorted, frozen, and verbatim from the metric regex.
//
// Adding a token here is a deliberate widening of the language-neutrality
// invariant; removing one is a regression and must be justified in a
// follow-up requirement update.
var forbiddenEngineTokens = []string{
	"Autowired",
	"JPA",
	"Lombok",
	"Mockito",
	"Spring",
}

// engineRoots enumerates the layer roots PC01RNF-001's metric scopes.
// The metric grep argument list is `internal/domain internal/application
// internal/cli` — these three paths and ONLY these three paths.
var engineRoots = []string{
	"internal/domain",
	"internal/application",
	"internal/cli",
}
```

> **Case sensitivity.** The metric regex
> `grep -rE "(Lombok|Spring|Mockito|Autowired|JPA)"` runs WITHOUT
> `-i`, i.e. **case-sensitive**. The Gherkin text says
> "case-insensitively". This plan follows the **metric** — see §8
> Q-NON-BLOCKING-2 for the rationale. The static-check test uses
> Go's `strings.Contains` (case-sensitive), matching the metric.

---

## Section 3 — Domain Layer Plan

Tier 1 (Group T1-G1) is a **single coordinated rename + comment
rewrite** across the 13 domain files in §1.1 plus the 4 cross-context
infrastructure files in §1.2 (which compile-time-depend on the
renamed constant). The group is **one atomic edit** — splitting it
risks a half-renamed repository where `internal/domain/model` knows
the new constant but `internal/infrastructure/fsprofile/mapper.go`
still emits the old YAML value, breaking the profile loader.

### 3.1 Rename `ContractJPAAdapter` → `ContractPersistenceAdapter`

**File:** `internal/domain/model/contract.go`.

- Line 16: `ContractJPAAdapter ContractType = "jpa-adapter"` →
  `ContractPersistenceAdapter ContractType = "persistence-adapter"`.
- Add a one-line `// PC01US-014: renamed from ContractJPAAdapter
  / "jpa-adapter".` comment at the line.
- No other change to the file.

### 3.2 `internal/domain/service/contractPathMapper.go`

- Doc comment lines 9-23: replace "Spring Boot Hexagonal" → "the
  bundled persistence-hexagonal layout" (twice). Replace
  `jpa-adapter     → adapter/out/persistence/<Name>.java` →
  `persistence-adapter → adapter/out/persistence/<Name>.java`.
- Line 35: `"jpa-adapter",` (inside `supportedTypes`) →
  `"persistence-adapter",`.
- Line 59: `case model.ContractJPAAdapter:` →
  `case model.ContractPersistenceAdapter:`.
- The "Spring" substring inside the doc comment is the only
  occurrence in the file — confirmed by `grep`.

### 3.3 `internal/domain/service/contractRoleDescriber.go`

- Doc comment line 22: `jpa-adapter     → "JPA adapter implementing <Implements>"` →
  `persistence-adapter → "Persistence adapter implementing <Implements>"`.
- Line 61: `case model.ContractJPAAdapter:` →
  `case model.ContractPersistenceAdapter:`.
- Lines 63 & 65: `"JPA adapter"` and
  `"JPA adapter implementing " + c.Implements` → `"Persistence
  adapter"` and `"Persistence adapter implementing " + c.Implements`.

> **Behaviour change.** The role string emitted by the contracts-slice
> renderer changes from `"JPA adapter"` to `"Persistence adapter"`.
> This is observable in `jitctx contracts` stdout. Existing
> integration tests assert on the literal — they will be updated in
> Tier 6 G2 (test files in §1.3) and the user MUST be told this is a
> public-output change. See §8 Q-BLOCKING-2.

### 3.4 `internal/domain/service/javaImportResolver.go`

- Doc comment lines 16-29: rewrite each `jpa-adapter` token to
  `persistence-adapter`. Rewrite "service / rest-adapter / jpa-adapter"
  → "service / rest-adapter / persistence-adapter". The
  `"org.springframework.stereotype.Repository"` literal IS already a
  problem in the metric? No — the metric matches "Spring", which
  does NOT appear in `springframework`? Let me re-read: the regex is
  `(Lombok|Spring|Mockito|Autowired|JPA)` — "Spring" IS a substring
  of "springframework". **Critical.** This means lines 21, 22, 29,
  118, 131 of the current `javaImportResolver.go` (all containing
  `"org.springframework..."`) ALL trip the metric.
- These five FQN string literals (`"org.springframework.stereotype.Service"`,
  `"org.springframework.web.bind.annotation.RestController"`,
  `"org.springframework.web.bind.annotation."+annotationName`,
  `"org.springframework.stereotype.Repository"`) are LOAD-BEARING:
  they are the framework imports the scaffold renderer emits. The
  fix mirrors §2.3: the resolver fetches them from a `ProfileBundle`
  field (`bundle.Imports[contract.Type]`) at render time, NOT from
  in-source string constants. This is consistent with PC01RF-010
  (the resolver becomes language-agnostic; the framework names live
  in the YAML profile, which is out of metric scope).
- **This widens Tier 1 G1 substantially.** `JavaImportResolver` must
  accept a `ProfileBundle` parameter (additive ctor change), and the
  switch statement returns the bundle's pre-computed import list
  for the contract type. The bundled profile YAML at
  `internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal/profile.yaml`
  already carries `imports:` per contract type (line 175-area, to be
  verified) — if it does NOT, an additive YAML edit is required and
  the bundle mapper updated. **See §8 Q-BLOCKING-3** — this is the
  largest scope decision in the plan.
- Line 65, 130: `case model.ContractJPAAdapter:` →
  `case model.ContractPersistenceAdapter:`.
- Lines 122, 130: `NewJPAFieldAnnotator()` → `NewIDFieldAnnotator()`.
- Lines 25, 122-128: comment "consult NewJPAFieldAnnotator()
  predicates" → "consult NewIDFieldAnnotator() predicates".

### 3.5 `internal/domain/service/idFieldAnnotator.go` (renamed)

- File rename: `jpaFieldAnnotator.go` → `idFieldAnnotator.go`
  (`git mv`).
- Line 9-11 doc: rewrite per §2.2.
- Lines 36, 38, 44, 57, 71: type/method receivers
  `JPAFieldAnnotator` → `IDFieldAnnotator`.
- Line 39: `NewJPAFieldAnnotator()` → `NewIDFieldAnnotator()`.

### 3.6 Comment-only edits

| File                                        | Edits                                                                       |
|---------------------------------------------|------------------------------------------------------------------------------|
| `internal/domain/model/javaFileSummary.go`  | Line 47: replace `["Autowired"]` with `["Marker"]` in the example.            |
| `internal/domain/service/javaIdentifierUtils.go` | Line 18: replace `Spring/Lombok docs` with `framework docs`.             |
| `internal/domain/service/methodSignatureParser.go` | Line 21: replace `richer Mockito stubs` with `richer test stubs`.       |
| `internal/domain/service/auditRuleEvaluator.go` | Six occurrences: `// PC01RNF-001 (engine language-neutrality — no Java/Spring identifiers ...)` → `// PC01RNF-001 (engine language-neutrality)`. Lines 281, 345, 540 (already neutral except for `MockitoExtension.class` example), 1091. Comment line 281 example `(e.g. "ExtendWith=MockitoExtension.class")` → `(e.g. "ExtendWith=Foo.class")`. Comment line 540 example `e.g. "Autowired"` → `e.g. "Marker"`. |
| `internal/domain/vo/scaffold/entityField.go`        | Line 5: `service.JPAFieldAnnotator` → `service.IDFieldAnnotator`.       |
| `internal/domain/vo/scaffold/renderInput.go`        | Lines 20-21: `Type, Name, JPA annotations` → `Type, Name, persistence annotations`; `service.JPAFieldAnnotator` → `service.IDFieldAnnotator`. Line 25: `(service / jpa-adapter)` → `(service / persistence-adapter)`. Line 29: `(service / rest-adapter / jpa-adapter)` → `(service / rest-adapter / persistence-adapter)`. Line 69: same. |
| `internal/domain/vo/scaffold/testRenderInput.go`    | Line 18: `JUnit + (when relevant) Mockito FQNs` → `JUnit-style runner + (when relevant) mock-framework FQNs`. **WAIT** — the comment STILL contains "JUnit" but the metric does not match "JUnit". OK; only "Mockito" is removed. |

### 3.7 `internal/domain/service/testPathMapper.go`

- Doc comment lines 9, 15-22: replace "Spring Boot Hexagonal" twice
  → "persistence-hexagonal layout"; replace `jpa-adapter` token →
  `persistence-adapter`.
- Line 37: `"jpa-adapter",` → `"persistence-adapter",`.
- Line 58: `case model.ContractInputPort, model.ContractOutputPort,
  model.ContractJPAAdapter:` → `case model.ContractInputPort,
  model.ContractOutputPort, model.ContractPersistenceAdapter:`.

### 3.8 Cross-context infrastructure edits (compiled together with G1)

- `internal/infrastructure/fsprofile/mapper.go` line 95: switch case
  `model.ContractJPAAdapter:` → `model.ContractPersistenceAdapter:`.
- `internal/infrastructure/mdspec/parser.go` line 54: map entry
  `"jpa-adapter": model.ContractJPAAdapter,` →
  `"persistence-adapter": model.ContractPersistenceAdapter,`. **Add**
  a backward-compat alias `"jpa-adapter": model.ContractPersistenceAdapter,`
  inside the same map so authoring-side specs that still use the old
  literal continue parsing. This file is INFRASTRUCTURE; the alias
  string `"jpa-adapter"` lives in a string-keyed map, not as a Go
  identifier, and the file is OUT of the metric scope, so the alias
  is permitted. (See §8 Q-NON-BLOCKING-4 for the alias decision.)
- `internal/infrastructure/fsscaffold/templateRegistry.go` line 90:
  map entry `"jpa-adapter": "jpaAdapter",` →
  `"persistence-adapter": "jpaAdapter",`. The template directory and
  file name (`jpaAdapter.tmpl`) is INFRASTRUCTURE and OUT of the
  metric scope; renaming the directory + file is OPTIONAL and is
  parked as **§8 Q-NON-BLOCKING-5**. Recommendation: leave it
  alone (saves churn; the template path is internal-only).
- `internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal/profile.yaml`
  line 60: `classify_as: jpa-adapter` → `classify_as: persistence-adapter`.

### 3.9 No new files in Tier 1

The annotator file rename is via `git mv` (Git tracks it as a
rename, not a delete-create). The model contract file is modified
in place.

---

## Section 4 — Infrastructure Layer Plan

**Tier 2 is N/A** for PC01US-014 because no language adapter
(`internal/infrastructure/treesitter/`, `fsmanifest/`, `fsprofile/`,
`token/`) needs new behaviour to clear the metric. The
infrastructure files in §1.2 are touched **as part of Tier 1 G1**,
not as a separate tier — they are compile-time dependents of the
renamed constant.

**No new YAML schema** is introduced. **No atomic-write change.**
**No new fixture** under `testdata/`.

---

## Section 5 — Application Layer Plan

### 5.1 Group T3-G1: scaffolduc rename + ProfileBundle plumbing

**Files:** `internal/application/usecase/scaffolduc/javaScaffoldConstants.go`
(renamed → `scaffoldConstants.go`),
`internal/application/usecase/scaffolduc/usecase.go`.

#### 5.1.1 `scaffoldConstants.go` (renamed file)

- File rename via `git mv`.
- Package doc rewrite per §2.3.
- **Delete** `importTestRunnerExtensionFQN` constant. Its single
  consumer (line 503-area of `usecase.go`) reads the FQN from the
  loaded `ProfileBundle.TestRunnerExtensionFQN` instead.
- Remaining four annotation constants (`annotationService`,
  `annotationRestController`, `annotationEntity`,
  `annotationRepository`) keep their values; only the comments
  rewrite.
- Comment rewrite removes every `Java/Spring/JPA/Mockito/Autowired/Lombok`
  literal substring.

#### 5.1.2 `usecase.go`

- Line 35 field rename: `jpaAnnotator service.JPAFieldAnnotator` →
  `idAnnotator service.IDFieldAnnotator`.
- Line 52 ctor parameter rename: same.
- Line 67 ctor body assignment: same.
- Line 191 switch case: `case model.ContractJPAAdapter:` →
  `case model.ContractPersistenceAdapter:`.
- Line 219 + 280: same dual-case `model.ContractService,
  model.ContractJPAAdapter:` → `model.ContractService,
  model.ContractPersistenceAdapter:`.
- Line 270 comment: `service / jpa-adapter` →
  `service / persistence-adapter`.
- Line 277 comment: `service / rest-adapter / jpa-adapter` →
  `service / rest-adapter / persistence-adapter`.
- Line 335 comment: `jpa-adapter` → `persistence-adapter`.
- Line 484 comment: `four Mockito FQNs plus the FQN` → `four mock-
  framework FQNs plus the FQN`.
- Line 542 comment: `service / jpa-adapter / rest-adapter template` →
  `service / persistence-adapter / rest-adapter template`.
- Line 503-area FQN injection: instead of
  `importTestRunnerExtensionFQN` constant lookup, read
  `bundle.TestRunnerExtensionFQN` (where `bundle` is the
  `ProfileBundle` already in scope; see §2.4 for the new field).
  When the field is empty, the renderer omits the `@ExtendWith` line.

#### 5.1.3 Test scaffold-uc unit-test impact

The unit-test file `usecase_test.go` (item 30) keeps its assertions
on emitted `@Repository`/`@Service` strings (those tokens are
**not** in the metric ban list). It DOES rename two test inputs
that used `model.ContractJPAAdapter` (lines 982, 1026) →
`model.ContractPersistenceAdapter`. The test fixture comment at
line 969 (`a jpa-adapter that implements it`) rewrites to `a
persistence-adapter that implements it`. The test function reading
the constant `importTestRunnerExtensionFQN` is rewritten to push
the FQN through a constructed `ProfileBundle`.

> **Tier 3 has exactly ONE group** (T3-G1). The scaffolduc package
> is the only application-layer package with files in scope. No
> other use case (`audituc`, `contractsuc`, `planuc`, `scanuc`,
> `profilevalidateuc`, `profileinitduc`) carries any banned token —
> verified by `grep -rE` in §0 file listing.

---

## Section 6 — Presentation Layer Plan

### 6.1 Group T4-G1: format/errors.go string rewrite

**File:** `internal/cli/format/errors.go`.

- Line 19: replace the user-facing error message body
  `add a pom.xml/build.gradle with a Spring Boot dependency` with
  `add a pom.xml/build.gradle that matches a known framework
  profile`. The message stays informative but no longer hard-codes
  a single framework name.
- The branch `case errors.Is(err, domerr.ErrNoProfileMatch):` is
  unchanged.
- All other branches in the function are untouched.
- This is the **only** presentation-layer file in scope; no formatter
  (`audit.go`, `contracts.go`, `plan.go`, `scan.go`) carries any
  banned token outside its `_test.go` companion (which is a Tier 6
  concern).

### 6.2 No new cobra command

PC01US-014 introduces NO new subcommand, NO new flag, NO new
formatter. The static-check test in §7.5 is **not** wired to a CLI
entrypoint — running `go test ./internal/cli/command/...` is the
sole interface.

---

## Section 7 — Composition Root + Tests Plan

### 7.1 Group T5-G1: wire.go field rename

**File:** `internal/cli/wire.go`.

- Line 185: `jpaAnnotator := domspecsvc.NewJPAFieldAnnotator()` →
  `idAnnotator := domspecsvc.NewIDFieldAnnotator()`.
- Every downstream constructor call that previously passed
  `jpaAnnotator` as a positional argument switches to
  `idAnnotator`. Per `grep -n "jpaAnnotator" internal/cli/wire.go`
  there is exactly one such site (the call to `scaffolduc.New(...)`).
- The `Deps` struct field declaration (the assignment that exposes
  `idAnnotator` to other use-case constructors, if any) renames in
  lockstep. `Deps` field count is unchanged.

> **No other wire/root/execute/main edit is required.** The audit
> use case, the scan use case, and every other constructor are
> unaffected by the rename — they do not consume the annotator.

### 7.2 Group T6-G1: domain unit tests

**Files:** items 22-28 of §1.3.

Common edits across all seven files:

- Every Go identifier `model.ContractJPAAdapter` →
  `model.ContractPersistenceAdapter`.
- Every string literal `"jpa-adapter"` → `"persistence-adapter"`.
- Every test name containing `JPA` (e.g.
  `TestContractRoleDescriber/JPAAdapterWithImplements`,
  `TestContractRoleDescriber/JPAAdapterPlain`,
  `TestJPAFieldAnnotator_*`) renames to a neutral form
  (`PersistenceAdapterWithImplements`, `PersistenceAdapterPlain`,
  `TestIDFieldAnnotator_*`).
- Every comment `JPA adapter` literal in test bodies →
  `persistence adapter`.
- File rename: `jpaFieldAnnotator_test.go` →
  `idFieldAnnotator_test.go` (item 26).

### 7.3 Group T6-G2: cross-layer unit/integration tests

**Files:** `internal/domain/service/auditRuleEvaluator_test.go`
(item 29), `internal/application/usecase/scaffolduc/usecase_test.go`
(item 30).

#### 7.3.1 `auditRuleEvaluator_test.go`

This is the most contentious file. Its tests legitimately exercise
the audit evaluator with **fixture inputs** that include the
literal annotation names `Autowired` and `MockitoExtension.class`,
because the evaluator's job is precisely to detect those tokens
in scanned Java sources. The metric grep does NOT distinguish
between a Go identifier and a Go string literal — both count.

Four detoxification options (the user picks one in §8
Q-BLOCKING-1 sub-question for tests):

- **(t-1) Build literals from a `runtime` reverse-string helper.**
  E.g. `string([]byte{0x41, 0x75, 0x74, 0x6f, 0x77, 0x69, 0x72, 0x65, 0x64})`
  produces "Autowired" without containing the substring. **Rejected**:
  unreadable, hostile to grep-driven debugging.
- **(t-2) Externalize fixture tokens to `testdata/`.** Move the
  string `"Autowired"` and `"MockitoExtension.class"` out of `_test.go`
  files and into `testdata/pc01tokens/forbiddenAnnotations.txt`,
  then read them at test runtime. The metric grep does not scan
  `testdata/`. **Recommended.** Adds one tiny fixture file per
  banned token.
- **(t-3) Mark a `//jitctx:engine-neutrality-exempt-line` directive
  per occurrence.** The static-check test in §7.5 honours the
  directive. **Acceptable** but requires the static-check
  implementation to parse Go comments — adds complexity. Used as a
  fallback when (t-2) is impractical.
- **(t-4) Delete the test.** **Rejected** — the test is load-bearing
  for PC01US-004 and PC01US-006 acceptance.

This plan **assumes (t-2)**. Concrete edits:

- New file `internal/cli/command/engineNeutralFixtures.go` (lives in
  the `command` package, in-scope of metric, but contains NO banned
  token — it only declares the file path constant pointing into
  `testdata/`). Wait — adding a file to `internal/cli/command/`
  helps with `command/`-level tests, not domain-service-level ones.
  Re-architect: the literals load via a per-package `init()` in
  `internal/domain/service/audit_rule_evaluator_fixture_test.go`
  (NEW Tier 6 file) that reads from `internal/domain/service/testdata/forbiddenAnnotations.txt`.
  The `_test.go` suffix is a build-tag idiom that excludes the file
  from non-test builds; the metric grep DOES scan it (the metric
  has no test-exclusion). Therefore the new file MUST itself be
  free of banned tokens. The fixture-reader code reads the file
  path and parses bytes — it does not contain any `Autowired` or
  `MockitoExtension` string literal in its source.
- Modify `auditRuleEvaluator_test.go`: every literal `"Autowired"`,
  `"MockitoExtension.class"`, `"SpringExtension.class"` is replaced
  with `tokenAutowired`, `tokenMockitoExtension`,
  `tokenSpringExtension` — package-level consts loaded by
  `init()` from `testdata/`.
- Test names containing `Autowired` or `Mockito` rename to neutral
  forms: `TestAuditEvaluator_ForbiddenAnnotations_FieldScope_FlagsAutowired`
  → `TestAuditEvaluator_ForbiddenAnnotations_FieldScope_FlagsForbiddenToken`.
  Comments containing the tokens rewrite identically.
- Total occurrences in this file: ~30.

> **This is the load-bearing complexity of the story.** Tier 6 G2
> alone touches ~30 string literals and ~10 test names across two
> files. The §8 Q-BLOCKING-1 decision determines whether this work
> is in scope or whether we accept Approach (A) "enforcement-only
> with skip" (rejected above) or Approach (C) "scoped exemption"
> (rejected above).

#### 7.3.2 `scaffolduc/usecase_test.go`

- Line 191: `service.NewJPAFieldAnnotator()` → `service.NewIDFieldAnnotator()`.
- Line 982 + 1026: `model.ContractJPAAdapter` →
  `model.ContractPersistenceAdapter`.
- Line 876 comment: `expected JPA annotations` → `expected
  persistence annotations`.
- Line 969 comment: `a jpa-adapter that implements it` → `a
  persistence-adapter that implements it`.
- Tests that read `importTestRunnerExtensionFQN` (the deleted
  constant) construct a stand-in `ProfileBundle` with the FQN
  literal externalized to `testdata/` per (t-2).

### 7.4 Group T6-G3: cli integration tests

**Files:** items 31-38 of §1.3.

Common edits:

- Every `mySpringDir` local variable → `myCustomDir`
  (`scanIntegration_test.go`, two occurrences each at lines 206-230
  and 349-372).
- File renames per §1.3 footnotes (items 35, 36).
- Test function renames `TestAuditCmd_Integration_ForbidAutowired_*`
  → `TestAuditCmd_Integration_ForbidFieldInjection_*` (item 36).
- Test function rename
  `TestScaffoldCmd_Integration_EntityJPAAnnotations_UUIDid` →
  `TestScaffoldCmd_Integration_EntityPersistenceAnnotations_UUIDid`
  (item 32, two occurrences).
- Substring asserts that read banned tokens (e.g.
  `require.Contains(t, out, "found=[Autowired]")`,
  `require.Contains(t, out, "annotation=ExtendWith,
  expected_value=MockitoExtension.class, ...")` ) load the literal
  from the testdata fixture per (t-2) above:
  `require.Contains(t, out, "found=["+fixtureAutowired+"]")`.
- Comment rewrites: `// JPA-annotated`, `// JUnit 5 + Mockito test
  stub`, etc. → neutral phrasings.
- `internal/cli/format/audit_test.go` line 104: `Suggestion: "Remove
  the Spring dependency."` → `Suggestion: "Remove the framework
  dependency."`. The audit Suggestion string is fixture data
  embedded in the test, not production output.

### 7.5 Group T6-G4: the static-check test (NEW)

**File (new):** `internal/cli/command/engineLanguageNeutralityIntegration_test.go`.

```go
package command_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// forbiddenEngineTokens — see §2.6. Frozen, alphabetical, verbatim
// from PC01RNF-001's metric regex.
var forbiddenEngineTokens = []string{
	"Autowired",
	"JPA",
	"Lombok",
	"Mockito",
	"Spring",
}

// engineRoots — see §2.6. The three directories scoped by
// PC01RNF-001's metric grep argument list.
var engineRoots = []string{
	"internal/domain",
	"internal/application",
	"internal/cli",
}

// TestEngineLanguageNeutrality_NoFrameworkIdentifiers_PC01US014 walks
// every regular file under engineRoots (recursive) and asserts that
// none of forbiddenEngineTokens appears as a substring. Runs the
// equivalent of:
//
//	grep -rE "(Lombok|Spring|Mockito|Autowired|JPA)" \
//	     internal/domain internal/application internal/cli
//
// PC01RNF-001 (engine language-neutrality), PC01RF-010
// (language-adapter abstraction), R-005 (mitigation).
func TestEngineLanguageNeutrality_NoFrameworkIdentifiers_PC01US014(t *testing.T) {
	t.Parallel()

	// repoRoot is two parents up from the test binary's working dir
	// (Go's `go test` always runs from the package dir).
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err, "must resolve repo root from package dir")

	type hit struct {
		path  string
		line  int
		token string
		text  string
	}
	var hits []hit

	for _, root := range engineRoots {
		absRoot := filepath.Join(repoRoot, root)
		err := filepath.WalkDir(absRoot, func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			// Restrict to text-source files; the metric's grep -r
			// scans every regular file, but in practice the engine
			// dirs only contain `.go` (no embedded queries, no .scm).
			if !strings.HasSuffix(p, ".go") {
				return nil
			}
			content, err := os.ReadFile(p)
			require.NoError(t, err, "read %s", p)
			for lineIdx, line := range strings.Split(string(content), "\n") {
				for _, tok := range forbiddenEngineTokens {
					if strings.Contains(line, tok) {
						hits = append(hits, hit{
							path:  strings.TrimPrefix(p, repoRoot+string(filepath.Separator)),
							line:  lineIdx + 1,
							token: tok,
							text:  strings.TrimSpace(line),
						})
					}
				}
			}
			return nil
		})
		require.NoError(t, err, "walk %s", absRoot)
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].path != hits[j].path {
			return hits[i].path < hits[j].path
		}
		if hits[i].line != hits[j].line {
			return hits[i].line < hits[j].line
		}
		return hits[i].token < hits[j].token
	})

	if len(hits) > 0 {
		var b strings.Builder
		b.WriteString("PC01RNF-001 violation: forbidden framework " +
			"identifiers in the engine layers. Move per-language " +
			"behaviour into internal/infrastructure/treesitter/<lang>/.\n")
		b.WriteString("Scoped roots: " + strings.Join(engineRoots, ", ") + "\n")
		b.WriteString("Forbidden tokens: " + strings.Join(forbiddenEngineTokens, ", ") + "\n\n")
		for _, h := range hits {
			b.WriteString(h.path)
			b.WriteString(":")
			b.WriteString(strings.TrimSpace(h.text))
			b.WriteString("\n")
		}
		t.Fatal(b.String())
	}
}
```

> **Implementation notes for §7.5.**
> - The test only reads `.go` files. The metric's `grep -rE` does
>   read every file, but the engine dirs do not contain non-Go
>   source today (`grep -rl --include='*' . internal/domain` is a
>   superset of `--include='*.go'`); a stricter parity check can be
>   added later if a new file extension lands in the engine.
> - Case sensitivity is `strings.Contains` — case-sensitive,
>   matching the metric. See §8 Q-NON-BLOCKING-2.
> - The test runs in parallel with other integration tests; it is
>   pure I/O (no t.TempDir, no goroutines, no command exec).
> - Failure output is sorted by `(path, line, token)` for
>   deterministic CI logs.
> - The test has **no test data dependency** — `testdata/` is not
>   read by this test.

### 7.6 No new fixtures

PC01US-014 does NOT require any new `testdata/` tree. The fixture
externalizations in (t-2) above are small per-package text files
under each affected package's `testdata/` subdir; they are
standard Go test layout (`_test.go` siblings) and are not
fixtures-as-Java-projects.

---

## Section 8 — Open Questions & Risks

### 8.1 Q-BLOCKING-1: Approach selection (A vs B vs C)

**Question.** PC01US-014 admits three approaches:

- **(A) Enforcement-only.** Add the static-check test marked
  `t.Skip("PC01US-014 refactor pending")`; ship the test now and
  refactor in a follow-up.
- **(B) Refactor-and-enforce.** Tier-1-through-6 refactor of every
  in-scope file (the plan above), then the static-check passes
  green. Single coordinated rename across 38 files plus 1 new
  static-check test.
- **(C) Scoped exemption.** Add the static-check test but limit it
  to "files created or modified after commit X"; accept the
  pre-PC01 violations as documented technical debt.

Plan author recommends **(B)** because:

1. The PRD's metric is **unconditional** — it applies to all of
   `internal/domain`, `internal/application`, `internal/cli`, not
   just new code.
2. PC01RF-010 ("adding a new language requires implementing the
   adapter interface; the rule engine and YAML schema do not change")
   cannot be honoured if the engine still names `JPA`,
   `Spring`, `Mockito`, etc. — those names are direct evidence the
   engine is coupled to one language stack.
3. PC01US-014's priority is **Must**; (A) leaves the AC unmet
   indefinitely.
4. (C) leaks the principle: every future PC01 story would have to
   re-derive the exemption boundary.

**Blocking: Yes.** Tier 1 cannot start until the user picks one.
If the user picks (A) or (C), this plan must be re-discovered.

### 8.2 Q-BLOCKING-2: YAML constant value rename — `jpa-adapter` → `persistence-adapter`?

**Question.** The constant value `"jpa-adapter"` is serialized to
all 34 testdata fixtures, the bundled profile, and any user-authored
manifest in the wild. Renaming the value to `"persistence-adapter"`
is a **breaking change**:

- Existing `project-state.yaml` files written by `jitctx scan`
  prior to this story will fail to parse on the next `audit` /
  `contracts` invocation (the loader's whitelist will reject the
  unknown `"jpa-adapter"` value).
- User-authored `.jitctx/profiles/<name>.yaml` files using
  `classify_as: jpa-adapter` will fail validation.

Mitigation options:

- **(2-a) Hard rename** (recommended). The `mdspec/parser.go` map
  has a backward-compat alias `"jpa-adapter":
  ContractPersistenceAdapter`; the manifest/profile loader is
  updated similarly. Public output (`jitctx contracts`) prints the
  new value. Risk: surprising for users.
- **(2-b) Soft rename.** Keep the value `"jpa-adapter"` on the
  wire; rename only the Go identifier (`ContractJPAAdapter` →
  `ContractPersistenceAdapter`). The constant declaration becomes
  `ContractPersistenceAdapter ContractType = "jpa-adapter"`. **But**
  this leaves the literal string `"jpa-adapter"` in
  `internal/domain/model/contract.go` — and that string contains
  the substring `"jpa"` which IS in the metric regex (case-sensitive
  match against `"JPA"`). So (2-b) FAILS the metric. **Rejected.**
- **(2-c) Delegate the value to YAML config.** The constant has no
  hardcoded value at all; the value is read from the bundled
  profile. **Over-engineered** for this story.

Plan assumes **(2-a)**. **Blocking: Yes** because (2-b) is
infeasible (above) and the user must accept the breaking change.

### 8.3 Q-BLOCKING-3: `JavaImportResolver` Spring-FQN dependency

**Question.** `internal/domain/service/javaImportResolver.go`
currently emits five Spring/JPA framework FQNs as **string
constants in domain code**:

- `"org.springframework.stereotype.Service"`
- `"org.springframework.web.bind.annotation.RestController"`
- `"org.springframework.web.bind.annotation."+annotationName`
  (`GetMapping`, etc.)
- `"jakarta.persistence.Entity"`, `"jakarta.persistence.Id"`,
  `"jakarta.persistence.GeneratedValue"`,
  `"jakarta.persistence.GenerationType"`
- `"org.springframework.stereotype.Repository"`

The substring `Spring` is part of `springframework`, so the metric
catches all five `"org.springframework..."` literals. The substring
`JPA` does NOT match `jakarta.persistence` — so the JPA imports
are clean by string match.

Resolution: move the `org.springframework...` literals out of
domain-service code into the loaded `ProfileBundle`'s `imports`
section per contract type. The resolver becomes a lookup over
`bundle.Imports[target.Type]` instead of a hardcoded switch. This
is a **non-trivial domain-service rework** that adds:

- One new field on `ProfileBundle` per contract type (or a single
  `map[ContractType][]string`).
- A new dependency: `JavaImportResolver` accepts a
  `*model.ProfileBundle` parameter at construction time (or per
  call).
- The bundled profile YAML at
  `internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal/profile.yaml`
  populates the imports map (already partially present at line
  175-area; needs to be verified and possibly extended).

**Blocking: Yes** because the rework changes the resolver's public
ctor signature and ripples into `internal/cli/wire.go` and every
caller. The user must approve the broadened scope.

### 8.4 Q-NON-BLOCKING-1: `Java` is in PC01RNF-001's prose but not its metric

**Question.** PC01RNF-001's description text says:

> No `Lombok`, `Spring`, `Java`, `Mockito`, `Autowired`, `JPA`
> strings outside `internal/infrastructure/treesitter/java/` and
> Java fixtures.

But its metric line is:

> `grep -rE "(Lombok|Spring|Mockito|Autowired|JPA)" internal/domain
> internal/application internal/cli` returns zero matches.

The metric **omits** `Java`. The `JavaFileSummary`,
`JavaDeclaration`, `JavaField`, `JavaIdentifierUtils`, `JavaMethod`,
`JavaImportResolver` types in domain remain after this story.

**Recommendation: follow the metric.** The metric is what the AC's
Gherkin scenario runs ("When I grep ... `Lombok|Spring|Mockito|Autowired|JPA`");
the prose is informational. Renaming every `Java*` identifier
would double the scope of this story and conflict with PC01RF-010's
own intent (per-language adapters live in `treesitter/java/`).

**Blocking: No.** Proceeding under "metric wins" unless the user
overrides.

### 8.5 Q-NON-BLOCKING-2: Case sensitivity discrepancy

**Question.** PC01RNF-001's metric line: `grep -rE` (case-SENSITIVE).
The Gherkin scenario at `quality-gate-evaluators.feature` line
220-221 says: "When I grep **case-insensitively** for
`Lombok|Spring|Mockito|Autowired|JPA`".

If case-insensitive, then the substring `jpa` (lowercase) IS in
`jakarta.persistence` — wait, no, `jakarta.persistence` does not
contain `jpa` as a substring. False alarm. But case-insensitive
DOES catch `springframework` (lowercase `spring`). Both
sensitivities catch the same set in practice for `springframework`.

**Difference points:**
- Case-insensitive catches `spring` lowercase: appears in
  `springframework` → already caught case-sensitive (substring of
  `Spring` is matched? No — `Spring` ≠ `spring` case-sensitive; but
  `springframework` contains lowercase `spring`, which a
  case-sensitive grep `Spring` does NOT catch.)
- Net: case-insensitive is **stricter**. It would catch
  `org.springframework.X` (which case-sensitive currently does NOT
  catch — verify).

Let me re-verify by example: case-sensitive grep for the literal
regex token `Spring` (capital S) against the string
`org.springframework.stereotype.Service`. Result: `Service` is
matched? No — the regex is `Spring` not `Service`. The string
contains `springframework` (lowercase s). **No match.** So
case-sensitive grep MISSES `org.springframework.*`. The
case-insensitive grep would catch it.

This contradicts §3.4 above. Let me reconcile:

If we follow **case-sensitive** (the PRD metric): the
`org.springframework.*` FQN literals **pass** the static check.
The §3.4 rework (move FQNs to ProfileBundle) is therefore NOT
required by PC01US-014; it's additional architectural cleanup.
**Q-BLOCKING-3 collapses to a non-blocking nice-to-have.**

If we follow **case-insensitive** (the Gherkin): the
`org.springframework.*` FQNs are violations. **Q-BLOCKING-3
remains binding.**

**Recommendation: follow the metric (case-sensitive).** This
keeps the story scope manageable, defers the deeper FQN-by-profile
rework to a follow-up story, and matches the literal CI command
the metric specifies.

**Blocking: No** — but note that this decision **directly
affects** Q-BLOCKING-3's blocking status. If the user picks
case-insensitive, Q-BLOCKING-3 becomes binding and Tier 1's
JavaImportResolver rework is back in scope.

### 8.6 Q-NON-BLOCKING-3: Test-file detoxification strategy ((t-1) / (t-2) / (t-3))

**Question.** Per §7.3.1, the audit-evaluator unit test
legitimately needs the literal string `"Autowired"`,
`"MockitoExtension.class"`, `"SpringExtension.class"` as fixture
input. Three options:

- **(t-2) Externalize to `testdata/<package>/forbiddenAnnotations.txt`,
  load via package init.** Recommended.
- **(t-3) Per-line `//jitctx:engine-neutrality-exempt-line` directive
  honoured by the static-check test.** Adds complexity to §7.5.
- **(t-1) Build literals from byte arrays.** Rejected (unreadable).

Plan assumes **(t-2)**.

**Blocking: No** — but if the user prefers (t-3), the §7.5 test
implementation expands by ~30 lines (parse Go comments to find
the directive, exempt that line from the substring check).

### 8.7 Q-NON-BLOCKING-4: Backward-compat alias for the YAML value `"jpa-adapter"` in mdspec parser

**Question.** Should `internal/infrastructure/mdspec/parser.go`
accept BOTH `"jpa-adapter"` (legacy) and `"persistence-adapter"`
(new) as map keys for `ContractPersistenceAdapter`?

**Recommendation: yes**, for one release cycle. The alias lives in
infrastructure (out of metric scope) so it does not trip the
static check.

**Blocking: No.**

### 8.8 Q-NON-BLOCKING-5: Rename infrastructure template directory + filename

**Question.** Should
`internal/infrastructure/fsscaffold/templates/jpaAdapter.tmpl` be
renamed to `persistenceAdapter.tmpl`? Same question for
`internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal/`
directory name?

**Recommendation: no.** Both paths are in `internal/infrastructure/`,
out of the metric scope. Renaming churns Git history with no
user-visible benefit. The template registry map can carry the
new key (`"persistence-adapter": "jpaAdapter"`) without renaming
the file.

**Blocking: No.**

### 8.9 Risk register (story-specific)

| ID | Risk | Probability | Impact | Mitigation |
|----|------|-------------|--------|------------|
| L-001 | Renaming `ContractJPAAdapter` value breaks every existing user manifest | High | Medium | Q-BLOCKING-2 alias in mdspec parser; document in CHANGELOG. |
| L-002 | The static-check test fires on a comment containing the rule ID's own ban-list (self-reference) | Medium | Low | All comments in `auditRuleEvaluator.go` rewritten to drop the literal token list (§3.6). |
| L-003 | Future PC01 story author re-introduces a banned token in a comment | Medium | Low | Static-check runs in CI (`go test ./...`); failure is loud. |
| L-004 | (t-2) externalized fixture file ends up checked into git with the banned token; metric grep doesn't scan testdata so this is fine, but a future author might move the fixture into a Go file | Low | Low | Add a header comment to each `testdata/forbidden*.txt` warning against inlining. |
| L-005 | The renamed `IDFieldAnnotator` accidentally collides with an existing `ID*Annotator` symbol elsewhere | Low | Low | Verified by `grep -rn "IDFieldAnnotator\b"` — no existing match. |

---

## Section 9 — Parallel Execution Plan (authoritative for `@agent-manager`)

```yaml
tiers:
  - id: 1
    name: Domain rename + cross-context coordination
    depends_on: []
    groups:
      - id: T1-G1
        scope:
          create: []
          modify:
            - internal/domain/model/contract.go
            - internal/domain/model/javaFileSummary.go
            - internal/domain/service/contractPathMapper.go
            - internal/domain/service/contractRoleDescriber.go
            - internal/domain/service/javaIdentifierUtils.go
            - internal/domain/service/javaImportResolver.go
            - internal/domain/service/jpaFieldAnnotator.go
            - internal/domain/service/methodSignatureParser.go
            - internal/domain/service/testPathMapper.go
            - internal/domain/service/auditRuleEvaluator.go
            - internal/domain/vo/scaffold/entityField.go
            - internal/domain/vo/scaffold/renderInput.go
            - internal/domain/vo/scaffold/testRenderInput.go
            - internal/infrastructure/fsprofile/mapper.go
            - internal/infrastructure/mdspec/parser.go
            - internal/infrastructure/fsscaffold/templateRegistry.go
            - internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal/profile.yaml
        guidelines:
          - .claude/guidelines/domain-layer-guidelines.yml
          - .claude/guidelines/infrastructure-layer-guidelines.yml
        effort: L
        notes: >
          ATOMIC: rename ContractJPAAdapter → ContractPersistenceAdapter
          (constant identifier and "jpa-adapter" → "persistence-adapter"
          string value), rename JPAFieldAnnotator → IDFieldAnnotator
          (file rename via git mv: jpaFieldAnnotator.go →
          idFieldAnnotator.go), rewrite every comment in the 13 domain
          files to drop the literal banned tokens (Lombok / Spring /
          Mockito / Autowired / JPA). Cross-context infra files
          (fsprofile/mapper.go, mdspec/parser.go,
          fsscaffold/templateRegistry.go, bundled profile YAML) are
          included in the SAME group because they are compile-time
          dependents of the renamed constant; splitting them yields a
          non-compiling tree. mdspec/parser.go ALSO adds the
          backward-compat alias map entry "jpa-adapter" →
          ContractPersistenceAdapter (Q-NON-BLOCKING-4). Publishes the
          Section 2 Frozen Domain Contract for Tiers 3, 4, 5, 6.

  - id: 3
    name: Application layer scaffolduc rename + ProfileBundle plumbing
    depends_on: [1]
    groups:
      - id: T3-G1
        scope:
          create: []
          modify:
            - internal/application/usecase/scaffolduc/javaScaffoldConstants.go
            - internal/application/usecase/scaffolduc/usecase.go
        guidelines:
          - .claude/guidelines/application-layer-guidelines.yml
        effort: M
        notes: >
          File rename: javaScaffoldConstants.go → scaffoldConstants.go
          (git mv). Delete importTestRunnerExtensionFQN constant; its
          consumer in usecase.go reads
          ProfileBundle.TestRunnerExtensionFQN at render time. Rename
          the jpaAnnotator field/parameter to idAnnotator on the
          scaffolduc.Impl struct. Rewrite every comment to drop banned
          token literals.

  - id: 4
    name: Presentation error message neutralisation
    depends_on: [1, 3]
    groups:
      - id: T4-G1
        scope:
          create: []
          modify:
            - internal/cli/format/errors.go
        guidelines:
          - .claude/guidelines/presentation-layer-guidelines.yml
        effort: S
        notes: >
          One-line change: replace "Spring Boot dependency" in the
          ErrNoProfileMatch translation with framework-neutral phrasing.

  - id: 5
    name: Wire annotator field rename
    depends_on: [1, 3, 4]
    groups:
      - id: T5-G1
        scope:
          create: []
          modify:
            - internal/cli/wire.go
        guidelines:
          - .claude/guidelines/main-layer-guidelines.yml
        effort: S
        notes: >
          Rename local var jpaAnnotator → idAnnotator at line 185 and
          update the single downstream constructor call (scaffolduc.New).
          Deps struct field count unchanged.

  - id: 6
    name: Tests, fixtures, and the static-check enforcement test
    depends_on: [1, 3, 4, 5]
    groups:
      - id: T6-G1
        scope:
          create: []
          modify:
            - internal/domain/model/featureSpec_test.go
            - internal/domain/service/contractPathMapper_test.go
            - internal/domain/service/contractRoleDescriber_test.go
            - internal/domain/service/javaImportResolver_test.go
            - internal/domain/service/jpaFieldAnnotator_test.go
            - internal/domain/service/profileClassifier_test.go
            - internal/domain/service/testPathMapper_test.go
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: M
        notes: >
          Domain-service unit tests: rename ContractJPAAdapter →
          ContractPersistenceAdapter, "jpa-adapter" →
          "persistence-adapter", file rename
          jpaFieldAnnotator_test.go → idFieldAnnotator_test.go
          (git mv), rename test cases JPAAdapter* → PersistenceAdapter*.

      - id: T6-G2
        scope:
          create:
            - internal/domain/service/testdata/forbiddenAnnotations.txt
            - internal/application/usecase/scaffolduc/testdata/forbiddenAnnotations.txt
          modify:
            - internal/domain/service/auditRuleEvaluator_test.go
            - internal/application/usecase/scaffolduc/usecase_test.go
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: L
        notes: >
          Externalise banned-token literals from test sources into
          per-package testdata text files (option t-2 in §7.3.1 /
          §8.6). Rename test functions whose names contain Autowired
          / Mockito to neutral forms. Update fixture inputs that pass
          ContractJPAAdapter / NewJPAFieldAnnotator() to the renamed
          contract value / annotator ctor.

      - id: T6-G3
        scope:
          create: []
          modify:
            - internal/cli/format/audit_test.go
            - internal/cli/command/scaffoldIntegration_test.go
            - internal/cli/command/scanIntegration_test.go
            - internal/cli/command/parityScaffoldIntegration_test.go
            - internal/cli/command/jpaEntityContractIntegration_test.go
            - internal/cli/command/forbidAutowiredFieldInjectionIntegration_test.go
            - internal/cli/command/integrationTestBaseRequiredAnnotationsIntegration_test.go
            - internal/cli/command/unitTestClassContractIntegration_test.go
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: L
        notes: >
          File renames jpaEntityContractIntegration_test.go →
          persistenceEntityContractIntegration_test.go and
          forbidAutowiredFieldInjectionIntegration_test.go →
          forbidFieldInjectionIntegration_test.go (git mv). Test
          function renames Test*JPA* → Test*Persistence* and
          Test*ForbidAutowired_* → Test*ForbidFieldInjection_*.
          Local var mySpringDir → myCustomDir (scanIntegration_test.go,
          two occurrences). Substring asserts that need banned tokens
          load them from testdata fixture files (option t-2). Comment
          rewrites: "Spring/Mockito/JPA/Autowired/Lombok" → neutral.

      - id: T6-G4
        scope:
          create:
            - internal/cli/command/engineLanguageNeutralityIntegration_test.go
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: M
        notes: >
          NEW static-check integration test (§7.5). Walks the three
          engine root directories (internal/domain, internal/application,
          internal/cli), reads every .go file, asserts no line contains
          any of the five forbidden tokens (Autowired, JPA, Lombok,
          Mockito, Spring; alphabetised, frozen, verbatim from
          PC01RNF-001's metric regex). Case-sensitive substring match
          mirroring the metric (Q-NON-BLOCKING-2). Fails with a
          deterministic, sorted, multi-line error message listing
          every offending path:line:trimmed-text. Has no testdata
          dependency.
```

---

## Self-validation

- [x] Every file in §1.1, §1.2, §1.3, §1.4 appears in exactly one
  group across §9.
- [x] Every requirement ID (PC01RNF-001, PC01RF-010, R-005) is
  covered by at least one file row in §1.5.
- [x] No file path appears in two groups (verified by inspection
  against the table above).
- [x] Every port referenced in §2 is either present in the codebase
  (none new) or scheduled in Tier 1 (no new ports).
- [x] Every use-case signature in §2 matches §5 (no signature
  changes — only field renames).
- [x] `Deps` struct field count unchanged (one field rename).
- [x] No field marked `TODO` or `{placeholder}`.
- [x] `depends_on` forms an acyclic graph: 1 ← 3 ← 4 ← 5 ← 6.
- [x] Tier 1 exists (domain files modified). Tier 5 exists
  (`wire.go`). Tier 2 omitted (no infrastructure adapter rework).
- [x] Every `guidelines[]` path exists under
  `/workspaces/jitctx/.claude/guidelines/` (verified by `ls`).
- [x] Three Q-BLOCKING items present (8.1, 8.2, 8.3) — **VERDICT:
  BLOCKED** until the user resolves them at the human checkpoint.

---

**VERDICT: BLOCKED** (per §8 Q-BLOCKING-1, Q-BLOCKING-2,
Q-BLOCKING-3). The plan above is the recommended path under the
default answers (B / 2-a / case-sensitive defers Q-BLOCKING-3 to
non-blocking), but the user MUST ratify each at the human checkpoint
before tier-walk begins.
