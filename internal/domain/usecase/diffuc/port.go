package diffuc

import (
	"context"

	diffvo "github.com/jitctx/jitctx/internal/domain/vo/diff"
)

// UseCase is the diff orchestrator.
//
// Pipeline:
//  1. Resolve & read spec (FindSpecFilePort or in.FilePath).
//  2. Parse spec (ParseSpecPort).
//  3. Load manifest (LoadManifestPort).
//  4. Index manifest contracts by Name (cross-module flat index).
//  5. Compute the diff via service.ContractDiffer.
//  6. Layer the CREATE/MODIFY subset via service.DependencyLayerer.
//  7. Sort and assemble DiffPlanOutput.
type UseCase interface {
	Execute(ctx context.Context, input diffvo.DiffPlanInput) (diffvo.DiffPlanOutput, error)
}
