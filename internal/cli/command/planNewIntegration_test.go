package command_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	appplannewuc "github.com/jitctx/jitctx/internal/application/usecase/plannewuc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/domain/service"
	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
	"github.com/jitctx/jitctx/internal/infrastructure/fsspec"
)

// stubLayersPlan satisfies planuc.UseCase but always returns an error —
// it is never called when the --new flag is active.
type stubLayersPlan struct{}

func (stubLayersPlan) Execute(_ context.Context, _ planvo.LayersInput) (planvo.LayersOutput, error) {
	return planvo.LayersOutput{}, errors.New("not used")
}

// newPlanCmdFor builds a NewPlanCmd wired with real adapters pointing at
// the given workDir / plansDir. Returns the command plus captured stdout
// and stderr buffers.
func newPlanCmdFor(t *testing.T, workDir, plansDir string) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	renderer := fsspec.New()
	writer := fsspec.NewWriter()
	resolver := service.NewSpecPathResolver()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	realPlanNew := appplannewuc.New(renderer, writer, resolver, logger)

	var stdout, stderr bytes.Buffer
	cmd := command.NewPlanCmd(stubLayersPlan{}, realPlanNew, workDir, plansDir, logger)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	return cmd, &stdout, &stderr
}

func TestPlanNewCmd_Integration_DefaultLocation(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cmd, stdout, _ := newPlanCmdFor(t, tmp, "")
	cmd.SetArgs([]string{"--new", "create-user", "--module", "user-management"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	specPath := filepath.Join(tmp, "jitctx-plans", "create-user.md")
	require.FileExists(t, specPath)

	content, err := os.ReadFile(specPath)
	require.NoError(t, err)
	body := string(content)

	require.Contains(t, body, "# Feature: create-user")
	require.Contains(t, body, "Module: user-management")
	require.Equal(t, 3, strings.Count(body, "## Contract: <Name>"))
	require.Contains(t, body, "Type: input-port")
	require.Contains(t, body, "Type: service")
	require.Contains(t, body, "Type: rest-adapter")
	require.Contains(t, body, "<TODO>")

	out := stdout.String()
	require.Contains(t, out, "created")
	require.Contains(t, out, specPath)
}

func TestPlanNewCmd_Integration_FileAlreadyExists(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	plansDir := filepath.Join(tmp, "jitctx-plans")
	require.NoError(t, os.MkdirAll(plansDir, 0o755))

	specPath := filepath.Join(plansDir, "create-user.md")
	require.NoError(t, os.WriteFile(specPath, []byte("original-content"), 0o644))

	cmd, _, stderr := newPlanCmdFor(t, tmp, "")
	cmd.SetArgs([]string{"--new", "create-user", "--module", "user-management"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)

	errMsg := err.Error() + stderr.String()
	require.Contains(t, errMsg, "spec file already exists")

	content, readErr := os.ReadFile(specPath)
	require.NoError(t, readErr)
	require.Equal(t, "original-content", string(content))
}

func TestPlanNewCmd_Integration_ConfiguredPlansDir(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cmd, _, _ := newPlanCmdFor(t, tmp, "specs/features")
	cmd.SetArgs([]string{"--new", "create-user", "--module", "user-management"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	specPath := filepath.Join(tmp, "specs", "features", "create-user.md")
	require.FileExists(t, specPath)
}

func TestPlanNewCmd_Integration_MissingModuleFlag(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cmd, _, _ := newPlanCmdFor(t, tmp, "")
	cmd.SetArgs([]string{"--new", "create-user"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "--module is required")
}

func TestPlanNewCmd_Integration_InvalidFeatureWithSlash(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cmd, _, _ := newPlanCmdFor(t, tmp, "")
	cmd.SetArgs([]string{"--new", "bad/feature", "--module", "x"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "path separators")
}
