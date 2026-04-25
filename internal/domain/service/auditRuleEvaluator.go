package service

import (
	"regexp"
	"strings"

	"github.com/jitctx/jitctx/internal/domain/model"
	auditvo "github.com/jitctx/jitctx/internal/domain/vo/audit"
)

// AuditEvaluator evaluates a single rule against a parsed module file.
// It is a pure function — no I/O, no goroutines. The application use case
// drives it.
type AuditEvaluator struct{}

// NewAuditEvaluator returns the singleton-like evaluator.
func NewAuditEvaluator() *AuditEvaluator { return &AuditEvaluator{} }

// EvaluateFile applies every rule against one parsed file's declarations
// and returns the violations produced. Caller decides the moduleID; the
// evaluator does not look at modules. Output is unsorted; the use case
// sorts the union before handing it to the renderer (RNF-003).
func (e *AuditEvaluator) EvaluateFile(
	moduleID string,
	summary model.JavaFileSummary,
	rules []model.AuditRule,
) []auditvo.AuditViolation {
	var violations []auditvo.AuditViolation
	for _, rule := range rules {
		var got []auditvo.AuditViolation
		switch rule.Kind {
		case model.AuditKindAnnotationPathMismatch:
			got = evalAnnotationPathMismatch(moduleID, summary, rule)
		case model.AuditKindImplementsPathMismatch:
			got = evalImplementsPathMismatch(moduleID, summary, rule)
		case model.AuditKindInterfaceNaming:
			got = evalInterfaceNaming(moduleID, summary, rule)
		case model.AuditKindForbiddenImport:
			got = evalForbiddenImport(moduleID, summary, rule)
		case model.AuditKindFieldTypeLayerViolation:
			got = evalFieldTypeLayerViolation(moduleID, summary, rule)
		default:
			// Unknown kinds are skipped — the loader is responsible for
			// rejecting unknown kinds; the evaluator must be defensive.
		}
		violations = append(violations, got...)
	}
	return violations
}

// Per-kind helpers (private). Signatures frozen here so test scaffolding
// in T6-G1 can be authored in parallel with the implementation.

// evalAnnotationPathMismatch — params:
//
//	"annotation": simple annotation name without @ (e.g. "Entity")
//	"path_required": substring that the file path MUST contain
func evalAnnotationPathMismatch(
	moduleID string, summary model.JavaFileSummary, rule model.AuditRule,
) []auditvo.AuditViolation {
	annotation := rule.Params["annotation"]
	pathRequired := rule.Params["path_required"]
	if annotation == "" || pathRequired == "" {
		return nil
	}

	if strings.Contains(summary.Path, pathRequired) {
		// File is already in the correct location; no violations.
		return nil
	}

	var violations []auditvo.AuditViolation
	for _, decl := range summary.Declarations {
		for _, ann := range decl.Annotations {
			if ann == annotation {
				ctx := map[string]string{
					"file":          summary.Path,
					"name":          decl.Name,
					"path_required": pathRequired,
				}
				violations = append(violations, makeViolation(moduleID, summary, rule, 0, ctx))
				break
			}
		}
	}
	return violations
}

// evalImplementsPathMismatch — params:
//
//	"implements_glob": e.g. "*UseCase"
//	"path_required_any": comma-joined list of substrings; ANY match is OK
func evalImplementsPathMismatch(
	moduleID string, summary model.JavaFileSummary, rule model.AuditRule,
) []auditvo.AuditViolation {
	implGlob := rule.Params["implements_glob"]
	pathRequiredAny := rule.Params["path_required_any"]
	if implGlob == "" || pathRequiredAny == "" {
		return nil
	}

	substrings := strings.Split(pathRequiredAny, ",")

	// Check if ANY required substring appears in the file path.
	pathOK := false
	for _, sub := range substrings {
		if strings.Contains(summary.Path, strings.TrimSpace(sub)) {
			pathOK = true
			break
		}
	}
	if pathOK {
		return nil
	}

	var violations []auditvo.AuditViolation
	for _, decl := range summary.Declarations {
		for _, iface := range decl.Implements {
			if matchGlob(implGlob, iface) {
				ctx := map[string]string{
					"file":              summary.Path,
					"name":              decl.Name,
					"implements_glob":   implGlob,
					"path_required_any": pathRequiredAny,
				}
				violations = append(violations, makeViolation(moduleID, summary, rule, 0, ctx))
				break
			}
		}
	}
	return violations
}

// evalInterfaceNaming — params:
//
//	"path_required": substring identifying the port directory (e.g. "/port/in/")
//	"name_suffix":   required name suffix (e.g. "UseCase")
//	"name_regex":    optional alternative — a Go regex the simple name MUST match
func evalInterfaceNaming(
	moduleID string, summary model.JavaFileSummary, rule model.AuditRule,
) []auditvo.AuditViolation {
	pathRequired := rule.Params["path_required"]
	nameSuffix := rule.Params["name_suffix"]
	nameRegex := rule.Params["name_regex"]

	if pathRequired == "" {
		return nil
	}
	if nameSuffix == "" && nameRegex == "" {
		return nil
	}
	if !strings.Contains(summary.Path, pathRequired) {
		return nil
	}

	var re *regexp.Regexp
	if nameRegex != "" {
		re = regexp.MustCompile(nameRegex)
	}

	var violations []auditvo.AuditViolation
	for _, decl := range summary.Declarations {
		if decl.NodeType != "interface_declaration" {
			continue
		}
		violated := false
		if nameSuffix != "" && !strings.HasSuffix(decl.Name, nameSuffix) {
			violated = true
		}
		if re != nil && !re.MatchString(decl.Name) {
			violated = true
		}
		if violated {
			ctx := map[string]string{
				"file":          summary.Path,
				"name":          decl.Name,
				"path_required": pathRequired,
				"name_suffix":   nameSuffix,
				"name_regex":    nameRegex,
			}
			violations = append(violations, makeViolation(moduleID, summary, rule, 0, ctx))
		}
	}
	return violations
}

// evalForbiddenImport — params:
//
//	"path_scope":      substring restricting which files this rule applies to (e.g. "/domain/")
//	"import_prefix":   forbidden import prefix (e.g. "org.springframework.")
func evalForbiddenImport(
	moduleID string, summary model.JavaFileSummary, rule model.AuditRule,
) []auditvo.AuditViolation {
	pathScope := rule.Params["path_scope"]
	importPrefix := rule.Params["import_prefix"]
	if pathScope == "" || importPrefix == "" {
		return nil
	}
	if !strings.Contains(summary.Path, pathScope) {
		return nil
	}

	var violations []auditvo.AuditViolation
	for _, imp := range summary.Imports {
		if strings.HasPrefix(imp, importPrefix) {
			ctx := map[string]string{
				"file":          summary.Path,
				"import":        imp,
				"import_prefix": importPrefix,
			}
			violations = append(violations, makeViolation(moduleID, summary, rule, 0, ctx))
		}
	}
	return violations
}

// evalFieldTypeLayerViolation — params:
//
//	"path_scope":              substring restricting the rule (e.g. "/service/")
//	"forbidden_type_suffix":   suffix on the field's TYPE that flags a violation (e.g. "Jpa", "Repository")
//	"forbidden_type_substring": optional alternative substring match
func evalFieldTypeLayerViolation(
	moduleID string, summary model.JavaFileSummary, rule model.AuditRule,
) []auditvo.AuditViolation {
	pathScope := rule.Params["path_scope"]
	forbiddenSuffix := rule.Params["forbidden_type_suffix"]
	forbiddenSubstr := rule.Params["forbidden_type_substring"]
	if pathScope == "" {
		return nil
	}
	if !strings.Contains(summary.Path, pathScope) {
		return nil
	}

	var violations []auditvo.AuditViolation
	for _, decl := range summary.Declarations {
		for _, field := range decl.Fields {
			triggered := false
			if forbiddenSuffix != "" && strings.HasSuffix(field.Type, forbiddenSuffix) {
				triggered = true
			}
			if forbiddenSubstr != "" && strings.Contains(field.Type, forbiddenSubstr) {
				triggered = true
			}
			if triggered {
				ctx := map[string]string{
					"file":       summary.Path,
					"name":       decl.Name,
					"field_name": field.Name,
					"field_type": field.Type,
				}
				// JavaField has no Line field; use decl line (0 — JavaDeclaration
				// also has no Line field in the current model).
				violations = append(violations, makeViolation(moduleID, summary, rule, 0, ctx))
			}
		}
	}
	return violations
}

// makeViolation constructs an AuditViolation from the common fields plus a
// substitution context for the message and suggestion templates.
func makeViolation(
	moduleID string,
	summary model.JavaFileSummary,
	rule model.AuditRule,
	line int,
	ctx map[string]string,
) auditvo.AuditViolation {
	// Ensure {file} is always available even if the caller did not supply it.
	if _, ok := ctx["file"]; !ok {
		ctx["file"] = summary.Path
	}
	return auditvo.AuditViolation{
		RuleID:     rule.ID,
		Kind:       rule.Kind,
		Severity:   rule.Severity,
		ModuleID:   moduleID,
		FilePath:   summary.Path,
		Line:       line,
		Message:    substituteSuggestion(rule.Description, ctx),
		Suggestion: substituteSuggestion(rule.Suggestion, ctx),
	}
}

// substituteSuggestion replaces "{key}" tokens in the template string using
// the provided context map. Unknown tokens are left as-is.
func substituteSuggestion(template string, ctx map[string]string) string {
	out := template
	for k, v := range ctx {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	return out
}

// matchGlob performs a minimal glob match: when the pattern starts with "*",
// it checks that the candidate ends with the suffix after the "*". Otherwise
// it does an exact match. This covers the only pattern used in the bundled
// profile ("*UseCase").
func matchGlob(pattern, candidate string) bool {
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(candidate, pattern[1:])
	}
	return pattern == candidate
}
