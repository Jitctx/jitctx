# Plan — EP01US-004: YAML Output Format

## Section 0 — Summary

- Feature: `jitctx query --module <id> --format yaml` renders the query
  result as a structured YAML document on stdout. Markdown remains the
  default when `--format` is absent.
- Requirement IDs: **EP01US-004**, **EP01RF-011**; leans on **EP01RF-010**
  for the default-is-markdown regression assertion (feature lines 162-166).
- Layers touched: **presentation only** — one new formatter file, one
  dispatcher edit, one flag alias on `queryCmd`. Plus tests. Domain,
  infrastructure, application, and wire layers are **unchanged** and
  reused from EP01US-002 (commit `9831686`) / EP01US-003 (commit
  `4f15e60`).
- Tiers active: **4, 6** (Tiers 1, 2, 3, 5 collapsed — nothing to do
  there).
- Guidelines loaded:
  - `.claude/guidelines/presentation-layer-guidelines.yml`
  - `.claude/guidelines/unit-test-layer-guidelines.yml`
  - `.claude/guidelines/integration-test-layer-guidelines.yml`
- Estimated file count: **3 new, 2 modified** (5 files total).

### What prior user stories already give us and must NOT be duplicated

Verified by reading the live source on `main`:

| Capability | Location | Reuse status |
|---|---|---|
| `QueryContextOutput{Module, Loaded, Trimmed, TotalTokens}` with every field the YAML needs (Path, Type, Tags, TokenEstimate, Body, Module.ID, Module.Contracts) | `internal/domain/vo/query/query.go` | **Reused verbatim.** |
| `format.WriteQueryResult(w, format, out)` dispatcher with `markdown` (default), `json`, `raw` branches | `internal/cli/format/markdown.go` lines 40-53 | **Modified** — add one `"yaml"` case; other branches untouched. |
| `queryuc.Impl.Execute` populates `Module.ID`, `Module.Contracts`, `Loaded[*]` (with `Body` read through `ReadContextBodyPort`), `TotalTokens`, `Trimmed` | `internal/application/usecase/queryuc/usecase.go` | **Reused verbatim.** |
| `NewQueryCmd` declares `--output` / `-o` bound to `opts.output`, default `"markdown"` | `internal/cli/command/queryCmd.go` lines 44-80 | **Modified** — add a second `StringVar` call registering `--format` against the *same* `opts.output` field (same pattern already used on lines 70-71 for `--dir`/`--path`). The short alias `-o` stays on `--output`. |
| `gopkg.in/yaml.v3` already listed in go.mod and used by infra / tests | `go.mod`; `internal/infrastructure/fsmanifest/*`; `internal/cli/command/queryIntegration_test.go` | **Reused** for presentation-local marshalling. |
| `writeJSON` private helper for the dispatcher | `internal/cli/format/markdown.go` lines 135-139 | **Reused as precedent** — a new `writeQueryYAML` mirrors its shape. |

### In-scope behaviour (delta on top of US-002 / US-003)

1. **New flag `--format`.** Registered with the same pointer as
   `--output`, default `markdown`, help text
   `"output format: markdown|json|raw|yaml (alias of --output)"`. When
   both flags appear, cobra's last-write-wins rule takes the flag parsed
   last (same as `--dir`/`--path`) — acceptable and undocumented; the
   feature only ever sets one.
2. **Accepted values extended to `yaml`.** No validation gate is added
   (matches the existing dispatcher contract — unknown format strings
   silently fall through to `markdown`); the existing test-suite locks
   that behaviour and we do not change it. YAML simply becomes the
   fourth recognised value.
3. **New YAML renderer** in `internal/cli/format/yaml.go`. Emits a
   document with exactly two top-level keys:
   - `metadata` — object containing `module`, `context_count`,
     `token_total`, `trimmed_count`, `contracts` (summary).
   - `contexts` — array of objects each with keys `path`, `type`,
     `tags`, `token_estimate`, `content`.
4. **Dispatcher edit.** `WriteQueryResult` gets a single new case
   `case "yaml": return writeQueryYAML(w, out)`. The `json`, `raw`, and
   default (markdown) branches are untouched.
5. **Default stays markdown** (feature lines 162-166). No code change is
   required for the regression because `--format` defaults to
   `"markdown"` and the dispatcher's `switch` already falls through to
   `writeQueryMarkdown`. Regression is *test-only*.

### Concrete YAML shape (design decision — not renegotiable in Tier 4)

Requirement EP01RF-011 §2 says:
> "The YAML output includes the same data as Markdown but in a
> machine-parseable structure: metadata (query params, counts, token
> total), contracts summary, and contexts array with path, type, tags,
> token_estimate, and content."

Feature scenario lines 156-160 asserts:
- output is valid YAML
- root key `metadata` with `module == "user-management"` AND
  `context_count == 2`
- root key `contexts` as an array of 2 items
- each context item has keys `path`, `type`, `tags`, `token_estimate`,
  `content`.

Chosen shape (satisfies both):

```yaml
metadata:
  module: user-management
  context_count: 2
  token_total: 27
  trimmed_count: 0
  contracts:
    - name: CreateUserUseCase
      type: input-port
      methods:
        - "UserResponse execute(CreateUserCommand cmd)"
contexts:
  - path: .jitctx/guidelines/java-conventions.md
    type: guidelines
    tags: [java, naming, hexagonal]
    token_estimate: 15
    content: |
      # Java Conventions
      ...
  - path: .jitctx/scenarios/user-scenarios.md
    type: scenarios
    tags: [user]
    token_estimate: 12
    content: |
      # User Scenarios
      ...
```

Notes:
- `metadata.module` is sourced from `out.Module.ID`. When the
  hypothetical future `--file` path is used and `Module.ID` is empty,
  the key is still emitted with an empty string value (valid YAML;
  feature only exercises the module-set path).
- `metadata.context_count` is `len(out.Loaded)`.
- `metadata.token_total` is `out.TotalTokens` (plain int — matches the
  markdown header `~N tokens`).
- `metadata.trimmed_count` is `len(out.Trimmed)`.
- `metadata.contracts` is the same `ModuleSummary.Contracts` projected
  by the markdown formatter — keeps the two renderers informationally
  equivalent as EP01RF-011 asks.
- `contexts[].content` carries the full body loaded by
  `ReadContextBodyPort`, matching the feature's `"content"` key name.
  `Body` is renamed to `content` **only at the presentation DTO level**;
  the domain VO keeps its existing name.
- No `metadata.types` or `metadata.tags` — the requirement says
  "metadata (query params, …)" but the Gherkin only pins `module` and
  `context_count`. We deliberately keep metadata minimal to avoid
  re-litigating a schema that is not yet consumed downstream. This is
  flagged in §8 as a non-blocking open question for a future iteration.

### Out of scope

- Emitting `metadata.types` / `metadata.tags` (the filter inputs). Non-
  blocking; documented in §8 Q-1.
- JSON shape changes. The existing `json` branch still emits the raw
  `QueryContextOutput` via `encoding/json`; untouched.
- Renaming the `--output` flag. Adding `--format` is sufficient and
  keeps the other three commands (`scan`, `plan`, `contracts`) on their
  existing vocabulary.
- Doc updates (README / help text). Out of requirement scope.

---

## Section 1 — File Set

| # | File | Action | Layer | Tier | Group |
|---|------|--------|-------|------|-------|
| 1 | `internal/cli/format/yaml.go` | create | pres | 4 | T4-G1 |
| 2 | `internal/cli/format/markdown.go` | modify | pres | 4 | T4-G1 |
| 3 | `internal/cli/command/queryCmd.go` | modify | pres | 4 | T4-G2 |
| 4 | `internal/cli/format/yaml_test.go` | create | test | 6 | T6-G1 |
| 5 | `internal/cli/command/queryYAMLIntegration_test.go` | create | test | 6 | T6-G2 |

Requirement coverage in Section 1:

- **EP01RF-011** (YAML output includes metadata, contracts summary,
  contexts array with path/type/tags/token_estimate/content): rows 1
  (renderer), 2 (dispatcher case), 4 (unit shape tests), 5 (integration
  happy path).
- **EP01US-004 §1** (`--format yaml` prints YAML): rows 3 (flag), 5
  (integration).
- **EP01US-004 §2** (default is markdown): row 5 (regression integration
  test — the code path is already correct in the dispatcher).

---

## Section 2 — Frozen Domain Contract

**No changes.** Every field the YAML renderer needs is already present on
`queryvo.QueryContextOutput`. The contract consumed by this feature is
reused verbatim from EP01US-002 / EP01US-003:

```go
// internal/domain/vo/query/query.go — frozen
package query

type QueryContextOutput struct {
    Module      ModuleSummary
    Loaded      []LoadedContext
    Trimmed     []LoadedContext
    TotalTokens int
}

type ModuleSummary struct {
    ID        string
    Contracts []ContractSummary
}

type ContractSummary struct {
    Name    string
    Type    string   // string form of model.ContractType
    Methods []string // method signatures in order
}

type LoadedContext struct {
    ID            string
    Type          vo.ArtifactType
    Path          string
    Tags          []string
    Body          string // renamed to `content` at the presentation DTO
    TokenEstimate int
}
```

```go
// internal/domain/usecase/queryuc — frozen
package queryuc

type UseCase interface {
    Execute(ctx context.Context, in queryvo.QueryContextInput) (queryvo.QueryContextOutput, error)
}
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

No new ports. No new error sentinels. No new use cases. No VO changes.

---

## Section 3 — Domain Layer Plan

**N/A.** EP01US-004 introduces zero domain files.

---

## Section 4 — Infrastructure Layer Plan

**N/A.** The YAML renderer is a **presentation** concern (no persistence,
no filesystem I/O beyond `io.Writer`). Using `gopkg.in/yaml.v3` inside
`internal/cli/format/` does not violate the "no YAML tags in domain"
rule from CLAUDE.md — the DTOs with yaml tags live in the presentation
package and are never imported by domain or infrastructure.

---

## Section 5 — Application Layer Plan

**N/A.** `queryuc.Impl.Execute` already returns everything the renderer
needs. No orchestration change.

---

## Section 6 — Presentation Layer Plan

Two parallel edits.

### 6.1 — T4-G1: new `yaml.go` renderer + one-line dispatcher addition

Files:
- `internal/cli/format/yaml.go` (create)
- `internal/cli/format/markdown.go` (modify — one added `case`)

#### 6.1.1 `yaml.go` — new file

Package: `format` (same as `markdown.go`).

Imports:
- `io`
- `gopkg.in/yaml.v3`
- `queryvo "github.com/jitctx/jitctx/internal/domain/vo/query"`

Exported entry point — keeps the dispatcher one-liner trivial:

```go
// writeQueryYAML serialises a QueryContextOutput as a YAML document on w.
// The DTO translation (Body -> content, etc.) is handled locally so the
// domain VO is not polluted with yaml struct tags (CLAUDE.md rule).
func writeQueryYAML(w io.Writer, out queryvo.QueryContextOutput) error {
    doc := newQueryYAMLDoc(out)
    enc := yaml.NewEncoder(w)
    enc.SetIndent(2)
    if err := enc.Encode(doc); err != nil {
        _ = enc.Close()
        return err
    }
    return enc.Close()
}
```

Presentation-local DTOs (all unexported, yaml tags only — no json
tags; JSON uses the domain shape directly via `encoding/json`):

```go
type queryYAMLDoc struct {
    Metadata queryYAMLMetadata `yaml:"metadata"`
    Contexts []queryYAMLContext `yaml:"contexts"`
}

type queryYAMLMetadata struct {
    Module       string                    `yaml:"module"`
    ContextCount int                       `yaml:"context_count"`
    TokenTotal   int                       `yaml:"token_total"`
    TrimmedCount int                       `yaml:"trimmed_count"`
    Contracts    []queryYAMLContractSummary `yaml:"contracts,omitempty"`
}

type queryYAMLContractSummary struct {
    Name    string   `yaml:"name"`
    Type    string   `yaml:"type"`
    Methods []string `yaml:"methods,omitempty"`
}

type queryYAMLContext struct {
    Path          string   `yaml:"path"`
    Type          string   `yaml:"type"`
    Tags          []string `yaml:"tags"`
    TokenEstimate int      `yaml:"token_estimate"`
    Content       string   `yaml:"content"`
}
```

Mapper:

```go
func newQueryYAMLDoc(out queryvo.QueryContextOutput) queryYAMLDoc {
    contexts := make([]queryYAMLContext, 0, len(out.Loaded))
    for _, c := range out.Loaded {
        tags := c.Tags
        if tags == nil {
            tags = []string{}
        }
        contexts = append(contexts, queryYAMLContext{
            Path:          c.Path,
            Type:          string(c.Type),
            Tags:          tags,
            TokenEstimate: c.TokenEstimate,
            Content:       c.Body,
        })
    }

    var contracts []queryYAMLContractSummary
    if len(out.Module.Contracts) > 0 {
        contracts = make([]queryYAMLContractSummary, 0, len(out.Module.Contracts))
        for _, cc := range out.Module.Contracts {
            methods := cc.Methods
            if methods == nil {
                methods = []string{}
            }
            contracts = append(contracts, queryYAMLContractSummary{
                Name:    cc.Name,
                Type:    cc.Type,
                Methods: methods,
            })
        }
    }

    return queryYAMLDoc{
        Metadata: queryYAMLMetadata{
            Module:       out.Module.ID,
            ContextCount: len(out.Loaded),
            TokenTotal:   out.TotalTokens,
            TrimmedCount: len(out.Trimmed),
            Contracts:    contracts,
        },
        Contexts: contexts,
    }
}
```

Invariants enforced by the mapper (all tested in T6-G1):
- `metadata.context_count == len(contexts)`.
- `tags` is always a YAML sequence (never `null`); nil slice is
  materialised as an empty slice before encode.
- `contracts` key is omitted (via `omitempty`) when the module has no
  contracts — keeps the happy-path document tidy without requiring the
  Gherkin to pin the key.
- Encoder uses 2-space indent — the existing repo convention (verified
  against `fsmanifest` writes).

#### 6.1.2 `markdown.go` — dispatcher edit

Single addition inside `WriteQueryResult`:

```go
func WriteQueryResult(w io.Writer, format string, out queryvo.QueryContextOutput) error {
    switch format {
    case "json":
        return writeJSON(w, out)
    case "raw":
        for _, c := range out.Loaded {
            if _, err := fmt.Fprintln(w, c.Body); err != nil {
                return err
            }
        }
        return nil
    case "yaml":                        // <-- NEW
        return writeQueryYAML(w, out)   // <-- NEW
    }
    return writeQueryMarkdown(w, out)
}
```

Nothing else in `markdown.go` changes. `writeQueryMarkdown`,
`WriteScanReport`, `WritePlan`, `WriteContracts`, `writeJSON` are all
untouched.

Do NOT split `markdown.go` into multiple files in this story — that is
an unrelated refactor.

### 6.2 — T4-G2: register `--format` alias on `queryCmd`

File: `internal/cli/command/queryCmd.go` (modify).

Single-line addition after the existing `--output` registration (current
line 77):

```go
cmd.Flags().StringVarP(&opts.output, "output", "o", "markdown", "output format: markdown|json|raw|yaml")
cmd.Flags().StringVar(&opts.output, "format", "markdown", "output format: markdown|json|raw|yaml (alias of --output)") // NEW
```

Also update the help text on the existing `--output` registration from
`"output format: markdown|json|raw"` to `"output format: markdown|json|raw|yaml"`.

No struct change: `opts.output` already holds the string. No short
alias on `--format` (cobra rejects duplicate shorts, and `-o` is already
bound to `--output`).

Why an alias and not a rename:
- `scanCmd.go`, `planCmd.go`, `contractsCmd.go` all still use
  `--output`. Renaming only on `queryCmd` would split the CLI
  vocabulary mid-binary.
- The feature only needs `--format` to accept `yaml`; it does not
  require removal of `--output`.
- Cobra lets two flags target the same pointer — same pattern already
  present at lines 70-71 for `--dir` / `--path`, so this matches prior
  art in the file.

Flag-precedence note: if a user passes `--output markdown --format yaml`
in the same invocation, cobra writes to the pointer in argv order, so
the last wins (yaml). This is acceptable and not asserted by any
scenario. Documented here only so Tier 4 agents don't "fix" it.

No other changes to `queryCmd.go` — `RunE`, flag required markers,
`parseArtifactTypes`, and all other existing flags stay exactly as they
are.

---

## Section 7 — Composition Root + Tests Plan

### 7.1 Composition root

**Unchanged.** `wire.go`, `root.go`, `execute.go`, `main.go`,
`internal/config/**` require zero edits. Tier 5 is collapsed.

### 7.2 Unit tests — T6-G1: `internal/cli/format/yaml_test.go` (create)

Package: `format_test` (black-box — matches existing `markdown_test.go`
convention).

Imports: `bytes`, `testing`, `gopkg.in/yaml.v3`,
`github.com/stretchr/testify/require`, `github.com/jitctx/jitctx/internal/cli/format`,
`github.com/jitctx/jitctx/internal/domain/vo` (for `ArtifactType`
constants), `queryvo "github.com/jitctx/jitctx/internal/domain/vo/query"`.

Each test builds a `QueryContextOutput` fixture, calls
`format.WriteQueryResult(&buf, "yaml", out)`, and asserts on `buf` —
either by `yaml.Unmarshal(buf.Bytes(), &map[string]any{})` and
map-walking, or by substring on the raw YAML.

| # | Test | Fixture | Assertions |
|---|------|---------|------------|
| 1 | `TestWriteQueryResult_YAML_ValidDocument` | output with 2 loaded contexts (java-conventions + user-scenarios), module `user-management`, 27 total tokens, 0 trimmed, one contract | (a) `yaml.Unmarshal(buf.Bytes(), &map[string]any{})` returns no error. (b) Root map has exactly keys `metadata` + `contexts` (no unexpected keys). |
| 2 | `TestWriteQueryResult_YAML_MetadataShape` | same as #1 | `metadata.module == "user-management"`, `metadata.context_count == 2`, `metadata.token_total == 27`, `metadata.trimmed_count == 0`. Pins the Gherkin-level assertions on lines 157-158. |
| 3 | `TestWriteQueryResult_YAML_ContextsArrayShape` | same as #1 | `contexts` is a `[]any` of length 2; each element has exactly the keys `path`, `type`, `tags`, `token_estimate`, `content`. Pins Gherkin lines 159-160. |
| 4 | `TestWriteQueryResult_YAML_ContentPreservesBody` | one loaded context with `Body` = multi-line markdown (including a line starting with `#`) | After `yaml.Unmarshal`, `contexts[0].content` equals the original body string byte-for-byte (round-trip test; catches accidental escaping). |
| 5 | `TestWriteQueryResult_YAML_TagsRenderAsSequence` | one loaded context with `Tags = []string{"java", "naming"}`; one with `Tags = nil` | Both render as YAML sequences (the nil case renders as `tags: []`, not `tags: null`). |
| 6 | `TestWriteQueryResult_YAML_ContractsSummaryIncluded` | output with module + one contract with methods | `metadata.contracts` is a non-empty array with the expected `name`/`type`/`methods` fields. |
| 7 | `TestWriteQueryResult_YAML_ContractsOmittedWhenEmpty` | output with `Module.Contracts == nil` | After `yaml.Unmarshal`, `metadata` map does NOT contain a `contracts` key (locks the `omitempty` behaviour). |
| 8 | `TestWriteQueryResult_YAML_EmptyLoadedStillValid` | output with `Loaded == nil` | `metadata.context_count == 0`; `contexts` key is present and is an empty array (not `null`). |
| 9 | `TestWriteQueryResult_YAML_DoesNotEmitHTMLComment` | same as #1 | Output does NOT start with `<!--` (confirms the renderer is independent of markdown). |

All tests call `t.Parallel()`. No `t.TempDir()` needed — pure
in-memory.

### 7.3 Integration tests — T6-G2: `internal/cli/command/queryYAMLIntegration_test.go` (create)

Package: `command_test` (reuses `syntheticManifest`, `writeTestManifest`,
and `discardLogger` from `queryIntegration_test.go` + `helpers_test.go`
— both live in the same package).

Style mirrors `queryIntegration_test.go`: real adapters
(`fsmanifest.New`, `fscontext.New`, `token.NewHeuristicEstimator`,
`appqueryuc.New`), real cobra command (`command.NewQueryCmd`), captured
stdout via `cmd.SetOut(&buf)`. `t.TempDir()` + `t.Parallel()` on every
test.

Two test functions, one per Gherkin scenario on feature lines 150-166.

#### Test 1 — `TestQueryCmd_Integration_YAMLHappyPath` (feature lines 153-160)

Fixture: the `user-management` module, two matching contexts
(`java-conventions` via `applies_to: [java]`, `user-scenarios` via
`module: user-management`), one non-matching `billing-scenarios`. This
is the exact fixture shape from `TestQueryCmd_Integration_ModuleHappyPath`
— the new test may copy-paste it to stay self-contained (no shared
helper churn).

CLI args: `[]string{"--dir", tmpDir, "--module", "user-management", "--format", "yaml"}`.

Assertions:
1. `yaml.Unmarshal(stdout.Bytes(), &doc)` into a `map[string]any`
   returns no error. (Gherkin line 156: "stdout is valid YAML".)
2. `doc["metadata"].(map[string]any)["module"] == "user-management"`.
   (Line 157.)
3. `doc["metadata"].(map[string]any)["context_count"] == 2`. (Line
   158.)
4. `doc["contexts"].([]any)` has length 2. (Line 159.)
5. Each item in `contexts` has exactly the five keys `path`, `type`,
   `tags`, `token_estimate`, `content`. (Line 160.)
6. The non-matching billing body does NOT appear in any `content`
   field (cross-check against US-002 filter guarantees).
7. `stderr` is empty (no warnings leaked).

#### Test 2 — `TestQueryCmd_Integration_DefaultIsMarkdown` (feature lines 162-166)

Fixture: same `user-management` module with at least one matching
context (so the markdown renderer has something to put between
`---` separators).

CLI args: `[]string{"--dir", tmpDir, "--module", "user-management"}` —
no `--format` flag.

Assertions:
1. First line of stdout matches `^<!-- jitctx:`. (Line 165.)
2. stdout contains the substring `"---"` (the delimiter between
   contexts). (Line 166.)
3. `yaml.Unmarshal(stdout.Bytes(), &map[string]any{})` returns an
   error — confirms the output is NOT YAML (belt-and-braces guard
   against a future default-flag-mixup regression).

Each test constructs its manifest inline via `writeTestManifest` (no
new `testdata/` fixtures). No regression on existing US-002 / US-003
tests because nothing in their code paths is edited.

---

## Section 8 — Open Questions & Risks

| # | Question / Risk | Blocking? | Resolution taken in plan |
|---|-----------------|-----------|--------------------------|
| 1 | Should `metadata` include the query filter inputs (types, tags, file)? The requirement text says "query params" but the Gherkin only pins `module` and `context_count`. | No | Not included in this story. The renderer can always grow keys later without breaking consumers (Gherkin's "has key X" assertions are satisfaction-based, not exhaustive). Revisit when an actual consumer (scripting integration) requests it. |
| 2 | `--format` vs `--output` flag name. Feature says `--format`; the other three commands use `--output`. | No | Add `--format` as a second flag bound to the same pointer on `queryCmd` only. `--output` stays for backwards compatibility. Renaming across `scan`/`plan`/`contracts` is out of scope. |
| 3 | YAML renderer places `Body` under key `content`. Is "content" canonical or should it be "body"? | No | `content` is what the Gherkin asserts (line 160: `"content"`). The domain VO keeps `Body`; the presentation DTO owns the wire name. |
| 4 | Should `metadata.contracts` exist when the module has no contracts? | No | Omitted via `omitempty`. The Gherkin never asserts its presence. Locked by unit test #7 in T6-G1. |
| 5 | Does the YAML path need to respect a `--tags` / `--type` filter in its metadata block? | No | See Q-1 — not in this story. Filtering is already applied by the use case; the YAML only reports results, not filter inputs. |
| 6 | `writeQueryYAML` uses `yaml.NewEncoder(w)` with 2-space indent. Does the repo have a golden style? | No | The existing fsmanifest writer uses `yaml.Marshal` (default 4-space) but that is infra, not presentation. 2-space matches tighter prose output; no Gherkin asserts on whitespace. Locked as an implementation detail. |

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
          create:
            - internal/cli/format/yaml.go
          modify:
            - internal/cli/format/markdown.go
        guidelines:
          - .claude/guidelines/presentation-layer-guidelines.yml
        effort: M
        notes: >
          Create yaml.go with writeQueryYAML + presentation-local DTOs
          (queryYAMLDoc / queryYAMLMetadata / queryYAMLContractSummary /
          queryYAMLContext) carrying yaml tags. Mapper newQueryYAMLDoc
          projects QueryContextOutput onto the DTO: Body -> content,
          Tags nil -> [] slice, module contracts go under metadata
          with omitempty. In markdown.go add exactly one case ("yaml")
          to the WriteQueryResult switch calling writeQueryYAML; do NOT
          touch the markdown, json, or raw branches, and do NOT move
          writeJSON. gopkg.in/yaml.v3 is already in go.mod.

      - id: T4-G2
        scope:
          create: []
          modify:
            - internal/cli/command/queryCmd.go
        guidelines:
          - .claude/guidelines/presentation-layer-guidelines.yml
        effort: S
        notes: >
          Register a new --format flag bound to opts.output (same
          pattern as the existing --dir/--path aliasing on lines
          70-71). Default "markdown". Help text "output format:
          markdown|json|raw|yaml (alias of --output)". Also update
          the existing --output help text to list yaml. No change to
          opts struct, RunE, parseArtifactTypes, or any other flag.
          No short alias on --format (cobra rejects duplicate -o).

  - id: 6
    name: Tests (parallel)
    depends_on: [4]
    groups:
      - id: T6-G1
        scope:
          create:
            - internal/cli/format/yaml_test.go
          modify: []
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: M
        notes: >
          Nine unit tests in package format_test, black-box style
          matching markdown_test.go. Each builds a QueryContextOutput
          fixture, calls format.WriteQueryResult(&buf, "yaml", out),
          and asserts (a) valid YAML via yaml.Unmarshal into
          map[string]any, (b) metadata shape (module, context_count,
          token_total, trimmed_count), (c) contexts array length and
          key set (path/type/tags/token_estimate/content), (d) body
          round-trip, (e) tags-as-sequence even when nil, (f)
          contracts summary inclusion, (g) contracts omitted when
          empty, (h) empty-Loaded still emits an empty contexts
          sequence, (i) output does not start with an HTML comment.
          All t.Parallel(). No t.TempDir().

      - id: T6-G2
        scope:
          create:
            - internal/cli/command/queryYAMLIntegration_test.go
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: M
        notes: >
          Two integration tests in package command_test reusing
          syntheticManifest + writeTestManifest + discardLogger from
          helpers_test.go / queryIntegration_test.go. Test 1
          (TestQueryCmd_Integration_YAMLHappyPath) covers feature
          lines 153-160: wires real adapters (fsmanifest.New,
          fscontext.New, token.NewHeuristicEstimator, appqueryuc.New),
          runs with --format yaml, asserts valid YAML + metadata
          + contexts shape. Test 2
          (TestQueryCmd_Integration_DefaultIsMarkdown) covers lines
          162-166: same fixture, no --format flag, asserts stdout
          starts with "<!-- jitctx:" and contains "---". Each test
          uses t.TempDir() and t.Parallel(). No new testdata/
          fixtures.
```

---

## Self-validation checklist

- [x] Every file in Section 1 appears in exactly one group in Section 9
      (5 files → 5 distinct entries across T4-G1, T4-G2, T6-G1, T6-G2).
- [x] Every requirement ID (EP01US-004, EP01RF-011) is covered by at
      least one Section 1 row (see coverage note under Section 1).
- [x] No file path appears in two groups (`yaml.go`, `markdown.go`,
      `queryCmd.go`, `yaml_test.go`, `queryYAMLIntegration_test.go` are
      all distinct).
- [x] Every port referenced in Section 2 exists in the codebase today
      (frozen; zero new ports required).
- [x] Use-case `Execute` signature unchanged — reconfirmed against
      `queryuc.UseCase`.
- [x] `Deps` struct: no new fields — reconfirmed in `wire.go`.
- [x] No `TODO`/`{placeholder}` in the plan.
- [x] DAG is acyclic: T6 depends on T4; T4 depends on nothing
      (domain/app/wire unchanged and already present from US-002/US-003).
- [x] Tier 1 is *not* introduced because no `internal/domain/**` file
      appears in Section 1; listed as N/A in Section 3.
- [x] Tier 5 is *not* introduced because no wiring file appears in
      Section 1; listed as N/A in Section 7.1.
- [x] Every `guidelines[]` path in Section 9 exists (verified:
      presentation, unit-test, integration-test guidelines all present
      under `.claude/guidelines/`).
- [x] No `Blocking: Yes` open question — proceed to implementation.
