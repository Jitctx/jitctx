package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
	"github.com/jitctx/jitctx/internal/domain/vo"
)

// TestFilterContexts_ModuleMatching covers the seven applies_to / module match
// branches specified in plan §7.2 T6-G2 (cases a-g).
func TestFilterContexts_ModuleMatching(t *testing.T) {
	t.Parallel()

	userModule := &model.Module{ID: "user-management", Tags: []string{}}
	javaModule := &model.Module{ID: "user-management", Tags: []string{"java"}}
	mixedCaseModule := &model.Module{ID: "user-management", Tags: []string{"Java"}}

	cases := []struct {
		name     string
		context  model.Context
		module   *model.Module
		wantKept bool
	}{
		{
			// a: context.module == module.ID, no applies_to — must keep
			name: "a-explicit-module-match-keeps",
			context: model.Context{
				ID:        "user-scenarios",
				Module:    "user-management",
				AppliesTo: nil,
			},
			module:   userModule,
			wantKept: true,
		},
		{
			// b: context.module set to different module — must drop
			name: "b-different-module-drops",
			context: model.Context{
				ID:        "billing-scenarios",
				Module:    "billing",
				AppliesTo: nil,
			},
			module:   userModule,
			wantKept: false,
		},
		{
			// c: module="", applies_to=[java], module.Tags=[java] — must keep
			name: "c-applies-to-overlaps-module-tags-keeps",
			context: model.Context{
				ID:        "java-conventions",
				Module:    "",
				AppliesTo: []string{"java"},
			},
			module:   javaModule,
			wantKept: true,
		},
		{
			// d: module="", applies_to=[python], module.Tags=[java] — must drop
			name: "d-applies-to-no-overlap-drops",
			context: model.Context{
				ID:        "python-conventions",
				Module:    "",
				AppliesTo: []string{"python"},
			},
			module:   javaModule,
			wantKept: false,
		},
		{
			// e: module="", applies_to=[], module.Tags=[java] — empty wildcard not a match, must drop
			name: "e-empty-applies-to-drops",
			context: model.Context{
				ID:        "generic-context",
				Module:    "",
				AppliesTo: []string{},
			},
			module:   javaModule,
			wantKept: false,
		},
		{
			// f: module="user-management", applies_to=[java], module.Tags=[] — module wins, must keep
			name: "f-explicit-module-wins-over-missing-tag-overlap",
			context: model.Context{
				ID:        "user-management-guide",
				Module:    "user-management",
				AppliesTo: []string{"java"},
			},
			module:   userModule,
			wantKept: true,
		},
		{
			// g: applies_to=[java] (lowercase) vs module.Tags=[Java] (uppercase) — case-insensitive, must keep
			name: "g-applies-to-case-insensitive-keeps",
			context: model.Context{
				ID:        "java-guide",
				Module:    "",
				AppliesTo: []string{"java"},
			},
			module:   mixedCaseModule,
			wantKept: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := service.FilterContexts(
				[]model.Context{tc.context},
				tc.module,
				nil, // tags filter — not under test here
				nil, // types filter — not under test here
				"",  // filePath — unused
			)

			if tc.wantKept {
				require.Len(t, got, 1, "expected context to be kept")
				require.Equal(t, tc.context.ID, got[0].ID)
			} else {
				require.Empty(t, got, "expected context to be dropped")
			}
		})
	}
}

// TestFilterContexts_TagFilter proves that tag filtering still works after the
// applies_to overlap changes (regression guard).
func TestFilterContexts_TagFilter(t *testing.T) {
	t.Parallel()

	contexts := []model.Context{
		{ID: "ctx-java", Module: "user-management", Tags: []string{"java", "backend"}},
		{ID: "ctx-frontend", Module: "user-management", Tags: []string{"react", "frontend"}},
		{ID: "ctx-all", Module: "user-management", Tags: []string{"java", "react"}},
	}
	module := &model.Module{ID: "user-management", Tags: []string{}}

	cases := []struct {
		name    string
		tags    []string
		wantIDs []string
	}{
		{
			name:    "single-tag-java-matches-two",
			tags:    []string{"java"},
			wantIDs: []string{"ctx-java", "ctx-all"},
		},
		{
			name:    "single-tag-frontend-matches-one",
			tags:    []string{"frontend"},
			wantIDs: []string{"ctx-frontend"},
		},
		{
			name:    "tag-not-present-drops-all",
			tags:    []string{"python"},
			wantIDs: []string{},
		},
		{
			name:    "no-tag-filter-keeps-all",
			tags:    nil,
			wantIDs: []string{"ctx-java", "ctx-frontend", "ctx-all"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := service.FilterContexts(contexts, module, tc.tags, nil, "")

			gotIDs := make([]string, len(got))
			for i, c := range got {
				gotIDs[i] = c.ID
			}
			require.ElementsMatch(t, tc.wantIDs, gotIDs)
		})
	}
}

// TestFilterContexts_TypeFilter proves that artifact type filtering still works
// after the applies_to overlap changes (regression guard).
func TestFilterContexts_TypeFilter(t *testing.T) {
	t.Parallel()

	contexts := []model.Context{
		{ID: "ctx-guidelines", Module: "user-management", Type: vo.ArtifactGuidelines},
		{ID: "ctx-scenarios", Module: "user-management", Type: vo.ArtifactScenarios},
		{ID: "ctx-contracts", Module: "user-management", Type: vo.ArtifactContracts},
	}
	module := &model.Module{ID: "user-management", Tags: []string{}}

	cases := []struct {
		name    string
		types   []vo.ArtifactType
		wantIDs []string
	}{
		{
			name:    "filter-by-guidelines-type",
			types:   []vo.ArtifactType{vo.ArtifactGuidelines},
			wantIDs: []string{"ctx-guidelines"},
		},
		{
			name:    "filter-by-scenarios-type",
			types:   []vo.ArtifactType{vo.ArtifactScenarios},
			wantIDs: []string{"ctx-scenarios"},
		},
		{
			name:    "filter-by-multiple-types",
			types:   []vo.ArtifactType{vo.ArtifactGuidelines, vo.ArtifactContracts},
			wantIDs: []string{"ctx-guidelines", "ctx-contracts"},
		},
		{
			name:    "no-type-filter-keeps-all",
			types:   nil,
			wantIDs: []string{"ctx-guidelines", "ctx-scenarios", "ctx-contracts"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := service.FilterContexts(contexts, module, nil, tc.types, "")

			gotIDs := make([]string, len(got))
			for i, c := range got {
				gotIDs[i] = c.ID
			}
			require.ElementsMatch(t, tc.wantIDs, gotIDs)
		})
	}
}

// TestFilterContexts_NoModuleFilter proves that passing nil module does not
// apply any module-based filtering (existing behaviour regression guard).
func TestFilterContexts_NoModuleFilter(t *testing.T) {
	t.Parallel()

	contexts := []model.Context{
		{ID: "ctx-a", Module: "billing"},
		{ID: "ctx-b", Module: "user-management"},
		{ID: "ctx-c", Module: ""},
	}

	got := service.FilterContexts(contexts, nil, nil, nil, "")

	require.Len(t, got, 3, "with nil module all contexts should pass through")
}

// TestFilterContexts_TagAndTypeCombined proves that tag + type filters compose
// with AND semantics — both must match.
func TestFilterContexts_TagAndTypeCombined(t *testing.T) {
	t.Parallel()

	module := &model.Module{ID: "user-management", Tags: []string{}}
	contexts := []model.Context{
		{ID: "ctx-java-guidelines", Module: "user-management", Type: vo.ArtifactGuidelines, Tags: []string{"java"}},
		{ID: "ctx-java-scenarios", Module: "user-management", Type: vo.ArtifactScenarios, Tags: []string{"java"}},
		{ID: "ctx-react-guidelines", Module: "user-management", Type: vo.ArtifactGuidelines, Tags: []string{"react"}},
	}

	got := service.FilterContexts(
		contexts,
		module,
		[]string{"java"},
		[]vo.ArtifactType{vo.ArtifactGuidelines},
		"",
	)

	require.Len(t, got, 1)
	require.Equal(t, "ctx-java-guidelines", got[0].ID)
}
