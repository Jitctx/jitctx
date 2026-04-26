package fsprofile_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
)

func TestBundled_LoadBundled_SpringBootHexagonal(t *testing.T) {
	// t.Parallel cannot be used with t.Chdir (mutates process cwd).
	// t.Chdir changes cwd to an empty temp dir, ensuring the load succeeds
	// purely from the binary embed and not from any file on disk.
	t.Chdir(t.TempDir())

	b := fsprofile.NewBundled()
	bundle, err := b.LoadBundled(context.Background(), "spring-boot-hexagonal")

	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.Equal(t, model.ProfileSourceBundled, bundle.Profile.Source)
	require.Equal(t, "", bundle.Dir)
	require.Equal(t, "spring-boot-hexagonal", bundle.Profile.Name)
}

func TestBundled_ListBundled_ContainsSpringBootHexagonal(t *testing.T) {
	t.Parallel()

	b := fsprofile.NewBundled()
	names, err := b.ListBundled(context.Background())

	require.NoError(t, err)
	require.Contains(t, names, "spring-boot-hexagonal")
}

func TestBundled_LoadBundled_NotFound(t *testing.T) {
	t.Parallel()

	b := fsprofile.NewBundled()
	_, err := b.LoadBundled(context.Background(), "fake")

	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrBundledProfileNotFound))
}

func TestBundled_LoadBundled_PathTraversal(t *testing.T) {
	t.Parallel()

	b := fsprofile.NewBundled()
	_, err := b.LoadBundled(context.Background(), "../../etc/passwd")

	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrProfileInvalid))
}
