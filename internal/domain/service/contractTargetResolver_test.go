package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/service"
)

func TestContractTargetResolver_Resolve(t *testing.T) {
	t.Parallel()

	resolver := service.NewContractTargetResolver()

	cases := []struct {
		name       string
		input      string
		wantName   string
		wantErrSub string
	}{
		{
			name:     "FullPath",
			input:    "src/main/java/com/app/UserServiceImpl.java",
			wantName: "UserServiceImpl",
		},
		{
			name:     "BareFilename",
			input:    "UserController.java",
			wantName: "UserController",
		},
		{
			name:     "NoExtension",
			input:    "User",
			wantName: "User",
		},
		{
			name:       "Empty",
			input:      "",
			wantErrSub: "must not be empty",
		},
		{
			name:       "Whitespace",
			input:      "   ",
			wantErrSub: "must not be empty",
		},
		{
			name:       "OnlyExtension",
			input:      ".java",
			wantErrSub: "produces empty contract name",
		},
		{
			name:       "Backslash",
			input:      `src\main\Foo.java`,
			wantErrSub: "must use '/' separators",
		},
		{
			name:     "MultiDot",
			input:    "Foo.bar.java",
			wantName: "Foo.bar",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolver.Resolve(tc.input)

			if tc.wantErrSub != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrSub)
				require.Empty(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantName, got)
		})
	}
}
