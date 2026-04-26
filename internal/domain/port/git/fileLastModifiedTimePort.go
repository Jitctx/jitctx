// Package git holds ISP ports describing read-only git history queries
// used by the refactor marker scanner to flag stale markers (EP03RF-009).
// One method per interface, one interface per file. Implementations live
// in internal/infrastructure/fsgit.
package git

import (
	"context"
	"time"
)

// FileLastModifiedTimePort returns the commit time of the most recent
// commit that touched the given file in the working tree's git history.
// repoRoot is the absolute path to the project root (the working tree
// root); filePath is forward-slash, relative to repoRoot.
//
// Returns:
//   - (commitTime, nil) when git history is available and the file is
//     tracked.
//   - (time.Time{}, ErrGitUnavailable) when the working tree is not a
//     git repository or the `git` binary is missing from PATH.
//   - (time.Time{}, err) for any other error (e.g., file not tracked,
//     git invocation failure). Caller treats per-marker errors as
//     non-fatal and leaves Stale=false for that marker.
type FileLastModifiedTimePort interface {
	Get(ctx context.Context, repoRoot, filePath string) (time.Time, error)
}
