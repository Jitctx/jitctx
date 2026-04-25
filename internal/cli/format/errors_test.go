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

func TestTranslateError_ContractTargetNotFound_BothSearched(t *testing.T) {
	t.Parallel()

	err := &domerr.ContractTargetNotFoundError{
		TargetFile:       "x.java",
		ContractName:     "X",
		SearchedSpec:     true,
		SearchedManifest: true,
	}

	got := format.TranslateError(err)
	require.NotNil(t, got)
	require.Contains(t, got.Error(), `could not find contract "X"`)
	require.Contains(t, got.Error(), "x.java")
	require.Contains(t, got.Error(), "jitctx scan")
	require.Contains(t, got.Error(), "jitctx plan")
}

func TestTranslateError_ContractTargetNotFound_SpecOnly(t *testing.T) {
	t.Parallel()

	err := &domerr.ContractTargetNotFoundError{
		TargetFile:       "x.java",
		ContractName:     "X",
		SearchedSpec:     true,
		SearchedManifest: false,
	}

	got := format.TranslateError(err)
	require.NotNil(t, got)
	require.Contains(t, got.Error(), `could not find contract "X"`)
	require.Contains(t, got.Error(), "x.java")
	require.Contains(t, got.Error(), "jitctx plan --new")
}

func TestTranslateError_ContractTargetNotFound_ManifestOnly(t *testing.T) {
	t.Parallel()

	err := &domerr.ContractTargetNotFoundError{
		TargetFile:       "x.java",
		ContractName:     "X",
		SearchedSpec:     false,
		SearchedManifest: true,
	}

	got := format.TranslateError(err)
	require.NotNil(t, got)
	require.Contains(t, got.Error(), `could not find contract "X"`)
	require.Contains(t, got.Error(), "x.java")
	require.Contains(t, got.Error(), "first to populate project-state.yaml")
	require.Contains(t, got.Error(), "--feature/--file")
}

func TestTranslateError_ScaffoldConflict(t *testing.T) {
	t.Parallel()

	err := &domerr.ScaffoldConflictError{Conflicts: []string{"/a.java", "/b.java"}}

	got := format.TranslateError(err)
	require.NotNil(t, got)
	require.Contains(t, got.Error(), "scaffold aborted")
	require.Contains(t, got.Error(), "/a.java")
	require.Contains(t, got.Error(), "/b.java")
	require.Contains(t, got.Error(), "delete the listed files")
}

func TestTranslateError_SpecMissingPackage(t *testing.T) {
	t.Parallel()

	got := format.TranslateError(domerr.ErrSpecMissingPackage)
	require.NotNil(t, got)
	require.Contains(t, got.Error(), "Package: <java.package>")
	require.Contains(t, got.Error(), "after the 'Module:' line")
}

func TestTranslateError_ScaffoldRenderError(t *testing.T) {
	t.Parallel()

	err := &domerr.ScaffoldRenderError{Contract: "Foo", Cause: errors.New("boom")}

	got := format.TranslateError(err)
	require.NotNil(t, got)
	require.Contains(t, got.Error(), "failed to render contract")
	require.Contains(t, got.Error(), "Foo")
	require.Contains(t, got.Error(), "boom")
}

func TestTranslateError_ScaffoldRenderError_NilCause(t *testing.T) {
	t.Parallel()

	err := &domerr.ScaffoldRenderError{Contract: "Foo", Cause: nil}

	require.NotPanics(t, func() {
		got := format.TranslateError(err)
		require.NotNil(t, got)
		require.Contains(t, got.Error(), "failed to render contract")
		require.Contains(t, got.Error(), "Foo")
	})
}
