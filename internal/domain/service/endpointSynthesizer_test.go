package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/service"
)

func TestEndpointSynthesizer_Parse(t *testing.T) {
	t.Parallel()

	synth := service.NewEndpointSynthesizer()

	t.Run("GetUsers", func(t *testing.T) {
		t.Parallel()

		got, err := synth.Parse("GET /users")
		require.NoError(t, err)
		require.Equal(t, "GET", got.Verb)
		require.Equal(t, "/users", got.Path)
		require.Equal(t, `@GetMapping("/users")`, got.Annotation)
		require.Equal(t, "getUsers", got.MethodName)
	})

	t.Run("PostUsers", func(t *testing.T) {
		t.Parallel()

		got, err := synth.Parse("POST /users")
		require.NoError(t, err)
		require.Equal(t, "postUsers", got.MethodName)
		require.Equal(t, `@PostMapping("/users")`, got.Annotation)
	})

	t.Run("GetUsersWithId", func(t *testing.T) {
		t.Parallel()

		got, err := synth.Parse("GET /users/{id}")
		require.NoError(t, err)
		// curly braces stripped from method name
		require.Equal(t, "getUsersId", got.MethodName)
		// annotation preserves the path verbatim
		require.Equal(t, `@GetMapping("/users/{id}")`, got.Annotation)
	})

	t.Run("DeleteSingleSegment", func(t *testing.T) {
		t.Parallel()

		got, err := synth.Parse("DELETE /")
		require.NoError(t, err)
		require.Equal(t, "delete", got.MethodName)
	})

	t.Run("PutMultiSegment", func(t *testing.T) {
		t.Parallel()

		got, err := synth.Parse("PUT /api/v1/orders")
		require.NoError(t, err)
		require.Equal(t, "putApiV1Orders", got.MethodName)
	})

	t.Run("UnknownVerb", func(t *testing.T) {
		t.Parallel()

		_, err := synth.Parse("FOO /bar")
		require.Error(t, err)
	})

	t.Run("SingleToken", func(t *testing.T) {
		t.Parallel()

		_, err := synth.Parse("GET")
		require.Error(t, err)
	})

	t.Run("ExtraTokens", func(t *testing.T) {
		t.Parallel()

		_, err := synth.Parse("GET /a /b")
		require.Error(t, err)
	})

	t.Run("LowercaseVerbAccepted", func(t *testing.T) {
		t.Parallel()

		got, err := synth.Parse("get /users")
		require.NoError(t, err)
		require.Equal(t, "GET", got.Verb)
		require.Contains(t, got.Annotation, "@GetMapping")
	})

	t.Run("PathWithDashesAndDigits", func(t *testing.T) {
		t.Parallel()

		got, err := synth.Parse("GET /api-v1/2025-orders")
		require.NoError(t, err)
		// Must not contain non-alnum chars and must start with "get"
		for _, r := range got.MethodName {
			require.True(t, (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'),
				"method name must only contain alnum chars, got: %s", got.MethodName)
		}
		require.True(t, len(got.MethodName) >= 3 && got.MethodName[:3] == "get",
			"method name must start with 'get', got: %s", got.MethodName)
	})
}
