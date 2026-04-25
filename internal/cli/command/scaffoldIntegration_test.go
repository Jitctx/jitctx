package command_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"io/fs"
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
	testMapper := service.NewTestPathMapper()
	importResolver := service.NewJavaImportResolver(mapper)
	endpointSynth := service.NewEndpointSynthesizer()
	idUtils := service.NewJavaIdentifierUtils()
	methodParser := service.NewMethodSignatureParser()
	registry := fsscaffold.NewRegistry()
	testRegistry := fsscaffold.NewTestRegistry()
	writer := fsscaffold.NewMultiFileWriter()
	realScaffold := appscaffolduc.New(specFinder, parser, mapper, testMapper, importResolver, endpointSynth, idUtils, methodParser, registry, testRegistry, writer, logger)

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

// expectedScaffoldPaths returns the five expected production file paths under
// workDir and the three expected test file paths. The fixture has:
//   - CreateUserUseCase (input-port)    → src/main/java/.../port/in/
//   - UserRepository   (output-port)   → src/main/java/.../port/out/
//   - UserServiceImpl  (service)        → src/main/java/.../application/
//   - UserController   (rest-adapter)  → src/main/java/.../adapter/in/web/
//   - User             (aggregate-root) → src/main/java/.../domain/
//
// Test stubs (non-interface contracts only):
//   - UserServiceImplTest  → src/test/java/.../application/
//   - UserControllerTest   → src/test/java/.../adapter/in/web/
//   - UserTest             → src/test/java/.../domain/
func expectedScaffoldPaths(workDir string) map[string]string {
	mainBase := filepath.Join(workDir, "src", "main", "java", "com", "app", "user")
	testBase := filepath.Join(workDir, "src", "test", "java", "com", "app", "user")
	return map[string]string{
		"usecase":        filepath.Join(mainBase, "port", "in", "CreateUserUseCase.java"),
		"repository":     filepath.Join(mainBase, "port", "out", "UserRepository.java"),
		"service":        filepath.Join(mainBase, "application", "UserServiceImpl.java"),
		"controller":     filepath.Join(mainBase, "adapter", "in", "web", "UserController.java"),
		"entity":         filepath.Join(mainBase, "domain", "User.java"),
		"serviceTest":    filepath.Join(testBase, "application", "UserServiceImplTest.java"),
		"controllerTest": filepath.Join(testBase, "adapter", "in", "web", "UserControllerTest.java"),
		"entityTest":     filepath.Join(testBase, "domain", "UserTest.java"),
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

	// Assert NONE of the other expected files exist.
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
	// 5 production files + 3 test files = 8 total (fixture: CreateUserUseCase,
	// UserRepository, UserServiceImpl, UserController, User; tests for the
	// three non-interface contracts).
	require.Len(t, written, 8)
}

// ── EP02US-006 integration scenarios ──────────────────────────────────────────

// TestScaffoldCmd_Integration_GeneratesServiceTestStub asserts that the
// scaffold command generates a JUnit 5 + Mockito test stub for UserServiceImpl
// (EP02RF-006, EP02RF-007).
func TestScaffoldCmd_Integration_GeneratesServiceTestStub(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	writeScaffoldFixture(t, workDir)

	cmd, _, _ := newScaffoldCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "create-user"})
	require.NoError(t, cmd.ExecuteContext(context.Background()))

	testPath := filepath.Join(workDir, "src", "test", "java", "com", "app", "user",
		"application", "UserServiceImplTest.java")

	data, err := os.ReadFile(testPath)
	require.NoError(t, err, "UserServiceImplTest.java should exist at %s", testPath)
	content := string(data)

	require.Contains(t, content, "@ExtendWith(MockitoExtension.class)")
	require.Contains(t, content, "@Mock")
	require.Contains(t, content, "private UserRepository userRepository;")
	require.Contains(t, content, "@InjectMocks")
	require.Contains(t, content, "private UserServiceImpl userServiceImpl;")
	require.Contains(t, content, "@Test")
	require.Contains(t, content, "void execute_shouldDoSomething()")
	require.Contains(t, content, "// TODO: implement test")
}

// TestScaffoldCmd_Integration_GeneratesEntityTestStub asserts that the scaffold
// command generates a minimal JUnit 5 test stub for the User aggregate-root
// (EP02RF-006). The stub must NOT contain @Mock or @ExtendWith because
// entity/aggregate tests have no Mockito dependencies.
func TestScaffoldCmd_Integration_GeneratesEntityTestStub(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	writeScaffoldFixture(t, workDir)

	cmd, _, _ := newScaffoldCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "create-user"})
	require.NoError(t, cmd.ExecuteContext(context.Background()))

	testPath := filepath.Join(workDir, "src", "test", "java", "com", "app", "user",
		"domain", "UserTest.java")

	data, err := os.ReadFile(testPath)
	require.NoError(t, err, "UserTest.java should exist at %s", testPath)
	content := string(data)

	require.Contains(t, content, "@Test")
	require.Contains(t, content, "// TODO: implement test")
	require.NotContains(t, content, "@Mock")
	require.NotContains(t, content, "@ExtendWith")
}

// TestScaffoldCmd_Integration_NoTestFileForInterfaceContracts asserts that
// scaffold does NOT generate test stubs for input-port (CreateUserUseCase) and
// output-port (UserRepository) contracts (EP02RF-006 — intentionally non-testable).
func TestScaffoldCmd_Integration_NoTestFileForInterfaceContracts(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	writeScaffoldFixture(t, workDir)

	cmd, _, _ := newScaffoldCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "create-user"})
	require.NoError(t, cmd.ExecuteContext(context.Background()))

	testRoot := filepath.Join(workDir, "src", "test", "java")

	// Walk src/test/java and assert that no file matching the interface
	// contract names appears anywhere in the tree.
	var found []string
	err := filepath.WalkDir(testRoot, func(path string, d fs.DirEntry, walkErr error) error {
		require.NoError(t, walkErr)
		if d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if base == "CreateUserUseCaseTest.java" || base == "UserRepositoryTest.java" {
			found = append(found, path)
		}
		return nil
	})
	require.NoError(t, err)
	require.Empty(t, found,
		"expected no test file for interface contracts; found: %v", found)
}

// TestScaffoldCmd_Integration_AbortsAtomicallyIfTestFileExists asserts that
// scaffold aborts with exit code 1 when a test-side file already exists, and
// that NO production or other test file is written (EP02RF-009 atomicity).
func TestScaffoldCmd_Integration_AbortsAtomicallyIfTestFileExists(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	writeScaffoldFixture(t, workDir)

	// Pre-create only the test file for UserServiceImpl; production files do
	// NOT exist on disk.
	preExistingPath := filepath.Join(workDir, "src", "test", "java", "com", "app", "user",
		"application", "UserServiceImplTest.java")
	require.NoError(t, os.MkdirAll(filepath.Dir(preExistingPath), 0o755))
	require.NoError(t, os.WriteFile(preExistingPath, []byte("// pre-existing test\n"), 0o644))

	cmd, _, stderrBuf := newScaffoldCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "create-user"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err, "scaffold should fail when a test file already exists")

	// stderr must contain the abort message and the pre-existing path.
	stderrText := err.Error() + stderrBuf.String()
	require.Contains(t, stderrText, "scaffold aborted: target files already exist:")
	require.Contains(t, stderrText, preExistingPath)

	// Walk src/main/java — should not exist at all.
	mainJava := filepath.Join(workDir, "src", "main", "java")
	_, statErr := os.Stat(mainJava)
	require.True(t, os.IsNotExist(statErr),
		"src/main/java must not exist after aborted scaffold; conflict guard must be atomic")

	// Walk src/test/java — only the pre-existing file may be present.
	var testFiles []string
	walkErr := filepath.WalkDir(filepath.Join(workDir, "src", "test", "java"),
		func(path string, d fs.DirEntry, we error) error {
			require.NoError(t, we)
			if !d.IsDir() {
				testFiles = append(testFiles, path)
			}
			return nil
		})
	require.NoError(t, walkErr)
	require.Equal(t, []string{preExistingPath}, testFiles,
		"only the pre-existing test file should be present under src/test/java")
}

// TestScaffoldCmd_Integration_TestStubsByteIdenticalAcrossRuns asserts that
// every generated test file has an identical SHA-256 between two independent
// scaffold runs (RNF-002 determinism, extended to test files).
func TestScaffoldCmd_Integration_TestStubsByteIdenticalAcrossRuns(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	writeScaffoldFixture(t, workDir)

	// First successful run.
	cmd, _, _ := newScaffoldCmdFor(t, workDir)
	cmd.SetArgs([]string{"--feature", "create-user"})
	require.NoError(t, cmd.ExecuteContext(context.Background()))

	// hashTestFiles computes SHA-256 for every file under src/test/java.
	hashTestFiles := func() map[string][sha256.Size]byte {
		t.Helper()
		hashes := make(map[string][sha256.Size]byte)
		testRoot := filepath.Join(workDir, "src", "test", "java")
		err := filepath.WalkDir(testRoot, func(path string, d fs.DirEntry, walkErr error) error {
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

	baseline := hashTestFiles()
	require.NotEmpty(t, baseline, "baseline must contain at least one test file")

	// Delete the entire src/ tree (both production and test files) so the
	// second run does not conflict on production files.
	require.NoError(t, os.RemoveAll(filepath.Join(workDir, "src")))

	// Second run.
	cmd2, _, _ := newScaffoldCmdFor(t, workDir)
	cmd2.SetArgs([]string{"--feature", "create-user"})
	require.NoError(t, cmd2.ExecuteContext(context.Background()))

	second := hashTestFiles()

	require.Equal(t, baseline, second,
		"test stubs must be byte-identical across scaffold runs (RNF-002)")
}
