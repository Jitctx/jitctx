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

// newAuditCmdForDeterministicViolationOutput builds a real cobra audit command
// wired with real infrastructure adapters pointing at the given workDir.
// Modelled on newAuditCmdForTxDecoratorContract per Q-DRY resolution
// (local copy acceptable; no upstream refactor in this PR).
func newAuditCmdForDeterministicViolationOutput(t *testing.T, workDir, manifestPath string) (*bytes.Buffer, *bytes.Buffer, func(args ...string) error) {
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

// TestAuditCmd_Integration_PC01US013_DeterministicViolationOutput_TwoRunsByteIdentical
// verifies that two consecutive runs of `jitctx audit` over the same fixed
// fixture produce byte-identical stdout (PC01RNF-003). The fixture deliberately
// declares its two required_annotations rules in REVERSE alphabetical order
// (B-... before A-... in YAML) so that the natural emission order from
// evalRequiredAnnotations is B-..., A-..., but the use-case sort comparator
// overrides this to A-..., B-... by ascending RuleID. The indexA < indexB
// assertion therefore catches any regression that removes the sort. Backs
// PC01US-013 AC.
func TestAuditCmd_Integration_PC01US013_DeterministicViolationOutput_TwoRunsByteIdentical(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us013DeterministicOutput", "projectFixed"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")

	// First run: construct a fresh cobra command with its own bytes.Buffer.
	stdout1, _, run1 := newAuditCmdForDeterministicViolationOutput(t, workDir, manifestPath)
	require.NoError(t, run1("--dir", workDir, "--manifest", manifestPath))
	first := stdout1.String()

	// Second run: construct ANOTHER fresh cobra command (separate bytes.Buffer,
	// separate command instance) on the same workDir + manifestPath.
	stdout2, _, run2 := newAuditCmdForDeterministicViolationOutput(t, workDir, manifestPath)
	require.NoError(t, run2("--dir", workDir, "--manifest", manifestPath))
	second := stdout2.String()

	// Assertion 1 (the AC): byte-identical output across consecutive runs (PC01RNF-003).
	require.Equal(t, first, second, "audit output must be byte-identical across consecutive runs (PC01RNF-003)")

	// Assertion 2: both rule-ID tokens are present in stdout, proving the
	// determinism assertion is over a multi-violation slice (not vacuous).
	require.Contains(t, first, "[A-pc01us013-required-service]",
		"stdout must contain the A-pc01us013-required-service rule-ID token")
	require.Contains(t, first, "[B-pc01us013-required-component]",
		"stdout must contain the B-pc01us013-required-component rule-ID token")

	// Assertion 3: indexA < indexB — the comparator's secondary RuleID-ascending
	// tiebreaker fired. The YAML profile declares the rules in REVERSE alphabetical
	// order (B-... listed first, A-... listed second), so the natural emission
	// order from evalRequiredAnnotations would be B-..., A-... without the sort.
	// The use-case sort comparator overrides this to A-..., B-... because
	// (ModuleID, FilePath, Line) agree for both violations and RuleID is the
	// deciding key.
	indexA := strings.Index(first, "[A-pc01us013-required-service]")
	indexB := strings.Index(first, "[B-pc01us013-required-component]")
	require.Less(t, indexA, indexB,
		"[A-pc01us013-required-service] must appear before [B-pc01us013-required-component] in stdout (ascending RuleID tiebreaker)")
}
