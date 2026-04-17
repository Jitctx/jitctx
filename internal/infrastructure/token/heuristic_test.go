package token_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/infrastructure/token"
)

func TestHeuristicEstimator(t *testing.T) {
	t.Parallel()

	e := token.NewHeuristicEstimator()
	ctx := context.Background()

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty string", "", 0},
		{"single word", "hello", 1},
		{"four words", "hello world foo bar", 5},
		{"unicode", "日本語 テスト", 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.Estimate(ctx, tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
