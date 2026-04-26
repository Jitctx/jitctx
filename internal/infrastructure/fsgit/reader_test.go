package fsgit_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	gitport "github.com/jitctx/jitctx/internal/domain/port/git"
	"github.com/jitctx/jitctx/internal/infrastructure/fsgit"
)

// Compile-time type-assertion lock (case 8).
// Confirms *fsgit.Reader satisfies FileLastModifiedTimePort directly,
// and that the value returned by LineReader() satisfies LineIntroducedTimePort.
var (
	_ gitport.FileLastModifiedTimePort = (*fsgit.Reader)(nil)
	_ gitport.LineIntroducedTimePort   = (*fsgit.Reader)(nil).LineReader()
)

// nopLogger returns a silent logger for use in tests.
func nopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 10}))
}

// initRepo creates a new git repository in dir, sets a deterministic
// user.email / user.name, and returns the repo root path.
func initRepo(t *testing.T, dir string) {
	t.Helper()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@example.com")
	run(t, dir, "git", "config", "user.name", "Test")
}

// run executes a git command inside dir and fails the test if it errors.
func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "command %v failed:\n%s", args, out)
}

// commitFile writes content to filename (relative to dir), stages it, and
// commits with the supplied RFC 3339 timestamp as both author and committer time.
// Returns the committed time.Time (UTC, seconds precision).
func commitFile(t *testing.T, dir, filename, content, rfc3339, message string) time.Time {
	t.Helper()
	fullPath := filepath.Join(dir, filename)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))

	run(t, dir, "git", "add", filename)

	// Set deterministic commit time via both --date flag and env vars.
	cmd := exec.Command("git", "commit", "--date="+rfc3339, "-m", message)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE="+rfc3339,
		"GIT_COMMITTER_DATE="+rfc3339,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git commit failed:\n%s", out)

	ts, err := time.Parse(time.RFC3339, rfc3339)
	require.NoError(t, err)
	return ts.UTC()
}

// hashRepoFiles walks dir recursively and returns a map[relPath]sha256hex
// for every regular file, including .git/COMMIT_EDITMSG etc. Used by case 7.
func hashRepoFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	result := make(map[string]string)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		sum := sha256.Sum256(data)
		rel, _ := filepath.Rel(dir, path)
		result[rel] = fmt.Sprintf("%x", sum)
		return nil
	})
	require.NoError(t, err)
	return result
}

// TestReader_HappyPath_FileLastModifiedTime verifies that Get returns the
// commit time for a tracked file within a 1-second tolerance (case 1).
func TestReader_HappyPath_FileLastModifiedTime(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)

	const rfc3339 = "2024-03-15T10:00:00Z"
	wantTime := commitFile(t, dir, "Hello.java", "line1\n", rfc3339, "first commit")

	r := fsgit.New(nopLogger())
	got, err := r.Get(context.Background(), dir, "Hello.java")
	require.NoError(t, err)
	require.WithinDuration(t, wantTime, got, time.Second, "commit time mismatch")
}

// TestReader_UntrackedFile verifies that Get returns a non-nil error (NOT
// ErrGitUnavailable) when the file exists in the work tree but was never
// staged / committed (case 2).
func TestReader_UntrackedFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)

	// Write a file but do NOT `git add` / commit it.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Untracked.java"), []byte("x\n"), 0o644))

	r := fsgit.New(nopLogger())
	_, err := r.Get(context.Background(), dir, "Untracked.java")
	require.Error(t, err)
	require.False(t, isErrGitUnavailable(err),
		"expected a non-ErrGitUnavailable error for untracked file, got: %v", err)
}

// TestReader_NoGitDir verifies that Get and LineReader().Get both return
// ErrGitUnavailable when repoRoot has no .git directory (case 3).
func TestReader_NoGitDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir() // no git init

	r := fsgit.New(nopLogger())
	ctx := context.Background()

	_, err := r.Get(ctx, dir, "anything.java")
	require.ErrorIs(t, err, domerr.ErrGitUnavailable)

	_, err = r.LineReader().Get(ctx, dir, "anything.java", 1)
	require.ErrorIs(t, err, domerr.ErrGitUnavailable)
}

// TestReader_BlameLine verifies that LineReader().Get returns the correct
// committer time for a specific line (case 4 — blame line).
//
// Setup:
//   - Commit 1 at T1: file with two identical lines "aaa\nbbb\n".
//   - Commit 2 at T2: replace line 2 with "ccc\n".
//
// Assertion: LineReader().Get(ctx, repo, file, 2) == T2.
func TestReader_BlameLine(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)

	const (
		t1RFC3339 = "2024-01-10T08:00:00Z"
		t2RFC3339 = "2024-06-20T14:30:00Z"
	)

	// Commit 1: add file with two lines.
	commitFile(t, dir, "Blame.java", "aaa\nbbb\n", t1RFC3339, "commit one")

	// Commit 2: update only line 2.
	wantT2 := commitFile(t, dir, "Blame.java", "aaa\nccc\n", t2RFC3339, "commit two")

	r := fsgit.New(nopLogger())
	got, err := r.LineReader().Get(context.Background(), dir, "Blame.java", 2)
	require.NoError(t, err)
	require.WithinDuration(t, wantT2, got, time.Second, "line 2 should carry commit-two time")
}

// TestReader_ModifiedLine verifies that LineReader().Get returns T2 for line 2
// after two separate commits, confirming the "last-touched" semantics (case 5).
//
// This is the same setup as case 4 but asserts the opposite line (line 1)
// still carries T1, validating per-line granularity.
func TestReader_ModifiedLine_PerLineGranularity(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)

	const (
		t1RFC3339 = "2024-02-01T09:00:00Z"
		t2RFC3339 = "2024-09-01T17:00:00Z"
	)

	wantT1 := commitFile(t, dir, "Lines.java", "first\nsecond\n", t1RFC3339, "initial")
	wantT2 := commitFile(t, dir, "Lines.java", "first\nchanged\n", t2RFC3339, "update line 2")

	r := fsgit.New(nopLogger())
	ctx := context.Background()

	gotLine1, err := r.LineReader().Get(ctx, dir, "Lines.java", 1)
	require.NoError(t, err)
	require.WithinDuration(t, wantT1, gotLine1, time.Second,
		"line 1 was not modified; should carry T1")

	gotLine2, err := r.LineReader().Get(ctx, dir, "Lines.java", 2)
	require.NoError(t, err)
	require.WithinDuration(t, wantT2, gotLine2, time.Second,
		"line 2 was modified at T2; should carry T2")
}

// TestReader_UncommittedLine verifies that LineReader().Get returns a non-nil
// error when the queried line has been modified in the working tree but not
// committed (case 6).
func TestReader_UncommittedLine(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)

	// Commit the file.
	commitFile(t, dir, "Work.java", "stable\nstable\n", "2024-04-01T12:00:00Z", "initial commit")

	// Modify line 2 without committing.
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "Work.java"),
		[]byte("stable\nmodified-not-committed\n"),
		0o644,
	))

	r := fsgit.New(nopLogger())
	_, err := r.LineReader().Get(context.Background(), dir, "Work.java", 2)
	require.Error(t, err, "uncommitted line should return an error")
}

// TestReader_ReadOnly_SHA256 verifies that a sequence of Get and LineReader().Get
// calls does not mutate any file in the repository (case 7 — RNF-002).
func TestReader_ReadOnly_SHA256(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)

	commitFile(t, dir, "Immutable.java", "alpha\nbeta\n", "2024-05-05T00:00:00Z", "seed")

	// Snapshot hashes before any fsgit calls.
	before := hashRepoFiles(t, dir)

	r := fsgit.New(nopLogger())
	ctx := context.Background()

	_, _ = r.Get(ctx, dir, "Immutable.java")
	_, _ = r.LineReader().Get(ctx, dir, "Immutable.java", 1)
	_, _ = r.LineReader().Get(ctx, dir, "Immutable.java", 2)

	// Snapshot hashes after fsgit calls.
	after := hashRepoFiles(t, dir)

	require.Equal(t, before, after, "fsgit must not mutate any repository file (RNF-002)")
}

// isErrGitUnavailable is a helper that returns true iff err wraps
// domerr.ErrGitUnavailable, without using require so the caller can assert
// the inverse.
func isErrGitUnavailable(err error) bool {
	if err == nil {
		return false
	}
	// errors.Is traversal.
	target := domerr.ErrGitUnavailable
	return err == target || func() bool {
		type unwrapper interface{ Unwrap() error }
		e := err
		for e != nil {
			if e == target {
				return true
			}
			u, ok := e.(unwrapper)
			if !ok {
				break
			}
			e = u.Unwrap()
		}
		return false
	}()
}
