package service

import (
	stdpath "path"
	"regexp"
	"slices"
	"strconv"
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
		case model.AuditKindForbiddenFieldTypePattern:
			got = evalForbiddenFieldTypePattern(moduleID, summary, rule)
		case model.AuditKindRequiredParameterizedSupertype:
			got = evalRequiredParameterizedSupertype(moduleID, summary, rule)
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
//	"path_scope":      substring restricting which files this rule applies to
//	                   (e.g. "src/main/java/"). REQUIRED.
//	"annotations":     comma-joined list of annotation simple names (without
//	                   the leading "@") that must ALL be present on every
//	                   matching declaration. REQUIRED, non-empty. Order is
//	                   preserved and used to derive deterministic
//	                   "missing=[...]" evidence.
//	"expected_values": OPTIONAL comma-joined list of "Annotation=Value" pairs
//	                   (e.g. "ExtendWith=MockitoExtension.class"). For each
//	                   pair, when the annotation IS present on a matching
//	                   declaration, the evaluator compares the text of its
//	                   first positional argument (decl.AnnotationArgs[ann])
//	                   against the right-hand value. The comparison is exact
//	                   string equality. A mismatch emits ONE additional
//	                   violation per pair, separate from the missing-set
//	                   violation. PC01RF-007.
//	                   Parsing rules:
//	                     - splits on "," only;
//	                     - each piece is split on the FIRST "=";
//	                     - whitespace around the annotation name AND value is
//	                       trimmed;
//	                     - a piece without "=" is ignored (defensive);
//	                     - duplicate keys: LAST occurrence wins (deterministic
//	                       on a given input string).
//	                   Limitation: argument values containing commas are NOT
//	                   supported. Profile authors needing such values must
//	                   wait for a future-extension key (out of scope; Q7).
//	"non_empty_value_annotations": OPTIONAL comma-joined list of annotation
//	                   simple names (without the leading "@"). For each listed
//	                   annotation that IS present on a matching declaration,
//	                   the evaluator checks whether decl.AnnotationArgs[ann]
//	                   is considered empty by isEmptyAnnotationArg (covers
//	                   the empty-string, bare double-quote pair, and bare
//	                   single-quote pair forms). When empty, ONE additional
//	                   violation is emitted per offending name with evidence
//	                   "annotation=<ann>, value=empty, expected=non-empty".
//	                   Names NOT present on the declaration are silently
//	                   skipped (the missing-violation path covers absence).
//	                   PC01RF-007 (non-empty matcher), PC01US-010.
//	"node_types":      optional comma-joined list of declaration node types
//	                   the rule applies to. Default "class_declaration". Use
//	                   "*" or empty to skip the node-type filter.
//
// Substitution context (per emitted violation):
//
//	{file}     — summary.Path
//	{name}     — declaration simple name
//	{required} — comma-joined params["annotations"] (verbatim, in order)
//	{evidence} — for the missing-violation: "missing=[A,B,...]" subset
//	             NOT present, ordered by params["annotations"];
//	           — for an arg-mismatch violation: literal
//	             "annotation=<ann>, expected_value=<expected>, actual=<actual>"
//	             where <actual> is decl.AnnotationArgs[<ann>] (may be "");
//	           — for a non-empty-value violation: literal
//	             "annotation=<ann>, value=empty, expected=non-empty"
//	             where <ann> is the offending annotation simple name.
//	{missing}  — same as {evidence} for the missing-violation (backward
//	             compat for templates that still use {missing}); not
//	             populated for arg-mismatch or non-empty-value violations.
//
// Determinism (PC01RNF-003):
//   - "expected_values" is parsed into an ORDERED slice of pairs in the
//     order the pairs appear in the input string.
//   - The evaluator iterates that ordered slice; it never iterates a Go
//     map for emit ordering.
//   - "non_empty_value_annotations" is split via splitNonEmpty and iterated
//     in input-string order, after the missing-violation and expected_values
//     violations.
//   - Per-declaration emit order: missing → expected_values mismatches →
//     non_empty_value_annotations empty-value violations.
//
// PC01RF-001 (all-of presence), PC01RF-007 (argument matching),
// PC01RNF-001 (engine language-neutrality — no Java/Spring identifiers
// referenced here), PC01RF-009 (evidence-rich messages), PC01US-010.
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

	expected := parseExpectedValues(rule.Params["expected_values"])

	var violations []auditvo.AuditViolation
	for _, decl := range summary.Declarations {
		if !nodeTypeAllowed(decl.NodeType, nodeTypes) {
			continue
		}
		missing := missingAnnotations(decl.Annotations, required)
		if len(missing) > 0 {
			missingEvidence := "missing=[" + strings.Join(missing, ",") + "]"
			ctx := map[string]string{
				"file":     summary.Path,
				"name":     decl.Name,
				"required": strings.Join(required, ","),
				"evidence": missingEvidence,
				// backward compat: {missing} keeps the same value as {evidence}
				// for templates authored before PC01US-006.
				"missing": missingEvidence,
			}
			violations = append(violations, makeViolation(moduleID, summary, rule, 0, ctx))
		}
		for _, pair := range expected {
			if !slices.Contains(decl.Annotations, pair.Annotation) {
				// Already covered by missing-violation (or not required); skip.
				continue
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
		// PC01US-010: non-empty-value branch.
		// For every annotation listed in non_empty_value_annotations that
		// IS present on this declaration, emit one violation if its
		// captured argument text is considered empty by isEmptyAnnotationArg.
		//
		// Determinism: input-string order via splitNonEmpty (PC01RNF-003).
		// Short-circuit: an annotation absent from decl.Annotations is
		// already covered by the missing-violation path; this branch only
		// fires for annotations actually present whose captured arg text is
		// empty.
		for _, ann := range splitNonEmpty(rule.Params["non_empty_value_annotations"]) {
			if !slices.Contains(decl.Annotations, ann) {
				continue
			}
			actual := decl.AnnotationArgs[ann]
			if !isEmptyAnnotationArg(actual) {
				continue
			}
			evidence := "annotation=" + ann + ", value=empty, expected=non-empty"
			ctxNe := map[string]string{
				"file":     summary.Path,
				"name":     decl.Name,
				"required": strings.Join(required, ","),
				"evidence": evidence,
			}
			violations = append(violations, makeViolation(moduleID, summary, rule, 0, ctxNe))
		}
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
func parseExpectedValues(s string) []expectedValuePair {
	if s == "" {
		return nil
	}
	pieces := strings.Split(s, ",")
	// Use a map to track the last seen value per annotation key, and a
	// separate slice to track the order of FIRST appearance.
	order := make([]string, 0, len(pieces))
	last := make(map[string]string, len(pieces))
	for _, piece := range pieces {
		annRaw, valRaw, ok := strings.Cut(piece, "=")
		if !ok {
			// No "=" — malformed piece; skip defensively.
			continue
		}
		ann := strings.TrimSpace(annRaw)
		val := strings.TrimSpace(valRaw)
		if ann == "" {
			continue
		}
		if _, seen := last[ann]; !seen {
			order = append(order, ann)
		}
		last[ann] = val
	}
	if len(order) == 0 {
		return nil
	}
	out := make([]expectedValuePair, 0, len(order))
	for _, ann := range order {
		out = append(out, expectedValuePair{Annotation: ann, Expected: last[ann]})
	}
	return out
}

// isEmptyAnnotationArg reports whether the captured annotation-argument text
// represents an "empty" value, per PC01RF-007. The Tree-sitter parser stores
// annotation arguments VERBATIM, including surrounding quotes for string
// literals. Therefore three forms map to "empty":
//
//   - ""     — marker annotation (no argument list captured); the parser
//     leaves the entry as the empty string.
//   - `""`   — explicit empty string literal @Ann("").
//   - `”`   — explicit empty char/string literal @Ann(”) (Java char
//     literal; defensive — most JVM toolchains reject this at compile
//     time, but the parser would still capture it verbatim).
//
// All other captures (including whitespace-only such as `" "` or `"0"` or
// `"false"`) are treated as NON-empty: the predicate is strictly about the
// parser-captured text, not about semantic emptiness. Profile authors
// needing semantic emptiness can layer expected_values on top.
//
// PC01RF-007 (annotation argument matching), PC01RNF-001 (no language
// identifier in this function — only verbatim parser output is matched),
// PC01RNF-003 (deterministic — pure string comparison), PC01US-010.
func isEmptyAnnotationArg(captured string) bool {
	switch captured {
	case "", `""`, `''`:
		return true
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

// evalForbiddenFieldTypePattern — params:
//
//	"path_scope":              substring restricting which files this rule applies to
//	                           (e.g. "src/main/java/"). REQUIRED.
//	"forbidden_type_patterns": comma-joined list of "Outer<Inner>" patterns where
//	                           Inner may contain a single "*" glob. REQUIRED.
//	"node_types":              optional comma-joined list of declaration node types.
//	                           Default "class_declaration".
//	"exempt_paths":            optional comma-joined list of forward-slash globs.
//	                           Any match exempts the file from this rule.
//
// Substitution context (per emitted violation):
//
//	{file}            — summary.Path
//	{name}            — declaration simple name
//	{field_name}      — field simple name
//	{type}            — resolved FQN of outer + "<" + inner + ">"
//	{matched_pattern} — the pattern that caused the match
//
// PC01RF-005 (parameterized type-argument matching), PC01RF-009 (evidence-rich
// messages), PC01RNF-001 (engine language-neutrality), PC01RNF-003 (deterministic).
func evalForbiddenFieldTypePattern(
	moduleID string, summary model.JavaFileSummary, rule model.AuditRule,
) []auditvo.AuditViolation {
	pathScope := rule.Params["path_scope"]
	patterns := splitNonEmpty(rule.Params["forbidden_type_patterns"])
	if pathScope == "" || len(patterns) == 0 {
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

	var violations []auditvo.AuditViolation
	for _, decl := range summary.Declarations {
		if !nodeTypeAllowed(decl.NodeType, nodeTypes) {
			continue
		}
		for _, field := range decl.Fields {
			for _, pattern := range patterns {
				outer, inner, matched := matchTypePattern(field.Type, pattern)
				if !matched {
					continue
				}
				fqnOuter := resolveFQN(outer, summary.Imports)
				typeStr := fqnOuter + "<" + inner + ">"
				ctx := map[string]string{
					"file":            summary.Path,
					"name":            decl.Name,
					"field_name":      field.Name,
					"type":            typeStr,
					"matched_pattern": pattern,
				}
				violations = append(violations, makeViolation(moduleID, summary, rule, field.Line, ctx))
				break // one violation per field (first matched pattern wins)
			}
		}
	}
	return violations
}

// matchTypePattern parses a parameterized field type (e.g. "List<OrderEntity>")
// and a pattern (e.g. "List<*Entity>"), then reports whether the outer type
// matches exactly and the inner type matches the glob. Non-parameterized field
// types (no angle brackets) return matched=false. Pattern without brackets also
// returns matched=false.
//
// Return values:
//
//	outer   — the outer type name (trimmed), even on no-match
//	inner   — the inner type argument (trimmed), empty when no brackets in fieldType
//	matched — true only when outer equals outerPat AND inner matches the innerPat glob
func matchTypePattern(fieldType, pattern string) (outer, inner string, matched bool) {
	firstLT := strings.Index(fieldType, "<")
	lastGT := strings.LastIndex(fieldType, ">")
	if firstLT < 0 || lastGT < 0 || lastGT <= firstLT {
		return strings.TrimSpace(fieldType), "", false
	}
	outer = strings.TrimSpace(fieldType[:firstLT])
	inner = strings.TrimSpace(fieldType[firstLT+1 : lastGT])

	patFirstLT := strings.Index(pattern, "<")
	patLastGT := strings.LastIndex(pattern, ">")
	if patFirstLT < 0 || patLastGT < 0 || patLastGT <= patFirstLT {
		return outer, inner, false
	}
	outerPat := strings.TrimSpace(pattern[:patFirstLT])
	innerPat := strings.TrimSpace(pattern[patFirstLT+1 : patLastGT])

	if outer != outerPat {
		return outer, inner, false
	}

	matched = globMatch(innerPat, inner)
	return outer, inner, matched
}

// resolveFQN resolves a simple class name to a fully-qualified name using the
// file's import list. Returns simple unchanged when no matching import is found.
// No java.lang.* synthesis — profile authors must supply explicit imports.
// PC01RF-005, Q3.
func resolveFQN(simple string, imports []string) string {
	for _, imp := range imports {
		if i := strings.LastIndex(imp, "."); i >= 0 && imp[i+1:] == simple {
			return imp
		}
	}
	return simple
}

// matchOuterGlob is the single-* glob matcher used for the outer-type name
// comparison in required_parameterized_supertype evaluation. Semantics are
// identical to the inner-glob branch extracted into globMatch.
func matchOuterGlob(pattern, candidate string) bool { return globMatch(pattern, candidate) }

// matchInnerGlob is the single-* glob matcher used for per-position type-argument
// comparison in required_parameterized_supertype evaluation. Semantics are
// identical to matchOuterGlob; kept distinct for readability at call sites.
func matchInnerGlob(pattern, candidate string) bool { return globMatch(pattern, candidate) }

// parseSupertypePattern splits a verbatim "Outer<arg1,arg2,...>" pattern into
// the outer string, the arity, and the slot tokens. Splitting on commas honours
// nested angle brackets — a `<` increments depth, a `>` decrements it, and only
// depth-zero commas split. When the pattern contains no `<>` brackets the
// function returns ("", 0, nil, false). The bool reports whether the pattern is
// well-formed (non-empty outer, at least one slot).
//
// Example: parseSupertypePattern("UseCase<*,*>") returns ("UseCase", 2, ["*","*"], true).
func parseSupertypePattern(pattern string) (outer string, arity int, slots []string, ok bool) {
	firstLT := strings.Index(pattern, "<")
	lastGT := strings.LastIndex(pattern, ">")
	if firstLT < 0 || lastGT < 0 || lastGT <= firstLT {
		return "", 0, nil, false
	}
	outer = strings.TrimSpace(pattern[:firstLT])
	if outer == "" {
		return "", 0, nil, false
	}
	inner := pattern[firstLT+1 : lastGT]
	slots = splitTopLevel(inner)
	if len(slots) == 0 {
		return "", 0, nil, false
	}
	return outer, len(slots), slots, true
}

// splitTopLevel splits a string on top-level commas (depth-zero with respect to
// angle brackets). Each returned token has surrounding whitespace trimmed.
func splitTopLevel(s string) []string {
	var out []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			depth++
		case '>':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	out = append(out, strings.TrimSpace(s[start:]))
	return out
}

// evalRequiredParameterizedSupertype — params:
//
//	"path_scope":         substring restricting which files this rule applies to. REQUIRED.
//	"expected_supertype": REQUIRED. The outer-type glob plus parameter slot pattern,
//	                      e.g. "UseCase<*,*>". The inner comma-separated tokens are
//	                      read for ARITY only; per-slot globs come from "args".
//	"args":               OPTIONAL comma-joined list of per-position globs. When
//	                      absent, all positions default to "*". When present, must
//	                      have arity matching expected_supertype; otherwise the rule
//	                      is skipped defensively.
//	"supertype_kind":     OPTIONAL — "extends" | "implements" | "" (default "").
//	                      When non-empty, only entries with the matching Kind are
//	                      considered. "actual=none" fires when zero entries remain.
//	"node_types":         OPTIONAL — defaults to "class_declaration".
//	"exempt_paths":       OPTIONAL comma-joined forward-slash globs; any match
//	                      exempts the file from this rule.
//
// Substitution context (per emitted violation):
//
//	{file}               — summary.Path
//	{name}               — declaration simple name
//	{expected_supertype} — verbatim params["expected_supertype"]
//	{expected_arity}     — strconv-formatted expected arity
//	{actual}             — "none" when no candidate matched outer glob; otherwise
//	                       Outer+"<"+strings.Join(TypeArgs,",")+">"+
//	{actual_arity}       — strconv-formatted len(TypeArgs); "0" when actual=="none"
//	{kind}               — candidate Kind; "" when actual=="none"
//
// PC01RF-006, PC01RF-009 (evidence-rich messages), PC01RNF-001 (no
// Java/Spring/Lombok identifiers in this function), PC01RNF-003 (deterministic).
func evalRequiredParameterizedSupertype(
	moduleID string, summary model.JavaFileSummary, rule model.AuditRule,
) []auditvo.AuditViolation {
	pathScope := rule.Params["path_scope"]
	expectedSupertypeRaw := rule.Params["expected_supertype"]
	if pathScope == "" || expectedSupertypeRaw == "" {
		return nil
	}
	if !strings.Contains(summary.Path, pathScope) {
		return nil
	}
	if pathExempt(rule, summary.Path) {
		return nil
	}

	expectedOuter, expectedArity, slots, ok := parseSupertypePattern(expectedSupertypeRaw)
	if !ok {
		// Malformed pattern — no angle brackets; skip rule defensively.
		return nil
	}

	// Resolve per-slot arg globs.
	argGlobs := splitNonEmpty(rule.Params["args"])
	if len(argGlobs) == 0 {
		argGlobs = make([]string, expectedArity)
		for i := range argGlobs {
			argGlobs[i] = "*"
		}
	} else if len(argGlobs) != expectedArity {
		// args arity mismatch — skip defensively.
		return nil
	}
	_ = slots // slots used only to derive expectedArity; argGlobs carry the actual per-position globs.

	supertypeKindFilter := strings.ToLower(strings.TrimSpace(rule.Params["supertype_kind"]))

	nodeTypes := splitNonEmpty(rule.Params["node_types"])
	if len(nodeTypes) == 0 {
		nodeTypes = []string{"class_declaration"}
	}

	expectedArityStr := strconv.Itoa(expectedArity)

	var violations []auditvo.AuditViolation
	for _, decl := range summary.Declarations {
		if !nodeTypeAllowed(decl.NodeType, nodeTypes) {
			continue
		}

		// Step 1: filter by supertype_kind.
		candidates := decl.ParameterizedSupertypes
		if supertypeKindFilter != "" {
			filtered := candidates[:0:0]
			for _, ps := range candidates {
				if string(ps.Kind) == supertypeKindFilter {
					filtered = append(filtered, ps)
				}
			}
			candidates = filtered
		}

		// Step 2: filter by outer-glob match.
		outerMatched := candidates[:0:0]
		for _, ps := range candidates {
			if matchOuterGlob(expectedOuter, ps.Outer) {
				outerMatched = append(outerMatched, ps)
			}
		}

		// Step 3: if no outer match → "actual=none" violation.
		if len(outerMatched) == 0 {
			ctx := map[string]string{
				"file":               summary.Path,
				"name":               decl.Name,
				"expected_supertype": expectedSupertypeRaw,
				"expected_arity":     expectedArityStr,
				"actual":             "none",
				"actual_arity":       "0",
				"kind":               "",
			}
			violations = append(violations, makeViolation(moduleID, summary, rule, 0, ctx))
			continue
		}

		// Step 4: pick the first matching candidate.
		c := outerMatched[0]
		actual := c.Outer + "<" + strings.Join(c.TypeArgs, ",") + ">"
		actualArity := strconv.Itoa(len(c.TypeArgs))
		kindStr := string(c.Kind)

		if len(c.TypeArgs) != expectedArity {
			ctx := map[string]string{
				"file":               summary.Path,
				"name":               decl.Name,
				"expected_supertype": expectedSupertypeRaw,
				"expected_arity":     expectedArityStr,
				"actual":             actual,
				"actual_arity":       actualArity,
				"kind":               kindStr,
			}
			violations = append(violations, makeViolation(moduleID, summary, rule, 0, ctx))
			continue
		}

		// Step 5: check per-slot arg globs.
		slotViolation := false
		for i, glob := range argGlobs {
			if !matchInnerGlob(glob, c.TypeArgs[i]) {
				slotViolation = true
				break
			}
		}
		if slotViolation {
			ctx := map[string]string{
				"file":               summary.Path,
				"name":               decl.Name,
				"expected_supertype": expectedSupertypeRaw,
				"expected_arity":     expectedArityStr,
				"actual":             actual,
				"actual_arity":       actualArity,
				"kind":               kindStr,
			}
			violations = append(violations, makeViolation(moduleID, summary, rule, 0, ctx))
		}
	}
	return violations
}
