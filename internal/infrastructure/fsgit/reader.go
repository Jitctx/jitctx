package fsgit

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	gitport "github.com/jitctx/jitctx/internal/domain/port/git"
)

// Reader is a read-only adapter that satisfies the domain git ports via
// subprocess invocations of the `git` binary. RNF-002: no mutating
// subcommand is ever invoked. All exec calls honour the provided context.
type Reader struct {
	logger *slog.Logger
}

// New returns a new Reader. The logger is used for debug-level diagnostics;
// it must not be nil.
func New(logger *slog.Logger) *Reader {
	return &Reader{logger: logger}
}

// Get returns the commit time of the most recent commit that touched
// filePath in the git history rooted at repoRoot. It satisfies
// gitport.FileLastModifiedTimePort.
//
//   - Returns (time.Time{}, domerr.ErrGitUnavailable) when git is absent or
//     repoRoot is not inside a work tree.
//   - Returns (time.Time{}, non-nil-err) when the file is untracked or the
//     git invocation fails. The caller treats per-marker errors as non-fatal.
func (r *Reader) Get(ctx context.Context, repoRoot, filePath string) (time.Time, error) {
	if !IsGitAvailable(ctx, repoRoot) {
		return time.Time{}, domerr.ErrGitUnavailable
	}

	// git log -1 --format=%ct -- <filePath>  prints a single Unix epoch second.
	cmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "log", "-1", "--format=%ct", "--", filePath)
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("git log: %w", err)
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		// Empty output means the file is not tracked by git.
		return time.Time{}, fmt.Errorf("file not tracked: %s", filePath)
	}
	sec, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("git log: parse commit time %q: %w", trimmed, err)
	}
	return time.Unix(sec, 0).UTC(), nil
}

// GetLine returns the committer timestamp of the commit that most recently
// modified the given 1-based line in filePath via git blame --line-porcelain.
//
//   - Returns (time.Time{}, domerr.ErrGitUnavailable) when git is absent or
//     repoRoot is not inside a work tree.
//   - Returns (time.Time{}, fmt.Errorf("line uncommitted")) when blame reports
//     the line has not been committed yet (committer-time == 0 with the
//     "Not Committed Yet" author marker).
//   - Returns (time.Time{}, non-nil-err) for all other failures.
//
// GetLine is intentionally NOT named Get to avoid a method-name collision with
// the FileLastModifiedTimePort Get on the same *Reader receiver. Use
// LineReader() to obtain a gitport.LineIntroducedTimePort adapter.
func (r *Reader) GetLine(ctx context.Context, repoRoot, filePath string, line int) (time.Time, error) {
	if !IsGitAvailable(ctx, repoRoot) {
		return time.Time{}, domerr.ErrGitUnavailable
	}

	lineRange := fmt.Sprintf("%d,%d", line, line)
	cmd := exec.CommandContext(ctx, "git", "-C", repoRoot,
		"blame", "--line-porcelain", "--follow",
		"-L", lineRange, "--", filePath)
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("git blame: %w", err)
	}

	// Parse committer-time and author fields from the porcelain output.
	// The porcelain block for a single line looks like:
	//   <sha> <orig-line> <final-line> <count>
	//   author <name>
	//   ...
	//   committer-time <unix-seconds>
	//   ...
	var committerTime int64 = -1
	isNotCommitted := false

	for _, raw := range bytes.Split(out, []byte("\n")) {
		line := strings.TrimSpace(string(raw))
		switch {
		case strings.HasPrefix(line, "committer-time "):
			val := strings.TrimPrefix(line, "committer-time ")
			sec, parseErr := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
			if parseErr != nil {
				return time.Time{}, fmt.Errorf("git blame: parse committer-time %q: %w", val, parseErr)
			}
			committerTime = sec
		case strings.HasPrefix(line, "author "):
			author := strings.TrimPrefix(line, "author ")
			if strings.TrimSpace(author) == "Not Committed Yet" {
				isNotCommitted = true
			}
		}
	}

	if committerTime < 0 {
		return time.Time{}, fmt.Errorf("git blame: committer-time not found in porcelain output")
	}

	// committer-time == 0 combined with "Not Committed Yet" author indicates
	// an uncommitted working-tree change.
	if isNotCommitted || committerTime == 0 {
		return time.Time{}, fmt.Errorf("line uncommitted")
	}

	return time.Unix(committerTime, 0).UTC(), nil
}

// lineReader is a trivial adapter that wraps *Reader.GetLine and presents
// it as gitport.LineIntroducedTimePort's Get method signature, resolving
// the method-name collision on *Reader (plan §4.1 / Q7).
type lineReader struct {
	r *Reader
}

// Get delegates to r.GetLine, satisfying gitport.LineIntroducedTimePort.
func (l lineReader) Get(ctx context.Context, repoRoot, filePath string, line int) (time.Time, error) {
	return l.r.GetLine(ctx, repoRoot, filePath, line)
}

// LineReader returns a gitport.LineIntroducedTimePort backed by this Reader.
// Wire injects gitReader.LineReader() for the LineIntroducedTimePort
// dependency while passing gitReader directly for FileLastModifiedTimePort.
func (r *Reader) LineReader() gitport.LineIntroducedTimePort {
	return lineReader{r: r}
}
