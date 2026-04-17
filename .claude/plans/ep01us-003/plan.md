# Plan — EP01US-003: Filter Query by Type and Tags

## Section 0 — Summary

- Feature: Filter `jitctx query` results by `--type` and `--tags`
  (composable with US-002's `--module`).
- Requirement IDs: **EP01US-003**, **EP01RF-008**, **EP01RF-009**; leans on
  **EP01RF-010** for the no-results Markdown branch.
- Layers touched: **presentation (queryCmd + markdown formatter)** and
  **tests (unit + integration)**. Domain, infrastructure, application, and
  wire layers are **already correct** and are reused unchanged from
  EP01US-002 (commit `9831686`).
- Tiers active: **4, 6** (Tier 1 "domain contract" is present only as a
  *frozen* reference section — no new domain files; Tiers 2, 3, 5 are
  collapsed because nothing in those layers changes).
- Guidelines loaded:
  - `.claude/guidelines/presentation-layer-guidelines.yml`
  - `.claude/guidelines/unit-test-layer-guidelines.yml`
  - `.claude/guidelines/integration-test-layer-guidelines.yml`
- Estimated file count: **3 new, 2 modified** (5 files total).

### What EP01US-002 already gives us and must NOT be duplicated

Verified by reading the live source on branch `main`:

| Capability | Location | Reuse status |
|---|---|---|
| `ArtifactType` VO with `Validate()` accepting `guidelines\|requirements\|scenarios\|contracts` | `internal/domain/vo/artifactType.go` | **Reused unchanged.** |
| `QueryContextInput{Tags []string, Types []vo.ArtifactType}` already carried to the use case | `internal/domain/vo/query/query.go` (lines 8-15) | **Reused unchanged.** |
| `FilterContexts(contexts, module, tags, types, filePath)` implements: module AND (tags OR) AND (types OR) with case-insensitive tag matching | `internal/domain/service/contextFilter.go` | **Reused unchanged.** |
| Unit tests for tag/type/combined filtering already present | `internal/domain/service/contextFilter_test.go` | **Reused — no additions needed.** |
| `queryuc.Impl.Execute` threads `input.Tags` and `input.Types` into `FilterContexts` | `internal/application/usecase/queryuc/usecase.go` (lines 61-67) | **Reused unchanged.** |
| `--type` flag already registered (takes comma-separated list) | `internal/cli/command/queryCmd.go` line 57 | **Reused; help text already mentions all four types.** |
| `--tag` flag registered as **singular** (`--tag`) | `internal/cli/command/queryCmd.go` line 56 | **RENAME to `--tags`** — feature file lines 118/128/138 use plural. |
| `queryCmd` casts `[]string → []ArtifactType` without validation | `internal/cli/command/queryCmd.go` lines 35-38 | **REPLACE with validating parser** — currently unknown types silently pass through and match nothing instead of producing the EP01RF-008 warn. |
| Markdown formatter emits header + contracts + loaded contexts | `internal/cli/format/markdown.go` `writeQueryMarkdown` | **EXTEND with empty-result branch.** |
| `format.WriteQueryResult` dispatches by format (markdown/json/raw) | `internal/cli/format/markdown.go` lines 40-53 | **Reused unchanged.** |
| `root.go` passes `d.Logger` into `NewQueryCmd(d.Query, d.Logger)` | `internal/cli/root.go` line 18 | **Reused — logger plumbing already in place.** |

### In-scope behaviour (delta on top of US-002)

1. `--type <list>` accepts comma-separated artifact types
   (`guidelines,scenarios,requirements,contracts`). Unknown values → a
   `logger.Warn` on stderr **and are dropped before reaching the use case**
   (EP01RF-008 "log a warning and ignore it").
2. `--tags <list>` accepts comma-separated tag strings. A context matches
   if it has **at least one** of the requested tags (EP01RF-009 OR logic).
   Passed verbatim to the use case; no validation.
3. `--module` + `--type` + `--tags` compose with **AND** across the three
   dimensions (already enforced by `FilterContexts`).
4. **Empty result** on stdout (markdown format): when
   `len(out.Loaded) == 0`, emit the header comment **then** a body
   containing the exact string `No contexts matched the given filters` plus
   a hint inviting broader filters (feature line 148 `suggests trying
   broader filters`). The module contracts section is still emitted if
   available — matching the existing ordering keeps the formatter simple.
5. The rename `--tag → --tags` is a **breaking change** within the CLI,
   but no existing test or doc asserts `--tag`; the only consumer is the
   feature file, which already uses `--tags`.

### Out of scope

- JSON/raw empty-state messages (the Gherkin only asserts `stdout
  includes ...` under the default format — scenario line 147).
- Changes to `FilterContexts` semantics (already correct).
- Any change to `QueryContextInput` / `QueryContextOutput` shape.

---

## Section 1 — File Set

| # | File                                                                 | Action | Layer  | Tier | Group  |
|---|----------------------------------------------------------------------|--------|--------|------|--------|
| 1 | `internal/cli/command/queryCmd.go`                                   | modify | pres   | 4    | T4-G1  |
| 2 | `internal/cli/format/markdown.go`                                    | modify | pres   | 4    | T4-G2  |
| 3 | `internal/cli/command/queryTypeParser_test.go`                       | create | test   | 6    | T6-G1  |
| 4 | `internal/cli/format/markdown_test.go`                               | modify | test   | 6    | T6-G2  |
| 5 | `internal/cli/command/queryFiltersIntegration_test.go`               | create | test   | 6    | T6-G3  |

Requirement coverage in Section 1:

- **EP01RF-008** (type filter + unknown-type warn): row 1 (parser), row 3
  (parser unit tests), row 5 (integration scenarios "Filter by single
  type", "Filter by multiple types").
- **EP01RF-009** (tags OR, combined AND): row 1 (`--tags` rename), row 5
  (integration scenarios "Filter by tags", "Filter by multiple tags",
  "Combined filter").
- **EP01US-003** acceptance on empty-result message: row 2, row 4
  (markdown unit test), row 5 (integration scenario "No results with
  combined filters").

---

## Section 2 — Frozen Domain Contract

**No changes.** The contract published by EP01US-002 is reused verbatim. For
downstream reference, the frozen elements consumed by this feature are:

```go
// internal/domain/vo/artifactType.go — frozen
package vo

type ArtifactType string

const (
    ArtifactGuidelines   ArtifactType = "guidelines"
    ArtifactRequirements ArtifactType = "requirements"
    ArtifactScenarios    ArtifactType = "scenarios"
    ArtifactContracts    ArtifactType = "contracts"
)

func (a ArtifactType) Validate() error // returns error when string is not one of the four constants
```

```go
// internal/domain/vo/query/query.go — frozen
package query

type QueryContextInput struct {
    WorkDir  string
    Module   string
    Tags     []string          // OR semantics, case-insensitive
    Types    []vo.ArtifactType // OR semantics
    FilePath string
    Budget   vo.TokenBudget
}
```

```go
// internal/domain/service/contextFilter.go — frozen
package service

// FilterContexts applies: module match AND (len(types)==0 || any(types == c.Type))
// AND (len(tags)==0 || any(c.Tags ∩ tags, case-insensitive)).
func FilterContexts(
    contexts []model.Context,
    module   *model.Module,
    tags     []string,
    types    []vo.ArtifactType,
    _        string,
) []model.Context
```

```go
// internal/cli/wire.go Deps — frozen
type Deps struct {
    ScanFactory command.ScanUseCaseFactory
    Query       queryuc.UseCase
    Plan        planuc.UseCase
    Contracts   contractsuc.UseCase
    Logger      *slog.Logger
}
```

No new ports. No new error sentinels. No new use cases.

---

## Section 3 — Domain Layer Plan

**N/A.** EP01US-003 does not introduce any domain file. `ArtifactType`,
`QueryContextInput`, and `FilterContexts` already satisfy all three
requirement IDs in scope.

---

## Section 4 — Infrastructure Layer Plan

**N/A.** No adapter additions, schema changes, or new fixtures under
`internal/infrastructure/`.

---

## Section 5 — Application Layer Plan

**N/A.** `queryuc.Impl.Execute` already forwards `input.Tags` and
`input.Types` to `service.FilterContexts` (file
`internal/application/usecase/queryuc/usecase.go` lines 61-67). No new
orchestration, no new port calls, no new errors.

---

## Section 6 — Presentation Layer Plan

Two disjoint edits, both at the presentation boundary. They do not touch
each other's files, so T4-G1 and T4-G2 run in parallel.

### 6.1 — T4-G1: `queryCmd.go` — rename `--tag` → `--tags` and validate `--type`

File: `internal/cli/command/queryCmd.go` (modify).

Required edits:

1. **Flag rename.** Line 56 changes from
   `cmd.Flags().StringSliceVar(&opts.tags, "tag", nil, "filter by tags")`
   to
   `cmd.Flags().StringSliceVar(&opts.tags, "tags", nil, "filter by tags (comma-separated; OR within the flag)")`.
   The local field `opts.tags` and the input field `Tags` are already
   plural — only the public flag name changes.

2. **Logger param.** Change the constructor from
   `func NewQueryCmd(uc queryuc.UseCase, _ *slog.Logger) *cobra.Command`
   to
   `func NewQueryCmd(uc queryuc.UseCase, logger *slog.Logger) *cobra.Command`.
   Existing call site in `root.go` already passes `d.Logger` — zero
   downstream changes.

3. **Type parsing helper.** Extract the `opts.types → []vo.ArtifactType`
   conversion into a package-private helper that validates each value:

   ```go
   // parseArtifactTypes converts raw --type values into domain ArtifactTypes.
   // Unknown values are logged via slog.Warn and dropped (EP01RF-008).
   func parseArtifactTypes(raw []string, logger *slog.Logger) []vo.ArtifactType {
       out := make([]vo.ArtifactType, 0, len(raw))
       for _, t := range raw {
           at := vo.ArtifactType(t)
           if err := at.Validate(); err != nil {
               logger.Warn("ignoring unknown --type value",
                   "value", t,
                   "accepted", "guidelines|requirements|scenarios|contracts",
               )
               continue
           }
           out = append(out, at)
       }
       return out
   }
   ```

   Called from `RunE` before building `QueryContextInput`. The helper
   lives alongside the command (same package) so its test sits in
   `command_test` without any new package creation. It satisfies the
   presentation guideline rule that commands "validate inputs" and do not
   contain business logic (validation here is delegated to the domain VO's
   `Validate()`).

4. **Help text.** Update the `--type` description to clarify that unknown
   values are ignored with a warning, e.g.
   `"artifact types: guidelines|requirements|scenarios|contracts (comma-separated; unknowns warn and are ignored)"`.

stdout/stderr contract: the warning goes to `logger` (stderr) only —
stdout remains clean for the markdown payload.

### 6.2 — T4-G2: `markdown.go` — empty-result branch

File: `internal/cli/format/markdown.go` (modify).

Inject the empty-result branch inside `writeQueryMarkdown` after the
contracts block and before the loop over `out.Loaded`. When `len(out.Loaded)
== 0`, emit (on stdout):

```
No contexts matched the given filters — try broader filters (drop --type or --tags, or widen --module).
```

followed by a blank line. The header comment (`<!-- jitctx: 0 contexts
loaded | ~0 tokens | trimmed: 0 -->`) remains — it already correctly
reflects zero loaded contexts. The contracts section, if present, is still
useful (the module exists; only context filtering emptied the result) and
stays. The `---` / source-comment loop is skipped because the range is
empty, so no change is needed there.

Proposed patch shape (illustrative — actual edit is surgical):

```go
// After the contracts block, before the `for _, c := range out.Loaded` loop:
if len(out.Loaded) == 0 {
    if _, err := fmt.Fprintln(w,
        "No contexts matched the given filters — try broader filters "+
            "(drop --type or --tags, or widen --module)."); err != nil {
        return err
    }
    if _, err := fmt.Fprintln(w); err != nil {
        return err
    }
    return nil
}
```

This keeps the JSON (`out` encoded verbatim) and raw (empty loop, prints
nothing) paths unchanged — the Gherkin only asserts on markdown.

**Design alternatives considered**

- Placing the empty-result logic in `queryCmd.go` after `Execute`:
  rejected — presentation guideline says the **formatter** owns output
  shape, not the command.
- Adding a new exported helper `WriteQueryEmpty`: rejected — it would
  duplicate the header-comment emission already in `writeQueryMarkdown`.

---

## Section 7 — Composition Root + Tests Plan

### 7.1 Composition root

**Unchanged.** `wire.go` already exposes `d.Logger`, and `root.go` already
passes it into `NewQueryCmd`. Tier 5 is collapsed.

### 7.2 Unit tests

#### T6-G1 — `internal/cli/command/queryTypeParser_test.go` (create)

Targets the new `parseArtifactTypes` helper. Scenarios:

| # | Scenario | Input `[]string` | Expected `[]vo.ArtifactType` | Warn expected |
|---|----------|------------------|------------------------------|---------------|
| 1 | all four known types | `["guidelines","requirements","scenarios","contracts"]` | all four constants in order | no |
| 2 | unknown type dropped with warn | `["guidelines","junk"]` | `[ArtifactGuidelines]` | yes (once, value=`"junk"`) |
| 3 | all unknown | `["junk","foo"]` | `[]` (empty, non-nil) | yes (twice) |
| 4 | empty slice | `nil` / `[]` | `[]` | no |
| 5 | case-sensitivity guard | `["Guidelines"]` | `[]` — current `Validate()` is case-sensitive; test locks that behaviour. Warn yes. | yes |

Use a buffered `slog.Handler` (bytes buffer + `slog.NewTextHandler`) so
tests can assert on the warn count without leaking to stderr. Test file
lives in `package command_test` consistent with the existing integration
tests; the parser must therefore be exported-from-test-via-thin-wrapper or
the test file must be `package command` (internal). Picking
`package command` keeps the helper unexported and the test in the same
package — consistent with guideline "commands validate inputs".

#### T6-G2 — extend `internal/cli/format/markdown_test.go` (modify)

Add three cases to the existing file:

1. `TestWriteQueryResult_EmptyLoadedShowsNoMatchMessage` — builds
   `QueryContextOutput{Loaded: nil}` with markdown format, asserts stdout
   contains both literal substrings `No contexts matched the given
   filters` and `try broader filters`, AND does NOT contain `---` (the
   source-comment delimiter).
2. `TestWriteQueryResult_EmptyLoadedStillPrintsHeader` — same input,
   asserts the first line matches `^<!-- jitctx: 0 contexts loaded`.
3. `TestWriteQueryResult_NonEmptyDoesNotShowEmptyMessage` — regression
   guard: builds output with one loaded context and asserts the
   empty-result substring is absent.

These three tests are added to the existing file (one modification) rather
than a new file — they share the existing package and imports.

### 7.3 Integration tests

#### T6-G3 — `internal/cli/command/queryFiltersIntegration_test.go` (create)

Style mirrors `queryIntegration_test.go` (the US-002 file):

- Uses `syntheticManifest` / `writeTestManifest` / `discardLogger` from
  the existing test helpers — these are package-level in `command_test`,
  so the new file in the same package sees them for free.
- One `t.TempDir()` per test (`t.Parallel()` on every test).
- Wires real adapters: `fsmanifest.New`, `fscontext.New`,
  `token.NewHeuristicEstimator`, `appqueryuc.New`.

Six test functions, one per Gherkin scenario on feature lines 93-148:

| # | Test name | CLI flags | Expected stdout substrings | Expected-absent |
|---|-----------|-----------|----------------------------|-----------------|
| 1 | `TestQueryCmd_Integration_FilterBySingleType` | `--module user-management --type guidelines` | body of `java-conventions` | bodies of `user-scenarios`, `user-requirements` |
| 2 | `TestQueryCmd_Integration_FilterByMultipleTypes` | `--module user-management --type guidelines,scenarios` | bodies of guidelines + scenarios | body of requirements |
| 3 | `TestQueryCmd_Integration_FilterByTags` | `--module user-management --tags security` | body of `security-guidelines` | bodies of `java-conventions`, `test-conventions` |
| 4 | `TestQueryCmd_Integration_FilterByMultipleTagsUsesOR` | `--module user-management --tags security,auth` | bodies of `security-guidelines` + `auth-guide` | body of `naming-guide` |
| 5 | `TestQueryCmd_Integration_CombinedModuleTypeTags` | `--module user-management --type guidelines --tags security` | body of `security-guidelines` | bodies of `security-scenarios`, `billing-security` |
| 6 | `TestQueryCmd_Integration_NoResultsShowsMessage` | `--module billing --type scenarios --tags graphql` | literal `No contexts matched the given filters` + `try broader filters`; first line matches `^<!-- jitctx: 0 contexts loaded` | none |

Additional regression test (bonus, low cost):

7. `TestQueryCmd_Integration_UnknownTypeWarnsAndIgnored` —
   `--module user-management --type guidelines,junk`: assert stdout
   contains `java-conventions` (guidelines still matched) and that a
   stderr buffer captured from a custom logger contains `ignoring unknown
   --type value` with `value=junk`. This locks the EP01RF-008
   warn-and-ignore contract.

Each test constructs a synthetic manifest with the exact contexts named in
its Gherkin Given-clauses and writes the corresponding `.md` body files
under `.jitctx/{type}/` so `fscontext.ReadContextBody` finds them (pattern
proven in `queryIntegration_test.go`).

No new `testdata/` fixtures are needed — synthetic manifests and inline
body writes are sufficient.

---

## Section 8 — Open Questions & Risks

| # | Question / Risk | Blocking? | Resolution taken in plan |
|---|-----------------|-----------|--------------------------|
| 1 | Is the flag named `--tag` (current code) or `--tags` (feature file)? | No | Feature is authoritative: renaming to `--tags`. No other caller depends on the old name — grep confirmed only `README.md` mentions `--tag`; that doc update is out of scope (not in requirements) but should be flagged to the user post-merge. |
| 2 | Where does "unknown type → warn and ignore" run? | No | Presentation layer (the new `parseArtifactTypes` helper). Rationale: the domain's `ArtifactType.Validate()` returns an error on unknown values — the presentation owns the I/O boundary and the logger, so it is the natural place to consume that error as "warn and drop". Placing it in the use case would mean the domain logs, violating the domain-layer guideline that domain code "does not perform I/O". |
| 3 | Should `--type contracts` actually match anything? | No | Left accepted; no contexts of type `contracts` exist today, so the filter yields empty — exercised by the no-results test path. Future work (EP02?) may emit synthetic contexts of that type. |
| 4 | Format of the empty-result hint — single line or two? | No | Single line with en-dash separator (`... — try broader filters ...`). The Gherkin requires both substrings to appear; a single line satisfies both literal checks. |
| 5 | Does the empty-result message appear in JSON / raw outputs? | No | No. Gherkin asserts only on the default (markdown) format — feature line 147: `stdout includes ...`. JSON-output users parse a structured empty array; raw users see nothing (a deliberate contract). |
| 6 | Is case-sensitivity of `--type` a documented concern? | No | `ArtifactType.Validate()` is case-sensitive. The plan locks that via a unit test but does NOT change the semantics — changing it would be out of scope. |

No blocking questions.

---

## Section 9 — Parallel Execution Plan (authoritative for @agent-manager)

```yaml
tiers:
  - id: 4
    name: Presentation (parallel)
    depends_on: []
    groups:
      - id: T4-G1
        scope:
          create: []
          modify:
            - internal/cli/command/queryCmd.go
        guidelines:
          - .claude/guidelines/presentation-layer-guidelines.yml
        effort: S
        notes: >
          Rename --tag to --tags (plural). Replace inline ArtifactType
          cast with a package-private parseArtifactTypes helper that calls
          vo.ArtifactType.Validate() and logger.Warn on unknowns (drops
          them). Swap the blank `_ *slog.Logger` parameter for a named
          logger and thread it into the parser. Zero changes to the
          QueryContextInput / QueryContextOutput shape.

      - id: T4-G2
        scope:
          create: []
          modify:
            - internal/cli/format/markdown.go
        guidelines:
          - .claude/guidelines/presentation-layer-guidelines.yml
        effort: S
        notes: >
          Add an empty-result branch to writeQueryMarkdown: when
          len(out.Loaded)==0, emit the no-match line on stdout (contains
          both "No contexts matched the given filters" and "try broader
          filters"), then return. Header comment and contracts section
          emission are preserved. JSON and raw paths are untouched.

  - id: 6
    name: Tests (parallel)
    depends_on: [4]
    groups:
      - id: T6-G1
        scope:
          create:
            - internal/cli/command/queryTypeParser_test.go
          modify: []
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: S
        notes: >
          Unit tests for parseArtifactTypes. Five cases: all-known,
          unknown-dropped-with-warn, all-unknown, empty/nil input,
          case-sensitivity lock. Use a buffered slog handler to assert on
          warn count and attributes. Test file in package `command`
          (internal) so the unexported helper is reachable.

      - id: T6-G2
        scope:
          create: []
          modify:
            - internal/cli/format/markdown_test.go
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: S
        notes: >
          Three new test functions appended to markdown_test.go —
          empty-loaded shows no-match message, empty-loaded still prints
          the header comment, non-empty does NOT print the empty message
          (regression guard).

      - id: T6-G3
        scope:
          create:
            - internal/cli/command/queryFiltersIntegration_test.go
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: M
        notes: >
          Seven integration tests covering the six Gherkin scenarios on
          lines 93-148 plus one unknown-type warn regression. Each test
          uses t.TempDir(), t.Parallel(), the existing syntheticManifest
          helper from helpers_test.go / queryIntegration_test.go, and
          real adapters (fsmanifest.New, fscontext.New,
          token.NewHeuristicEstimator, appqueryuc.New). No new
          testdata/ fixtures.
```

---

## Self-validation checklist

- [x] Every file in Section 1 appears in exactly one group in Section 9
      (5 files → 5 group entries, distinct paths).
- [x] Every requirement ID (EP01US-003, EP01RF-008, EP01RF-009) is
      covered by at least one Section 1 row (mapped in Section 1 coverage
      table).
- [x] No file path appears in two groups.
- [x] Every port referenced in Section 2 exists in the codebase today
      (frozen from EP01US-002; no new ports required).
- [x] No use-case `Execute` signature changes — reconfirmed against
      `queryuc.UseCase`.
- [x] `Deps` struct: no new fields — reconfirmed in `wire.go`.
- [x] No `TODO`/`{placeholder}` in the plan.
- [x] DAG is acyclic: T6 depends on T4; T4 depends on nothing
      (domain/app/wire unchanged and already present).
- [x] Tier 1 is *not* introduced because no `internal/domain/**` file is
      in Section 1; listed as N/A in Section 3.
- [x] Tier 5 is *not* introduced because no wiring file appears in
      Section 1; listed as N/A in Section 7.1.
- [x] Every `guidelines[]` path in Section 9 exists (verified:
      presentation, unit-test, integration-test guidelines all present
      under `.claude/guidelines/`).
- [x] No `Blocking: Yes` open question — proceed to implementation.
