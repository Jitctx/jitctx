package fsspec_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/infrastructure/fsspec"
)

func TestRender(t *testing.T) {
	t.Parallel()

	t.Run("contains-feature-header-line", func(t *testing.T) {
		t.Parallel()

		r := fsspec.New()
		out, err := r.Render(context.Background(), "create-user", "user-management")
		require.NoError(t, err)

		content := string(out)
		require.True(t,
			strings.Contains(content, "# Feature: create-user\n"),
			"expected exact line '# Feature: create-user' in rendered output",
		)
	})

	t.Run("contains-module-line", func(t *testing.T) {
		t.Parallel()

		r := fsspec.New()
		out, err := r.Render(context.Background(), "create-user", "user-management")
		require.NoError(t, err)

		content := string(out)
		require.True(t,
			strings.Contains(content, "Module: user-management\n"),
			"expected exact line 'Module: user-management' in rendered output",
		)
	})

	t.Run("contains-exactly-three-contract-blocks", func(t *testing.T) {
		t.Parallel()

		r := fsspec.New()
		out, err := r.Render(context.Background(), "create-user", "user-management")
		require.NoError(t, err)

		count := strings.Count(string(out), "## Contract: <Name>")
		require.Equal(t, 3, count,
			"expected exactly 3 occurrences of '## Contract: <Name>'",
		)
	})

	t.Run("contains-required-type-values", func(t *testing.T) {
		t.Parallel()

		r := fsspec.New()
		out, err := r.Render(context.Background(), "create-user", "user-management")
		require.NoError(t, err)

		content := string(out)
		require.Contains(t, content, "Type: input-port")
		require.Contains(t, content, "Type: service")
		require.Contains(t, content, "Type: rest-adapter")
	})

	t.Run("contains-at-least-three-todo-markers", func(t *testing.T) {
		t.Parallel()

		r := fsspec.New()
		out, err := r.Render(context.Background(), "create-user", "user-management")
		require.NoError(t, err)

		count := strings.Count(string(out), "<TODO>")
		require.GreaterOrEqual(t, count, 3,
			"expected at least 3 occurrences of '<TODO>'",
		)
	})

	t.Run("deterministic-byte-identical-output", func(t *testing.T) {
		t.Parallel()

		r := fsspec.New()
		first, err := r.Render(context.Background(), "create-user", "user-management")
		require.NoError(t, err)

		second, err := r.Render(context.Background(), "create-user", "user-management")
		require.NoError(t, err)

		require.Equal(t, first, second,
			"two consecutive Render calls must produce byte-identical output",
		)
	})

	t.Run("cancelled-ctx-returns-context-canceled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		r := fsspec.New()
		_, err := r.Render(ctx, "create-user", "user-management")
		require.Error(t, err)
		require.True(t, errors.Is(err, context.Canceled),
			"expected errors.Is(err, context.Canceled), got: %v", err,
		)
	})
}
