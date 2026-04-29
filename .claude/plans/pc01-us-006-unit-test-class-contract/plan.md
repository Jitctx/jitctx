# Plan — PC01US-006 Unit-Test Class Contract (`@ExtendWith` + `@DisplayName`)

## Section 0 — Summary

- Feature: **Require `@ExtendWith` and `@DisplayName` on unit-test classes**, with optional argument-value matching for the trigger annotation (`@ExtendWith(MockitoExtension.class)`).
- User Story: **PC01US-006**
- Requirement IDs covered:
  - **PC01RF-001** — multi-annotation required-set, all-of semantics (extension to current evaluator).
  - **PC01RF-007** — annotation-with-arguments matching (NEW capability landed by this story).
  - **PC01RNF-001** — engine language-neutrality.
  - **PC01RNF-003** — deterministic output.
  - **PC01RNF-006** — real Tree-sitter on real `.java` fixtures.
  - **PC01RF-009** — evidence-rich messages (literal substring assertions).
- Acceptance scenarios mapped 1:1 in §6/§7:
  - **AC1** (clean) — both annotations + correct argument value → zero violations.
  - **AC2** (wrong arg) — `@ExtendWith(SpringExtension.class)` against rule expecting `MockitoExtension.class` → message contains `annotation=ExtendWith, expected_value=MockitoExtension.class, actual=SpringExtension.class`.
  - **AC3** (missing `@DisplayName`) → message contains `missing=[DisplayName]`.
- Layers touched: **domain**, **infrastructure** (treesitter only — no `fsprofile/mapper.go` change because `required_annotations` is already whitelisted), **tests** (unit + integration + fixtures).
- Tiers active: **1, 2, 6**. Tiers 3, 4, 5 are `N/A` (no use-case orchestration change, no new cobra command, no wiring change — `evalRequiredAnnotations` already plugs into `AuditEvaluator.EvaluateFile`).
- Guidelines loaded:
  - `.claude/guidelines/domain-layer-guidelines.yml`
  - `.claude/guidelines/infrastructure-layer-guidelines.yml`
  - `.claude/guidelines/unit-test-layer-guidelines.yml`
  - `.claude/guidelines/integration-test-layer-guidelines.yml`
- Estimated file count: **8 new** (3 fixture trees, 1 integration test, 1 plan, plus 3 fixture YAMLs counted below) and **3 modified**.

---

## Section 1 — File Set

| #  | File                                                                                                                              | Action  | Layer  | Tier | Group  | Requirements |
|----|-----------------------------------------------------------------------------------------------------------------------------------|---------|--------|------|--------|--------------|
| 1  | `internal/domain/model/javaFileSummary.go`                                                                                        | modify  | domain | 1    | T1-G1  | PC01RF-007 |
| 2  | `internal/domain/service/auditRuleEvaluator.go`                                                                                   | modify  | domain | 1    | T1-G1  | PC01RF-001, PC01RF-007, PC01RNF-001, PC01RNF-003, PC01RF-009 |
| 3  | `internal/infrastructure/treesitter/parser.go`                                                                                    | modify  | infra  | 2    | T2-G1  | PC01RF-007, PC01RNF-006 |
| 4  | `internal/infrastructure/treesitter/parser_test.go`                                                                               | modify  | infra  | 6    | T6-G1  | PC01RF-007, PC01RNF-006 |
| 5  | `internal/domain/service/auditRuleEvaluator_test.go`                                                                              | modify  | domain | 6    | T6-G2  | PC01RF-001, PC01RF-007, PC01RF-009, PC01RNF-001, PC01RNF-003 |
| 6  | `internal/cli/command/unitTestClassContractIntegration_test.go`                                                                    | create  | tests  | 6    | T6-G3  | PC01RF-001, PC01RF-007, PC01RF-009, PC01RNF-003, PC01RNF-006 |
| 7  | `testdata/pc01us006UnitTestClassContract/projectClean/pom.xml`                                                                    | create  | tests  | 6    | T6-G4  | PC01RNF-006 |
| 8  | `testdata/pc01us006UnitTestClassContract/projectClean/project-state.yaml`                                                         | create  | tests  | 6    | T6-G4  | PC01RNF-006 |
| 9  | `testdata/pc01us006UnitTestClassContract/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml`                                | create  | tests  | 6    | T6-G4  | PC01RF-001, PC01RF-007 |
| 10 | `testdata/pc01us006UnitTestClassContract/projectClean/src/main/java/com/acme/UserServiceTest.java`                                | create  | tests  | 6    | T6-G4  | PC01RF-001, PC01RF-007 |
| 11 | `testdata/pc01us006UnitTestClassContract/projectWrongArg/pom.xml`                                                                 | create  | tests  | 6    | T6-G5  | PC01RNF-006 |
| 12 | `testdata/pc01us006UnitTestClassContract/projectWrongArg/project-state.yaml`                                                      | create  | tests  | 6    | T6-G5  | PC01RNF-006 |
| 13 | `testdata/pc01us006UnitTestClassContract/projectWrongArg/.jitctx/profiles/spring-boot-hexagonal.yaml`                             | create  | tests  | 6    | T6-G5  | PC01RF-007 |
| 14 | `testdata/pc01us006UnitTestClassContract/projectWrongArg/src/main/java/com/acme/UserServiceTest.java`                             | create  | tests  | 6    | T6-G5  | PC01RF-007 |
| 15 | `testdata/pc01us006UnitTestClassContract/projectMissingDisplayName/pom.xml`                                                       | create  | tests  | 6    | T6-G6  | PC01RNF-006 |
| 16 | `testdata/pc01us006UnitTestClassContract/projectMissingDisplayName/project-state.yaml`                                            | create  | tests  | 6    | T6-G6  | PC01RNF-006 |
| 17 | `testdata/pc01us006UnitTestClassContract/projectMissingDisplayName/.jitctx/profiles/spring-boot-hexagonal.yaml`                   | create  | tests  | 6    | T6-G6  | PC01RF-001 |
| 18 | `testdata/pc01us006UnitTestClassContract/projectMissingDisplayName/src/main/java/com/acme/UserServiceTest.java`                   | create  | tests  | 6    | T6-G6  | PC01RF-001 |

**Coverage notes:**
- File 1 (model) and File 2 (evaluator) ride together in one Tier 1 group because the evaluator immediately consumes the new `JavaDeclaration.AnnotationArgs` field — splitting them would require a stub round-trip.
- File 3 (parser) is the only Tier 2 producer; it depends on File 1's freshly published shape but never on File 2's behavior.
- All test files Tier 6, partitioned by file — see §9.

---

## Section 2 — Frozen Domain Contract

The contract below is **frozen**: downstream tiers consume it verbatim. No tier may rename a field, add a parameter, or change a substitution-token spelling.

### 2.1 `model.JavaDeclaration` — new field `AnnotationArgs`

```go
// JavaDeclaration represents a top-level type declaration in a Java file.
type JavaDeclaration struct {
    NodeType             string       // "class_declaration" | "interface_declaration" | "enum_declaration" | "record_declaration"
    Name                 string       // simple name
    Annotations          []string     // simple names, no leading @ (e.g. ["ExtendWith", "DisplayName"])
    QualifiedAnnotations []string     // same length and order as Annotations
    Implements           []string
    Extends              []string
    Methods              []JavaMethod
    Fields               []JavaField

    // AnnotationArgs maps simple annotation name → the text of the first
    // positional argument as it appears in source, including:
    //   - quotes for string literals: "User service tests"
    //   - ".class" suffix for class literals: MockitoExtension.class
    //
    // The map entry is set to "" when the annotation is a marker
    // annotation (no argument list, e.g. @Override). Multi-positional
    // arguments are out of scope for PC01US-006: only the first
    // positional argument's text is captured (Q2 in §8).
    //
    // The map is keyed by SIMPLE annotation name (matching entries in
    // Annotations[]). When the same simple annotation appears more than
    // once on the same declaration (rare and outside AC scope) the LAST
    // occurrence wins; that ambiguity is documented in §8 as Q9.
    //
    // Empty map (or nil) when the language adapter did not extract
    // arguments — evaluators must treat both nil and "" entries as
    // "no argument captured".
    //
    // PC01RF-007.
    AnnotationArgs map[string]string
}
```

### 2.2 Evaluator extension — `evalRequiredAnnotations` accepts new optional param `expected_values`

**Decision (Q1):** EXTEND the existing `evalRequiredAnnotations` function in place. **No new `AuditRuleKind`** is introduced. Rationale: PC01RF-001 already binds the `required_annotations` kind to "multi-annotation required-set", and PC01RF-007 is a refinement (same path-scope, same target nodes, same all-of semantics) that adds *value* checking on the same set. A new kind would force profile authors to duplicate `path_scope`, `annotations`, and `node_types` between two rules.

#### Updated header doc-comment (frozen)

```go
// evalRequiredAnnotations — params:
//
//   "path_scope":      substring restricting which files this rule applies to
//                      (e.g. "src/main/java/"). REQUIRED.
//   "annotations":     comma-joined list of annotation simple names (without
//                      the leading "@") that must ALL be present on every
//                      matching declaration. REQUIRED, non-empty. Order is
//                      preserved and used to derive deterministic
//                      "missing=[...]" evidence.
//   "expected_values": OPTIONAL comma-joined list of "Annotation=Value" pairs
//                      (e.g. "ExtendWith=MockitoExtension.class"). For each
//                      pair, when the annotation IS present on a matching
//                      declaration, the evaluator compares the text of its
//                      first positional argument (decl.AnnotationArgs[ann])
//                      against the right-hand value. The comparison is exact
//                      string equality. A mismatch emits ONE additional
//                      violation per pair, separate from the missing-set
//                      violation. PC01RF-007.
//                      Parsing rules:
//                        - splits on "," only;
//                        - each piece is split on the FIRST "=";
//                        - whitespace around the annotation name AND value is
//                          trimmed;
//                        - a piece without "=" is ignored (defensive);
//                        - duplicate keys: LAST occurrence wins (deterministic
//                          on a given input string).
//                      Limitation: argument values containing commas are NOT
//                      supported. Profile authors needing such values must
//                      wait for a future-extension key (out of scope; Q7).
//   "node_types":      optional comma-joined list of declaration node types
//                      the rule applies to. Default "class_declaration". Use
//                      "*" or empty to skip the node-type filter.
//
// Substitution context (per emitted violation):
//
//   {file}     — summary.Path
//   {name}     — declaration simple name
//   {required} — comma-joined params["annotations"] (verbatim, in order)
//   {evidence} — for the missing-violation: "missing=[A,B,...]" subset
//                NOT present, ordered by params["annotations"];
//              — for an arg-mismatch violation: literal
//                "annotation=<ann>, expected_value=<expected>, actual=<actual>"
//                where <actual> is decl.AnnotationArgs[<ann>] (may be "").
//
// The rule's Description template SHOULD include "{evidence}" — the
// evaluator does not embed the prefix "missing=" or "annotation=" into the
// message itself; those literals are produced by the evaluator and inserted
// via the {evidence} substitution token. Profile authors keep ONE
// description template and the evaluator chooses which evidence string to
// substitute per violation.
//
// Determinism (PC01RNF-003):
//   - "expected_values" is parsed into an ORDERED slice of pairs in the
//     order the pairs appear in the input string (Q4).
//   - The evaluator iterates that ordered slice; it never iterates a Go
//     map for emit ordering.
//   - When BOTH a missing-violation AND one or more arg-mismatch
//     violations apply to the same declaration, the missing violation is
//     emitted FIRST, then arg-mismatch violations in the
//     "expected_values" order.
//
// PC01RF-001 (all-of presence), PC01RF-007 (argument matching),
// PC01RNF-001 (engine language-neutrality — no Java/Spring identifiers
// referenced here), PC01RF-009 (evidence-rich messages).
func evalRequiredAnnotations(
    moduleID string, summary model.JavaFileSummary, rule model.AuditRule,
) []auditvo.AuditViolation
```

#### Frozen behavior matrix

| Declaration state                                                | Violations emitted (in order)                                                                                  |
|------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------|
| All required present, all expected_values match                  | none                                                                                                            |
| Some required missing, no expected_values entries trigger        | 1 — `{evidence}` = `missing=[X,Y]`                                                                              |
| All required present, one expected_values mismatch               | 1 — `{evidence}` = `annotation=A, expected_value=E, actual=X`                                                   |
| Some required missing AND an expected_values entry mismatches on a *present* annotation | 2 — first the missing-violation, then the mismatch (preserving "expected_values" order)               |
| `expected_values` references an annotation that is missing       | NO mismatch violation for that pair — already covered by the missing-violation; per-pair iteration `continue`s. |
| `expected_values` references a marker annotation present on decl | mismatch fires when expected != "" (actual = "")                                                                |

#### Private helper added to `auditRuleEvaluator.go`

```go
// expectedValuePair is the parsed form of one "Annotation=Value" entry from
// rule.Params["expected_values"]. The slice form is REQUIRED for
// deterministic iteration (PC01RNF-003); a Go map's iteration order would
// reorder violations between runs.
type expectedValuePair struct {
    Annotation string // simple name (left side, trimmed)
    Expected   string // verbatim value text (right side, trimmed)
}

// parseExpectedValues splits a comma-joined list of "Ann=Value" pairs into
// an ordered slice. Splits on "," then on FIRST "=". Pieces without "=" are
// skipped. Whitespace around both sides is trimmed. Duplicate annotations
// preserve the LAST occurrence's value (deterministic on a given input).
// Returns nil for an empty input. See §8 Q4 / Q7 for limitations.
func parseExpectedValues(s string) []expectedValuePair
```

### 2.3 Parser extension — `extractAnnotations` returns argument text

```go
// extractAnnotations collects annotation names AND their first positional
// argument text from a `modifiers` node. Returns three parallel slices of
// equal length:
//   - simple    — simple annotation names (no leading @)
//   - qualified — fully-qualified or simple, matching source spelling
//   - args      — first positional argument text VERBATIM (incl. quotes
//                 for string literals and ".class" for class literals);
//                 "" when the annotation has no argument list (i.e. is a
//                 `marker_annotation`) or its argument list is empty.
//
// Tree-sitter Java grammar background:
//   - `marker_annotation`  → @Override                (no argument_list child)
//   - `annotation`         → @ExtendWith(X.class)      (has argument_list; first
//                                                       positional child's text
//                                                       captured)
//   - For element-value pairs (e.g. @Foo(name="bar")) the FIRST element-value
//     pair's VALUE text is captured. Subsequent pairs are ignored — out of
//     PC01US-006 scope.
//
// Invariants:
//   - len(simple) == len(qualified) == len(args)
//   - Malformed annotations are silently skipped (preserve invariant).
//   - Order matches source order.
func extractAnnotations(node *sitter.Node, src []byte) (simple, qualified, args []string)
```

`extractClassDeclaration` is updated to:

```go
case nodeModifiers:
    var args []string
    decl.Annotations, decl.QualifiedAnnotations, args = extractAnnotations(child, src)
    if len(args) == len(decl.Annotations) && len(args) > 0 {
        m := make(map[string]string, len(args))
        for i, name := range decl.Annotations {
            m[name] = args[i]
        }
        decl.AnnotationArgs = m
    }
```

The same callers `extractInterfaceDeclaration`, `extractEnumDeclaration`, `extractRecordDeclaration` are updated identically. `extractMethods` is updated to consume only the first two return slices (method-arg capture is out of scope for PC01US-006). The third return slice may be discarded with `_`.

### 2.4 Profile schema — new param key `expected_values`

```yaml
audit_rules:
  - id: unit-test-class-contract
    kind: required_annotations
    severity: ERROR
    description: 'Unit-test class {name} violates the test-class contract: {evidence}'
    suggestion: 'Annotate {name} with all of [{required}], using the expected argument values.'
    params:
      path_scope: src/main/java/
      annotations: 'ExtendWith,DisplayName'
      expected_values: 'ExtendWith=MockitoExtension.class'
      node_types: class_declaration
```

The `expected_values` key is OPTIONAL — pre-existing rules with no `expected_values` continue to behave exactly as today. **No `fsprofile/mapper.go` change is required**: `required_annotations` is already in `knownAuditRuleKinds`, and the loader passes `params` through as a `map[string]string`. Verified by reading the mapper at `/workspaces/jitctx/internal/infrastructure/fsprofile/mapper.go:11-20`.

### 2.5 No new use-case interface, no new port, no new error sentinel

This story extends an existing evaluator on an existing kind. No `internal/domain/port/**` or `internal/domain/usecase/**` change. No `Deps` struct change in `internal/cli/wire.go`. No new error sentinel — defensive paths return `nil` (consistent with sibling evaluators).

---

## Section 3 — Domain Layer Plan (Tier 1, single group T1-G1)

### 3.1 `internal/domain/model/javaFileSummary.go`

Add the `AnnotationArgs map[string]string` field to `JavaDeclaration` exactly as in §2.1. Field is the LAST field of the struct (preserves git diffs over `Methods`/`Fields`). The field is unexported-vs-exported question: `AnnotationArgs` is exported because `internal/infrastructure/treesitter/parser.go` writes to it from a different package.

### 3.2 `internal/domain/service/auditRuleEvaluator.go`

1. Add the `expectedValuePair` type and `parseExpectedValues` helper at the bottom of the file (after `pathExempt`, before `evalMethodNaming`). Keep helpers private; this keeps the public API surface unchanged.
2. Modify `evalRequiredAnnotations`:
   - After the existing `path_scope` / `annotations` validation, parse `rule.Params["expected_values"]` into `expected []expectedValuePair`.
   - The current substitution context uses `{required}` and `{missing}`. Migrate the `missing` template token to `{evidence}` so the same description string can render BOTH evidence flavors.
   - Before constructing the missing-violation, build:
     ```go
     missingEvidence := "missing=[" + strings.Join(missing, ",") + "]"
     ctx := map[string]string{
         "file":     summary.Path,
         "name":     decl.Name,
         "required": strings.Join(required, ","),
         "evidence": missingEvidence,
     }
     ```
   - Append the missing-violation to `violations` if `len(missing) > 0`.
   - Then iterate `expected` (ordered slice):
     ```go
     for _, pair := range expected {
         if !slices.Contains(decl.Annotations, pair.Annotation) {
             continue // already covered by missing[]
         }
         actual := decl.AnnotationArgs[pair.Annotation]
         if actual == pair.Expected {
             continue
         }
         mismatchEvidence := "annotation=" + pair.Annotation +
             ", expected_value=" + pair.Expected +
             ", actual=" + actual
         ctxMm := map[string]string{
             "file":     summary.Path,
             "name":     decl.Name,
             "required": strings.Join(required, ","),
             "evidence": mismatchEvidence,
         }
         violations = append(violations, makeViolation(moduleID, summary, rule, 0, ctxMm))
     }
     ```
3. **Backward compatibility for `{missing}` token:** the existing `requiredAnnotationsRule` test fixture uses both `{missing}` and `{required}` in its Description. To avoid breaking PC01US-001's existing test, keep populating `{missing}` in the missing-violation `ctx` (set it to the same `missingEvidence` literal `missing=[A,B,...]`). The arg-mismatch context does NOT populate `{missing}` (that token is only meaningful for the presence violation). This mirrors how `evalForbiddenAnnotations` populates `{forbidden}` and `{found}` with parallel keys.
4. The `missing=` literal stays inside the evaluator (NOT in the YAML) — this is the same pattern used by `evalForbiddenAnnotations` for `found=[...]`. PC01RNF-001 is preserved: no `Java`, `Spring`, `Mockito`, `Test`, `DisplayName`, `ExtendWith` identifier appears in the evaluator code.

### 3.3 No domain-port, VO, or use-case changes

- `auditvo.AuditViolation` is unchanged.
- `model.AuditRule` and `model.AuditRuleKind` are unchanged (no new constant).
- No new sentinel error.

---

## Section 4 — Infrastructure Layer Plan (Tier 2, single group T2-G1)

### 4.1 `internal/infrastructure/treesitter/parser.go`

1. Extend `extractAnnotations` to return a third slice of first-positional-argument texts (signature in §2.3). Implementation:
   - Walk the children of the modifiers node as today.
   - For each `nodeMarkerAnnotation` (no args possible) → append `""` to `args`.
   - For each `nodeAnnotation` / `nodeNormalAnnotation`:
     - Find the `argument_list` child.
     - Find the first child of `argument_list` whose type is one of `expression`, `field_access`, `class_literal`, `string_literal`, `identifier`, `scoped_identifier`, `element_value_pair` (or any non-punctuation node — punctuation children are `(`, `)`, `,`).
     - If the first non-punctuation child is `element_value_pair`, descend into it and capture the VALUE child's verbatim text (the `argument_list` form `@Foo(name="bar")` is rare; AC scenarios use positional args, but we keep the fallback for robustness).
     - Otherwise capture the verbatim text of the first non-punctuation child.
   - Edge case: missing or empty `argument_list` → `""`.
2. Update all four `extractXxxDeclaration` functions to consume the third slice and populate `JavaDeclaration.AnnotationArgs` per §2.3 sketch.
3. Update `extractMethods` to discard the third slice — it does not populate `JavaMethod.AnnotationArgs` (no such field; method-arg capture is out of PC01US-006 scope).
4. Add new node-type constants if any are needed for the argument lookup — check `nodeRegistry.go` (or wherever `nodeAnnotation`, `nodeMarkerAnnotation`, `nodeNormalAnnotation` are declared) and add `nodeArgumentList`, `nodeElementValuePair` if they are not yet defined. (Verify by `grep nodeArgumentList internal/infrastructure/treesitter/`.)
5. Atomic-write rule N/A — parser is read-only.

### 4.2 No `fsprofile`/`fsmanifest`/`fsconfig`/`token` adapter changes

`required_annotations` is already whitelisted in `mapper.go:17`. The optional `expected_values` key is passed through `params map[string]string` like every other kind-specific knob; no schema change in `profileDTO`.

---

## Section 5 — Application Layer Plan

**N/A.** The Application layer (`internal/application/usecase/audituc/**`) calls `service.AuditEvaluator.EvaluateFile` with a list of `model.AuditRule`. Since this story does not introduce a new `AuditRuleKind` and does not change the `EvaluateFile` signature, no application-layer file is touched.

---

## Section 6 — Presentation Layer Plan

**N/A.** No new cobra command, no new flag, no new formatter. The existing `audit` command's text output already substitutes the rule's `Description` template via `makeViolation` → `substituteSuggestion`; the new `{evidence}` token plugs into that pipeline transparently.

---

## Section 7 — Composition Root + Tests Plan (Tier 6)

### 7.1 No wiring changes

`internal/cli/wire.go`, `internal/cli/root.go`, `internal/cli/execute.go`, `cmd/jitctx/main.go`, `internal/config/**` — UNTOUCHED. The `Deps` struct already exposes `service.AuditEvaluator`, which is mutated in place at the file level only.

### 7.2 Parser unit tests — `internal/infrastructure/treesitter/parser_test.go` (T6-G1)

Add tests that drive the new `args` return slice and `JavaDeclaration.AnnotationArgs` map:

- `TestParser_ClassWithExtendWithArg_PopulatesAnnotationArgs`
  - Source contains `@ExtendWith(MockitoExtension.class)`, `@DisplayName("User service tests")` on a class.
  - Asserts `decl.AnnotationArgs["ExtendWith"] == "MockitoExtension.class"`.
  - Asserts `decl.AnnotationArgs["DisplayName"] == "\"User service tests\""` (literal quotes preserved — backs Q2).
- `TestParser_ClassWithMarkerAnnotation_AnnotationArgEmpty`
  - Source contains `@Override` on a method-bearing class — class itself bears one marker annotation (e.g. `@Deprecated`).
  - Asserts the marker annotation maps to `""` in `decl.AnnotationArgs`.
- `TestParser_ClassWithoutAnnotations_AnnotationArgsNilOrEmpty`
  - Source has zero class-level annotations.
  - Asserts `decl.AnnotationArgs == nil` OR `len(decl.AnnotationArgs) == 0` (either is acceptable).
- Existing tests that touch `Annotations`/`QualifiedAnnotations` must continue to pass — the third return value is a non-breaking append.

### 7.3 Evaluator unit tests — `internal/domain/service/auditRuleEvaluator_test.go` (T6-G2)

Add a new section header `// unit-test-class-contract  (PC01US-006 / PC01RF-007 / PC01RF-009)` and a private fixture function `unitTestClassContractRule()` that returns:

```go
model.AuditRule{
    ID:          "unit-test-class-contract",
    Kind:        model.AuditKindRequiredAnnotations,
    Severity:    model.AuditSeverityError,
    Description: "{name}: {evidence}",
    Suggestion:  "Apply the contract to {name}",
    Params: map[string]string{
        "path_scope":      "src/main/java/",
        "annotations":     "ExtendWith,DisplayName",
        "expected_values": "ExtendWith=MockitoExtension.class",
        "node_types":      "class_declaration",
    },
}
```

Tests to add (each `t.Parallel()`, each maps to one acceptance scenario):

- `TestAuditEvaluator_RequiredAnnotations_UnitTestClassWithBothAnnotationsAndCorrectArgPasses` — backs **AC1**.
  - `JavaDeclaration` carries `Annotations: ["ExtendWith", "DisplayName"]` and `AnnotationArgs: {"ExtendWith":"MockitoExtension.class", "DisplayName":"\"User service tests\""}`. Asserts `require.Empty(got)`.
- `TestAuditEvaluator_RequiredAnnotations_UnitTestClassWrongExtensionArgFlagsViolation` — backs **AC2**.
  - `Annotations: ["ExtendWith", "DisplayName"]`, `AnnotationArgs: {"ExtendWith":"SpringExtension.class", "DisplayName":"\"User service tests\""}`.
  - Asserts `require.Len(got, 1)`.
  - Asserts `require.Contains(got[0].Message, "annotation=ExtendWith, expected_value=MockitoExtension.class, actual=SpringExtension.class")` (the literal AC2 substring).
- `TestAuditEvaluator_RequiredAnnotations_UnitTestClassMissingDisplayNameFlagsViolation` — backs **AC3**.
  - `Annotations: ["ExtendWith"]`, `AnnotationArgs: {"ExtendWith":"MockitoExtension.class"}`.
  - Asserts `require.Len(got, 1)` AND `require.Contains(got[0].Message, "missing=[DisplayName]")` (the literal AC3 substring).
- `TestAuditEvaluator_RequiredAnnotations_BothMissingAndMismatchEmitTwoOrderedViolations`
  - `Annotations: ["ExtendWith"]`, `AnnotationArgs: {"ExtendWith":"SpringExtension.class"}`.
  - Asserts `require.Len(got, 2)`.
  - Asserts `got[0].Message` contains `missing=[DisplayName]`.
  - Asserts `got[1].Message` contains `annotation=ExtendWith, expected_value=MockitoExtension.class, actual=SpringExtension.class`.
  - Backs PC01RNF-003 ordering invariant in §2.2.
- `TestAuditEvaluator_RequiredAnnotations_ExpectedValuesParsing_DuplicateKeyLastWins`
  - Calls `parseExpectedValues("Foo=A,Foo=B")` and asserts the resulting slice has the value `B` for `Foo` (LAST occurrence wins, per §2.2). This nails Q4 / determinism on bad input.
- `TestAuditEvaluator_RequiredAnnotations_ExpectedValuesIgnoresMalformedPiece`
  - Asserts `parseExpectedValues("Foo,Bar=B")` produces ONE pair (`Bar=B`).
- `TestAuditEvaluator_RequiredAnnotations_NoExpectedValues_BehavesLikeBefore`
  - Existing PC01US-001 fixture (the `requiredAnnotationsRule()` rule) MUST still pass. This test re-runs the existing fixture and asserts no behavior drift — defensive against accidental refactor of the {missing}→{evidence} migration.

### 7.4 Integration test — `internal/cli/command/unitTestClassContractIntegration_test.go` (T6-G3)

Mirror the structure of `testMethodNamingIntegration_test.go`:

- Local helper `newAuditCmdForUnitTestClassContract(t, workDir, manifestPath)` — builds the same wiring as `newAuditCmdForTestMethodNaming` (per Q3 of US-005, no DRY refactor across helpers).
- `TestAuditCmd_Integration_UnitTestClassContract_CleanFixture_NoViolation` — backs **AC1**.
  - Copies `testdata/pc01us006UnitTestClassContract/projectClean` into a `t.TempDir()`.
  - Asserts `require.NotContains(out, "[unit-test-class-contract]")`.
- `TestAuditCmd_Integration_UnitTestClassContract_WrongExtensionArg_FlagsViolation` — backs **AC2**.
  - Copies `projectWrongArg`.
  - Asserts `require.Contains(out, "[unit-test-class-contract]")`.
  - Asserts `require.Contains(out, "annotation=ExtendWith, expected_value=MockitoExtension.class, actual=SpringExtension.class")` (literal AC2 substring).
  - Asserts exactly ONE `[unit-test-class-contract]` line via `require.Equal(1, strings.Count(out, "[unit-test-class-contract]"))`.
- `TestAuditCmd_Integration_UnitTestClassContract_MissingDisplayName_FlagsViolation` — backs **AC3**.
  - Copies `projectMissingDisplayName`.
  - Asserts `require.Contains(out, "missing=[DisplayName]")` (literal AC3 substring).
- `TestAuditCmd_Integration_UnitTestClassContract_Determinism` — backs PC01RNF-003.
  - Runs the wrong-arg fixture twice in two separate temp dirs; normalises the temp-dir prefix to `<TMP>` and asserts byte-identical stdout.

### 7.5 Fixture trees

All fixtures live UNDER `src/main/java/...` because `treesitter.Walker.WalkJavaFiles` only scans `src/main/java/`. (Lesson from PC01US-005.) Each fixture project ships:

- `pom.xml` containing the literal `org.springframework.boot` so `fsprofile.Detector` activates the bundled profile.
- `project-state.yaml` with `schema_version: 2`, one module rooted at `src/main/java/com/acme`, an empty `methods` list per contract — minimal viable manifest for the audit use case.
- `.jitctx/profiles/spring-boot-hexagonal.yaml` containing only the rule under test (`unit-test-class-contract`) so the integration test does not get noise from `test-naming`, `no-field-injection`, etc.
- One `.java` file per fixture under `src/main/java/com/acme/UserServiceTest.java`.

#### Fixture content sketches (T6-G4 / T6-G5 / T6-G6)

**`projectClean/.../UserServiceTest.java`**:
```java
package com.acme;
import org.junit.jupiter.api.extension.ExtendWith;
import org.junit.jupiter.api.DisplayName;
import org.mockito.junit.jupiter.MockitoExtension;

@ExtendWith(MockitoExtension.class)
@DisplayName("User service tests")
public class UserServiceTest {
}
```

**`projectWrongArg/.../UserServiceTest.java`**:
```java
package com.acme;
import org.junit.jupiter.api.extension.ExtendWith;
import org.junit.jupiter.api.DisplayName;
import org.springframework.test.context.junit.jupiter.SpringExtension;

@ExtendWith(SpringExtension.class)
@DisplayName("User service tests")
public class UserServiceTest {
}
```

**`projectMissingDisplayName/.../UserServiceTest.java`**:
```java
package com.acme;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.junit.jupiter.MockitoExtension;

@ExtendWith(MockitoExtension.class)
public class UserServiceTest {
}
```

**Profile YAML (clean / wrongArg / missingDisplayName — same content, only the audit rule):**
```yaml
name: spring-boot-hexagonal
languages: [java]
query_lang: java
detect:
  files:
    - name: pom.xml
      contains: "org.springframework.boot"
module_detection:
  strategy: hexagonal
  roots:
    - src/main/java/**
  markers:
    - kind: path_contains
      value: /port/in/
rules: []
audit_rules:
  - id: unit-test-class-contract
    kind: required_annotations
    severity: ERROR
    description: 'Unit-test class {name} violates the contract: {evidence}'
    suggestion: 'Annotate {name} with all of [{required}] using the expected argument values.'
    params:
      path_scope: src/main/java/
      annotations: 'ExtendWith,DisplayName'
      expected_values: 'ExtendWith=MockitoExtension.class'
      node_types: class_declaration
```

`testdata/` is gitignored by repo convention; fixture authors must `git add -f` all four files per fixture tree. (Same posture as US-004 / US-005 — see `testdata/pc01us005TestMethodNaming/`.)

### 7.6 Bundled `spring-boot-hexagonal` profile

**Decision (Q8):** the bundled profile (`internal/infrastructure/fsprofile/bundled/spring-boot-hexagonal.yaml` or wherever it ships) is **NOT** modified in this PR. The rule lives only in fixtures, mirroring the posture taken in PC01US-004 and PC01US-005. A follow-up story can graft the rule into the bundled profile once it has soaked.

---

## Section 8 — Open Questions & Risks

All resolved. Nothing blocks discovery.

### Q1 — Extend `evalRequiredAnnotations` vs. add a new `AuditRuleKind`?
**Resolved: extend.** Blocking: No.
PC01RF-001 already binds `required_annotations` to "multi-annotation required-set with all-of semantics". Adding a parallel kind would force profile authors to duplicate `path_scope`, `annotations`, and `node_types`. The optional `expected_values` parameter is the path of least surprise.

### Q2 — Argument representation in `JavaDeclaration.AnnotationArgs`.
**Resolved: VERBATIM source text, single first-positional, simple-name keyed.** Blocking: No.
- AC2 expects `actual=SpringExtension.class` — the `.class` suffix is part of the captured text.
- The same field captures `"User service tests"` (with the literal `"`) for `@DisplayName` — even though no AC asserts on the latter literal, this keeps the contract uniform: the evaluator does not strip quotes.
- Multi-arg annotations are out of scope; PC01US-006 does not stress them. Q9 below documents the ambiguity for repeated annotations on one decl.

### Q3 — Violation count when both presence-missing AND arg-mismatch trigger.
**Resolved: ONE missing-violation + ONE per arg-mismatch (no collapsing).** Blocking: No.
This is a deliberate evidence-richness choice (PC01RF-009): collapsing would lose the distinction between "you forgot the annotation entirely" and "the annotation is there but uses the wrong value". Test `BothMissingAndMismatchEmitTwoOrderedViolations` (§7.3) locks this behavior.

### Q4 — Deterministic iteration order over `expected_values`.
**Resolved: pre-parse to ordered slice (`[]expectedValuePair`).** Blocking: No.
Go's `map` iteration order is randomised; iterating a parsed `map[string]string` would re-order the emitted mismatch violations between runs and violate PC01RNF-003. The `parseExpectedValues` helper preserves the order pieces appear in the source string.

### Q5 — How to handle marker annotations when `expected_values` mentions them.
**Resolved: treat empty argument as `actual=""`; emit mismatch unless `expected` is also `""`.** Blocking: No.
A marker annotation present on the declaration but configured with a non-empty expected value means the profile author wrote `Foo=Bar` and the source has `@Foo` (no args). Producing `annotation=Foo, expected_value=Bar, actual=` is correct and PC01RF-009-compliant — the trailing empty `actual=` is the evidence.

### Q6 — Walker scope.
**Resolved: fixtures live under `src/main/java/...`.** Blocking: No.
`treesitter.Walker.WalkJavaFiles` only scans `src/main/java/`. Even though the AC scenarios discuss "unit-test classes", the walker pitfall demands fixtures under `src/main/java/`. The path-scope param `src/main/java/` accepts those paths. This is consistent with how PC01US-005 handled its test-method-naming rule on `src/main/java/com/acme/UserServiceTest.java`.

### Q7 — Argument values containing commas.
**Resolved: not supported; document limitation.** Blocking: No.
`expected_values` parses on `,` then on first `=`. AC values (`MockitoExtension.class`) have no commas. A future-extension key (e.g. `expected_values_json` or per-annotation child YAML keys) can lift this; out of scope for PC01US-006. Documented in the doc-comment in §2.2.

### Q8 — Should the bundled `spring-boot-hexagonal` profile gain this rule?
**Resolved: NO.** Blocking: No.
Same posture as US-004 / US-005. Keeps the bundled profile stable; the rule is exercised only via fixtures. A future story can graft the rule onto the bundled profile once authoring has soaked.

### Q9 — Repeated occurrences of the same simple annotation on one declaration.
**Resolved: LAST occurrence wins for `AnnotationArgs[<name>]`.** Blocking: No.
Java does not generally allow repeating the same annotation type without a `@Repeatable` declaration; in practice this is not seen. The "last wins" behavior is deterministic and matches Go's `m[k] = v` semantics during parser population. No AC scenario stresses this.

### Risks (informational, non-blocking)

- **R1**: A future story that introduces multi-positional argument matching (e.g. `@RequestMapping(value="/", method=POST)`) will need a richer `AnnotationArgs` shape. Migration is a domain-only change — the YAML key name may change but the existing single-positional code path remains.
- **R2**: Tree-sitter Java grammar updates between releases could change child-node ordering inside `argument_list`. The new tests in §7.2 protect against silent regressions.
- **R3**: Verbatim quote preservation for string literal arguments may surprise future profile authors (e.g. they write `expected_values: 'DisplayName=User service tests'` and the rule does not trigger because actual is `"User service tests"`). The doc-comment in §2.2 calls this out explicitly.

---

## Section 9 — Parallel Execution Plan (authoritative for @agent-manager)

```yaml
tiers:
  - id: 1
    name: Domain contract
    depends_on: []
    groups:
      - id: T1-G1
        scope:
          create: []
          modify:
            - internal/domain/model/javaFileSummary.go
            - internal/domain/service/auditRuleEvaluator.go
        guidelines:
          - .claude/guidelines/domain-layer-guidelines.yml
        effort: M
        notes: >
          Adds JavaDeclaration.AnnotationArgs and extends evalRequiredAnnotations
          with the optional expected_values param plus the parseExpectedValues
          helper. Migrates the missing/mismatch evidence to a single {evidence}
          substitution token. Backwards compatible: existing PC01US-001 rule
          continues to substitute {missing} via the same context map, and no
          new AuditRuleKind is added.

  - id: 2
    name: Infrastructure adapters (parallel)
    depends_on: [1]
    groups:
      - id: T2-G1
        scope:
          create: []
          modify:
            - internal/infrastructure/treesitter/parser.go
        guidelines:
          - .claude/guidelines/infrastructure-layer-guidelines.yml
        effort: M
        notes: >
          Extends extractAnnotations to return a third slice of first-positional
          argument texts (verbatim source text, including quotes for string
          literals and ".class" for class literals). Updates the four
          extractXxxDeclaration callers to populate JavaDeclaration.AnnotationArgs.
          extractMethods discards the third slice (method-arg capture is out of
          scope for PC01US-006). No fsprofile/mapper.go change required because
          required_annotations is already whitelisted; the optional
          expected_values key passes through params map[string]string unchanged.

  - id: 6
    name: Tests + fixtures (parallel)
    depends_on: [1, 2]
    groups:
      - id: T6-G1
        scope:
          create: []
          modify:
            - internal/infrastructure/treesitter/parser_test.go
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: S
        notes: >
          Adds parser tests proving AnnotationArgs is populated for class-level
          @ExtendWith(MockitoExtension.class), @DisplayName("User service tests"),
          and marker annotations (empty string).

      - id: T6-G2
        scope:
          create: []
          modify:
            - internal/domain/service/auditRuleEvaluator_test.go
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: M
        notes: >
          Adds unitTestClassContractRule() fixture and unit tests for AC1, AC2,
          AC3, the both-missing-and-mismatch ordering invariant, parseExpectedValues
          duplicate-key/malformed handling, and a backward-compatibility guard
          that re-runs the existing PC01US-001 requiredAnnotationsRule fixture.

      - id: T6-G3
        scope:
          create:
            - internal/cli/command/unitTestClassContractIntegration_test.go
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: M
        notes: >
          Real cobra audit command wired to real treesitter / fsmanifest /
          fsprofile / fsconfig adapters. Three fixture-driven scenarios plus
          a determinism check. Local helper newAuditCmdForUnitTestClassContract
          mirrors testMethodNamingIntegration_test.go's posture (no DRY refactor
          across helpers).

      - id: T6-G4
        scope:
          create:
            - testdata/pc01us006UnitTestClassContract/projectClean/pom.xml
            - testdata/pc01us006UnitTestClassContract/projectClean/project-state.yaml
            - testdata/pc01us006UnitTestClassContract/projectClean/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us006UnitTestClassContract/projectClean/src/main/java/com/acme/UserServiceTest.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Clean fixture (AC1): both annotations present with the correct argument
          value. Force-add (testdata/ is gitignored). pom.xml carries
          "org.springframework.boot". Java class lives under src/main/java/ to
          satisfy the Walker scope.

      - id: T6-G5
        scope:
          create:
            - testdata/pc01us006UnitTestClassContract/projectWrongArg/pom.xml
            - testdata/pc01us006UnitTestClassContract/projectWrongArg/project-state.yaml
            - testdata/pc01us006UnitTestClassContract/projectWrongArg/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us006UnitTestClassContract/projectWrongArg/src/main/java/com/acme/UserServiceTest.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Wrong-argument fixture (AC2): @ExtendWith(SpringExtension.class) instead
          of MockitoExtension.class; @DisplayName present with a string literal.
          Profile rule expected_values requires "ExtendWith=MockitoExtension.class".

      - id: T6-G6
        scope:
          create:
            - testdata/pc01us006UnitTestClassContract/projectMissingDisplayName/pom.xml
            - testdata/pc01us006UnitTestClassContract/projectMissingDisplayName/project-state.yaml
            - testdata/pc01us006UnitTestClassContract/projectMissingDisplayName/.jitctx/profiles/spring-boot-hexagonal.yaml
            - testdata/pc01us006UnitTestClassContract/projectMissingDisplayName/src/main/java/com/acme/UserServiceTest.java
          modify: []
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Missing-DisplayName fixture (AC3): only @ExtendWith(MockitoExtension.class)
          on the class; the rule's missing-set evidence "missing=[DisplayName]"
          is the literal substring asserted by AC3.
```

---

## Self-Validation

- [x] Every file in §1 appears in exactly one group across §9 (18 unique paths, no duplicates).
- [x] Every requirement ID (PC01US-006, PC01RF-001, PC01RF-007, PC01RNF-001, PC01RNF-003, PC01RNF-006, PC01RF-009) appears in at least one §1 row.
- [x] No file path appears in two groups.
- [x] Every port / type / function referenced in §2 either already exists in the codebase (read-verified) or is scheduled in T1-G1 / T2-G1.
- [x] No new use-case interface, no new `Deps` field, no new error sentinel — explicit `N/A` recorded for §3.3 / §4.2 / §5 / §6.
- [x] DAG: T1 → T2 → T6 — acyclic; `depends_on` lists conform.
- [x] Tier 1 exists because `internal/domain/**` files are touched. Tier 5 omitted because no wiring file is touched (verified).
- [x] All `guidelines[]` paths exist under `/workspaces/jitctx/.claude/guidelines/` (verified by `ls`).
- [x] Open Questions section has zero `Blocking: Yes` entries.
- [x] Walker-scope lesson incorporated (§7.5, Q6).
- [x] Determinism for `expected_values` iteration locked via the `expectedValuePair` ordered slice (§2.2, Q4).
- [x] All three AC literal substrings asserted via `require.Contains` in §7.3 and §7.4.
