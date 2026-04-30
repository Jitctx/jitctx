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

// newAuditCmdForDomainNoEntityCollection builds a real cobra audit command
// wired with real infrastructure adapters pointing at the given workDir.
// This is a local copy per Q3 resolution (no upstream DRY refactor in this PR).
func newAuditCmdForDomainNoEntityCollection(t *testing.T, workDir, manifestPath string) (*bytes.Buffer, *bytes.Buffer, func(args ...string) error) {
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

// TestAuditCmd_Integration_DomainNoEntityCollection_NonEntityCollection_NoViolation
// verifies that a domain class with only non-entity parameterized types (e.g.
// List<String>) produces no [domain-no-entity-collection] violations.
// Backs PC01US-007 AC1.
func TestAuditCmd_Integration_DomainNoEntityCollection_NonEntityCollection_NoViolation(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us007DomainNoEntityCollection", "projectClean"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForDomainNoEntityCollection(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.NotContains(t, out, "[domain-no-entity-collection]",
		"compliant domain class must not trigger the domain-no-entity-collection rule")
}

// TestAuditCmd_Integration_DomainNoEntityCollection_ListOfEntity_FlagsViolationWithFqnAndPattern
// verifies that a domain class with a List<OrderEntity> field triggers exactly one
// [domain-no-entity-collection] violation containing the FQN evidence substring.
// Backs PC01US-007 AC2.
func TestAuditCmd_Integration_DomainNoEntityCollection_ListOfEntity_FlagsViolationWithFqnAndPattern(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us007DomainNoEntityCollection", "projectListEntity"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForDomainNoEntityCollection(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.Contains(t, out, "[domain-no-entity-collection]",
		"List<OrderEntity> field must trigger the domain-no-entity-collection rule")
	require.Contains(t, out, "type=java.util.List<OrderEntity>, matched_pattern=List<*Entity>",
		"violation message must contain the literal FQN and pattern evidence substring required by AC2")
	require.Contains(t, out, "Order.java:9",
		"violation must report the field's source line")
	require.Equal(t, 1, strings.Count(out, "[domain-no-entity-collection]"),
		"exactly one violation must be reported for a single offending field")
}

// TestAuditCmd_Integration_DomainNoEntityCollection_SetOfEntity_ReportsFieldLine
// verifies that a domain class with a Set<UserEntity> field triggers a
// [domain-no-entity-collection] violation reported on the field's source line.
// Backs PC01US-007 AC3.
func TestAuditCmd_Integration_DomainNoEntityCollection_SetOfEntity_ReportsFieldLine(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us007DomainNoEntityCollection", "projectSetEntity"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdForDomainNoEntityCollection(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	out := stdout.String()
	require.Contains(t, out, "[domain-no-entity-collection]",
		"Set<UserEntity> field must trigger the domain-no-entity-collection rule")
	require.Contains(t, out, "User.java:11",
		"violation must report the field's source line")
	require.Contains(t, out, "type=java.util.Set<UserEntity>",
		"violation message must contain the fully-qualified parameterized type")
	require.Contains(t, out, "matched_pattern=Set<*Entity>",
		"violation message must contain the matched pattern")
}

// TestAuditCmd_Integration_DomainNoEntityCollection_Determinism runs the audit against
// the projectListEntity fixture in two separate temp dirs and asserts byte-identical
// output after normalising the temp-dir path prefix.
// Backs PC01RNF-003 (deterministic output).
func TestAuditCmd_Integration_DomainNoEntityCollection_Determinism(t *testing.T) {
	t.Parallel()

	// First run.
	workDir1 := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us007DomainNoEntityCollection", "projectListEntity"), workDir1)
	manifestPath1 := filepath.Join(workDir1, "project-state.yaml")
	stdout1, _, run1 := newAuditCmdForDomainNoEntityCollection(t, workDir1, manifestPath1)
	require.NoError(t, run1("--dir", workDir1, "--manifest", manifestPath1))

	// Second run (separate temp dir — no shared state).
	workDir2 := t.TempDir()
	copyFixture(t, fixtureDir(t, "pc01us007DomainNoEntityCollection", "projectListEntity"), workDir2)
	manifestPath2 := filepath.Join(workDir2, "project-state.yaml")
	stdout2, _, run2 := newAuditCmdForDomainNoEntityCollection(t, workDir2, manifestPath2)
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
