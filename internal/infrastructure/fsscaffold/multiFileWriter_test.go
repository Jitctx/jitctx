package fsscaffold_test

import (
	"context"
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
		files := []scaffoldvo.ProductionFile{
			{Path: filepath.Join(tmp, "src/main/java/com/app/port/in/CreateUserUseCase.java"), Content: []byte("// CreateUserUseCase\n")},
			{Path: filepath.Join(tmp, "src/main/java/com/app/domain/User.java"), Content: []byte("// User\n")},
			{Path: filepath.Join(tmp, "src/main/java/com/app/service/UserService.java"), Content: []byte("// UserService\n")},
			{Path: filepath.Join(tmp, "src/main/java/com/app/adapter/rest/UserController.java"), Content: []byte("// UserController\n")},
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
		written, err := w.WriteAll(context.Background(), []scaffoldvo.ProductionFile{})

		require.NoError(t, err)
		require.Equal(t, []string{}, written)
	})

	t.Run("Conflict_PreExistingFile", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()

		conflicting := filepath.Join(tmp, "src/main/java/com/app/domain/User.java")
		require.NoError(t, os.MkdirAll(filepath.Dir(conflicting), 0o755))
		require.NoError(t, os.WriteFile(conflicting, []byte("existing"), 0o644))

		files := []scaffoldvo.ProductionFile{
			{Path: filepath.Join(tmp, "src/main/java/com/app/port/in/CreateUserUseCase.java"), Content: []byte("a")},
			{Path: conflicting, Content: []byte("b")},
			{Path: filepath.Join(tmp, "src/main/java/com/app/service/UserService.java"), Content: []byte("c")},
			{Path: filepath.Join(tmp, "src/main/java/com/app/adapter/rest/UserController.java"), Content: []byte("d")},
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

		files := []scaffoldvo.ProductionFile{
			{Path: filepath.Join(tmp, "src/main/java/com/app/port/in/CreateUserUseCase.java"), Content: []byte("a")},
			{Path: conflict1, Content: []byte("b")},
			{Path: conflict2, Content: []byte("c")},
			{Path: filepath.Join(tmp, "src/main/java/com/app/adapter/rest/UserController.java"), Content: []byte("d")},
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

	t.Run("NestedDirsCreated", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()
		deepPath := filepath.Join(tmp, "a/b/c/d/Foo.java")

		files := []scaffoldvo.ProductionFile{
			{Path: deepPath, Content: []byte("// Foo\n")},
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
		files := []scaffoldvo.ProductionFile{
			{Path: filepath.Join(tmp, "z/Z.java"), Content: []byte("Z")},
			{Path: filepath.Join(tmp, "m/M.java"), Content: []byte("M")},
			{Path: filepath.Join(tmp, "a/A.java"), Content: []byte("A")},
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

	t.Run("CtxCancelled", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		files := []scaffoldvo.ProductionFile{
			{Path: filepath.Join(tmp, "src/A.java"), Content: []byte("A")},
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
