# Code Review Report — ep01us-006 (Tree-sitter Java Integration)

**Date:** 2026-04-17
**Reviewer:** @code-reviewer (inline by @qa-coordinator)
**Requirements:** /workspaces/jitctx/docs/ep01/requirements.md
**Scope:**
- internal/cli/command/scanIntegration_test.go
- internal/domain/model/javaFileSummary.go
- internal/infrastructure/treesitter/parser.go
- internal/infrastructure/treesitter/parser_test.go
- internal/infrastructure/treesitter/queries.go
- testdata/springBootMinimal/fixtures/UserWithTableAnnotation.java

---

## Summary

| Level    | Count |
|----------|-------|
| BLOCKER  | 0     |
| WARNING  | 3     |
| INFO     | 3     |

**Verdict:** PASS WITH WARNINGS. No BLOCKERs; all warnings are
non-blocking refactors.

**Build:** `go vet ./...` clean, `gofmt -l` clean, all tests pass
(`./internal/infrastructure/treesitter/...`, `./internal/cli/command/...`).

---

## Dimension 1 — Architectural Conformity

Implementation lives correctly under `internal/infrastructure/treesitter`.
The domain model (`internal/domain/model/javaFileSummary.go`) is a pure
struct with no YAML/JSON tags and no framework imports — matches
CLAUDE.md "Sem tags YAML/JSON no domínio". The parser satisfies the port
contract (`ParseJavaFile(ctx, fsys, path)` — context first, `fs.FS` for
injection, returns domain value object).

No architectural violations.

## Dimension 2 — Go Idioms & Naming

Filenames are camelCase as required (`javaFileSummary.go`, `parser.go`,
`queries.go`). Errors are wrapped with `%w` against sentinels in
`internal/domain/errors`. No panics. Context is always first parameter.

### W-001 (WARNING) — `int(node.ChildCount())` conversion repeated ~15 times

- **File:** internal/infrastructure/treesitter/parser.go (pervasive)
- **Issue:** The pattern `for i := 0; i < int(node.ChildCount()); i++`
  appears in 11 functions. The cast is necessary (`ChildCount` returns
  `uint32`), but the repetition obscures the intent and slightly inflates
  the function bodies. Consider a small helper
  `iterChildren(node, func(i int, c *sitter.Node))` or simply
  `eachChild(node) iter.Seq[*sitter.Node]` (Go 1.25 range-over-func).
- **Non-blocking:** The loop is clear and correct as-is; this is a
  readability nit.

## Dimension 3 — Code-smell Metrics

### W-002 (WARNING) — `buildMethodSignature` uses a fragile "first identifier is the method name" heuristic

- **File:** internal/infrastructure/treesitter/parser.go:307-331
- **Issue:** Line 319 comment says
  `// first identifier is the method name`. This is correct for the
  current Tree-sitter Java grammar, but the function relies on child-order
  rather than on the `name` field of `method_declaration`
  (Tree-sitter exposes named fields via `ChildByFieldName("name")`).
  Using the field name makes the code self-documenting and robust against
  grammar revisions that reorder children.
- **Suggested fix:** `name = nodeText(node.ChildByFieldName("name"), src)`
  (and similarly for `type` and `parameters`). Same applies in
  `extractClassDeclaration`, `extractInterfaceDeclaration`, etc., for
  `name`, `superclass`, `interfaces`, `body`.
- **Non-blocking:** Works today; refactor recommended when a grammar bump
  lands.

### W-003 (WARNING) — `extractSingleParam` has two mutually-exclusive identifier branches

- **File:** internal/infrastructure/treesitter/parser.go:347-378
- **Issue:** The function handles `variable_declarator_id` (lines 355-362)
  and a bare `identifier` (lines 363-366) as fallbacks for the parameter
  name. The bare-identifier branch guards against `paramType != ""`, which
  means order-of-children matters. In Tree-sitter Java, formal_parameter
  always exposes the name via a `variable_declarator` / `name` field — the
  bare-identifier branch is likely dead code. If it is intentional as a
  defensive fallback, add a comment. Otherwise remove it.
- **Non-blocking:** Not incorrect, but hard to reason about.

### INFO-001 — `extractTypeList` handles both `type_list` and `interface_type_list`

- **File:** internal/infrastructure/treesitter/parser.go:262-281
- **Note:** This dual handling looks like grammar-version bridging.
  Consider adding a comment explaining why both are needed (likely
  `super_interfaces` wraps `type_list` in recent grammar versions and
  `interface_type_list` in older ones).

### INFO-002 — `JavaDeclaration.Extends` is `[]string` with "length 0 or 1"

- **File:** internal/domain/model/javaFileSummary.go:19
- **Note:** Java allows at most one superclass on a class, but multiple
  `extends` on an interface. The field comment says "length 0 or 1" which
  is wrong for interfaces (`interface Foo extends A, B` produces 2+
  entries, handled by `extractInterfaceDeclaration → extractTypeList`).
  Consider clarifying the comment: "length 0 or 1 for classes; any length
  for interfaces."

## Dimension 4 — Test Consistency

Tests cover: happy path, partial parse, qualified annotations, multi-
annotation + `@Table(name=...)` arguments, implements, method signatures
with generics (return and parameter), static imports, wildcard imports,
and mixed-valid-and-broken source. Integration tests exercise the full
scan pipeline including the new multi-annotation fixture
(`TestScanCmd_Integration_MultiAnnotationClass`).

All assertions trace to Gherkin acceptance criteria where applicable
(e.g. "Profile: spring-boot-hexagonal" exact substring, manifest
traversal rejection).

### INFO-003 — Fixture uses snake_case package `com.app.user_management`

- **File:** testdata/springBootMinimal/fixtures/UserWithTableAnnotation.java:1
- **Note:** The Java convention is lowercase concatenated
  (`com.app.usermanagement`), but the rest of the test corpus appears to
  use `user_management`, so this is consistent with the broader fixture
  set — not a finding against this feature. Flagged only for awareness.

---

## Auto-fixable BLOCKERs

None.

---

## Test / Build Results

- `go vet ./...` → clean
- `gofmt -l <scope>` → clean
- `go test ./internal/infrastructure/treesitter/... ./internal/cli/command/...` → PASS
