---
name: PC01 (proposal-changes-01) progress
description: Status of user-story plans under docs/propose-changes-01/quality-gate-evaluators.feature; each US lands as a new audit-rule kind or evaluator extension.
type: project
---

PC01 = "proposal-changes-01" — declarative quality-gate evaluators driven by the framework profile YAML. Each user story adds a new `AuditRuleKind` (or extends one) plus its evaluator function in `internal/domain/service/auditRuleEvaluator.go`, and whitelists it in `internal/infrastructure/fsprofile/mapper.go`.

**Why:** the engine is intentionally language-neutral — `Entity`/`Spring`/`JUnit` literals must NEVER appear in `internal/domain` or `internal/application`; they live only in profile YAML and testdata fixtures. PC01RNF-001 enforces this; the discovery skill greps for these tokens before declaring a Tier 1 group done.

**How to apply:** when the user asks for a new PC01US-NNN plan, follow the established shape — one new `AuditKindXxx` constant, one `evalXxx` function, mapper whitelist, plus testdata fixtures named `pc01usNNN<FeatureCamel>/projectXxx`. Tiers usually collapse to {1, 2, 6} (no application/presentation/wiring change). Reuse helpers: `splitNonEmpty`, `nodeTypeAllowed`, `pathExempt`, `matchPathGlob`, `makeViolation`, `substituteSuggestion`.

User-story plan status (as of 2026-04-29):
- PC01US-002 (usecase-impl-stereotype) — plan landed.
- PC01US-003 (jpa-entity-contract) — plan landed.
- PC01US-004 (forbid-autowired-field-injection) — plan landed; introduced `target=field`, `exempt_paths`, `matchPathGlob`.
- PC01US-005 (test-method-naming) — plan landed; introduced `JavaMethod.{Name,Annotations,Line}`, `triggered_by`, `name_pattern`.
- PC01US-006 (unit-test-class-contract) — plan landed; introduced `JavaDeclaration.AnnotationArgs`, `expected_values` param on `required_annotations`.
- PC01US-007 (domain-no-entity-collection) — plan landed 2026-04-29 at `.claude/plans/pc01-us-007-domain-no-entity-collection/plan.md`. Introduces `AuditKindForbiddenFieldTypePattern`, `evalForbiddenFieldTypePattern`, helpers `matchTypePattern` (single inner type-parameter, single-`*` glob) and `resolveFQN` (imports-only, no `java.lang.` synthesis). Tiers active: 1, 2, 6.
- PC01US-008 (usecase-supertype, parameterized `UseCase<I, O>`) — not yet planned.

Reserved param keys after PC01US-007: `path_scope`, `annotations`, `expected_values`, `node_types`, `target`, `exempt_paths`, `triggered_by`, `name_pattern`, `path_required`, `path_required_any`, `name_suffix`, `name_regex`, `forbidden_type_suffix`, `forbidden_type_substring`, **`forbidden_type_patterns`**, `import_prefix`, `implements_glob`, `annotation`.
