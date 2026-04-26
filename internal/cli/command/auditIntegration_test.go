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

	"github.com/stretchr/testify/require"

	appaudituc "github.com/jitctx/jitctx/internal/application/usecase/audituc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/domain/service"
	"github.com/jitctx/jitctx/internal/infrastructure/fsconfig"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
	"github.com/jitctx/jitctx/internal/infrastructure/treesitter"
)

// newAuditCmdFor builds a real cobra audit command wired with real infrastructure
// adapters pointing at the given workDir. The profilesDir is resolved from
// <workDir>/.jitctx/profiles so fixtures are fully self-contained.
func newAuditCmdFor(t *testing.T, workDir, manifestPath string) (*bytes.Buffer, *bytes.Buffer, func(args ...string) error) {
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
	bundleLoader := fsprofile.NewBundleLoader(logger)
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
		bundleAuditRulesLoader, // EP04US-004
		resolver,               // EP04US-004
		filepath.Join(workDir, ".jitctx", "profiles"), // EP04US-004
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

// ─── auditClean — golden match ────────────────────────────────────────────────

// TestAuditCmd_Integration_CleanProjectGoldenMatch copies the auditClean
// fixture into a tempdir, runs `jitctx audit`, and asserts the stdout output
// matches testdata/auditClean/expected/report.md byte-for-byte (RNF-003 golden).
func TestAuditCmd_Integration_CleanProjectGoldenMatch(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "auditClean", "project"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdFor(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	expectedPath := fixtureDir(t, "auditClean", "expected", "report.md")
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err)

	require.Equal(t, string(expected), stdout.String(),
		"audit report for clean fixture must match golden byte-for-byte")
}

// ─── auditViolations — golden match ───────────────────────────────────────────

// TestAuditCmd_Integration_ViolationsGoldenMatch copies the auditViolations
// fixture, runs `jitctx audit`, and asserts the stdout matches the golden
// report which contains at least one violation per triggered rule kind.
func TestAuditCmd_Integration_ViolationsGoldenMatch(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "auditViolations", "project"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newAuditCmdFor(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	expectedPath := fixtureDir(t, "auditViolations", "expected", "report.md")
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err)

	require.Equal(t, string(expected), stdout.String(),
		"audit report for violations fixture must match golden byte-for-byte")
}

// ─── Determinism (RNF-003) ────────────────────────────────────────────────────

// TestAuditCmd_Integration_Determinism runs `jitctx audit` twice on the same
// fixture and asserts the stdout is byte-identical on both runs.
func TestAuditCmd_Integration_Determinism(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "auditViolations", "project"), workDir)
	manifestPath := filepath.Join(workDir, "project-state.yaml")

	// First run.
	stdout1, _, run1 := newAuditCmdFor(t, workDir, manifestPath)
	require.NoError(t, run1("--dir", workDir, "--manifest", manifestPath))
	first := stdout1.String()

	// Second run (same workDir, same manifest — nothing changed).
	stdout2, _, run2 := newAuditCmdFor(t, workDir, manifestPath)
	require.NoError(t, run2("--dir", workDir, "--manifest", manifestPath))
	second := stdout2.String()

	require.Equal(t, first, second,
		"audit output must be byte-identical across consecutive runs (RNF-003)")
}

// ─── Read-only guarantee (RNF-002) ────────────────────────────────────────────

// TestAuditCmd_Integration_ReadOnly SHA-256s every source file before and
// after running `jitctx audit` against the violations fixture, then asserts
// all hashes are unchanged (RNF-002: audit must never modify source files).
func TestAuditCmd_Integration_ReadOnly(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "auditViolations", "project"), workDir)
	manifestPath := filepath.Join(workDir, "project-state.yaml")

	// Compute SHA-256 hashes of all source files before the audit.
	hashSourceFiles := func() map[string][sha256.Size]byte {
		t.Helper()
		hashes := make(map[string][sha256.Size]byte)
		srcDir := filepath.Join(workDir, "src")
		err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, walkErr error) error {
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

	before := hashSourceFiles()
	require.NotEmpty(t, before, "fixture must contain at least one source file")

	_, _, run := newAuditCmdFor(t, workDir, manifestPath)
	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	after := hashSourceFiles()

	require.Equal(t, before, after,
		"audit must not modify any source file (RNF-002 read-only guarantee)")
}

// ─── Manifest missing error path ──────────────────────────────────────────────

// TestAuditCmd_Integration_ManifestMissing runs `jitctx audit` against the
// auditMissingManifest fixture (no project-state.yaml) and asserts:
//  1. The command returns an error (non-nil).
//  2. The error message contains "project-state.yaml not found".
//  3. The error message contains "jitctx scan".
//  4. stdout is empty.
func TestAuditCmd_Integration_ManifestMissing(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "auditMissingManifest", "project"), workDir)

	// Point at a manifest that does not exist.
	manifestPath := filepath.Join(workDir, "project-state.yaml")

	stdout, _, run := newAuditCmdFor(t, workDir, manifestPath)

	err := run("--dir", workDir, "--manifest", manifestPath)
	require.Error(t, err, "audit against a missing manifest must return an error")

	msg := err.Error()
	require.Contains(t, msg, "project-state.yaml not found",
		"error must mention the missing manifest file")
	require.Contains(t, msg, "jitctx scan",
		"error must hint the user to run jitctx scan first")

	require.Empty(t, stdout.String(),
		"stdout must be empty when the manifest is missing")
}

// ─── auditDisabledRules — scenario 1: disabled rule golden match ──────────────

// TestAuditCmd_Integration_DisabledRule_GoldenMatch copies the auditDisabledRules
// fixture (which has config.yaml disabling domain-leak) into a tempdir, runs
// `jitctx audit`, and asserts:
//  1. The command returns no error.
//  2. stdout matches the expected/report.md golden byte-for-byte — the disabled
//     rule's violations are absent; only the other rule's violations appear.
//  3. stderr is empty (no unknown-rule warnings expected).
//
// Covers EP03US-005 scenario 1 (disabled rule silently dropped from report).
func TestAuditCmd_Integration_DisabledRule_GoldenMatch(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "auditDisabledRules", "project"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, stderr, run := newAuditCmdFor(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	expectedPath := fixtureDir(t, "auditDisabledRules", "expected", "report.md")
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err)

	require.Equal(t, string(expected), stdout.String(),
		"disabled rule violations must be absent from the audit report (scenario 1)")
	require.Empty(t, stderr.String(),
		"stderr must be empty when all disabled rules are known (no unknown-rule warnings)")
}

// ─── auditDisabledRules — scenario 2: unknown disabled rule warns on stderr ───

// TestAuditCmd_Integration_UnknownDisabledRule_StderrWarning copies the
// auditDisabledRules fixture but overwrites .jitctx/config.yaml with a rule ID
// that does not exist in the active profile. It asserts:
//  1. The command returns no error (unknown rule is a warning, not a failure).
//  2. stderr contains the exact line: unknown audit rule 'made-up-rule'
//  3. stdout contains ALL violations from the fixture (nothing was actually
//     disabled since the unknown rule matched nothing).
//
// Covers EP03US-005 scenario 2 (unknown disabled ID warns but does not fail).
func TestAuditCmd_Integration_UnknownDisabledRule_StderrWarning(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "auditDisabledRules", "project"), workDir)

	// Overwrite the config with an unknown rule ID so nothing is actually disabled.
	configPath := filepath.Join(workDir, ".jitctx", "config.yaml")
	unknownConfig := "audit:\n  disabled_rules:\n    - made-up-rule\n"
	require.NoError(t, os.WriteFile(configPath, []byte(unknownConfig), 0o644))

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, stderr, run := newAuditCmdFor(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath),
		"unknown disabled rule must not cause the audit to fail")

	require.Contains(t, stderr.String(), "unknown audit rule 'made-up-rule'\n",
		"stderr must contain the warning for the unknown disabled rule ID")

	// With no real rule disabled, the report must contain ALL violations from
	// the fixture (both domain-leak and entity-path-mismatch).
	require.Contains(t, stdout.String(), "[domain-leak]",
		"domain-leak violation must appear when the rule was not actually disabled")
	require.Contains(t, stdout.String(), "[entity-path-mismatch]",
		"entity-path-mismatch violation must appear in the full report")
}

// ─── scenario 3: no config file behaves as default ───────────────────────────

// TestAuditCmd_Integration_NoConfig_BehavesAsDefault copies the auditViolations
// fixture (which has no .jitctx/config.yaml) and runs the audit through the
// updated pipeline (config loader present). It asserts:
//  1. The command returns no error.
//  2. stdout matches auditViolations/expected/report.md byte-for-byte — all
//     violations present, nothing suppressed.
//  3. stderr is empty.
//
// Covers EP03US-005 scenario 3 (missing config = empty disabled list = no-op
// filter = backward compatible output).
func TestAuditCmd_Integration_NoConfig_BehavesAsDefault(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "auditViolations", "project"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, stderr, run := newAuditCmdFor(t, workDir, manifestPath)

	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	expectedPath := fixtureDir(t, "auditViolations", "expected", "report.md")
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err)

	require.Equal(t, string(expected), stdout.String(),
		"absent config must produce the same report as if no rules were disabled (scenario 3)")
	require.Empty(t, stderr.String(),
		"stderr must be empty when no config file exists")
}

// ─── RNF-002: read-only guarantee for auditDisabledRules fixture ─────────────

// TestAuditCmd_Integration_ReadOnly_DisabledRules SHA-256s every file in the
// auditDisabledRules project tree before and after running `jitctx audit`, then
// asserts all hashes are unchanged (RNF-002: audit must never modify any file).
func TestAuditCmd_Integration_ReadOnly_DisabledRules(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "auditDisabledRules", "project"), workDir)
	manifestPath := filepath.Join(workDir, "project-state.yaml")

	// hashAllFiles SHA-256s every file under workDir (including config and profiles).
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

	_, _, run := newAuditCmdFor(t, workDir, manifestPath)
	require.NoError(t, run("--dir", workDir, "--manifest", manifestPath))

	after := hashAllFiles()

	require.Equal(t, before, after,
		"audit must not modify any file in the project tree (RNF-002 read-only guarantee)")
}
