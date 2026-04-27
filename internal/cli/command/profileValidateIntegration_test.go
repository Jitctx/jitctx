package command_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	appprofilevalidateuc "github.com/jitctx/jitctx/internal/application/usecase/profilevalidateuc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
)

// buildProfileValidateCmd constructs a "profile validate" cobra command wired
// with a real *fsprofile.BundleLoader. The languageQueries dependency is nil —
// matches existing test sites (the loader is nil-tolerant, see bundleLoader.go:45-49).
func buildProfileValidateCmd(t *testing.T) *cobra.Command {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(
		// Discard all log output — tests assert on cobra stdout/stderr only.
		nopWriter{},
		&slog.HandlerOptions{Level: slog.LevelError},
	))
	loader := fsprofile.NewBundleLoader(logger, nil)
	uc := appprofilevalidateuc.New(loader, logger)
	return command.NewProfileValidateCmd(uc, logger)
}

// nopWriter is an io.Writer that discards all bytes, used to suppress slog output.
type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

// TestProfileValidate_CleanProfile_ExitsZero verifies that a structurally
// valid profile directory produces "Profile valid" on stdout and no error.
// EP04US-007 / scenario 1 (clean profile, exit 0).
func TestProfileValidate_CleanProfile_ExitsZero(t *testing.T) {
	t.Parallel()

	fixture := fixtureDir(t, "ep04us007", "cleanProfile")

	cmd := buildProfileValidateCmd(t)

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{fixture})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Profile valid")
	// Stderr must not contain any fatal/error lines.
	require.NotContains(t, stderr.String(), "error")
}

// TestProfileValidate_MissingName_ExitsOne verifies that a profile.yaml
// without a "name:" field causes the command to return a
// *domerr.ProfileValidationError whose message contains the canonical literal.
// EP04US-007 / scenario 2 (missing name, exit 1).
func TestProfileValidate_MissingName_ExitsOne(t *testing.T) {
	t.Parallel()

	fixture := fixtureDir(t, "ep04us007", "missingName")

	cmd := buildProfileValidateCmd(t)

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{fixture})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)
	// format.TranslateError wraps ProfileValidationError into errors.New(string),
	// so errors.As cannot find the typed error at this point. Assert on the
	// rendered message string which carries the canonical literal.
	require.Contains(t, err.Error(), "missing required field: name")
}

// TestProfileValidate_MissingTemplate_ExitsOne verifies that a profile.yaml
// declaring a template file that is absent on disk causes the command to
// return an error whose message names the missing template.
// EP04US-007 / scenario 3 (missing template, exit 1).
func TestProfileValidate_MissingTemplate_ExitsOne(t *testing.T) {
	t.Parallel()

	fixture := fixtureDir(t, "ep04us007", "missingTemplate")

	cmd := buildProfileValidateCmd(t)

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{fixture})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)
	// The fixture declares template: missing.tmpl; TemplateMissingError.Error()
	// embeds the filename verbatim (see plan Section 0 discovery finding 6).
	require.Contains(t, err.Error(), "missing.tmpl")
}

// TestProfileValidate_UnknownClassificationField_WarnsButPasses verifies
// that an unknown classification key (e.g. "implementss") is surfaced as a
// warning on stderr but does NOT cause a non-zero exit.
// EP04US-007 / scenario 4 (unknown classification field, exit 0 with warning).
func TestProfileValidate_UnknownClassificationField_WarnsButPasses(t *testing.T) {
	t.Parallel()

	fixture := fixtureDir(t, "ep04us007", "unknownClassificationField")

	cmd := buildProfileValidateCmd(t)

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{fixture})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err, "unknown classification field must warn but not fail")
	require.Contains(t, stderr.String(), "unknown classification field 'implementss'")
	require.Contains(t, stdout.String(), "Profile valid")
}

// TestProfileValidate_DuplicateTypeIds_ExitsOne verifies that two types[]
// entries sharing the same id cause the command to return an error whose
// message contains the canonical literal "duplicate type id: service".
// EP04US-007 / scenario 5 (duplicate type ids, exit 1).
func TestProfileValidate_DuplicateTypeIds_ExitsOne(t *testing.T) {
	t.Parallel()

	fixture := fixtureDir(t, "ep04us007", "duplicateTypeIds")

	cmd := buildProfileValidateCmd(t)

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{fixture})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate type id: service")
}
