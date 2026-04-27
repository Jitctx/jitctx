# Code review — EP04US-005 (Tree-sitter Queries Bundled by Language)

Reviewer: QA Coordinator (code-review pillar)
Date: 2026-04-27
Requirements: `docs/ep04/epic-04-requirements.md` (US-005, lines 388-416)
Plan: `.claude/plans/ep04us005/plan.md`
Scope: 22 files (13 production + 8 tests + 1 .scm fixture + 3 testdata YAML)
Cycles: 1 (B-001 remediated in QA cycle 1)

## Acceptance scenarios

| # | Scenario                                                          | Coverage                                                                                          |
|---|-------------------------------------------------------------------|---------------------------------------------------------------------------------------------------|
| 1 | Profile declares language → bundled queries activated             | `TestBundleLoader_LoadsLanguageJava_AttachesBundledQueries` (also asserts no `queries/` subdir)   |
| 2 | Unsupported language → fails with `language 'cobol' is not supported; available: java` | `TestBundleLoader_UnsupportedLanguage_FailsWithListing` + `TestLoadLanguageQueries_UnknownLanguage_ReturnsUnsupportedError` |
| 3 | Two profiles same language share `*model.LanguageQuerySet` pointer | `TestBundleLoader_TwoProfilesSameLanguage_ShareQuerySet` + `TestLoadLanguageQueries_Java_ReturnsCachedSet` |

All three scenarios pass. Pinned literal error string matches the .feature.

## Findings

### B-001 — Package name `bundledQueries` violates infrastructure naming convention  [RESOLVED]
**Severity:** BLOCKER (resolved in QA cycle 1)
**Files changed:**
- Renamed directory `internal/infrastructure/treesitter/bundledQueries/` → `bundledqueries/`
- Updated `package` declarations: `bundled.go`, `registry.go`, `registry_test.go`
- Updated import path in `internal/cli/wire.go:38` and doc comment at `wire.go:114`
- Updated import path in `internal/infrastructure/fsprofile/bundleLoader_test.go:19`

**Reference:** `.claude/guidelines/infrastructure-layer-guidelines.yml` —
`naming_conventions.package: 'fs{domain} (fsmanifest, fsprofile) or {tech} (treesitter, token)'`

**Verification:**
- `go build ./...`  : clean
- `go vet ./...`    : clean
- `gofmt -l .`      : clean
- `go test ./...`   : 26/26 packages PASS
- `go test ./... -run Integration -count=1` : 26/26 packages PASS

The package name is now lowercase single-word `bundledqueries`, matching
the convention used by every other infrastructure subpackage in the repo.

### W-001 — Duplicate embed-load flow between `Bundled.LoadBundled` and `BundleLoader.loadFromBundled`
**Severity:** WARNING (non-blocking)
**Files:** `internal/infrastructure/fsprofile/bundled.go:31-56` vs
`internal/infrastructure/fsprofile/bundleLoader.go:95-119`
**Finding:** The two functions duplicate name validation, `fs.Sub`,
`fs.Stat("profile.yaml")`, and the post-`loadFromFS` Source/Dir assignment.
The only material difference is that `BundleLoader.loadFromBundled` then
calls `attachLanguageQueries`. The duplication was introduced because
`loadFromFS` was widened to return `(bundle, dto, err)` so the bundle
loader can read the verbatim `language:` value, but `Bundled.LoadBundled`
discards the dto with `_`.

**Why this matters:** `Bundled` and `BundleLoader` both implement
embed-backed loading, but `Bundled.LoadBundled` does NOT attach language
queries. So two profiles loaded through different code paths (resolver →
`loadFromBundled` vs `profile init` → `Bundled.LoadBundled`) end up with
different `bundle.LanguageQueries` populations for the same on-disk YAML.
This is subtle and silently surprising.

**Suggested fix (Tier 2, deferable):** Have `Bundled.LoadBundled` delegate
to a thin internal helper (`loadEmbeddedByName`) that both adapters share,
then layer query attachment on top in `BundleLoader.loadFromBundled` only.
Or: have `Bundled` accept an optional `parserport.LoadLanguageQueriesPort`
and apply the same attachment policy. **Not blocking US-005**, but should
be tracked for follow-up to prevent divergent behaviour bugs.

### W-002 — Coarse-grained mutex in `Registry.LoadLanguageQueries`
**Severity:** WARNING (non-blocking)
**File:** `internal/infrastructure/treesitter/bundledqueries/registry.go:81`
**Finding:** The mutex is held across `fs.ReadDir` and `fs.ReadFile`,
serialising loads across distinct languages even though the cache is keyed
per-language. For a CLI this is fine (load-once, serial usage), but the
idiomatic pattern in Go for "compute-once-per-key" is `sync.Map` with
`LoadOrStore` of a `sync.Once`-wrapped lazy value, or a per-key mutex map.
**Not blocking** — the comment in `registry.go:25` documents the
load-bearing invariant ("SAME pointer"), and the mutex is sufficient to
maintain it. Future-proofing concern only.

### W-003 — Language registration is implicit (filesystem-as-config)
**Severity:** WARNING (informational)
**File:** `internal/infrastructure/treesitter/bundledqueries/registry.go:47-65`
**Finding:** Supported languages are derived by reading the embed root at
construction time. A typo in a directory name (e.g. `Java/` capitalised)
silently registers a "language" with that name. There's no manifest
asserting which languages should be present.

**Suggested fix (Tier 3, optional):** Add a `languages.yml` or a Go-level
`var registered = []vo.Language{LanguageJava}` next to the embed and
cross-check at construction time. Not required for US-005 but worth a
follow-up ticket once US-006/US-007 add Go/TypeScript/Python.

### I-001 — `LanguageQuerySet.Queries` map iteration order is undefined
**Severity:** INFO
**File:** `internal/domain/model/languageQuerySet.go:22`
**Finding:** Doc-comment says "Iteration order is undefined; callers that
need deterministic iteration sort the keys themselves." This is accurate;
no action required. Future consumers (parser refactor) need to honour this
contract.

### I-002 — `vo.ParseLanguage` knows about `go`, `python`, `typescript` but only `java` is bundled
**Severity:** INFO
**File:** `internal/domain/vo/language.go:8-13`
**Finding:** The VO recognises four ids (`go`, `java`, `typescript`,
`python`) but only `java` has an embed directory. This is intentional per
the test `TestLoadLanguageQueries_RecognisedButUnembedded` which asserts
the same failure mode for "recognised-but-unembedded" as for "unknown".
No action required.

### I-003 — Tree-sitter `.scm` is a placeholder
**Severity:** INFO
**File:** `internal/infrastructure/treesitter/bundledqueries/java/declarations.scm`
**Finding:** Comment in the file states "EP04US-005 placeholder. Real query
content lands when the parser refactor (future US) starts consuming the
bundled registry." Acceptable — the test asserts only that
`len(set.Queries) >= 1`, not the content.

## Test consistency

| Acceptance criterion | Test                                                                            | Result |
|----------------------|---------------------------------------------------------------------------------|--------|
| AC-1: queries activated, no `queries/` subdir | `TestBundleLoader_LoadsLanguageJava_AttachesBundledQueries` | PASS   |
| AC-2: pinned error literal                     | `TestBundleLoader_UnsupportedLanguage_FailsWithListing` + `TestLanguageUnsupportedError_ErrorString` | PASS   |
| AC-3: pointer equality for shared sets         | `TestBundleLoader_TwoProfilesSameLanguage_ShareQuerySet` (pointer-equal assertion) | PASS   |

`TestLoadLanguageQueries_Java_ReturnsCachedSet` uses `set1 == set2`
identity — the load-bearing pattern documented in the registry doc-comment
matches the test, both pointing at the same invariant.

## Architectural conformity

- ISP: `LoadLanguageQueriesPort` and `ListSupportedLanguagesPort` each
  have one method. PASS.
- Domain purity: `model.LanguageQuerySet` and `vo.Language` have no
  framework imports. PASS.
- Composition root: `wire.go` constructs the registry, hands it to the
  bundle loader and to the `LanguageQueries` deps slot. PASS.
- Errors: `ErrLanguageUnsupported` sentinel + `LanguageUnsupportedError`
  typed wrapper, `Is()` chains to `ErrProfileInvalid`. PASS.
- Logging: `slog.Default()` fallback at every entry point; production
  wiring passes the configured logger. PASS.
- File naming: All new files are camelCase (`languageQuerySet.go`,
  `loadLanguageQueriesPort.go`, `bundleLoader.go`, etc.). PASS.
- Package naming: After Cycle 1 fix, package `bundledqueries` matches the
  `{tech}` lowercase convention used across `/internal/infrastructure/`. PASS.

## Summary

- BLOCKERs:  **0** (1 found, 1 resolved)
- WARNINGs:  **3**
- INFOs:     **3**

The feature passes review after Cycle 1.
