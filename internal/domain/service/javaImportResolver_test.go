package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
)

func TestJavaImportResolver_Resolve(t *testing.T) {
	t.Parallel()

	resolver := service.NewJavaImportResolver(service.NewContractPathMapper())

	t.Run("ServiceWithImplementsAndDependsOn", func(t *testing.T) {
		t.Parallel()

		spec := model.FeatureSpec{
			Feature: "create-user",
			Module:  "user-management",
			Package: "com.app.user",
			Contracts: []model.SpecContract{
				{
					Name:    "CreateUserUseCase",
					Type:    model.ContractInputPort,
					Methods: []string{"UserResponse execute(CreateUserCommand cmd)"},
				},
				{
					Name: "UserRepository",
					Type: model.ContractOutputPort,
				},
				{
					Name:       "UserServiceImpl",
					Type:       model.ContractService,
					Implements: "CreateUserUseCase",
					DependsOn:  []string{"UserRepository"},
				},
			},
		}

		target := spec.Contracts[2] // UserServiceImpl
		imports, err := resolver.Resolve(spec, target, "com.app.user")
		require.NoError(t, err)

		require.Contains(t, imports, "com.app.user.port.in.CreateUserUseCase")
		require.Contains(t, imports, "com.app.user.port.out.UserRepository")
		require.Contains(t, imports, "org.springframework.stereotype.Service")

		// Verify sorted order
		for i := 1; i < len(imports); i++ {
			require.LessOrEqual(t, imports[i-1], imports[i], "imports must be sorted alphabetically")
		}
	})

	t.Run("RestAdapterWithUsesAndEndpoints", func(t *testing.T) {
		t.Parallel()

		spec := model.FeatureSpec{
			Feature: "create-user",
			Module:  "user-management",
			Package: "com.app.user",
			Contracts: []model.SpecContract{
				{
					Name:    "CreateUserUseCase",
					Type:    model.ContractInputPort,
					Methods: []string{"UserResponse execute(CreateUserCommand cmd)"},
				},
				{
					Name:      "UserController",
					Type:      model.ContractRestAdapter,
					Uses:      []string{"CreateUserUseCase"},
					Endpoints: []string{"POST /users", "GET /users/{id}"},
				},
			},
		}

		target := spec.Contracts[1] // UserController
		imports, err := resolver.Resolve(spec, target, "com.app.user")
		require.NoError(t, err)

		require.Contains(t, imports, "com.app.user.port.in.CreateUserUseCase")
		require.Contains(t, imports, "org.springframework.web.bind.annotation.RestController")
		require.Contains(t, imports, "org.springframework.web.bind.annotation.PostMapping")
		require.Contains(t, imports, "org.springframework.web.bind.annotation.GetMapping")

		// Each verb mapping appears exactly once
		count := 0
		for _, imp := range imports {
			if imp == "org.springframework.web.bind.annotation.PostMapping" {
				count++
			}
		}
		require.Equal(t, 1, count, "PostMapping must appear exactly once")
	})

	t.Run("EntityWithFields", func(t *testing.T) {
		t.Parallel()

		spec := model.FeatureSpec{
			Feature: "create-user",
			Module:  "user-management",
			Package: "com.app.user",
			Contracts: []model.SpecContract{
				{
					Name:   "UserEntity",
					Type:   model.ContractEntity,
					Fields: []string{"UUID id", "String name"},
				},
			},
		}

		target := spec.Contracts[0]
		imports, err := resolver.Resolve(spec, target, "com.app.user")
		require.NoError(t, err)

		require.Contains(t, imports, "jakarta.persistence.Entity")
	})

	t.Run("EntityLongIdAndStringEmail_AddsIdAndGeneratedValue", func(t *testing.T) {
		t.Parallel()

		spec := model.FeatureSpec{
			Feature: "create-user",
			Module:  "user-management",
			Package: "com.app.user",
			Contracts: []model.SpecContract{
				{
					Name:   "UserEntity",
					Type:   model.ContractEntity,
					Fields: []string{"Long id", "String email"},
				},
			},
		}

		target := spec.Contracts[0]
		imports, err := resolver.Resolve(spec, target, "com.app.user")
		require.NoError(t, err)

		require.Contains(t, imports, "jakarta.persistence.Entity")
		require.Contains(t, imports, "jakarta.persistence.Id")
		require.Contains(t, imports, "jakarta.persistence.GeneratedValue")
		require.Contains(t, imports, "jakarta.persistence.GenerationType")
	})

	t.Run("EntityUUIDId_AddsIdOnly", func(t *testing.T) {
		t.Parallel()

		spec := model.FeatureSpec{
			Feature: "create-user",
			Module:  "user-management",
			Package: "com.app.user",
			Contracts: []model.SpecContract{
				{
					Name:   "UserEntity",
					Type:   model.ContractEntity,
					Fields: []string{"UUID id"},
				},
			},
		}

		target := spec.Contracts[0]
		imports, err := resolver.Resolve(spec, target, "com.app.user")
		require.NoError(t, err)

		require.Contains(t, imports, "jakarta.persistence.Entity")
		require.Contains(t, imports, "jakarta.persistence.Id")
		require.NotContains(t, imports, "jakarta.persistence.GeneratedValue")
		require.NotContains(t, imports, "jakarta.persistence.GenerationType")
	})

	t.Run("EntityStringEmailOnly_NoExtraImports", func(t *testing.T) {
		t.Parallel()

		spec := model.FeatureSpec{
			Feature: "create-user",
			Module:  "user-management",
			Package: "com.app.user",
			Contracts: []model.SpecContract{
				{
					Name:   "UserEntity",
					Type:   model.ContractEntity,
					Fields: []string{"String email"},
				},
			},
		}

		target := spec.Contracts[0]
		imports, err := resolver.Resolve(spec, target, "com.app.user")
		require.NoError(t, err)

		require.Contains(t, imports, "jakarta.persistence.Entity")
		require.NotContains(t, imports, "jakarta.persistence.Id")
		require.NotContains(t, imports, "jakarta.persistence.GeneratedValue")
		require.NotContains(t, imports, "jakarta.persistence.GenerationType")
	})

	t.Run("AggregateRootLongId_AddsIdAndGeneratedValue", func(t *testing.T) {
		t.Parallel()

		spec := model.FeatureSpec{
			Feature: "create-order",
			Module:  "order-management",
			Package: "com.app.order",
			Contracts: []model.SpecContract{
				{
					Name:   "Order",
					Type:   model.ContractAggregate,
					Fields: []string{"Long id"},
				},
			},
		}

		target := spec.Contracts[0]
		imports, err := resolver.Resolve(spec, target, "com.app.order")
		require.NoError(t, err)

		require.Contains(t, imports, "jakarta.persistence.Entity")
		require.Contains(t, imports, "jakarta.persistence.Id")
		require.Contains(t, imports, "jakarta.persistence.GeneratedValue")
		require.Contains(t, imports, "jakarta.persistence.GenerationType")
	})

	t.Run("JpaAdapter", func(t *testing.T) {
		t.Parallel()

		spec := model.FeatureSpec{
			Feature: "create-user",
			Module:  "user-management",
			Package: "com.app.user",
			Contracts: []model.SpecContract{
				{
					Name:       "UserRepositoryJpa",
					Type:       model.ContractJPAAdapter,
					Implements: "UserRepository",
				},
				{
					Name: "UserRepository",
					Type: model.ContractOutputPort,
				},
			},
		}

		target := spec.Contracts[0]
		imports, err := resolver.Resolve(spec, target, "com.app.user")
		require.NoError(t, err)

		require.Contains(t, imports, "org.springframework.stereotype.Repository")
	})

	t.Run("InputPortNoImports", func(t *testing.T) {
		t.Parallel()

		spec := model.FeatureSpec{
			Feature: "create-user",
			Module:  "user-management",
			Package: "com.app.user",
			Contracts: []model.SpecContract{
				{
					Name:    "CreateUserUseCase",
					Type:    model.ContractInputPort,
					Methods: []string{"UserResponse execute(CreateUserCommand cmd)"},
				},
			},
		}

		target := spec.Contracts[0]
		imports, err := resolver.Resolve(spec, target, "com.app.user")
		require.NoError(t, err)

		require.Empty(t, imports)
	})

	t.Run("ExternalReferenceSilentlyDropped", func(t *testing.T) {
		t.Parallel()

		spec := model.FeatureSpec{
			Feature: "create-user",
			Module:  "user-management",
			Package: "com.app.user",
			Contracts: []model.SpecContract{
				{
					Name:      "UserServiceImpl",
					Type:      model.ContractService,
					DependsOn: []string{"UnknownThing"},
				},
			},
		}

		target := spec.Contracts[0]
		imports, err := resolver.Resolve(spec, target, "com.app.user")
		require.NoError(t, err)

		for _, imp := range imports {
			require.NotContains(t, imp, "UnknownThing", "external reference must not appear in imports")
		}
		// Framework import for @Service is still present
		require.Contains(t, imports, "org.springframework.stereotype.Service")
	})

	t.Run("SortedAndDeduplicated", func(t *testing.T) {
		t.Parallel()

		// Two refs that map to the same FQN: same contract type+name
		spec := model.FeatureSpec{
			Feature: "create-user",
			Module:  "user-management",
			Package: "com.app.user",
			Contracts: []model.SpecContract{
				{
					Name: "CreateUserUseCase",
					Type: model.ContractInputPort,
				},
				{
					Name: "DuplicateService",
					Type: model.ContractService,
					// DependsOn lists CreateUserUseCase twice
					DependsOn: []string{"CreateUserUseCase", "CreateUserUseCase"},
				},
			},
		}

		target := spec.Contracts[1]
		imports, err := resolver.Resolve(spec, target, "com.app.user")
		require.NoError(t, err)

		fqn := "com.app.user.port.in.CreateUserUseCase"
		count := 0
		for _, imp := range imports {
			if imp == fqn {
				count++
			}
		}
		require.Equal(t, 1, count, "deduplicated: FQN must appear exactly once")

		for i := 1; i < len(imports); i++ {
			require.LessOrEqual(t, imports[i-1], imports[i], "imports must be sorted alphabetically")
		}
	})

	t.Run("SkipSelfReference", func(t *testing.T) {
		t.Parallel()

		spec := model.FeatureSpec{
			Feature: "create-user",
			Module:  "user-management",
			Package: "com.app.user",
			Contracts: []model.SpecContract{
				{
					Name:       "XService",
					Type:       model.ContractService,
					Implements: "XService", // self-reference
				},
			},
		}

		target := spec.Contracts[0]
		imports, err := resolver.Resolve(spec, target, "com.app.user")
		require.NoError(t, err)

		for _, imp := range imports {
			require.NotContains(t, imp, "XService", "self-reference must not appear in imports")
		}
	})
}
