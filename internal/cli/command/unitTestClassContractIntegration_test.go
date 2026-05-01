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

// newAuditCmdForUnitTestClassContract builds a real cobra audit command
// wired with real infrastructure adapters pointing at the given workDir.
// This is a local copy per Q3 resolution (no upstream DRY refactor in this PR).
func newAuditCmdForUnitTestClassContract(t *testing.T, workDir, manifestPath string) (*bytes.Buffer, *bytes.Buffer, func(args ...string) error) {
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

// TestAuditCmd_Integration_UnitTestClassContract_CleanFixture_NoViolation verifies
// that a project with both @ExtendWith(the correct runner extension) and @DisplayName
// produces no [unit-test-class-contract] violations.
// Backs PC01US-006 AC1.
func TestAuditCmd_Integration_UnitTestClassContract_CleanFixture_NoViolation(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us006UnitTestClassContract", "projectClean"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForUnitTestClassContract(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.NotContains(t, out, "[unit-test-class-contract]",
		"compliant test class must not trigger the unit-test-class-contract rule")
}

// TestAuditCmd_Integration_UnitTestClassContract_WrongExtensionArg_FlagsViolation
// verifies that a test class using @ExtendWith with a wrong runner extension instead of
// the expected one triggers exactly one [unit-test-class-contract] violation
// with the correct evidence substring.
// Backs PC01US-006 AC2.
func TestAuditCmd_Integration_UnitTestClassContract_WrongExtensionArg_FlagsViolation(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us006UnitTestClassContract", "projectWrongArg"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForUnitTestClassContract(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.Contains(t, out, "[unit-test-class-contract]",
		"wrong extension arg must trigger the unit-test-class-contract rule")
	mockitoExt := loadForbiddenToken(t, 2) + "Extension.class"
	springExt := loadForbiddenToken(t, 1) + "Extension.class"
	require.Contains(t, out, "annotation=ExtendWith, expected_value="+mockitoExt+", actual="+springExt,
		"violation message must contain the literal evidence substring required by AC2")
	require.Equal(t, 1, strings.Count(out, "[unit-test-class-contract]"),
		"exactly one violation must be reported for a single offending class")
}

// TestAuditCmd_Integration_UnitTestClassContract_MissingDisplayName_FlagsViolation
// verifies that a test class missing @DisplayName triggers a [unit-test-class-contract]
// violation with the correct evidence substring.
// Backs PC01US-006 AC3.
func TestAuditCmd_Integration_UnitTestClassContract_MissingDisplayName_FlagsViolation(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us006UnitTestClassContract", "projectMissingDisplayName"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForUnitTestClassContract(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.Contains(t, out, "[unit-test-class-contract]",
		"missing @DisplayName must trigger the unit-test-class-contract rule")
	require.Contains(t, out, "missing=[DisplayName]",
		"violation message must contain the literal evidence substring required by AC3")
}

// TestAuditCmd_Integration_UnitTestClassContract_Determinism runs the audit against
// the projectWrongArg fixture in two separate temp dirs and asserts byte-identical
// output after normalising the temp-dir path prefix.
// Backs PC01RNF-003 (deterministic output).
func TestAuditCmd_Integration_UnitTestClassContract_Determinism(t *testing.T) {
	t.Parallel()

	// First run.
	workDir1 := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us006UnitTestClassContract", "projectWrongArg"), workDir1)
	manifestPath1 := filepath.Join(workDir1, "project-state.yaml")
	stdout1, _, run1 := newAuditCmdForUnitTestClassContract(t, workDir1, manifestPath1)
	require.NoError(t, run1("--dir", workDir1, "--manifest", manifestPath1))

	// Second run (separate temp dir — no shared state).
	workDir2 := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us006UnitTestClassContract", "projectWrongArg"), workDir2)
	manifestPath2 := filepath.Join(workDir2, "project-state.yaml")
	stdout2, _, run2 := newAuditCmdForUnitTestClassContract(t, workDir2, manifestPath2)
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
