package fsprofile

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
)

func TestBundleAuditRulesAdapter_HappyPath(t *testing.T) {
	t.Parallel()

	rules := []model.AuditRule{
		{
			ID:       "rule-one",
			Kind:     model.AuditKindAnnotationPathMismatch,
			Severity: model.AuditSeverityError,
			Params:   map[string]string{"annotation": "Service", "path_required": "service"},
		},
		{
			ID:       "rule-two",
			Kind:     model.AuditKindInterfaceNaming,
			Severity: model.AuditSeverityWarning,
			Params:   nil,
		},
	}
	bundle := &model.ProfileBundle{
		RawAuditRules: rules,
	}

	adapter := NewBundleAuditRulesAdapter()
	got, err := adapter.LoadBundleAuditRules(t.Context(), bundle)

	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, rules[0].ID, got[0].ID)
	require.Equal(t, rules[0].Kind, got[0].Kind)
	require.Equal(t, rules[0].Severity, got[0].Severity)
	require.Equal(t, rules[0].Params, got[0].Params)
	require.Equal(t, rules[1].ID, got[1].ID)
	require.Equal(t, rules[1].Kind, got[1].Kind)
	require.Equal(t, rules[1].Severity, got[1].Severity)
}

func TestBundleAuditRulesAdapter_EmptyBundle(t *testing.T) {
	t.Parallel()

	bundle := &model.ProfileBundle{
		RawAuditRules: nil,
	}

	adapter := NewBundleAuditRulesAdapter()
	got, err := adapter.LoadBundleAuditRules(t.Context(), bundle)

	require.NoError(t, err)
	require.Nil(t, got)
}

func TestBundleAuditRulesAdapter_NilBundle(t *testing.T) {
	t.Parallel()

	adapter := NewBundleAuditRulesAdapter()
	got, err := adapter.LoadBundleAuditRules(t.Context(), nil)

	// Production code: nil bundle → (nil, nil)
	require.NoError(t, err)
	require.Nil(t, got)
}
