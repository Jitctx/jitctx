<p align="center">
  <h1 align="center">jitctx</h1>
  <p align="center">
    <strong>Stop feeding your AI the entire project. Load only what it needs.</strong>
  </p>
  <p align="center">
    <a href="#quick-start">Quick Start</a> •
    <a href="#why-jitctx">Why</a> •
    <a href="#how-it-works">How It Works</a> •
    <a href="#commands">Commands</a> •
    <a href="#framework-profiles">Framework Profiles</a> •
    <a href="#roadmap">Roadmap</a> •
    <a href="README.pt-BR.md">Português</a>
  </p>
  <p align="center">
    <img src="https://img.shields.io/badge/status-Epics%2001%E2%80%9304%20shipped-brightgreen" alt="Status" />
    <img src="https://img.shields.io/github/license/jitctx/jitctx" alt="License" />
    <img src="https://img.shields.io/badge/lang-Go%201.25%2B-00ADD8?logo=go" alt="Go" />
  </p>
</p>

---

**jitctx** is a CLI tool that gives AI coding agents exactly the context they need — nothing more, nothing less — **and** pre-declares the architectural skeleton of a feature so multiple agents can implement it in parallel without conflicts. Instead of dumping your entire codebase, guidelines, and specs into the context window, jitctx loads only the relevant fragments based on what the agent is working on; instead of asking each agent to re-emit boilerplate (packages, imports, constructors, method signatures), jitctx scaffolds it mechanically from a deterministic spec.

Think of it as **two leverages on the same problem**:

- **Lazy loading for AI context** — agents pull only the slice they need (`jitctx query`, `jitctx contracts`), scoped to a module, file, or tag.
- **Deterministic scaffolding** — a rigid markdown spec is turned into parallel execution layers (`jitctx plan`) and production stubs (`jitctx scaffold`) by Go code, never by an LLM. Agents spend their output tokens on business logic, not on class declarations they would all hallucinate slightly differently.

## The Problem

Every AI coding tool today - Claude Code, Cursor, Aider, Copilot - struggles with the same bottleneck: **context management is manual and wasteful.**

You're implementing a user authentication endpoint, but your AI agent receives:
- Guidelines for React components it won't touch
- Requirements for the billing module it doesn't need
- Test scenarios for features completely unrelated to the task

The result? **70-80% of loaded context is noise.** You're burning tokens, paying for irrelevant input, and diluting the model's attention - which degrades the quality of its output.

## Why jitctx

| Tool | Approach | Limitation |
|------|----------|------------|
| **Repomix** | Packs everything into one file | All-or-nothing, no filtering |
| **Aider repo-map** | Ranks code by graph relevance | Only maps code, not project artifacts |
| **Edgee / Headroom** | Compresses tokens at transport layer | Compresses noise instead of removing it |
| **jitctx** | **Loads only relevant context on demand** | - |

jitctx doesn't compress what's already in the context window. It prevents irrelevant context from entering in the first place.

## Quick Start

```bash
# Build from source (Go 1.25+)
go build ./cmd/jitctx

# 1. Scan — produce project-state.yaml from your codebase
jitctx scan --path /path/to/project

# 2. Query — emit only the context your agent needs
jitctx query --module user-management --type guidelines --tags security
jitctx query --file src/main/java/com/app/user/UserController.java   # infer module
jitctx query --module user-management --format yaml                  # YAML output

# 3. Plan — author a feature spec, then derive parallel execution layers
jitctx plan --new add-password-reset --module user-management        # scaffold a spec template
$EDITOR .jitctx/specs/add-password-reset.md
jitctx plan --feature add-password-reset                             # show layered DAG
jitctx plan --feature add-password-reset --diff                      # diff spec vs current state

# 4. Scaffold — generate production stubs from the spec (templates + contracts)
jitctx scaffold --feature add-password-reset

# 5. Contracts — emit the contract slice an agent needs to implement one file
jitctx contracts --feature add-password-reset \
                 --for src/main/java/com/app/user/adapter/in/web/PasswordResetController.java

# 6. List & Audit — inspect the manifest and check the source against profile rules
jitctx list modules
jitctx list tags
jitctx audit                                                          # uses profile audit_rules
```

> **Current status:** Epics 01–04 are shipped. The CLI exposes `scan`, `query`,
> `plan` (with `--new`, layers, and `--diff` modes), `scaffold`, `contracts`,
> `list`, and `audit`. The active profile is fully **declarative** — types,
> classification rules, code templates, and audit rules all live in the
> profile YAML. The reference profile is `spring-boot-hexagonal`.

## How It Works

jitctx is a deterministic Go CLI — **no LLM calls inside jitctx**. It operates
in two complementary loops:

### The "map" loop — scan / query / list / audit

```
Your Codebase ──► jitctx scan ──► project-state.yaml ──► query | list | audit
```

`scan` parses source code with [Tree-sitter](https://tree-sitter.github.io/),
applies the active **framework profile** (declarative YAML), and writes a
`project-state.yaml` manifest. The manifest is a structured map of modules,
contracts, dependencies, endpoints and the engineering artifacts under
`.jitctx/` (guidelines, requirements, scenarios).

- `query` filters that manifest by module, type, tags, or source file and
  emits only the slice the AI agent needs to stdout (Markdown, YAML, JSON,
  or raw).
- `list` prints the modules or tags discovered.
- `audit` re-parses the source and reports violations of the profile's
  declarative `audit_rules` (e.g. `@Entity` outside `domain/`,
  `@RestController` outside `adapter/in/web/`). Source code is **never**
  modified — the report is for you or your agent to act on.

### The "build" loop — plan / scaffold / contracts

```
spec.md ──► jitctx plan ──► layered DAG ──┐
        ╲                                 ├──► parallel agents
         ──► jitctx scaffold ──► stubs ───┤
        ╲                                 │
         ──► jitctx contracts ────────────┘  (per-file contract slice)
```

You (or your LLM) write a feature spec in a rigid markdown format
(`docs/SPEC_FORMAT.md`). Then:

- `plan --new <feature>` scaffolds a starter spec.
- `plan --feature <name>` computes the contract dependency graph and emits
  parallel execution **layers** — files in the same layer have no
  cross-dependencies, so multiple agents can implement them concurrently
  without conflicts.
- `plan --feature <name> --diff` compares the desired spec against
  `project-state.yaml` and lists what's added, removed, or out-of-sync —
  the brownfield-friendly equivalent of layers mode.
- `scaffold --feature <name>` materializes production stubs (interfaces,
  classes, tests) using the templates declared in the active profile. Stubs
  contain package, imports, class declaration, constructor and method
  signatures — agents only fill in business logic.
- `contracts --feature <name> --for <path>` extracts the minimal contract
  slice an agent needs to implement a single target file — input ports,
  output ports, dependencies — and prints them ready to inject.

### Result

Two compounding wins:

1. **Input savings** — agents never receive irrelevant guidelines or
   modules.
2. **Output savings** — agents never re-emit boilerplate (package, imports,
   constructors, method signatures), because jitctx generated it
   mechanically.

## Commands

| Command | Purpose | Status |
|---|---|---|
| `jitctx scan` | Parse source + profile rules → `project-state.yaml`. Flags: `--path`/`--dir`, `--profile`, `--manifest`, `--output`, `--refactors` (list `TODO(jitctx)` markers instead of writing the manifest). | ✅ |
| `jitctx query` | Emit filtered context to stdout. Flags: `--module`/`-m`, `--file` (infer module from a source path), `--type`, `--tags`, `--budget` (reserved — token budget enforcement is incremental), `--format`/`--output` (`markdown`, `yaml`, `json`, `raw`). | ✅ |
| `jitctx list modules\|tags` | Print modules or tags from the manifest (alphabetical). | ✅ |
| `jitctx plan --new <feature> --module <id>` | Scaffold a starter spec under `.jitctx/specs/<feature>.md`. | ✅ |
| `jitctx plan --feature <name>` | Compute the parallel execution layers from the spec's contract DAG. `--format text\|json`. | ✅ |
| `jitctx plan --feature <name> --diff` | Compare the spec against `project-state.yaml`; report adds/removes/drift. | ✅ |
| `jitctx scaffold --feature <name>` | Render production + test stubs from the spec via profile templates. | ✅ |
| `jitctx contracts --for <path> [--feature <name>]` | Emit the contract slice required to implement a single target file. | ✅ |
| `jitctx audit` | Audit the codebase against the profile's `audit_rules`. Read-only — never edits source. | ✅ |

> All commands respect the project-wide convention: **stdout** carries
> machine-readable tool output (Markdown/JSON/YAML), **stderr** carries
> `slog` logs. Pipe stdout into your agent without filtering noise.

## In Practice with Claude Code

Add this to your `CLAUDE.md`:

```markdown
## Context Loading

Before implementing any task, load context using jitctx:

    jitctx query --module <module> --type <types> [--tags <tags>]

Examples:
- Backend feature: `jitctx query --module user-management --type guidelines,requirements --tags java,api`
- Tests: `jitctx query --module billing --type scenarios`
- Implementing a specific file: `jitctx query --file src/main/java/...UserController.java`

For new features:

    jitctx plan --new <feature-name> --module <module-id>      # scaffold spec
    jitctx plan --feature <feature-name>                       # see parallel layers
    jitctx scaffold --feature <feature-name>                   # generate stubs
    jitctx contracts --feature <feature-name> --for <path>     # contract slice for one file

For brownfield work:

    jitctx audit                                               # find violations
    jitctx scan --refactors                                    # list TODO(jitctx) markers
    jitctx plan --feature <name> --diff                        # spec vs current state

Always use jitctx. Never read .jitctx/ files directly.
```

### Example Output

```bash
$ jitctx query --module user-management --type guidelines --tags security
```

```markdown
<!-- jitctx: 2 contexts loaded | ~830 tokens | trimmed: 0 -->

## Contracts — user-management

- **CreateUserUseCase** (input-port)
  - `UserResponse execute(CreateUserCommand cmd)`

---
<!-- context: java-conventions | type: guidelines | tags: java, api, security -->

# Java Conventions
...

---
<!-- context: security-guidelines | type: guidelines | tags: security, auth -->

# Security Guidelines
...
```

The same query in YAML:

```bash
$ jitctx query --module user-management --type guidelines --tags security --format yaml
```

```yaml
metadata:
  module: user-management
  context_count: 2
  token_total: 830
  trimmed_count: 0
  contracts:
    - name: CreateUserUseCase
      type: input-port
      methods:
        - "UserResponse execute(CreateUserCommand cmd)"
contexts:
  - path: .jitctx/guidelines/java-conventions.md
    type: guidelines
    tags: [java, api, security]
    token_estimate: 450
    content: |
      # Java Conventions
      ...
  - path: .jitctx/guidelines/security.md
    type: guidelines
    tags: [security, auth]
    token_estimate: 380
    content: |
      # Security Guidelines
      ...
```

## Project State Schema

The `project-state.yaml` generated by `jitctx scan` follows a universal schema, regardless of language:

```yaml
generated_at: "2026-04-16T14:30:00-03:00"
stack:
  languages: [java, typescript]
  frameworks: [spring-boot, nextjs]

modules:
  - id: user-management
    path: src/main/java/com/app/user
    tags: [auth, rbac, signup, login, password]
    contracts:
      - name: CreateUserUseCase
        type: input-port
        path: port/in/CreateUserUseCase.java
        methods:
          - signature: "UserResponse execute(CreateUserCommand cmd)"
      - name: UserRepository
        type: output-port
        path: port/out/UserRepository.java
        methods:
          - signature: "Optional<User> findByEmail(String email)"
          - signature: "User save(User user)"
    dependencies: [notification]

contexts:
  - id: java-conventions
    type: guidelines
    applies_to: [java]
    tags: [naming, architecture, hexagonal]
    path: .jitctx/guidelines/java-conventions.md
    token_estimate: 450

  - id: user-scenarios
    type: scenarios
    module: user-management
    tags: [registration, auth, password-reset]
    path: .jitctx/scenarios/user-management.feature.md
    token_estimate: 620
```

## Spec → Plan → Scaffold → Contracts

jitctx pre-declares the skeleton of your architecture — interfaces, types,
and method signatures — **before** any implementation exists. This unlocks
**deterministic parallelism** for multi-agent workflows.

The full reference for the spec format is in
[`docs/SPEC_FORMAT.md`](docs/SPEC_FORMAT.md).

### 1. Author the spec

```bash
$ jitctx plan --new add-password-reset --module user-management
# writes .jitctx/specs/add-password-reset.md with header + sample contracts
```

```markdown
# Feature: add-password-reset
Module: user-management

## Contract: ResetPasswordUseCase
Type: input-port
Methods:
  - void execute(String email)

## Contract: PasswordResetRepository
Type: output-port
Methods:
  - void save(PasswordResetToken token)

## Contract: PasswordResetService
Type: service
Implements: ResetPasswordUseCase
DependsOn: PasswordResetRepository, NotificationGateway

## Contract: PasswordResetController
Type: rest-adapter
Uses: ResetPasswordUseCase
Endpoints:
  - POST /users/password-reset
```

### 2. Compute parallel layers

```bash
$ jitctx plan --feature add-password-reset
```

```
Layer 0 [parallel — no dependencies]
  ├── ResetPasswordUseCase     (input-port)
  └── PasswordResetRepository  (output-port)

Layer 1 [parallel — depends on layer 0]
  ├── PasswordResetService     (service → implements ResetPasswordUseCase)
  └── PasswordResetController  (rest-adapter → uses ResetPasswordUseCase)

Layer 2 [sequential — depends on layer 1]
  └── PasswordResetServiceTest
```

Need machine-readable output? `--format json`. Need to know how this spec
diverges from the live manifest? `--diff`.

### 3. Scaffold production stubs

```bash
$ jitctx scaffold --feature add-password-reset
```

`scaffold` reads templates from the active profile and writes interface,
class, and test stubs at the conventional paths. Stubs contain everything
**except** business logic — package, imports, annotations, constructor,
method signatures, fields.

### 4. Hand each agent its contract slice

```bash
$ jitctx contracts --feature add-password-reset \
                   --for src/main/java/com/app/user/adapter/in/web/PasswordResetController.java
```

```markdown
## ResetPasswordUseCase (input-port)
public interface ResetPasswordUseCase {
    void execute(String email);
}
```

Agent A implements `PasswordResetService` against the same `ResetPasswordUseCase`
contract. Agent B implements `PasswordResetController`. They run in parallel,
on different files, with no merge conflicts — because the contract was
materialized **before** either agent started.

## Framework Profiles

jitctx doesn't ship one parser per language. It uses
[Tree-sitter](https://tree-sitter.github.io/) for universal AST parsing and
**fully declarative framework profiles** (YAML) to interpret what the AST
means in your architecture.

```
Source Code ──► Tree-sitter (AST) ──► Framework Profile ──► project-state.yaml
                                       │
                                       ├── types          (vocabulary)
                                       ├── rules          (classification, first-match wins)
                                       ├── audit_rules    (consumed by `jitctx audit`)
                                       └── templates      (consumed by `jitctx scaffold`)
```

After Epic 04, **every architectural decision lives in the profile** — there
is no Java- or Spring-specific Go code in the core. Adding a new framework
is a YAML + templates contribution, not a code change.

### What a profile contains

```yaml
# profiles/spring-boot-hexagonal.yaml
name: spring-boot-hexagonal
languages: [java]
query_lang: java

# When does this profile match?
detect:
  files:
    - name: pom.xml
      contains: "org.springframework.boot"
    - name: build.gradle
      contains: "org.springframework.boot"

# Module discovery (port/in, port/out, @Entity → module roots)
module_detection:
  strategy: hexagonal
  roots: [src/main/java/**/domain, src/main/java/**]

# Architectural vocabulary — declarative, no Go enum
types:
  - name: input-port
  - name: output-port
  - name: entity
  - name: service
  - name: rest-adapter
  - name: jpa-adapter

# Classification — first match wins; supports multi-classification
rules:
  - match: { node_type: interface_declaration, path_contains: /port/in/ }
    classify_as: input-port
  - match: { node_type: class_declaration, has_annotation: Entity }
    classify_as: entity
  - match: { node_type: class_declaration, has_annotation: RestController }
    classify_as: rest-adapter

# Read-only checks for `jitctx audit`
audit_rules:
  - id: entity-path-mismatch
    kind: annotation_path_mismatch
    severity: ERROR
    params: { annotation: Entity, path_required: /domain/ }

# Templates for `jitctx scaffold` (per type, per file)
templates:
  input-port: { ... }
  service:    { ... }
```

### Sample profiles

jitctx **does not embed** profiles in the binary. Copy the sample you need
into your project's `.jitctx/profiles/` and the binary auto-detects it at
scan time.

```bash
# From the root of your project:
cp /path/to/jitctx/profiles/spring-boot-hexagonal.yaml .jitctx/profiles/
jitctx scan
```

| Profile | Detects | Classifies | Status |
|---|---|---|---|
| `spring-boot-hexagonal` | `pom.xml`, `build.gradle`, `build.gradle.kts` with `org.springframework.boot` | Ports, adapters, entities, controllers, services, JPA repositories | ✅ Shipped (Epics 01–04) |
| `nextjs-app-router` | `package.json`, `next.config.*` | Routes, components, API handlers, hooks, types | 📋 Planned |
| `go-standard` | `go.mod` | Packages, interfaces, structs, handlers | 📋 Planned |

### Adding a new framework

Supporting a new framework means writing a YAML profile (and a small set of
templates), **not Go code**:

```bash
cp /path/to/jitctx/profiles/spring-boot-hexagonal.yaml .jitctx/profiles/my-framework.yaml
$EDITOR .jitctx/profiles/my-framework.yaml
jitctx scan
```

See [`profiles/README.md`](profiles/README.md) for the full contribution
guide and [`docs/SPEC_FORMAT.md`](docs/SPEC_FORMAT.md) for the spec format
the profile templates render against.

### Tree-sitter query sets

Each language needs a small Tree-sitter query set (~10–15 lines) to surface
the AST nodes the profile rules reference. Java is shipped today; TypeScript,
Go, and Python are planned.

## Roadmap

### Epic 01 — End-to-End MVP (Scan + Query) ✅ Shipped

- [x] DDD + Clean architecture in Go 1.25+
- [x] YAML manifest schema, loader, atomic-rename writer
- [x] Module / context discovery from `.jitctx/`
- [x] Token-estimate heuristic (runes / 4)
- [x] Inter-module dependency detection from imports
- [x] Tree-sitter Java integration (classes, interfaces, enums, records;
      multi-annotation, qualified names, generics, partial-parse tolerance)
- [x] `spring-boot-hexagonal` profile with path + annotation + implements
      classification rules
- [x] Query engine (filter by module, type, tags; `--file` to infer module
      from a source path)
- [x] `jitctx scan`, `jitctx query`, `jitctx list` commands
- [x] Output formatters: Markdown (default), YAML, JSON, raw
- [x] Typed errors with actionable hints

### Epic 02 — Plan + Scaffold for Parallel Agent Execution ✅ Shipped

- [x] Spec format v1 (rigid markdown, see `docs/SPEC_FORMAT.md`)
- [x] `jitctx plan --new` — scaffold a starter spec
- [x] `jitctx plan --feature` — compute parallel execution layers from
      contract dependencies
- [x] `jitctx scaffold` — render production + test stubs via profile
      templates
- [x] `jitctx contracts --for` — emit per-file contract slice for an agent
- [x] Path resolution (explicit `--file` → convention → optional config)

### Epic 03 — Brownfield + Refactoring Support ✅ Shipped

- [x] `jitctx audit` — declarative `audit_rules` from the active profile;
      read-only, never edits source
- [x] `jitctx plan --diff` — spec vs current manifest, adds/removes/drift
- [x] `jitctx scan --refactors` — index `TODO(jitctx)` markers from source
- [x] Stale marker detection via git history (markers older than N commits
      surface in the report)
- [x] Two-section audit report (architecture violations + refactor markers)

### Epic 04 — Profile Generalization ✅ Shipped

- [x] All Java/Spring assumptions removed from the Go core
- [x] Declarative `types` vocabulary in the profile (no hardcoded enum)
- [x] Multi-classification (a node may carry several types) — manifest
      schema v2 with the `Types` field
- [x] Profiles loaded from a directory layout (`*.yaml` + templates +
      query sets), not a single file
- [x] `spring-boot-hexagonal` rebuilt under the new declarative
      architecture as proof-of-concept

### Future epics

- [ ] Token-budget enforcement (`--budget` flag, priority-based trimming —
      flag is parsed today but unenforced)
- [ ] Tree-sitter query sets for TypeScript, Go, Python
- [ ] Profile: `nextjs-app-router`
- [ ] Profile: `go-standard`
- [ ] `jitctx stats` (manifest health + module distribution)
- [ ] First-class integration recipes for Claude Code, Cursor, Aider

## Contributing

jitctx is in early development. If you're interested in contributing - especially framework profiles or Tree-sitter query sets for new languages - open an issue to discuss before submitting a PR.

## License

[MIT](LICENSE)

---

<p align="center">
  <sub>Built by developers who got tired of paying for tokens their AI never needed.</sub>
</p>
