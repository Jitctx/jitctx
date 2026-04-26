package profileinituc

import (
	"context"

	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// UseCase orchestrates "jitctx profile init <name>". Steps:
//  1. Validate Input.Name (non-empty, no path separators).
//  2. Compute target dir: <Input.WorkDir>/<Input.ProfilesDir>/<Input.Name>/.
//  3. Bail with *ProfileTargetExistsError if target dir exists.
//  4. Verify the bundled name exists by calling
//     ListBundledProfilesPort.ListBundled and matching Input.Name;
//     on miss, bail with *UnknownBundledProfileError carrying the
//     sorted list.
//  5. Call ExtractBundledProfilePort.Extract to write the profile
//     verbatim under the target dir.
//  6. Return ProfileInitOutput with the resolved target path and a
//     summary of the files written.
type UseCase interface {
	Execute(ctx context.Context, input profilevo.ProfileInitInput) (profilevo.ProfileInitOutput, error)
}
