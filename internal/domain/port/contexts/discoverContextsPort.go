package contexts

import (
	"context"
	"io/fs"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// DiscoverContextsPort walks .jitctx/{guidelines,requirements,scenarios,contracts}/**/*.md
// and returns model.Context entries WITHOUT token_estimate populated.
type DiscoverContextsPort interface {
	// DiscoverContexts walks .jitctx/{guidelines,requirements,scenarios}/**/*.md
	// inside fsys and returns model.Context entries WITHOUT the token_estimate
	// populated — estimation is the use case's responsibility via EstimateTokensPort.
	DiscoverContexts(ctx context.Context, fsys fs.FS) ([]model.Context, error)
}
