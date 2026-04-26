package profile

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// ClassifyDeclarativePort evaluates a profile's declarative type rules
// against a single code-element input and returns the IDs of every type
// that matches, in declared order.
//
// Implementations MUST:
//   - return an empty slice (never nil errors) when no type matches;
//   - never return errors — the engine's only failure mode is "no match",
//     which is a normal result, not an error;
//   - be deterministic — the same (input, types) pair must always yield
//     the same result slice in the same order;
//   - honour ctx.Err() before returning a non-empty result.
type ClassifyDeclarativePort interface {
	ClassifyDeclarative(
		ctx context.Context,
		input profilevo.ClassificationInput,
		types []model.ProfileTypeDeclaration,
	) ([]string, error)
}
