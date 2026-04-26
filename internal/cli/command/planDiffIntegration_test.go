package command_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	appdiffuc "github.com/jitctx/jitctx/internal/application/usecase/diffuc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/domain/service"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
	"github.com/jitctx/jitctx/internal/infrastructure/fsspec"
	"github.com/jitctx/jitctx/internal/infrastructure/mdspec"
)

// newDiffCmdFor builds a real cobra plan command wired with real diff
// infrastructure adapters pointing at the given workDir. Returns the command
// plus captured stdout and stderr buffers.
func newDiffCmdFor(t *testing.T, workDir string) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	specFinder := fsspec.NewFinder()
	mdParser := mdspec.New()
	manifestStore := fsmanifest.New(manifestPath)
	differ := service.NewContractDiffer(service.NewSignatureNormalizer())
	layerer := service.NewDependencyLayerer()

	diffUC := appdiffuc.New(specFinder, mdParser, manifestStore, differ, layerer, logger)

	var stdout, stderr bytes.Buffer
	cmd := command.NewPlanCmd(stubLayersPlan{}, stubPlanNew{}, diffUC, workDir, "", logger)
	cmd.SilenceUsage = true
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	return cmd, &stdout, &stderr
}

// ─── planDiffMissing — missing contract ──────────────────────────────────────

// TestPlanDiff_Integration_MissingContract copies the planDiffMissing fixture,
// runs `jitctx plan --feature update-user-flow --diff`, and asserts the report
// contains a CREATE action for ChangeUserStatusUseCase in Layer 0.
func TestPlanDiff_Integration_MissingContract(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "planDiffMissing", "project"), workDir)

	cmd, stdout, _ := newDiffCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "update-user-flow", "--diff"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	out := stdout.String()
	require.Contains(t, out, "Layer 0")
	require.Contains(t, out, "CREATE: ChangeUserStatusUseCase (input-port)")

	// Golden match.
	expectedPath := fixtureDir(t, "planDiffMissing", "expected", "report.md")
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	require.Equal(t, string(expected), out,
		"diff report for planDiffMissing must match golden byte-for-byte")
}

// ─── planDiffSignature — signature divergence ────────────────────────────────

// TestPlanDiff_Integration_SignatureDivergence copies the planDiffSignature
// fixture, runs `jitctx plan --feature x --diff`, and asserts the report
// contains a MODIFY action for UserRepository with added: save, removed: persist.
func TestPlanDiff_Integration_SignatureDivergence(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "planDiffSignature", "project"), workDir)

	cmd, stdout, _ := newDiffCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "x", "--diff"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	out := stdout.String()
	require.Contains(t, out, "MODIFY: UserRepository")
	require.Contains(t, out, "added: save")
	require.Contains(t, out, "removed: persist")

	// Golden match.
	expectedPath := fixtureDir(t, "planDiffSignature", "expected", "report.md")
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	require.Equal(t, string(expected), out,
		"diff report for planDiffSignature must match golden byte-for-byte")
}

// ─── planDiffExtra — extra contract in manifest ──────────────────────────────

// TestPlanDiff_Integration_ExtraContract copies the planDiffExtra fixture,
// runs `jitctx plan --feature x --diff`, and asserts the report contains
// an EXTRA action for DeprecatedHelper in the Extras section.
func TestPlanDiff_Integration_ExtraContract(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "planDiffExtra", "project"), workDir)

	cmd, stdout, _ := newDiffCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "x", "--diff"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	out := stdout.String()
	require.Contains(t, out, "Extras")
	require.Contains(t, out, "🔵 INFO")
	require.Contains(t, out, "EXTRA: DeprecatedHelper")

	// Golden match.
	expectedPath := fixtureDir(t, "planDiffExtra", "expected", "report.md")
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	require.Equal(t, string(expected), out,
		"diff report for planDiffExtra must match golden byte-for-byte")
}

// ─── planDiffClean — no diff detected ───────────────────────────────────────

// TestPlanDiff_Integration_Clean copies the planDiffClean fixture, runs
// `jitctx plan --feature x --diff`, and asserts the stdout is exactly the
// verbatim clean line.
func TestPlanDiff_Integration_Clean(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "planDiffClean", "project"), workDir)

	cmd, stdout, _ := newDiffCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "x", "--diff"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	require.Equal(t, "No diff detected. Current state matches spec.\n", stdout.String())
}

// ─── planDiffLayered — dependency ordering ───────────────────────────────────

// TestPlanDiff_Integration_Layered copies the planDiffLayered fixture, runs
// `jitctx plan --feature x --diff`, and asserts ContractB in Layer 0 and
// ContractA in Layer 1.
func TestPlanDiff_Integration_Layered(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "planDiffLayered", "project"), workDir)

	cmd, stdout, _ := newDiffCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "x", "--diff"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	out := stdout.String()

	layer0Idx := indexOfStr(out, "Layer 0")
	layer1Idx := indexOfStr(out, "Layer 1")
	require.True(t, layer0Idx >= 0, "Layer 0 must appear in output")
	require.True(t, layer1Idx > layer0Idx, "Layer 1 must appear after Layer 0")

	layer0Section := out[layer0Idx:layer1Idx]
	require.Contains(t, layer0Section, "ContractB")
	require.NotContains(t, layer0Section, "ContractA")

	layer1Section := out[layer1Idx:]
	require.Contains(t, layer1Section, "ContractA")

	// Golden match.
	expectedPath := fixtureDir(t, "planDiffLayered", "expected", "report.md")
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	require.Equal(t, string(expected), out,
		"diff report for planDiffLayered must match golden byte-for-byte")
}

// ─── planDiffWhitespace — whitespace-only method difference ──────────────────

// TestPlanDiff_Integration_WhitespaceTolerance copies the planDiffWhitespace
// fixture (spec has "User save( User user )", manifest has "User save(User user)"),
// runs `jitctx plan --feature x --diff`, and asserts no MODIFY action is emitted.
func TestPlanDiff_Integration_WhitespaceTolerance(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "planDiffWhitespace", "project"), workDir)

	cmd, stdout, _ := newDiffCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "x", "--diff"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	out := stdout.String()
	require.NotContains(t, out, "MODIFY")
	require.Equal(t, "No diff detected. Current state matches spec.\n", out)
}

// ─── Determinism (RNF-003) ────────────────────────────────────────────────────

// TestPlanDiff_Integration_Determinism runs `jitctx plan --diff` twice on the
// planDiffMissing fixture and asserts byte-identical stdout on both runs.
func TestPlanDiff_Integration_Determinism(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "planDiffMissing", "project"), workDir)

	// First run.
	cmd1, stdout1, _ := newDiffCmdFor(t, workDir)
	cmd1.SetArgs([]string{"--feature", "update-user-flow", "--diff"})
	require.NoError(t, cmd1.ExecuteContext(context.Background()))
	first := stdout1.String()

	// Second run (same workDir — nothing changed).
	cmd2, stdout2, _ := newDiffCmdFor(t, workDir)
	cmd2.SetArgs([]string{"--feature", "update-user-flow", "--diff"})
	require.NoError(t, cmd2.ExecuteContext(context.Background()))
	second := stdout2.String()

	require.Equal(t, first, second,
		"plan --diff output must be byte-identical across consecutive runs (RNF-003)")
}

// ─── Read-only guarantee (RNF-002) ────────────────────────────────────────────

// TestPlanDiff_Integration_ReadOnly SHA-256s every file in the workdir before
// and after running `jitctx plan --diff` on the planDiffMissing fixture, then
// asserts all hashes are unchanged (RNF-002: diff must never modify project files).
func TestPlanDiff_Integration_ReadOnly(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "planDiffMissing", "project"), workDir)

	hashAllFiles := func() map[string][sha256.Size]byte {
		t.Helper()
		hashes := make(map[string][sha256.Size]byte)
		err := filepath.WalkDir(workDir, func(path string, d os.DirEntry, walkErr error) error {
			require.NoError(t, walkErr)
			if d.IsDir() {
				return nil
			}
			f, err := os.Open(path)
			require.NoError(t, err)
			defer f.Close()

			h := sha256.New()
			_, err = io.Copy(h, f)
			require.NoError(t, err)

			var sum [sha256.Size]byte
			copy(sum[:], h.Sum(nil))
			hashes[path] = sum
			return nil
		})
		require.NoError(t, err)
		return hashes
	}

	before := hashAllFiles()
	require.NotEmpty(t, before, "fixture must contain at least one file")

	cmd, _, _ := newDiffCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "update-user-flow", "--diff"})
	require.NoError(t, cmd.ExecuteContext(context.Background()))

	after := hashAllFiles()
	require.Equal(t, before, after,
		"plan --diff must not modify any project file (RNF-002 read-only guarantee)")
}

// ─── Error paths ──────────────────────────────────────────────────────────────

// TestPlanDiff_Integration_ManifestMissing runs `jitctx plan --diff` against a
// fixture that has no project-state.yaml and asserts the error message mentions
// the missing manifest and hints to run 'jitctx scan'.
func TestPlanDiff_Integration_ManifestMissing(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	// Write only a spec file — no project-state.yaml.
	plansDir := filepath.Join(workDir, "jitctx-plans")
	require.NoError(t, os.MkdirAll(plansDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(plansDir, "x.md"), []byte(`# Feature: x
Module: m

## Contract: Foo
Type: service
`), 0o644))

	cmd, stdout, _ := newDiffCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "x", "--diff"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)
	require.Empty(t, stdout.String())
	require.Contains(t, err.Error(), "project-state.yaml not found")
	require.Contains(t, err.Error(), "jitctx scan")
}

// TestPlanDiff_Integration_SpecMissing runs `jitctx plan --diff` in a workdir
// that has a manifest but no spec file for the requested feature, and asserts
// the error message mentions the missing spec.
func TestPlanDiff_Integration_SpecMissing(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	// Write only a manifest — no spec files.
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "project-state.yaml"), []byte(`schema_version: 2
generated_at: 2026-04-25T00:00:00Z
stack:
  languages:
    - java
  frameworks:
    - spring-boot-hexagonal
modules: []
contexts: []
`), 0o644))

	cmd, stdout, _ := newDiffCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "x", "--diff"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)
	require.Empty(t, stdout.String())
	require.Contains(t, err.Error(), "spec file not found")
	require.Contains(t, err.Error(), "x")
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// indexOfStr returns the index of substr in s, or -1.
func indexOfStr(s, substr string) int {
	idx := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return idx + i
		}
	}
	return -1
}
