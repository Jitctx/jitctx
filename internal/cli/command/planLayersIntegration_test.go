package command_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	appplanuc "github.com/jitctx/jitctx/internal/application/usecase/planuc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/domain/service"
	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
	"github.com/jitctx/jitctx/internal/infrastructure/fsspec"
	"github.com/jitctx/jitctx/internal/infrastructure/mdspec"
)

// canonicalSpec is the 4-contract fixture used by most plan-layers integration tests.
const canonicalSpec = `# Feature: create-user
Module: user-management

## Contract: CreateUserUseCase
Type: input-port
Methods:
- UserResponse execute(CreateUserCommand cmd)

## Contract: UserRepository
Type: output-port
Methods:
- Optional<User> findByEmail(String email)
- User save(User user)

## Contract: UserServiceImpl
Type: service
Implements: CreateUserUseCase
DependsOn: UserRepository

## Contract: UserController
Type: rest-adapter
Uses: CreateUserUseCase
Endpoints:
- POST /users
`

// stubPlanNew satisfies plannewuc.UseCase but always errors — it is never
// triggered by the --feature / --file tests in this file.
type stubPlanNew struct{}

func (stubPlanNew) Execute(_ context.Context, _ planvo.NewTemplateInput) (planvo.NewTemplateOutput, error) {
	return planvo.NewTemplateOutput{}, errors.New("not used")
}

// newPlanLayersCmdFor builds a real cobra plan command wired with all real
// adapters (no mocks). Returns the command plus captured stdout and stderr buffers.
// stderrBuf collects both slog logger output AND cobra error text.
func newPlanLayersCmdFor(t *testing.T, workDir, plansDir string) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	stderrBuf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(stderrBuf, nil))

	specFinder := fsspec.NewFinder()
	parser := mdspec.New()
	layerer := service.NewDependencyLayerer()
	mapper := service.NewContractPathMapper()
	realLayers := appplanuc.New(specFinder, parser, layerer, mapper, logger)

	cmd := command.NewPlanCmd(realLayers, stubPlanNew{}, nil, workDir, plansDir, logger)

	stdoutBuf := &bytes.Buffer{}
	cmd.SetOut(stdoutBuf)
	cmd.SetErr(stderrBuf)

	return cmd, stdoutBuf, stderrBuf
}

// writeSpec writes content to <dir>/<feature>.md (creating parent dirs) and
// returns the absolute path.
func writeSpec(t *testing.T, dir, feature, content string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, feature+".md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestPlanLayers_Integration_HappyPath_Text(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeSpec(t, filepath.Join(tmp, "jitctx-plans"), "create-user", canonicalSpec)

	cmd, stdout, _ := newPlanLayersCmdFor(t, tmp, "")
	cmd.SetArgs([]string{"--feature", "create-user"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	out := stdout.String()

	// Layer 0 must contain CreateUserUseCase and UserRepository.
	require.Contains(t, out, "Layer 0")
	require.Contains(t, out, "CreateUserUseCase")
	require.Contains(t, out, "UserRepository")

	// Layer 1 must contain UserController and UserServiceImpl.
	require.Contains(t, out, "Layer 1")
	require.Contains(t, out, "UserController")
	require.Contains(t, out, "UserServiceImpl")

	// Within-layer alphabetical order: CreateUserUseCase before UserRepository in layer 0.
	layer0Start := strings.Index(out, "Layer 0")
	layer1Start := strings.Index(out, "Layer 1")
	require.True(t, layer0Start >= 0 && layer1Start > layer0Start, "Layer 0 must appear before Layer 1")

	layer0Section := out[layer0Start:layer1Start]
	idxCreate := strings.Index(layer0Section, "CreateUserUseCase")
	idxRepo := strings.Index(layer0Section, "UserRepository")
	require.True(t, idxCreate >= 0 && idxRepo > idxCreate,
		"CreateUserUseCase must appear before UserRepository in Layer 0")

	// Within Layer 1: UserController before UserServiceImpl.
	layer1Section := out[layer1Start:]
	idxController := strings.Index(layer1Section, "UserController")
	idxServiceImpl := strings.Index(layer1Section, "UserServiceImpl")
	require.True(t, idxController >= 0 && idxServiceImpl > idxController,
		"UserController must appear before UserServiceImpl in Layer 1")
}

func TestPlanLayers_Integration_JSON(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeSpec(t, filepath.Join(tmp, "jitctx-plans"), "create-user", canonicalSpec)

	cmd, stdout, _ := newPlanLayersCmdFor(t, tmp, "")
	cmd.SetArgs([]string{"--feature", "create-user", "--format", "json"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))

	// Assert top-level keys.
	require.Equal(t, "create-user", result["feature"])
	require.Equal(t, "user-management", result["module"])

	layers, ok := result["layers"].([]any)
	require.True(t, ok, "layers must be a JSON array")
	require.Len(t, layers, 2)

	// Layer 0 must be a non-empty array of contract objects.
	layer0, ok := layers[0].([]any)
	require.True(t, ok, "layers[0] must be a JSON array")
	require.NotEmpty(t, layer0)

	// Each contract object must have name, type, target_path keys.
	first, ok := layer0[0].(map[string]any)
	require.True(t, ok, "layers[0][0] must be a JSON object")
	require.Contains(t, first, "name")
	require.Contains(t, first, "type")
	require.Contains(t, first, "target_path")

	// Find CreateUserUseCase and assert its target_path.
	var found bool
	for _, item := range layer0 {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if obj["name"] == "CreateUserUseCase" {
			require.Equal(t, "port/in/CreateUserUseCase.java", obj["target_path"])
			found = true
		}
	}
	require.True(t, found, "CreateUserUseCase must be present in layer 0")
}

func TestPlanLayers_Integration_Cycle(t *testing.T) {
	t.Parallel()

	cycleSpec := `# Feature: broken
Module: broken-module

## Contract: A
Type: service
DependsOn: B

## Contract: B
Type: service
DependsOn: A
`

	tmp := t.TempDir()
	writeSpec(t, filepath.Join(tmp, "jitctx-plans"), "broken", cycleSpec)

	cmd, _, stderr := newPlanLayersCmdFor(t, tmp, "")
	cmd.SetArgs([]string{"--feature", "broken"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)

	errText := err.Error() + stderr.String()
	require.Contains(t, errText, "dependency cycle detected")
	require.Contains(t, errText, "A -> B -> A")
}

func TestPlanLayers_Integration_ExternalRefWarning(t *testing.T) {
	t.Parallel()

	externalSpec := `# Feature: with-external
Module: ext-module

## Contract: Solo
Type: service
Uses: ExternalThing
`

	tmp := t.TempDir()
	writeSpec(t, filepath.Join(tmp, "jitctx-plans"), "with-external", externalSpec)

	cmd, stdout, stderr := newPlanLayersCmdFor(t, tmp, "")
	cmd.SetArgs([]string{"--feature", "with-external"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	// Solo must appear in layer 0.
	require.Contains(t, stdout.String(), "Layer 0")
	require.Contains(t, stdout.String(), "Solo")

	// Stderr must contain the external reference warning.
	stderrStr := stderr.String()
	require.Contains(t, stderrStr, "external reference")
	require.Contains(t, stderrStr, "'ExternalThing'")
}

func TestPlanLayers_Integration_FilePath(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	absPath := writeSpec(t, filepath.Join(tmp, "anywhere"), "myspec", canonicalSpec)

	cmd, stdout, _ := newPlanLayersCmdFor(t, tmp, "")
	cmd.SetArgs([]string{"--file", absPath})

	require.NoError(t, cmd.ExecuteContext(context.Background()))
	require.Contains(t, stdout.String(), "Layer 0")
}

func TestPlanLayers_Integration_NotFound(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir() // empty — no spec files

	cmd, _, stderr := newPlanLayersCmdFor(t, tmp, "")
	cmd.SetArgs([]string{"--feature", "missing"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)

	errText := err.Error() + stderr.String()
	require.Contains(t, errText, "spec file not found")
	require.Contains(t, errText, "jitctx-plans/missing.md")
	require.Contains(t, errText, ".jitctx/plans/missing.md")
}

func TestPlanLayers_Integration_MultipleLocationsWarn(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// Write the same spec to BOTH conventional locations.
	writeSpec(t, filepath.Join(tmp, "jitctx-plans"), "create-user", canonicalSpec)
	writeSpec(t, filepath.Join(tmp, ".jitctx", "plans"), "create-user", canonicalSpec)

	cmd, _, stderr := newPlanLayersCmdFor(t, tmp, "")
	cmd.SetArgs([]string{"--feature", "create-user"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	// A warning about the additional location must appear in stderr.
	stderrStr := stderr.String()
	require.Contains(t, stderrStr, "additional")
	dotJitctxPath := filepath.Join(tmp, ".jitctx", "plans", "create-user.md")
	require.Contains(t, stderrStr, dotJitctxPath)
}
