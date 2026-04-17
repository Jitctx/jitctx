package service

import (
	"sort"
	"strings"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// ResolveDependencies analyzes imports in summaries belonging to module
// and returns a sorted list of other known module IDs that module depends on.
func ResolveDependencies(
	summaries []model.JavaFileSummary,
	module model.Module,
	all []model.Module,
) []string {
	// Build a set of known module IDs for fast lookup.
	knownModules := make(map[string]bool, len(all))
	for _, m := range all {
		knownModules[m.ID] = true
	}

	deps := make(map[string]bool)
	modulePath := strings.TrimSuffix(module.Path, "/") + "/"

	for _, summary := range summaries {
		summaryPath := strings.TrimSuffix(summary.Path, "/")
		// Check if this file belongs to our module.
		if !strings.HasPrefix(summaryPath, modulePath) && summaryPath != strings.TrimSuffix(modulePath, "/") {
			continue
		}

		for _, imp := range summary.Imports {
			depID := extractModuleIDFromImport(imp)
			if depID == "" || depID == module.ID {
				continue
			}
			if knownModules[depID] {
				deps[depID] = true
			}
		}
	}

	result := make([]string, 0, len(deps))
	for id := range deps {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

// extractModuleIDFromImport extracts a module ID from a Java import statement.
// Import format: com.app.<module>.<rest>
// We use groupDepth=2, so segments[2] is the module segment.
func extractModuleIDFromImport(imp string) string {
	parts := strings.Split(imp, ".")
	if len(parts) <= groupDepth {
		return ""
	}
	seg := parts[groupDepth]
	return strings.ReplaceAll(strings.ToLower(seg), "_", "-")
}
