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

	apprefactoruc "github.com/jitctx/jitctx/internal/application/usecase/refactoruc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/domain/service"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
	"github.com/jitctx/jitctx/internal/infrastructure/treesitter"
)

// newScanCmdForRefactors wires a real cobra scan command with real infrastructure
// adapters configured for the --refactors branch. The factory passed to NewScanCmd
// uses buildScanFactoryWithLogger so the non-refactors path still compiles and wires.
func newScanCmdForRefactors(t *testing.T, workDir, manifestPath string) (*bytes.Buffer, *bytes.Buffer, func(args ...string) error) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	manifestStore := fsmanifest.New(manifestPath)
	tsParser := treesitter.New()
	tsWalker := treesitter.NewWalker()
	markerParser := service.NewMarkerParser()

	refactorUC := apprefactoruc.New(
		manifestStore,
		tsWalker,
		tsParser,
		markerParser,
		logger,
	)

	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")
	factory := buildScanFactoryWithLogger(profilesDir, logger)

	var stdout, stderr bytes.Buffer
	cmd := command.NewScanCmd(factory, refactorUC, logger)
	cmd.SilenceUsage = true
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	return &stdout, &stderr, func(args ...string) error {
		cmd.SetArgs(args)
		return cmd.ExecuteContext(context.Background())
	}
}

// ─── multiModule — golden match ───────────────────────────────────────────────

// TestScanRefactorsCmd_Integration_MultiModuleGoldenMatch copies the multiModule
// fixture into a tempdir, runs `jitctx scan --refactors`, and asserts stdout
// matches testdata/scanRefactors/multiModule/expected/report.md byte-for-byte.
// Covers Gherkin: "Scanner finds markers across multiple files" and
// "Scanner produces grouped report by module".
func TestScanRefactorsCmd_Integration_MultiModuleGoldenMatch(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "scanRefactors", "multiModule", "project"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newScanCmdForRefactors(t, workDir, manifestPath)

	require.NoError(t, run("--refactors", "--dir", workDir, "--manifest", manifestPath))

	expectedPath := fixtureDir(t, "scanRefactors", "multiModule", "expected", "report.md")
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err)

	require.Equal(t, string(expected), stdout.String(),
		"scan --refactors report for multiModule fixture must match golden byte-for-byte")
}

// ─── unknownAndMalformed ──────────────────────────────────────────────────────

// TestScanRefactorsCmd_Integration_UnknownAndMalformed verifies that:
//   - A comment with an unknown marker type is bucketed as "other" and the
//     original unknown type name is emitted to stderr.
//   - A comment that matches the marker prefix but lacks " - " is classified
//     as "unparseable" and its original text appears in the report.
//
// Covers Gherkin: "Scanner handles unknown marker type" and
// "Scanner handles malformed marker".
func TestScanRefactorsCmd_Integration_UnknownAndMalformed(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "scanRefactors", "unknownAndMalformed", "project"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, stderr, run := newScanCmdForRefactors(t, workDir, manifestPath)

	require.NoError(t, run("--refactors", "--dir", workDir, "--manifest", manifestPath))

	// stderr MUST contain the warning for the unknown type.
	require.Contains(t, stderr.String(), "unknown marker type 'weird-thing'",
		"stderr must warn about unknown marker type")

	// stdout golden match.
	expectedPath := fixtureDir(t, "scanRefactors", "unknownAndMalformed", "expected", "report.md")
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err)

	require.Equal(t, string(expected), stdout.String(),
		"scan --refactors report for unknownAndMalformed fixture must match golden byte-for-byte")
}

// ─── blockAndIgnored ──────────────────────────────────────────────────────────

// TestScanRefactorsCmd_Integration_BlockAndIgnored verifies that:
//   - A block-comment marker is recognised and listed.
//   - Comments that do not start with "TODO(jitctx):" are silently ignored.
//
// Covers Gherkin: "Scanner recognizes block comment markers" and
// "Scanner ignores comments not matching marker prefix".
func TestScanRefactorsCmd_Integration_BlockAndIgnored(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "scanRefactors", "blockAndIgnored", "project"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, _, run := newScanCmdForRefactors(t, workDir, manifestPath)

	require.NoError(t, run("--refactors", "--dir", workDir, "--manifest", manifestPath))

	// stdout golden match — only the block-comment marker must appear.
	expectedPath := fixtureDir(t, "scanRefactors", "blockAndIgnored", "expected", "report.md")
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err)

	require.Equal(t, string(expected), stdout.String(),
		"scan --refactors report for blockAndIgnored fixture must match golden byte-for-byte")
}

// ─── Determinism (RNF-003) ────────────────────────────────────────────────────

// TestScanRefactorsCmd_Integration_Determinism runs `jitctx scan --refactors`
// twice on the multiModule fixture and asserts byte-identical stdout (RNF-003).
func TestScanRefactorsCmd_Integration_Determinism(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "scanRefactors", "multiModule", "project"), workDir)
	manifestPath := filepath.Join(workDir, "project-state.yaml")

	// First run.
	stdout1, _, run1 := newScanCmdForRefactors(t, workDir, manifestPath)
	require.NoError(t, run1("--refactors", "--dir", workDir, "--manifest", manifestPath))
	first := stdout1.String()

	// Second run — same workDir, same manifest, nothing changed.
	stdout2, _, run2 := newScanCmdForRefactors(t, workDir, manifestPath)
	require.NoError(t, run2("--refactors", "--dir", workDir, "--manifest", manifestPath))
	second := stdout2.String()

	require.Equal(t, first, second,
		"scan --refactors output must be byte-identical across consecutive runs (RNF-003)")
}

// ─── Read-only guarantee (RNF-002) ────────────────────────────────────────────

// TestScanRefactorsCmd_Integration_ReadOnly SHA-256s every source file under
// src/ before and after running `jitctx scan --refactors`, then asserts all
// hashes are unchanged (RNF-002: scan --refactors must never modify source files).
func TestScanRefactorsCmd_Integration_ReadOnly(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "scanRefactors", "multiModule", "project"), workDir)
	manifestPath := filepath.Join(workDir, "project-state.yaml")

	// hashSourceFiles computes SHA-256 hashes of every file under src/.
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

	_, _, run := newScanCmdForRefactors(t, workDir, manifestPath)
	require.NoError(t, run("--refactors", "--dir", workDir, "--manifest", manifestPath))

	after := hashSourceFiles()

	require.Equal(t, before, after,
		"scan --refactors must not modify any source file (RNF-002 read-only guarantee)")
}
