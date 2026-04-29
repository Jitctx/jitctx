package service

import (
	stdpath "path"
	"regexp"
	"slices"
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
		case model.AuditKindRequiredAnnotations:
			got = evalRequiredAnnotations(moduleID, summary, rule)
		case model.AuditKindForbiddenAnnotations:
			got = evalForbiddenAnnotations(moduleID, summary, rule)
		case model.AuditKindMethodNaming:
			got = evalMethodNaming(moduleID, summary, rule)
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
		if slices.Contains(decl.Annotations, annotation) {
			ctx := map[string]string{
				"file":          summary.Path,
				"name":          decl.Name,
				"path_required": pathRequired,
			}
			violations = append(violations, makeViolation(moduleID, summary, rule, 0, ctx))
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

// evalRequiredAnnotations — params:
//
//	"path_scope":    substring restricting which files this rule applies to
//	                 (e.g. "/domain/model/"). REQUIRED.
//	"annotations":   comma-joined list of annotation simple names (without
//	                 the leading "@") that must ALL be present on every
//	                 matching declaration. REQUIRED, non-empty.
//	"node_types":    optional comma-joined list of declaration node types
//	                 the rule applies to. Default "class_declaration". Use
//	                 "*" or empty to skip the node-type filter.
//
// The evaluator is language-neutral: it inspects only generic
// JavaDeclaration.Annotations entries (simple names extracted by the
// language adapter). It emits one violation per declaration that is
// missing at least one required annotation. The substitution context
// always populates:
//
//	{required} — comma-joined required annotations, in the order declared
//	{missing}  — "[A,B,...]" of the subset NOT present on the declaration
//	{name}     — the declaration's simple name
//	{file}     — summary.Path
//
// PC01RF-001 (all-of semantics) and PC01RF-009 (evidence-rich messages).
func evalRequiredAnnotations(
	moduleID string, summary model.JavaFileSummary, rule model.AuditRule,
) []auditvo.AuditViolation {
	pathScope := rule.Params["path_scope"]
	required := splitNonEmpty(rule.Params["annotations"])
	if pathScope == "" || len(required) == 0 {
		// Defensive: malformed rule. The loader/validator is expected to
		// reject these at profile load time; the evaluator is permissive
		// to keep test surface predictable.
		return nil
	}
	if !strings.Contains(summary.Path, pathScope) {
		return nil
	}

	nodeTypes := splitNonEmpty(rule.Params["node_types"])
	if len(nodeTypes) == 0 {
		nodeTypes = []string{"class_declaration"}
	}

	var violations []auditvo.AuditViolation
	for _, decl := range summary.Declarations {
		if !nodeTypeAllowed(decl.NodeType, nodeTypes) {
			continue
		}
		missing := missingAnnotations(decl.Annotations, required)
		if len(missing) == 0 {
			continue
		}
		ctx := map[string]string{
			"file":     summary.Path,
			"name":     decl.Name,
			"required": strings.Join(required, ","),
			"missing":  "[" + strings.Join(missing, ",") + "]",
		}
		violations = append(violations, makeViolation(moduleID, summary, rule, 0, ctx))
	}
	return violations
}

// splitNonEmpty splits a comma-joined string into trimmed, non-empty
// segments, preserving the original order. Returns nil for an empty input.
func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// nodeTypeAllowed reports whether the declaration's node type is in the
// configured filter. The wildcard token "*" matches any node type.
func nodeTypeAllowed(nodeType string, allowed []string) bool {
	if slices.Contains(allowed, "*") {
		return true
	}
	return slices.Contains(allowed, nodeType)
}

// missingAnnotations returns the subset of required entries NOT present in
// declared, preserving the order of required. Comparison is exact-match on
// simple names — the language adapter is responsible for stripping any "@"
// prefix and producing simple names in JavaDeclaration.Annotations.
func missingAnnotations(declared, required []string) []string {
	if len(required) == 0 {
		return nil
	}
	have := make(map[string]struct{}, len(declared))
	for _, a := range declared {
		have[a] = struct{}{}
	}
	var missing []string
	for _, r := range required {
		if _, ok := have[r]; !ok {
			missing = append(missing, r)
		}
	}
	return missing
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

// evalForbiddenAnnotations — params:
//
//	"path_scope":   substring restricting which files this rule applies to
//	               (e.g. "/src/main/java/"). REQUIRED.
//	"annotations":  comma-joined list of forbidden annotation simple names
//	               (without the leading "@"), e.g. "Autowired". The rule
//	               fires when ANY listed annotation is present on a
//	               matching target. REQUIRED, non-empty.
//	"target":       one of "class" | "field". Default "class".
//	               - "class"  → inspect decl.Annotations on every
//	                            JavaDeclaration whose NodeType is in
//	                            node_types (default class_declaration).
//	               - "field"  → inspect annotations on every
//	                            JavaField inside every JavaDeclaration
//	                            whose NodeType is in node_types.
//	"node_types":   optional comma-joined list of declaration node types.
//	               Default "class_declaration". "*" matches any.
//	"exempt_paths": optional comma-joined list of forward-slash globs.
//	               Each glob is matched against summary.Path with
//	               matchPathGlob. Any match exempts the file from this
//	               rule only.
//
// Substitution context:
//
//	{file}     — summary.Path
//	{name}     — declaration simple name (target=class) OR field name (target=field)
//	{forbidden} — comma-joined params["annotations"] (verbatim, in order)
//	{found}    — "[A,B,...]" of the subset of forbidden annotations actually
//	             present on the offending target (deterministic, in the
//	             order the annotations were declared in params).
//
// Violation Line:
//
//   - target=class  → 0 (class line is not currently captured).
//   - target=field  → field.Line (1-based; PC01US-004 Scenario 1 asserts
//     "violation reported on the field's line").
//
// PC01RF-002 / PC01RF-003 / PC01RF-008 / PC01RF-009.
func evalForbiddenAnnotations(
	moduleID string, summary model.JavaFileSummary, rule model.AuditRule,
) []auditvo.AuditViolation {
	pathScope := rule.Params["path_scope"]
	forbidden := splitNonEmpty(rule.Params["annotations"])
	if pathScope == "" || len(forbidden) == 0 {
		return nil
	}
	if !strings.Contains(summary.Path, pathScope) {
		return nil
	}
	if pathExempt(rule, summary.Path) {
		return nil
	}

	nodeTypes := splitNonEmpty(rule.Params["node_types"])
	if len(nodeTypes) == 0 {
		nodeTypes = []string{"class_declaration"}
	}

	target := rule.Params["target"]
	if target == "" {
		target = "class"
	}

	forbiddenRaw := rule.Params["annotations"]

	var violations []auditvo.AuditViolation
	for _, decl := range summary.Declarations {
		if !nodeTypeAllowed(decl.NodeType, nodeTypes) {
			continue
		}
		switch target {
		case "class":
			found := intersectAnnotations(decl.Annotations, forbidden)
			if len(found) > 0 {
				ctx := map[string]string{
					"file":      summary.Path,
					"name":      decl.Name,
					"forbidden": forbiddenRaw,
					"found":     "[" + strings.Join(found, ",") + "]",
				}
				violations = append(violations, makeViolation(moduleID, summary, rule, 0, ctx))
			}
		case "field":
			for _, field := range decl.Fields {
				found := intersectAnnotations(field.Annotations, forbidden)
				if len(found) > 0 {
					ctx := map[string]string{
						"file":      summary.Path,
						"name":      field.Name,
						"forbidden": forbiddenRaw,
						"found":     "[" + strings.Join(found, ",") + "]",
					}
					violations = append(violations, makeViolation(moduleID, summary, rule, field.Line, ctx))
				}
			}
		default:
			// Unknown target — defensive, no violations.
		}
	}
	return violations
}

// intersectAnnotations returns the subset of forbidden entries that appear in
// declared, preserving the order of forbidden (deterministic output per
// PC01RNF-003).
func intersectAnnotations(declared, forbidden []string) []string {
	have := make(map[string]struct{}, len(declared))
	for _, a := range declared {
		have[a] = struct{}{}
	}
	var found []string
	for _, f := range forbidden {
		if _, ok := have[f]; ok {
			found = append(found, f)
		}
	}
	return found
}

// matchPathGlob reports whether path matches the forward-slash glob pattern.
// Supported syntax:
//
//	"/literal/segment/"         — substring match (no glob meta-chars).
//	"*foo" / "foo*" / "*foo*"   — single-segment globs (path.Match style).
//	"**/seg/**" / "**/seg"       — "**" matches zero or more "/"-separated
//	                               segments (including none).
//
// Implementation: split pattern and path on "/"; walk both concurrently with
// "**" consuming any number of path segments. No regex compilation;
// deterministic; allocation-free in the common case.
//
// Returns (matched bool). Never returns an error: a malformed pattern is
// treated as "no match" so a profile typo never panics the run.
func matchPathGlob(pattern, path string) bool {
	patSegs := strings.Split(pattern, "/")
	pathSegs := strings.Split(path, "/")
	return matchSegments(patSegs, pathSegs)
}

// matchSegments is the recursive worker for matchPathGlob.
func matchSegments(patSegs, pathSegs []string) bool {
	for len(patSegs) > 0 {
		pat := patSegs[0]
		if pat == "**" {
			patSegs = patSegs[1:]
			if len(patSegs) == 0 {
				// "**" at end matches zero or more remaining segments.
				return true
			}
			// Try consuming 0..N path segments before the next pattern segment.
			for i := 0; i <= len(pathSegs); i++ {
				if matchSegments(patSegs, pathSegs[i:]) {
					return true
				}
			}
			return false
		}
		// Non-"**" segment: requires at least one path segment.
		if len(pathSegs) == 0 {
			return false
		}
		ok, err := stdpathMatch(pat, pathSegs[0])
		if err != nil || !ok {
			return false
		}
		patSegs = patSegs[1:]
		pathSegs = pathSegs[1:]
	}
	return len(pathSegs) == 0
}

// stdpathMatch delegates to path.Match for single-segment glob matching.
// Returns (false, nil) for an empty pattern so callers can treat it as
// no-match without propagating errors.
func stdpathMatch(pattern, name string) (bool, error) {
	if pattern == "" {
		return name == "", nil
	}
	return stdpath.Match(pattern, name)
}

// pathExempt reports whether the given path matches any glob in
// rule.Params["exempt_paths"] (comma-joined). Empty/missing key returns false.
// Used by evalForbiddenAnnotations; reusable for future per-rule-exemption
// evaluators (PC01RF-008 cross-cutting).
func pathExempt(rule model.AuditRule, path string) bool {
	globs := splitNonEmpty(rule.Params["exempt_paths"])
	for _, g := range globs {
		if matchPathGlob(g, path) {
			return true
		}
	}
	return false
}

// evalMethodNaming — params:
//
//	"path_scope":    substring restricting which files this rule applies to
//	                (e.g. "src/test/java/"). REQUIRED.
//	"triggered_by": single annotation simple name (without "@") that must be
//	                present on the method for the rule to evaluate it
//	                (e.g. "Override"). REQUIRED.
//	"name_pattern": Go regular expression the method name MUST match
//	                (e.g. "^should[A-Z].*_when[A-Z].*$"). REQUIRED.
//	"node_types":   optional comma-joined list of declaration node types the
//	                rule applies to. Default "class_declaration".
//	"exempt_paths": optional comma-joined list of forward-slash globs.
//	                Any match exempts the file from this rule.
//
// The evaluator emits one violation per method that carries the trigger
// annotation AND whose name does NOT match name_pattern. Substitution context:
//
//	{file}             — summary.Path
//	{name}             — method.Name
//	{expected_pattern} — params["name_pattern"]
//	{triggered_by}     — params["triggered_by"]
//
// PC01RF-004 (method-scoped rules with regex name patterns), PC01RF-009
// (evidence-rich messages).
func evalMethodNaming(
	moduleID string, summary model.JavaFileSummary, rule model.AuditRule,
) []auditvo.AuditViolation {
	pathScope := rule.Params["path_scope"]
	triggeredBy := rule.Params["triggered_by"]
	namePattern := rule.Params["name_pattern"]
	if pathScope == "" || triggeredBy == "" || namePattern == "" {
		return nil
	}
	if !strings.Contains(summary.Path, pathScope) {
		return nil
	}
	if pathExempt(rule, summary.Path) {
		return nil
	}

	re, err := regexp.Compile(namePattern)
	if err != nil {
		// Malformed regex — skip rule defensively; profile-validate should
		// catch this at load time.
		return nil
	}

	nodeTypes := splitNonEmpty(rule.Params["node_types"])
	if len(nodeTypes) == 0 {
		nodeTypes = []string{"class_declaration"}
	}

	var violations []auditvo.AuditViolation
	for _, decl := range summary.Declarations {
		if !nodeTypeAllowed(decl.NodeType, nodeTypes) {
			continue
		}
		for _, method := range decl.Methods {
			if !slices.Contains(method.Annotations, triggeredBy) {
				continue
			}
			if re.MatchString(method.Name) {
				continue
			}
			ctx := map[string]string{
				"file":             summary.Path,
				"name":             method.Name,
				"expected_pattern": namePattern,
				"triggered_by":     triggeredBy,
			}
			violations = append(violations, makeViolation(moduleID, summary, rule, method.Line, ctx))
		}
	}
	return violations
}
