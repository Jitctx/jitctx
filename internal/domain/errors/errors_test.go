package errors_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/stretchr/testify/require"
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
		{"ErrSpecParse", domerr.ErrSpecParse},
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

func TestSpecParseError_IsErrSpecParse(t *testing.T) {
	t.Parallel()

	err := &domerr.SpecParseError{Line: 5, Message: "test"}

	require.True(t, errors.Is(err, domerr.ErrSpecParse))
}

func TestSpecParseError_ErrorIncludesLineNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		line    int
		message string
		wantIn  string
	}{
		{
			name:    "line-5-test-message",
			line:    5,
			message: "test",
			wantIn:  "line 5: test",
		},
		{
			name:    "line-1-missing-feature-header",
			line:    1,
			message: "missing feature header",
			wantIn:  "line 1: missing feature header",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := &domerr.SpecParseError{Line: tc.line, Message: tc.message}

			require.Contains(t, err.Error(), tc.wantIn)
		})
	}
}

func TestDuplicateContractError_IsErrSpecParse(t *testing.T) {
	t.Parallel()

	err := &domerr.DuplicateContractError{
		Name:      "UserRepository",
		FirstLine: 10,
		DupeLine:  25,
	}

	require.True(t, errors.Is(err, domerr.ErrSpecParse))
}

func TestDuplicateContractError_ErrorIncludesLineNumbersAndName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		contract  string
		firstLine int
		dupeLine  int
	}{
		{
			name:      "user-repository-duplicate",
			contract:  "UserRepository",
			firstLine: 10,
			dupeLine:  25,
		},
		{
			name:      "create-user-use-case-duplicate",
			contract:  "CreateUserUseCase",
			firstLine: 3,
			dupeLine:  42,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := &domerr.DuplicateContractError{
				Name:      tc.contract,
				FirstLine: tc.firstLine,
				DupeLine:  tc.dupeLine,
			}

			msg := err.Error()
			require.True(t, strings.Contains(msg, tc.contract),
				"error message %q should contain contract name %q", msg, tc.contract)
			require.Contains(t, msg, fmt.Sprintf("%d", tc.firstLine))
			require.Contains(t, msg, fmt.Sprintf("%d", tc.dupeLine))
		})
	}
}

func TestSpecParseWarning_ErrorIncludesLineNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		line    int
		message string
	}{
		{
			name:    "unknown-field-warning",
			line:    7,
			message: "unknown field: Magic",
		},
		{
			name:    "line-42-warning",
			line:    42,
			message: "unrecognized content",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := &domerr.SpecParseWarning{Line: tc.line, Message: tc.message}

			msg := w.Error()
			require.Contains(t, msg, fmt.Sprintf("line %d:", tc.line))
			require.Contains(t, msg, tc.message)
		})
	}
}
