package spec

import (
	"context"

	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

// WriteProductionFilesPort writes a batch of rendered production files
// atomically per EP02RF-009. Implementations MUST execute three phases:
//
//  1. Conflict check — stat every Path; if ANY exists, return
//     *domerr.ScaffoldConflictError BEFORE any disk write.
//  2. Temp write — write each file to a tmp path in the same parent
//     directory (so rename is intra-volume); on any failure, remove all
//     created temps and return a wrapped ErrSpecWriteFailed.
//  3. Rename — os.Rename each temp → final. On rename failure, best-effort
//     remove already-renamed targets AND remaining temps; return wrapped
//     ErrSpecWriteFailed.
//
// On success, returns the list of final paths actually written, sorted
// alphabetically.
type WriteProductionFilesPort interface {
	WriteAll(ctx context.Context, files []scaffoldvo.ProductionFile) (written []string, err error)
}
