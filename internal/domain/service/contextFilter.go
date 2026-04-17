package service

import (
	"strings"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/vo"
)

func FilterContexts(
	contexts []model.Context,
	module *model.Module,
	tags []string,
	types []vo.ArtifactType,
	_ string,
) []model.Context {
	out := make([]model.Context, 0, len(contexts))
	for _, c := range contexts {
		if module != nil && c.Module != "" && c.Module != module.ID {
			continue
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
