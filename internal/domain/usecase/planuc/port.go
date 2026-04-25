package planuc

import (
	"context"

	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
)

// UseCase computes the parallel execution layers for a feature spec.
// Orchestrates: spec resolution (FindSpecFilePort) → markdown parse
// (ParseSpecPort) → external-reference detection → topological sort
// (DependencyLayerer) → target-path mapping (ContractPathMapper).
type UseCase interface {
	Execute(ctx context.Context, input planvo.LayersInput) (planvo.LayersOutput, error)
}
