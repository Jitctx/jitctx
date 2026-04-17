package manifest

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
)

type LoadManifestPort interface {
	Load(ctx context.Context) (*model.ProjectState, error)
}
