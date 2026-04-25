package fsscaffold_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
	"github.com/jitctx/jitctx/internal/infrastructure/fsscaffold"
)

func TestMultiFileWriter_WriteAll(t *testing.T) {
	t.Parallel()

	t.Run("HappyPath", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()
		files := []scaffoldvo.ScaffoldFile{
			{Path: filepath.Join(tmp, "src/main/java/com/app/port/in/CreateUserUseCase.java"), Content: []byte("// CreateUserUseCase\n"), Kind: scaffoldvo.KindProduction},
			{Path: filepath.Join(tmp, "src/main/java/com/app/domain/User.java"), Content: []byte("// User\n"), Kind: scaffoldvo.KindProduction},
			{Path: filepath.Join(tmp, "src/main/java/com/app/service/UserService.java"), Content: []byte("// UserService\n"), Kind: scaffoldvo.KindProduction},
			{Path: filepath.Join(tmp, "src/main/java/com/app/adapter/rest/UserController.java"), Content: []byte("// UserController\n"), Kind: scaffoldvo.KindProduction},
		}

		w := fsscaffold.NewMultiFileWriter()
		written, err := w.WriteAll(context.Background(), files)

		require.NoError(t, err)
		require.Len(t, written, 4)

		// Returned slice must be alphabetically sorted.
		sorted := make([]string, len(written))
		copy(sorted, written)
		sort.Strings(sorted)
		require.Equal(t, sorted, written)

		// Every file must exist on disk with the expected content.
		for _, f := range files {
			got, readErr := os.ReadFile(f.Path)
			require.NoError(t, readErr, "expected file to exist: %s", f.Path)
			require.Equal(t, f.Content, got)
		}
	})

	t.Run("EmptyInput", func(t *testing.T) {
		t.Parallel()

		w := fsscaffold.NewMultiFileWriter()
		written, err := w.WriteAll(context.Background(), []scaffoldvo.ScaffoldFile{})

		require.NoError(t, err)
		require.Equal(t, []string{}, written)
	})

	t.Run("Conflict_PreExistingFile", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()

		conflicting := filepath.Join(tmp, "src/main/java/com/app/domain/User.java")
		require.NoError(t, os.MkdirAll(filepath.Dir(conflicting), 0o755))
		require.NoError(t, os.WriteFile(conflicting, []byte("existing"), 0o644))

		files := []scaffoldvo.ScaffoldFile{
			{Path: filepath.Join(tmp, "src/main/java/com/app/port/in/CreateUserUseCase.java"), Content: []byte("a"), Kind: scaffoldvo.KindProduction},
			{Path: conflicting, Content: []byte("b"), Kind: scaffoldvo.KindProduction},
			{Path: filepath.Join(tmp, "src/main/java/com/app/service/UserService.java"), Content: []byte("c"), Kind: scaffoldvo.KindProduction},
			{Path: filepath.Join(tmp, "src/main/java/com/app/adapter/rest/UserController.java"), Content: []byte("d"), Kind: scaffoldvo.KindProduction},
		}

		w := fsscaffold.NewMultiFileWriter()
		_, err := w.WriteAll(context.Background(), files)

		require.Error(t, err)

		var sce *domerr.ScaffoldConflictError
		require.True(t, errors.As(err, &sce), "expected *ScaffoldConflictError, got %T: %v", err, err)
		require.Equal(t, []string{conflicting}, sce.Conflicts)

		// Non-conflicting files must NOT exist on disk after the call.
		nonConflicting := []string{
			filepath.Join(tmp, "src/main/java/com/app/port/in/CreateUserUseCase.java"),
			filepath.Join(tmp, "src/main/java/com/app/service/UserService.java"),
			filepath.Join(tmp, "src/main/java/com/app/adapter/rest/UserController.java"),
		}
		for _, p := range nonConflicting {
			_, statErr := os.Stat(p)
			require.True(t, os.IsNotExist(statErr), "expected file to NOT exist: %s", p)
		}

		// No .tmp files must have leaked.
		require.NoError(t, filepath.WalkDir(tmp, func(path string, _ os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			require.False(t, filepath.Ext(path) == ".tmp", "leaked .tmp file: %s", path)
			return nil
		}))
	})

	t.Run("Conflict_MultipleConflicts", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()

		conflict1 := filepath.Join(tmp, "src/main/java/com/app/domain/User.java")
		conflict2 := filepath.Join(tmp, "src/main/java/com/app/service/UserService.java")

		for _, p := range []string{conflict1, conflict2} {
			require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
			require.NoError(t, os.WriteFile(p, []byte("existing"), 0o644))
		}

		files := []scaffoldvo.ScaffoldFile{
			{Path: filepath.Join(tmp, "src/main/java/com/app/port/in/CreateUserUseCase.java"), Content: []byte("a"), Kind: scaffoldvo.KindProduction},
			{Path: conflict1, Content: []byte("b"), Kind: scaffoldvo.KindProduction},
			{Path: conflict2, Content: []byte("c"), Kind: scaffoldvo.KindProduction},
			{Path: filepath.Join(tmp, "src/main/java/com/app/adapter/rest/UserController.java"), Content: []byte("d"), Kind: scaffoldvo.KindProduction},
		}

		w := fsscaffold.NewMultiFileWriter()
		_, err := w.WriteAll(context.Background(), files)

		require.Error(t, err)

		var sce *domerr.ScaffoldConflictError
		require.True(t, errors.As(err, &sce))

		wantConflicts := []string{conflict1, conflict2}
		sort.Strings(wantConflicts)
		require.Equal(t, wantConflicts, sce.Conflicts)
	})

	t.Run("Conflict_MixedProdTestOnlyTestExists", func(t *testing.T) {
		t.Parallel()

		// Scenario: batch has one production file and one test file.
		// Only the test path exists on disk. Expect *ScaffoldConflictError
		// whose Conflicts slice contains ONLY the test path.
		tmp := t.TempDir()

		prodPath := filepath.Join(tmp, "src/main/java/com/app/application/UserServiceImpl.java")
		testPath := filepath.Join(tmp, "src/test/java/com/app/application/UserServiceImplTest.java")

		// Pre-create only the test file; production file does NOT exist.
		require.NoError(t, os.MkdirAll(filepath.Dir(testPath), 0o755))
		require.NoError(t, os.WriteFile(testPath, []byte("// pre-existing test"), 0o644))

		files := []scaffoldvo.ScaffoldFile{
			{Path: prodPath, Content: []byte("// UserServiceImpl\n"), Kind: scaffoldvo.KindProduction},
			{Path: testPath, Content: []byte("// UserServiceImplTest\n"), Kind: scaffoldvo.KindTest},
		}

		w := fsscaffold.NewMultiFileWriter()
		_, err := w.WriteAll(context.Background(), files)

		require.Error(t, err)

		var sce *domerr.ScaffoldConflictError
		require.True(t, errors.As(err, &sce), "expected *ScaffoldConflictError, got %T: %v", err, err)
		require.Equal(t, []string{testPath}, sce.Conflicts, "Conflicts must contain only the test path")

		// Production file must NOT have been created.
		_, statErr := os.Stat(prodPath)
		require.True(t, os.IsNotExist(statErr), "production file must not exist after conflict abort")
	})

	t.Run("NestedDirsCreated", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()
		deepPath := filepath.Join(tmp, "a/b/c/d/Foo.java")

		files := []scaffoldvo.ScaffoldFile{
			{Path: deepPath, Content: []byte("// Foo\n"), Kind: scaffoldvo.KindProduction},
		}

		w := fsscaffold.NewMultiFileWriter()
		written, err := w.WriteAll(context.Background(), files)

		require.NoError(t, err)
		require.Len(t, written, 1)

		got, readErr := os.ReadFile(deepPath)
		require.NoError(t, readErr)
		require.Equal(t, []byte("// Foo\n"), got)
	})

	t.Run("DeterministicSortedReturn", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()

		// Provide files in reverse-alphabetic order.
		files := []scaffoldvo.ScaffoldFile{
			{Path: filepath.Join(tmp, "z/Z.java"), Content: []byte("Z"), Kind: scaffoldvo.KindProduction},
			{Path: filepath.Join(tmp, "m/M.java"), Content: []byte("M"), Kind: scaffoldvo.KindProduction},
			{Path: filepath.Join(tmp, "a/A.java"), Content: []byte("A"), Kind: scaffoldvo.KindProduction},
		}

		w := fsscaffold.NewMultiFileWriter()
		written, err := w.WriteAll(context.Background(), files)

		require.NoError(t, err)
		require.Len(t, written, 3)

		sorted := make([]string, len(written))
		copy(sorted, written)
		sort.Strings(sorted)
		require.Equal(t, sorted, written, "returned slice must be alphabetically sorted")
	})

	t.Run("Determinism_SHA256IdenticalAcrossRuns", func(t *testing.T) {
		t.Parallel()

		// RNF-002: write the same merged prod+test batch twice (deleting all
		// files between runs) and assert SHA-256 of every produced file is
		// byte-identical across runs.
		tmp := t.TempDir()

		prodPath := filepath.Join(tmp, "src/main/java/com/app/application/UserServiceImpl.java")
		testPath := filepath.Join(tmp, "src/test/java/com/app/application/UserServiceImplTest.java")

		batch := []scaffoldvo.ScaffoldFile{
			{Path: prodPath, Content: []byte("// UserServiceImpl\npublic class UserServiceImpl {}\n"), Kind: scaffoldvo.KindProduction},
			{Path: testPath, Content: []byte("// UserServiceImplTest\npublic class UserServiceImplTest {}\n"), Kind: scaffoldvo.KindTest},
		}

		hashFiles := func() map[string][sha256.Size]byte {
			result := make(map[string][sha256.Size]byte)
			for _, f := range batch {
				data, err := os.ReadFile(f.Path)
				require.NoError(t, err, "file must exist after write: %s", f.Path)
				result[f.Path] = sha256.Sum256(data)
			}
			return result
		}

		deleteFiles := func() {
			for _, f := range batch {
				_ = os.Remove(f.Path)
			}
		}

		w := fsscaffold.NewMultiFileWriter()

		// First run.
		_, err := w.WriteAll(context.Background(), batch)
		require.NoError(t, err)
		firstHashes := hashFiles()

		deleteFiles()

		// Second run.
		_, err = w.WriteAll(context.Background(), batch)
		require.NoError(t, err)
		secondHashes := hashFiles()

		for path, h1 := range firstHashes {
			h2, ok := secondHashes[path]
			require.True(t, ok, "file missing from second run: %s", path)
			require.Equal(t, h1, h2, "SHA-256 mismatch between runs for file: %s", path)
		}
	})

	t.Run("CtxCancelled", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		files := []scaffoldvo.ScaffoldFile{
			{Path: filepath.Join(tmp, "src/A.java"), Content: []byte("A"), Kind: scaffoldvo.KindProduction},
		}

		w := fsscaffold.NewMultiFileWriter()
		_, err := w.WriteAll(ctx, files)

		require.Error(t, err)
		require.True(t, errors.Is(err, context.Canceled), "expected context.Canceled, got: %v", err)

		// No files should have been written.
		_, statErr := os.Stat(filepath.Join(tmp, "src/A.java"))
		require.True(t, os.IsNotExist(statErr), "expected no file to be written on cancelled context")
	})
}
