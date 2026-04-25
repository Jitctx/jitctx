package spec

import (
	"context"

	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

// WriteProductionFilesPort writes a batch of rendered scaffold files
// (production + test combined) atomically per EP02RF-009. Implementations
// MUST execute three phases:
//
//  1. Conflict check — stat every Path; if ANY exists, return
//     *domerr.ScaffoldConflictError BEFORE any disk write. The Conflicts
//     slice MUST contain BOTH production and test paths when both clash.
//  2. Temp write — write each file to a tmp path in the same parent
//     directory; on any failure, remove all created temps and return a
//     wrapped ErrSpecWriteFailed.
//  3. Rename — os.Rename each temp → final. On rename failure, best-effort
//     remove already-renamed targets AND remaining temps; return wrapped
//     ErrSpecWriteFailed.
//
// On success, returns the list of final paths actually written, sorted
// alphabetically (RNF-002).
//
// NOTE: The port name retains the historical "Production" suffix to keep
// blast radius small; semantically it is now the unified scaffold writer.
// A future story may rename the file/interface to WriteScaffoldFilesPort.
type WriteProductionFilesPort interface {
	WriteAll(ctx context.Context, files []scaffoldvo.ScaffoldFile) (written []string, err error)
}
