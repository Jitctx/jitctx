package command_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	appprofilevalidateuc "github.com/jitctx/jitctx/internal/application/usecase/profilevalidateuc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
)

// TestProfileAuthoringDoc_WorkedExampleIsValid pins the EP04US-008 acceptance
// gate "documentation includes worked example producing valid profile" as a
// CI-enforced invariant. The test runs `jitctx profile validate` against the
// worked-example fixture and asserts the canonical "Profile valid" stdout
// literal. Any drift in the doc's YAML — that is not also reflected in the
// fixture — will be caught by an editor of the .md (the doc's worked example
// MUST mirror this fixture byte-for-byte, see plan Section 8 Q1).
//
// EP04US-008 / Scenario: "Documentation includes worked example producing
// valid profile" (epic-04-profile-generalization.feature lines 243-247).
func TestProfileAuthoringDoc_WorkedExampleIsValid(t *testing.T) {
	t.Parallel()

	fixture := fixtureDir(t, "ep04us008", "workedExample")

	logger := slog.New(slog.NewTextHandler(
		// Discard slog output — assertions only target cobra stdout/stderr.
		nopLogWriter{},
		&slog.HandlerOptions{Level: slog.LevelError},
	))
	loader := fsprofile.NewBundleLoader(logger, nil)
	uc := appprofilevalidateuc.New(loader, logger)
	cmd := command.NewProfileValidateCmd(uc, logger)

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{fixture})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Profile valid")
	require.NotContains(t, stderr.String(), "error")
}

// nopLogWriter discards all bytes — used to suppress slog output in the test.
// Named differently from nopWriter in profileValidateIntegration_test.go to
// avoid duplicate-symbol errors in the same command_test package.
type nopLogWriter struct{}

func (nopLogWriter) Write(p []byte) (int, error) { return len(p), nil }
