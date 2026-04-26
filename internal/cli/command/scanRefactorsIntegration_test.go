package command_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	apprefactoruc "github.com/jitctx/jitctx/internal/application/usecase/refactoruc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/domain/service"
	"github.com/jitctx/jitctx/internal/infrastructure/fsgit"
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
	gitReader := fsgit.New(logger)

	refactorUC := apprefactoruc.New(
		manifestStore,
		tsWalker,
		tsParser,
		gitReader,
		gitReader.LineReader(),
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

// ─── Stale skipped note ───────────────────────────────────────────────────────

// TestScanRefactorsCmd_Integration_StaleSkippedNote verifies that when no .git
// directory is present, stdout matches the existing golden AND stderr contains
// the "git not detected, stale flag skipped" note.
func TestScanRefactorsCmd_Integration_StaleSkippedNote(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "scanRefactors", "multiModule", "project"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	stdout, stderr, run := newScanCmdForRefactors(t, workDir, manifestPath)

	require.NoError(t, run("--refactors", "--dir", workDir, "--manifest", manifestPath))

	// stdout must still match the existing golden.
	expectedPath := fixtureDir(t, "scanRefactors", "multiModule", "expected", "report.md")
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	require.Equal(t, string(expected), stdout.String(),
		"stdout must match multiModule golden when git is absent")

	// stderr must contain the stale-skipped note.
	require.Contains(t, stderr.String(), "git not detected, stale flag skipped",
		"stderr must contain stale-skipped note when .git is absent")
}

// ─── Synthetic git repo helpers ───────────────────────────────────────────────

// runGit runs a git command inside repo and fails the test on error.
func runGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, string(out))
}

// writeFileAndCommit writes content to relPath inside repo, stages it, and
// commits with the given ISO-8601 date (used for both author and committer).
func writeFileAndCommit(t *testing.T, repo, relPath, content, isoDate string) {
	t.Helper()
	path := filepath.Join(repo, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	runGit(t, repo, "add", relPath)
	cmd := exec.Command("git", "-C", repo, "commit", "-m", "seed", "--date="+isoDate)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE="+isoDate,
		"GIT_COMMITTER_DATE="+isoDate,
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

// buildSyntheticGitRepo creates a minimal git repository inside repo with:
//   - A Java file at src/main/java/com/app/Foo.java whose marker line was
//     introduced 30 days ago and whose non-marker line was last modified 5 days ago.
//   - A project-state.yaml declaring one module "app" covering "src/main/java/com/app".
//   - A .jitctx/profiles/spring-boot-hexagonal.yaml (copied from multiModule fixture).
//
// Returns the absolute path to the repo.
func buildSyntheticGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()

	// Initialise git.
	cmd := exec.Command("git", "init", repo)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test")

	oldDate := "2026-03-26T00:00:00+00:00" // ~30 days ago relative to 2026-04-25
	newDate := "2026-04-20T00:00:00+00:00" // ~5 days ago

	// First commit: the Java file with the marker comment.
	javaContent := `package com.app;
// TODO(jitctx): rename - rename foo to handleFoo
public class Foo { public void foo() {} }
`
	writeFileAndCommit(t, repo, "src/main/java/com/app/Foo.java", javaContent, oldDate)

	// Write project-state.yaml and profile in the first commit as well.
	manifest := `generated_at: 2026-04-25T00:00:00Z
stack:
  languages:
    - java
  frameworks:
    - spring-boot-hexagonal
modules:
  - id: app
    path: src/main/java/com/app
    tags: []
    contracts: []
    dependencies: []
contexts: []
`
	writeFileAndCommit(t, repo, "project-state.yaml", manifest, oldDate)

	// Copy the profile from the multiModule fixture.
	profileSrc := fixtureDir(t, "scanRefactors", "multiModule", "project", ".jitctx", "profiles", "spring-boot-hexagonal.yaml")
	profileData, err := os.ReadFile(profileSrc)
	require.NoError(t, err)
	profileDest := filepath.Join(repo, ".jitctx", "profiles", "spring-boot-hexagonal.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(profileDest), 0o755))
	require.NoError(t, os.WriteFile(profileDest, profileData, 0o644))
	runGit(t, repo, "add", ".jitctx/profiles/spring-boot-hexagonal.yaml")
	commitCmd := exec.Command("git", "-C", repo, "commit", "-m", "add profile", "--date="+oldDate)
	commitCmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE="+oldDate,
		"GIT_COMMITTER_DATE="+oldDate,
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	commitOut, commitErr := commitCmd.CombinedOutput()
	require.NoError(t, commitErr, string(commitOut))

	// Second commit: modify a non-marker line (the class body) to advance file mtime.
	javaContentV2 := `package com.app;
// TODO(jitctx): rename - rename foo to handleFoo
public class Foo { public void foo() { /* updated */ } }
`
	writeFileAndCommit(t, repo, "src/main/java/com/app/Foo.java", javaContentV2, newDate)

	return repo
}

// ─── StaleFlaggedWithGit ──────────────────────────────────────────────────────

// TestScanRefactorsCmd_Integration_StaleFlaggedWithGit builds a synthetic git
// repo where the file was modified after the marker was introduced, then asserts
// the report contains "(stale candidate)" and stderr does NOT contain the
// stale-skipped note.
func TestScanRefactorsCmd_Integration_StaleFlaggedWithGit(t *testing.T) {
	t.Parallel()

	repo := buildSyntheticGitRepo(t)
	manifestPath := filepath.Join(repo, "project-state.yaml")
	stdout, stderr, run := newScanCmdForRefactors(t, repo, manifestPath)

	require.NoError(t, run("--refactors", "--dir", repo, "--manifest", manifestPath))

	expectedPath := fixtureDir(t, "scanRefactors", "staleGitRepo", "expected", "report.md")
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err)

	require.Equal(t, string(expected), stdout.String(),
		"scan --refactors report for staleGitRepo must match golden byte-for-byte")
	require.NotContains(t, stderr.String(), "git not detected, stale flag skipped",
		"stderr must NOT contain stale-skipped note when .git is present")
}

// ─── StaleNotFlaggedOnUnmodified ──────────────────────────────────────────────

// TestScanRefactorsCmd_Integration_StaleNotFlaggedOnUnmodified uses a synthetic
// git repo with only one commit (file never modified after marker introduction),
// asserts the marker does NOT have the "(stale candidate)" suffix.
func TestScanRefactorsCmd_Integration_StaleNotFlaggedOnUnmodified(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	cmd := exec.Command("git", "init", repo)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test")

	singleDate := "2026-03-26T00:00:00+00:00"

	javaContent := `package com.app;
// TODO(jitctx): rename - rename foo to handleFoo
public class Foo { public void foo() {} }
`
	writeFileAndCommit(t, repo, "src/main/java/com/app/Foo.java", javaContent, singleDate)

	manifest := `generated_at: 2026-04-25T00:00:00Z
stack:
  languages:
    - java
  frameworks:
    - spring-boot-hexagonal
modules:
  - id: app
    path: src/main/java/com/app
    tags: []
    contracts: []
    dependencies: []
contexts: []
`
	writeFileAndCommit(t, repo, "project-state.yaml", manifest, singleDate)

	profileSrc := fixtureDir(t, "scanRefactors", "multiModule", "project", ".jitctx", "profiles", "spring-boot-hexagonal.yaml")
	profileData, err := os.ReadFile(profileSrc)
	require.NoError(t, err)
	profileDest := filepath.Join(repo, ".jitctx", "profiles", "spring-boot-hexagonal.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(profileDest), 0o755))
	require.NoError(t, os.WriteFile(profileDest, profileData, 0o644))
	runGit(t, repo, "add", ".jitctx/profiles/spring-boot-hexagonal.yaml")
	profileCommit := exec.Command("git", "-C", repo, "commit", "-m", "add profile", "--date="+singleDate)
	profileCommit.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE="+singleDate,
		"GIT_COMMITTER_DATE="+singleDate,
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	profileOut, profileErr := profileCommit.CombinedOutput()
	require.NoError(t, profileErr, string(profileOut))

	manifestPath := filepath.Join(repo, "project-state.yaml")
	stdout, stderr, run := newScanCmdForRefactors(t, repo, manifestPath)

	require.NoError(t, run("--refactors", "--dir", repo, "--manifest", manifestPath))

	require.NotContains(t, stdout.String(), "(stale candidate)",
		"marker must NOT be stale when file has not been modified after line introduction")
	require.NotContains(t, stderr.String(), "git not detected, stale flag skipped",
		"stderr must NOT contain stale-skipped note when .git is present")
}

// ─── StaleReadOnly (RNF-002) ──────────────────────────────────────────────────

// TestScanRefactorsCmd_Integration_StaleReadOnly SHA-256s every non-.git file
// in the synthetic repo before and after the run; asserts all hashes are unchanged.
func TestScanRefactorsCmd_Integration_StaleReadOnly(t *testing.T) {
	t.Parallel()

	repo := buildSyntheticGitRepo(t)
	manifestPath := filepath.Join(repo, "project-state.yaml")

	hashNonGitFiles := func() map[string][sha256.Size]byte {
		t.Helper()
		hashes := make(map[string][sha256.Size]byte)
		err := filepath.WalkDir(repo, func(path string, d os.DirEntry, walkErr error) error {
			require.NoError(t, walkErr)
			// Skip the .git directory to avoid nondeterminism from git index updates.
			if d.IsDir() && d.Name() == ".git" {
				return filepath.SkipDir
			}
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

	before := hashNonGitFiles()
	require.NotEmpty(t, before, "synthetic repo must contain at least one non-.git file")

	_, _, run := newScanCmdForRefactors(t, repo, manifestPath)
	require.NoError(t, run("--refactors", "--dir", repo, "--manifest", manifestPath))

	after := hashNonGitFiles()

	require.Equal(t, before, after,
		"scan --refactors must not modify any file in the synthetic git repo (RNF-002)")
}

// ─── StaleDeterminism (RNF-003) ───────────────────────────────────────────────

// TestScanRefactorsCmd_Integration_StaleDeterminism runs `jitctx scan --refactors`
// twice on the synthetic git repo and asserts byte-identical stdout (RNF-003).
func TestScanRefactorsCmd_Integration_StaleDeterminism(t *testing.T) {
	t.Parallel()

	repo := buildSyntheticGitRepo(t)
	manifestPath := filepath.Join(repo, "project-state.yaml")

	stdout1, _, run1 := newScanCmdForRefactors(t, repo, manifestPath)
	require.NoError(t, run1("--refactors", "--dir", repo, "--manifest", manifestPath))
	first := stdout1.String()

	stdout2, _, run2 := newScanCmdForRefactors(t, repo, manifestPath)
	require.NoError(t, run2("--refactors", "--dir", repo, "--manifest", manifestPath))
	second := stdout2.String()

	require.Equal(t, first, second,
		"scan --refactors output must be byte-identical across consecutive runs on a git repo (RNF-003)")
}
