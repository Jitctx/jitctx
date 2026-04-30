# Security Audit — pc01-us-007-domain-no-entity-collection

**Scope**: changes for PC01US-007 (Forbid collections of entities in domain models).

## Files Audited

Modified Go:
- internal/domain/model/auditRule.go
- internal/domain/service/auditRuleEvaluator.go
- internal/domain/service/auditRuleEvaluator_test.go
- internal/infrastructure/fsprofile/mapper.go

New Go:
- internal/cli/command/domainNoEntityCollectionIntegration_test.go

New testdata fixtures (Java + YAML + pom.xml under
`testdata/pc01us007DomainNoEntityCollection/{projectClean,projectListEntity,projectSetEntity}`).

## Pillar Findings

### 1. Dependency CVEs
No new third-party imports. The PR uses only `strings`, `slices` (stdlib) and
pre-existing internal packages. The pre-existing imports `path`, `regexp`
remain untouched. Vendored Tree-sitter, cobra, yaml.v3 unchanged. **No findings.**

### 2. Filesystem Safety
- `evalForbiddenFieldTypePattern`, `matchTypePattern`, `resolveFQN` operate on
  in-memory strings only — no filesystem I/O.
- The integration test uses `t.TempDir()` and the existing `copyFixture` helper;
  no path concatenation outside the temp root.
- `resolveFQN` walks `summary.Imports` — a slice of strings already produced by
  the parser; no shell-out, no os.Open, no path traversal opportunity.

**No findings.**

### 3. Hardcoded Secrets
None. Strings introduced (`"type="`, `", matched_pattern="`, `"<"`, `">"`,
`"*"`, `","`, `"="`) are formatting/parser tokens. Fixture YAML/Java carry only
test identifiers. Profile YAML key literals (`forbidden_type_patterns`,
`path_scope`, `node_types`, `exempt_paths`) are configuration keys, not
secrets.

### 4. Insecure Configuration
- New evaluator is defensively coded:
  - `pathScope == "" || len(patterns) == 0` short-circuits to `nil`.
  - Non-parameterized field types (`<`/`>` absent or out of order) return
    `matched=false` rather than panicking on negative indices.
  - Pattern without brackets returns `matched=false` (defensive).
  - `resolveFQN` falls back to the simple name when no matching import is
    found — no java.lang.* synthesis, no implicit network/disk lookup.
- No regex compilation in the new evaluator — pattern matching uses
  `strings.HasPrefix`/`HasSuffix` only, so a malicious profile cannot trigger
  ReDoS through this rule kind.
- Iteration order is deterministic: `splitNonEmpty(patterns)` preserves the
  comma-source order; the evaluator iterates that slice and breaks on the
  first match (PC01RNF-003).
- The whitelist in `fsprofile/mapper.go` correctly admits the new kind; an
  unknown kind in a profile YAML is still rejected at load time.

**No findings.**

## Verdict

**CLEAN — no security findings (no severity > INFORMATIONAL).**
