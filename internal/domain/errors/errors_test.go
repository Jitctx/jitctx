package errors_test

import (
	"errors"
	"fmt"
	"testing"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
)

func TestSentinels_Wrapped(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		target error
	}{
		{"ErrNoProfileMatch", domerr.ErrNoProfileMatch},
		{"ErrParseFailure", domerr.ErrParseFailure},
		{"ErrPartialParse", domerr.ErrPartialParse},
		{"ErrManifestWrite", domerr.ErrManifestWrite},
		{"ErrManifestNotFound", domerr.ErrManifestNotFound},
		{"ErrModuleNotFound", domerr.ErrModuleNotFound},
		{"ErrProfileInvalid", domerr.ErrProfileInvalid},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			wrapped := fmt.Errorf("outer: %w", tc.target)
			if !errors.Is(wrapped, tc.target) {
				t.Errorf("errors.Is(%v, %v) = false; want true", wrapped, tc.target)
			}
		})
	}
}
