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

// newAuditCmdForTxDecoratorContract builds a real cobra audit command wired
// with real infrastructure adapters pointing at the given workDir.
// Modelled on newAuditCmdForIntegrationTestBaseRequiredAnnotations per Q-DRY
// resolution (local copy acceptable; no upstream refactor in this PR).
func newAuditCmdForTxDecoratorContract(t *testing.T, workDir, manifestPath string) (*bytes.Buffer, *bytes.Buffer, func(args ...string) error) {
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

// TestAuditCmd_Integration_TxDecoratorContract_PrimaryAndQualifierWithNonEmptyValuePass
// verifies that a service class with both @Transactional and a non-empty
// @Qualifier annotation produces zero [tx-decorator-contract] violations.
// Backs PC01US-010 AC1.
func TestAuditCmd_Integration_TxDecoratorContract_PrimaryAndQualifierWithNonEmptyValuePass(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us010TxDecoratorContract", "projectClean"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForTxDecoratorContract(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.NotContains(t, out, "[tx-decorator-contract]",
		"compliant service with @Transactional and non-empty @Qualifier must not trigger the tx-decorator-contract rule")
	require.NotContains(t, out, "missing=",
		"clean project must not emit any missing= evidence")
	require.NotContains(t, out, "value=empty",
		"clean project must not emit any value=empty evidence")
}

// TestAuditCmd_Integration_TxDecoratorContract_PrimaryOnlyEmitsMissingQualifier
// verifies that a service class with @Transactional but no @Qualifier
// triggers exactly one [tx-decorator-contract] violation whose evidence
// contains the literal substring missing=[Qualifier]. Backs PC01US-010 AC2.
func TestAuditCmd_Integration_TxDecoratorContract_PrimaryOnlyEmitsMissingQualifier(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us010TxDecoratorContract", "projectMissingQualifier"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForTxDecoratorContract(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.Contains(t, out, "[tx-decorator-contract]",
		"service with @Transactional but no @Qualifier must trigger the tx-decorator-contract rule")
	require.Equal(t, 1, strings.Count(out, "[tx-decorator-contract]"),
		"exactly one violation must be reported for the missing-qualifier scenario")
	require.Contains(t, out, "missing=[Qualifier]",
		"violation message must contain the literal substring missing=[Qualifier] as required by AC2")
	require.NotContains(t, out, "value=empty",
		"missing-qualifier scenario must not emit value=empty evidence (short-circuited by missing-violation path)")
}

// TestAuditCmd_Integration_TxDecoratorContract_EmptyQualifierValueEmitsNonEmptyEvidence
// verifies that a service class with @Transactional and @Qualifier("") (empty
// value) triggers exactly one [tx-decorator-contract] violation whose evidence
// contains the literal substring annotation=Qualifier, value=empty,
// expected=non-empty. Backs PC01US-010 AC3.
func TestAuditCmd_Integration_TxDecoratorContract_EmptyQualifierValueEmitsNonEmptyEvidence(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us010TxDecoratorContract", "projectEmptyQualifier"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForTxDecoratorContract(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.Contains(t, out, "[tx-decorator-contract]",
		"service with @Transactional and empty @Qualifier value must trigger the tx-decorator-contract rule")
	require.Equal(t, 1, strings.Count(out, "[tx-decorator-contract]"),
		"exactly one violation must be reported for the empty-qualifier-value scenario")
	require.Contains(t, out, "annotation=Qualifier, value=empty, expected=non-empty",
		"violation message must contain the literal substring annotation=Qualifier, value=empty, expected=non-empty as required by AC3")
	require.NotContains(t, out, "missing=",
		"empty-qualifier-value scenario must not emit missing= evidence (both required annotations are present)")
}
