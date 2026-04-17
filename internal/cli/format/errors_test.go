package format_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/cli/format"
	domerr "github.com/jitctx/jitctx/internal/domain/errors"
)

func TestTranslateError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{
			name:     "nil returns nil",
			err:      nil,
			contains: "",
		},
		{
			name:     "ErrNoProfileMatch",
			err:      domerr.ErrNoProfileMatch,
			contains: "no matching framework profile found",
		},
		{
			name:     "ErrManifestWrite",
			err:      domerr.ErrManifestWrite,
			contains: "failed to write project-state.yaml",
		},
		{
			name:     "ErrManifestNotFound",
			err:      domerr.ErrManifestNotFound,
			contains: "project-state.yaml not found",
		},
		{
			name:     "ErrModuleNotFound",
			err:      domerr.ErrModuleNotFound,
			contains: "module not found in manifest",
		},
		{
			name:     "ErrProfileInvalid",
			err:      domerr.ErrProfileInvalid,
			contains: "framework profile is invalid",
		},
		{
			name:     "wrapped ErrNoProfileMatch",
			err:      errors.Join(domerr.ErrNoProfileMatch),
			contains: "no matching framework profile found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := format.TranslateError(tc.err)
			if tc.err == nil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Contains(t, got.Error(), tc.contains)
		})
	}
}
