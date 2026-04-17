package profile

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
)

type LoadProfilePort interface {
	Load(ctx context.Context, name string) (*model.FrameworkProfile, error)
}
