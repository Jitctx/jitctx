---
name: PC01 (proposal-changes-01) progress
description: Status of user-story plans under docs/propose-changes-01/quality-gate-evaluators.feature; each US lands as a new audit-rule kind or evaluator extension.
type: project
---

PC01 = "proposal-changes-01" — declarative quality-gate evaluators driven by the framework profile YAML. Each user story adds a new `AuditRuleKind` (or extends one) plus its evaluator function in `internal/domain/service/auditRuleEvaluator.go`, and whitelists it in `internal/infrastructure/fsprofile/mapper.go`.

**Why:** the engine is intentionally language-neutral — `Entity`/`Spring`/`JUnit` literals must NEVER appear in `internal/domain` or `internal/application`; they live only in profile YAML and testdata fixtures. PC01RNF-001 enforces this; the discovery skill greps for these tokens before declaring a Tier 1 group done.

**How to apply:** when the user asks for a new PC01US-NNN plan, follow the established shape — one new `AuditKindXxx` constant, one `evalXxx` function, mapper whitelist, plus testdata fixtures named `pc01usNNN<FeatureCamel>/projectXxx`. Tiers usually collapse to {1, 2, 6} (no application/presentation/wiring change). Reuse helpers: `splitNonEmpty`, `nodeTypeAllowed`, `pathExempt`, `matchPathGlob`, `makeViolation`, `substituteSuggestion`.

User-story plan status (as of 2026-04-30):
- PC01US-002 (usecase-impl-stereotype) — plan landed.
- PC01US-003 (jpa-entity-contract) — plan landed.
- PC01US-004 (forbid-autowired-field-injection) — plan landed; introduced `target=field`, `exempt_paths`, `matchPathGlob`.
- PC01US-005 (test-method-naming) — plan landed; introduced `JavaMethod.{Name,Annotations,Line}`, `triggered_by`, `name_pattern`.
- PC01US-006 (unit-test-class-contract) — plan landed; introduced `JavaDeclaration.AnnotationArgs`, `expected_values` param on `required_annotations`.
- PC01US-007 (domain-no-entity-collection) — plan landed 2026-04-29; introduces `AuditKindForbiddenFieldTypePattern`, helpers `matchTypePattern` and `resolveFQN`.
- PC01US-008 (usecase-supertype, parameterized `UseCase<I, O>`) — plan landed; introduces `AuditKindRequiredParameterizedSupertype`, `JavaDeclaration.ParameterizedSupertypes`, helpers `parseSupertypePattern`/`splitGenericArgs`. Implementation already in main as of commit 8746a60.
- PC01US-009 (integration-test-base required annotations: `@SpringBootTest` + `@Testcontainers` + `@ActiveProfiles("test")`) — plan landed 2026-04-30 at `.claude/plans/pc01-us-009-it-base-required-annotations/plan.md` as a **tests-only ratification**. The existing `required_annotations` kind with `expected_values` param (PC01US-006) already covers AC1/AC2/AC3 — Tiers 1-5 are N/A; only Tier 6 ships (1 integration test + 3 fixture trees). Q3 documents that the parser captures string-literal args verbatim WITH quotes, so `expected_value="test", actual="prod"` is the literal output (not the AC's `expected_value=test, actual=prod` without quotes); the integration test asserts on substrings.
- PC01US-010 (transactional decorator: @Primary + @Qualifier(*)) — plan landed 2026-04-30 at `.claude/plans/pc01-us-010-tx-decorator-contract/plan.md` and implemented same day. Existing `required_annotations` kind covers AC1/AC2 (all-of semantics); AC3 ("non-empty Qualifier value") needed a minimal Tier 1 extension — additive optional sibling param `non_empty_value_annotations` on `evalRequiredAnnotations` plus private helper `isEmptyAnnotationArg` (treats `""`, `"\"\""`, `"''"` as empty). Determinism order: missing → expected_values → non-empty. Tiers active: 1, 6 (Tiers 2-5 N/A — no model/mapper/UC/CLI changes). Engine-neutrality (PC01RNF-001) preserved — `Primary`/`Qualifier`/`TxDecorator` literals confined to fixtures, integration tests, and the unit-test rule helper.

Reserved param keys after PC01US-010: `path_scope`, `annotations`, `expected_values`, `node_types`, `target`, `exempt_paths`, `triggered_by`, `name_pattern`, `forbidden_type_patterns`, `expected_supertype`, `args`, `supertype_kind`, `path_required`, `path_required_any`, `name_suffix`, `name_regex`, `forbidden_type_suffix`, `forbidden_type_substring`, `import_prefix`, `implements_glob`, `annotation`, **`non_empty_value_annotations`**.
