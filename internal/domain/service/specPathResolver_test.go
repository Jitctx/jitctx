package service_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/service"
)

func TestSpecPathResolver_Resolve(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		feature       string
		baseDir       string
		wantErrSubstr string
		wantPath      string
	}{
		{
			name:     "happy_path_relative_base",
			feature:  "create-user",
			baseDir:  "jitctx-plans",
			wantPath: filepath.Join("jitctx-plans", "create-user.md"),
		},
		{
			name:     "happy_path_absolute_base",
			feature:  "foo",
			baseDir:  "/tmp/specs",
			wantPath: filepath.Join("/tmp/specs", "foo.md"),
		},
		{
			name:          "empty_feature_rejected",
			feature:       "",
			baseDir:       "jitctx-plans",
			wantErrSubstr: "must not be empty",
		},
		{
			name:          "feature_with_slash_rejected",
			feature:       "a/b",
			baseDir:       "jitctx-plans",
			wantErrSubstr: "path separators",
		},
		{
			name:          "feature_dot_rejected",
			feature:       ".",
			baseDir:       "jitctx-plans",
			wantErrSubstr: "path separators",
		},
		{
			name:          "empty_base_rejected",
			feature:       "foo",
			baseDir:       "",
			wantErrSubstr: "base dir must not",
		},
	}

	resolver := service.NewSpecPathResolver()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolver.Resolve(tc.feature, tc.baseDir)

			if tc.wantErrSubstr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrSubstr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantPath, got)
		})
	}
}
