package model

// AuditRuleKind is the small enum of evaluator kinds. Each value maps to a
// fixed Go evaluator in internal/domain/service/auditRuleEvaluator.go.
// Adding a new kind is a Go change; adding a new rule under an existing
// kind is profile-YAML-only (RNF-004).
type AuditRuleKind string

const (
	AuditKindAnnotationPathMismatch  AuditRuleKind = "annotation_path_mismatch"
	AuditKindImplementsPathMismatch  AuditRuleKind = "implements_path_mismatch"
	AuditKindInterfaceNaming         AuditRuleKind = "interface_naming"
	AuditKindForbiddenImport         AuditRuleKind = "forbidden_import"
	AuditKindFieldTypeLayerViolation AuditRuleKind = "field_type_layer_violation"
	// AuditKindRequiredAnnotations enforces all-of semantics: the rule
	// declares a list of annotation simple names that MUST be present on
	// every matching declaration. The evaluator emits one violation per
	// declaration that is missing at least one of them, with the missing
	// subset surfaced as evidence under the {missing} substitution token.
	// PC01RF-001 (proposal-changes-01).
	AuditKindRequiredAnnotations AuditRuleKind = "required_annotations"

	// AuditKindForbiddenAnnotations enforces that NONE of the listed
	// annotation simple names are present on a target. The target scope
	// is selected by params["target"] ∈ {"class", "field"} (default
	// "class"). Per-rule path exemptions are honoured via
	// params["exempt_paths"] (comma-joined list of forward-slash globs).
	// PC01RF-002, PC01RF-003, PC01RF-008.
	AuditKindForbiddenAnnotations AuditRuleKind = "forbidden_annotations"

	// AuditKindMethodNaming enforces that every method carrying a configured
	// trigger annotation (params["triggered_by"]) has a name matching a
	// configured Go regex (params["name_pattern"]). The rule is scoped to
	// files whose path contains params["path_scope"]. Per-rule path
	// exemptions are honoured via params["exempt_paths"].
	// PC01RF-004, PC01RF-009.
	AuditKindMethodNaming AuditRuleKind = "method_naming"

	// AuditKindForbiddenFieldTypePattern flags fields whose type matches any
	// of the configured "Outer<Inner>" patterns. Inner supports a single
	// "*" glob (suffix/prefix/full wildcard). Non-parameterized field types
	// are silently skipped (Q4 of plan §8). PC01RF-005.
	AuditKindForbiddenFieldTypePattern AuditRuleKind = "forbidden_field_type_pattern"

	// AuditKindRequiredParameterizedSupertype enforces that every class
	// declaration matching the rule scope declares a parameterized supertype
	// (extends or implements) whose outer type matches a configured single-*
	// glob and whose number of type arguments matches a configured arity,
	// optionally with per-argument glob constraints.
	//
	// Non-parameterized supertypes (e.g. `extends Object` or `implements
	// Cloneable` written without `<...>`) do NOT match the outer pattern —
	// a class that declares only non-parameterized supertypes is treated
	// identically to a class with no supertype clauses and produces an
	// "actual=none" violation.
	//
	// Exactly ONE violation is emitted per declaration (mismatch driven by
	// the FIRST matching candidate — deterministic on a given AST per
	// PC01RNF-003). The outer pattern uses single-* glob semantics
	// (leading-* suffix-match, trailing-* prefix-match, bare * wildcard,
	// exact otherwise).
	//
	// PC01RF-006 (parameterized-supertype matching).
	AuditKindRequiredParameterizedSupertype AuditRuleKind = "required_parameterized_supertype"
)

// AuditSeverity is the severity badge attached to a violation.
// It is a domain-level concept (used by both the rule definition in the
// profile and the violation produced by the evaluator).
type AuditSeverity string

const (
	AuditSeverityError   AuditSeverity = "ERROR"
	AuditSeverityWarning AuditSeverity = "WARNING"
	AuditSeverityInfo    AuditSeverity = "INFO"
)

// AuditRule is one declarative rule loaded from the active profile YAML.
// The Params map carries the kind-specific knobs; the evaluator selects
// the keys it needs. Unknown keys are tolerated (forward-compatible).
type AuditRule struct {
	ID          string            // stable id used in report output and (future) disable list
	Kind        AuditRuleKind     // selects the Go evaluator
	Severity    AuditSeverity     // ERROR | WARNING | INFO
	Description string            // human-readable rule description
	Suggestion  string            // template suggestion text; may contain "{file}" / "{path}" tokens
	Params      map[string]string // kind-specific parameters (annotation, path_required, import_prefix, etc.)
}
