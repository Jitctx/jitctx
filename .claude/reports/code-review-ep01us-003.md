# Code Review Report — EP01US-003 (Filter Query by Type and Tags)

**Date**: 2026-04-17
**Scope**: 5 files (modified + created for EP01US-003)
**Reviewer**: @code-reviewer
**Verdict**: PASS WITH WARNINGS

---

## Scope

Modified:
- `internal/cli/command/queryCmd.go`
- `internal/cli/format/markdown.go`
- `internal/cli/format/markdown_test.go`

Created:
- `internal/cli/command/queryFiltersIntegration_test.go`
- `internal/cli/command/queryTypeParser_test.go`

---

## Summary

| Severity  | Count |
| --------- | ----- |
| BLOCKER   | 0     |
| WARNING   | 2     |
| INFO      | 3     |

### Acceptance-criteria coverage (feature file lines 93-148)

| Scenario (Gherkin)                        | Covering test                                                     | Status |
| ----------------------------------------- | ----------------------------------------------------------------- | ------ |
| Filter by single type (96-104)            | `TestQueryCmd_Integration_FilterBySingleType`                     | PASS   |
| Filter by multiple types (106-111)        | `TestQueryCmd_Integration_FilterByMultipleTypes`                  | PASS   |
| Filter by tags (113-121)                  | `TestQueryCmd_Integration_FilterByTag`                            | PASS   |
| Multiple tags OR logic (123-131)          | `TestQueryCmd_Integration_FilterByMultipleTagsOR`                 | PASS   |
| Combined module+type+tags (133-141)       | `TestQueryCmd_Integration_CombinedFilters`                        | PASS   |
| No-results message (143-148)              | `TestQueryCmd_Integration_NoResults`                              | PASS   |
| EP01RF-008 unknown-type warn-and-ignore   | `TestQueryCmd_Integration_UnknownTypeWarnsAndIgnored`,            | PASS   |
|                                           | `TestParseArtifactTypes`, `TestParseArtifactTypes_WarnContainsValue` |      |

Build clean (`go build ./...`), all tests pass
(`go test ./... -run Integration -v` and full `go test ./...`),
`go vet ./...` clean, `gofmt -l` returns no files.

No BLOCKERs. The WARNING/INFO items below are surfaced for awareness and
do NOT trigger a fix cycle per the QA coordinator contract.

---

## Dimension 1 — Architectural conformity

No violations.

- `queryCmd.go` depends only on the domain use case interface
  (`queryuc.UseCase`), domain VOs (`vo.ArtifactType`, `queryvo.*`), and
  the sibling formatter package — conforms to
  `presentation-layer-guidelines.yml` "no infra imports, no facades".
- `parseArtifactTypes` is a **presentation-layer input-normalisation
  helper** (trim + validate against a domain enum, drop unknowns with a
  log). It contains no business decisions: the filter semantics live in
  `internal/domain/service/contextFilter.go` (vetted under US-002). This
  matches the "commands may preprocess flag text into domain VOs"
  permission without crossing into business logic.
- `markdown.go` remains a pure formatter — `io.Writer` + domain
  `QueryContextOutput` → `error`. No domain mutation, no I/O beyond the
  writer.
- No new ports or adapters introduced; US-003 composes existing
  `FilterContexts` args that were already wired in US-002 (`input.Tags`,
  `input.Types`). Correct ISP posture preserved.

---

## Dimension 2 — Go idioms & naming

### W-001 — `%v` formatting of `[]string` tags in markdown output

**Severity**: WARNING
**Location**: `internal/cli/format/markdown.go:92`

```go
fmt.Fprintf(w, "---\n<!-- source: %s | tags: %v -->\n\n%s\n\n",
    c.Path, c.Tags, c.Body)
```

`%v` on a `[]string` renders as `[java naming hexagonal]` — Go debug
format, not a documented agent-facing contract. The README/feature-file
examples do not lock the exact tag serialisation, so this is not a
BLOCKER; however `[]` and space-separation are fragile if any tag ever
contains whitespace or quoting, and the output is noisier than a CSV.

**Suggested** (non-blocking): `strings.Join(c.Tags, ", ")` with a nil
guard, producing `tags: java, naming, hexagonal`. Pre-existing code from
US-002 — not introduced by US-003, but visible in the diff via
`markdown.go` changes.

### W-002 — Accepted-list duplicated as a string literal

**Severity**: WARNING
**Location**: `internal/cli/command/queryCmd.go:35` and `queryCmd.go:74`

```go
logger.Warn("ignoring unknown --type value",
    "value", trimmed,
    "accepted", "guidelines|requirements|scenarios|contracts",
)
// ...
cmd.Flags().StringSliceVar(&opts.types, "type", nil,
    "artifact types: guidelines|requirements|scenarios|contracts ...")
```

The authoritative list of valid artifact types lives in
`internal/domain/vo/artifactType.go` (`ArtifactGuidelines`,
`ArtifactRequirements`, `ArtifactScenarios`, `ArtifactContracts`).
Duplicating the pipe-joined string in two places couples drift risk:
adding a fifth type requires edits in three files rather than one.

**Suggested** (non-blocking): expose a `vo.ArtifactTypesList() []string`
or `AllArtifactTypes` slice, and derive the help text / warn attribute
via `strings.Join`. Defer until a fifth type is added or US-004 touches
the same area.

### I-001 — `ArtifactType(t)` then `.Validate()` is idiomatic

**Severity**: INFO
**Location**: `internal/cli/command/queryCmd.go:31-32`

The pattern
```go
at := vo.ArtifactType(trimmed)
if err := at.Validate(); err != nil { ... }
```
mirrors the `vo.NewTokenBudget` style and keeps the VO construction rule
("constructor or `Validate` for invariants") without forcing a
`NewArtifactType` factory. Acceptable as-is; flagging only because a
`NewArtifactType(s string) (ArtifactType, error)` would be marginally
more discoverable.

### I-002 — Flag name drift vs. guideline example

**Severity**: INFO
**Location**: `internal/cli/command/queryCmd.go:73`

```go
cmd.Flags().StringSliceVar(&opts.tags, "tags", nil, ...)
```

`presentation-layer-guidelines.yml` examples use the singular `--tag`
(line 168 of that file). The feature file (`ep01.feature` lines 118,
128, 138, 146) uses the plural `--tags`, and so does the command. The
Gherkin is the contract — plural wins. Guideline example is stale;
update in a future housekeeping pass. No code change required here.

### I-003 — `out.Module.ID != ""` gating contracts-section emission

**Severity**: INFO
**Location**: `internal/cli/format/markdown.go:62`

The new `len(out.Loaded) == 0` branch now flows **after** the contracts
section — meaning an empty-result query still prints contracts if the
module had any. `TestWriteQueryResult_MarkdownEmptyResult_WithContracts`
locks that behaviour. Good: it preserves the module-header UX while
still surfacing the "broader filters" hint. Non-issue, recorded for
audit trail.

---

## Dimension 3 — Code-smell metrics

| Metric                                            | Value | Threshold | Verdict |
| ------------------------------------------------- | ----- | --------- | ------- |
| `parseArtifactTypes` cyclomatic complexity        | 3     | < 10      | OK      |
| `NewQueryCmd` cyclomatic complexity               | 3     | < 10      | OK      |
| `writeQueryMarkdown` cyclomatic complexity        | 9     | < 10      | OK (borderline) |
| Longest function in scope (`writeQueryMarkdown`)  | 43 LOC | < 60     | OK      |
| Duplication (markdown format / helper)            | none detected | —   | OK   |
| Public surface added                              | `parseArtifactTypes` kept package-private | — | OK |

`writeQueryMarkdown` is approaching the conditional-branch budget (header,
contracts-present, empty-loaded, loaded-loop) — if US-004 (YAML output)
adds one more branch, extract the contracts-section writer to its own
function. Noted, not required.

---

## Dimension 4 — Test consistency & requirements coverage

### Requirements cross-check

- **EP01RF-008 (Query by Type)** — "If an unknown type is specified, log
  a warning and ignore it." Covered explicitly by
  `TestParseArtifactTypes` (7 table cases) and
  `TestQueryCmd_Integration_UnknownTypeWarnsAndIgnored`. The warn log
  asserts both the message and the `value=` attribute (end-to-end
  contract-locking).
- **EP01RF-009 (Query by Tags)** — "OR logic within tags, AND with
  other filters." Covered by `TestQueryCmd_Integration_FilterByTag`,
  `..._FilterByMultipleTagsOR`, and `..._CombinedFilters`.
- **EP01RF-010 (Markdown Output Format)** — no-match message asserted
  in `TestWriteQueryResult_MarkdownEmptyResult`,
  `..._WithContracts`, and the integration `..._NoResults`. Header
  comment format locked by `TestWriteQueryResult_MarkdownHeaderFormat`.

### Test quality

- All leaf tests and subtests call `t.Parallel()` as required by
  `unit-test-layer-guidelines.yml` and `integration-test-layer-guidelines.yml`.
- Table-driven tests in `queryTypeParser_test.go` use kebab-case case
  names (`"empty-input-returns-empty-slice"`, etc.) per the guideline
  convention.
- Fakes avoided — the parser helper is tested with a real `slog.Logger`
  backed by a bytes buffer, which is the correct approach for a
  logger-only collaborator (no port to stub).
- `discardLogger()` reuse from `helpers_test.go` keeps the integration
  tests quiet and matches the stderr-noise rule
  (`output_contract_assertions` rule 3).
- `filterTestContext` / `buildFilterFixture` reduce boilerplate without
  introducing a "builder at this stage" (the guideline forbids heavy
  builders; a typed parameter struct + helper is acceptable and the
  simpler path here).

### I-only observations on tests

- `TestQueryCmd_Integration_UnknownTypeWarnsAndIgnored` rewires the
  command manually rather than going through `runQueryCmd` because it
  needs a non-discard logger. That is the right call — folding a
  `logger` parameter into `runQueryCmd` would over-parameterise the
  helper for one caller. `runQueryCmd` already accepts a `logger`
  argument though, so the duplication in this test could shrink by
  reusing it (`runQueryCmd(t, tmpDir, args, warnLogger)`). Trivial
  nicety; not a review action.
- `append(userManagementModules, ...)` at line 193 mutates shared slice
  backing array risk: `userManagementModules` has `len==cap==1`, so
  `append` allocates a new array — safe in practice, but in a future
  edit where the base slice gains capacity this would alias. Flagging
  as INFO on style, not correctness.

---

## Unresolved from prior cycles

None applicable — US-002's WARNINGs (`W-001` redundant `fs.FS`
assertion, `W-002` dead `newLoadedContextFromModel`) are outside this
scope and remain as noted in
`.claude/reports/code-review-ep01us-002.md`.

---

## Verdict

**PASS WITH WARNINGS** — zero BLOCKERs, 2 WARNINGs that are optional
polish, 3 INFOs for the audit trail. The feature is ready to merge.
