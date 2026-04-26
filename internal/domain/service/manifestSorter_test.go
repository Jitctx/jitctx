package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
	"github.com/jitctx/jitctx/internal/domain/vo"
)

func TestSortProjectState_Deterministic(t *testing.T) {
	t.Parallel()

	state := &model.ProjectState{
		GeneratedAt: time.Now(),
		Stack: model.Stack{
			Languages:  []string{"java", "go"},
			Frameworks: []string{"spring-boot", "cobra"},
		},
		Modules: []model.Module{
			{
				ID:           "z-module",
				Path:         "src/main/java/com/app/z_module",
				Tags:         []string{"B", "A", "A"},
				Dependencies: []string{"z-module", "b-module", "a-module"},
				Contracts: []model.Contract{
					{Name: "B", Types: []string{string(model.ContractInputPort)}, Path: "b.java"},
					{Name: "A", Types: []string{string(model.ContractInputPort)}, Path: "a.java"},
				},
			},
			{
				ID:   "a-module",
				Path: "src/main/java/com/app/a_module",
				Tags: nil,
			},
		},
		Contexts: []model.Context{
			{ID: "z-ctx", Type: vo.ArtifactGuidelines, Tags: []string{"b", "a"}},
			{ID: "a-ctx", Type: vo.ArtifactGuidelines, Tags: nil},
		},
	}

	service.SortProjectState(state)

	// Modules sorted by id.
	require.Equal(t, "a-module", state.Modules[0].ID)
	require.Equal(t, "z-module", state.Modules[1].ID)

	// Tags sorted and deduplicated (lowercased).
	require.Equal(t, []string{"a", "b"}, state.Modules[1].Tags)

	// Dependencies sorted, deduped, self-dep removed.
	require.Equal(t, []string{"a-module", "b-module"}, state.Modules[1].Dependencies)

	// Contracts sorted by path then name.
	require.Equal(t, "A", state.Modules[1].Contracts[0].Name)
	require.Equal(t, "B", state.Modules[1].Contracts[1].Name)

	// Stack sorted.
	require.Equal(t, []string{"go", "java"}, state.Stack.Languages)

	// Contexts sorted by id.
	require.Equal(t, "a-ctx", state.Contexts[0].ID)
	require.Equal(t, "z-ctx", state.Contexts[1].ID)

	// Tags of empty slice stays empty.
	require.Empty(t, state.Modules[0].Tags)
}
