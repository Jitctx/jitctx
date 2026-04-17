package planuc

import (
	"context"

	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
)

type UseCase interface {
	Execute(ctx context.Context, input planvo.PlanModuleInput) (planvo.PlanModuleOutput, error)
}
