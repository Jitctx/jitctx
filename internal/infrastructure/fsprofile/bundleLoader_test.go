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

func ep04us002Fixture(t *testing.T, scenario string) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "testdata", "ep04us002", scenario)
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

// ---------------------------------------------------------------------------
// EP04US-002 tests — declarative type classification fields (T6-G3)
// ---------------------------------------------------------------------------

// TestBundleLoader_LoadsSixEP03Types asserts that a profile.yaml declaring
// all six EP-03 type IDs is loaded into a bundle whose RawTypes slice contains
// exactly those 6 entries in declared order.
func TestBundleLoader_LoadsSixEP03Types(t *testing.T) {
	t.Parallel()

	dir := ep04us002Fixture(t, "sixTypes")
	loader := fsprofile.NewBundleLoader(nopBundleLogger())

	bundle, err := loader.LoadBundle(t.Context(), bundleInput(dir, ""))
	require.NoError(t, err)
	require.NotNil(t, bundle)

	wantIDs := []string{
		"input-port",
		"output-port",
		"entity",
		"aggregate-root",
		"service",
		"rest-adapter",
	}
	require.Len(t, bundle.RawTypes, len(wantIDs), "expected exactly %d type entries", len(wantIDs))

	gotIDs := make([]string, len(bundle.RawTypes))
	for i, rt := range bundle.RawTypes {
		gotIDs[i] = rt.ID
	}
	require.Equal(t, wantIDs, gotIDs, "RawTypes IDs must match declared order")

	// Verify the classification slice for the one entry that declares it.
	var serviceType *model.ProfileTypeDeclaration
	for i := range bundle.RawTypes {
		if bundle.RawTypes[i].ID == "service" {
			serviceType = &bundle.RawTypes[i]
			break
		}
	}
	require.NotNil(t, serviceType, "service type must be present")
	require.NotEmpty(t, serviceType.Classification, "service type must have at least one classification rule")
}

// TestBundleLoader_LoadsCustomDomainEventType asserts that a profile.yaml
// declaring a single "domain-event" type with an implements_all classification
// rule is loaded correctly — proving new types can be added purely declaratively.
func TestBundleLoader_LoadsCustomDomainEventType(t *testing.T) {
	t.Parallel()

	dir := ep04us002Fixture(t, "customDomainEvent")
	loader := fsprofile.NewBundleLoader(nopBundleLogger())

	bundle, err := loader.LoadBundle(t.Context(), bundleInput(dir, ""))
	require.NoError(t, err)
	require.NotNil(t, bundle)

	var domainEventType *model.ProfileTypeDeclaration
	for i := range bundle.RawTypes {
		if bundle.RawTypes[i].ID == "domain-event" {
			domainEventType = &bundle.RawTypes[i]
			break
		}
	}
	require.NotNil(t, domainEventType, "bundle must contain a type with ID \"domain-event\"")
	require.NotEmpty(t, domainEventType.Classification, "domain-event must have at least one classification rule")

	firstRule := domainEventType.Classification[0]
	require.Equal(t, []string{"DomainEvent"}, firstRule.ImplementsAll,
		"first classification rule must have ImplementsAll == [DomainEvent]")
}

// TestBundleLoader_LoadsOrSemanticsType asserts that a profile.yaml declaring
// a type with TWO classification entries is loaded with both entries preserved
// in order — the loader must not collapse OR entries.
func TestBundleLoader_LoadsOrSemanticsType(t *testing.T) {
	t.Parallel()

	dir := ep04us002Fixture(t, "orSemantics")
	loader := fsprofile.NewBundleLoader(nopBundleLogger())

	bundle, err := loader.LoadBundle(t.Context(), bundleInput(dir, ""))
	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.NotEmpty(t, bundle.RawTypes, "bundle must have at least one type entry")

	// The OR-semantics fixture declares exactly one type; grab it.
	orType := bundle.RawTypes[0]
	require.GreaterOrEqual(t, len(orType.Classification), 2,
		"type must have at least 2 classification entries to express OR semantics")
}
