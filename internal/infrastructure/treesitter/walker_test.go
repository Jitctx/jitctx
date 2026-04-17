package treesitter_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/infrastructure/treesitter"
)

func TestWalker_OnlyJavaUnderSrcMain(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"src/main/java/com/app/User.java":           &fstest.MapFile{},
		"src/main/java/com/app/Service.java":        &fstest.MapFile{},
		"src/test/java/com/app/UserTest.java":       &fstest.MapFile{},
		"src/main/java/com/app/util/Helper.java":    &fstest.MapFile{},
		"src/main/resources/application.properties": &fstest.MapFile{},
	}

	w := treesitter.NewWalker()
	files, err := w.WalkJavaFiles(context.Background(), fsys)
	require.NoError(t, err)
	require.Len(t, files, 3)
	// Should NOT include test file.
	for _, f := range files {
		require.NotContains(t, f, "src/test")
	}
}

func TestWalker_SortedOutput(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"src/main/java/com/app/Z.java": &fstest.MapFile{},
		"src/main/java/com/app/A.java": &fstest.MapFile{},
		"src/main/java/com/app/M.java": &fstest.MapFile{},
	}

	w := treesitter.NewWalker()
	files, err := w.WalkJavaFiles(context.Background(), fsys)
	require.NoError(t, err)
	require.Len(t, files, 3)
	require.Equal(t, "src/main/java/com/app/A.java", files[0])
	require.Equal(t, "src/main/java/com/app/M.java", files[1])
	require.Equal(t, "src/main/java/com/app/Z.java", files[2])
}

func TestWalker_EmptyWhenNoJavaRoot(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"README.md": &fstest.MapFile{},
	}

	w := treesitter.NewWalker()
	files, err := w.WalkJavaFiles(context.Background(), fsys)
	require.NoError(t, err)
	require.Empty(t, files)
}
