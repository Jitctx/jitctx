package fsprofile_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
)

// resolverDiscardLogger returns a *slog.Logger that discards output.
func resolverDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}

// makeUserProfileDir writes a minimal valid profile.yaml under
// <baseDir>/<profileName>/profile.yaml so the BundleLoader can load it.
func makeUserProfileDir(t *testing.T, baseDir, profileName string) {
	t.Helper()
	dir := filepath.Join(baseDir, profileName)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	content := "name: " + profileName + "\nlanguage: java\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "profile.yaml"), []byte(content), 0o644))
}

// newTestResolver constructs a Resolver backed by real adapters and a
// discard logger.
func newTestResolver(t *testing.T) *fsprofile.Resolver {
	t.Helper()
	loader := fsprofile.NewBundleLoader(resolverDiscardLogger())
	bundled := fsprofile.NewBundled()
	return fsprofile.NewResolver(loader, bundled, resolverDiscardLogger())
}

// TestResolver_ExplicitName_UserDirWins verifies that when a user-dir profile
// with the requested name exists, it is returned with Source == ProfileSourceCustom.
func TestResolver_ExplicitName_UserDirWins(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	profilesDir := ".jitctx/profiles"
	makeUserProfileDir(t, filepath.Join(tmp, profilesDir), "spring-boot-hexagonal")

	r := newTestResolver(t)
	bundle, err := r.Resolve(context.Background(), profilevo.ResolveProfileInput{
		Name:        "spring-boot-hexagonal",
		WorkDir:     tmp,
		ProfilesDir: profilesDir,
	})

	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.Equal(t, model.ProfileSourceCustom, bundle.Profile.Source)
	require.NotEmpty(t, bundle.Dir)
}

// TestResolver_ExplicitName_BundledFallback verifies that when no user-dir
// exists for the requested name, the bundled embed is returned with
// Source == ProfileSourceBundled.
func TestResolver_ExplicitName_BundledFallback(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir() // empty — no user-dir profiles

	r := newTestResolver(t)
	bundle, err := r.Resolve(context.Background(), profilevo.ResolveProfileInput{
		Name:        "spring-boot-hexagonal",
		WorkDir:     tmp,
		ProfilesDir: ".jitctx/profiles",
	})

	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.Equal(t, model.ProfileSourceBundled, bundle.Profile.Source)
	require.Empty(t, bundle.Dir)
}

// TestResolver_ExplicitName_NotFound verifies that requesting a non-existent
// profile name returns *UnknownBundledProfileError listing available profiles.
func TestResolver_ExplicitName_NotFound(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	r := newTestResolver(t)
	_, err := r.Resolve(context.Background(), profilevo.ResolveProfileInput{
		Name:        "never-exists",
		WorkDir:     tmp,
		ProfilesDir: ".jitctx/profiles",
	})

	require.Error(t, err)

	var ubp *domerr.UnknownBundledProfileError
	require.True(t, errors.As(err, &ubp), "expected *UnknownBundledProfileError, got %T: %v", err, err)
	require.Equal(t, "never-exists", ubp.Name)
	require.Contains(t, ubp.Available, "spring-boot-hexagonal")
	require.True(t, errors.Is(err, domerr.ErrBundledProfileNotFound))
}

// TestResolver_AutoDetect_UserDirWinsFirst verifies that in auto-detect mode
// the alphabetically first valid user-dir profile is returned with
// Source == ProfileSourceCustom.
func TestResolver_AutoDetect_UserDirWinsFirst(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	profilesDir := ".jitctx/profiles"
	baseDir := filepath.Join(tmp, profilesDir)

	// "a" sorts before "spring-boot-hexagonal".
	makeUserProfileDir(t, baseDir, "a")
	makeUserProfileDir(t, baseDir, "spring-boot-hexagonal")

	r := newTestResolver(t)
	bundle, err := r.Resolve(context.Background(), profilevo.ResolveProfileInput{
		Name:        "",
		WorkDir:     tmp,
		ProfilesDir: profilesDir,
	})

	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.Equal(t, "a", bundle.Profile.Name)
	require.Equal(t, model.ProfileSourceCustom, bundle.Profile.Source)
}

// TestResolver_AutoDetect_BundledFallback verifies that when no user-dir
// profiles exist, auto-detect returns the first bundled profile with
// Source == ProfileSourceBundled.
func TestResolver_AutoDetect_BundledFallback(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir() // empty profiles directory

	r := newTestResolver(t)
	bundle, err := r.Resolve(context.Background(), profilevo.ResolveProfileInput{
		Name:        "",
		WorkDir:     tmp,
		ProfilesDir: ".jitctx/profiles",
	})

	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.Equal(t, model.ProfileSourceBundled, bundle.Profile.Source)
	require.Equal(t, "spring-boot-hexagonal", bundle.Profile.Name)
}

// TestResolver_AutoDetect_NoneAvailable verifies that ErrNoProfileMatch is
// returned when there are no user-dir or bundled profiles.
//
// The plan (Section 7.4) calls for injecting a fake ListBundledProfilesPort
// returning []. NewResolver accepts *BundleLoader and *Bundled (concrete
// types), so injecting a fake is not possible without changing the
// constructor — a Tier 2 concern out of scope for T6-G2. The test is
// skipped to document the gap; it will be enabled once the constructor
// is updated to accept interfaces.
func TestResolver_AutoDetect_NoneAvailable(t *testing.T) {
	t.Parallel()
	t.Skip("ErrNoProfileMatch auto-detect path requires interface injection into NewResolver; " +
		"constructor accepts concrete *Bundled — refactor tracked as Tier 2 follow-up")
}

// TestResolver_AutoDetect_SkipsMalformedUserDir verifies that a user-dir
// without a valid profile.yaml is silently skipped and the next valid
// candidate is returned.
func TestResolver_AutoDetect_SkipsMalformedUserDir(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	profilesDir := ".jitctx/profiles"
	baseDir := filepath.Join(tmp, profilesDir)

	// "broken" sorts before "spring-boot-hexagonal" and contains no profile.yaml.
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "broken"), 0o755))
	// Valid user-dir profile at a name that sorts after "broken".
	makeUserProfileDir(t, baseDir, "spring-boot-hexagonal")

	r := newTestResolver(t)
	bundle, err := r.Resolve(context.Background(), profilevo.ResolveProfileInput{
		Name:        "",
		WorkDir:     tmp,
		ProfilesDir: profilesDir,
	})

	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.Equal(t, "spring-boot-hexagonal", bundle.Profile.Name)
	require.Equal(t, model.ProfileSourceCustom, bundle.Profile.Source)
}

// TestResolver_ContextCancelled verifies that a pre-cancelled context causes
// Resolve to return ctx.Err() without performing I/O.
func TestResolver_ContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := newTestResolver(t)
	_, err := r.Resolve(ctx, profilevo.ResolveProfileInput{
		Name:        "spring-boot-hexagonal",
		WorkDir:     t.TempDir(),
		ProfilesDir: ".jitctx/profiles",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled))
}
