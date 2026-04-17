package service

import (
	"sort"
	"strings"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// SortProjectState sorts all lists in state deterministically per contract §3.2.
// Mutates state in place.
func SortProjectState(s *model.ProjectState) {
	// Sort stack.
	sort.Strings(s.Stack.Languages)
	sort.Strings(s.Stack.Frameworks)

	// Sort modules by id.
	sort.Slice(s.Modules, func(i, j int) bool {
		return s.Modules[i].ID < s.Modules[j].ID
	})

	for i := range s.Modules {
		m := &s.Modules[i]

		// Sort and dedupe tags.
		m.Tags = sortDedupe(m.Tags)

		// Sort contracts by (path, name).
		sort.Slice(m.Contracts, func(a, b int) bool {
			if m.Contracts[a].Path != m.Contracts[b].Path {
				return m.Contracts[a].Path < m.Contracts[b].Path
			}
			return m.Contracts[a].Name < m.Contracts[b].Name
		})

		// Sort methods within each contract.
		for j := range m.Contracts {
			sort.Slice(m.Contracts[j].Methods, func(a, b int) bool {
				return m.Contracts[j].Methods[a].Signature < m.Contracts[j].Methods[b].Signature
			})
		}

		// Sort and dedupe dependencies; remove self-dependency.
		deps := make([]string, 0, len(m.Dependencies))
		seen := make(map[string]bool)
		for _, d := range m.Dependencies {
			if d != m.ID && !seen[d] {
				deps = append(deps, d)
				seen[d] = true
			}
		}
		sort.Strings(deps)
		m.Dependencies = deps
	}

	// Sort contexts by id.
	sort.Slice(s.Contexts, func(i, j int) bool {
		return s.Contexts[i].ID < s.Contexts[j].ID
	})

	for i := range s.Contexts {
		c := &s.Contexts[i]
		c.Tags = sortDedupe(c.Tags)
		sort.Strings(c.AppliesTo)
	}
}

// sortDedupe sorts a string slice and removes duplicates (case-preserving but lowercases first).
func sortDedupe(in []string) []string {
	if len(in) == 0 {
		return in
	}
	lowered := make([]string, len(in))
	for i, s := range in {
		lowered[i] = strings.ToLower(s)
	}
	sort.Strings(lowered)
	out := lowered[:0:len(lowered)]
	seen := make(map[string]bool)
	for _, s := range lowered {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
