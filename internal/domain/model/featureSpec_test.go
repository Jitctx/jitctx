package model_test

import (
	"testing"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/stretchr/testify/require"
)

func TestFeatureSpec_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var spec model.FeatureSpec

	require.Equal(t, "", spec.Feature)
	require.Equal(t, "", spec.Module)
	require.Nil(t, spec.Contracts)
}

func TestSpecContract_FieldsAccessible(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		contractType model.ContractType
	}{
		{"input-port", model.ContractInputPort},
		{"output-port", model.ContractOutputPort},
		{"entity", model.ContractEntity},
		{"aggregate-root", model.ContractAggregate},
		{"service", model.ContractService},
		{"rest-adapter", model.ContractRestAdapter},
		{"jpa-adapter", model.ContractJPAAdapter},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sc := model.SpecContract{
				Name: "SomeContract",
				Type: tc.contractType,
			}

			require.Equal(t, "SomeContract", sc.Name)
			require.Equal(t, tc.contractType, sc.Type)
		})
	}
}

func TestFeatureSpec_PopulatedRetainsOrderAndValues(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature: "create-user",
		Module:  "user-management",
		Contracts: []model.SpecContract{
			{
				Name:        "CreateUserUseCase",
				Type:        model.ContractInputPort,
				Methods:     []string{"UserResponse execute(CreateUserCommand cmd)"},
				Implements:  "",
				DependsOn:   nil,
				Endpoints:   nil,
				Annotations: nil,
			},
			{
				Name:      "UserRepository",
				Type:      model.ContractOutputPort,
				Methods:   []string{"Optional<User> findByEmail(String email)"},
				Fields:    nil,
				Uses:      nil,
				DependsOn: nil,
				Endpoints: nil,
			},
			{
				Name:       "CreateUserService",
				Type:       model.ContractService,
				Implements: "CreateUserUseCase",
				DependsOn:  []string{"UserRepository"},
			},
			{
				Name:      "UserController",
				Type:      model.ContractRestAdapter,
				Uses:      []string{"CreateUserUseCase"},
				Endpoints: []string{"POST /users"},
			},
		},
	}

	require.Equal(t, "create-user", spec.Feature)
	require.Equal(t, "user-management", spec.Module)
	require.Len(t, spec.Contracts, 4)

	// Verify order is retained
	require.Equal(t, "CreateUserUseCase", spec.Contracts[0].Name)
	require.Equal(t, model.ContractInputPort, spec.Contracts[0].Type)
	require.Equal(t, []string{"UserResponse execute(CreateUserCommand cmd)"}, spec.Contracts[0].Methods)

	require.Equal(t, "UserRepository", spec.Contracts[1].Name)
	require.Equal(t, model.ContractOutputPort, spec.Contracts[1].Type)

	require.Equal(t, "CreateUserService", spec.Contracts[2].Name)
	require.Equal(t, model.ContractService, spec.Contracts[2].Type)
	require.Equal(t, "CreateUserUseCase", spec.Contracts[2].Implements)
	require.Equal(t, []string{"UserRepository"}, spec.Contracts[2].DependsOn)

	require.Equal(t, "UserController", spec.Contracts[3].Name)
	require.Equal(t, model.ContractRestAdapter, spec.Contracts[3].Type)
	require.Equal(t, []string{"CreateUserUseCase"}, spec.Contracts[3].Uses)
	require.Equal(t, []string{"POST /users"}, spec.Contracts[3].Endpoints)
}
