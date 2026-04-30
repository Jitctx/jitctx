# Code Review — pc01-us-007-domain-no-entity-collection

**Scope**: changes for PC01US-007 (Forbid collections of entities in domain models).
**Requirements**: docs/propose-changes-01/quality-gate-evaluators.feature
**Plan**: .claude/plans/pc01-us-007-domain-no-entity-collection/plan.md

## Architectural Conformity

- New `AuditKindForbiddenFieldTypePattern` constant lives in
  `internal/domain/model/auditRule.go` — pure enum, no framework deps.
  Compliant with domain-layer guideline.
- New evaluator `evalForbiddenFieldTypePattern` plus helpers
  `matchTypePattern` and `resolveFQN` stay in
  `internal/domain/service/auditRuleEvaluator.go` — domain service, no I/O.
  Compliant.
- `internal/infrastructure/fsprofile/mapper.go` only adds the new kind to the
  whitelist; no business logic leaks into infrastructure. Compliant.
- New dispatch arm in `EvaluateFile` routes `AuditKindForbiddenFieldTypePattern`
  to its evaluator — symmetric with the eight existing arms. Compliant with
  the open-closed pattern used by AuditEvaluator.
- No new ports introduced; no `internal/infrastructure` imports inside
  `internal/application` or `internal/domain`. Verified.
- Integration test lives under `internal/cli/command/` (presentation layer
  black-box) and wires real adapters via the same pattern as PC01US-004/005/006.
  Compliant.

## Go Idioms & Naming

- Filenames: `domainNoEntityCollectionIntegration_test.go` follows the
  camelCase project convention. Compliant.
- `evalForbiddenFieldTypePattern`, `matchTypePattern`, `resolveFQN` are all
  lowercase package-private — appropriate; only the public dispatch arm via
  `EvaluateFile` exercises them.
- Named returns `(outer, inner string, matched bool)` on `matchTypePattern`
  — used because the function returns parsed components even on no-match for
  potential debugging; documented in the doc-comment. Acceptable.
- `strings.Index` / `strings.LastIndex` / `strings.HasPrefix` / `HasSuffix`
  — stdlib idioms; no regex needed.
- No panics; defensive `return strings.TrimSpace(fieldType), "", false` on
  malformed brackets matches the surrounding evaluator style.
- The break-on-first-match (line 871) is correctly placed inside the inner
  pattern loop and outside the field loop — one violation per field.

## Code-Smell Metrics

- `evalForbiddenFieldTypePattern` is ~45 lines, single responsibility, one
  level of nesting (decl → field → pattern). Acceptable.
- `matchTypePattern` ~30 lines, six early-return branches; the switch on
  `innerPat` glob shape is clear and exhaustive (`*`, `*X`, `X*`, exact).
  Acceptable.
- `resolveFQN` is 7 lines; trivially correct.
- No duplication: pattern parsing reuses `splitNonEmpty` from PC01US-006;
  `pathExempt` / `nodeTypeAllowed` are shared with the other evaluators.

## Test Consistency vs Requirements

Acceptance Criteria from the .feature file are honoured:

- **AC1 (non-entity collection passes)**: covered by
  `TestAuditEvaluator_ForbiddenFieldTypePattern_NonEntityCollection_NoViolation`
  and `TestAuditCmd_Integration_DomainNoEntityCollection_NonEntityCollection_NoViolation`
  (List<String> on `Tag.java`).
- **AC2 (List<OrderEntity> flagged with FQN+pattern evidence)**: literal
  evidence `type=java.util.List<OrderEntity>, matched_pattern=List<*Entity>`
  asserted both in unit (`auditRuleEvaluator_test.go:1514`) and integration
  (`domainNoEntityCollectionIntegration_test.go:112`). Spaces after each comma
  match the substitution-template literal in the profile YAML
  (`description: '...type={type}, matched_pattern={matched_pattern}'`) verbatim.
  Field line 9 asserted by the integration test (`Order.java:9`).
- **AC3 (Set<UserEntity> on field line)**: violation reported on `User.java:11`
  via `TestAuditCmd_Integration_DomainNoEntityCollection_SetOfEntity_ReportsFieldLine`
  and the unit test on Line=11. The fixture has the `private` declaration on
  line 11 and the unit test fixes `Line: 11` directly, exercising the same
  emit path the parser would produce.
- **PC01RNF-003 (deterministic output)**: 
  `TestAuditCmd_Integration_DomainNoEntityCollection_Determinism` runs the
  same fixture in two distinct temp dirs and asserts byte-identical output
  after temp-dir normalisation. The evaluator iterates patterns via
  `splitNonEmpty` (preserves comma-source order), iterates declarations and
  fields in source order, and breaks on the first matching pattern per field.
  No map iteration on the emit path. Verified.
- **PC01RNF-001 (engine language-neutrality)**: grep across the new evaluator
  block (lines 809-935) shows zero NEW Java/Spring/JUnit/Lombok identifier
  literals. The 4 occurrences are: (1) `model.JavaFileSummary` parameter type
  — pre-existing model name; (2-3) doc-comment examples explaining the
  pattern syntax (`List<OrderEntity>`, `List<*Entity>`); (4) doc-comment note
  about `java.lang.*` not being synthesised. None are engine identifiers; the
  evaluator's logic names only profile-config keys (`path_scope`,
  `forbidden_type_patterns`, `node_types`, `exempt_paths`). Compliant.
- **PC01RF-005 (parameterized type-argument matching)**: covered by all
  six unit tests; edge cases for non-parameterized type, out-of-scope, and
  missing-import fallback all locked.
- **PC01RF-009 (evidence-rich messages)**: `{type}` and `{matched_pattern}`
  substitution tokens populated for every violation; fallback to simple name
  asserted by `…ImportNotFound_FallsBackToSimpleName`.
- **Backward compatibility**: pre-existing PC01US-001/002/003/004/005/006
  evaluator tests continue to pass — verified via `go test ./... -count=1`
  (all packages green).

Edge cases also covered by the unit suite:
- Non-parameterized field type → silently skipped (no false positives).
- Out-of-scope path → silently skipped.
- Missing import → simple-name fallback (still flags, but no FQN prefix).

## BLOCKERs

None.

## WARNINGs

None material. Two minor observations (not blockers):

- W-001 (style): `matchTypePattern` always returns `outer`/`inner` even on
  no-match. Currently no caller uses these on the no-match path. This is
  documented in the doc-comment and useful for future debug instrumentation;
  retain as-is.
- W-002 (docs): The doc-comment block above `evalForbiddenFieldTypePattern`
  (lines 809-829) is dense — consistent with surrounding evaluators. No change
  required.

## INFOs

- I-001: Three fixture profiles AND the three pom.xml files are
  byte-identical (verified via `diff`). The orchestrator's late fix to
  `projectListEntity` and `projectSetEntity` (overwrite from `projectClean`)
  is sound.
- I-002: `resolveFQN` uses `LastIndex` to find the terminal segment of an
  import — first matching import wins (stable, because import order is
  source-order). Profile authors who care about which import resolves first
  must order their imports accordingly; documented in the doc-comment.
- I-003: The break statement at evaluator line 871 ensures one violation per
  field even when a field's type matches multiple patterns — first matched
  pattern wins. Asserted by the integration test's
  `Equal(t, 1, strings.Count(out, "[domain-no-entity-collection]"))`.
- I-004: `matchTypePattern` uses `firstLT` / `lastGT` rather than a balanced
  bracket walk; this correctly handles `Map<K, List<V>>`-style nested
  parameters by capturing the outermost bracket pair. Not exercised by the
  current AC but forward-compatible.

## Verdict

**CLEAN — no BLOCKERs, no WARNINGs that block merge.**
