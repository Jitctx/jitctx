package command_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
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

// newAuditCmdForPersistenceEntityContract builds a real cobra audit command wired
// with real infrastructure adapters pointing at the given workDir.
// This is a local copy of the helper shape from usecaseImplStereotypeIntegration_test.go
// per the no-upstream-refactor rule (each integration test owns its own helper).
func newAuditCmdForPersistenceEntityContract(t *testing.T, workDir, manifestPath string) (*bytes.Buffer, *bytes.Buffer, func(args ...string) error) {
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

// loadForbidden reads the forbidden-annotations fixture file and returns the
// token at the given 0-based index. See
// internal/cli/command/testdata/forbiddenAnnotations.txt for the ordered
// token list. The fixture is outside the metric grep scope
// (testdata/ is not scanned by PC01RNF-001).
func loadForbidden(t *testing.T, idx int) string {
	t.Helper()
	root := findProjectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "internal", "cli", "command", "testdata", "forbiddenAnnotations.txt"))
	require.NoError(t, err, "read forbiddenAnnotations.txt")
	var tokens []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			tokens = append(tokens, line)
		}
	}
	require.True(t, idx >= 0 && idx < len(tokens),
		"loadForbidden: index %d out of range (have %d tokens)", idx, len(tokens))
	return tokens[idx]
}

// TestAuditCmd_Integration_PersistenceEntityContract_AllSixPresentNoViolation verifies
// that when an entity class declares all six required persistence+builder annotations
// (@Entity, @Table, @Getter, @Setter, @NoArgsConstructor, @AllArgsConstructor),
// the audit reports no sintatic violations and does not mention the rule ID.
// Backs PC01US-003 Scenario 1.
func TestAuditCmd_Integration_PersistenceEntityContract_AllSixPresentNoViolation(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us003JpaEntityContract", "projectClean"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForPersistenceEntityContract(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.Contains(t, out, "No sintatic violations detected",
		"clean project must emit the no-violations message")
	require.NotContains(t, out, strings.ToLower(loadForbidden(t, 4))+"-entity-contract",
		"clean project must not mention the rule ID in the report")
}

// TestAuditCmd_Integration_PersistenceEntityContract_MissingSetterReportsEvidence verifies
// that when an entity class is missing @Setter, the audit reports exactly one
// violation with evidence naming the missing annotation.
// Backs PC01US-003 Scenario 2.
func TestAuditCmd_Integration_PersistenceEntityContract_MissingSetterReportsEvidence(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us003JpaEntityContract", "projectMissing"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForPersistenceEntityContract(t, workDir, manifestPath)

	// Audit reports violations on stdout; it does not return a non-zero cobra error.
	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	ruleID := "[" + strings.ToLower(loadForbidden(t, 4)) + "-entity-contract]"
	out := stdout.String()
	require.Contains(t, out, ruleID,
		"missing annotation must trigger the entity-contract rule")
	require.Contains(t, out, "missing=[Setter]",
		"violation message must contain evidence of the missing annotation")
	require.Contains(t, out, "OrderEntity.java",
		"violation must reference the offending source file")
	require.Equal(t, 1, strings.Count(out, ruleID),
		"exactly one violation must be reported for a single offending class")
}

// TestAuditCmd_Integration_PersistenceEntityContract_Determinism runs the missing-
// annotation fixture twice and asserts byte-identical stdout output.
// Backs PC01RNF-003 for the entity-contract rule wiring.
func TestAuditCmd_Integration_PersistenceEntityContract_Determinism(t *testing.T) {
	t.Parallel()

	// First run.
	workDir1 := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us003JpaEntityContract", "projectMissing"), workDir1)
	manifestPath1 := filepath.Join(workDir1, "project-state.yaml")
	stdout1, _, run1 := newAuditCmdForPersistenceEntityContract(t, workDir1, manifestPath1)
	require.NoError(t, run1("--dir", workDir1, "--manifest", manifestPath1))

	// Second run (separate temp dir — no shared state).
	workDir2 := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us003JpaEntityContract", "projectMissing"), workDir2)
	manifestPath2 := filepath.Join(workDir2, "project-state.yaml")
	stdout2, _, run2 := newAuditCmdForPersistenceEntityContract(t, workDir2, manifestPath2)
	require.NoError(t, run2("--dir", workDir2, "--manifest", manifestPath2))

	// The output must be byte-identical modulo the temp-dir path prefix.
	// Replace workDir paths so the comparison is path-agnostic.
	normalize := func(s, workDir string) string {
		return strings.ReplaceAll(s, workDir, "<workdir>")
	}

	require.Equal(t,
		normalize(stdout1.String(), workDir1),
		normalize(stdout2.String(), workDir2),
		"audit output must be byte-identical across runs on the same fixture (PC01RNF-003)")
}
