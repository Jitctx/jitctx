package fsprofile

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
	profileport "github.com/jitctx/jitctx/internal/domain/port/profile"
)

// BundleAuditRulesAdapter implements profile.LoadBundleAuditRulesPort by
// returning bundle.RawAuditRules, which is populated at load time by
// bundleMapper.toBundleDomain. No re-decoding of YAML occurs at call time.
//
// A nil bundle or a bundle with a nil/empty RawAuditRules slice produces
// (nil, nil) — callers must treat nil as "no audit rules declared".
type BundleAuditRulesAdapter struct{}

// NewBundleAuditRulesAdapter returns a BundleAuditRulesAdapter.
func NewBundleAuditRulesAdapter() *BundleAuditRulesAdapter {
	return &BundleAuditRulesAdapter{}
}

// LoadBundleAuditRules implements profile.LoadBundleAuditRulesPort.
// It returns the pre-mapped audit rules from bundle.RawAuditRules.
// Returns nil, nil when bundle is nil or has no audit_rules declared.
func (a *BundleAuditRulesAdapter) LoadBundleAuditRules(ctx context.Context, bundle *model.ProfileBundle) ([]model.AuditRule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if bundle == nil || len(bundle.RawAuditRules) == 0 {
		return nil, nil
	}
	return bundle.RawAuditRules, nil
}

// Compile-time assertion.
var _ profileport.LoadBundleAuditRulesPort = (*BundleAuditRulesAdapter)(nil)
