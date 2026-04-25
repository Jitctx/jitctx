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
