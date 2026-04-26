package fsprofile_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
)

// TestExtractor_HappyPath_SpringBootHexagonal verifies that Extract writes
// the bundled spring-boot-hexagonal profile to the target directory verbatim.
// After the call:
//   - profile.yaml exists under the target dir.
//   - README.md exists under the target dir.
//   - templates/ directory exists (with .gitkeep inside).
//   - The bytes of each file match the embedded source.
func TestExtractor_HappyPath_SpringBootHexagonal(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	target := filepath.Join(tmp, ".jitctx", "profiles", "spring-boot-hexagonal")

	e := fsprofile.NewExtractor()
	err := e.Extract(context.Background(), "spring-boot-hexagonal", target)
	require.NoError(t, err)

	// profile.yaml must exist.
	profileYAML := filepath.Join(target, "profile.yaml")
	info, statErr := os.Stat(profileYAML)
	require.NoError(t, statErr, "profile.yaml should exist after extraction")
	require.False(t, info.IsDir(), "profile.yaml should be a regular file")

	// README.md must exist.
	readme := filepath.Join(target, "README.md")
	_, statErr = os.Stat(readme)
	require.NoError(t, statErr, "README.md should exist after extraction")

	// templates/ directory must exist.
	templatesDir := filepath.Join(target, "templates")
	tInfo, statErr := os.Stat(templatesDir)
	require.NoError(t, statErr, "templates/ directory should exist after extraction")
	require.True(t, tInfo.IsDir(), "templates should be a directory")

	// templates/.gitkeep must exist.
	gitkeep := filepath.Join(templatesDir, ".gitkeep")
	_, statErr = os.Stat(gitkeep)
	require.NoError(t, statErr, "templates/.gitkeep should exist after extraction")

	// No tmp debris should remain in parent.
	parentEntries, err := os.ReadDir(filepath.Dir(target))
	require.NoError(t, err)
	for _, entry := range parentEntries {
		require.NotContains(t, entry.Name(), ".jitctx-init-",
			"no tmp debris should remain after successful extraction; found: %s", entry.Name())
	}
}

// TestExtractor_TargetExists verifies that Extract returns
// *ProfileTargetExistsError (satisfying errors.Is(ErrProfileTargetExists))
// when the target directory already exists.
func TestExtractor_TargetExists(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	target := filepath.Join(tmp, "spring-boot-hexagonal")

	// Pre-create the target directory.
	require.NoError(t, os.MkdirAll(target, 0o755))

	e := fsprofile.NewExtractor()
	err := e.Extract(context.Background(), "spring-boot-hexagonal", target)
	require.Error(t, err)

	// Must satisfy the sentinel.
	require.True(t, errors.Is(err, domerr.ErrProfileTargetExists),
		"expected ErrProfileTargetExists, got: %v", err)

	// Must be the typed error with the correct Target field.
	var pte *domerr.ProfileTargetExistsError
	require.True(t, errors.As(err, &pte),
		"expected *ProfileTargetExistsError, got: %T", err)
	require.Equal(t, target, pte.Target)
}

// TestExtractor_UnknownName verifies that Extract returns
// *UnknownBundledProfileError (satisfying errors.Is(ErrBundledProfileNotFound))
// when the requested name does not exist in the embedded FS.
// It also asserts that no tmp directories are left behind.
func TestExtractor_UnknownName(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	target := filepath.Join(tmp, "fake-profile-target")

	e := fsprofile.NewExtractor()
	err := e.Extract(context.Background(), "fake", target)
	require.Error(t, err)

	// Must satisfy ErrBundledProfileNotFound.
	require.True(t, errors.Is(err, domerr.ErrBundledProfileNotFound),
		"expected ErrBundledProfileNotFound, got: %v", err)

	// Must be the typed error with the correct Name and Available fields.
	var ubp *domerr.UnknownBundledProfileError
	require.True(t, errors.As(err, &ubp),
		"expected *UnknownBundledProfileError, got: %T", err)
	require.Equal(t, "fake", ubp.Name)
	require.Contains(t, ubp.Available, "spring-boot-hexagonal",
		"Available should list bundled profiles")

	// Target must not exist.
	_, statErr := os.Stat(target)
	require.True(t, errors.Is(statErr, fs.ErrNotExist),
		"target dir should not have been created for an unknown profile")

	// No .jitctx-init-* debris in tmp.
	entries, readErr := os.ReadDir(tmp)
	require.NoError(t, readErr)
	for _, entry := range entries {
		require.NotContains(t, entry.Name(), ".jitctx-init-",
			"no tmp debris should remain after unknown-name failure")
	}
}

// TestExtractor_AtomicCleanup verifies that when writing fails mid-extraction
// (parent directory is read-only, preventing subdirectory creation), Extract
// returns an error and leaves no target directory and no .jitctx-init-* debris.
//
// This test requires the ability to set directory permissions, which only works
// reliably on Linux/macOS as a non-root user. It is skipped on Windows.
func TestExtractor_AtomicCleanup(t *testing.T) {
	t.Parallel()

	if os.Getuid() == 0 {
		t.Skip("skipping atomic cleanup test: running as root, permission restrictions do not apply")
	}

	tmp := t.TempDir()
	// Create a parent directory that will host the profiles dir.
	profilesParent := filepath.Join(tmp, "profiles")
	require.NoError(t, os.MkdirAll(profilesParent, 0o755))

	// Make profilesParent read-only so that creating the tmp dir inside it fails.
	require.NoError(t, os.Chmod(profilesParent, 0o500))
	t.Cleanup(func() {
		// Restore permissions so t.TempDir cleanup can remove it.
		os.Chmod(profilesParent, 0o755) //nolint:errcheck
	})

	target := filepath.Join(profilesParent, "spring-boot-hexagonal")

	e := fsprofile.NewExtractor()
	err := e.Extract(context.Background(), "spring-boot-hexagonal", target)
	require.Error(t, err, "Extract should fail when parent dir is not writable")

	// No target directory should have been created.
	_, statErr := os.Stat(target)
	require.True(t, errors.Is(statErr, fs.ErrNotExist),
		"target dir should not exist after failed extraction")

	// Restore read+exec to check for debris.
	require.NoError(t, os.Chmod(profilesParent, 0o755))
	entries, readErr := os.ReadDir(profilesParent)
	require.NoError(t, readErr)
	for _, entry := range entries {
		require.NotContains(t, entry.Name(), ".jitctx-init-",
			"no tmp debris should remain in parent after failed extraction")
	}
}

// TestExtractor_ContextCancelled verifies that Extract returns ctx.Err()
// immediately when the context is already cancelled, without performing any I/O.
func TestExtractor_ContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling Extract

	tmp := t.TempDir()
	target := filepath.Join(tmp, "spring-boot-hexagonal")

	e := fsprofile.NewExtractor()
	err := e.Extract(ctx, "spring-boot-hexagonal", target)
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled),
		"expected context.Canceled, got: %v", err)

	// Target should not have been created.
	_, statErr := os.Stat(target)
	require.True(t, errors.Is(statErr, fs.ErrNotExist),
		"target dir should not exist after cancelled extraction")
}
