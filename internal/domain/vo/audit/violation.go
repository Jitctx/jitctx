package audit

import "github.com/jitctx/jitctx/internal/domain/model"

// AuditViolation is one detected rule violation. Lives in the audit VO
// package (not in model/) because it is a use-case output value, not a
// domain entity. It is referenced by AuditProjectOutput.Sintatic.
type AuditViolation struct {
	RuleID     string
	Kind       model.AuditRuleKind
	Severity   model.AuditSeverity
	ModuleID   string
	FilePath   string // forward-slash, project-relative
	Line       int    // 1-based; 0 when the violation has no specific line (e.g. file-level forbidden_import)
	Message    string // rule.Description with placeholders substituted
	Suggestion string // rule.Suggestion with placeholders substituted; "" when none
}
