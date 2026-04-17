<p align="center">
  <h1 align="center">jitctx</h1>
  <p align="center">
    <strong>Stop feeding your AI the entire project. Load only what it needs.</strong>
  </p>
  <p align="center">
    <a href="#quick-start">Quick Start</a> •
    <a href="#why-jitctx">Why</a> •
    <a href="#how-it-works">How It Works</a> •
    <a href="#tree-sitter--framework-profiles">Framework Profiles</a> •
    <a href="#roadmap">Roadmap</a> •
    <a href="README.pt-BR.md">Português</a>
  </p>
  <p align="center">
    <img src="https://img.shields.io/badge/status-work%20in%20progress-yellow" alt="Status: WIP" />
    <img src="https://img.shields.io/github/license/jitctx/jitctx" alt="License" />
    <img src="https://img.shields.io/badge/lang-Go-00ADD8?logo=go" alt="Go" />
  </p>
</p>

---

**jitctx** is a CLI tool that gives AI coding agents exactly the context they need - nothing more, nothing less. Instead of dumping your entire codebase, guidelines, and specs into the context window, jitctx loads only the relevant fragments based on what the agent is working on.

Think of it as **lazy loading for AI context**. Your agent asks for what it needs, and jitctx delivers - fast, structured, and within a token budget you control.

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
# Install
go install github.com/jitctx/jitctx@latest

# Scan your project (generates project-state.yaml)
jitctx scan

# Query context for a specific module
jitctx query --module user-management

# Query by tags
jitctx query --tag auth,security --type guidelines

# Infer module from the file being edited
jitctx query --file src/main/java/com/app/user/domain/User.java

# Query with a token budget
jitctx query --module billing --budget 2000

# See the parallel execution plan
jitctx plan --module user-management

# List available modules and tags
jitctx list modules
jitctx list tags
```

## How It Works

jitctx operates in two phases: **scan** and **query**.

### Phase 1: Scan

```
Your Codebase ──► jitctx scan ──► project-state.yaml
```

The `scan` command analyzes your project structure and generates a `project-state.yaml` manifest. jitctx uses [Tree-sitter](https://tree-sitter.github.io/) to parse source code into ASTs across 100+ languages, then applies **framework profiles** (declarative YAML rules) to classify what it finds. A Spring Boot profile knows that `@Entity` means an aggregate, that `port/in/` contains input ports, and that `@RestController` marks a REST adapter.

The manifest is a structured map of your project: modules, entities, ports, adapters, endpoints, contracts, dependencies, and links to your engineering artifacts (guidelines, requirements, test scenarios).

### Phase 2: Query

```
AI Agent ──► jitctx query --module billing --type guidelines ──► stdout (filtered context)
```

The `query` command reads the manifest, filters by module, type, tags, or file path, respects the token budget, and outputs only the relevant context fragments to stdout. Your AI agent calls it via bash and receives exactly what it needs.

### In Practice with Claude Code

Add this to your `CLAUDE.md`:

```markdown
## Context Loading

Before implementing any task, load context using jitctx:

    jitctx query --module <module> --type <types> [--tags <tags>] [--budget <tokens>]

Examples:
- Backend feature: `jitctx query --module user-management --type guidelines,requirements --tags java,api`
- Tests: `jitctx query --module billing --type scenarios`
- Full module context: `jitctx query --module notifications`

Always use jitctx. Never read .jitctx/ files directly.
```

### Example Output

```bash
$ jitctx query --module user-management --type guidelines --tags security
```

```markdown
<!-- jitctx: 2 contexts loaded | ~830 tokens | module=user-management -->

---
<!-- source: .jitctx/guidelines/java-conventions.md | tags: java, api, security -->

# Java Conventions
...

---
<!-- source: .jitctx/guidelines/security.md | tags: security, auth -->

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

## Contracts and Parallel Execution

jitctx can pre-declare the skeleton of your architecture - interfaces, types, and method signatures - before a single line of implementation exists. This unlocks **deterministic parallelism** for multi-agent workflows.

### How it works

```bash
$ jitctx plan --module user-management
```

```
Layer 0 [parallel - no dependencies]:
  ├── User.java (aggregate-root)
  ├── Role.java (entity)
  ├── CreateUserUseCase.java (input-port)
  └── UserRepository.java (output-port)

Layer 1 [parallel - depends on layer 0]:
  ├── UserServiceImpl.java (service → implements CreateUserUseCase)
  ├── UserController.java (rest-adapter → uses CreateUserUseCase)
  └── UserRepositoryJpa.java (jpa-adapter → implements UserRepository)

Layer 2 [sequential - depends on layer 1]:
  └── UserIntegrationTest.java
```

Each agent receives the contracts it needs via `jitctx contracts`:

```bash
$ jitctx contracts --for adapter/in/web/UserController.java
```

```markdown
## CreateUserUseCase (input-port)
Path: com.app.user.port.in.CreateUserUseCase

public interface CreateUserUseCase {
    UserResponse execute(CreateUserCommand cmd);
}
```

Agent B can implement `UserController` against this contract while Agent A implements `UserServiceImpl` - in parallel, without conflicts.

## Tree-sitter + Framework Profiles

jitctx doesn't ship one parser per language. It uses [Tree-sitter](https://tree-sitter.github.io/) for universal AST parsing and **framework profiles** (declarative YAML) to interpret what the AST means in your architecture.

```
Source Code ──► Tree-sitter (AST) ──► Framework Profile (rules) ──► project-state.yaml
```

### How it works

Tree-sitter parses your code into an AST. The framework profile tells jitctx how to classify what it finds:

```yaml
# profiles/spring-boot-hexagonal.yaml
name: spring-boot-hexagonal
detect:
  files: ["pom.xml", "build.gradle"]
  dependencies: ["org.springframework.boot"]

rules:
  - match:
      node_type: interface_declaration
      path_contains: "port/in"
    classify_as: input-port

  - match:
      node_type: class_declaration
      has_annotation: "Entity"
    classify_as: entity

  - match:
      node_type: class_declaration
      has_annotation: "RestController"
    classify_as: rest-adapter
    extract_endpoints: true

  - match:
      node_type: class_declaration
      implements: "*UseCase"
    classify_as: service
```

### Built-in profiles

| Profile | Detects | Classifies | Status |
|---------|---------|------------|--------|
| `spring-boot-hexagonal` | `pom.xml`, `build.gradle` | Ports, adapters, entities, controllers, services | 🚧 In progress |
| `nextjs-app-router` | `package.json`, `next.config.*` | Routes, components, API handlers, hooks, types | 📋 Planned |
| `go-standard` | `go.mod` | Packages, interfaces, structs, handlers | 📋 Planned |

### Adding a new framework

Supporting a new framework means writing a YAML file, not Go code. No recompilation needed:

```bash
# Drop a profile into your project
cp my-django-profile.yaml .jitctx/profiles/

# jitctx auto-discovers and applies it
jitctx scan
```

This is what makes jitctx scalable. The community can contribute profiles for Django, FastAPI, Gin, Axum, NestJS, Laravel, Rails, or any other framework without touching the core codebase. A profile is ~20-40 lines of YAML.

### Tree-sitter query sets

Each language needs a small set of Tree-sitter queries (~10-15 lines) to extract structural elements. These are `.scm` files bundled with the binary:

```scheme
;; queries/java.scm
(interface_declaration name: (identifier) @name) @interface
(class_declaration name: (identifier) @name) @class
(method_declaration name: (identifier) @name) @method
(annotation name: (identifier) @annotation)
```

```scheme
;; queries/typescript.scm
(interface_declaration name: (type_identifier) @name) @interface
(class_declaration name: (type_identifier) @name) @class
(export_statement declaration: (_) @exported)
```

jitctx ships with query sets for Java, TypeScript, Go, and Python out of the box. Adding a new language is a matter of writing a `.scm` file with the relevant node types.

## Token Budget Control

The `--budget` flag is how you keep costs predictable:

```bash
$ jitctx query --module user-management --budget 2000
```

jitctx sorts contexts by priority, accumulates estimated token counts, and stops when the budget is reached. The output header tells the agent what was trimmed:

```markdown
<!-- jitctx: 3/5 contexts loaded | ~1850 tokens | budget=2000 | trimmed: 2 low-priority scenarios -->
```

The agent knows more context exists and can request it in a follow-up call if needed.

## Roadmap

- [x] Project concept and architecture design
- [ ] Core: YAML manifest schema and parser
- [ ] Core: Query engine (module, tag, type, file, budget)
- [ ] Core: Output formatter (markdown, JSON, raw)
- [ ] Core: Dependency graph resolver
- [ ] Core: Plan generator (parallel execution layers)
- [ ] Core: Contracts extractor
- [ ] Tree-sitter: Integration and AST extraction engine
- [ ] Tree-sitter: Query sets for Java, TypeScript, Go, Python
- [ ] Profile: `spring-boot-hexagonal`
- [ ] Profile: `nextjs-app-router`
- [ ] Profile: `go-standard`
- [ ] CLI: `jitctx scan`, `jitctx query`, `jitctx plan`, `jitctx contracts`, `jitctx list`
- [ ] CLI: `jitctx stats` (token savings metrics)
- [ ] Integration docs for Claude Code, Cursor, Aider
- [ ] Scaffold command (generate interface stubs from contracts)

## Contributing

jitctx is in early development. If you're interested in contributing - especially framework profiles or Tree-sitter query sets for new languages - open an issue to discuss before submitting a PR.

## License

[MIT](LICENSE)

---

<p align="center">
  <sub>Built by developers who got tired of paying for tokens their AI never needed.</sub>
</p>
