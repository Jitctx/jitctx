// Package fsprofile (legacyHasAnnotation.go).
//
// PC01US-012 — backward compatibility for the legacy `has_annotation: X`
// audit-rule shortcut. Profiles authored before the modern
// kind/params shape was introduced may declare an audit rule as:
//
//	audit_rules:
//	  - id: legacy-rule
//	    has_annotation: Service
//	    severity: ERROR
//	    description: '...'
//	    suggestion: '...'
//	    params:
//	      path_scope: application/usecase/
//
// The translation helper folds this into the modern equivalent:
//
//   - id: legacy-rule
//     kind: required_annotations
//     severity: ERROR
//     description: '...'
//     suggestion: '...'
//     params:
//     path_scope: application/usecase/
//     annotations: Service
//
// Per PC01RF-012, the legacy key keeps working with no deprecation
// warning in this release. Per PC01RNF-001, the helper introduces
// no framework-specific identifier — only the neutral schema words
// `has_annotation` and `required_annotations`.
//
// Translation rules (FIRST match wins):
//  1. has_annotation absent  → no translation.
//  2. kind explicitly set    → modern kind wins; has_annotation is
//     ignored (no error, no warning).
//  3. kind absent + has_annotation present → translate to
//     kind=required_annotations,
//     params.annotations = has_annotation
//     (UNLESS params.annotations is already set — in which case
//     the existing params value wins).
//
// Both audit-rule loaders (auditLoader.go::LoadAuditRules and
// bundleMapper.go::toBundleDomain) call the helper FIRST in the
// per-rule pipeline, BEFORE the kind whitelist and BEFORE the
// PC01US-011 schema validator. Running translation first allows
// downstream gates to treat the rule uniformly with hand-authored
// modern rules.
package fsprofile

import "maps"

// translateLegacyHasAnnotation folds the legacy `has_annotation: X`
// audit-rule shortcut into the modern kind/params shape. Pure
// function — no filesystem, no logger, no model import, no error
// return.
//
// The returned effParams is ALWAYS a fresh allocation (never aliases
// params), so callers can safely mutate or pass it through to the
// validator and the model.AuditRule conversion without worrying
// about shared state.
func translateLegacyHasAnnotation(
	kind, hasAnnotation string, params map[string]string,
) (effKind string, effParams map[string]string, translated bool) {
	// Always return a fresh copy of params so callers don't share
	// state with the DTO map.
	effParams = make(map[string]string, len(params)+1)
	maps.Copy(effParams, params)

	// Rule 1 — no legacy field set: pass-through.
	if hasAnnotation == "" {
		return kind, effParams, false
	}

	// Rule 2 — modern kind wins when both are set.
	if kind != "" {
		return kind, effParams, false
	}

	// Rule 3 — translate. Existing params.annotations wins over
	// has_annotation when both supply the list (more-specific
	// intent). When params.annotations is empty/absent, fold the
	// legacy scalar value into it.
	if existing, hasParam := effParams["annotations"]; !hasParam || existing == "" {
		effParams["annotations"] = hasAnnotation
	}
	// Else: params.annotations is already non-empty — preserve it.
	// The legacy field becomes informational metadata.

	return "required_annotations", effParams, true
}
