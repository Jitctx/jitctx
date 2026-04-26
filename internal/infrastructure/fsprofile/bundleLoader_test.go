package fsprofile_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
)

// repoRoot resolves the repository root from the package directory.
// Go tests run with cwd = the package directory, so three levels up
// from internal/infrastructure/fsprofile/ yields the repo root.
func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../../..")
	require.NoError(t, err)
	return abs
}

func ep04Fixture(t *testing.T, scenario string) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "testdata", "ep04us001", scenario)
}

func nopBundleLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func bundleInput(dir, bundledName string) profilevo.LoadProfileBundleInput {
	return profilevo.LoadProfileBundleInput{Dir: dir, BundledName: bundledName}
}

// ---------------------------------------------------------------------------
// Scenario 1: valid directory with profile.yaml and templates/
// ---------------------------------------------------------------------------

func TestBundleLoader_LoadsValidDirectory(t *testing.T) {
	t.Parallel()

	dir := ep04Fixture(t, "valid-profile")
	loader := fsprofile.NewBundleLoader(nopBundleLogger())

	bundle, err := loader.LoadBundle(context.Background(), bundleInput(dir, ""))
	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.NotNil(t, bundle.Profile)
	require.Equal(t, "spring-boot-hexagonal", bundle.Profile.Name)
	require.Equal(t, model.ProfileSourceCustom, bundle.Profile.Source)
	require.True(t, strings.HasSuffix(bundle.Dir, "valid-profile"),
		"Dir %q should end with valid-profile", bundle.Dir)
}

func TestBundleLoader_LoadsTemplatesIntoMemory(t *testing.T) {
	t.Parallel()

	dir := ep04Fixture(t, "valid-profile")
	loader := fsprofile.NewBundleLoader(nopBundleLogger())

	bundle, err := loader.LoadBundle(context.Background(), bundleInput(dir, ""))
	require.NoError(t, err)
	require.NotNil(t, bundle)

	tmplBytes, ok := bundle.GetTemplate("service.java.tmpl")
	require.True(t, ok, "GetTemplate should return true for service.java.tmpl")
	require.NotEmpty(t, tmplBytes, "template bytes should be non-empty")
}

// ---------------------------------------------------------------------------
// Scenario 2: missing profile.yaml
// ---------------------------------------------------------------------------

func TestBundleLoader_MissingProfileYaml(t *testing.T) {
	t.Parallel()

	dir := ep04Fixture(t, "missing-yaml")
	loader := fsprofile.NewBundleLoader(nopBundleLogger())

	_, err := loader.LoadBundle(context.Background(), bundleInput(dir, ""))
	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrProfileYamlMissing),
		"expected errors.Is(err, ErrProfileYamlMissing), got: %v", err)
	require.Contains(t, err.Error(), "profile.yaml not found")
}

// ---------------------------------------------------------------------------
// Scenario 3: type referencing a missing template fails at load time
// ---------------------------------------------------------------------------

func TestBundleLoader_MissingTemplate(t *testing.T) {
	t.Parallel()

	dir := ep04Fixture(t, "missing-template")
	loader := fsprofile.NewBundleLoader(nopBundleLogger())

	_, err := loader.LoadBundle(context.Background(), bundleInput(dir, ""))
	require.Error(t, err)

	// Must be (or wrap) *TemplateMissingError.
	var tmplErr *domerr.TemplateMissingError
	require.True(t, errors.As(err, &tmplErr),
		"expected *TemplateMissingError, got %T: %v", err, err)

	// errors.Is must match both sentinels per Section 2 of the plan.
	require.True(t, errors.Is(err, domerr.ErrTemplateMissing),
		"errors.Is(err, ErrTemplateMissing) must be true")
	require.True(t, errors.Is(err, domerr.ErrProfileInvalid),
		"errors.Is(err, ErrProfileInvalid) must be true")

	// Error message must mention both the template filename and the type ID.
	require.Contains(t, err.Error(), "service.java.tmpl",
		"error message should contain template filename")
	require.Contains(t, err.Error(), "service",
		"error message should contain type ID")
}

// ---------------------------------------------------------------------------
// Additional robustness tests
// ---------------------------------------------------------------------------

func TestBundleLoader_PathIsFileNotDir(t *testing.T) {
	t.Parallel()

	// Point Dir at a regular file (profile.yaml inside valid-profile).
	file := filepath.Join(ep04Fixture(t, "valid-profile"), "profile.yaml")
	loader := fsprofile.NewBundleLoader(nopBundleLogger())

	_, err := loader.LoadBundle(context.Background(), bundleInput(file, ""))
	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrProfileInvalid),
		"expected wrapped ErrProfileInvalid, got: %v", err)
}

func TestBundleLoader_NeitherDirNorBundled(t *testing.T) {
	t.Parallel()

	loader := fsprofile.NewBundleLoader(nopBundleLogger())

	_, err := loader.LoadBundle(context.Background(), bundleInput("", ""))
	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrProfileInvalid),
		"expected wrapped ErrProfileInvalid for empty input, got: %v", err)
}

func TestBundleLoader_ContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	loader := fsprofile.NewBundleLoader(nopBundleLogger())

	_, err := loader.LoadBundle(ctx, bundleInput(ep04Fixture(t, "valid-profile"), ""))
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled),
		"expected context.Canceled, got: %v", err)
}
