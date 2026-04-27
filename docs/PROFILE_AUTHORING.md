# jitctx Profile Authoring Guide

This document is the end-to-end reference for writing a jitctx **EP-04 profile** from scratch. It covers directory shape, the `profile.yaml` schema in full, classification rules, audit rule kinds, packaging, multi-classification semantics, and a worked example whose output passes `jitctx profile validate`.

Companion commands referenced throughout:

- `jitctx profile init <bundled-name>` — extract a bundled profile as an editable starting point.
- `jitctx profile validate <path>` — validate a profile directory against the schema.
- `jitctx scan` — run the scanner; applies classification and collects contracts.
- `jitctx audit` — run audit rules against the project-state manifest.

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [Directory Layout](#2-directory-layout)
3. [profile.yaml Schema Reference](#3-profileyaml-schema-reference)
4. [Type Declarations (`types[]` deep dive)](#4-type-declarations-types-deep-dive)
5. [Classification Rule Reference](#5-classification-rule-reference)
6. [Audit Rule Reference](#6-audit-rule-reference)
7. [Packaging Block](#7-packaging-block)
8. [Multi-Classification Deep Dive](#8-multi-classification-deep-dive)
9. [Worked Example: Build a Minimal Profile from Scratch](#9-worked-example-build-a-minimal-profile-from-scratch)
10. [Validation](#10-validation)
11. [Init from Bundled](#11-init-from-bundled)
12. [Legacy / EP-03 Backward-Compat Appendix](#12-legacy--ep-03-backward-compat-appendix)

---

## 1. Introduction

A **profile** is a named collection of declarative rules that tells jitctx how to classify code elements (classes, interfaces) in a particular framework or architectural style, and which architectural conventions to audit.

### When to author a profile

- Your project uses a framework or architecture not covered by the bundled profiles.
- You want to add project-specific audit rules on top of a bundled profile.
- You are adding a new canonical profile to the jitctx repository.

### Where profiles live

| Location | Path | Precedence |
|----------|------|------------|
| Bundled (embedded in the binary) | compiled in via `//go:embed` inside `internal/infrastructure/fsprofile/bundled/<name>/` | Lower |
| User (project-local override) | `.jitctx/profiles/<name>/` | Higher — user profile wins |

The resolve order (user profile wins over bundled when both names match) is implemented in the profile port chain; see `internal/domain/port/profile/` for the interfaces.

### Getting started

```bash
# Inspect an existing bundled profile and use it as a starting point:
jitctx profile init spring-boot-hexagonal

# The command writes to .jitctx/profiles/spring-boot-hexagonal/
# Edit the files there, then validate:
jitctx profile validate .jitctx/profiles/spring-boot-hexagonal/
```

The canonical bundled profile lives at `internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal/profile.yaml`. Every field used in that file is documented in this guide.

---

## 2. Directory Layout

A profile directory has the following shape:

```
<profile-name>/
  profile.yaml           # required — schema documented in Section 3
  templates/
    <basename>.tmpl      # one file per type declared in profile.yaml
    ...
  README.md              # optional — human-readable description
```

Rules:

- `profile.yaml` is the only required file. A directory without it fails validation with a "missing profile.yaml" error.
- Each `template:` value in `profile.yaml` must have a matching file inside `templates/`. The match is **case-sensitive** and is the basename only (no path separators). A missing template file produces a `*TemplateMissingError` (see [Section 10](#10-validation)).
- There is **no** `queries/` subdirectory in the EP-04 directory shape. Tree-sitter queries are bundled by language inside the binary (`internal/infrastructure/treesitter/bundledqueries/java/`). Profile authors do not write `.scm` files.

---

## 3. `profile.yaml` Schema Reference

The DTO that the loader decodes into is `bundleDTO` at `internal/infrastructure/fsprofile/bundleDto.go:15-27`. The top-level keys are listed in the table below; each is described in the subsections that follow.

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `name` | string | Yes | Profile identifier; used in error messages and `profile init` output. |
| `language` | string | Yes (EP-04) | Source language. Only `java` ships bundled Tree-sitter queries today. |
| `types` | list | No | EP-04 declarative type declarations (see Section 4). |
| `audit_rules` | list | No | Declarative audit rules (see Section 6). |
| `packaging` | structured block | No | Reserved/forward-compat block (see Section 7). |

Legacy EP-03 keys (`detect`, `module_detection`, `rules`, `query_lang`, `languages`) are documented in the [Legacy Appendix](#12-legacy--ep-03-backward-compat-appendix).

### 3.1 `name`

```yaml
name: my-profile
```

- Required string.
- Used in validation error messages and in the output of `jitctx profile init`.
- An empty or missing `name` fails validation: `missing required field: name`.

### 3.2 `language`

```yaml
language: java
```

- Required string in EP-04.
- Selects the Tree-sitter query set for source parsing.
- **EP-04 (this release) ships only the `java` query set.** The value objects `vo.ParseLanguage` (US-005) also recognise `go`, `typescript`, and `python`, but those languages do not have bundled queries; using them produces an unsupported-language error at load time: `language 'go' is not supported; available: <ids>`. Follow-up epics (US-007/US-009 and beyond) will expand the language set.
- Quoting from the .feature: `language 'cobol' is not supported` — any unrecognised value is rejected with that pattern.

### 3.3 `types[]`

A list of architectural type declarations. Each entry maps a type `id` to a `template` file, an optional human-readable `description`, and a `classification` list that defines which code elements are classified as this type. Details in [Section 4](#4-type-declarations-types-deep-dive).

```yaml
types:
  - id: service
    template: service.java.tmpl
    description: Application service.
    classification:
      - kind: class
        has_annotation: Service
```

### 3.4 `audit_rules[]`

A list of declarative audit rules evaluated by `jitctx audit`. Each entry selects a built-in evaluator (`kind`) and supplies kind-specific `params`. Details in [Section 6](#6-audit-rule-reference).

```yaml
audit_rules:
  - id: entity-path-mismatch
    kind: annotation_path_mismatch
    severity: ERROR
    description: 'Class @Entity must live under domain/'
    suggestion: 'Move {file} to a path containing "/domain/"'
    params:
      annotation: Entity
      path_required: /domain/
```

### 3.5 `packaging`

Reserved forward-compat block. See [Section 7](#7-packaging-block).

---

## 4. Type Declarations (`types[]` deep dive)

Each entry in the `types` list is a `bundleTypeDTO` (see `bundleDto.go:67-72`). Fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique identifier within the file. Becomes a `types[]` entry in `model.Contract`. |
| `template` | string | Yes | Basename of a file in `templates/`. Case-sensitive. |
| `description` | string | No | Human-readable description of the type. |
| `classification` | list of classification rules | No | OR-combined matching rules (see Section 5). |

### 4.1 `id`

- Must be unique within `profile.yaml`. Duplicate ids fail validation: `duplicate type id: <id>`.
- Profile authors may use any string. The bundled `spring-boot-hexagonal` profile uses ids such as `input-port`, `output-port`, `entity`, `service`, `rest-adapter`.
- The id appears in the manifest `types: []string` field of `model.Contract` (see `internal/domain/model/contract.go:29`).

### 4.2 `template`

- Basename (no directory separators) of a file inside the `templates/` subdirectory.
- The match is case-sensitive on all platforms.
- A missing template file produces a typed error: `profile %q: type %q references missing template %q` (`*TemplateMissingError`, `internal/domain/errors/errors.go:291-294`).

Example mapping:

```yaml
types:
  - id: input-port
    template: inputPort.tmpl      # → templates/inputPort.tmpl must exist
```

### 4.3 `description`

Optional free-form string. Not evaluated; only stored and surfaced in profile-level tooling output.

### 4.4 `classification[]`

The list of classification rules for this type. The matching semantics are:

- **OR across entries**: a code element is classified as this type when ANY rule in the list matches.
- **AND within a rule**: within a single rule, every non-empty field is an additional constraint that must hold simultaneously.

An empty `classification: []` list means the type never matches any code element declaratively (the bundled `aggregate-root` type uses this as an accepted gap; see the `Q5` comment in the bundled `profile.yaml`).

Multi-classification semantics — one element matching multiple type ids — are covered in [Section 8](#8-multi-classification-deep-dive).

---

## 5. Classification Rule Reference

Each entry in a `classification` list is a `bundleClassificationDTO` (see `bundleDto.go:89-95`) that maps directly to `model.ClassificationRule` (`internal/domain/model/classificationRule.go:29-35`).

| Field | Type | Description |
|-------|------|-------------|
| `kind` | string | `"class"` or `"interface"`. Empty = no constraint. |
| `implements_all` | list of strings | Subset match; see below. Empty/nil = no constraint. |
| `implements_none` | list of strings | Exclusion match; see below. Empty/nil = no constraint. |
| `has_annotation` | string | Single annotation name. Empty = no constraint. |
| `path_contains` | string | Substring of the file path. Empty = no constraint. |

### 5.1 `kind`

Constrains the syntactic kind of the code element.

- `"class"` — matches only class declarations.
- `"interface"` — matches only interface declarations.
- Empty string — matches both kinds (no constraint).
- Unknown values never match in the current implementation (US-002).

```yaml
classification:
  - kind: interface
    path_contains: /port/in/
```

### 5.2 `implements_all`

A list of glob patterns. For a rule to match, **every pattern** in the list must match at least one name in the code element's implemented-interface list. This is a **subset match** — the code element may implement additional interfaces beyond those in the list; extras are allowed.

Glob syntax:

- `"*UseCase"` — matches any name ending with `UseCase`.
- `"Foo*"` — matches any name beginning with `Foo`.
- `"FooBar"` — requires literal equality (no glob character → exact match).
- Only a single `*` is supported; it anchors at the start or end of the name.

```yaml
classification:
  - kind: class
    implements_all:
      - "*UseCase"
    path_contains: /service/
```

**Subset matching — extras allowed.** A rule with `implements_all: [CreateUserUseCase]` matches a class that implements `[CreateUserUseCase, ChangeUserStatusUseCase, DeleteUserUseCase]`. The extra interfaces do not block the match. This behaviour is pinned by the EP04US-002 scenario "Subset matching with extras allowed".

### 5.3 `implements_none`

Same glob syntax as `implements_all`. If **any** name in the code element's implemented-interface list matches **any** pattern in this slice, the rule fails. Use this to exclude elements that match a broader rule but should not be classified as this type.

```yaml
classification:
  - kind: class
    implements_all:
      - Repository
    implements_none:
      - Marker
```

A class implementing `[Repository, Marker]` does NOT match this rule (the `Marker` exclusion fires). A class implementing only `[Repository]` does match. This is pinned by the EP04US-002 scenario "implements_none excludes matching classes".

### 5.4 `has_annotation`

A single annotation name string. Matching is case-insensitive; a leading `@` prefix is tolerated.

- `"Service"`, `"service"`, and `"@Service"` all match a `@Service`-annotated class.
- Only a single value is accepted in the EP-04 schema (list form deferred per the comment at `bundleDto.go:87-88`).

```yaml
classification:
  - kind: class
    has_annotation: Entity
```

### 5.5 `path_contains`

A substring match against the forward-slash file path of the code element. Empty string = no constraint.

```yaml
classification:
  - kind: interface
    path_contains: /port/out/
```

The match is a plain substring check (not a glob, not a regex).

---

## 6. Audit Rule Reference

Each entry in `audit_rules` is a `bundleAuditRuleDTO` (see `bundleDto.go:74-83`) that maps directly to `model.AuditRule` (`internal/domain/model/auditRule.go:31-38`).

### Common fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Stable identifier; appears in report output and the per-project disable list. |
| `kind` | string | Yes | One of the five evaluator kinds (see subsections below). |
| `severity` | string | Yes | `ERROR`, `WARNING`, or `INFO`. |
| `description` | string | No | Human-readable rule description. |
| `suggestion` | string | No | Suggestion template; may contain tokens such as `{file}`, `{path}`, `{name}`. For the full token list, see `internal/domain/service/auditRuleEvaluator.go`. |
| `params` | map of strings | Depends on kind | Kind-specific parameters; see each subsection. Unknown keys are tolerated (forward-compatible). |

Severity vocabulary (from `model.AuditSeverity`, `internal/domain/model/auditRule.go:22-26`):

- `ERROR` — architectural violation; blocks clean audit.
- `WARNING` — convention issue; reported but does not block.
- `INFO` — informational; lowest severity.

### 6.1 `annotation_path_mismatch`

**What it checks:** A class annotated with a specific annotation must live under a required path fragment.

**Params:**

| Key | Required | Description |
|-----|----------|-------------|
| `annotation` | Yes | Annotation name to match (case-insensitive, `@` prefix tolerated). |
| `path_required` | Yes | Substring that must appear in the file path. |

**Examples from `spring-boot-hexagonal/profile.yaml`:**

```yaml
- id: entity-path-mismatch
  kind: annotation_path_mismatch
  severity: ERROR
  description: 'Class @Entity must live under domain/'
  suggestion: 'Move {file} to a path containing "/domain/"'
  params:
    annotation: Entity
    path_required: /domain/

- id: rest-controller-path-mismatch
  kind: annotation_path_mismatch
  severity: ERROR
  description: 'Class @RestController must live under adapter/in/web/'
  suggestion: 'Move {file} to a path containing "/adapter/in/web/"'
  params:
    annotation: RestController
    path_required: /adapter/in/web/

- id: repository-path-mismatch
  kind: annotation_path_mismatch
  severity: ERROR
  description: 'Class @Repository must live under adapter/out/persistence/'
  suggestion: 'Move {file} to a path containing "/adapter/out/persistence/"'
  params:
    annotation: Repository
    path_required: /adapter/out/persistence/
```

### 6.2 `implements_path_mismatch`

**What it checks:** A class implementing an interface that matches a glob must live under at least one of a comma-separated list of required path fragments.

**Params:**

| Key | Required | Description |
|-----|----------|-------------|
| `implements_glob` | Yes | Glob pattern (single `*`) matched against implemented interface names. |
| `path_required_any` | Yes | Comma-separated list of path substrings; the file path must contain at least one. |

**Example from `spring-boot-hexagonal/profile.yaml`:**

```yaml
- id: usecase-impl-path-mismatch
  kind: implements_path_mismatch
  severity: ERROR
  description: 'Class implementing *UseCase must live under application/ or service/'
  suggestion: 'Move {file} to a path containing "/application/" or "/service/"'
  params:
    implements_glob: '*UseCase'
    path_required_any: '/application/,/service/'
```

### 6.3 `interface_naming`

**What it checks:** Interfaces within a specific path must conform to a naming convention expressed as a suffix or a regular expression.

**Params:**

| Key | Required | Description |
|-----|----------|-------------|
| `path_required` | Yes | Path substring that scopes the rule to a package subtree. |
| `name_suffix` | One of these two | Interface name must end with this literal string. |
| `name_regex` | One of these two | Interface name must match this Go regular expression. |

**Examples from `spring-boot-hexagonal/profile.yaml`:**

```yaml
- id: port-naming
  kind: interface_naming
  severity: WARNING
  description: 'Interface in port/in/ must end with UseCase'
  suggestion: 'Rename {name} to end with UseCase'
  params:
    path_required: /port/in/
    name_suffix: UseCase

- id: outbound-port-naming
  kind: interface_naming
  severity: WARNING
  description: 'Interface in port/out/ must follow Repository or Gateway naming'
  suggestion: 'Rename {name} to end with Repository or Gateway'
  params:
    path_required: /port/out/
    name_regex: '.*(Repository|Gateway)$'
```

### 6.4 `forbidden_import`

**What it checks:** Files under a specific path must not import a particular package or package prefix.

**Params:**

| Key | Required | Description |
|-----|----------|-------------|
| `path_scope` | Yes | Path substring that scopes the rule to a file subtree. |
| `import_prefix` | One of these two | The import statement must NOT start with this prefix. |
| `import_prefix_substring` | One of these two | The import statement must NOT contain this substring. |

**Examples from `spring-boot-hexagonal/profile.yaml`:**

```yaml
- id: domain-leak
  kind: forbidden_import
  severity: ERROR
  description: 'Files under domain/ must not import org.springframework.*'
  suggestion: 'Remove the Spring import from {file}; move framework code to an adapter'
  params:
    path_scope: /domain/
    import_prefix: org.springframework.

- id: domain-adapter-inversion
  kind: forbidden_import
  severity: ERROR
  description: 'Files under domain/ must not import adapter packages'
  suggestion: 'Remove the adapter import from {file}; depend on output ports instead'
  params:
    path_scope: /domain/
    import_prefix_substring: .adapter.
```

### 6.5 `field_type_layer_violation`

**What it checks:** Classes within a specific path must not declare fields whose type name ends with a forbidden suffix. Used to prevent direct injection of concrete adapter implementations into service classes.

**Params:**

| Key | Required | Description |
|-----|----------|-------------|
| `path_scope` | Yes | Path substring that scopes the rule to a file subtree. |
| `forbidden_type_suffix` | Yes | Field type names must NOT end with this suffix. |

**Example from `spring-boot-hexagonal/profile.yaml`:**

```yaml
- id: adapter-injection
  kind: field_type_layer_violation
  severity: ERROR
  description: 'Service must inject the output port, not an adapter implementation directly'
  suggestion: 'In {file}, replace the adapter field type with the corresponding output port interface'
  params:
    path_scope: /service/
    forbidden_type_suffix: Jpa
```

---

## 7. Packaging Block

The `packaging` key is a **reserved, forward-compatible** structured block.

- **DTO field:** `bundleDTO.Packaging *yaml.Node` (`bundleDto.go:26`).
- **Storage:** the loader round-trips the raw bytes into `model.ProfileBundle.RawPackaging []byte` (`bundleMapper.go:125-131`, `profileBundle.go:32-35`).
- **Production consumers:** none today. No evaluator reads `RawPackaging` in EP-04. The bundled `spring-boot-hexagonal/profile.yaml` does not include a `packaging:` block.

Authors may include arbitrary YAML under this key; the loader will accept and preserve it for use by forthcoming features. Do not rely on any specific interpretation of the content until a future EP defines the DSL.

```yaml
# Forward-compat — safe to include arbitrary structured content.
# No operators are defined in EP-04; this block is a no-op today.
packaging:
  # Reserved for future packaging DSL (post-EP-04).
```

Because no operators are defined in EP-04, there are no operators to document. If a follow-up epic introduces packaging operators, this section will be extended in that PR.

---

## 8. Multi-Classification Deep Dive

### 8.1 OR-of-rules / AND-within-rule

Recall from Section 4:

- **OR across `classification` entries**: a code element is classified as a given type if ANY rule in the list matches.
- **AND within a rule**: all non-empty fields within one rule entry must hold simultaneously.

### 8.2 Subset matching — extras allowed

A rule with `implements_all: [X]` matches a class that implements `[X, Y, Z]`. The extra interfaces (`Y`, `Z`) do not block the match.

**Concrete scenario** (derived from the bundled `service` classification and EP04US-002, scenario "Subset matching with extras allowed"):

```yaml
classification:
  - kind: class
    implements_all:
      - "*UseCase"
    path_contains: /service/
```

A class located at `application/UserServiceImpl.java` that implements `[CreateUserUseCase, ChangeUserStatusUseCase, DeleteUserUseCase]` **matches** this rule because:

1. `kind: class` — satisfied (it is a class).
2. `implements_all: ["*UseCase"]` — every pattern (`*UseCase`) matches at least one implemented interface (`CreateUserUseCase`). The others are extra; they do not disqualify.
3. `path_contains: /service/` — `application/UserServiceImpl.java` does not contain `/service/`. In this specific example the path constraint would NOT match; use `/application/` as an additional OR-entry or adjust the path. The important point is that the extra interfaces in `implements_all` matching is the subset story.

For a concrete match: the same rule with `path_contains: /application/` would match `application/UserServiceImpl.java` implementing `[CreateUserUseCase, ChangeUserStatusUseCase, DeleteUserUseCase]`.

### 8.3 `implements_none` — exclusion example

`implements_none` disqualifies elements that match a broader positive rule.

**Scenario** (EP04US-002, "implements_none excludes matching classes"):

```yaml
classification:
  - kind: class
    implements_all:
      - Repository
    implements_none:
      - Marker
```

| Code element | implements | Matches? | Reason |
|---|---|---|---|
| `UserRepo` | `[Repository]` | Yes | Positive rule satisfied, no exclusion fires. |
| `UserRepo` | `[Repository, Marker]` | No | `Marker` matches `implements_none`; rule fails. |

### 8.4 Multi-classification — one element, multiple types

A single code element can match the classification rules of multiple type declarations. When it does, the manifest entry for that element carries **all** matching type ids in its `types: []string` field (see `model.Contract.Types`, `internal/domain/model/contract.go:29`).

**Worked scenario** (anchored on EP04US-003 scenario "Multi-classification stores all matched types"):

Suppose a profile declares two types:

```yaml
types:
  - id: output-adapter
    template: outputAdapter.java.tmpl
    classification:
      - kind: class
        implements_all:
          - Repository

  - id: cacheable-component
    template: cacheableComponent.java.tmpl
    classification:
      - kind: class
        implements_all:
          - Cacheable
```

A class implementing both `Repository` and `Cacheable` matches both type declarations. The resulting manifest entry is:

```yaml
# project-state.yaml (excerpt)
contracts:
  - name: UserRepositoryImpl
    types:
      - output-adapter
      - cacheable-component
    path: adapter/out/persistence/UserRepositoryImpl.java
```

Types are listed in the order declared in `profile.yaml`. A class matching zero rules has `types: []` (empty, non-nil).

---

## 9. Worked Example: Build a Minimal Profile from Scratch

<!-- This example mirrors testdata/ep04us008/workedExample/. Keep them in sync when editing. -->

This section walks you through creating a minimal profile that passes `jitctx profile validate`. The resulting files are identical to those in `testdata/ep04us008/workedExample/`.

### Step 1 — Create the profile directory

```bash
mkdir -p myProfile/templates
```

### Step 2 — Create `myProfile/profile.yaml`

```yaml
name: myProfile
language: java
types:
  - id: service
    template: service.java.tmpl
    description: Application service implementing one input port.
    classification:
      - kind: class
        has_annotation: Service
audit_rules:
  - id: service-naming
    kind: interface_naming
    severity: WARNING
    description: 'Service classes should use the *Service suffix.'
    suggestion: 'Rename {name} to end with Service'
    params:
      path_required: /service/
      name_suffix: Service
```

### Step 3 — Create `myProfile/templates/service.java.tmpl`

```bash
echo '// myProfile service stub' > myProfile/templates/service.java.tmpl
```

Or create the file manually with the content:

```
// myProfile service stub
```

### Step 4 — Validate the profile

```bash
jitctx profile validate myProfile/
```

### Step 5 — Expected output

```
Profile valid
```

Exit code: `0`.

Any validation error (missing name, duplicate type id, missing template, unsupported language) is printed to stderr. See [Section 10](#10-validation) for the common errors and their fixes.

---

## 10. Validation

Run `jitctx profile validate <path>` (added in US-007, commit `f438281`) to check a profile directory before committing it.

### Common errors

| Error message | Cause | Fix |
|---|---|---|
| `missing required field: name` | `name:` is absent or empty in `profile.yaml`. | Add `name: <your-profile-name>` at the top level. |
| `duplicate type id: <id>` | Two entries in `types[]` share the same `id`. | Make every `id` value unique within the file. |
| `unknown classification field '<field>'` | A classification entry contains a key that is not in the canonical set. | Check the key against `{kind, implements_all, implements_none, has_annotation, path_contains}`. A common cause is a typo such as `implementss`. |
| `profile %q: type %q references missing template %q` | A `template:` value has no matching file in `templates/`. | Create `templates/<basename>` with at least minimal content. |
| `language '<value>' is not supported; available: <ids>` | The `language:` value is not in the recognised set, or has bundled queries. | Change to `java` (the only language with bundled Tree-sitter queries in EP-04). |

### Validation is profile-only

`jitctx profile validate` checks the profile directory against the schema. It does not scan your source code. To classify and audit actual code, use `jitctx scan` followed by `jitctx audit`.

---

## 11. Init from Bundled

`jitctx profile init <bundled-name>` extracts a bundled profile into `.jitctx/profiles/<bundled-name>/` so you can edit it.

Bundled profiles available in EP-04:

| Name | Description |
|------|-------------|
| `spring-boot-hexagonal` | Spring Boot project following hexagonal (ports-and-adapters) architecture. |

```bash
# Extract to .jitctx/profiles/spring-boot-hexagonal/
jitctx profile init spring-boot-hexagonal

# Validate the extracted copy (should print "Profile valid"):
jitctx profile validate .jitctx/profiles/spring-boot-hexagonal/

# Edit the profile:
$EDITOR .jitctx/profiles/spring-boot-hexagonal/profile.yaml

# Re-validate after editing:
jitctx profile validate .jitctx/profiles/spring-boot-hexagonal/
```

The bundled `spring-boot-hexagonal` profile's own documentation is at `internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal/README.md`.

---

## 12. Legacy / EP-03 Backward-Compat Appendix

> **Note for new authors:** the fields in this appendix are preserved for backward compatibility with EP-03 profiles created before the EP-04 directory shape was introduced. If you are writing a new profile, use the EP-04 schema (`language`, `types`, `audit_rules`) and ignore this appendix.

The loader (`internal/infrastructure/fsprofile/bundleLoader.go:160-166`) uses `KnownFields(false)` so legacy keys decode silently without breaking the load. The following legacy keys are carried into the domain model via `bundleMapper.toBundleDomain`.

### Legacy top-level keys

| Key | DTO field | EP-03 semantics |
|-----|-----------|-----------------|
| `query_lang` | `bundleDTO.QueryLang string` | Explicit language override for Tree-sitter; superseded by `language` in EP-04. |
| `languages` | `bundleDTO.Languages []string` | Plural language list; singular `language` is the EP-04 replacement. |
| `detect` | `bundleDTO.Detect bundleDetectDTO` | Auto-detection rules: list of `{name, contains}` file matchers that trigger profile activation. |
| `module_detection` | `bundleDTO.ModuleDetection bundleModuleDetDTO` | Module detection strategy (`hexagonal`), root globs, and marker rules. |
| `rules` | `bundleDTO.Rules []bundleRuleDTO` | First-match-wins classification rules for the EP-03 scanner. Superseded by `types[].classification` in EP-04. |

### `detect` shape

```yaml
detect:
  files:
    - name: pom.xml
      contains: "org.springframework.boot"
```

### `module_detection` shape

```yaml
module_detection:
  strategy: hexagonal
  roots:
    - src/main/java/**/domain
    - src/main/java/**
  markers:
    - kind: path_contains
      value: /port/in/
    - kind: annotation
      value: Entity
```

### `rules` shape (EP-03 first-match-wins)

```yaml
rules:
  - match:
      node_type: interface_declaration
      path_contains: /port/in/
    classify_as: input-port

  - match:
      node_type: class_declaration
      has_annotation: Entity
    classify_as: entity
```

The `rules` block is evaluated by the EP-03 scanner path. When `types[]` is non-empty, the EP-04 declarative classifier takes precedence; the two systems coexist in the bundled `spring-boot-hexagonal` profile for full backward parity.
