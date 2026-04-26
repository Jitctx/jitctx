package refactoruc

import (
	"context"

	refactorvo "github.com/jitctx/jitctx/internal/domain/vo/refactor"
)

// UseCase is the domain contract for the refactor marker scan use case.
// Execute extracts all TODO(jitctx) markers from Java sources in the
// project directory, resolves each marker's owning module from the
// manifest (when present), and returns a sorted, grouped result.
type UseCase interface {
	Execute(
		ctx context.Context,
		input refactorvo.ScanRefactorsInput,
	) (refactorvo.ScanRefactorsOutput, error)
}
