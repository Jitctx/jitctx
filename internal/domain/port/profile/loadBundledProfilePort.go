package profile

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// LoadBundledProfilePort loads a profile by name from the binary embed,
// without any filesystem access. Returns errors wrapping
// domerr.ErrBundledProfileNotFound when the named profile is not
// embedded; otherwise returns a fully-populated ProfileBundle whose
// Profile.Source is ProfileSourceBundled.
type LoadBundledProfilePort interface {
	LoadBundled(ctx context.Context, name string) (*model.ProfileBundle, error)
}
