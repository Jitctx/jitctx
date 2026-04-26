package profile

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// LoadProfileBundlePort loads a profile from a directory layout
// (profile.yaml + templates/ subdirectory) into an in-memory aggregate.
// Implementations MUST:
//   - return errors wrapping domerr.ErrProfileYamlMissing when the
//     directory exists but profile.yaml is absent (literal Error() text
//     contains "profile.yaml not found");
//   - return *domerr.TemplateMissingError when profile.yaml declares a
//     type whose template file is absent under templates/;
//   - return errors wrapping domerr.ErrProfileInvalid for any malformed
//     YAML or schema violation;
//   - eagerly read every file under <dir>/templates/ into the returned
//     bundle's Templates map (EP04RNF-006 — load-time error surfacing).
type LoadProfileBundlePort interface {
	LoadBundle(ctx context.Context, input profilevo.LoadProfileBundleInput) (*model.ProfileBundle, error)
}
