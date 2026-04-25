package spec

import (
	"context"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
)

// ParseSpecPort parses markdown content into a FeatureSpec.
// On success, returns the FeatureSpec and any non-fatal warnings.
// On fatal error, returns a zero FeatureSpec and an error wrapping
// ErrSpecParse (use errors.Is/errors.As to inspect).
type ParseSpecPort interface {
	ParseSpec(ctx context.Context, content string) (model.FeatureSpec, []domerr.SpecParseWarning, error)
}
