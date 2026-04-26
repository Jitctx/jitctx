package git

import (
	"context"
	"time"
)

// LineIntroducedTimePort returns the commit time of the commit that
// most recently modified the given line in the given file (the line's
// "blame" timestamp). Per EP03RF-009 and the EP03US-006 brief, this is
// implemented via `git blame --line-porcelain --follow` so that file
// renames are followed.
//
// repoRoot is the absolute path to the working tree root; filePath is
// forward-slash, relative to repoRoot; line is 1-based.
//
// Returns:
//   - (commitTime, nil) when blame succeeds for the given line.
//   - (time.Time{}, ErrGitUnavailable) when the working tree is not a
//     git repository or the `git` binary is missing from PATH.
//   - (time.Time{}, err) for any other error (e.g., file not tracked,
//     line out of range, uncommitted modification). Caller treats
//     per-marker errors as non-fatal and leaves Stale=false for that
//     marker.
type LineIntroducedTimePort interface {
	Get(ctx context.Context, repoRoot, filePath string, line int) (time.Time, error)
}
