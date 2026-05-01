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

// newAuditCmdForForbidFieldInjection builds a real cobra audit command
// wired with real infrastructure adapters pointing at the given workDir.
// This is a local copy per Q3 resolution (no upstream DRY refactor in this PR).
func newAuditCmdForForbidFieldInjection(t *testing.T, workDir, manifestPath string) (*bytes.Buffer, *bytes.Buffer, func(args ...string) error) {
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

// TestAuditCmd_Integration_ForbidFieldInjection_OnProductionFieldFlagsViolation verifies
// that a field annotated with the forbidden injection annotation in a production Java source
// file triggers exactly one [no-field-injection] violation with correct file:line evidence.
// Backs PC01US-004 Scenario 1 (AC: violation reported on the field's line).
func TestAuditCmd_Integration_ForbidFieldInjection_OnProductionFieldFlagsViolation(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us004ForbidFieldInjection", "projectViolating"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForForbidFieldInjection(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.Contains(t, out, "[no-field-injection]",
		"production field with forbidden injection annotation must trigger the no-field-injection rule")
	require.Contains(t, out, "Foo.java:7",
		"violation must reference the field_declaration line (line 7 is the forbidden annotation)")
	require.Contains(t, out, "found=["+loadForbiddenToken(t, 3)+"]",
		"violation message must contain evidence of the forbidden annotation found")
	require.Equal(t, 1, strings.Count(out, "[no-field-injection]"),
		"exactly one violation must be reported for a single offending field")
}

// TestAuditCmd_Integration_ForbidFieldInjection_OnTestSupportFieldIsExempted verifies
// that a field annotated with the forbidden injection annotation whose file path matches the
// **/testsupport/** exempt_paths glob does NOT trigger [no-field-injection].
// Backs PC01US-004 Scenario 2 (AC: test-support files are exempt).
func TestAuditCmd_Integration_ForbidFieldInjection_OnTestSupportFieldIsExempted(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us004ForbidFieldInjection", "projectExempt"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForForbidFieldInjection(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.NotContains(t, out, "[no-field-injection]",
		"file under **/testsupport/** must be exempt from the no-field-injection rule")
}

// TestAuditCmd_Integration_ForbidFieldInjection_OnConstructorParameterIsAllowed verifies
// that a constructor parameter annotated with the injection annotation does NOT trigger
// [no-field-injection] because the rule targets fields only (target=field).
// Backs PC01US-004 Scenario 3 (AC: constructor-param injection annotation is allowed).
func TestAuditCmd_Integration_ForbidFieldInjection_OnConstructorParameterIsAllowed(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us004ForbidFieldInjection", "projectClean"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForForbidFieldInjection(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.NotContains(t, out, "[no-field-injection]",
		"constructor-param injection annotation must not trigger the no-field-injection rule (target=field discrimination)")
}

// TestAuditCmd_Integration_ForbidFieldInjection_Determinism runs the audit against the
// violating fixture in two separate temp dirs and asserts byte-identical output
// after normalising the temp-dir path prefix.
// Backs PC01RNF-003 (deterministic output).
func TestAuditCmd_Integration_ForbidFieldInjection_Determinism(t *testing.T) {
	t.Parallel()

	// First run.
	workDir1 := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us004ForbidFieldInjection", "projectViolating"), workDir1)
	manifestPath1 := filepath.Join(workDir1, "project-state.yaml")
	stdout1, _, run1 := newAuditCmdForForbidFieldInjection(t, workDir1, manifestPath1)
	require.NoError(t, run1("--dir", workDir1, "--manifest", manifestPath1))

	// Second run (separate temp dir — no shared state).
	workDir2 := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us004ForbidFieldInjection", "projectViolating"), workDir2)
	manifestPath2 := filepath.Join(workDir2, "project-state.yaml")
	stdout2, _, run2 := newAuditCmdForForbidFieldInjection(t, workDir2, manifestPath2)
	require.NoError(t, run2("--dir", workDir2, "--manifest", manifestPath2))

	// Normalise temp-dir path prefix so the comparison is path-agnostic.
	normalize := func(s, workDir string) string {
		return strings.ReplaceAll(s, workDir, "<TMP>")
	}

	require.Equal(t,
		normalize(stdout1.String(), workDir1),
		normalize(stdout2.String(), workDir2),
		"audit output must be byte-identical across runs on the same fixture (PC01RNF-003)")
}
