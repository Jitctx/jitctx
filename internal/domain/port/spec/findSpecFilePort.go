package spec

import "context"

// FindSpecFilePort encapsulates the read-side resolution required by
// EP02RF-008. Implementations MUST honour this priority order:
//
//  1. <baseDir>/jitctx-plans/<feature>.md
//  2. <baseDir>/.jitctx/plans/<feature>.md
//  3. <baseDir>/<plansDir>/<feature>.md  if plansDir != ""
//
// On success, Path is the first matching absolute (or BaseDir-joined)
// path; Content is its bytes; Alts is the sorted list of OTHER paths
// from the priority list that ALSO contained a file (used by the
// caller to log a "found in N locations" warning per EP02US-007).
//
// On no-match, the returned error wraps domerr.ErrSpecFileNotFound and
// embeds a *domerr.SpecFileNotFoundError carrying the searched paths.
type FindSpecFilePort interface {
	Find(ctx context.Context, feature, baseDir, plansDir string) (path string, content []byte, alts []string, err error)
}
