package audit

import "github.com/jitctx/jitctx/internal/domain/model"

// SeverityBadge returns the markdown emoji + label badge for a severity
// per EP03RF-007. Centralised in the domain so renderer and tests share
// the canonical string.
func SeverityBadge(s model.AuditSeverity) string {
	switch s {
	case model.AuditSeverityError:
		return "🔴 ERROR"
	case model.AuditSeverityWarning:
		return "🟡 WARNING"
	case model.AuditSeverityInfo:
		return "🔵 INFO"
	}
	return string(s)
}
