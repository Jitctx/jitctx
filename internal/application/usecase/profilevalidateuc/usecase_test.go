package profilevalidateuc_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/application/usecase/profilevalidateuc"
	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// fakeLoadProfileBundlePort is a hand-rolled fake for
// profileport.LoadProfileBundlePort. It records calls and returns
// configurable results.
type fakeLoadProfileBundlePort struct {
	loadBundle func(ctx context.Context, input profilevo.LoadProfileBundleInput) (*model.ProfileBundle, error)
	callCount  int
}

func (f *fakeLoadProfileBundlePort) LoadBundle(ctx context.Context, input profilevo.LoadProfileBundleInput) (*model.ProfileBundle, error) {
	f.callCount++
	if f.loadBundle != nil {
		return f.loadBundle(ctx, input)
	}
	return &model.ProfileBundle{}, nil
}

// succeededLoader returns a fakeLoadProfileBundlePort that always succeeds
// (returns nil error and an empty ProfileBundle).
func succeededLoader() *fakeLoadProfileBundlePort {
	return &fakeLoadProfileBundlePort{}
}

// failingLoader returns a fakeLoadProfileBundlePort that always returns err.
func failingLoader(err error) *fakeLoadProfileBundlePort {
	return &fakeLoadProfileBundlePort{
		loadBundle: func(_ context.Context, _ profilevo.LoadProfileBundleInput) (*model.ProfileBundle, error) {
			return nil, err
		},
	}
}

// writeProfileYAML writes content to <dir>/profile.yaml and returns dir.
func writeProfileYAML(t *testing.T, dir, content string) string {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "profile.yaml"), []byte(content), 0o644))
	return dir
}

// TestValidateUC_CleanProfile verifies that a valid profile with all required
// fields and no unknown classification keys returns no errors and no warnings.
func TestValidateUC_CleanProfile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProfileYAML(t, dir, `name: clean
language: java
types:
  - id: service
    template: service.java.tmpl
    classification:
      - kind: service
`)
	// Create the template stub so the bundle loader would not fail.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "templates"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "templates", "service.java.tmpl"), []byte{}, 0o644))

	loader := succeededLoader()
	uc := profilevalidateuc.New(loader, nil)

	out, err := uc.Execute(context.Background(), profilevo.ValidateProfileInput{Path: dir})

	require.NoError(t, err)
	require.Empty(t, out.Errors)
	require.Empty(t, out.Warnings)
}

// TestValidateUC_PathDoesNotExist verifies that a non-existent path causes
// an immediate fatal with the expected message. The loader must NOT be called.
func TestValidateUC_PathDoesNotExist(t *testing.T) {
	t.Parallel()

	nonExistent := filepath.Join(t.TempDir(), "does-not-exist")
	loader := succeededLoader()
	uc := profilevalidateuc.New(loader, nil)

	out, err := uc.Execute(context.Background(), profilevo.ValidateProfileInput{Path: nonExistent})

	require.Error(t, err)

	var pve *domerr.ProfileValidationError
	require.ErrorAs(t, err, &pve)
	require.ErrorIs(t, err, domerr.ErrProfileValidationFailed)

	require.NotEmpty(t, out.Errors, "Errors slice must contain the path-not-found issue")
	require.Contains(t, out.Errors[0].Message, "does not exist")

	require.Equal(t, 0, loader.callCount, "loader must NOT be called when path does not exist")
}

// TestValidateUC_MissingName verifies that a profile.yaml without a "name:"
// field yields a fatal with the pinned literal.
func TestValidateUC_MissingName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// No name field; loader succeeds (returns nil error).
	writeProfileYAML(t, dir, `language: java
types: []
`)
	loader := succeededLoader()
	uc := profilevalidateuc.New(loader, nil)

	out, err := uc.Execute(context.Background(), profilevo.ValidateProfileInput{Path: dir})

	require.Error(t, err)

	var pve *domerr.ProfileValidationError
	require.ErrorAs(t, err, &pve)

	msgs := collectMessages(out.Errors)
	require.Contains(t, msgs, "missing required field: name")
}

// TestValidateUC_MissingTemplate verifies that when the loader returns a
// *TemplateMissingError the use case surfaces a fatal whose message names the
// missing template file.
func TestValidateUC_MissingTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProfileYAML(t, dir, `name: p
language: java
types:
  - id: service
    template: service.java.tmpl
`)
	// Fake loader returns the template-missing error regardless of disk state.
	tmplErr := &domerr.TemplateMissingError{
		ProfileName: "p",
		TypeID:      "service",
		Template:    "service.java.tmpl",
	}
	loader := failingLoader(tmplErr)
	uc := profilevalidateuc.New(loader, nil)

	out, err := uc.Execute(context.Background(), profilevo.ValidateProfileInput{Path: dir})

	require.Error(t, err)

	var pve *domerr.ProfileValidationError
	require.ErrorAs(t, err, &pve)

	msgs := collectMessages(out.Errors)
	found := false
	for _, m := range msgs {
		if containsStr(m, "service.java.tmpl") {
			found = true
			break
		}
	}
	require.True(t, found, "expected one of %v to mention 'service.java.tmpl'", msgs)
}

// TestValidateUC_UnknownClassificationField_IsWarning verifies that a typo'd
// classification key (e.g., "implementss") is surfaced as a non-fatal warning
// and Execute returns nil (exit 0).
func TestValidateUC_UnknownClassificationField_IsWarning(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProfileYAML(t, dir, `name: p
language: java
types:
  - id: service
    template: service.java.tmpl
    classification:
      - implementss: [Foo]
`)
	// Loader succeeds (KnownFields(false) masks the typo in the real impl).
	loader := succeededLoader()
	uc := profilevalidateuc.New(loader, nil)

	out, err := uc.Execute(context.Background(), profilevo.ValidateProfileInput{Path: dir})

	require.NoError(t, err, "unknown classification field must not be a fatal")
	require.Empty(t, out.Errors)

	warnMsgs := collectMessages(out.Warnings)
	found := false
	for _, m := range warnMsgs {
		if containsStr(m, "unknown classification field 'implementss'") {
			found = true
			break
		}
	}
	require.True(t, found, "expected a warning containing \"unknown classification field 'implementss'\"; got %v", warnMsgs)
}

// TestValidateUC_DuplicateTypeIDs verifies that two types with the same id
// produce a fatal containing the pinned literal.
func TestValidateUC_DuplicateTypeIDs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProfileYAML(t, dir, `name: p
language: java
types:
  - id: service
    template: service.java.tmpl
  - id: service
    template: service.java.tmpl
`)
	// Loader succeeds (the duplicate scan is done by the use case, not loader).
	loader := succeededLoader()
	uc := profilevalidateuc.New(loader, nil)

	out, err := uc.Execute(context.Background(), profilevo.ValidateProfileInput{Path: dir})

	require.Error(t, err)

	var pve *domerr.ProfileValidationError
	require.ErrorAs(t, err, &pve)

	msgs := collectMessages(out.Errors)
	found := false
	for _, m := range msgs {
		if containsStr(m, "duplicate type id: service") {
			found = true
			break
		}
	}
	require.True(t, found, "expected one of %v to contain 'duplicate type id: service'", msgs)
}

// TestValidateUC_FatalsAreSorted verifies that the Errors slice in the
// returned *ProfileValidationError is in alphabetical order (stable for
// format/errors.go rendering and integration-test asserts).
func TestValidateUC_FatalsAreSorted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Produce two fatals: missing name + duplicate type id. Their messages
	// must appear alphabetically sorted in the ProfileValidationError.Errors
	// slice. "duplicate type id: service" < "missing required field: name"
	// alphabetically.
	writeProfileYAML(t, dir, `language: java
types:
  - id: service
    template: service.java.tmpl
  - id: service
    template: service.java.tmpl
`)
	loader := succeededLoader()
	uc := profilevalidateuc.New(loader, nil)

	_, err := uc.Execute(context.Background(), profilevo.ValidateProfileInput{Path: dir})

	require.Error(t, err)
	var pve *domerr.ProfileValidationError
	require.ErrorAs(t, err, &pve)
	require.GreaterOrEqual(t, len(pve.Errors), 2, "expected at least 2 fatals")

	// Verify the slice is sorted.
	for i := 1; i < len(pve.Errors); i++ {
		require.LessOrEqual(t, pve.Errors[i-1], pve.Errors[i],
			"Errors not sorted at index %d: %q > %q", i, pve.Errors[i-1], pve.Errors[i])
	}
}

// TestValidateUC_WarningsPresentWhenFatalPresent verifies the "warning + fatal
// combo" scenario: when a typo'd classification field AND a duplicate type id
// are both present, the returned error is non-nil and Warnings is non-empty.
func TestValidateUC_WarningsPresentWhenFatalPresent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Both a typo'd classification key (warning) and a duplicate type id (fatal).
	writeProfileYAML(t, dir, `name: p
language: java
types:
  - id: service
    template: service.java.tmpl
    classification:
      - implementss: [Foo]
  - id: service
    template: service.java.tmpl
`)
	loader := succeededLoader()
	uc := profilevalidateuc.New(loader, nil)

	out, err := uc.Execute(context.Background(), profilevo.ValidateProfileInput{Path: dir})

	require.Error(t, err, "duplicate type id must cause a fatal")

	var pve *domerr.ProfileValidationError
	require.ErrorAs(t, err, &pve)
	require.NotEmpty(t, pve.Errors, "expected at least one fatal in ProfileValidationError")
	require.NotEmpty(t, out.Warnings, "expected at least one warning in output")
}

// collectMessages extracts the Message field from a slice of ValidationIssue.
func collectMessages(issues []profilevo.ValidationIssue) []string {
	msgs := make([]string, 0, len(issues))
	for _, i := range issues {
		msgs = append(msgs, i.Message)
	}
	return msgs
}

// containsStr reports whether s contains substr.
func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}
