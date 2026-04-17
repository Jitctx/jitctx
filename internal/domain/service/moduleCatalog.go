package service

import (
	"sort"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// ModuleIDsSorted returns the ids of every module in state, alphabetically
// sorted. Pure function, no allocation beyond the return slice.
func ModuleIDsSorted(state *model.ProjectState) []string {
	ids := make([]string, 0, len(state.Modules))
	for i := range state.Modules {
		ids = append(ids, state.Modules[i].ID)
	}
	sort.Strings(ids)
	return ids
}
