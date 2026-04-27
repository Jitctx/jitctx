package bundledqueries_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/vo"
	tsbundled "github.com/jitctx/jitctx/internal/infrastructure/treesitter/bundledqueries"
)

func nopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewRegistry_DiscoversJava(t *testing.T) {
	t.Parallel()
	r := tsbundled.NewRegistry(nopLogger())
	require.NotNil(t, r)

	supported, err := r.ListSupportedLanguages(context.Background())
	require.NoError(t, err)
	require.True(t, slices.Contains(supported, vo.LanguageJava),
		"expected java in supported list, got %v", supported)
	// sorted
	for i := 1; i < len(supported); i++ {
		require.LessOrEqual(t, supported[i-1], supported[i],
			"supported languages must be sorted (got %v)", supported)
	}
}

func TestLoadLanguageQueries_Java_ReturnsCachedSet(t *testing.T) {
	t.Parallel()
	r := tsbundled.NewRegistry(nopLogger())
	ctx := context.Background()

	set1, err := r.LoadLanguageQueries(ctx, vo.LanguageJava)
	require.NoError(t, err)
	require.NotNil(t, set1)
	require.Equal(t, vo.LanguageJava, set1.Language)
	require.GreaterOrEqual(t, len(set1.Queries), 1,
		"expected at least one bundled .scm file for java")

	set2, err := r.LoadLanguageQueries(ctx, vo.LanguageJava)
	require.NoError(t, err)
	// Pointer equality is the load-bearing invariant for EP04US-005 Scenario 3:
	// two callers asking for the same language receive the SAME pointer, proving
	// the binary holds the queries only once.
	require.Truef(t, set1 == set2,
		"expected pointer equality for cached java set; got %p vs %p", set1, set2)
}

func TestLoadLanguageQueries_UnknownLanguage_ReturnsUnsupportedError(t *testing.T) {
	t.Parallel()
	r := tsbundled.NewRegistry(nopLogger())
	ctx := context.Background()

	_, err := r.LoadLanguageQueries(ctx, vo.Language("cobol"))
	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrLanguageUnsupported))
	require.True(t, errors.Is(err, domerr.ErrProfileInvalid))

	var typed *domerr.LanguageUnsupportedError
	require.True(t, errors.As(err, &typed))
	require.Equal(t, "cobol", typed.Language)
	require.Contains(t, typed.SupportedSorted, "java")
	require.Contains(t, err.Error(), "language 'cobol' is not supported")
	require.Contains(t, err.Error(), "java")
}

func TestLoadLanguageQueries_RecognisedButUnembedded(t *testing.T) {
	t.Parallel()
	r := tsbundled.NewRegistry(nopLogger())
	ctx := context.Background()

	// vo.LanguageGo is a recognised constant but the embed root has no
	// go/ subdirectory in US-005 — same failure mode as an unrecognised id.
	_, err := r.LoadLanguageQueries(ctx, vo.LanguageGo)
	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrLanguageUnsupported))

	var typed *domerr.LanguageUnsupportedError
	require.True(t, errors.As(err, &typed))
	require.Equal(t, "go", typed.Language)
}

func TestLoadLanguageQueries_ContextCancelled(t *testing.T) {
	t.Parallel()
	r := tsbundled.NewRegistry(nopLogger())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := r.LoadLanguageQueries(ctx, vo.LanguageJava)
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled))
}

func TestNewRegistry_NilLogger_DoesNotPanic(t *testing.T) {
	t.Parallel()
	require.NotPanics(t, func() {
		r := tsbundled.NewRegistry(nil)
		require.NotNil(t, r)
	})
}
