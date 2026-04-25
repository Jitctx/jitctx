package service_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
)

func TestContractPathMapper_Map(t *testing.T) {
	t.Parallel()

	mapper := service.NewContractPathMapper()

	cases := []struct {
		name      string
		ctype     model.ContractType
		inputName string
		wantPath  string
		wantErr   bool
	}{
		{
			name:      "input_port",
			ctype:     model.ContractInputPort,
			inputName: "Foo",
			wantPath:  "port/in/Foo.java",
		},
		{
			name:      "output_port",
			ctype:     model.ContractOutputPort,
			inputName: "Bar",
			wantPath:  "port/out/Bar.java",
		},
		{
			name:      "service",
			ctype:     model.ContractService,
			inputName: "X",
			wantPath:  "application/X.java",
		},
		{
			name:      "rest_adapter",
			ctype:     model.ContractRestAdapter,
			inputName: "X",
			wantPath:  "adapter/in/web/X.java",
		},
		{
			name:      "entity",
			ctype:     model.ContractEntity,
			inputName: "User",
			wantPath:  "domain/User.java",
		},
		{
			name:      "aggregate_root",
			ctype:     model.ContractAggregate,
			inputName: "Order",
			wantPath:  "domain/Order.java",
		},
		{
			name:      "jpa_adapter",
			ctype:     model.ContractJPAAdapter,
			inputName: "X",
			wantPath:  "adapter/out/persistence/X.java",
		},
		{
			name:      "unsupported",
			ctype:     model.ContractType("weird-thing"),
			inputName: "X",
			wantErr:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := mapper.Map(tc.ctype, tc.inputName)

			if tc.wantErr {
				require.Error(t, err)
				require.True(t, errors.Is(err, domerr.ErrUnsupportedContractType))

				var uct *domerr.UnsupportedContractTypeError
				require.True(t, errors.As(err, &uct))
				require.Equal(t, "weird-thing", uct.Type)
				require.Equal(t, []string{
					"aggregate-root",
					"entity",
					"input-port",
					"jpa-adapter",
					"output-port",
					"rest-adapter",
					"service",
				}, uct.SupportedSorted)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantPath, got)
		})
	}
}
