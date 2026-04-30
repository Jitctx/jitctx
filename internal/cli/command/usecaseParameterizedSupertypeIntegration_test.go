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

// newAuditCmdForUseCaseParameterizedSupertype builds a real cobra audit command
// wired with real infrastructure adapters pointing at the given workDir.
// Modelled on newAuditCmdForDomainNoEntityCollection per Q-DRY resolution
// (local copy acceptable; no upstream refactor in this PR).
func newAuditCmdForUseCaseParameterizedSupertype(t *testing.T, workDir, manifestPath string) (*bytes.Buffer, *bytes.Buffer, func(args ...string) error) {
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

// TestAuditCmd_Integration_UseCaseParameterizedSupertype_MatchingSupertypePasses
// verifies that a use-case class correctly implementing the parameterized
// supertype (e.g. implements UseCase<String, User>) produces no
// [usecase-supertype] violation. Backs PC01US-008 AC1.
func TestAuditCmd_Integration_UseCaseParameterizedSupertype_MatchingSupertypePasses(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us008UseCaseParameterizedSupertype", "projectClean"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForUseCaseParameterizedSupertype(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.NotContains(t, out, "[usecase-supertype]",
		"compliant use-case class with correct parameterized supertype must not trigger the usecase-supertype rule")
}

// TestAuditCmd_Integration_UseCaseParameterizedSupertype_NoSupertypeFiresActualNone
// verifies that a use-case class with no implements clause triggers exactly one
// [usecase-supertype] violation containing the evidence substring
// "expected_supertype=UseCase<*,*>, actual=none". Backs PC01US-008 AC2 (PC01RF-009).
func TestAuditCmd_Integration_UseCaseParameterizedSupertype_NoSupertypeFiresActualNone(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us008UseCaseParameterizedSupertype", "projectMissingSupertype"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForUseCaseParameterizedSupertype(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.Contains(t, out, "[usecase-supertype]",
		"use-case class with no parameterized supertype must trigger the usecase-supertype rule")
	require.Contains(t, out, "expected_supertype=UseCase<*,*>, actual=none",
		"violation message must contain the literal AC2 evidence substring required by PC01RF-009")
	require.Equal(t, 1, strings.Count(out, "[usecase-supertype]"),
		"exactly one violation must be reported for a single offending class")
}

// TestAuditCmd_Integration_UseCaseParameterizedSupertype_WrongArityFiresWithEvidence
// verifies that a use-case class implementing UseCase with wrong arity
// (e.g. implements UseCase<String>) triggers exactly one [usecase-supertype]
// violation containing arity evidence substrings. Backs PC01US-008 AC3 (PC01RF-009).
func TestAuditCmd_Integration_UseCaseParameterizedSupertype_WrongArityFiresWithEvidence(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us008UseCaseParameterizedSupertype", "projectWrongArity"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForUseCaseParameterizedSupertype(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.Contains(t, out, "[usecase-supertype]",
		"use-case class with wrong supertype arity must trigger the usecase-supertype rule")
	require.Contains(t, out, "expected_arity=2",
		"violation message must contain expected_arity=2 as required by AC3 (PC01RF-009)")
	require.Contains(t, out, "actual_arity=1",
		"violation message must contain actual_arity=1 as required by AC3 (PC01RF-009)")
	require.Equal(t, 1, strings.Count(out, "[usecase-supertype]"),
		"exactly one violation must be reported for a single offending class")
}
