package profile

import (
	"context"
	"io/fs"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// DetectProfilePort picks the first profile whose detect rules match the project.
type DetectProfilePort interface {
	// Detect picks the first profile (custom first, then bundled) whose
	// Detect rules match against fsys. Returns ErrNoProfileMatch when no
	// profile matches. The returned profile is fully loaded.
	Detect(ctx context.Context, fsys fs.FS) (*model.FrameworkProfile, error)
}
