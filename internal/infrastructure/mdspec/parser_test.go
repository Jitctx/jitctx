package mdspec_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/infrastructure/mdspec"
)

func TestParser_ParseSpec_HappyPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		input           string
		wantFeature     string
		wantModule      string
		wantContracts   int
		assertContracts func(t *testing.T, contracts []model.SpecContract)
	}{
		{
			name: "complete-spec-multiple-contracts",
			input: func() string {
				data, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "mdspec", "valid", "createUser.md"))
				if err != nil {
					panic("missing fixture: " + err.Error())
				}
				return string(data)
			}(),
			wantFeature:   "create-user",
			wantModule:    "user-management",
			wantContracts: 4,
			assertContracts: func(t *testing.T, contracts []model.SpecContract) {
				t.Helper()
				require.Equal(t, "CreateUserUseCase", contracts[0].Name)
				require.Equal(t, model.ContractInputPort, contracts[0].Type)
				require.Equal(t, []string{"UserResponse execute(CreateUserCommand cmd)"}, contracts[0].Methods)

				require.Equal(t, "UserRepository", contracts[1].Name)
				require.Equal(t, model.ContractOutputPort, contracts[1].Type)
				require.Equal(t, []string{"Optional<User> findByEmail(String email)", "User save(User user)"}, contracts[1].Methods)

				require.Equal(t, "UserServiceImpl", contracts[2].Name)
				require.Equal(t, model.ContractService, contracts[2].Type)
				require.Equal(t, "CreateUserUseCase", contracts[2].Implements)
				require.Equal(t, []string{"UserRepository"}, contracts[2].DependsOn)

				require.Equal(t, "UserController", contracts[3].Name)
				require.Equal(t, model.ContractRestAdapter, contracts[3].Type)
				require.Equal(t, []string{"CreateUserUseCase"}, contracts[3].Uses)
				require.Equal(t, []string{"POST /users"}, contracts[3].Endpoints)
			},
		},
		{
			name:          "inline-list-field",
			input:         "# Feature: f\nModule: m\n\n## Contract: A\nType: service\nUses: X, Y, Z\n",
			wantFeature:   "f",
			wantModule:    "m",
			wantContracts: 1,
			assertContracts: func(t *testing.T, contracts []model.SpecContract) {
				t.Helper()
				require.Equal(t, []string{"X", "Y", "Z"}, contracts[0].Uses)
			},
		},
		{
			name:          "multiline-list-field",
			input:         "# Feature: f\nModule: m\n\n## Contract: A\nType: service\nMethods:\n- foo()\n- bar(int x)\n",
			wantFeature:   "f",
			wantModule:    "m",
			wantContracts: 1,
			assertContracts: func(t *testing.T, contracts []model.SpecContract) {
				t.Helper()
				require.Equal(t, []string{"foo()", "bar(int x)"}, contracts[0].Methods)
			},
		},
		{
			name:          "single-contract-minimal",
			input:         "# Feature: minimal\nModule: core\n\n## Contract: OnlyType\nType: entity\n",
			wantFeature:   "minimal",
			wantModule:    "core",
			wantContracts: 1,
			assertContracts: func(t *testing.T, contracts []model.SpecContract) {
				t.Helper()
				require.Equal(t, "OnlyType", contracts[0].Name)
				require.Equal(t, model.ContractEntity, contracts[0].Type)
				require.Empty(t, contracts[0].Methods)
				require.Empty(t, contracts[0].Fields)
				require.Empty(t, contracts[0].Uses)
				require.Empty(t, contracts[0].Implements)
				require.Empty(t, contracts[0].DependsOn)
				require.Empty(t, contracts[0].Endpoints)
				require.Empty(t, contracts[0].Annotations)
			},
		},
		{
			name:          "entity-with-fields",
			input:         "# Feature: f\nModule: m\n\n## Contract: User\nType: aggregate-root\nFields:\n- UUID id\n- String email\n",
			wantFeature:   "f",
			wantModule:    "m",
			wantContracts: 1,
			assertContracts: func(t *testing.T, contracts []model.SpecContract) {
				t.Helper()
				require.Equal(t, model.ContractAggregate, contracts[0].Type)
				require.Equal(t, []string{"UUID id", "String email"}, contracts[0].Fields)
			},
		},
		{
			name:          "implements-single-value",
			input:         "# Feature: f\nModule: m\n\n## Contract: Svc\nType: service\nImplements: CreateUserUseCase\n",
			wantFeature:   "f",
			wantModule:    "m",
			wantContracts: 1,
			assertContracts: func(t *testing.T, contracts []model.SpecContract) {
				t.Helper()
				require.Equal(t, "CreateUserUseCase", contracts[0].Implements)
			},
		},
		{
			name:          "depends-on-inline",
			input:         "# Feature: f\nModule: m\n\n## Contract: Svc\nType: service\nDependsOn: A, B\n",
			wantFeature:   "f",
			wantModule:    "m",
			wantContracts: 1,
			assertContracts: func(t *testing.T, contracts []model.SpecContract) {
				t.Helper()
				require.Equal(t, []string{"A", "B"}, contracts[0].DependsOn)
			},
		},
		{
			name:          "endpoints-multiline",
			input:         "# Feature: f\nModule: m\n\n## Contract: Ctrl\nType: rest-adapter\nEndpoints:\n- POST /users\n- GET /users/{id}\n",
			wantFeature:   "f",
			wantModule:    "m",
			wantContracts: 1,
			assertContracts: func(t *testing.T, contracts []model.SpecContract) {
				t.Helper()
				require.Equal(t, []string{"POST /users", "GET /users/{id}"}, contracts[0].Endpoints)
			},
		},
		{
			name:          "annotations-inline",
			input:         "# Feature: f\nModule: m\n\n## Contract: Svc\nType: service\nAnnotations: Transactional, Cacheable\n",
			wantFeature:   "f",
			wantModule:    "m",
			wantContracts: 1,
			assertContracts: func(t *testing.T, contracts []model.SpecContract) {
				t.Helper()
				require.Equal(t, []string{"Transactional", "Cacheable"}, contracts[0].Annotations)
			},
		},
		{
			name:          "blank-lines-between-contracts",
			input:         "# Feature: f\nModule: m\n\n## Contract: A\nType: entity\n\n\n## Contract: B\nType: service\n",
			wantFeature:   "f",
			wantModule:    "m",
			wantContracts: 2,
			assertContracts: func(t *testing.T, contracts []model.SpecContract) {
				t.Helper()
				require.Equal(t, "A", contracts[0].Name)
				require.Equal(t, "B", contracts[1].Name)
			},
		},
		{
			name:          "trailing-blank-lines",
			input:         "# Feature: f\nModule: m\n\n## Contract: A\nType: entity\n\n\n",
			wantFeature:   "f",
			wantModule:    "m",
			wantContracts: 1,
			assertContracts: func(t *testing.T, contracts []model.SpecContract) {
				t.Helper()
				require.Equal(t, "A", contracts[0].Name)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := mdspec.New()
			spec, warnings, err := p.ParseSpec(context.Background(), tc.input)
			require.NoError(t, err)
			require.Empty(t, warnings)
			require.Equal(t, tc.wantFeature, spec.Feature)
			require.Equal(t, tc.wantModule, spec.Module)
			require.Len(t, spec.Contracts, tc.wantContracts)
			if tc.assertContracts != nil {
				tc.assertContracts(t, spec.Contracts)
			}
		})
	}
}

func TestParser_ParseSpec_ErrorPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		input            string
		wantErrSpecParse bool
		wantDuplicate    bool
		errMsgContains   string
	}{
		{
			name:             "missing-feature-header",
			input:            "hello\n",
			wantErrSpecParse: true,
			errMsgContains:   "expected '# Feature:",
		},
		{
			name:             "missing-module",
			input:            "# Feature: x\n\n## Contract: A\nType: entity\n",
			wantErrSpecParse: true,
			errMsgContains:   "Module",
		},
		{
			name:             "duplicate-contract-name",
			input:            "# Feature: f\nModule: m\n\n## Contract: User\nType: entity\n\n## Contract: User\nType: service\n",
			wantErrSpecParse: true,
			wantDuplicate:    true,
			errMsgContains:   "User",
		},
		{
			name:             "missing-contract-type",
			input:            "# Feature: f\nModule: m\n\n## Contract: NoType\n",
			wantErrSpecParse: true,
			errMsgContains:   "Type",
		},
		{
			name:             "empty-content",
			input:            "",
			wantErrSpecParse: true,
			errMsgContains:   "Feature",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := mdspec.New()
			_, _, err := p.ParseSpec(context.Background(), tc.input)
			require.Error(t, err)

			if tc.wantErrSpecParse {
				require.True(t, errors.Is(err, domerr.ErrSpecParse),
					"expected errors.Is(err, ErrSpecParse) to be true, got: %v", err)
			}

			if tc.wantDuplicate {
				var dupErr *domerr.DuplicateContractError
				require.True(t, errors.As(err, &dupErr),
					"expected errors.As(err, &DuplicateContractError) to be true, got: %v", err)
				require.Greater(t, dupErr.FirstLine, 0)
				require.Greater(t, dupErr.DupeLine, dupErr.FirstLine)
			}

			if tc.errMsgContains != "" {
				require.Contains(t, err.Error(), tc.errMsgContains)
			}
		})
	}
}

func TestParser_ParseSpec_WarningPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		input           string
		wantWarnings    int
		warnMsgContains string
	}{
		{
			name:            "unknown-field-produces-warning",
			input:           "# Feature: f\nModule: m\n\n## Contract: A\nType: service\nMagic: yes\n",
			wantWarnings:    1,
			warnMsgContains: "unknown field",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := mdspec.New()
			spec, warnings, err := p.ParseSpec(context.Background(), tc.input)
			require.NoError(t, err)
			require.NotEmpty(t, spec.Feature)
			require.Len(t, warnings, tc.wantWarnings)
			if tc.warnMsgContains != "" {
				require.Contains(t, warnings[0].Message, tc.warnMsgContains)
				require.Greater(t, warnings[0].Line, 0)
			}
		})
	}
}

func TestParser_ParseSpec_CancelledContext(t *testing.T) {
	t.Parallel()

	p := mdspec.New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := p.ParseSpec(ctx, "# Feature: f\nModule: m\n")
	require.ErrorIs(t, err, context.Canceled)
}

func TestParser_PackageField_Recognized(t *testing.T) {
	t.Parallel()

	input := "# Feature: x\nModule: m\nPackage: com.example.x\n\n## Contract: Foo\nType: input-port\nMethods:\n- void hello()\n"

	p := mdspec.New()
	spec, warnings, err := p.ParseSpec(context.Background(), input)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Equal(t, "x", spec.Feature)
	require.Equal(t, "m", spec.Module)
	require.Equal(t, "com.example.x", spec.Package)
	require.Len(t, spec.Contracts, 1)
}

func TestParser_PackageField_OptionalAbsence(t *testing.T) {
	t.Parallel()

	input := "# Feature: x\nModule: m\n\n## Contract: Foo\nType: input-port\nMethods:\n- void hello()\n"

	p := mdspec.New()
	spec, warnings, err := p.ParseSpec(context.Background(), input)

	require.NoError(t, err)
	require.Equal(t, "", spec.Package)
	for _, w := range warnings {
		require.NotContains(t, w.Message, "Package")
	}
}

func TestParser_PackageField_BeforeModule(t *testing.T) {
	t.Parallel()

	input := "# Feature: x\nPackage: com.example.x\nModule: m\n\n## Contract: Foo\nType: input-port\nMethods:\n- void hello()\n"

	p := mdspec.New()
	spec, warnings, err := p.ParseSpec(context.Background(), input)

	require.NoError(t, err)
	require.Equal(t, "com.example.x", spec.Package)
	require.Equal(t, "m", spec.Module)
	require.NotEmpty(t, warnings)
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "Package") {
			found = true
			break
		}
	}
	require.True(t, found, "expected at least one warning mentioning 'Package', got: %v", warnings)
}
