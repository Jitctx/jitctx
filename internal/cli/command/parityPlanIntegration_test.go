package command_test

// parityPlanIntegration_test.go — EP04US-004 scenario 4 (plan parity).
//
// Composes appplanuc.New with real infrastructure adapters and asserts that
// stdout is byte-identical to the captured EP-03 baselines stored under
// testdata/ep04us004/parity/plan/baseline/.
//
// The fixture spec (createUser.md) is a copy of testdata/scaffold/createUser/spec.md.
//
// If the text formatter output ever needs intentional adjustment, re-capture
// the baseline with:
//
//	go test ./internal/cli/command/... -run TestParity_Plan_TextFormat_MatchesEP03Baseline -v
//
// and copy the printed stdout into testdata/ep04us004/parity/plan/baseline/plan.txt.
// Same for plan.json.

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	appplanuc "github.com/jitctx/jitctx/internal/application/usecase/planuc"
	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/service"
	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
	"github.com/jitctx/jitctx/internal/infrastructure/fsspec"
	"github.com/jitctx/jitctx/internal/infrastructure/mdspec"
)

// newParityPlanUC constructs a real appplanuc.Impl with all real adapters
// and a discard logger (no slog noise).
func newParityPlanUC(t *testing.T) *appplanuc.Impl {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return appplanuc.New(
		fsspec.NewFinder(),
		mdspec.New(),
		service.NewDependencyLayerer(),
		service.NewContractPathMapper(),
		logger,
	)
}

func TestParity_Plan_TextFormat_MatchesEP03Baseline(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Copy the fixture spec into the temp directory so the use case can read it.
	fixtureSpec := fixtureDir(t, "ep04us004", "parity", "plan", "fixture", "specs", "createUser.md")
	specsDir := filepath.Join(tmpDir, "specs")
	require.NoError(t, os.MkdirAll(specsDir, 0o755))

	specData, err := os.ReadFile(fixtureSpec)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(specsDir, "createUser.md"), specData, 0o644))

	uc := newParityPlanUC(t)

	out, err := uc.Execute(t.Context(), planvo.LayersInput{
		FilePath: filepath.Join(specsDir, "createUser.md"),
	})
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, format.WriteLayersText(&buf, out))

	baseline, err := os.ReadFile(fixtureDir(t, "ep04us004", "parity", "plan", "baseline", "plan.txt"))
	require.NoError(t, err)

	require.Equal(t, string(baseline), buf.String(),
		"plan text output must be byte-identical to the EP03 baseline")
}

func TestParity_Plan_JSONFormat_MatchesEP03Baseline(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Copy the fixture spec into the temp directory.
	fixtureSpec := fixtureDir(t, "ep04us004", "parity", "plan", "fixture", "specs", "createUser.md")
	specsDir := filepath.Join(tmpDir, "specs")
	require.NoError(t, os.MkdirAll(specsDir, 0o755))

	specData, err := os.ReadFile(fixtureSpec)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(specsDir, "createUser.md"), specData, 0o644))

	uc := newParityPlanUC(t)

	out, err := uc.Execute(t.Context(), planvo.LayersInput{
		FilePath: filepath.Join(specsDir, "createUser.md"),
	})
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, format.WriteLayersJSON(&buf, out))

	baseline, err := os.ReadFile(fixtureDir(t, "ep04us004", "parity", "plan", "baseline", "plan.json"))
	require.NoError(t, err)

	// Use structural JSON equality to avoid sensitivity to trailing newlines or
	// map key ordering differences across Go versions.
	require.JSONEq(t, string(baseline), buf.String(),
		"plan JSON output must be structurally identical to the EP03 baseline")
}
