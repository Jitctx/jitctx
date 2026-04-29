# Plan — PC01US-005 Enforce test method naming convention

## Section 0 — Summary

- Feature: a profile author can declare a single `method_naming` audit rule
  (`id: test-naming`) that fires on every method whose annotation set
  contains a configured trigger annotation (e.g. `@Test`) AND whose name
  does NOT match a configured Go regex pattern. The rule is method-scoped:
  it iterates `decl.Methods` of every visited declaration, inspects
  `method.Annotations` and `method.Name`, and emits one violation per
  offending method with evidence `name={method.Name}, expected_pattern={pattern}`
  so AC2's literal substring `name=testFindUser, expected_pattern=^should[A-Z].*_when[A-Z].*$`
  is producible verbatim.
- Requirement IDs covered: **PC01US-005** (story); directly **PC01RF-004**
  (method-scoped rules with regex name patterns), **PC01RF-009**
  (evidence-rich messages); cross-cutting **PC01RNF-001** (engine
  language-neutrality), **PC01RNF-003** (deterministic output),
  **PC01RNF-006** (real Tree-sitter on real `.java` fixtures).
- Layers touched: **domain (model, service), infrastructure (fsprofile
  mapper + treesitter parser), tests (unit + integration), testdata
  fixtures**. No `internal/application/usecase`, no `internal/cli/command`,
  no `wire.go`, no `format/` changes. Audit pipeline already orchestrates
  the new evaluator transparently because `EvaluateFile` dispatch is just
  one extra `case` arm.
- Tiers active: **1** (domain), **2** (infrastructure adapters), **6**
  (tests + fixtures). Tiers 3, 4, 5 collapse — see §5, §6, §7.
- Guidelines loaded:
  `.claude/guidelines/domain-layer-guidelines.yml`,
  `.claude/guidelines/infrastructure-layer-guidelines.yml`,
  `.claude/guidelines/unit-test-layer-guidelines.yml`,
  `.claude/guidelines/integration-test-layer-guidelines.yml`.
- Estimated file count: **0 new + 4 modified** in production source
  (`auditRule.go`, `javaFileSummary.go`, `auditRuleEvaluator.go`,
  `mapper.go`, `parser.go` modified) plus **1 modified unit-test file**,
  **1 new integration-test file**, **8 new fixture files** (two project
  fixtures: clean + violating; no exempt fixture — PC01US-005 does not
  require per-rule exemptions).

### Verified working-tree facts (2026-04-29)

1. `evalForbiddenAnnotations` at `internal/domain/service/auditRuleEvaluator.go:461-523`
   is the closest precedent: annotation-aware, target-scoped (class | field),
   honours `pathExempt`. The new `evalMethodNaming` follows the same shape
   but iterates `decl.Methods` and intersects against a SINGLE trigger
   annotation, then matches `method.Name` against a compiled regex.
2. `model.JavaMethod` at `internal/domain/model/javaFileSummary.go:34-37`
   currently exposes only `Signature string`. Method-level annotations,
   method names, and method-line numbers are NOT extracted today; they
   MUST be added in Tier 1 (model) and Tier 2 (parser), or the evaluator
   cannot inspect methods nor satisfy AC1/AC2.
3. `internal/infrastructure/treesitter/parser.go:292-305` — `extractMethods`
   already enumerates direct `method_declaration` children of a class or
   interface body and currently builds only a signature string. Adding
   `Name`, `Annotations`, and `Line` is purely additive: walk the same
   `node_declaration` children, capture the first `nodeIdentifier` (the
   method name — already done in `buildMethodSignature:319-322` but
   discarded), reuse the existing `extractAnnotations` helper at
   `parser.go:138-155` against the `nodeModifiers` child, and capture
   `int(node.StartPoint().Row) + 1` for the line.
4. `extractAnnotations(node, src)` at `parser.go:138-155` accepts ANY node
   whose children include `nodeAnnotation | nodeMarkerAnnotation |
   nodeNormalAnnotation`. `method_declaration`'s `modifiers` child uses
   the same `nodeModifiers` constant — verified by reading the
   tree-sitter Java grammar shape used by `extractClassDeclaration`. No
   new node-type constant is required.
5. `auditRuleDTO` at `internal/infrastructure/fsprofile/dto.go` decodes
   `params: map[string]string`. The two new param keys (`name_pattern`,
   `triggered_by`) piggyback on the existing string-map mechanism — no
   new DTO field, no DTO migration. Whitelist update in `mapper.go:11-19`
   is one new entry.
6. The user-profile loader (`fsprofile.NewDetectorWithLogger`) requires
   the project's `pom.xml` to contain `org.springframework.boot` (per
   `testdata/pc01us004ForbidAutowiredFieldInjection/projectViolating/pom.xml`).
   Fixtures MUST include this `pom.xml` verbatim or the user profile is
   silently ignored and the bundled profile (which lacks the new rule)
   is used instead.
7. The audit walker yields paths WITHOUT a leading slash (`src/test/java/...`).
   Per the path-scope substring quirk noted in PC01US-004 (T6-G5 had to fix
   `path_scope: /src/main/java/` → `path_scope: src/main/java/`), fixtures
   MUST use `path_scope: src/test/java/` (no leading slash).
8. `evalInterfaceNaming:142-188` is the only existing precedent for regex
   compilation in the evaluator. It uses `regexp.MustCompile(nameRegex)`
   on a profile-author-supplied pattern — i.e. it would PANIC on a malformed
   pattern. The new evaluator deviates from this: it uses `regexp.Compile`
   (NOT `MustCompile`); a compile error silently skips the rule (defensive
   parity with `evalRequiredAnnotations:294-297` and
   `evalForbiddenAnnotations:466-471`). Profile-validate (PC01US-011) is
   the layer that should reject malformed regexes at load time.
9. The reserved param-key registry (PC01US-004 plan §2.10) currently lists:
   `path_scope`, `annotations`, `node_types`, `target` (enum class | field),
   `exempt_paths`. PC01US-005 extends `target` with `method` and reserves
   two new keys: `name_pattern` (Go regex) and `triggered_by` (single
   annotation simple name). The shared `pathExempt` helper is reused
   for free if a profile author opts in to `exempt_paths` on a
   `method_naming` rule, but the AC does not require it.
10. Per `CLAUDE.md` "Nomes de arquivos Go", new Go file is camelCase
    (`testMethodNamingIntegration_test.go` for the integration test;
    no new evaluator unit-test file — additive edits to
    `auditRuleEvaluator_test.go`).
11. RNF-001 forbidden-token gate (`internal/qualitygate/exemptions.go`)
    lists `JUnit`, `Mockito`, `javax`, `spring`, `@Entity`,
    `@RestController`, `port/in`, `application/`. The trigger annotation
    `@Test` is a profile-author CONFIG value (lives in YAML), never an
    engine identifier — `Test` is not in the forbidden-token list either,
    but the evaluator code path NEVER names `Test` as a literal: it reads
    the value from `rule.Params["triggered_by"]`. Production source MUST
    NOT contain `Test`/`JUnit`/`Mockito`/`Spring`/`Lombok`/`Autowired`/`JPA`
    literals (PC01RNF-001) — this is automatic because the engine code
    is data-driven.

## Section 1 — File Set

| #  | File                                                                                                                  | Action  | Layer  | Tier | Group |
|----|-----------------------------------------------------------------------------------------------------------------------|---------|--------|------|-------|
| 1  | `internal/domain/model/auditRule.go`                                                                                   | modify  | domain | 1    | T1-G1 |
| 2  | `internal/domain/model/javaFileSummary.go`                                                                             | modify  | domain | 1    | T1-G1 |
| 3  | `internal/domain/service/auditRuleEvaluator.go`                                                                        | modify  | domain | 1    | T1-G1 |
| 4  | `internal/infrastructure/fsprofile/mapper.go`                                                                          | modify  | infra  | 2    | T2-G1 |
| 5  | `internal/infrastructure/treesitter/parser.go`                                                                         | modify  | infra  | 2    | T2-G2 |
| 6  | `internal/domain/service/auditRuleEvaluator_test.go`                                                                   | modify  | tests  | 6    | T6-G3 |
| 7  | `testdata/pc01us005TestMethodNaming/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml`                          | create  | tests  | 6    | T6-G1 |
| 8  | `testdata/pc01us005TestMethodNaming/projectClean/pom.xml`                                                              | create  | tests  | 6    | T6-G1 |
| 9  | `testdata/pc01us005TestMethodNaming/projectClean/project-state.yaml`                                                   | create  | tests  | 6    | T6-G1 |
| 10 | `testdata/pc01us005TestMethodNaming/projectClean/src/test/java/com/acme/UserServiceTest.java`                          | create  | tests  | 6    | T6-G1 |
| 11 | `testdata/pc01us005TestMethodNaming/projectViolating/.jitctx/profiles/spring-boot-hexagonal.yaml`                       | create  | tests  | 6    | T6-G2 |
| 12 | `testdata/pc01us005TestMethodNaming/projectViolating/pom.xml`                                                          | create  | tests  | 6    | T6-G2 |
| 13 | `testdata/pc01us005TestMethodNaming/projectViolating/project-state.yaml`                                               | create  | tests  | 6    | T6-G2 |
| 14 | `testdata/pc01us005TestMethodNaming/projectViolating/src/test/java/com/acme/UserServiceTest.java`                      | create  | tests  | 6    | T6-G2 |
| 15 | `internal/cli/command/testMethodNamingIntegration_test.go`                                                              | create  | tests  | 6    | T6-G4 |

(File counts: **0 new production Go files in T1, 0 new infra files in T2 —
all four production-source changes are additive edits to existing files**.
**1 modified unit-test file (T6-G3 appends new test functions to
`auditRuleEvaluator_test.go`)**, **1 new integration-test Go file**,
**8 new testdata fixture files**.)

## Section 2 — Frozen Domain Contract

Every signature below is consumed by Tier 2 and Tier 6. Once approved, no
downstream group may rename a field, swap a type, or add a positional
parameter.

### 2.1 New `AuditRuleKind` enum value

In `internal/domain/model/auditRule.go`:

```go
const (
    AuditKindAnnotationPathMismatch  AuditRuleKind = "annotation_path_mismatch"
    AuditKindImplementsPathMismatch  AuditRuleKind = "implements_path_mismatch"
    AuditKindInterfaceNaming         AuditRuleKind = "interface_naming"
    AuditKindForbiddenImport         AuditRuleKind = "forbidden_import"
    AuditKindFieldTypeLayerViolation AuditRuleKind = "field_type_layer_violation"
    AuditKindRequiredAnnotations     AuditRuleKind = "required_annotations"
    AuditKindForbiddenAnnotations    AuditRuleKind = "forbidden_annotations"

    // AuditKindMethodNaming enforces a regex on method names, gated by
    // the presence of a single trigger annotation. The rule fires on
    // every method inside a visited declaration whose annotation set
    // contains params["triggered_by"] AND whose Name does NOT match
    // params["name_pattern"] (a Go regexp). PC01RF-004, PC01RF-009.
    AuditKindMethodNaming AuditRuleKind = "method_naming"
)
```

### 2.2 Extended `JavaMethod` (additive)

In `internal/domain/model/javaFileSummary.go`:

```go
// JavaMethod represents a method extracted from a Java declaration.
type JavaMethod struct {
    Signature   string   // "ReturnType name(params)" — existing field, unchanged.
    Name        string   // method identifier, e.g. "shouldReturnUser_whenIdExists". NEW.
    Annotations []string // simple names, no leading @ (e.g. ["Test"]). NEW. Empty when not extracted.
    Line        int      // 1-based line of the method_declaration node. NEW. 0 if unknown.
}
```

The three new fields are **append-only** at the end of the struct. Existing
callers that construct `JavaMethod{Signature: ...}` keep compiling
unchanged because Go zero-values the new fields.

### 2.3 New evaluator dispatch arm

In `internal/domain/service/auditRuleEvaluator.go`, the switch in
`EvaluateFile` adds exactly one arm:

```go
case model.AuditKindMethodNaming:
    got = evalMethodNaming(moduleID, summary, rule)
```

### 2.4 New private evaluator function (frozen signature)

```go
// evalMethodNaming — params:
//
//   "path_scope":    substring restricting which files this rule applies to
//                    (e.g. "src/test/java/"). REQUIRED.
//   "triggered_by":  the SINGLE annotation simple name (without leading "@")
//                    that gates the rule. The rule only inspects methods
//                    whose Annotations slice contains this value.
//                    REQUIRED, non-empty.
//   "name_pattern":  a Go regexp the method.Name MUST match. A method whose
//                    Name does NOT match emits one violation. REQUIRED,
//                    non-empty. Compile failure (malformed regex) skips
//                    the rule silently — profile-validate (PC01US-011)
//                    is the layer that rejects malformed regexes at load
//                    time.
//   "node_types":    optional comma-joined list of declaration node types
//                    whose Methods are scanned. Default "class_declaration".
//                    "*" matches any.
//   "exempt_paths":  optional comma-joined list of forward-slash globs.
//                    Reuses the shared pathExempt helper.
//
// Substitution context:
//
//   {file}             — summary.Path
//   {name}             — method.Name
//   {expected_pattern} — params["name_pattern"] (verbatim)
//   {triggered_by}     — params["triggered_by"] (verbatim)
//
// Violation Line:
//
//   method.Line (1-based; 0 when the parser did not capture it).
//
// PC01RF-004 / PC01RF-009.
func evalMethodNaming(
    moduleID string, summary model.JavaFileSummary, rule model.AuditRule,
) []auditvo.AuditViolation
```

### 2.5 Reuse of existing helpers

The new evaluator reuses these helpers already defined in
`auditRuleEvaluator.go`:

- `splitNonEmpty(s string) []string` — for `node_types`.
- `nodeTypeAllowed(nodeType string, allowed []string) bool` — node-type
  filter.
- `pathExempt(rule model.AuditRule, path string) bool` — shared
  exempt-paths gate. NO new behaviour; PC01US-004 already published the
  contract.
- `makeViolation(...)` — uniform violation construction with substitution
  context.
- `substituteSuggestion(template string, ctx map[string]string) string` —
  handled inside `makeViolation`.

NO `intersectAnnotations` reuse — this evaluator checks for a single
annotation via `slices.Contains`, not a multi-annotation intersection.
NO new helpers introduced.

### 2.6 Profile-loader whitelist

`internal/infrastructure/fsprofile/mapper.go`:

```go
var knownAuditRuleKinds = map[model.AuditRuleKind]bool{
    model.AuditKindAnnotationPathMismatch:  true,
    model.AuditKindImplementsPathMismatch:  true,
    model.AuditKindInterfaceNaming:         true,
    model.AuditKindForbiddenImport:         true,
    model.AuditKindFieldTypeLayerViolation: true,
    model.AuditKindRequiredAnnotations:     true,
    model.AuditKindForbiddenAnnotations:    true,
    model.AuditKindMethodNaming:            true, // NEW
}
```

### 2.7 New error sentinels

**None.** A malformed `method_naming` rule (empty `path_scope`, empty
`triggered_by`, empty/malformed `name_pattern`) is silently skipped by
the evaluator (defensive posture, mirroring `evalRequiredAnnotations:294-297`
and `evalForbiddenAnnotations:466-471`). PC01US-011 plan — not this story —
is responsible for adding profile-validate-time errors on these conditions.

### 2.8 `Deps` struct

**Unchanged.** No new use case; no new port. The existing
`Deps.Audit audituc.UseCase` consumes the new rule transparently.

### 2.9 Reserved param-key registry (extended)

The following keys are reserved in `AuditRule.Params`. PC01US-005 adds two
new rows and extends the `target` enum domain:

| Key            | Type                  | Owner kinds                                                                                       |
|----------------|-----------------------|---------------------------------------------------------------------------------------------------|
| `path_scope`   | substring             | `forbidden_import`, `field_type_layer_violation`, `required_annotations`, `forbidden_annotations`, `method_naming` |
| `annotations`  | comma-joined list     | `required_annotations`, `forbidden_annotations`                                                   |
| `node_types`   | comma-joined list     | `required_annotations`, `forbidden_annotations`, `method_naming`                                  |
| `target`       | enum string           | `forbidden_annotations` (class \| field). PC01US-005 does NOT consume `target` because the kind itself is method-scoped. Future stories MAY extend the enum (e.g. `target=method`, `target=supertype`). |
| `exempt_paths` | comma-joined list     | `forbidden_annotations`, `method_naming`                                                          |
| `name_pattern` | Go regex              | `method_naming` (NEW; PC01RF-004)                                                                 |
| `triggered_by` | single annotation     | `method_naming` (NEW; PC01RF-004)                                                                 |

Rationale for NOT reusing `target=method`: `method_naming` has
fundamentally different parameters from `forbidden_annotations` (regex
vs. annotation-set, single-trigger vs. multi-annotation). A separate
kind keeps the per-kind contract focused and the dispatch table flat.
The `target=method` enum value is RESERVED for a future
`forbidden_annotations` extension that flags methods carrying a
forbidden annotation (out of scope here).

## Section 3 — Domain Layer Plan (Tier 1 — single group T1-G1)

### 3.1 `model/auditRule.go`

Append `AuditKindMethodNaming` at the bottom of the existing
`const ( ... )` block per §2.1. No other changes.

### 3.2 `model/javaFileSummary.go`

Append `Name string`, `Annotations []string`, and `Line int` to
`JavaMethod` per §2.2. Update the field's doc comment to document the
new fields. No other changes.

### 3.3 `service/auditRuleEvaluator.go`

Two additive edits:

1. New `case model.AuditKindMethodNaming` arm in the `EvaluateFile`
   switch — single line plus the existing pattern.
2. New `evalMethodNaming` function per §2.4. Body shape:

   ```text
   pathScope    ← rule.Params["path_scope"]
   triggeredBy  ← rule.Params["triggered_by"]
   namePattern  ← rule.Params["name_pattern"]
   if pathScope == "" or triggeredBy == "" or namePattern == "" → return nil  // defensive
   if !strings.Contains(summary.Path, pathScope) → return nil
   if pathExempt(rule, summary.Path) → return nil
   re, err ← regexp.Compile(namePattern)
   if err != nil → return nil   // malformed regex; profile-validate is the gate
   nodeTypes    ← splitNonEmpty(params["node_types"]); default ["class_declaration"]
   for each decl in summary.Declarations:
     if !nodeTypeAllowed(decl.NodeType, nodeTypes) → continue
     for each method in decl.Methods:
       if !slices.Contains(method.Annotations, triggeredBy) → continue
       if re.MatchString(method.Name) → continue
       ctx ← {
         "file":             summary.Path,
         "name":             method.Name,
         "expected_pattern": namePattern,
         "triggered_by":     triggeredBy,
       }
       emit violation(line=method.Line, ctx=ctx)
   return violations
   ```

   Notes:
   - Order is deterministic: outer loop walks `summary.Declarations`
     (slice — declaration order); inner loop walks `decl.Methods`
     (slice — declaration order). No map iteration anywhere. Multiple
     violations within a single file are emitted in declaration order.
   - `regexp.Compile` (NOT `MustCompile`) — see §2.4 doc and Open
     Question Q1.
   - `slices.Contains` is the existing dependency already imported by
     `evalAnnotationPathMismatch:80`.

### 3.4 No new domain ports

The evaluator is a pure function over `JavaFileSummary`. No new port.
PC01RF-010 (language-adapter abstraction) is satisfied because the
evaluator never names Java/Spring/Lombok/Autowired/Test/JUnit in its
source — `triggered_by` is a profile YAML value. **Verify by grep
before merging T1-G1**:

```
grep -E '(?i)(autowired|spring|lombok|junit|mockito|jpa)' \
     internal/domain/service/auditRuleEvaluator.go
```

The expected match-set is empty.

## Section 4 — Infrastructure Layer Plan (Tier 2 — two groups, parallel)

### 4.1 T2-G1 — `fsprofile/mapper.go` whitelist

Single-line addition per §2.6. No DTO change. No mapper logic change
(the existing pass-through of `params: map[string]string` already
carries `triggered_by`, `name_pattern`, `exempt_paths`, etc. through
unmodified).

### 4.2 T2-G2 — `treesitter/parser.go` `extractMethods` + `buildMethodSignature`

Goal: populate `JavaMethod.Name`, `JavaMethod.Annotations`, and
`JavaMethod.Line`.

**Edit 1 (`parser.go` `extractMethods` at line 292-305):**

Refactor `extractMethods` so that, for each `method_declaration` child,
it produces a fully-populated `JavaMethod` with:

- `Signature` — preserves the existing behaviour by calling
  `buildMethodSignature(child, src)`.
- `Name` — extracted by walking `method_declaration` children and
  capturing the first `nodeIdentifier` (this is exactly what
  `buildMethodSignature:319-322` already does, but that local is
  discarded). The simplest refactor extracts a small helper
  `extractMethodName(node *sitter.Node, src []byte) string` that
  returns the first `nodeIdentifier` child as text; both
  `extractMethods` and `buildMethodSignature` call it. (Alternatively,
  inline the loop directly in `extractMethods`.)
- `Annotations` — walk children for a `nodeModifiers` child and pass
  it to the existing `extractAnnotations(modifiersChild, src)` helper.
  Discard the qualified slice (FQNs are unused for this story; future
  stories can add a `JavaMethod.QualifiedAnnotations` mirror of
  `JavaDeclaration.QualifiedAnnotations`).
- `Line` — `int(node.StartPoint().Row) + 1`. Same convention as
  `extractFieldDeclaration:121` and `comments.go` line capture.

A method is appended to the result slice only when
`buildMethodSignature` returns a non-empty signature AND `Name != ""`
(parity with the existing guard at line 299).

**Edit 2 (no change to `buildMethodSignature`):**

`buildMethodSignature` continues to compute the signature string
independently. Optionally factor out the shared name-extraction loop
into `extractMethodName` for DRY; non-blocking refactor. The frozen
contract only requires `JavaMethod.Name` to be populated by the
caller — implementation detail.

**Edit 3 (no `.scm` query work):**

The Tree-sitter Java grammar already exposes `method_declaration` and
`modifiers`/`annotation` children directly via the AST walk used by
`extractMethods`. The bundled `.scm` queries (`bundledqueries/`) are
used by the classifier, not by `parser.go`. No `bundledqueries/` edit
is required for this story.

**Edit 4 (no change to `fields.go`):**

`fields.go` `extractClassFields` does NOT call `extractMethods` (it is
the field-only view used by `ListJavaFields`). PC01US-005 does not
need to populate methods in the field-only view, so `fields.go` is
unmodified.

### 4.3 No `fsmanifest` change

Same rationale as PC01US-004 §4.3. Violations are use-case output, not
manifest content.

## Section 5 — Application Layer Plan

**N/A.** `internal/application/usecase/audituc/usecase.go` orchestrates
the audit pipeline. The new `AuditKindMethodNaming` is dispatched
inside the evaluator's own switch — application code does not name the
kind.

PC01RNF-003 (deterministic output) is preserved: the existing sort key
already includes `Line`, so method-line violations on different lines
sort stably; same-line violations sort by rule ID (already
deterministic).

## Section 6 — Presentation Layer Plan

**N/A.** `auditCmd.go` is unchanged. `format/audit.go` already prints
`[ruleID] file:line  message` for any non-zero `Line`. AC2's
`name=testFindUser, expected_pattern=^should[A-Z].*_when[A-Z].*$`
literal is satisfied by the rule's `description` template (in YAML)
plus the `{name}` and `{expected_pattern}` substitution tokens
populated by `evalMethodNaming`.

## Section 7 — Composition Root + Tests Plan

### 7.1 Wiring

**No edits** to `wire.go`, `root.go`, `execute.go`, `cmd/jitctx/main.go`,
`internal/config/`. The audit vertical slice is complete.

### 7.2 Unit tests — T6-G3

**File**: `internal/domain/service/auditRuleEvaluator_test.go` (additive
edits — new test functions appended; existing tests unchanged).

Helper:

```go
// methodNamingRule returns the canonical method-scope rule fixture for
// PC01US-005: "Test methods must follow the should*_when* naming convention".
func methodNamingRule() model.AuditRule {
    return model.AuditRule{
        ID:          "test-naming",
        Kind:        model.AuditKindMethodNaming,
        Severity:    model.AuditSeverityError,
        Description: "Test method {name} violates naming convention; expected_pattern={expected_pattern}",
        Suggestion:  "Rename {name} to match {expected_pattern}",
        Params: map[string]string{
            "path_scope":   "src/test/java/",
            "triggered_by": "Test",
            "name_pattern": "^should[A-Z].*_when[A-Z].*$",
            "node_types":   "class_declaration",
        },
    }
}
```

New table-driven test functions (each `t.Parallel()`):

1. `TestAuditEvaluator_MethodNaming_Compliant_NoViolation`
   — backs **AC1**. One declaration with one method named
   `shouldReturnUser_whenIdExists`, annotations `["Test"]`. Asserts
   zero violations.
2. `TestAuditEvaluator_MethodNaming_NonCompliant_FlagsWithEvidence`
   — backs **AC2**. Method `testFindUser`, annotations `["Test"]`.
   Asserts one violation; `v.Message` contains the literal substring
   `name=testFindUser, expected_pattern=^should[A-Z].*_when[A-Z].*$`.
3. `TestAuditEvaluator_MethodNaming_UntriggeredMethodIgnored`
   — method `testFindUser` but NO `@Test` annotation. Asserts zero
   violations (rule is gated by `triggered_by`).
4. `TestAuditEvaluator_MethodNaming_OutsidePathScopeIgnored`
   — file path `src/main/java/...`, otherwise-violating method. Asserts
   zero violations.
5. `TestAuditEvaluator_MethodNaming_RespectsExemptPaths`
   — file path `src/test/java/com/acme/legacy/OldTest.java`, rule
   `exempt_paths: **/legacy/**`. Asserts zero violations even though
   the method violates the regex.
6. `TestAuditEvaluator_MethodNaming_LinePropagation`
   — `method.Line == 17`. Asserts `v.Line == 17` (PC01RF-009 evidence
   surfacing).
7. `TestAuditEvaluator_MethodNaming_MalformedRuleEmitsNothing`
   — empty `triggered_by`, empty `name_pattern`, malformed regex
   (`name_pattern: "[unclosed"`), missing `path_scope` — each yields
   zero violations (defensive parity with
   `RequiredAnnotations_MalformedRuleEmitsNothing`).
8. `TestAuditEvaluator_MethodNaming_MultipleMethodsMixed`
   — three methods on one class: one compliant, one non-compliant, one
   without `@Test`. Asserts exactly one violation pointing to the
   non-compliant method's line and name.

All tests build `model.JavaFileSummary` literals directly — no parser,
no infra. Pure-function coverage.

### 7.3 Integration test — T6-G4

**File**: `internal/cli/command/testMethodNamingIntegration_test.go`.

**Helper**: replicate the `newAuditCmdFor*(t, workDir, manifestPath)`
pattern from `forbidAutowiredFieldInjectionIntegration_test.go:26-72`
inline as `newAuditCmdForTestMethodNaming`. Same constructor
arguments to `appaudituc.New(...)`. (Same Q3-style resolution as
PC01US-004: no upstream DRY refactor — the helper-extraction
discussion is a follow-up PR.)

**Test functions** (each `t.Parallel()`):

1. `TestAuditCmd_Integration_TestMethodNaming_CompliantNoViolation`
   — copies `projectClean` into `t.TempDir()`, runs
   `audit --dir <tmp> --manifest <tmp>/project-state.yaml`, asserts
   `require.NoError(t, err)`, stdout does NOT contain `[test-naming]`.
   **Backs PC01US-005 AC1** (`PC01RF-004`).
2. `TestAuditCmd_Integration_TestMethodNaming_NonCompliantFlagsViolation`
   — copies `projectViolating` into `t.TempDir()`, runs the same
   command, asserts no error, stdout contains `[test-naming]`,
   stdout contains the literal `name=testFindUser, expected_pattern=^should[A-Z].*_when[A-Z].*$`,
   stdout contains `UserServiceTest.java:<line>` (the line of the
   `testFindUser` method declaration in the pinned fixture),
   `strings.Count(out, "[test-naming]") == 1`. **Backs PC01US-005
   AC2** (`PC01RF-004`, `PC01RF-009`).
3. `TestAuditCmd_Integration_TestMethodNaming_Determinism`
   — runs the `projectViolating` fixture twice in two separate
   `t.TempDir()`s, normalises temp-dir prefix, asserts byte-identical
   stdout. **Backs PC01RNF-003** for the new rule wiring.

Why no golden file: same rationale as PC01US-004 — `require.Contains`
on rule ID, evidence, and offending file path tracks the AC text
exactly without coupling to formatter cosmetics.

### 7.4 Fixture content

#### 7.4.1 Profile YAML (both projects share an identical YAML body)

The profile head (`name`, `languages`, `query_lang`, `detect`,
`module_detection`, classification `rules`) is copied verbatim from
`testdata/pc01us004ForbidAutowiredFieldInjection/projectViolating/.jitctx/profiles/spring-boot-hexagonal.yaml`
lines 1-81. The audit rule body is replaced with a single rule:

```yaml
audit_rules:
  - id: test-naming
    kind: method_naming
    severity: ERROR
    description: 'Test method violates naming convention: name={name}, expected_pattern={expected_pattern}'
    suggestion: 'Rename {name} to match {expected_pattern}'
    params:
      path_scope: src/test/java/
      triggered_by: 'Test'
      name_pattern: '^should[A-Z].*_when[A-Z].*$'
      node_types: class_declaration
```

The `description` template embeds BOTH literal substrings the
integration test asserts: `name={name}` and `expected_pattern={expected_pattern}`.
After substitution by `makeViolation`, AC2 produces the verbatim
`name=testFindUser, expected_pattern=^should[A-Z].*_when[A-Z].*$`.

`path_scope: src/test/java/` (no leading slash) per fact #7 above.

No `exempt_paths` because PC01US-005 does not require per-rule
exemptions; the engine `pathExempt` helper is shared and inert when
the key is absent.

#### 7.4.2 Java fixtures

**`projectClean/src/test/java/com/acme/UserServiceTest.java`** — one
test method named `shouldReturnUser_whenIdExists`, annotated `@Test`.
Expected: zero violations. Sketch (line numbers explicit so the file
is reproducible):

```
1  package com.acme;
2
3  import org.junit.jupiter.api.Test;
4
5  public class UserServiceTest {
6
7      @Test
8      public void shouldReturnUser_whenIdExists() {
9      }
10 }
```

**`projectViolating/src/test/java/com/acme/UserServiceTest.java`** —
one test method named `testFindUser`, annotated `@Test`. Expected:
ONE violation pointing to the method line. The method MUST sit on a
known line so the integration test can assert
`UserServiceTest.java:<line>`. Suggested layout (the
`method_declaration` start row is line 8 — same as PC01US-004's
`field_declaration:8` reasoning, where the annotation is a sibling
`modifiers` child whose start row is line 7 but the
`method_declaration` node itself starts on line 8):

```
1  package com.acme;
2
3  import org.junit.jupiter.api.Test;
4
5  public class UserServiceTest {
6
7      @Test
8      public void testFindUser() {
9      }
10 }
```

> **Line-number pinning**: the integration test asserts
> `strings.Contains(out, "UserServiceTest.java:7")` OR
> `UserServiceTest.java:8` depending on Tree-sitter's
> `method_declaration` start row. Per Tree-sitter Java grammar and
> by analogy with PC01US-004's verified field-line behaviour
> (`field_declaration` includes the modifiers child), the
> `method_declaration` node's `StartPoint().Row` is the row of the
> FIRST modifier (the `@Test` line, row 7 → `Line = 7`). The unit
> test in T6-G3 (case 6) MUST be aligned with this: assert
> `v.Line == 7` against a `JavaMethod{Line: 7}` literal. The
> integration test (T6-G4 case 2) asserts
> `strings.Contains(out, "UserServiceTest.java:7")`. See Q5.

#### 7.4.3 `pom.xml` (both projects, identical)

Byte-for-byte copy of
`testdata/pc01us004ForbidAutowiredFieldInjection/projectViolating/pom.xml`
(contains `org.springframework.boot` so the user-profile detector
loads the new rule).

#### 7.4.4 `project-state.yaml` (both projects)

Minimal `schema_version: 2` manifest declaring one module rooted at
the file's package. Suggested shape for both `projectClean` and
`projectViolating`:

```yaml
schema_version: 2
generated_at: 2026-04-29T00:00:00Z
stack:
  languages:
    - java
  frameworks:
    - spring-boot-hexagonal
modules:
  - id: com.acme
    path: src/test/java/com/acme
    tags: []
    contracts:
      - name: UserServiceTest
        types:
          - service
        path: src/test/java/com/acme/UserServiceTest.java
        methods: []
    dependencies: []
contexts: []
```

## Section 8 — Open Questions & Risks

### Open questions (all resolved — no Blocking: Yes)

| #  | Question                                                                                                                                       | Blocking | Resolution |
|----|------------------------------------------------------------------------------------------------------------------------------------------------|----------|------------|
| Q1 | `regexp.MustCompile` (panicking) vs `regexp.Compile` (defensive) — which does the new evaluator use?                                            | No | `regexp.Compile`. A malformed pattern silently skips the rule (parity with `evalRequiredAnnotations:294-297` and `evalForbiddenAnnotations:466-471`). `evalInterfaceNaming:161` uses `MustCompile` — that is a known divergence; PC01US-011 (profile-validate) is the layer that should reject malformed regexes at load time. The new evaluator MUST NOT panic on user input. |
| Q2 | Should `method_naming` reuse `target=method` instead of being a separate kind?                                                                  | No | Separate kind. `method_naming` has fundamentally different params (regex + single trigger annotation) from `forbidden_annotations` (annotation set intersection). Conflating them would force the dispatch table to peek at `target` AND `name_pattern`/`triggered_by` simultaneously. Keeping `method_naming` as its own kind keeps each evaluator's contract focused. The `target=method` enum value is RESERVED for a future `forbidden_annotations` extension. |
| Q3 | Should the rule's `triggered_by` accept a comma-joined list (so multiple framework annotations like `Test`, `ParameterizedTest` can gate)?      | No | Single value for this story. AC1/AC2 only mention `@Test`. Future stories MAY widen `triggered_by` to a comma-joined list — the param key already accommodates that without a kind rename. The frozen evaluator MUST treat it as a single simple-name comparison via `slices.Contains` for now. |
| Q4 | What if the profile author writes `name_pattern` as a literal regex containing a comma — would `splitNonEmpty` corrupt it?                       | No | `name_pattern` is read directly via `rule.Params["name_pattern"]` and never passed through `splitNonEmpty`. Only `node_types` and `exempt_paths` are split. The frozen contract is explicit on this. |
| Q5 | Does Tree-sitter put `method_declaration.StartPoint().Row` at the `@Test` line or at the `public void ...` line?                                | No | At the `@Test` line. Tree-sitter's `method_declaration` includes the `modifiers` child (which contains the annotation node) — the node's start byte/row is the start of the FIRST modifier. Verified by analogy with PC01US-004 fact #4 (`extractFieldDeclaration:121` captures `int(node.StartPoint().Row) + 1` and the integration test asserts `Foo.java:7` for `@Autowired` on line 7 + field declarator on line 8 — see `forbidAutowiredFieldInjectionIntegration_test.go:92-93`). The fixture and the AC assertions are aligned to `Line = 7` for both unit and integration tests. |
| Q6 | Should the bundled spring-boot-hexagonal profile gain the new `method_naming` rule?                                                              | No | Out of scope (same rationale as PC01US-004 Q6). Bundled-defaults expansion is a separate decision tracked under PROFILE_AUTHORING.md updates. The new rule lives only in the two new fixture profiles. |
| Q7 | `JavaMethod.Annotations` addition — will it break existing fixture-snapshot tests in `parser_test.go`?                                          | No | The three new fields are zero-valued for tests that build `JavaMethod` literals without naming them. Only tests that do `require.Equal(t, expectedSlice, gotSlice)` on the whole `[]JavaMethod` need patching. T2-G2 owners run `go test ./internal/infrastructure/treesitter/...` and patch any literal-equality failures by inserting the new fields. Strictly mechanical. |
| Q8 | Constructor declarations — do they appear as `method_declaration` and would they accidentally trigger the rule?                                 | No | Tree-sitter's Java grammar emits `constructor_declaration` (NOT `method_declaration`) for constructors. `extractMethods` already filters on `nodeMethodDecl` (`"method_declaration"`) and does NOT pick up constructor declarations. Verified by reading `parser.go:296-303` and `queries.go:32`. |
| Q9 | Does PC01RF-008 (`exempt_paths` cross-cutting) actually apply to `method_naming`?                                                                 | No | Yes — engine-wide. `pathExempt` is invoked at the top of `evalMethodNaming` BEFORE the regex compile. Tested by case 5 in T6-G3 (RespectsExemptPaths). The rule keyword does not appear in AC1/AC2 but the engine helper is free to share. |
| Q10 | What about JUnit 5 `@org.junit.jupiter.api.Test` written as a fully-qualified annotation?                                                        | No | `extractAnnotations` in `parser.go:138-155` returns BOTH simple names and qualified names; we only consume the simple-names slice (`Annotations`). Fully-qualified `@org.junit.jupiter.api.Test` produces simple `Test` in the slice, so `slices.Contains(method.Annotations, "Test")` matches. Same behaviour as `forbidden_annotations` for class/field annotations. Out of AC1/AC2 but locked by the existing parser. |

### Risks

| ID    | Risk                                                                                                                                  | Probability | Impact | Mitigation |
|-------|---------------------------------------------------------------------------------------------------------------------------------------|-------------|--------|------------|
| R-001 | Tree-sitter does not surface annotations on `method_declaration` via `nodeModifiers` (different grammar shape than `class_declaration`) | Low         | High   | `extractAnnotations:138-155` is node-agnostic — it walks any node's children for `nodeAnnotation | nodeMarkerAnnotation | nodeNormalAnnotation`. The tree-sitter Java grammar uses `modifiers` as the parent node for both class and method annotations. T2-G2 must include a parser unit test that parses a real `.java` source with `@Test` on a method and asserts `method.Annotations == ["Test"]`. (Add to `parser_test.go` as part of T2-G2.) |
| R-002 | `JavaMethod` field additions break fixture-equality tests in `parser_test.go`                                                          | Low         | Medium | Same posture as PC01US-004 R-002: zero-valued for non-asserting tests; T2-G2 runs `go test ./internal/infrastructure/treesitter/...` and patches any literal-equality failures. |
| R-003 | Two consecutive runs produce different violation order due to map iteration                                                            | Low         | High   | `evalMethodNaming` walks `summary.Declarations` and `decl.Methods` in declaration order (slices). `regexp.MatchString` is deterministic. T6-G4 Determinism test asserts byte equality. |
| R-004 | RNF-001 grep at PC01US-014 catches `Test` / `JUnit` in the new test file                                                                | Low         | Low    | RNF-001 is scoped to `internal/domain`, `internal/application`, `internal/cli/*.go` (non-test). The new integration-test file is `*_test.go` — exempt by the same convention used in PC01US-004 (`forbidAutowiredFieldInjectionIntegration_test.go` already names `Autowired` extensively). The fixture YAML and Java files live under `testdata/` which the gate filters out. |
| R-005 | Integration test asserts `UserServiceTest.java:7` but fixture line drift breaks the test                                                | Low         | Low    | Fixture content is committed verbatim and reviewed; the file is small (10 lines). Mitigation: assert the literal `UserServiceTest.java:7` and document the pinning in a fixture comment. Same posture as PC01US-004 R-008. |
| R-006 | `name_pattern` regex contains characters interpreted by YAML (e.g. `:` in `^should[A-Z].*_when[A-Z].*$` could accidentally trigger flow scalar)  | Low         | Medium | The pattern as written contains only `^`, `[`, `]`, `*`, `_`, `$`, letters — none of which trigger YAML special handling. The fixture wraps the value in single quotes (`'^should[A-Z].*_when[A-Z].*$'`) for belt-and-suspenders safety. |
| R-007 | A method with multiple matching annotations (e.g. `@Test @ParameterizedTest`) is double-counted                                          | Very Low    | Low    | The `triggered_by` value is a SINGLE simple name. `slices.Contains` returns once; the violation is emitted once per method. Even if the method has multiple annotations matching the trigger (impossible for a single string `triggered_by`), the inner loop has no `break` requirement because it only emits one violation per method (regex match is per-method, not per-annotation). |
| R-008 | `regexp.Compile` is invoked once per file × per rule, which is wasteful                                                                  | Low         | Low    | Out of AC scope. Optimisation (compile lazily once per rule and cache on the rule struct) is a follow-up. The audit pipeline is not on a hot path; correctness > microsecond perf for this story. |
| R-009 | The user writes `triggered_by: "@Test"` (with leading `@`) and the simple-name comparison fails                                           | Low         | Medium | The frozen contract documents `triggered_by` as "without leading `@`" (per §2.4 doc). Profile-validate (PC01US-011) is the layer to reject the leading-`@` typo at load time. The evaluator is permissive: a typo silently flags zero methods. |

## Section 9 — Parallel Execution Plan

```yaml
tiers:
  - id: 1
    name: Domain contract — kind enum, method model, evaluator dispatch
    depends_on: []
    groups:
      - id: T1-G1
        scope:
          create: []
          modify:
            - internal/domain/model/auditRule.go
            - internal/domain/model/javaFileSummary.go
            - internal/domain/service/auditRuleEvaluator.go
        guidelines:
          - .claude/guidelines/domain-layer-guidelines.yml
        effort: M
        notes: >
          Single coordinated edit across three domain files. Adds
          AuditKindMethodNaming to the AuditRuleKind enum, appends
          Name, Annotations, and Line fields to JavaMethod, and adds
          one private helper to auditRuleEvaluator.go: evalMethodNaming
          (the new dispatch arm). Reuses pathExempt, nodeTypeAllowed,
          splitNonEmpty, and makeViolation from PC01US-004 already on
          main; introduces no new shared helper. Uses regexp.Compile
          (NOT MustCompile) — see Q1. The evaluator file MUST stay
          free of any Java/Spring/Lombok/JUnit identifier
          (PC01RNF-001) — verified by grep before merge. Param-key
          registry (Section 2.9) is the authoritative contract; no
          downstream tier may rename a key.

  - id: 2
    name: Infrastructure adapters (parallel)
    depends_on: [1]
    groups:
      - id: T2-G1
        scope:
          create: []
          modify:
            - internal/infrastructure/fsprofile/mapper.go
        guidelines:
          - .claude/guidelines/infrastructure-layer-guidelines.yml
        effort: S
        notes: >
          Single-line whitelist addition: knownAuditRuleKinds gains
          model.AuditKindMethodNaming. No DTO change; params
          map[string]string already round-trips arbitrary keys
          including triggered_by, name_pattern, node_types,
          exempt_paths. Independent of T2-G2.

      - id: T2-G2
        scope:
          create: []
          modify:
            - internal/infrastructure/treesitter/parser.go
        guidelines:
          - .claude/guidelines/infrastructure-layer-guidelines.yml
        effort: M
        notes: >
          Populate JavaMethod.Name, JavaMethod.Annotations, and
          JavaMethod.Line from the Tree-sitter AST. extractMethods at
          parser.go:292-305 walks each method_declaration child; for
          each one, capture line = int(node.StartPoint().Row) + 1
          once, walk the method_declaration children for a
          nodeIdentifier (the name — first identifier child) and a
          nodeModifiers child (passed to the existing
          extractAnnotations helper for simple-name extraction). The
          existing buildMethodSignature continues to compute the
          signature; the result struct populates all four JavaMethod
          fields. No .scm query work — the bundled queries directory
          is untouched. Existing parser_test.go assertions that
          compare full JavaMethod structs may need their literals
          patched to include the new zero-valued fields — owners run
          go test ./internal/infrastructure/treesitter/... and patch
          any equality failures. Adds at least one new positive
          parser_test.go assertion confirming method.Annotations
          contains "Test" when the source declares @Test on a method
          (R-001 mitigation). Independent of T2-G1.

  - id: 6
    name: Tests + fixtures (parallel within tier)
    depends_on: [1, 2]
    groups:
      - id: T6-G1
        scope:
          create:
            - testdata/pc01us005TestMethodNaming/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us005TestMethodNaming/projectClean/pom.xml
            - testdata/pc01us005TestMethodNaming/projectClean/project-state.yaml
            - testdata/pc01us005TestMethodNaming/projectClean/src/test/java/com/acme/UserServiceTest.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Clean fixture for PC01US-005 AC1 (compliant test method).
          pom.xml and profile head copied verbatim from
          testdata/pc01us004ForbidAutowiredFieldInjection/projectViolating.
          The single audit rule is test-naming / kind method_naming /
          triggered_by Test / name_pattern ^should[A-Z].*_when[A-Z].*$ /
          path_scope src/test/java/ / node_types class_declaration.
          UserServiceTest.java declares one method named
          shouldReturnUser_whenIdExists annotated @Test (compliant) so
          the evaluator emits zero violations. Independent of T6-G2 /
          T6-G3 / T6-G4 — parallel-safe.

      - id: T6-G2
        scope:
          create:
            - testdata/pc01us005TestMethodNaming/projectViolating/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us005TestMethodNaming/projectViolating/pom.xml
            - testdata/pc01us005TestMethodNaming/projectViolating/project-state.yaml
            - testdata/pc01us005TestMethodNaming/projectViolating/src/test/java/com/acme/UserServiceTest.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Violating fixture for PC01US-005 AC2. Profile YAML, pom.xml,
          project-state.yaml byte-identical to T6-G1.
          UserServiceTest.java is pinned to 10 lines so the
          method_declaration starts on line 7 (the @Test annotation
          line) — integration test asserts
          strings.Contains(out, "UserServiceTest.java:7") and
          strings.Contains(out, "name=testFindUser, expected_pattern=^should[A-Z].*_when[A-Z].*$").
          The single test method is named testFindUser, annotated
          @Test (non-compliant: fails the should*_when* regex).
          Independent of T6-G1 / T6-G3 / T6-G4.

      - id: T6-G3
        scope:
          create: []
          modify:
            - internal/domain/service/auditRuleEvaluator_test.go
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: M
        notes: >
          Append eight new test functions per Section 7.2:
          - TestAuditEvaluator_MethodNaming_Compliant_NoViolation (AC1)
          - TestAuditEvaluator_MethodNaming_NonCompliant_FlagsWithEvidence (AC2)
          - TestAuditEvaluator_MethodNaming_UntriggeredMethodIgnored
          - TestAuditEvaluator_MethodNaming_OutsidePathScopeIgnored
          - TestAuditEvaluator_MethodNaming_RespectsExemptPaths
          - TestAuditEvaluator_MethodNaming_LinePropagation
          - TestAuditEvaluator_MethodNaming_MalformedRuleEmitsNothing
          - TestAuditEvaluator_MethodNaming_MultipleMethodsMixed
          Each test builds model.JavaFileSummary literals directly
          (no parser, no infra). All t.Parallel(). Independent of
          T6-G1 / T6-G2 / T6-G4.

      - id: T6-G4
        scope:
          create:
            - internal/cli/command/testMethodNamingIntegration_test.go
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: M
        notes: >
          Integration test exercising audit end-to-end via real
          treesitter, fsprofile, fsmanifest, audituc adapters.
          Replicates the newAuditCmdFor*(t, workDir, manifestPath)
          helper inline (renamed newAuditCmdForTestMethodNaming) —
          same constructor parameters as
          forbidAutowiredFieldInjectionIntegration_test.go:26-72.
          Three test functions per Section 7.3:
          - TestAuditCmd_Integration_TestMethodNaming_CompliantNoViolation (AC1)
          - TestAuditCmd_Integration_TestMethodNaming_NonCompliantFlagsViolation (AC2)
          - TestAuditCmd_Integration_TestMethodNaming_Determinism (PC01RNF-003)
          Each uses t.TempDir() + t.Parallel(). Asserts
          string-contains rather than golden file. Reuses the existing
          copyFixture / fixtureDir helpers from
          internal/cli/command/helpers_test.go. Depends on T6-G1 /
          T6-G2 fixture paths existing at runtime; no compile-time
          dependency on other Tier 6 groups.
```
