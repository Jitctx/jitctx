package profile

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// LoadBundleAuditRulesPort returns the audit rules declared in a
// directory-shaped profile bundle (EP04US-001 / EP04US-006 path).
//
// The implementation reads from bundle.RawAuditRules — the verbatim
// audit_rules section preserved at load time by the bundle loader.
// Returns an empty slice (not an error) when the bundle has no
// audit_rules: key. Unknown rule kinds are dropped and logged at
// WARN, mirroring fsprofile.Loader.LoadAuditRules. Unknown severity
// values are fatal and return a wrapped ErrProfileInvalid.
//
// This port is a SIBLING to LoadAuditRulesPort (which still loads
// from the legacy single-file YAML at <userDir>/<name>.yaml). The
// audit use case selects between them based on whether the active
// profile is bundle-shaped or single-file shaped — see
// audituc.Impl.Execute step 5.
type LoadBundleAuditRulesPort interface {
	LoadBundleAuditRules(ctx context.Context, bundle *model.ProfileBundle) ([]model.AuditRule, error)
}
