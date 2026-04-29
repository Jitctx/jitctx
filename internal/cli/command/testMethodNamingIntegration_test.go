package command_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	appaudituc "github.com/jitctx/jitctx/internal/application/usecase/audituc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/domain/service"
	"github.com/jitctx/jitctx/internal/infrastructure/fsconfig"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
	"github.com/jitctx/jitctx/internal/infrastructure/treesitter"
)

// newAuditCmdForTestMethodNaming builds a real cobra audit command
// wired with real infrastructure adapters pointing at the given workDir.
// This is a local copy per Q3 resolution (no upstream DRY refactor in this PR).
func newAuditCmdForTestMethodNaming(t *testing.T, workDir, manifestPath string) (*bytes.Buffer, *bytes.Buffer, func(args ...string) error) {
	t.Helper()

	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	profileDetector := fsprofile.NewDetectorWithLogger(profilesDir, logger)
	auditRulesLoader := fsprofile.NewAuditRulesLoader(profilesDir, logger)
	manifestStore := fsmanifest.New(manifestPath)
	tsParser := treesitter.New()
	tsWalker := treesitter.NewWalker()
	evaluator := service.NewAuditEvaluator()
	configLoader := fsconfig.New(logger)
	auditFilter := service.NewAuditRuleFilter()

	bundleAuditRulesLoader := fsprofile.NewBundleAuditRulesAdapter()
	bundled := fsprofile.NewBundled()
	bundleLoader := fsprofile.NewBundleLoader(logger, nil)
	resolver := fsprofile.NewResolver(bundleLoader, bundled, logger)

	uc := appaudituc.New(
		manifestStore,
		profileDetector,
		auditRulesLoader,
		tsWalker,
		tsParser,
		tsParser,
		configLoader,
		auditFilter,
		evaluator,
		logger,
		bundleAuditRulesLoader,
		resolver,
		filepath.Join(workDir, ".jitctx", "profiles"),
	)

	var stdout, stderr bytes.Buffer
	cmd := command.NewAuditCmd(uc, logger)
	cmd.SilenceUsage = true
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	return &stdout, &stderr, func(args ...string) error {
		cmd.SetArgs(args)
		return cmd.ExecuteContext(context.Background())
	}
}

// TestAuditCmd_Integration_TestMethodNaming_CompliantNoViolation verifies that
// a project whose test methods already follow the required naming convention
// produces no [test-naming] violations.
// Backs PC01US-005 AC1.
func TestAuditCmd_Integration_TestMethodNaming_CompliantNoViolation(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us005TestMethodNaming", "projectClean"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForTestMethodNaming(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.NotContains(t, out, "[test-naming]",
		"compliant test methods must not trigger the test-naming rule")
}

// TestAuditCmd_Integration_TestMethodNaming_NonCompliantFlagsViolation verifies
// that a test method whose name does not match the required pattern triggers
// exactly one [test-naming] violation with the correct file:line evidence and
// the literal evidence substring required by AC2.
// Backs PC01US-005 AC2.
func TestAuditCmd_Integration_TestMethodNaming_NonCompliantFlagsViolation(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us005TestMethodNaming", "projectViolating"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForTestMethodNaming(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.Contains(t, out, "[test-naming]",
		"non-compliant test method name must trigger the test-naming rule")
	require.Contains(t, out, "UserServiceTest.java:7",
		"violation must reference the method_declaration line (line 7 is where @Test annotation starts)")
	require.Contains(t, out, "name=testFindUser, expected_pattern=^should[A-Z].*_when[A-Z].*$",
		"violation message must contain the literal evidence substring required by AC2")
	require.Equal(t, 1, strings.Count(out, "[test-naming]"),
		"exactly one violation must be reported for a single offending method")
}

// TestAuditCmd_Integration_TestMethodNaming_Determinism runs the audit against
// the violating fixture in two separate temp dirs and asserts byte-identical
// output after normalising the temp-dir path prefix.
// Backs PC01RNF-003 (deterministic output).
func TestAuditCmd_Integration_TestMethodNaming_Determinism(t *testing.T) {
	t.Parallel()

	// First run.
	workDir1 := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us005TestMethodNaming", "projectViolating"), workDir1)
	manifestPath1 := filepath.Join(workDir1, "project-state.yaml")
	stdout1, _, run1 := newAuditCmdForTestMethodNaming(t, workDir1, manifestPath1)
	require.NoError(t, run1("--dir", workDir1, "--manifest", manifestPath1))

	// Second run (separate temp dir — no shared state).
	workDir2 := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us005TestMethodNaming", "projectViolating"), workDir2)
	manifestPath2 := filepath.Join(workDir2, "project-state.yaml")
	stdout2, _, run2 := newAuditCmdForTestMethodNaming(t, workDir2, manifestPath2)
	require.NoError(t, run2("--dir", workDir2, "--manifest", manifestPath2))

	// Normalise temp-dir path prefix so the comparison is path-agnostic.
	normalize := func(s, tmpDir string) string {
		return strings.ReplaceAll(s, tmpDir, "<TMP>")
	}

	require.Equal(t,
		normalize(stdout1.String(), workDir1),
		normalize(stdout2.String(), workDir2),
		"audit output must be byte-identical across runs on the same fixture (PC01RNF-003)")
}
