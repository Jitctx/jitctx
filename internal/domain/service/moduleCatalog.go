package service

import (
	"sort"
	"strings"

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

// ResolveModuleByPath returns the ModuleID of the longest-prefix matching
// module for path, or "<unmoduled>" when no module matches. Path
// separators are normalised to "/" for comparison. Pure function; safe to
// call concurrently.
//
// NOTE FOR REVIEWERS: this helper is added but the audit use case is NOT
// migrated in this plan. EP03US-002 keeps its inline closure. A followup
// can inline-replace it behind a separate PR.
func ResolveModuleByPath(modules []model.Module, path string) string {
	normPath := strings.ReplaceAll(path, "\\", "/")

	bestID := ""
	bestLen := -1

	for _, m := range modules {
		normModule := strings.ReplaceAll(m.Path, "\\", "/")
		// Ensure prefix comparison is directory-boundary-safe.
		prefix := normModule
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		if strings.HasPrefix(normPath+"/", prefix) {
			if len(normModule) > bestLen {
				bestLen = len(normModule)
				bestID = m.ID
			}
		}
	}

	if bestID == "" {
		return "<unmoduled>"
	}
	return bestID
}
