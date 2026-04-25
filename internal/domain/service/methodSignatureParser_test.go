package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/service"
)

func TestMethodSignatureParser_Parse(t *testing.T) {
	t.Parallel()

	parser := service.NewMethodSignatureParser()

	t.Run("VoidNoParams", func(t *testing.T) {
		t.Parallel()

		got, err := parser.Parse("void execute()")
		require.NoError(t, err)
		require.Equal(t, "void", got.ReturnType)
		require.Equal(t, "execute", got.Name)
		require.Nil(t, got.Params)
	})

	t.Run("SingleParam", func(t *testing.T) {
		t.Parallel()

		got, err := parser.Parse("UserResponse execute(CreateUserCommand cmd)")
		require.NoError(t, err)
		require.Equal(t, "UserResponse", got.ReturnType)
		require.Equal(t, "execute", got.Name)
		require.Len(t, got.Params, 1)
		require.Equal(t, service.ParsedParam{Type: "CreateUserCommand", Name: "cmd"}, got.Params[0])
	})

	t.Run("GenericsReturnTypeOneParam", func(t *testing.T) {
		t.Parallel()

		got, err := parser.Parse("Optional<User> findByEmail(String email)")
		require.NoError(t, err)
		require.Equal(t, "Optional<User>", got.ReturnType)
		require.Equal(t, "findByEmail", got.Name)
		require.Len(t, got.Params, 1)
		require.Equal(t, service.ParsedParam{Type: "String", Name: "email"}, got.Params[0])
	})

	t.Run("GenericsWithCommaInTypeTwoParams", func(t *testing.T) {
		t.Parallel()

		got, err := parser.Parse("Map<String,List<User>> bulkLookup(List<String> emails, int limit)")
		require.NoError(t, err)
		require.Equal(t, "Map<String,List<User>>", got.ReturnType)
		require.Equal(t, "bulkLookup", got.Name)
		require.Len(t, got.Params, 2)
		require.Equal(t, service.ParsedParam{Type: "List<String>", Name: "emails"}, got.Params[0])
		require.Equal(t, service.ParsedParam{Type: "int", Name: "limit"}, got.Params[1])
	})

	t.Run("TrimsTrailingSemicolonAndWhitespace", func(t *testing.T) {
		t.Parallel()

		got, err := parser.Parse("  void  ping();  ")
		require.NoError(t, err)
		require.Equal(t, "void", got.ReturnType)
		require.Equal(t, "ping", got.Name)
		require.Nil(t, got.Params)
	})

	malformedCases := []struct {
		name string
		raw  string
	}{
		{"EmptyString", ""},
		{"ReturnTypeOnly", "void"},
		{"ReturnTypeAndName", "void foo"},
		{"MissingClosingParen", "void foo("},
		{"MissingOpenParen", "foo)"},
	}

	for _, tc := range malformedCases {
		t.Run("Malformed_"+tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := parser.Parse(tc.raw)
			require.Error(t, err)
			require.Contains(t, err.Error(), "method signature must be")
		})
	}
}
