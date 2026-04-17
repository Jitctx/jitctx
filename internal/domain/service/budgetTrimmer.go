package service

import (
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/vo"
	queryvo "github.com/jitctx/jitctx/internal/domain/vo/query"
)

func TrimToBudget(
	contexts []model.Context,
	budget vo.TokenBudget,
) (loaded, trimmed []queryvo.LoadedContext, total int) {
	loaded = make([]queryvo.LoadedContext, 0, len(contexts))
	trimmed = make([]queryvo.LoadedContext, 0)
	for _, c := range contexts {
		lc := queryvo.LoadedContext{
			ID:            c.ID,
			Type:          c.Type,
			Path:          c.Path,
			Tags:          c.Tags,
			TokenEstimate: c.TokenEstimate,
		}
		if !budget.Unlimited() && total+c.TokenEstimate > budget.Max() {
			trimmed = append(trimmed, lc)
			continue
		}
		total += c.TokenEstimate
		loaded = append(loaded, lc)
	}
	return loaded, trimmed, total
}
