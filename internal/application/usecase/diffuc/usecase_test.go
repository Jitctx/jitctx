package diffuc_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/application/usecase/diffuc"
	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
	diffvo "github.com/jitctx/jitctx/internal/domain/vo/diff"
)

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

type fakeFinder struct {
	find func(ctx context.Context, feature, baseDir, plansDir string) (string, []byte, []string, error)
}

func (f fakeFinder) Find(ctx context.Context, feature, baseDir, plansDir string) (string, []byte, []string, error) {
	return f.find(ctx, feature, baseDir, plansDir)
}

type fakeParser struct {
	parse func(ctx context.Context, content string) (model.FeatureSpec, []domerr.SpecParseWarning, error)
}

func (f fakeParser) ParseSpec(ctx context.Context, content string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
	return f.parse(ctx, content)
}

type fakeManifestPort struct {
	load func(ctx context.Context) (*model.ProjectState, error)
}

func (f *fakeManifestPort) Load(ctx context.Context) (*model.ProjectState, error) {
	return f.load(ctx)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func nopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// finderOK returns a finder that always succeeds with an empty content blob.
func finderOK(path string) fakeFinder {
	return fakeFinder{find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
		return path, []byte("ignored"), nil, nil
	}}
}

// parserWith returns a parser that yields the given FeatureSpec.
func parserWith(spec model.FeatureSpec) fakeParser {
	return fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return spec, nil, nil
	}}
}

// manifestWith returns a LoadManifestPort fake backed by the given state.
func manifestWith(state *model.ProjectState) *fakeManifestPort {
	return &fakeManifestPort{load: func(_ context.Context) (*model.ProjectState, error) {
		return state, nil
	}}
}

// emptyState is a ProjectState with no modules / contracts.
func emptyState() *model.ProjectState {
	return &model.ProjectState{}
}

func newUC(
	finder fakeFinder,
	parser fakeParser,
	manifests *fakeManifestPort,
) *diffuc.Impl {
	return diffuc.New(
		finder,
		parser,
		manifests,
		service.NewContractDiffer(service.NewSignatureNormalizer()),
		service.NewDependencyLayerer(),
		nopLogger(),
	)
}

// ---------------------------------------------------------------------------
// Test: happy path — simple spec + simple manifest
// ---------------------------------------------------------------------------

// TestDiffUseCase_HappyPath drives a spec with two independent contracts
// (both absent from the manifest) and asserts the sorted action list.
//
// Expected outcome:
//   - Two CREATE actions (no manifest entries).
//   - Both land in Layer 0 (no inter-dependencies).
//   - Sorted by ContractName ASC within the same layer.
//   - Actions slice length == 2.
//   - HasChanges == true.
func TestDiffUseCase_HappyPath(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature: "order-processing",
		Module:  "orders",
		Contracts: []model.SpecContract{
			{Name: "OrderRepository", Type: model.ContractOutputPort},
			{Name: "PlaceOrderUseCase", Type: model.ContractInputPort},
		},
	}

	uc := newUC(finderOK("/plans/order-processing.md"), parserWith(spec), manifestWith(emptyState()))

	out, err := uc.Execute(context.Background(), diffvo.DiffPlanInput{Feature: "order-processing"})

	require.NoError(t, err)
	require.Equal(t, "order-processing", out.Feature)
	require.Equal(t, "orders", out.Module)
	require.True(t, out.HasChanges)
	require.Len(t, out.Actions, 2)

	// Both are CREATE; sorted alphabetically by ContractName within Layer 0.
	require.Equal(t, diffvo.DiffActionCreate, out.Actions[0].Type)
	require.Equal(t, "OrderRepository", out.Actions[0].ContractName)
	require.Equal(t, 0, out.Actions[0].Layer)

	require.Equal(t, diffvo.DiffActionCreate, out.Actions[1].Type)
	require.Equal(t, "PlaceOrderUseCase", out.Actions[1].ContractName)
	require.Equal(t, 0, out.Actions[1].Layer)
}

// ---------------------------------------------------------------------------
// Test: manifest not found
// ---------------------------------------------------------------------------

// TestDiffUseCase_ManifestNotFound asserts that ErrManifestNotFound is
// propagated UNWRAPPED so errors.Is still matches at the presentation layer.
func TestDiffUseCase_ManifestNotFound(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature:   "irrelevant",
		Contracts: []model.SpecContract{{Name: "Foo", Type: model.ContractInputPort}},
	}
	manifests := &fakeManifestPort{load: func(_ context.Context) (*model.ProjectState, error) {
		return nil, domerr.ErrManifestNotFound
	}}

	uc := newUC(finderOK("/plans/irrelevant.md"), parserWith(spec), manifests)

	_, err := uc.Execute(context.Background(), diffvo.DiffPlanInput{Feature: "irrelevant"})

	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrManifestNotFound),
		"ErrManifestNotFound must be visible through errors.Is")
}

// ---------------------------------------------------------------------------
// Test: spec not found
// ---------------------------------------------------------------------------

// TestDiffUseCase_SpecNotFound asserts that *SpecFileNotFoundError is
// propagated UNWRAPPED (matching planuc's pattern).
func TestDiffUseCase_SpecNotFound(t *testing.T) {
	t.Parallel()

	notFound := &domerr.SpecFileNotFoundError{
		Feature:  "missing-feature",
		Searched: []string{"/plans/missing-feature.md"},
	}
	finder := fakeFinder{find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
		return "", nil, nil, notFound
	}}

	uc := diffuc.New(
		finder,
		parserWith(model.FeatureSpec{}),
		manifestWith(emptyState()),
		service.NewContractDiffer(service.NewSignatureNormalizer()),
		service.NewDependencyLayerer(),
		nopLogger(),
	)

	_, err := uc.Execute(context.Background(), diffvo.DiffPlanInput{Feature: "missing-feature"})

	require.Error(t, err)
	var sfnf *domerr.SpecFileNotFoundError
	require.True(t, errors.As(err, &sfnf),
		"*SpecFileNotFoundError must be visible through errors.As")
}

// ---------------------------------------------------------------------------
// Test: empty diff (spec and manifest match perfectly)
// ---------------------------------------------------------------------------

// TestDiffUseCase_EmptyDiff verifies that when spec contracts exactly match
// manifest contracts the Actions slice is empty and HasChanges is false.
func TestDiffUseCase_EmptyDiff(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature: "user-management",
		Module:  "users",
		Contracts: []model.SpecContract{
			{Name: "UserRepository", Type: model.ContractOutputPort, Methods: []string{"User save(User user)"}},
		},
	}
	state := &model.ProjectState{
		Modules: []model.Module{
			{
				ID: "users",
				Contracts: []model.Contract{
					{
						Name: "UserRepository",
						Type: model.ContractOutputPort,
						Methods: []model.Method{
							{Signature: "User save(User user)"},
						},
					},
				},
			},
		},
	}

	uc := newUC(finderOK("/plans/user-management.md"), parserWith(spec), manifestWith(state))

	out, err := uc.Execute(context.Background(), diffvo.DiffPlanInput{Feature: "user-management"})

	require.NoError(t, err)
	require.Empty(t, out.Actions, "Actions must be empty when spec and manifest match")
	require.False(t, out.HasChanges, "HasChanges must be false when Actions is empty")
}

// ---------------------------------------------------------------------------
// Test: read-only guardrail (RNF-002) via reflection
// ---------------------------------------------------------------------------

// TestDiffUseCase_ReadOnlyGuardrail uses reflect to enumerate Impl's fields
// and asserts that none carry a write-shaped port type (no field whose type
// name ends in "SavePort", "WritePort", "WriterPort", or "SaverPort").
func TestDiffUseCase_ReadOnlyGuardrail(t *testing.T) {
	t.Parallel()

	forbiddenSuffixes := []string{
		"SavePort",
		"WritePort",
		"WriterPort",
		"SaverPort",
		"PersistPort",
	}

	implType := reflect.TypeOf(diffuc.Impl{})
	for i := range implType.NumField() {
		field := implType.Field(i)
		typeName := field.Type.Name()
		// Interface field types have their name in the underlying element.
		if field.Type.Kind() == reflect.Interface {
			typeName = field.Type.Name()
		}
		for _, suffix := range forbiddenSuffixes {
			require.False(t,
				strings.HasSuffix(typeName, suffix),
				"Impl field %q has a write-port type %q (suffix %q) — violates RNF-002",
				field.Name, typeName, suffix,
			)
		}
	}
}

// ---------------------------------------------------------------------------
// Test: determinism
// ---------------------------------------------------------------------------

// TestDiffUseCase_Determinism verifies that two consecutive Execute calls
// with the same fakes produce outputs that compare deeply equal.
func TestDiffUseCase_Determinism(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature: "payment-processing",
		Module:  "payments",
		Contracts: []model.SpecContract{
			{Name: "PaymentRepository", Type: model.ContractOutputPort},
			{Name: "ProcessPaymentUseCase", Type: model.ContractInputPort},
			{Name: "PaymentServiceImpl", Type: model.ContractService, Implements: "ProcessPaymentUseCase", DependsOn: []string{"PaymentRepository"}},
		},
	}

	state := &model.ProjectState{
		Modules: []model.Module{
			{
				ID: "payments",
				Contracts: []model.Contract{
					{Name: "LegacyPaymentHelper", Type: model.ContractService},
				},
			},
		},
	}

	makeUC := func() *diffuc.Impl {
		return newUC(finderOK("/plans/payment-processing.md"), parserWith(spec), manifestWith(state))
	}

	in := diffvo.DiffPlanInput{Feature: "payment-processing"}

	out1, err1 := makeUC().Execute(context.Background(), in)
	require.NoError(t, err1)

	out2, err2 := makeUC().Execute(context.Background(), in)
	require.NoError(t, err2)

	require.Equal(t, out1, out2,
		"two consecutive Execute calls with identical fakes must return deeply equal output")
}
