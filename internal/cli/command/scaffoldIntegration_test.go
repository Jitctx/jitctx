package command_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	appscaffolduc "github.com/jitctx/jitctx/internal/application/usecase/scaffolduc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/domain/service"
	"github.com/jitctx/jitctx/internal/infrastructure/fsscaffold"
	"github.com/jitctx/jitctx/internal/infrastructure/fsspec"
	"github.com/jitctx/jitctx/internal/infrastructure/mdspec"
)

// newScaffoldCmdFor builds a real cobra scaffold command wired with all real
// adapters (no mocks). Returns the command plus captured stdout and stderr buffers.
func newScaffoldCmdFor(t *testing.T, workDir string) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	stderrBuf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(stderrBuf, nil))

	specFinder := fsspec.NewFinder()
	parser := mdspec.New()
	mapper := service.NewContractPathMapper()
	importResolver := service.NewJavaImportResolver(mapper)
	endpointSynth := service.NewEndpointSynthesizer()
	idUtils := service.NewJavaIdentifierUtils()
	registry := fsscaffold.NewRegistry()
	writer := fsscaffold.NewMultiFileWriter()
	realScaffold := appscaffolduc.New(specFinder, parser, mapper, importResolver, endpointSynth, idUtils, registry, writer, logger)

	cmd := command.NewScaffoldCmd(realScaffold, workDir, "", logger)

	stdoutBuf := &bytes.Buffer{}
	cmd.SetOut(stdoutBuf)
	cmd.SetErr(stderrBuf)

	return cmd, stdoutBuf, stderrBuf
}

// writeScaffoldFixture copies testdata/scaffold/createUser/spec.md into
// <workDir>/jitctx-plans/create-user.md.
func writeScaffoldFixture(t *testing.T, workDir string) {
	t.Helper()
	root := findProjectRoot(t)
	src := filepath.Join(root, "testdata", "scaffold", "createUser", "spec.md")
	data, err := os.ReadFile(src)
	require.NoError(t, err, "read scaffold fixture %s", src)

	destDir := filepath.Join(workDir, "jitctx-plans")
	require.NoError(t, os.MkdirAll(destDir, 0o755))
	dest := filepath.Join(destDir, "create-user.md")
	require.NoError(t, os.WriteFile(dest, data, 0o644))
}

// expectedScaffoldPaths returns the four expected generated file paths under workDir.
func expectedScaffoldPaths(workDir string) map[string]string {
	base := filepath.Join(workDir, "src", "main", "java", "com", "app", "user")
	return map[string]string{
		"usecase":    filepath.Join(base, "port", "in", "CreateUserUseCase.java"),
		"repository": filepath.Join(base, "port", "out", "UserRepository.java"),
		"service":    filepath.Join(base, "application", "UserServiceImpl.java"),
		"controller": filepath.Join(base, "adapter", "in", "web", "UserController.java"),
	}
}

func TestScaffoldCmd_Integration_HappyPath_FourContracts(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	writeScaffoldFixture(t, workDir)

	cmd, _, _ := newScaffoldCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "create-user"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	paths := expectedScaffoldPaths(workDir)

	for key, p := range paths {
		_, err := os.Stat(p)
		require.NoError(t, err, "expected generated file for %s at %s", key, p)
	}

	// Read UserServiceImpl.java and assert content.
	content, err := os.ReadFile(paths["service"])
	require.NoError(t, err)
	serviceContent := string(content)

	require.Contains(t, serviceContent, "@Service")
	require.Contains(t, serviceContent, "implements CreateUserUseCase")
	require.Contains(t, serviceContent, "private final UserRepository userRepository")
	require.Contains(t, serviceContent, `throw new UnsupportedOperationException("Not yet implemented")`)
}

func TestScaffoldCmd_Integration_ImportsResolved(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	writeScaffoldFixture(t, workDir)

	cmd, _, _ := newScaffoldCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "create-user"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	paths := expectedScaffoldPaths(workDir)
	content, err := os.ReadFile(paths["controller"])
	require.NoError(t, err)

	require.Contains(t, string(content), "import com.app.user.port.in.CreateUserUseCase;")
}

func TestScaffoldCmd_Integration_AtomicConflict(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	writeScaffoldFixture(t, workDir)

	paths := expectedScaffoldPaths(workDir)

	// Pre-create UserRepository.java to trigger a conflict.
	repoPath := paths["repository"]
	require.NoError(t, os.MkdirAll(filepath.Dir(repoPath), 0o755))
	require.NoError(t, os.WriteFile(repoPath, []byte("// pre-existing\n"), 0o644))

	cmd, _, stderrBuf := newScaffoldCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "create-user"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)

	stderrText := err.Error() + stderrBuf.String()
	require.Contains(t, stderrText, "scaffold aborted")
	require.Contains(t, stderrText, repoPath)

	// Assert NONE of the other 3 expected files exist.
	for key, p := range paths {
		if p == repoPath {
			continue
		}
		_, statErr := os.Stat(p)
		require.True(t, os.IsNotExist(statErr),
			"expected file %s (%s) to not exist after aborted scaffold", key, p)
	}
}

func TestScaffoldCmd_Integration_UnsupportedType(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()

	// Write a fixture with one contract of unsupported Type.
	brokenSpec := `# Feature: broken
Module: x
Package: com.x

## Contract: Foo
Type: weird-thing
`
	destDir := filepath.Join(workDir, "jitctx-plans")
	require.NoError(t, os.MkdirAll(destDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(destDir, "broken.md"), []byte(brokenSpec), 0o644))

	cmd, _, stderrBuf := newScaffoldCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "broken"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)

	stderrText := err.Error() + stderrBuf.String()
	require.Contains(t, stderrText, "unsupported contract type")
	require.Contains(t, stderrText, "weird-thing")

	// Assert src/main/java does NOT exist.
	srcDir := filepath.Join(workDir, "src", "main", "java")
	_, statErr := os.Stat(srcDir)
	require.True(t, os.IsNotExist(statErr), "src/main/java should not exist after unsupported-type error")
}

func TestScaffoldCmd_Integration_DeterministicAcrossRuns(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	writeScaffoldFixture(t, workDir)

	// First run.
	cmd, _, _ := newScaffoldCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "create-user"})
	require.NoError(t, cmd.ExecuteContext(context.Background()))

	// Compute SHA-256 of every generated file.
	hashFiles := func() map[string][sha256.Size]byte {
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

	first := hashFiles()

	// Delete entire src/ directory.
	require.NoError(t, os.RemoveAll(filepath.Join(workDir, "src")))

	// Second run.
	cmd2, _, _ := newScaffoldCmdFor(t, workDir)
	cmd2.SetArgs([]string{"--feature", "create-user"})
	require.NoError(t, cmd2.ExecuteContext(context.Background()))

	second := hashFiles()

	require.Equal(t, first, second, "scaffold output must be byte-equal across runs (RNF-002)")
}

func TestScaffoldCmd_Integration_MissingPackageError(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()

	// Write fixture WITHOUT Package: line.
	noPackageSpec := `# Feature: create-user
Module: user-management

## Contract: Foo
Type: input-port
Methods:
- void hello()
`
	destDir := filepath.Join(workDir, "jitctx-plans")
	require.NoError(t, os.MkdirAll(destDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(destDir, "create-user.md"), []byte(noPackageSpec), 0o644))

	cmd, _, stderrBuf := newScaffoldCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "create-user"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)

	stderrText := err.Error() + stderrBuf.String()
	require.Contains(t, stderrText, "Package:")
	require.Contains(t, stderrText, "Module:")

	// Assert src/main/java does NOT exist.
	srcDir := filepath.Join(workDir, "src", "main", "java")
	_, statErr := os.Stat(srcDir)
	require.True(t, os.IsNotExist(statErr), "src/main/java should not exist after missing-package error")
}

func TestScaffoldCmd_Integration_JSON(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	writeScaffoldFixture(t, workDir)

	cmd, stdoutBuf, _ := newScaffoldCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "create-user", "--format", "json"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdoutBuf.Bytes(), &result))

	require.Equal(t, "create-user", result["feature"])
	require.Equal(t, "user-management", result["module"])
	require.Equal(t, "com.app.user", result["package"])

	written, ok := result["written"].([]any)
	require.True(t, ok, "written must be a JSON array")
	require.Len(t, written, 4)
}
