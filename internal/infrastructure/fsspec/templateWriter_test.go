package fsspec_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/infrastructure/fsspec"
)

func TestTemplateWriter_Write_HappyPath(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	target := filepath.Join(tmp, "specs", "foo.md")
	content := []byte("# Feature: foo\nModule: bar\n")

	w := fsspec.NewWriter()
	written, err := w.Write(context.Background(), target, content)

	require.NoError(t, err)
	require.Equal(t, filepath.Clean(target), written)

	got, readErr := os.ReadFile(written)
	require.NoError(t, readErr)
	require.Equal(t, content, got)
}

func TestTemplateWriter_Write_FileAlreadyExists(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	target := filepath.Join(tmp, "existing.md")
	original := []byte("original-content")

	require.NoError(t, os.WriteFile(target, original, 0o644))

	w := fsspec.NewWriter()
	_, err := w.Write(context.Background(), target, []byte("new-content"))

	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrSpecFileExists))

	// Original file must remain unchanged.
	got, readErr := os.ReadFile(target)
	require.NoError(t, readErr)
	require.Equal(t, original, got)
}

func TestTemplateWriter_Write_TypedErrorExtractsPath(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	target := filepath.Join(tmp, "conflict.md")
	require.NoError(t, os.WriteFile(target, []byte("original-content"), 0o644))

	w := fsspec.NewWriter()
	_, err := w.Write(context.Background(), target, []byte("new-content"))

	require.Error(t, err)

	var spefe *domerr.SpecFileExistsError
	require.True(t, errors.As(err, &spefe))
	require.Equal(t, filepath.Clean(target), spefe.Path)
}

func TestTemplateWriter_Write_CtxCancelled(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	target := filepath.Join(tmp, "never.md")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	w := fsspec.NewWriter()
	_, err := w.Write(ctx, target, []byte("content"))

	require.True(t, errors.Is(err, context.Canceled))
}

func TestTemplateWriter_Write_NestedDirCreated(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	target := filepath.Join(tmp, "a", "b", "c", "foo.md")
	content := []byte("nested content")

	w := fsspec.NewWriter()
	written, err := w.Write(context.Background(), target, content)

	require.NoError(t, err)
	require.Equal(t, filepath.Clean(target), written)

	// Verify all 3 intermediate directories were created.
	for _, dir := range []string{
		filepath.Join(tmp, "a"),
		filepath.Join(tmp, "a", "b"),
		filepath.Join(tmp, "a", "b", "c"),
	} {
		info, statErr := os.Stat(dir)
		require.NoError(t, statErr)
		require.True(t, info.IsDir())
	}

	got, readErr := os.ReadFile(written)
	require.NoError(t, readErr)
	require.Equal(t, content, got)
}
