package fsprofile

// Per-kind structural validation of audit-rule DTOs. PC01US-011.
//
// Both audit-rule loaders (auditLoader.go::LoadAuditRules for the legacy
// single-file shape and bundleMapper.go::toBundleDomain for the EP-04
// directory shape) call validateAuditRuleParams immediately after the
// kind whitelist + severity whitelist checks, before constructing the
// model.AuditRule. The helper is pure: no filesystem, no logger, no
// model dependency beyond the AuditRuleKind constants. It consumes the
// structural subset of either DTO via auditRuleSchema.
//
// Message catalogue (closed for PC01US-011):
//
//	M1: "rule '<id>': required_annotations must declare at least one annotation"
//	M2: "rule '<id>': target must be one of [class, field, method, supertype]"
//	M3: "rule '<id>': forbidden_annotations must declare at least one annotation"
//	M4: "rule '<id>': forbidden_field_type_pattern must declare at least one pattern"
//
// All four are emitted as plain fmt.Errorf values. The caller (each
// loader call site) wraps the error with the profile-context prefix and
// the domerr.ErrProfileInvalid sentinel via
// fmt.Errorf("...: %w: %w", err, domerr.ErrProfileInvalid).

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// auditRuleSchema is the structural subset of auditRuleDTO and
// bundleAuditRuleDTO that the validator inspects. Both loader call
// sites convert inline.
type auditRuleSchema struct {
	ID     string
	Kind   string
	Params map[string]string
}

// validAuditRuleTargets is the closed enum for params["target"]. The
// list and ORDER are pinned to match the verbatim substring asserted
// by PC01US-011 AC2 ("[class, field, method, supertype]" — comma-space
// joined, square-bracketed).
var validAuditRuleTargets = []string{"class", "field", "method", "supertype"}

// validateAuditRuleParams enforces per-kind structural validation on
// the given audit-rule descriptor. Returns nil on success.
//
// Per-kind checks (executed in this order — the FIRST failing check
// short-circuits and returns):
//
//  1. params["target"]: when non-empty, must be in
//     validAuditRuleTargets. Applies to ALL kinds (the param is a
//     cross-kind selector — present on required_annotations,
//     forbidden_annotations, method_naming).
//  2. kind == required_annotations: splitNonEmpty(params["annotations"])
//     must be non-empty.
//  3. kind == forbidden_annotations: splitNonEmpty(params["annotations"])
//     must be non-empty.
//  4. kind == forbidden_field_type_pattern:
//     splitNonEmpty(params["forbidden_type_patterns"]) must be
//     non-empty.
//
// Other kinds (interface_naming, forbidden_import,
// field_type_layer_violation, method_naming,
// required_parameterized_supertype, annotation_path_mismatch,
// implements_path_mismatch) currently have NO PC01US-011 schema
// constraints — the helper returns nil for them. Future stories may
// extend the catalogue.
func validateAuditRuleParams(s auditRuleSchema) error {
	if t := strings.TrimSpace(s.Params["target"]); t != "" {
		if !targetIsValid(t) {
			return fmt.Errorf("rule '%s': target must be one of [%s]",
				s.ID, strings.Join(validAuditRuleTargets, ", "))
		}
	}
	switch model.AuditRuleKind(s.Kind) {
	case model.AuditKindRequiredAnnotations:
		if len(splitNonEmpty(s.Params["annotations"])) == 0 {
			return fmt.Errorf(
				"rule '%s': required_annotations must declare at least one annotation",
				s.ID)
		}
	case model.AuditKindForbiddenAnnotations:
		if len(splitNonEmpty(s.Params["annotations"])) == 0 {
			return fmt.Errorf(
				"rule '%s': forbidden_annotations must declare at least one annotation",
				s.ID)
		}
	case model.AuditKindForbiddenFieldTypePattern:
		if len(splitNonEmpty(s.Params["forbidden_type_patterns"])) == 0 {
			return fmt.Errorf(
				"rule '%s': forbidden_field_type_pattern must declare at least one pattern",
				s.ID)
		}
	}
	return nil
}

// targetIsValid checks t against the closed enum; helper extracted
// to keep the message-construction site readable.
func targetIsValid(t string) bool {
	return slices.Contains(validAuditRuleTargets, t)
}

// splitNonEmpty splits s on commas, trims whitespace from each
// element, and drops empties. Mirrors the function of the same name
// in internal/domain/service/auditRuleEvaluator.go (kept private to
// each package per Go style — duplication is intentional, the two
// functions live in different layers and may diverge in future
// stories). PC01RNF-001 engine-neutrality is preserved: no
// framework-specific literal is introduced.
func splitNonEmpty(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
