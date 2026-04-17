package fscontext_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/vo"
	"github.com/jitctx/jitctx/internal/infrastructure/fscontext"
)

func TestDiscoverer_WithFrontMatter(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		".jitctx/guidelines/java-conventions.md": &fstest.MapFile{
			Data: []byte("---\ntags: [java, naming, hexagonal]\n---\n# Java Conventions\n"),
		},
	}

	d := fscontext.New()
	ctxs, err := d.DiscoverContexts(context.Background(), fsys)
	require.NoError(t, err)
	require.Len(t, ctxs, 1)
	require.Equal(t, "java-conventions", ctxs[0].ID)
	require.Equal(t, vo.ArtifactGuidelines, ctxs[0].Type)
	require.Contains(t, ctxs[0].Tags, "java")
	require.Contains(t, ctxs[0].Tags, "naming")
	require.Contains(t, ctxs[0].Tags, "hexagonal")
}

func TestDiscoverer_WithoutFrontMatter(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		".jitctx/guidelines/naming/java.md": &fstest.MapFile{
			Data: []byte("# Naming Conventions\n"),
		},
	}

	d := fscontext.New()
	ctxs, err := d.DiscoverContexts(context.Background(), fsys)
	require.NoError(t, err)
	require.Len(t, ctxs, 1)
	require.Equal(t, "java", ctxs[0].ID)
	// Tags should be inferred from path.
	require.Contains(t, ctxs[0].Tags, "naming")
}

func TestDiscoverer_NonMarkdownIgnored(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		".jitctx/guidelines/notes.txt": &fstest.MapFile{
			Data: []byte("some text"),
		},
		".jitctx/guidelines/java-conventions.md": &fstest.MapFile{
			Data: []byte("# Conventions"),
		},
	}

	d := fscontext.New()
	ctxs, err := d.DiscoverContexts(context.Background(), fsys)
	require.NoError(t, err)
	require.Len(t, ctxs, 1)
}

func TestDiscoverer_ReadContextBody(t *testing.T) {
	t.Parallel()

	body := "# Java Conventions\n\nUse camelCase."
	fsys := fstest.MapFS{
		".jitctx/guidelines/java-conventions.md": &fstest.MapFile{
			Data: []byte("---\ntags: [java]\n---\n" + body),
		},
	}

	d := fscontext.New()
	got, err := d.ReadContextBody(context.Background(), fsys, ".jitctx/guidelines/java-conventions.md")
	require.NoError(t, err)
	require.Contains(t, got, "Java Conventions")
	// Front matter should be stripped.
	require.NotContains(t, got, "tags:")
}
