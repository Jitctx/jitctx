package fsspec_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/infrastructure/fsspec"
)

func TestFinder_Find(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("findsInJitctxPlansFirst", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()
		dir := filepath.Join(tmp, "jitctx-plans")
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "foo.md"), []byte("x"), 0o644))

		finder := fsspec.NewFinder()
		path, content, alts, err := finder.Find(ctx, "foo", tmp, "")

		require.NoError(t, err)
		require.True(t, strings.HasSuffix(path, "/jitctx-plans/foo.md"),
			"expected path to end with /jitctx-plans/foo.md, got %q", path)
		require.Equal(t, []byte("x"), content)
		require.Empty(t, alts)
	})

	t.Run("prefersJitctxPlansOverDotJitctxPlans", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()

		dir1 := filepath.Join(tmp, "jitctx-plans")
		require.NoError(t, os.MkdirAll(dir1, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir1, "foo.md"), []byte("primary"), 0o644))

		dir2 := filepath.Join(tmp, ".jitctx", "plans")
		require.NoError(t, os.MkdirAll(dir2, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir2, "foo.md"), []byte("secondary"), 0o644))

		finder := fsspec.NewFinder()
		path, _, alts, err := finder.Find(ctx, "foo", tmp, "")

		require.NoError(t, err)
		require.True(t, strings.HasSuffix(path, "/jitctx-plans/foo.md"),
			"expected primary path to end with /jitctx-plans/foo.md, got %q", path)
		require.Len(t, alts, 1)
		require.True(t, strings.HasSuffix(alts[0], "/.jitctx/plans/foo.md"),
			"expected alt to end with /.jitctx/plans/foo.md, got %q", alts[0])
	})

	t.Run("fallsBackToDotJitctxPlans", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()
		dir := filepath.Join(tmp, ".jitctx", "plans")
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "foo.md"), []byte("fallback"), 0o644))

		finder := fsspec.NewFinder()
		path, content, alts, err := finder.Find(ctx, "foo", tmp, "")

		require.NoError(t, err)
		require.True(t, strings.HasSuffix(path, "/.jitctx/plans/foo.md"),
			"expected path to end with /.jitctx/plans/foo.md, got %q", path)
		require.Equal(t, []byte("fallback"), content)
		require.Empty(t, alts)
	})

	t.Run("respectsConfiguredPlansDir", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()
		dir := filepath.Join(tmp, "specs", "features")
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "foo.md"), []byte("configured"), 0o644))

		finder := fsspec.NewFinder()
		path, content, alts, err := finder.Find(ctx, "foo", tmp, "specs/features")

		require.NoError(t, err)
		require.True(t, strings.HasSuffix(path, "/specs/features/foo.md"),
			"expected path to end with /specs/features/foo.md, got %q", path)
		require.Equal(t, []byte("configured"), content)
		require.Empty(t, alts)
	})

	t.Run("combinedThreeLocations", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()

		dir1 := filepath.Join(tmp, "jitctx-plans")
		require.NoError(t, os.MkdirAll(dir1, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir1, "foo.md"), []byte("loc1"), 0o644))

		dir2 := filepath.Join(tmp, ".jitctx", "plans")
		require.NoError(t, os.MkdirAll(dir2, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir2, "foo.md"), []byte("loc2"), 0o644))

		dir3 := filepath.Join(tmp, "specs", "features")
		require.NoError(t, os.MkdirAll(dir3, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir3, "foo.md"), []byte("loc3"), 0o644))

		finder := fsspec.NewFinder()
		path, _, alts, err := finder.Find(ctx, "foo", tmp, "specs/features")

		require.NoError(t, err)
		require.True(t, strings.HasSuffix(path, "/jitctx-plans/foo.md"),
			"expected primary to end with /jitctx-plans/foo.md, got %q", path)
		require.Len(t, alts, 2)
		require.True(t, strings.HasSuffix(alts[0], "/.jitctx/plans/foo.md"),
			"expected alts[0] to end with /.jitctx/plans/foo.md, got %q", alts[0])
		require.True(t, strings.HasSuffix(alts[1], "/specs/features/foo.md"),
			"expected alts[1] to end with /specs/features/foo.md, got %q", alts[1])
	})

	t.Run("noneFoundReturnsTypedError", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()

		finder := fsspec.NewFinder()
		_, _, _, err := finder.Find(ctx, "foo", tmp, "")

		require.Error(t, err)

		var sfnf *domerr.SpecFileNotFoundError
		require.True(t, errors.As(err, &sfnf), "expected *SpecFileNotFoundError, got %T: %v", err, err)
		require.Equal(t, "foo", sfnf.Feature)
		require.Len(t, sfnf.Searched, 2, "expected exactly 2 candidate paths when plansDir is empty")
		require.True(t, strings.HasSuffix(sfnf.Searched[0], "/jitctx-plans/foo.md"),
			"searched[0] should end with /jitctx-plans/foo.md, got %q", sfnf.Searched[0])
		require.True(t, strings.HasSuffix(sfnf.Searched[1], "/.jitctx/plans/foo.md"),
			"searched[1] should end with /.jitctx/plans/foo.md, got %q", sfnf.Searched[1])
	})
}
