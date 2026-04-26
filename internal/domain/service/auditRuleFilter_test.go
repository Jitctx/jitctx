package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
)

// ruleSet is a test helper that builds a slice of AuditRule from a list of IDs.
func ruleSet(ids ...string) []model.AuditRule {
	rules := make([]model.AuditRule, 0, len(ids))
	for _, id := range ids {
		rules = append(rules, model.AuditRule{ID: id})
	}
	return rules
}

func TestAuditRuleFilter_Filter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		rules       []model.AuditRule
		disabled    []string
		wantKeptIDs []string
		wantUnknown []string
	}{
		{
			name:        "nil-rules-nil-disabled",
			rules:       nil,
			disabled:    nil,
			wantKeptIDs: []string{},
			wantUnknown: []string{},
		},
		{
			name:        "non-nil-rules-nil-disabled",
			rules:       ruleSet("A", "B", "C"),
			disabled:    nil,
			wantKeptIDs: []string{"A", "B", "C"},
			wantUnknown: []string{},
		},
		{
			name:        "non-nil-rules-empty-disabled",
			rules:       ruleSet("A", "B", "C"),
			disabled:    []string{},
			wantKeptIDs: []string{"A", "B", "C"},
			wantUnknown: []string{},
		},
		{
			name:        "single-disabled-matches",
			rules:       ruleSet("A", "B", "C"),
			disabled:    []string{"B"},
			wantKeptIDs: []string{"A", "C"},
			wantUnknown: []string{},
		},
		{
			name:        "single-disabled-does-not-match",
			rules:       ruleSet("A", "B", "C"),
			disabled:    []string{"Z"},
			wantKeptIDs: []string{"A", "B", "C"},
			wantUnknown: []string{"Z"},
		},
		{
			name:        "multiple-disabled-all-match",
			rules:       ruleSet("A", "B", "C"),
			disabled:    []string{"B", "A"},
			wantKeptIDs: []string{"C"},
			wantUnknown: []string{},
		},
		{
			name:        "mix-matching-and-non-matching-unknown-sorted-asc",
			rules:       ruleSet("A", "B", "C"),
			disabled:    []string{"Z", "A", "Y"},
			wantKeptIDs: []string{"B", "C"},
			wantUnknown: []string{"Y", "Z"},
		},
		{
			name:        "duplicate-disabled-ids-deduped-no-unknown",
			rules:       ruleSet("A", "B", "C"),
			disabled:    []string{"A", "A", "A"},
			wantKeptIDs: []string{"B", "C"},
			wantUnknown: []string{},
		},
		{
			name:        "empty-string-ids-silently-dropped",
			rules:       ruleSet("A", "B", "C"),
			disabled:    []string{""},
			wantKeptIDs: []string{"A", "B", "C"},
			wantUnknown: []string{},
		},
	}

	f := service.NewAuditRuleFilter()

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			kept, unknown := f.Filter(tc.rules, tc.disabled)

			// Extract IDs from kept rules for readability.
			keptIDs := make([]string, 0, len(kept))
			for _, r := range kept {
				keptIDs = append(keptIDs, r.ID)
			}

			require.Equal(t, tc.wantKeptIDs, keptIDs)
			require.Equal(t, tc.wantUnknown, unknown)

			// Both return values must always be non-nil slices.
			require.NotNil(t, kept)
			require.NotNil(t, unknown)
		})
	}
}

func TestAuditRuleFilter_Filter_UnknownSortedAscDeterminism(t *testing.T) {
	t.Parallel()

	// Verify that regardless of the iteration order of the internal map,
	// unknown is always sorted ascending (RNF-003).
	f := service.NewAuditRuleFilter()

	rules := ruleSet("A")
	disabled := []string{"Z", "M", "B", "Y", "C"}

	for i := 0; i < 20; i++ {
		_, unknown := f.Filter(rules, disabled)
		require.Equal(t, []string{"B", "C", "M", "Y", "Z"}, unknown,
			"unknown must be sorted ASC on every invocation (run %d)", i+1)
	}
}
