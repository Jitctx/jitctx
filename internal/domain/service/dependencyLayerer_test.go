package service_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
)

// mkContract is a small helper that creates a SpecContract with the given
// name and type. Optional functional options configure DependsOn, Uses,
// and Implements.
type optFn func(*model.SpecContract)

func withDependsOn(deps ...string) optFn {
	return func(c *model.SpecContract) { c.DependsOn = deps }
}

func withUses(uses ...string) optFn {
	return func(c *model.SpecContract) { c.Uses = uses }
}

func withImplements(impl string) optFn {
	return func(c *model.SpecContract) { c.Implements = impl }
}

func mkContract(name string, t model.ContractType, opts ...optFn) model.SpecContract {
	c := model.SpecContract{Name: name, Type: t}
	for _, o := range opts {
		o(&c)
	}
	return c
}

func TestDependencyLayerer_Layer(t *testing.T) {
	t.Parallel()

	layerer := service.NewDependencyLayerer()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		layers, externals, err := layerer.Layer(nil)

		require.NoError(t, err)
		require.Empty(t, layers)
		require.Empty(t, externals)
	})

	t.Run("single", func(t *testing.T) {
		t.Parallel()

		contracts := []model.SpecContract{
			mkContract("Foo", model.ContractInputPort),
		}

		layers, externals, err := layerer.Layer(contracts)

		require.NoError(t, err)
		require.Len(t, layers, 1)
		require.Equal(t, 0, layers[0].Index)
		require.Len(t, layers[0].Targets, 1)
		require.Equal(t, "Foo", layers[0].Targets[0].Name)
		require.Empty(t, externals)
	})

	t.Run("chain", func(t *testing.T) {
		t.Parallel()

		// A (service, DependsOn: B), B (service, DependsOn: C), C (output-port)
		// Expected layers: [C], [B], [A]
		contracts := []model.SpecContract{
			mkContract("A", model.ContractService, withDependsOn("B")),
			mkContract("B", model.ContractService, withDependsOn("C")),
			mkContract("C", model.ContractOutputPort),
		}

		layers, externals, err := layerer.Layer(contracts)

		require.NoError(t, err)
		require.Empty(t, externals)
		require.Len(t, layers, 3)

		require.Equal(t, 0, layers[0].Index)
		require.Len(t, layers[0].Targets, 1)
		require.Equal(t, "C", layers[0].Targets[0].Name)

		require.Equal(t, 1, layers[1].Index)
		require.Len(t, layers[1].Targets, 1)
		require.Equal(t, "B", layers[1].Targets[0].Name)

		require.Equal(t, 2, layers[2].Index)
		require.Len(t, layers[2].Targets, 1)
		require.Equal(t, "A", layers[2].Targets[0].Name)
	})

	t.Run("multipleRoots", func(t *testing.T) {
		t.Parallel()

		// 4 contracts mirroring .feature line 117-121:
		//   CreateUserUseCase (input-port)
		//   UserRepository (output-port)
		//   UserServiceImpl (service, Implements: CreateUserUseCase, DependsOn: UserRepository)
		//   UserController (rest-adapter, Uses: CreateUserUseCase)
		//
		// Expected 2 layers:
		//   layer 0 = [CreateUserUseCase, UserRepository]  (alphabetical)
		//   layer 1 = [UserController, UserServiceImpl]    (alphabetical)
		contracts := []model.SpecContract{
			mkContract("CreateUserUseCase", model.ContractInputPort),
			mkContract("UserRepository", model.ContractOutputPort),
			mkContract("UserServiceImpl", model.ContractService,
				withImplements("CreateUserUseCase"),
				withDependsOn("UserRepository"),
			),
			mkContract("UserController", model.ContractRestAdapter,
				withUses("CreateUserUseCase"),
			),
		}

		layers, externals, err := layerer.Layer(contracts)

		require.NoError(t, err)
		require.Empty(t, externals)
		require.Len(t, layers, 2)

		// Layer 0: CreateUserUseCase, UserRepository (alphabetical)
		require.Equal(t, 0, layers[0].Index)
		require.Len(t, layers[0].Targets, 2)
		require.Equal(t, "CreateUserUseCase", layers[0].Targets[0].Name)
		require.Equal(t, "UserRepository", layers[0].Targets[1].Name)

		// Layer 1: UserController, UserServiceImpl (alphabetical)
		require.Equal(t, 1, layers[1].Index)
		require.Len(t, layers[1].Targets, 2)
		require.Equal(t, "UserController", layers[1].Targets[0].Name)
		require.Equal(t, "UserServiceImpl", layers[1].Targets[1].Name)
	})

	t.Run("cycle2", func(t *testing.T) {
		t.Parallel()

		// A (service, DependsOn: B), B (service, DependsOn: A)
		contracts := []model.SpecContract{
			mkContract("A", model.ContractService, withDependsOn("B")),
			mkContract("B", model.ContractService, withDependsOn("A")),
		}

		_, _, err := layerer.Layer(contracts)

		require.Error(t, err)
		require.True(t, errors.Is(err, domerr.ErrDependencyCycle))

		var cycleErr *service.CycleError
		require.ErrorAs(t, err, &cycleErr)
		require.Equal(t, []string{"A", "B", "A"}, cycleErr.Path)
	})

	t.Run("cycle3", func(t *testing.T) {
		t.Parallel()

		// A→B→C→A; alphabetically smallest start is "A"
		contracts := []model.SpecContract{
			mkContract("A", model.ContractService, withDependsOn("B")),
			mkContract("B", model.ContractService, withDependsOn("C")),
			mkContract("C", model.ContractService, withDependsOn("A")),
		}

		_, _, err := layerer.Layer(contracts)

		require.Error(t, err)
		require.True(t, errors.Is(err, domerr.ErrDependencyCycle))

		var cycleErr *service.CycleError
		require.ErrorAs(t, err, &cycleErr)
		require.Equal(t, []string{"A", "B", "C", "A"}, cycleErr.Path)
	})

	t.Run("externalsAreIgnored", func(t *testing.T) {
		t.Parallel()

		// A (service, Uses: ExternalThing) — ExternalThing is not in spec
		contracts := []model.SpecContract{
			mkContract("A", model.ContractService, withUses("ExternalThing")),
		}

		layers, externals, err := layerer.Layer(contracts)

		require.NoError(t, err)
		require.Equal(t, []string{"ExternalThing"}, externals)
		require.Len(t, layers, 1)
		require.Equal(t, 0, layers[0].Index)
		require.Len(t, layers[0].Targets, 1)
		require.Equal(t, "A", layers[0].Targets[0].Name)
	})

	t.Run("withinLayerAlphabetical", func(t *testing.T) {
		t.Parallel()

		// 3 independent input-port contracts; all have no deps → all in layer 0
		contracts := []model.SpecContract{
			mkContract("Zebra", model.ContractInputPort),
			mkContract("Apple", model.ContractInputPort),
			mkContract("Mango", model.ContractInputPort),
		}

		layers, externals, err := layerer.Layer(contracts)

		require.NoError(t, err)
		require.Empty(t, externals)
		require.Len(t, layers, 1)
		require.Equal(t, 0, layers[0].Index)

		names := make([]string, len(layers[0].Targets))
		for i, tgt := range layers[0].Targets {
			names[i] = tgt.Name
		}
		require.Equal(t, []string{"Apple", "Mango", "Zebra"}, names)
	})

	t.Run("planTargetFields", func(t *testing.T) {
		t.Parallel()

		// Verify raw fields (Uses, Implements, DependsOn) are echoed verbatim.
		// TargetPath must be empty (set by use case, not layerer).
		contracts := []model.SpecContract{
			mkContract("UserServiceImpl", model.ContractService,
				withImplements("CreateUserUseCase"),
				withDependsOn("UserRepository"),
				withUses("SomeHelper"),
			),
			mkContract("CreateUserUseCase", model.ContractInputPort),
			mkContract("UserRepository", model.ContractOutputPort),
			mkContract("SomeHelper", model.ContractService),
		}

		layers, _, err := layerer.Layer(contracts)

		require.NoError(t, err)

		// Find UserServiceImpl target (it will be in a deeper layer).
		var target *planvo.PlanTarget
		for li := range layers {
			for ti := range layers[li].Targets {
				if layers[li].Targets[ti].Name == "UserServiceImpl" {
					t := layers[li].Targets[ti]
					target = &t
				}
			}
		}
		require.NotNil(t, target)
		require.Equal(t, "UserServiceImpl", target.Name)
		require.Equal(t, "service", target.Type)
		require.Equal(t, "", target.TargetPath)
		require.Equal(t, "CreateUserUseCase", target.Implements)
		require.Equal(t, []string{"UserRepository"}, target.DependsOn)
		require.Equal(t, []string{"SomeHelper"}, target.Uses)
	})
}
