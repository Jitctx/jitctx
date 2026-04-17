package queryuc

import (
	"context"
	"fmt"
	"log/slog"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/port/manifest"
	"github.com/jitctx/jitctx/internal/domain/port/token"
	"github.com/jitctx/jitctx/internal/domain/service"
	queryvo "github.com/jitctx/jitctx/internal/domain/vo/query"
)

type Impl struct {
	manifest  manifest.LoadManifestPort
	estimator token.EstimateTokensPort
	logger    *slog.Logger
}

func New(m manifest.LoadManifestPort, e token.EstimateTokensPort, l *slog.Logger) *Impl {
	return &Impl{manifest: m, estimator: e, logger: l}
}

func (u *Impl) Execute(ctx context.Context, input queryvo.QueryContextInput) (queryvo.QueryContextOutput, error) {
	if err := ctx.Err(); err != nil {
		return queryvo.QueryContextOutput{}, err
	}
	state, err := u.manifest.Load(ctx)
	if err != nil {
		return queryvo.QueryContextOutput{}, fmt.Errorf("load manifest: %w", err)
	}
	module, ok := state.FindModule(input.Module)
	if !ok {
		return queryvo.QueryContextOutput{}, domerr.ErrModuleNotFound
	}
	candidates := service.FilterContexts(state.Contexts, module, input.Tags, input.Types, input.FilePath)
	loaded, trimmed, total := service.TrimToBudget(candidates, input.Budget)
	return queryvo.QueryContextOutput{
		Loaded:      loaded,
		Trimmed:     trimmed,
		TotalTokens: total,
	}, nil
}
