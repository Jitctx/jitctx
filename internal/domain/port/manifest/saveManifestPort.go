package manifest

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
)

type SaveManifestPort interface {
	Save(ctx context.Context, state *model.ProjectState) error
}
