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

// newAuditCmdForIntegrationTestBaseRequiredAnnotations builds a real cobra audit
// command wired with real infrastructure adapters pointing at the given workDir.
// Modelled on newAuditCmdForUseCaseParameterizedSupertype per Q-DRY resolution
// (local copy acceptable; no upstream refactor in this PR).
func newAuditCmdForIntegrationTestBaseRequiredAnnotations(t *testing.T, workDir, manifestPath string) (*bytes.Buffer, *bytes.Buffer, func(args ...string) error) {
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

// TestAuditCmd_Integration_IntegrationTestBaseRequiredAnnotations_AllThreePresentNoViolation
// verifies that a BaseIntegrationTest class declaring all three required
// annotations (@SpringBootTest, @Testcontainers, @ActiveProfiles("test"))
// produces zero [integration-test-base] violations. Backs PC01US-009 AC1.
func TestAuditCmd_Integration_IntegrationTestBaseRequiredAnnotations_AllThreePresentNoViolation(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us009IntegrationTestBaseRequiredAnnotations", "projectClean"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForIntegrationTestBaseRequiredAnnotations(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.NotContains(t, out, "[integration-test-base]",
		"compliant BaseIntegrationTest with all three annotations must not trigger the integration-test-base rule")
	require.NotContains(t, out, "missing=",
		"clean project must not emit any missing= evidence")
	require.NotContains(t, out, "annotation=ActiveProfiles",
		"clean project must not emit any annotation=ActiveProfiles arg-mismatch evidence")
}

// TestAuditCmd_Integration_IntegrationTestBaseRequiredAnnotations_WrongActiveProfileFiresArgMismatch
// verifies that a BaseIntegrationTest class with @ActiveProfiles("prod") instead
// of @ActiveProfiles("test") triggers exactly one [integration-test-base]
// violation whose evidence contains the arg-mismatch substrings. Backs
// PC01US-009 AC2 (PC01RF-007, PC01RF-009).
func TestAuditCmd_Integration_IntegrationTestBaseRequiredAnnotations_WrongActiveProfileFiresArgMismatch(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us009IntegrationTestBaseRequiredAnnotations", "projectWrongActiveProfile"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForIntegrationTestBaseRequiredAnnotations(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.Contains(t, out, "[integration-test-base]",
		"BaseIntegrationTest with wrong @ActiveProfiles arg must trigger the integration-test-base rule")
	require.Equal(t, 1, strings.Count(out, "[integration-test-base]"),
		"exactly one violation must be reported for the arg-mismatch scenario")
	require.Contains(t, out, "annotation=ActiveProfiles",
		"violation message must contain annotation=ActiveProfiles as required by AC2 (PC01RF-009)")
	require.Contains(t, out, `expected_value="test"`,
		"violation message must contain expected_value=\"test\" as required by AC2 (PC01RF-009)")
	require.Contains(t, out, `actual="prod"`,
		"violation message must contain actual=\"prod\" as required by AC2 (PC01RF-009)")
	require.NotContains(t, out, "missing=",
		"arg-mismatch scenario must not emit missing= evidence (all three annotations are present)")
}

// TestAuditCmd_Integration_IntegrationTestBaseRequiredAnnotations_MissingTestcontainersFiresWithEvidence
// verifies that a BaseIntegrationTest class missing @Testcontainers triggers
// exactly one [integration-test-base] violation whose evidence contains the
// literal substring missing=[Testcontainers]. Backs PC01US-009 AC3
// (PC01RF-001, PC01RF-009).
func TestAuditCmd_Integration_IntegrationTestBaseRequiredAnnotations_MissingTestcontainersFiresWithEvidence(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us009IntegrationTestBaseRequiredAnnotations", "projectMissingTestcontainers"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForIntegrationTestBaseRequiredAnnotations(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.Contains(t, out, "[integration-test-base]",
		"BaseIntegrationTest missing @Testcontainers must trigger the integration-test-base rule")
	require.Equal(t, 1, strings.Count(out, "[integration-test-base]"),
		"exactly one violation must be reported for the missing annotation scenario")
	require.Contains(t, out, "missing=[Testcontainers]",
		"violation message must contain the literal substring missing=[Testcontainers] as required by AC3 (PC01RF-009)")
	require.NotContains(t, out, "annotation=ActiveProfiles",
		"missing-annotation scenario must not emit annotation=ActiveProfiles arg-mismatch evidence")
}
