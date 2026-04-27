// Package profilevalidateuc defines the UseCase interface for
// "jitctx profile validate <path>". EP04US-007.
package profilevalidateuc

import (
	"context"

	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// UseCase orchestrates "jitctx profile validate <path>". Steps:
//  1. Verify the path exists and is a directory; otherwise return a
//     wrapped *ProfileValidationError carrying the io error so the
//     translator renders an exit-1 message (per EP04RF-013 exception
//     "Path that doesn't exist causes immediate exit 1").
//  2. Re-read the raw bytes of <path>/profile.yaml and walk them as a
//     generic yaml.Node tree to detect unknown classification field
//     keys (these become non-fatal warnings).
//  3. Call LoadProfileBundlePort.LoadBundle to leverage existing
//     structural checks (missing yaml, missing template, missing type
//     id, unknown language). Capture any returned error as a fatal.
//  4. Run two extra checks the loader is too lenient about today:
//     (a) `dto.Name == ""` → fatal "missing required field: name"
//     (b) duplicate `dto.Types[].ID` → fatal "duplicate type id: <id>"
//  5. Aggregate fatals + warnings into ValidateProfileOutput. When
//     fatals is non-empty, return a *domerr.ProfileValidationError
//     carrying the same lists; otherwise return Output, nil.
type UseCase interface {
	Execute(ctx context.Context, input profilevo.ValidateProfileInput) (profilevo.ValidateProfileOutput, error)
}
