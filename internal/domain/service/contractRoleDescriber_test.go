package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
)

func TestContractRoleDescriber_Describe(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		contract model.SpecContract
		wantRole string
	}{
		{
			name:     "InputPort",
			contract: model.SpecContract{Type: model.ContractInputPort},
			wantRole: "Input port (use case interface)",
		},
		{
			name:     "OutputPort",
			contract: model.SpecContract{Type: model.ContractOutputPort},
			wantRole: "Output port (driven port)",
		},
		{
			name:     "Entity",
			contract: model.SpecContract{Type: model.ContractEntity},
			wantRole: "Domain entity",
		},
		{
			name:     "Aggregate",
			contract: model.SpecContract{Type: model.ContractAggregate},
			wantRole: "Aggregate root",
		},
		{
			name: "ServiceWithBoth",
			contract: model.SpecContract{
				Type:       model.ContractService,
				Implements: "X",
				DependsOn:  []string{"Y", "Z"},
			},
			wantRole: "Service implementing X; depends on Y, Z",
		},
		{
			name: "ServiceWithImplementsOnly",
			contract: model.SpecContract{
				Type:       model.ContractService,
				Implements: "X",
			},
			wantRole: "Service implementing X",
		},
		{
			name: "ServiceWithDependsOnly",
			contract: model.SpecContract{
				Type:      model.ContractService,
				DependsOn: []string{"Y"},
			},
			wantRole: "Service depends on Y",
		},
		{
			name:     "ServicePlain",
			contract: model.SpecContract{Type: model.ContractService},
			wantRole: "Service",
		},
		{
			name: "RestAdapterWithUses",
			contract: model.SpecContract{
				Type: model.ContractRestAdapter,
				Uses: []string{"X", "Y"},
			},
			wantRole: "REST adapter; calls X, Y",
		},
		{
			name:     "RestAdapterEmpty",
			contract: model.SpecContract{Type: model.ContractRestAdapter},
			wantRole: "REST adapter",
		},
		{
			name: "JPAAdapterWithImplements",
			contract: model.SpecContract{
				Type:       model.ContractJPAAdapter,
				Implements: "X",
			},
			wantRole: "JPA adapter implementing X",
		},
		{
			name:     "JPAAdapterPlain",
			contract: model.SpecContract{Type: model.ContractJPAAdapter},
			wantRole: "JPA adapter",
		},
		{
			name:     "Unknown",
			contract: model.SpecContract{Type: model.ContractType("weird")},
			wantRole: "Contract of type weird",
		},
	}

	describer := service.NewContractRoleDescriber()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := describer.Describe(tc.contract)
			require.Equal(t, tc.wantRole, got)
		})
	}
}
