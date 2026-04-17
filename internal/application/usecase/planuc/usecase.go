package planuc

import (
	"context"
	"log/slog"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/port/manifest"
	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
)

type Impl struct {
	manifest manifest.LoadManifestPort
	logger   *slog.Logger
}

func New(m manifest.LoadManifestPort, l *slog.Logger) *Impl {
	return &Impl{manifest: m, logger: l}
}

func (u *Impl) Execute(ctx context.Context, _ planvo.PlanModuleInput) (planvo.PlanModuleOutput, error) {
	if err := ctx.Err(); err != nil {
		return planvo.PlanModuleOutput{}, err
	}
	return planvo.PlanModuleOutput{}, domerr.ErrNotImplemented
}
