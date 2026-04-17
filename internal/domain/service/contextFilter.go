package service

import (
	"strings"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/vo"
)

// FilterContexts returns the subset of contexts that match the given criteria.
//
// Module matching semantics (EP01RF-007): when module != nil, a context is
// included if ANY of the following is true:
//   - c.Module == module.ID (explicit module assignment wins)
//   - overlap(c.AppliesTo, module.Tags) is true (case-insensitive)
//
// When c.Module is set but does not equal module.ID and no AppliesTo overlap
// exists, the context is filtered out. When c.Module == "" AND c.AppliesTo is
// empty, the context is excluded from the module scope (empty-wildcard is not a
// match). When c.Module == "" AND there is any AppliesTo element, overlap is
// required.
func FilterContexts(
	contexts []model.Context,
	module *model.Module,
	tags []string,
	types []vo.ArtifactType,
	_ string,
) []model.Context {
	out := make([]model.Context, 0, len(contexts))
	for _, c := range contexts {
		if module != nil {
			if !matchesModule(c, module) {
				continue
			}
		}
		if len(types) > 0 && !hasType(types, c.Type) {
			continue
		}
		if len(tags) > 0 && !hasAnyTag(c.Tags, tags) {
			continue
		}
		out = append(out, c)
	}
	return out
}

// matchesModule returns true when context c is in scope for the given module.
func matchesModule(c model.Context, module *model.Module) bool {
	// Explicit module assignment always wins.
	if c.Module == module.ID {
		return true
	}
	// If the context is pinned to a different module, exclude it.
	if c.Module != "" {
		return false
	}
	// c.Module == "": empty AppliesTo is not a wildcard — exclude.
	if len(c.AppliesTo) == 0 {
		return false
	}
	// AppliesTo overlap required (case-insensitive).
	return overlap(c.AppliesTo, module.Tags)
}

// overlap returns true when at least one element from a appears in b,
// compared case-insensitively.
func overlap(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if strings.EqualFold(x, y) {
				return true
			}
		}
	}
	return false
}

func hasType(types []vo.ArtifactType, t vo.ArtifactType) bool {
	for _, x := range types {
		if x == t {
			return true
		}
	}
	return false
}

func hasAnyTag(have, want []string) bool {
	for _, w := range want {
		for _, h := range have {
			if strings.EqualFold(h, w) {
				return true
			}
		}
	}
	return false
}
