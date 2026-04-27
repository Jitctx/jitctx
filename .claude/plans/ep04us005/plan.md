# Plan — EP04US-005 Tree-sitter Queries Bundled by Language

## Section 0 — Summary

- Feature: introduce a per-language bundled Tree-sitter query registry that
  is selected at profile-load time by the singular `language:` field on
  `profile.yaml`. The bundled adapter side owns an `embed.FS` of `.scm`
  query files indexed by language id; the domain exposes a single ISP port
  that resolves a `vo.Language` into a `model.LanguageQuerySet`. Profiles
  no longer need a `queries/` subdirectory; declaring `language: java`
  activates the bundled Java set automatically. Unsupported languages fail
  loudly at load time with a typed error that lists the supported ids.
- Requirement IDs: **EP04US-005**, **EP04RF-014**.
- Layers touched: `[domain, infra, application, wire, tests]`.
- Tiers active: `[1, 2, 3, 5, 6]` — no Tier 4 (no command/formatter edits;
  the new behavior is wired through existing load paths).
- Guidelines loaded:
  - `.claude/guidelines/domain-layer-guidelines.yml`
  - `.claude/guidelines/infrastructure-layer-guidelines.yml`
  - `.claude/guidelines/application-layer-guidelines.yml`
  - `.claude/guidelines/main-layer-guidelines.yml`
  - `.claude/guidelines/unit-test-layer-guidelines.yml`
  - `.claude/guidelines/integration-test-layer-guidelines.yml`
- Estimated file count: **9 new, 6 modified**.

### Key discovery findings (verified against the codebase 2026-04-27)

1. **`vo.Language` already exists** at `internal/domain/vo/language.go` with
   constants `LanguageGo`, `LanguageJava`, `LanguageTypeScript`,
   `LanguagePython`. **No `Validate()` method, no parser** — but the
   `vo.LanguageJava` constant is already consumed by `scanuc`
   (`internal/application/usecase/scanuc/usecase.go:172`). Reuse this VO;
   add a `ParseLanguage` helper rather than introducing a parallel type.
2. **`bundleDTO.Language` already exists** at
   `internal/infrastructure/fsprofile/bundleDto.go:17` (`yaml:"language"`),
   alongside legacy `Languages []string` (line 18). The mapper
   (`bundleMapper.go:25-31`) already prefers the singular form and
   back-fills `FrameworkProfile.Languages`. **No DTO change is required.**
3. **`FrameworkProfile.Languages` is the *only* language-bearing field
   today.** `model.FrameworkProfile` has no scalar `Language`. Multi-string
   `Languages []string` exists for legacy EP-03 schema fidelity. EP04US-005
   needs the *scalar* identity for the bundled-query lookup, so this plan
   adds a new `Language vo.Language` field to `FrameworkProfile` (zero
   value `""` is treated as "no bundled queries selected", which is the
   only correct semantics for legacy `query_lang` profiles that pre-date
   EP-04). The existing `Languages []string` slice stays intact for EP-03
   parity.
4. **The bundled `spring-boot-hexagonal/profile.yaml` already declares
   `language: java`** (verbatim line 2). After this US lands, that file
   becomes the canonical fixture for "Profile A" and "Profile B" share
   bundled queries — we add a second bundled fixture
   (`testdata/ep04us005/profile-b/profile.yaml`) declaring the same
   language to satisfy Scenario 3 without copying the bundled set.
5. **No Tree-sitter `.scm` files exist in the repository today.** The
   parser at `internal/infrastructure/treesitter/parser.go` walks the
   tree-sitter syntax tree node-by-node using hard-coded node-type
   constants (`internal/infrastructure/treesitter/queries.go` is a
   constants file, NOT an `.scm` query file — its name is misleading but
   the code is purely Go string constants). **There is no current
   query-loading path to extend; we are introducing one for the first
   time.** This means US-005 is a green-field embed/registry — the parser
   itself is not modified by this US.
6. **The query-set payload is intentionally minimal in this US.** Per
   EP04RF-014 the binary is required to embed a query set per language;
   the .feature only asserts that the loader *activates* the set and
   that a profile no longer needs a local `queries/` directory. It does
   NOT require the parser to start consuming the embedded queries.
   Consequently this US ships:
     - the new domain port for resolving a query set by language,
     - the embed-backed registry under
       `internal/infrastructure/treesitter/bundledQueries/<lang>/*.scm`,
     - a single placeholder Java query file that round-trips the
       bytes (so Scenarios 1 and 3 have something to inspect),
     - the load-path wiring that fails with `ErrLanguageUnsupported`
       when `profile.yaml` declares an unknown language.
   The parser refactor that *consumes* the bundled queries remains
   future work and is explicitly out of scope (Open Question 8.1
   resolved (b)). This is consistent with the "Tree-sitter Queries
   Bundled by Language" framing in EP04RF-014: **bundling** is the
   subject; **consumption** is downstream.
7. **Decision: the new port lives at
   `internal/domain/port/parser/loadLanguageQueriesPort.go`.** The
   parser folder already groups `ParseJavaFilePort`, `WalkJavaFilesPort`,
   `ListJavaCommentsPort`, `ListJavaFieldsPort` — every port whose
   subject is "things the Tree-sitter side produces". Putting the
   query-set lookup here (rather than under `port/profile/`) keeps the
   parser port folder cohesive and lets the future parser refactor
   inject this single port without crossing folder boundaries. See
   Open Question 8.2 for the trade-off.
8. **Decision: the supported-language set is owned by infrastructure.**
   The `vo.Language` constants enumerate the *recognized* identifiers;
   the *supported* set is the subset for which an embedded query
   directory exists. The infrastructure-side registry computes the
   sorted list of supported ids by walking
   `embed.FS` entries at construction time. The domain port surfaces
   this list via a second method on the same adapter
   (`SupportedLanguages`), but the *domain port* itself stays
   single-method ISP — `SupportedLanguages` is exposed via a sibling
   port (`ListSupportedLanguagesPort`) that the same adapter struct
   satisfies. Both ports live in the same file group on Tier 1.
9. **Decision: error literal is pinned.** The .feature pins
   `language 'cobol' is not supported` as the user-facing substring.
   We pin the sentinel `ErrLanguageUnsupported` and the typed
   `LanguageUnsupportedError` whose `Error()` returns exactly:

   ```
   language 'cobol' is not supported; available: go, java, python, typescript
   ```

   The single-quote form (not `%q`'s double quotes) is required by the
   feature file. The available list is taken from the registry's
   `SupportedLanguages()` and joined with `", "`.
10. **Decision: bundle loader is the failure point.** The check
    `unsupported language → fail` happens inside the bundle loader's
    post-decode validation — *not* inside the mapper, because the
    mapper is pure (no infra deps) and the supported-language list
    must be queried from the registry. Concretely we wire a new
    `LoadLanguageQueriesPort` into `BundleLoader` (constructor
    parameter, optional / nil-tolerant for tests that don't care).
    When `dto.Language != ""` the loader calls
    `port.LoadLanguageQueries(ctx, lang)`; if the port returns
    `ErrLanguageUnsupported` the loader wraps with the directory
    context and returns. If the port succeeds, the loader stores the
    resolved `vo.Language` on `bundle.Profile.Language` AND attaches
    the resolved `*model.LanguageQuerySet` to a new field
    `bundle.LanguageQueries` on `*model.ProfileBundle`. The
    bundle-load contract thus surfaces the activation evidence
    directly.
11. **Decision: registry caches by language id (Q2 → answer (b)).** The
    `bundledQueries.Registry` reads each language's `.scm` directory
    once on first request and memoises the resulting
    `*model.LanguageQuerySet`. The same pointer is returned on every
    subsequent call for the same language. Tests assert
    `set1 == set2` (pointer equality) for two loads of profiles
    declaring `language: java`. This is the simplest invariant that
    proves "the binary contains the Java queries only once" without
    resorting to `reflect.SliceHeader` shenanigans.

---

## Section 1 — File Set

| #  | File                                                                                          | Action  | Layer  | Tier | Group  |
|----|-----------------------------------------------------------------------------------------------|---------|--------|------|--------|
| 1  | `internal/domain/vo/language.go`                                                              | modify  | domain | 1    | T1-G1  |
| 2  | `internal/domain/model/languageQuerySet.go`                                                   | create  | domain | 1    | T1-G1  |
| 3  | `internal/domain/model/profileBundle.go`                                                      | modify  | domain | 1    | T1-G1  |
| 4  | `internal/domain/model/frameworkProfile.go`                                                   | modify  | domain | 1    | T1-G1  |
| 5  | `internal/domain/port/parser/loadLanguageQueriesPort.go`                                      | create  | domain | 1    | T1-G1  |
| 6  | `internal/domain/port/parser/listSupportedLanguagesPort.go`                                   | create  | domain | 1    | T1-G1  |
| 7  | `internal/domain/errors/errors.go`                                                            | modify  | domain | 1    | T1-G1  |
| 8  | `internal/infrastructure/treesitter/bundledQueries/registry.go`                               | create  | infra  | 2    | T2-G1  |
| 9  | `internal/infrastructure/treesitter/bundledQueries/bundled.go`                                | create  | infra  | 2    | T2-G1  |
| 10 | `internal/infrastructure/treesitter/bundledQueries/java/declarations.scm`                     | create  | infra  | 2    | T2-G1  |
| 11 | `internal/infrastructure/fsprofile/bundleLoader.go`                                           | modify  | infra  | 2    | T2-G2  |
| 12 | `internal/infrastructure/fsprofile/bundleMapper.go`                                           | modify  | infra  | 2    | T2-G2  |
| 13 | `internal/cli/wire.go`                                                                        | modify  | wire   | 5    | T5-G1  |
| 14 | `internal/domain/errors/errors_test.go`                                                       | modify  | tests  | 6    | T6-G1  |
| 15 | `internal/infrastructure/treesitter/bundledQueries/registry_test.go`                          | create  | tests  | 6    | T6-G2  |
| 16 | `internal/infrastructure/fsprofile/bundleLoader_test.go`                                      | modify  | tests  | 6    | T6-G3  |
| 17 | `testdata/ep04us005/profile-a/profile.yaml`                                                   | create  | tests  | 6    | T6-G3  |
| 18 | `testdata/ep04us005/profile-a/templates/.gitkeep`                                              | create  | tests  | 6    | T6-G3  |
| 19 | `testdata/ep04us005/profile-b/profile.yaml`                                                   | create  | tests  | 6    | T6-G3  |
| 20 | `testdata/ep04us005/profile-b/templates/.gitkeep`                                              | create  | tests  | 6    | T6-G3  |
| 21 | `testdata/ep04us005/profile-cobol/profile.yaml`                                               | create  | tests  | 6    | T6-G3  |
| 22 | `testdata/ep04us005/profile-cobol/templates/.gitkeep`                                          | create  | tests  | 6    | T6-G3  |

**Coverage check:** EP04US-005 covered by rows 1-22; EP04RF-014 covered by
rows 5-12 and 15.

---

## Section 2 — Frozen Domain Contract

These signatures are the cross-tier contract. Tier 2 onwards must match
verbatim — any deviation requires a re-discovery.

### 2.1 Value object — `internal/domain/vo/language.go`

```go
package vo

// Language identifies a programming language by its canonical, lowercase id.
// Used by both the framework-profile metadata block and the bundled
// Tree-sitter query registry.
type Language string

const (
	LanguageGo         Language = "go"
	LanguageJava       Language = "java"
	LanguageTypeScript Language = "typescript"
	LanguagePython     Language = "python"
)

// String returns the canonical id as a plain string. Implements fmt.Stringer
// so that error messages can interpolate language values without repeated
// type conversions.
func (l Language) String() string { return string(l) }

// ParseLanguage returns the Language matching the given canonical id.
// Returns the zero value ("") and false when the id is empty or unknown.
// The set of recognised ids is the const block above; the *supported* set
// (i.e. those for which a bundled query directory exists) is owned by the
// infrastructure registry, not by this VO.
func ParseLanguage(id string) (Language, bool) {
	switch Language(id) {
	case LanguageGo, LanguageJava, LanguageTypeScript, LanguagePython:
		return Language(id), true
	default:
		return Language(""), false
	}
}
```

### 2.2 Model — `internal/domain/model/languageQuerySet.go`

```go
package model

import "github.com/jitctx/jitctx/internal/domain/vo"

// LanguageQuerySet is a frozen, in-memory view of every Tree-sitter query
// (.scm) file the binary embeds for one language. Returned by
// parser.LoadLanguageQueriesPort. The returned pointer is the SAME pointer
// across repeated calls for the same Language — the registry caches by
// language id, so two profiles declaring the same language share one
// LanguageQuerySet without binary duplication. Tests rely on this
// pointer-equality invariant (see EP04US-005 Scenario 3).
type LanguageQuerySet struct {
	// Language is the canonical id this set was loaded for. Set by the
	// registry; callers MUST NOT mutate.
	Language vo.Language

	// Queries is keyed by .scm file basename relative to the language
	// directory (e.g. "declarations.scm"). Values are the raw bytes —
	// the parser side parses them via tree-sitter's QueryNew when it
	// starts to consume them in a future US. Iteration order is
	// undefined; callers that need deterministic iteration sort the
	// keys themselves.
	Queries map[string][]byte
}
```

### 2.3 Model patch — `internal/domain/model/frameworkProfile.go`

```go
package model

import "github.com/jitctx/jitctx/internal/domain/vo"

// (existing fields unchanged — Name, Source, Detect, ModuleDetection,
//  Rules, QueryLang, Languages.)

type FrameworkProfile struct {
	Name            string
	Source          ProfileSource
	Detect          ProfileDetect
	ModuleDetection ModuleDetection
	Rules           []ProfileRule
	QueryLang       string
	Languages       []string

	// Language is the singular, canonical EP-04 language id derived from
	// the profile.yaml `language:` scalar. Empty when the profile pre-dates
	// EP-04 (legacy `query_lang` / `languages:[…]` only). When non-empty,
	// the bundled query registry has resolved it successfully — the
	// loader fails with ErrLanguageUnsupported before reaching this
	// assignment otherwise.
	Language vo.Language
}
```

### 2.4 Model patch — `internal/domain/model/profileBundle.go`

```go
package model

// (existing godoc and fields unchanged — Profile, Dir, Templates,
//  RawTypes, RawPackaging, RawAuditRules.)

type ProfileBundle struct {
	Profile       *FrameworkProfile
	Dir           string
	Templates     map[string][]byte
	RawTypes      []ProfileTypeDeclaration
	RawPackaging  []byte
	RawAuditRules []AuditRule

	// LanguageQueries is the bundled Tree-sitter query set the loader
	// resolved from Profile.Language at load time. Nil when the profile
	// did not declare a language (legacy schema). When non-nil, the
	// pointer is shared across every ProfileBundle whose Profile.Language
	// matches — the registry caches by language id (EP04US-005 Scenario 3).
	LanguageQueries *LanguageQuerySet
}
```

### 2.5 Port — `internal/domain/port/parser/loadLanguageQueriesPort.go`

```go
package parser

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/vo"
)

// LoadLanguageQueriesPort resolves a Language id to its bundled Tree-sitter
// query set. Returns errors.ErrLanguageUnsupported (wrapped in
// LanguageUnsupportedError) when no embedded query directory exists for
// the requested language. Implementations cache by language id so that
// repeated calls return the SAME *model.LanguageQuerySet pointer — this is
// the load-bearing invariant for EP04US-005 Scenario 3.
type LoadLanguageQueriesPort interface {
	LoadLanguageQueries(ctx context.Context, lang vo.Language) (*model.LanguageQuerySet, error)
}
```

### 2.6 Port — `internal/domain/port/parser/listSupportedLanguagesPort.go`

```go
package parser

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/vo"
)

// ListSupportedLanguagesPort returns the language ids for which the binary
// embeds a query set, sorted alphabetically. Used by the bundle loader to
// produce the "available: …" suffix on LanguageUnsupportedError. A single
// adapter struct typically satisfies both this port and
// LoadLanguageQueriesPort (ISP composition by satisfaction, not
// inheritance).
type ListSupportedLanguagesPort interface {
	ListSupportedLanguages(ctx context.Context) ([]vo.Language, error)
}
```

### 2.7 Errors patch — `internal/domain/errors/errors.go`

Append, alongside the existing US-001 / US-006 EP-04 sentinels:

```go
// EP04US-005 sentinels.
var (
	// ErrLanguageUnsupported is returned by parser.LoadLanguageQueriesPort
	// (and surfaced by the profile bundle loader) when a profile.yaml
	// declares a language id that has no bundled Tree-sitter query
	// directory in the binary. Wraps ErrProfileInvalid for errors.Is
	// matching so existing translators that only switch on
	// ErrProfileInvalid continue to work.
	ErrLanguageUnsupported = errors.New("language is not supported")
)

// LanguageUnsupportedError carries the offending language id and the
// alphabetically-sorted list of supported ids so the user-facing message
// is reproducible. Error() returns the literal pinned by the EP04US-005
// .feature file:
//
//   language 'cobol' is not supported; available: go, java, python, typescript
//
// errors.Is(err, ErrLanguageUnsupported) and errors.Is(err, ErrProfileInvalid)
// both return true.
type LanguageUnsupportedError struct {
	Language        string   // verbatim profile.yaml `language:` value (may be empty)
	SupportedSorted []string // alphabetically sorted canonical ids
}

func (e *LanguageUnsupportedError) Error() string {
	return fmt.Sprintf("language '%s' is not supported; available: %s",
		e.Language, strings.Join(e.SupportedSorted, ", "))
}

func (e *LanguageUnsupportedError) Is(target error) bool {
	return target == ErrLanguageUnsupported || errors.Is(target, ErrProfileInvalid)
}
```

### 2.8 Wire patch — `internal/cli/wire.go`

The `Deps` struct gains one field; the constructor wires the new adapter
into `BundleLoader`. No use case wiring changes (no use case consumes
queries yet — see Finding 6).

```go
// (existing imports unchanged; one new import added.)
import (
	parserport "github.com/jitctx/jitctx/internal/domain/port/parser"
	tsbundled "github.com/jitctx/jitctx/internal/infrastructure/treesitter/bundledQueries"
)

type Deps struct {
	// (… all existing fields preserved verbatim …)

	// LanguageQueries satisfies both parser.LoadLanguageQueriesPort and
	// parser.ListSupportedLanguagesPort. Backed by *bundledQueries.Registry.
	// EP04US-005.
	LanguageQueries interface {
		parserport.LoadLanguageQueriesPort
		parserport.ListSupportedLanguagesPort
	}
}

// Inside Wire(...):
//   languageQueries := tsbundled.NewRegistry(logger)
//   profileBundleLoader := fsprofile.NewBundleLoader(logger, languageQueries)
//                                                            ^^^^^^^^^^^^^^
//   // (new constructor parameter — see §4.2)
//   …
//   return Deps{
//       …,
//       LanguageQueries: languageQueries,
//   }
```

---

## Section 3 — Domain Layer Plan

### T1-G1 — Domain contract for language-bundled queries

One coordinated edit covering the value object extension, the new model,
the model patches, the two ports, and the new error sentinel/typed error.
The whole layer ships in one group because every file references the
others (the port refers to `LanguageQuerySet`; the model patch on
`ProfileBundle` references `LanguageQuerySet`; the error sentinel is the
return value contract for the port).

**Files in this group:**

1. `internal/domain/vo/language.go` — modify. Add `String()` method and
   `ParseLanguage(id string) (Language, bool)`. Do NOT introduce a
   `Validate()` method — the port-level error is the supported-set check;
   the VO's `ParseLanguage` only checks "recognised", not "supported".
2. `internal/domain/model/languageQuerySet.go` — create. New aggregate
   exactly as in §2.2.
3. `internal/domain/model/frameworkProfile.go` — modify. Add the
   `Language vo.Language` field at the end of the struct. Append-only —
   do not reorder existing fields, to avoid noise in mapper diffs.
4. `internal/domain/model/profileBundle.go` — modify. Add
   `LanguageQueries *LanguageQuerySet` at the end of the struct, with a
   docstring referencing the registry caching invariant. Do NOT add a
   getter — readers access the field directly, consistent with how
   `Templates` is exposed.
5. `internal/domain/port/parser/loadLanguageQueriesPort.go` — create.
   Verbatim §2.5.
6. `internal/domain/port/parser/listSupportedLanguagesPort.go` — create.
   Verbatim §2.6. One method per file (ISP rigid).
7. `internal/domain/errors/errors.go` — modify. Append the §2.7 block at
   the bottom of the existing var blocks. The typed error must end up
   inside the existing `import "strings"` import set (already imported
   at line 6).

**Naming sanity check:** Filenames are camelCase per project convention
(`languageQuerySet.go`, `loadLanguageQueriesPort.go`,
`listSupportedLanguagesPort.go`). No underscore prefixes.

**ISP audit:** `LoadLanguageQueriesPort` and
`ListSupportedLanguagesPort` are each one-method interfaces. The same
infra adapter satisfies both — that is composition by satisfaction, not
inheritance, exactly as `*fsprofile.Bundled` already does for
`LoadBundledProfilePort` + `ListBundledProfilesPort`.

---

## Section 4 — Infrastructure Layer Plan

### T2-G1 — Tree-sitter bundled query registry

**Package layout:**

```
internal/infrastructure/treesitter/bundledQueries/
├── bundled.go           # //go:embed all:java …  (just the embed FS root)
├── registry.go          # Registry struct + LoadLanguageQueries + ListSupportedLanguages
└── java/
    └── declarations.scm # placeholder Java query content (US-005 only ships this one file)
```

**Why a dedicated subpackage?** The existing `treesitter` package is a
parser, not a registry. Mixing the embed.FS into `parser.go` would couple
the parser to the embed (and force every test of the parser to load the
registry). Following the same pattern as
`internal/infrastructure/fsscaffold/templates/` and
`internal/infrastructure/fsprofile/bundled/`, the embed lives in its own
subpackage so the parser can later consume the registry through the
domain port without an import cycle.

#### `bundled.go`

```go
package bundledQueries

import "embed"

//go:embed all:java
var bundledFS embed.FS
```

The `all:` prefix is required so dot-prefixed files (e.g. `.gitkeep` in
languages whose query set is currently empty) survive the embed filter,
matching the EP04US-001 precedent.

For US-005 we only ship the `java/` subdir. **The other languages
recognised by `vo.Language` (`go`, `typescript`, `python`) are NOT
embedded yet** — declaring `language: typescript` therefore yields
`ErrLanguageUnsupported`. This is intentional: the .feature scenario for
"unsupported language" uses `cobol`, but the implementation gives the
same answer for any language whose embed directory is absent. The
"supported" list returned by `ListSupportedLanguages` is computed from
the embed at construction time, so it stays accurate when later USes
ship more `.scm` directories without touching the registry code.

#### `registry.go`

```go
package bundledQueries

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
	"sync"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	parserport "github.com/jitctx/jitctx/internal/domain/port/parser"
	"github.com/jitctx/jitctx/internal/domain/vo"
)

// Registry implements parser.LoadLanguageQueriesPort and
// parser.ListSupportedLanguagesPort. It owns an in-memory cache of
// LanguageQuerySet aggregates keyed by language id; the cache is
// populated lazily on first request and subsequent requests for the same
// language return the SAME pointer (load-bearing invariant for the
// "binary contains the queries only once" feature scenario).
type Registry struct {
	logger    *slog.Logger
	supported []vo.Language          // computed once at construction
	mu        sync.Mutex
	cache     map[vo.Language]*model.LanguageQuerySet
}

// NewRegistry constructs a Registry, eagerly walking the embed root to
// populate the supported-language list. .scm files are read lazily on
// first LoadLanguageQueries call. When logger is nil, slog.Default() is
// used.
func NewRegistry(logger *slog.Logger) *Registry { /* … */ }

func (r *Registry) LoadLanguageQueries(ctx context.Context, lang vo.Language) (*model.LanguageQuerySet, error) {
	// 1. ctx.Err() check.
	// 2. lock; consult cache; if present return the cached pointer.
	// 3. otherwise fs.Sub(bundledFS, string(lang)); fs.Stat — when
	//    ErrNotExist, return &domerr.LanguageUnsupportedError{Language:
	//    string(lang), SupportedSorted: stringSlice(r.supported)} wrapped:
	//    fmt.Errorf("bundled queries for language %q: %w", lang, &…Error)
	// 4. read every *.scm under the language dir into a map[string][]byte;
	//    construct *model.LanguageQuerySet; store in cache; return.
}

func (r *Registry) ListSupportedLanguages(ctx context.Context) ([]vo.Language, error) {
	// Returns a copy of r.supported. ctx.Err() checked first.
}

var (
	_ parserport.LoadLanguageQueriesPort   = (*Registry)(nil)
	_ parserport.ListSupportedLanguagesPort = (*Registry)(nil)
)
```

**Concurrency:** `sync.Mutex` is sufficient — query loading is rare
(once per language per process) and the contention is bounded by the
language count. We do NOT use `sync.Once` per language because that
makes "second load returns same pointer" harder to test deterministically
under failure modes.

**Pointer-equality contract:** The Registry MUST return the same pointer
on repeated calls for the same successful language. Never construct two
`*LanguageQuerySet` for the same id. Tests verify with `==`.

#### `java/declarations.scm`

Placeholder content — a minimal Tree-sitter Java query covering one
node-type capture so the file isn't empty:

```scheme
;; EP04US-005 placeholder. Real query content lands when the parser
;; refactor (future US) starts consuming the bundled registry.
(class_declaration name: (identifier) @class.name)
```

The content is not parsed by the parser in this US (Finding 6); the
file's only role is to make the embed non-empty so registry tests can
assert `len(set.Queries) >= 1`.

### T2-G2 — Bundle loader & mapper integration

#### `internal/infrastructure/fsprofile/bundleLoader.go` — modify

Constructor gains a second optional parameter:

```go
type BundleLoader struct {
	logger          *slog.Logger
	languageQueries parserport.LoadLanguageQueriesPort // nil-tolerant
}

// NewBundleLoader returns a BundleLoader. When logger is nil,
// slog.Default() is used. When languageQueries is nil, the loader
// skips bundled-query resolution (legacy code paths and tests that
// don't care about queries continue to work). Production wiring in
// internal/cli/wire.go always passes a non-nil registry.
func NewBundleLoader(logger *slog.Logger, languageQueries parserport.LoadLanguageQueriesPort) *BundleLoader {
	// …
}
```

`loadFromOS` and the bundled-fallback branch both delegate to a new
helper `(l *BundleLoader) attachLanguageQueries(ctx, bundle)` after
`loadFromFS` returns and the source/dir fields are set:

```go
func (l *BundleLoader) attachLanguageQueries(ctx context.Context, bundle *model.ProfileBundle) error {
	if l.languageQueries == nil {
		return nil
	}
	if bundle.Profile.Language == "" {
		return nil // legacy schema — no language declared
	}
	set, err := l.languageQueries.LoadLanguageQueries(ctx, bundle.Profile.Language)
	if err != nil {
		// already wrapped by the registry
		return err
	}
	bundle.LanguageQueries = set
	return nil
}
```

The `Bundled.LoadBundled` adapter does NOT call this helper directly —
the bundled-fallback branch in `BundleLoader.LoadBundle` calls
`Bundled.LoadBundled` and then runs `attachLanguageQueries` on the
returned bundle, so a freestanding `*Bundled` instance (used by tests
that bypass `BundleLoader`) is unaffected. This preserves the
single-responsibility split between `BundleLoader` (orchestrator) and
`Bundled` (embed reader).

#### `internal/infrastructure/fsprofile/bundleMapper.go` — modify

Inside `toBundleDomain`, after the existing language back-fill block
(line 27-31), add the canonical language assignment:

```go
// EP04US-005: derive the singular vo.Language for the bundled-query
// registry. ParseLanguage returns (zero, false) for unrecognised ids —
// the loader's attachLanguageQueries step is what surfaces that as
// ErrLanguageUnsupported. We do NOT fail here for unrecognised ids;
// the supported-language check is the registry's responsibility.
profile.Language, _ = vo.ParseLanguage(lang)
```

The mapper does NOT consult the registry — it stays pure (no infra deps,
no I/O). The registry wiring is exclusively in `BundleLoader`.

**Edge case — empty `lang`:** `ParseLanguage("")` returns
`(Language(""), false)`. `profile.Language` ends up as the zero value,
which `attachLanguageQueries` recognises as "no language declared" and
skips. No error.

**Edge case — `lang = "cobol"`:** `ParseLanguage("cobol")` returns
`(Language(""), false)` — but we still want the loader to fail with the
verbatim user-supplied id in the error message (the .feature pins
`'cobol'`, not `''`). Therefore `profile.Language` is *not* the source
of truth for the error message; we instead pass the raw `lang` string
through to the loader. Concretely the mapper publishes the raw value via
a new field on `model.FrameworkProfile`? **No** — that would leak DTO
concerns into the domain. Instead, the loader checks
`dto.Language != ""` BEFORE calling the mapper and validates against the
registry directly:

Revised flow (replaces the simple `attachLanguageQueries` sketch above):

```go
// In loadFromFS, after decoding dto:
if dto.Language != "" {
    parsed, ok := vo.ParseLanguage(dto.Language)
    if !ok {
        // Unrecognised id — fail fast with the user-supplied verbatim
        // string and the registry's supported list.
        sup, _ := l.listSupportedOrEmpty(ctx)
        return nil, &domerr.LanguageUnsupportedError{
            Language:        dto.Language,
            SupportedSorted: sup,
        }
    }
    // … known id; the registry call below confirms it is also *supported*
    //     (i.e., has a bundled .scm directory). For US-005 this collapses to
    //     the same answer because vo.LanguageJava is the only embedded one,
    //     but the architecture supports new vo.Language constants landing
    //     before their .scm directories do.
}
```

**Refinement — single source of truth for error message:** to keep the
two failure modes (unrecognised vs recognised-but-not-embedded)
coherent, the registry's `LoadLanguageQueries` accepts the raw string
*through* `vo.Language` (which is just a string alias), so the loader
can call it with `vo.Language(dto.Language)` without parsing first. The
registry sees the unknown id, looks up the embed dir, fails to stat,
and returns `LanguageUnsupportedError` with the verbatim id. **This is
the chosen design** — it removes the ParseLanguage gate entirely from
the failure path, leaving `ParseLanguage` purely for the mapper's
convenience-typing of `profile.Language` on success.

Final `attachLanguageQueries` flow:

```go
func (l *BundleLoader) attachLanguageQueries(ctx context.Context, dto bundleDTO, bundle *model.ProfileBundle) error {
    if l.languageQueries == nil || dto.Language == "" {
        return nil
    }
    set, err := l.languageQueries.LoadLanguageQueries(ctx, vo.Language(dto.Language))
    if err != nil {
        return err // already typed as *LanguageUnsupportedError when applicable
    }
    bundle.LanguageQueries = set
    bundle.Profile.Language = set.Language // canonical, lowercased
    return nil
}
```

Because the helper now needs the `dto`, the caller signature changes
slightly — `loadFromFS` returns `(*model.ProfileBundle, bundleDTO, error)`
or, simpler, `loadFromFS` invokes a registry-callback closure.
Implementation chooses between these two; the contract above is what
matters.

**Compile-time port assertions** at the bottom of `bundleLoader.go`
are extended to include the new dependency type:

```go
var (
    _ profileport.LoadProfileBundlePort = (*BundleLoader)(nil)
)
// Registry-side assertions live in registry.go (see T2-G1).
```

---

## Section 5 — Application Layer Plan

**No application-layer changes in this US.** Per Finding 6, the parser
is not yet refactored to consume the bundled query bytes — the queries
are simply *bundled* and *attached* to `ProfileBundle.LanguageQueries`.
Any use case that holds a `*model.ProfileBundle` therefore now has
access to the resolved query set, but no use case reads it. When the
parser refactor lands in a future US (likely as part of the US-007
`scan` migration to declarative bundles), the consumer wiring will
introduce the `LoadLanguageQueriesPort` injection at the use-case level.

Tier 3 is therefore inactive (`N/A`).

---

## Section 6 — Presentation Layer Plan

**N/A.** No new cobra command, no flag changes, no formatter additions.
The `LanguageUnsupportedError` typed value is surfaced through existing
error-translation paths in `internal/cli/format/errors.go` automatically
because `errors.Is(err, ErrProfileInvalid)` returns true (the typed
error's `Is` method delegates to both `ErrLanguageUnsupported` and
`ErrProfileInvalid`). Any existing translator that maps
`ErrProfileInvalid` to a non-zero exit code therefore handles the new
error correctly with no source changes.

The error message itself is what the user sees; no presentation-side
formatting is required because the literal already contains the
"available: …" suffix.

Tier 4 is therefore inactive (`N/A`).

---

## Section 7 — Composition Root + Tests Plan

### T5-G1 — Wiring (`internal/cli/wire.go`)

Edits, in order:

1. Import `parserport "github.com/jitctx/jitctx/internal/domain/port/parser"`
   and `tsbundled "github.com/jitctx/jitctx/internal/infrastructure/treesitter/bundledQueries"`.
2. Add the anonymous-interface `LanguageQueries` field to `Deps` per §2.8.
3. In `Wire(...)`, immediately after `tsParser := treesitter.New()` and
   before `profileBundleLoader := …`, insert:

   ```go
   languageQueries := tsbundled.NewRegistry(logger)
   ```

4. Change the `BundleLoader` construction to:

   ```go
   profileBundleLoader := fsprofile.NewBundleLoader(logger, languageQueries)
   ```

5. Add `LanguageQueries: languageQueries,` to the returned `Deps{}` literal.

No other constructor sites change. `fsprofile.NewBundleLoader` is called
exactly once in `wire.go` (verified by grep). Test sites that construct
`BundleLoader` directly will pass `nil` for the second parameter to
preserve their existing behaviour — see Tier 6 plan.

### T6-G1 — Domain error tests (`internal/domain/errors/errors_test.go`)

Append three table-driven tests:

1. `Test_LanguageUnsupportedError_ErrorString` — asserts the literal
   `"language 'cobol' is not supported; available: go, java, python, typescript"`
   for input `Language="cobol", SupportedSorted=[]string{"go","java","python","typescript"}`.
2. `Test_LanguageUnsupportedError_IsLanguageUnsupported` — `errors.Is(err, ErrLanguageUnsupported)` true.
3. `Test_LanguageUnsupportedError_IsProfileInvalid` — `errors.Is(err, ErrProfileInvalid)` true.

Plus one negative test that an empty `Language` field renders to
`"language '' is not supported; available: …"` — confirms the format
string handles the zero value without `%q`-style quoting.

### T6-G2 — Registry tests (`internal/infrastructure/treesitter/bundledQueries/registry_test.go`)

Tests:

1. `Test_NewRegistry_DiscoversJava` — constructs the registry, calls
   `ListSupportedLanguages(ctx)`, asserts it contains `vo.LanguageJava`
   and is sorted.
2. `Test_LoadLanguageQueries_Java_ReturnsCachedSet` — calls
   `LoadLanguageQueries(ctx, vo.LanguageJava)` twice; asserts
   `set1 == set2` (pointer equality) AND
   `len(set1.Queries) >= 1` AND `set1.Language == vo.LanguageJava`.
3. `Test_LoadLanguageQueries_UnknownLanguage_ReturnsUnsupportedError` —
   calls with `vo.Language("cobol")`; asserts the returned error
   satisfies `errors.Is(err, ErrLanguageUnsupported)` AND
   `errors.Is(err, ErrProfileInvalid)` AND has the literal
   `"language 'cobol' is not supported; available: java"` as the
   `Error()` output (only `java` is embedded in US-005).
4. `Test_LoadLanguageQueries_RecognisedButUnembedded` — calls with
   `vo.LanguageGo`; asserts the same `ErrLanguageUnsupported` because
   no `go/` directory exists under the embed root.
5. `Test_LoadLanguageQueries_ContextCancelled` — passes a cancelled
   context; asserts `ctx.Err()` is returned without touching the embed.

The test file lives in `internal/infrastructure/treesitter/bundledQueries/`
(same package as `registry.go`); because the registry's exported
surface is small enough to test from outside, these are
**`bundledQueries_test`** external tests.

### T6-G3 — Bundle loader integration tests (`bundleLoader_test.go`) + fixtures

**New fixture directories** under `testdata/ep04us005/`:

| Fixture                            | `profile.yaml` essentials                                  |
|------------------------------------|------------------------------------------------------------|
| `profile-a/profile.yaml`           | `name: profile-a`, `language: java`, no `types:`            |
| `profile-b/profile.yaml`           | `name: profile-b`, `language: java`, no `types:`            |
| `profile-cobol/profile.yaml`       | `name: profile-cobol`, `language: cobol`, no `types:`       |

Each fixture also has an empty `templates/` directory (`.gitkeep`).

**Test additions** (extending the existing
`internal/infrastructure/fsprofile/bundleLoader_test.go`):

1. `TestBundleLoader_LoadsLanguageJava_AttachesBundledQueries` —
   constructs a `BundleLoader` with `tsbundled.NewRegistry(nopLogger)`;
   loads `testdata/ep04us005/profile-a`; asserts:
   - `bundle.Profile.Language == vo.LanguageJava`
   - `bundle.LanguageQueries != nil`
   - `bundle.LanguageQueries.Language == vo.LanguageJava`
   - `len(bundle.LanguageQueries.Queries) >= 1`
   - `bundle.Dir` does NOT contain a `queries/` subdirectory (asserted
     by `os.Stat(filepath.Join(dir,"queries"))` returning `ErrNotExist`).
     This is the explicit Scenario 1 clause.
2. `TestBundleLoader_TwoProfilesSameLanguage_ShareQuerySet` — loads
   `profile-a` and `profile-b` in sequence with a *single* registry
   instance shared across both loads; asserts
   `bundleA.LanguageQueries == bundleB.LanguageQueries` (pointer
   equality). Scenario 3.
3. `TestBundleLoader_UnsupportedLanguage_FailsWithListing` — loads
   `profile-cobol`; asserts:
   - `errors.Is(err, ErrLanguageUnsupported)` true
   - `errors.Is(err, ErrProfileInvalid)` true
   - `err.Error()` contains `"language 'cobol' is not supported"`
   - `err.Error()` contains `"available:"`
   - `err.Error()` contains `"java"` (the only supported language in
     US-005). Scenario 2.
4. `TestBundleLoader_LegacyProfileNoLanguageField_NoQueriesAttached` —
   loads the existing `testdata/ep04us001/valid-profile` (which has
   `language: java` but for safety we also add a fixture without any
   `language:` field if the existing fixture declares one; verify
   first). Asserts `bundle.LanguageQueries == nil` AND no error —
   confirms backward compatibility for legacy schemas.
5. `TestBundleLoader_NilRegistry_LegacyBehavior` — constructs a
   `BundleLoader` with `nil` registry; loads `profile-a`; asserts no
   error AND `bundle.LanguageQueries == nil`. Confirms the
   nil-tolerance contract used by tests that don't care about queries.

**Existing tests in `bundleLoader_test.go`** must be updated to pass
`nil` as the second argument to `fsprofile.NewBundleLoader`. This is a
mechanical edit; the existing assertions are unchanged because the
nil-registry path is a no-op.

---

## Section 8 — Open Questions & Risks

### 8.1 Does the parser start consuming the bundled queries in US-005?

**Resolved: No.** US-005 ships *bundling* only. The parser
(`internal/infrastructure/treesitter/parser.go`) keeps its hard-coded
node-type walker. Consumption is deferred to the future parser refactor
(likely a US in EP-05 or later in EP-04 once the multi-language
expansion lands). EP04RF-014's "Expected Output: Loaded query set ready
for use by the parsing engine" is satisfied by attaching the set to
`ProfileBundle.LanguageQueries` — readiness, not consumption.

**Blocking: No.**

### 8.2 Where does `LoadLanguageQueriesPort` live — `port/parser/` or `port/profile/`?

**Resolved: `port/parser/`.** The subject is "queries the parser side
will use". Putting it under `port/profile/` would imply the profile
loader is the consumer; in fact the profile loader is just the
*trigger*, with the parser as the eventual consumer (US-005 ships only
the trigger). Living next to `parseJavaFilePort.go` and
`walkJavaFilesPort.go` keeps the parser-port folder cohesive.

**Trade-off:** During US-005 the only caller is the bundle loader,
which lives in `fsprofile` — so an outside reader might mistake the
port placement for a profile concern. Mitigation: the port docstring
explicitly cites the parser as the future consumer, and the wire-time
adapter is constructed in the `treesitter` namespace.

**Blocking: No.**

### 8.3 How is "binary contains the Java queries only once" asserted?

**Resolved: pointer equality of the cached `*model.LanguageQuerySet`.**
The registry uses a single mutex-protected `map[vo.Language]*…Set`. Two
calls for the same language return the SAME pointer; tests assert with
`==`. This is strictly stronger than "no binary duplication" (which is
guaranteed by `embed.FS` itself reading the same byte slice every
time), but it's the cleanest invariant to write a Go test against. Any
change that breaks pointer equality (e.g., switching to per-call
allocation) will fail the test loudly.

**Blocking: No.**

### 8.4 Do we need to validate that recognised-but-unembedded languages fail with the SAME error as unrecognised ones?

**Resolved: yes, via the registry path.** The registry doesn't care
whether the requested id is a recognised `vo.Language` constant or
not — it just stats the embed subdirectory. `vo.LanguageGo` (recognised
but no embed) and `vo.Language("cobol")` (unrecognised) both produce
`LanguageUnsupportedError` with the verbatim id. The error message
distinguishes nothing because, from the user's standpoint, "this
language has no bundled query set" is the only relevant fact. This
matches the .feature scenario which uses `cobol` as a stand-in for any
unsupported id.

**Blocking: No.**

### 8.5 Risk — bundled `spring-boot-hexagonal/profile.yaml` already declares `language: java`; will US-001/US-002/US-003/US-004 tests break?

**Mitigation: low risk, no production code path consumes
`bundle.LanguageQueries` yet.** The wiring change (`NewBundleLoader`
gains a second argument) is the only source of breakage; every test
site that calls `NewBundleLoader` will be updated to pass `nil` in the
same Tier 6 group. Re-running the existing suite under the patched
constructor with `nil` is a no-op for all assertions because the
nil-tolerance branch in `attachLanguageQueries` short-circuits before
touching the registry. Verified by inspection of
`bundleLoader_test.go:bundleInput` and call sites.

**Blocking: No.**

### 8.6 Risk — adding `Language vo.Language` to `model.FrameworkProfile` introduces an import cycle.

**Mitigation: none — `model` already imports nothing except stdlib;
adding `vo` is a one-way import from `model` to `vo`, and `vo` does NOT
import `model`.** Verified: `internal/domain/vo/language.go` has zero
imports. Tier 1 plan confirms.

**Blocking: No.**

---

## Section 9 — Parallel Execution Plan (authoritative for @agent-manager)

```yaml
tiers:
  - id: 1
    name: Domain contract for bundled language queries
    depends_on: []
    groups:
      - id: T1-G1
        scope:
          create:
            - internal/domain/model/languageQuerySet.go
            - internal/domain/port/parser/loadLanguageQueriesPort.go
            - internal/domain/port/parser/listSupportedLanguagesPort.go
          modify:
            - internal/domain/vo/language.go
            - internal/domain/model/frameworkProfile.go
            - internal/domain/model/profileBundle.go
            - internal/domain/errors/errors.go
        guidelines:
          - .claude/guidelines/domain-layer-guidelines.yml
        effort: M
        notes: >
          Publishes the contract consumed by Tier 2 (registry +
          loader). Single coordinated edit because the new model
          (LanguageQuerySet) is referenced by the new port AND by
          the patched ProfileBundle struct, and the typed error is
          the return-value contract for the port.

  - id: 2
    name: Infrastructure — bundled query registry + loader integration
    depends_on: [1]
    groups:
      - id: T2-G1
        scope:
          create:
            - internal/infrastructure/treesitter/bundledQueries/bundled.go
            - internal/infrastructure/treesitter/bundledQueries/registry.go
            - internal/infrastructure/treesitter/bundledQueries/java/declarations.scm
        guidelines:
          - .claude/guidelines/infrastructure-layer-guidelines.yml
        effort: M
        notes: >
          New subpackage under treesitter/. Embeds java/*.scm via
          //go:embed all:java. Registry caches *LanguageQuerySet per
          language id — pointer-equality is the load-bearing invariant
          for Scenario 3.
      - id: T2-G2
        scope:
          modify:
            - internal/infrastructure/fsprofile/bundleLoader.go
            - internal/infrastructure/fsprofile/bundleMapper.go
        guidelines:
          - .claude/guidelines/infrastructure-layer-guidelines.yml
        effort: M
        notes: >
          BundleLoader gains a second constructor parameter
          (LoadLanguageQueriesPort, nil-tolerant). After loadFromFS,
          attachLanguageQueries calls the registry; failure surfaces
          LanguageUnsupportedError. Mapper assigns
          profile.Language via vo.ParseLanguage on success only.
          Runs in parallel with T2-G1 because the only cross-group
          symbols are domain types from Tier 1 — bundleLoader.go does
          not import bundledQueries (the port is injected as an
          interface).

  - id: 5
    name: Composition root
    depends_on: [2]
    groups:
      - id: T5-G1
        scope:
          modify:
            - internal/cli/wire.go
        guidelines:
          - .claude/guidelines/main-layer-guidelines.yml
        effort: S
        notes: >
          Constructs *bundledQueries.Registry, threads it into
          fsprofile.NewBundleLoader, exposes it on Deps as
          LanguageQueries. No use-case wiring changes (no consumer yet).

  - id: 6
    name: Tests + fixtures
    depends_on: [5]
    groups:
      - id: T6-G1
        scope:
          modify:
            - internal/domain/errors/errors_test.go
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: S
        notes: >
          Tests for LanguageUnsupportedError literal + Is() chain.
      - id: T6-G2
        scope:
          create:
            - internal/infrastructure/treesitter/bundledQueries/registry_test.go
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: M
        notes: >
          Registry unit tests — discovery, caching (pointer equality),
          unsupported failure, recognised-but-unembedded failure,
          context cancellation.
      - id: T6-G3
        scope:
          modify:
            - internal/infrastructure/fsprofile/bundleLoader_test.go
          create:
            - testdata/ep04us005/profile-a/profile.yaml
            - testdata/ep04us005/profile-a/templates/.gitkeep
            - testdata/ep04us005/profile-b/profile.yaml
            - testdata/ep04us005/profile-b/templates/.gitkeep
            - testdata/ep04us005/profile-cobol/profile.yaml
            - testdata/ep04us005/profile-cobol/templates/.gitkeep
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: M
        notes: >
          Five new test functions covering all three feature scenarios
          plus the nil-registry legacy path plus the no-language-field
          legacy path. Existing call sites in this file are mechanically
          updated to pass nil as the second argument to
          fsprofile.NewBundleLoader.
```
