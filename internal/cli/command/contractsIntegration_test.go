package command_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	appcontractsuc "github.com/jitctx/jitctx/internal/application/usecase/contractsuc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
	"github.com/jitctx/jitctx/internal/infrastructure/fsspec"
	"github.com/jitctx/jitctx/internal/infrastructure/mdspec"
)

// newContractsCmdFor builds a real cobra contracts command wired with all real
// adapters (no mocks). Returns the command plus captured stdout and stderr buffers.
func newContractsCmdFor(t *testing.T, workDir, plansDir string) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	stderrBuf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(stderrBuf, nil))

	specFinder := fsspec.NewFinder()
	parser := mdspec.New()
	mapper := service.NewContractPathMapper()
	describer := service.NewContractRoleDescriber()
	resolver := service.NewContractTargetResolver()
	manifestStore := fsmanifest.New(filepath.Join(workDir, "project-state.yaml"))

	realContracts := appcontractsuc.New(specFinder, parser, mapper, describer, resolver, manifestStore, logger)
	cmd := command.NewContractsCmd(realContracts, workDir, plansDir, logger)

	stdoutBuf := &bytes.Buffer{}
	cmd.SetOut(stdoutBuf)
	cmd.SetErr(stderrBuf)

	return cmd, stdoutBuf, stderrBuf
}

// writeFixture copies the fixture spec from testdata/contractsSlice/specs/<feature>.md
// into <workDir>/jitctx-plans/<feature>.md.
func writeFixture(t *testing.T, workDir, feature string) {
	t.Helper()
	root := findProjectRoot(t)
	src := filepath.Join(root, "testdata", "contractsSlice", "specs", feature+".md")
	data, err := os.ReadFile(src)
	require.NoError(t, err, "read fixture file %s", src)

	destDir := filepath.Join(workDir, "jitctx-plans")
	require.NoError(t, os.MkdirAll(destDir, 0o755))
	dest := filepath.Join(destDir, feature+".md")
	require.NoError(t, os.WriteFile(dest, data, 0o644))
}

func TestContracts_Integration_ServiceImplementation(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFixture(t, tmp, "create-user")

	cmd, stdout, _ := newContractsCmdFor(t, tmp, "")
	cmd.SetArgs([]string{
		"--for", "src/main/java/com/app/user/application/UserServiceImpl.java",
		"--feature", "create-user",
	})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	out := stdout.String()
	require.Contains(t, out, "Target: UserServiceImpl")
	require.Contains(t, out, "Source: spec")
	require.Contains(t, out, "CreateUserUseCase")
	require.Contains(t, out, "UserRepository")
	require.Contains(t, out, "UserResponse execute(CreateUserCommand cmd)")
	require.Contains(t, out, "Optional<User> findByEmail(String email)")
}

func TestContracts_Integration_Controller_NoUnrelated(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFixture(t, tmp, "create-user")

	cmd, stdout, _ := newContractsCmdFor(t, tmp, "")
	cmd.SetArgs([]string{
		"--for", "src/main/java/com/app/user/adapter/web/UserController.java",
		"--feature", "create-user",
	})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	out := stdout.String()
	require.Contains(t, out, "UserController")
	require.Contains(t, out, "CreateUserUseCase")
	require.Contains(t, out, "POST /users")

	require.NotContains(t, out, "UserServiceImpl")
	require.NotContains(t, out, "UserRepository")
}

func TestContracts_Integration_UnknownFile_ExitsAndSuggests(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir() // empty — no spec, no manifest

	cmd, _, stderr := newContractsCmdFor(t, tmp, "")
	cmd.SetArgs([]string{
		"--for", "src/main/java/Random.java",
	})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)

	errText := err.Error() + stderr.String()
	require.Contains(t, errText, "could not find contract")
	require.Contains(t, errText, "Random")
	require.Contains(t, errText, "jitctx scan")
	require.Contains(t, errText, "jitctx plan")
}

func TestContracts_Integration_JSONFormat(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFixture(t, tmp, "create-user")

	cmd, stdout, _ := newContractsCmdFor(t, tmp, "")
	cmd.SetArgs([]string{
		"--for", "src/main/java/com/app/user/adapter/web/UserController.java",
		"--feature", "create-user",
		"--format", "json",
	})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))

	require.Contains(t, result, "source")
	require.Contains(t, result, "target")
	require.Contains(t, result, "related")

	target, ok := result["target"].(map[string]any)
	require.True(t, ok, "target must be a JSON object")
	require.Contains(t, target, "name")
	require.Equal(t, "UserController", target["name"])
}

func TestContracts_Integration_ManifestFallback(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// Build a minimal ProjectState containing one module with one Contract named "UserServiceImpl".
	state := &model.ProjectState{
		GeneratedAt: time.Now().UTC(),
		Modules: []model.Module{
			{
				ID:   "user-management",
				Path: "src/main/java/com/app/user",
				Contracts: []model.Contract{
					{
						Name: "UserServiceImpl",
						Type: model.ContractService,
						Path: "application/UserServiceImpl.java",
						Methods: []model.Method{
							{Signature: "UserResponse execute(CreateUserCommand cmd)"},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	manifestStore := fsmanifest.New(filepath.Join(tmp, "project-state.yaml"))
	require.NoError(t, manifestStore.Save(ctx, state))

	// No --feature, no --file → manifest-only mode.
	cmd, stdout, _ := newContractsCmdFor(t, tmp, "")
	cmd.SetArgs([]string{
		"--for", "src/main/java/com/app/user/application/UserServiceImpl.java",
	})

	require.NoError(t, cmd.ExecuteContext(ctx))

	out := stdout.String()
	require.Contains(t, out, "Source: manifest")
	require.Contains(t, out, "UserServiceImpl")
}

func TestContracts_Integration_BothFlagsRejected(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	cmd, _, _ := newContractsCmdFor(t, tmp, "")
	cmd.SetArgs([]string{
		"--for", "x.java",
		"--feature", "f",
		"--file", "/tmp/y.md",
	})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "mutually exclusive")
}
