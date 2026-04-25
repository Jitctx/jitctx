package profile

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// LoadAuditRulesPort returns the audit rules declared in the active
// profile. Returns an empty slice (not an error) when the profile has no
// audit_rules: key — that is a valid clean-state profile.
type LoadAuditRulesPort interface {
	LoadAuditRules(ctx context.Context, profileName string) ([]model.AuditRule, error)
}
