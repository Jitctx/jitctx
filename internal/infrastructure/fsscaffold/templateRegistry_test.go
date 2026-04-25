package fsscaffold_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
	"github.com/jitctx/jitctx/internal/infrastructure/fsscaffold"
)

func TestTemplateRegistry_Render(t *testing.T) {
	t.Parallel()

	t.Run("InputPort", func(t *testing.T) {
		t.Parallel()

		reg := fsscaffold.NewRegistry()
		in := scaffoldvo.RenderInput{
			ContractType: "input-port",
			Package:      "com.app.user.port.in",
			ClassName:    "CreateUserUseCase",
			Imports:      []string{},
			Methods: []scaffoldvo.RenderedMethod{
				{Signature: "UserResponse execute(CreateUserCommand cmd)", Override: false, Body: ""},
			},
		}

		out, err := reg.Render(context.Background(), in)
		require.NoError(t, err)

		got := string(out)
		require.Contains(t, got, "package com.app.user.port.in;")
		require.Contains(t, got, "public interface CreateUserUseCase")
		require.Contains(t, got, "UserResponse execute(CreateUserCommand cmd);")
		require.NotContains(t, got, "public class")
		require.NotContains(t, got, "@Override")
	})

	t.Run("OutputPort", func(t *testing.T) {
		t.Parallel()

		reg := fsscaffold.NewRegistry()
		in := scaffoldvo.RenderInput{
			ContractType: "output-port",
			Package:      "com.app.user.port.out",
			ClassName:    "UserRepository",
			Imports:      []string{},
			Methods: []scaffoldvo.RenderedMethod{
				{Signature: "Optional<User> findById(Long id)", Override: false, Body: ""},
			},
		}

		out, err := reg.Render(context.Background(), in)
		require.NoError(t, err)

		got := string(out)
		require.Contains(t, got, "public interface UserRepository")
		require.NotContains(t, got, "public class")
	})

	t.Run("Entity", func(t *testing.T) {
		t.Parallel()

		reg := fsscaffold.NewRegistry()
		in := scaffoldvo.RenderInput{
			ContractType:     "entity",
			Package:          "com.app.user.domain",
			ClassName:        "User",
			Imports:          []string{"jakarta.persistence.Entity"},
			ClassAnnotations: []string{"@Entity"},
			Fields:           []string{"Long id", "String email"},
		}

		out, err := reg.Render(context.Background(), in)
		require.NoError(t, err)

		got := string(out)
		require.Contains(t, got, "@Entity")
		require.Contains(t, got, "public class User")
		require.Contains(t, got, "private Long id;")
		require.Contains(t, got, "private String email;")
		require.Contains(t, got, "public User()")
		require.NotContains(t, got, "@Override")
	})

	t.Run("AggregateRoot", func(t *testing.T) {
		t.Parallel()

		reg := fsscaffold.NewRegistry()
		in := scaffoldvo.RenderInput{
			ContractType:     "aggregate-root",
			Package:          "com.app.order.domain",
			ClassName:        "Order",
			Imports:          []string{"jakarta.persistence.Entity"},
			ClassAnnotations: []string{"@Entity"},
			Fields:           []string{"Long id", "Long customerId"},
		}

		out, err := reg.Render(context.Background(), in)
		require.NoError(t, err)

		got := string(out)
		require.Contains(t, got, "@Entity")
		require.Contains(t, got, "public class Order")
		require.Contains(t, got, "private Long id;")
		require.Contains(t, got, "private Long customerId;")
		require.Contains(t, got, "public Order()")
		require.NotContains(t, got, "@Override")
	})

	t.Run("Service", func(t *testing.T) {
		t.Parallel()

		reg := fsscaffold.NewRegistry()
		in := scaffoldvo.RenderInput{
			ContractType: "service",
			Package:      "com.app.user.application",
			ClassName:    "UserServiceImpl",
			Imports: []string{
				"com.app.user.port.in.CreateUserUseCase",
				"com.app.user.port.out.UserRepository",
				"org.springframework.stereotype.Service",
			},
			ClassAnnotations: []string{"@Service"},
			Implements:       "CreateUserUseCase",
			Dependencies: []scaffoldvo.ConstructorDep{
				{Type: "UserRepository", FieldName: "userRepository"},
			},
			Methods: []scaffoldvo.RenderedMethod{
				{
					Signature: "UserResponse execute(CreateUserCommand cmd)",
					Override:  true,
					Body:      `throw new UnsupportedOperationException("Not yet implemented");`,
				},
			},
		}

		out, err := reg.Render(context.Background(), in)
		require.NoError(t, err)

		got := string(out)
		require.Contains(t, got, "@Service")
		require.Contains(t, got, "public class UserServiceImpl implements CreateUserUseCase")
		require.Contains(t, got, "private final UserRepository userRepository;")
		require.Contains(t, got, "public UserServiceImpl(UserRepository userRepository)")
		require.Contains(t, got, "this.userRepository = userRepository;")
		require.Contains(t, got, "@Override")
		require.Contains(t, got, "public UserResponse execute(CreateUserCommand cmd)")
		require.Contains(t, got, `throw new UnsupportedOperationException("Not yet implemented");`)
		require.Contains(t, got, "import com.app.user.port.in.CreateUserUseCase;")
	})

	t.Run("RestAdapter", func(t *testing.T) {
		t.Parallel()

		reg := fsscaffold.NewRegistry()
		in := scaffoldvo.RenderInput{
			ContractType: "rest-adapter",
			Package:      "com.app.user.adapter.in.web",
			ClassName:    "UserController",
			Imports: []string{
				"com.app.user.port.in.CreateUserUseCase",
				"org.springframework.web.bind.annotation.PostMapping",
				"org.springframework.web.bind.annotation.RestController",
			},
			ClassAnnotations: []string{"@RestController"},
			Dependencies: []scaffoldvo.ConstructorDep{
				{Type: "CreateUserUseCase", FieldName: "createUserUseCase"},
			},
			Endpoints: []scaffoldvo.RenderedEndpoint{
				{
					Annotation: `@PostMapping("/users")`,
					Method:     "postUsers",
					Body:       `throw new UnsupportedOperationException("Not yet implemented");`,
				},
			},
		}

		out, err := reg.Render(context.Background(), in)
		require.NoError(t, err)

		got := string(out)
		require.Contains(t, got, "@RestController")
		require.Contains(t, got, "public class UserController")
		require.Contains(t, got, "private final CreateUserUseCase createUserUseCase;")
		require.Contains(t, got, `@PostMapping("/users")`)
		require.Contains(t, got, "public Object postUsers()")
		require.Contains(t, got, `throw new UnsupportedOperationException(`)
	})

	t.Run("JpaAdapter", func(t *testing.T) {
		t.Parallel()

		reg := fsscaffold.NewRegistry()
		in := scaffoldvo.RenderInput{
			ContractType:     "jpa-adapter",
			Package:          "com.app.user.adapter.out.persistence",
			ClassName:        "UserJpaAdapter",
			Imports:          []string{"org.springframework.stereotype.Repository"},
			ClassAnnotations: []string{"@Repository"},
			Implements:       "UserRepository",
			Dependencies:     []scaffoldvo.ConstructorDep{},
			Methods: []scaffoldvo.RenderedMethod{
				{
					Signature: "Optional<User> findById(Long id)",
					Override:  true,
					Body:      `throw new UnsupportedOperationException("Not yet implemented");`,
				},
			},
		}

		out, err := reg.Render(context.Background(), in)
		require.NoError(t, err)

		got := string(out)
		require.Contains(t, got, "@Repository")
		require.Contains(t, got, "implements UserRepository")
		require.Contains(t, got, "@Override")
		require.Contains(t, got, `throw new UnsupportedOperationException("Not yet implemented");`)
	})

	t.Run("UnsupportedType_ReturnsTypedError", func(t *testing.T) {
		t.Parallel()

		reg := fsscaffold.NewRegistry()
		in := scaffoldvo.RenderInput{
			ContractType: "weird-thing",
		}

		_, err := reg.Render(context.Background(), in)
		require.Error(t, err)

		var typedErr *domerr.UnsupportedContractTypeError
		require.True(t, errors.As(err, &typedErr))
		require.Equal(t, "weird-thing", typedErr.Type)
		require.NotEmpty(t, typedErr.SupportedSorted)

		supported := typedErr.SupportedSorted
		require.Contains(t, supported, "input-port")
		require.Contains(t, supported, "service")
		require.Contains(t, supported, "rest-adapter")
		// Verify list is sorted.
		for i := 1; i < len(supported); i++ {
			require.LessOrEqual(t, supported[i-1], supported[i],
				"SupportedSorted must be sorted alphabetically")
		}
	})

	t.Run("Deterministic", func(t *testing.T) {
		t.Parallel()

		reg := fsscaffold.NewRegistry()
		in := scaffoldvo.RenderInput{
			ContractType: "service",
			Package:      "com.app.user.application",
			ClassName:    "UserServiceImpl",
			Imports: []string{
				"com.app.user.port.in.CreateUserUseCase",
				"org.springframework.stereotype.Service",
			},
			ClassAnnotations: []string{"@Service"},
			Implements:       "CreateUserUseCase",
			Dependencies: []scaffoldvo.ConstructorDep{
				{Type: "CreateUserUseCase", FieldName: "createUserUseCase"},
			},
			Methods: []scaffoldvo.RenderedMethod{
				{
					Signature: "UserResponse execute(CreateUserCommand cmd)",
					Override:  true,
					Body:      `throw new UnsupportedOperationException("Not yet implemented");`,
				},
			},
		}

		first, err := reg.Render(context.Background(), in)
		require.NoError(t, err)

		second, err := reg.Render(context.Background(), in)
		require.NoError(t, err)

		require.Equal(t, first, second, "renders must be byte-identical (RNF-002)")
	})

	t.Run("CtxCancelled", func(t *testing.T) {
		t.Parallel()

		reg := fsscaffold.NewRegistry()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		in := scaffoldvo.RenderInput{
			ContractType: "input-port",
			Package:      "com.app.user.port.in",
			ClassName:    "CreateUserUseCase",
		}

		_, err := reg.Render(ctx, in)
		require.Error(t, err)
		require.True(t, errors.Is(err, context.Canceled),
			"expected context.Canceled, got: %v", err)
	})
}
