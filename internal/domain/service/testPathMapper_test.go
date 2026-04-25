package service_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
)

func TestTestPathMapper_Map(t *testing.T) {
	t.Parallel()

	mapper := service.NewTestPathMapper()

	cases := []struct {
		name      string
		ctype     model.ContractType
		inputName string
		wantPath  string
		wantErr   bool
	}{
		{
			name:      "service",
			ctype:     model.ContractService,
			inputName: "UserServiceImpl",
			wantPath:  "application/UserServiceImplTest.java",
		},
		{
			name:      "rest_adapter",
			ctype:     model.ContractRestAdapter,
			inputName: "UserController",
			wantPath:  "adapter/in/web/UserControllerTest.java",
		},
		{
			name:      "entity",
			ctype:     model.ContractEntity,
			inputName: "User",
			wantPath:  "domain/UserTest.java",
		},
		{
			name:      "aggregate_root",
			ctype:     model.ContractAggregate,
			inputName: "User",
			wantPath:  "domain/UserTest.java",
		},
		{
			name:      "input_port",
			ctype:     model.ContractInputPort,
			inputName: "CreateUserUseCase",
			wantPath:  "",
		},
		{
			name:      "output_port",
			ctype:     model.ContractOutputPort,
			inputName: "UserRepository",
			wantPath:  "",
		},
		{
			name:      "jpa_adapter",
			ctype:     model.ContractJPAAdapter,
			inputName: "UserRepositoryJpa",
			wantPath:  "",
		},
		{
			name:      "unsupported",
			ctype:     model.ContractType("foo"),
			inputName: "Anything",
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
				require.Equal(t, "foo", uct.Type)
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
