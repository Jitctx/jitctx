package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/port/profile"
	"github.com/jitctx/jitctx/internal/domain/service"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// fakeClassifyDeclarativePort is a hand-rolled stub for profile.ClassifyDeclarativePort.
// The fn field is set per test case to script any response.
type fakeClassifyDeclarativePort struct {
	fn func(ctx context.Context, input profilevo.ClassificationInput, types []model.ProfileTypeDeclaration) ([]string, error)
}

func (f *fakeClassifyDeclarativePort) ClassifyDeclarative(
	ctx context.Context,
	input profilevo.ClassificationInput,
	types []model.ProfileTypeDeclaration,
) ([]string, error) {
	return f.fn(ctx, input, types)
}

// Compile-time assertion that fakeClassifyDeclarativePort satisfies the port.
var _ profile.ClassifyDeclarativePort = (*fakeClassifyDeclarativePort)(nil)

// twoTypeDecls returns the two-type profile used by the multi-tag bundle fixture:
//   - "output-adapter" matched by implements_all=[Repository]
//   - "cacheable-component" matched by implements_all=[Cacheable]
func twoTypeDecls() []model.ProfileTypeDeclaration {
	return []model.ProfileTypeDeclaration{
		{
			ID: "output-adapter",
			Classification: []model.ClassificationRule{
				{ImplementsAll: []string{"Repository"}},
			},
		},
		{
			ID: "cacheable-component",
			Classification: []model.ClassificationRule{
				{ImplementsAll: []string{"Cacheable"}},
			},
		},
	}
}

// singleDeclarationSummary builds a JavaFileSummary with one declaration.
func singleDeclarationSummary(name string, implements []string) model.JavaFileSummary {
	return model.JavaFileSummary{
		Path:    "src/main/java/com/app/cache/adapter/out/UserCacheAdapter.java",
		Package: "com.app.cache.adapter.out",
		Declarations: []model.JavaDeclaration{
			{
				NodeType:   "class_declaration",
				Name:       name,
				Implements: implements,
			},
		},
	}
}

func TestClassifyAndBuildContracts_Scenario1_MultipleMatchingTypes(t *testing.T) {
	t.Parallel()

	// Scenario 1: a class implementing both Repository and Cacheable
	// gets Types=["output-adapter", "cacheable-component"] in declared profile order.
	typesDecl := twoTypeDecls()
	summary := singleDeclarationSummary("UserCacheAdapter", []string{"Repository", "Cacheable"})

	// The fake returns both type IDs in declared profile order (as a real classifier would).
	stub := &fakeClassifyDeclarativePort{
		fn: func(_ context.Context, input profilevo.ClassificationInput, _ []model.ProfileTypeDeclaration) ([]string, error) {
			if input.Name == "UserCacheAdapter" {
				return []string{"output-adapter", "cacheable-component"}, nil
			}
			return []string{}, nil
		},
	}

	contracts, err := service.ClassifyAndBuildContracts(t.Context(), stub, summary, typesDecl)
	require.NoError(t, err)
	require.Len(t, contracts, 1)
	require.Equal(t, "UserCacheAdapter", contracts[0].Name)
	require.Equal(t, []string{"output-adapter", "cacheable-component"}, contracts[0].Types)
}

func TestClassifyAndBuildContracts_Scenario2_NoMatchingType(t *testing.T) {
	t.Parallel()

	// Scenario 2: a class with no matching type produces a Contract with Types=[]
	// (empty non-nil slice, per the builder's normalisation rule).
	typesDecl := twoTypeDecls()
	summary := singleDeclarationSummary("UnrelatedHelper", []string{})

	stub := &fakeClassifyDeclarativePort{
		fn: func(_ context.Context, _ profilevo.ClassificationInput, _ []model.ProfileTypeDeclaration) ([]string, error) {
			return []string{}, nil
		},
	}

	contracts, err := service.ClassifyAndBuildContracts(t.Context(), stub, summary, typesDecl)
	require.NoError(t, err)
	require.Len(t, contracts, 1)
	require.Equal(t, "UnrelatedHelper", contracts[0].Name)
	// Must be non-nil empty slice (not nil) — RF-005 Scenario 2.
	require.NotNil(t, contracts[0].Types)
	require.Empty(t, contracts[0].Types)
}

func TestClassifyAndBuildContracts_EmptyTypesDecl_ReturnsNil(t *testing.T) {
	t.Parallel()

	// When typesDecl is empty, the function returns (nil, nil) — the caller
	// is responsible for the legacy fallback path.
	summary := singleDeclarationSummary("SomeClass", []string{"Foo"})
	stub := &fakeClassifyDeclarativePort{
		fn: func(_ context.Context, _ profilevo.ClassificationInput, _ []model.ProfileTypeDeclaration) ([]string, error) {
			return []string{"some-type"}, nil
		},
	}

	contracts, err := service.ClassifyAndBuildContracts(t.Context(), stub, summary, nil)
	require.NoError(t, err)
	require.Nil(t, contracts)
}

func TestClassifyAndBuildContracts_CtxCancellationPropagates(t *testing.T) {
	t.Parallel()

	typesDecl := twoTypeDecls()
	summary := singleDeclarationSummary("SomeClass", []string{"Repository"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling

	stub := &fakeClassifyDeclarativePort{
		fn: func(ctx context.Context, _ profilevo.ClassificationInput, _ []model.ProfileTypeDeclaration) ([]string, error) {
			return nil, ctx.Err()
		},
	}

	contracts, err := service.ClassifyAndBuildContracts(ctx, stub, summary, typesDecl)
	require.Nil(t, contracts)
	require.True(t, errors.Is(err, context.Canceled))
}

func TestClassifyAndBuildContracts_DeclaredOrderPreserved(t *testing.T) {
	t.Parallel()

	// Three types declared; the class matches all three.
	// The returned order must match the order the fake returns (which mirrors declared profile order).
	typesDecl := []model.ProfileTypeDeclaration{
		{ID: "alpha"},
		{ID: "beta"},
		{ID: "gamma"},
	}
	summary := model.JavaFileSummary{
		Path:    "src/main/java/com/app/mod/SomeClass.java",
		Package: "com.app.mod",
		Declarations: []model.JavaDeclaration{
			{NodeType: "class_declaration", Name: "SomeClass"},
		},
	}

	stub := &fakeClassifyDeclarativePort{
		fn: func(_ context.Context, _ profilevo.ClassificationInput, _ []model.ProfileTypeDeclaration) ([]string, error) {
			return []string{"alpha", "beta", "gamma"}, nil
		},
	}

	contracts, err := service.ClassifyAndBuildContracts(t.Context(), stub, summary, typesDecl)
	require.NoError(t, err)
	require.Len(t, contracts, 1)
	require.Equal(t, []string{"alpha", "beta", "gamma"}, contracts[0].Types)
}

func TestClassifyAndBuildContracts_DeduplicationPassthrough(t *testing.T) {
	t.Parallel()

	// If the classifier (stub) returns duplicates, ClassifyAndBuildContracts assigns
	// the slice verbatim — deduplication is the classifier's responsibility, not the builder's.
	// This test documents the contract: the builder does NOT deduplicate.
	typesDecl := []model.ProfileTypeDeclaration{
		{ID: "output-adapter"},
	}
	summary := singleDeclarationSummary("Dup", []string{"Repository"})

	stub := &fakeClassifyDeclarativePort{
		fn: func(_ context.Context, _ profilevo.ClassificationInput, _ []model.ProfileTypeDeclaration) ([]string, error) {
			return []string{"output-adapter", "output-adapter"}, nil
		},
	}

	contracts, err := service.ClassifyAndBuildContracts(t.Context(), stub, summary, typesDecl)
	require.NoError(t, err)
	require.Len(t, contracts, 1)
	// Builder preserves whatever the classifier returns.
	require.Equal(t, []string{"output-adapter", "output-adapter"}, contracts[0].Types)
}

func TestClassifyAndBuildContracts_MultipleDeclarations(t *testing.T) {
	t.Parallel()

	// A file with two declarations; each is classified independently.
	typesDecl := twoTypeDecls()
	summary := model.JavaFileSummary{
		Path:    "src/main/java/com/app/order/adapter/out/OrderRepo.java",
		Package: "com.app.order.adapter.out",
		Declarations: []model.JavaDeclaration{
			{NodeType: "class_declaration", Name: "OrderRepoImpl", Implements: []string{"Repository"}},
			{NodeType: "class_declaration", Name: "OrderCacheImpl", Implements: []string{"Cacheable"}},
		},
	}

	stub := &fakeClassifyDeclarativePort{
		fn: func(_ context.Context, input profilevo.ClassificationInput, _ []model.ProfileTypeDeclaration) ([]string, error) {
			switch input.Name {
			case "OrderRepoImpl":
				return []string{"output-adapter"}, nil
			case "OrderCacheImpl":
				return []string{"cacheable-component"}, nil
			default:
				return []string{}, nil
			}
		},
	}

	contracts, err := service.ClassifyAndBuildContracts(t.Context(), stub, summary, typesDecl)
	require.NoError(t, err)
	require.Len(t, contracts, 2)
	require.Equal(t, "OrderRepoImpl", contracts[0].Name)
	require.Equal(t, []string{"output-adapter"}, contracts[0].Types)
	require.Equal(t, "OrderCacheImpl", contracts[1].Name)
	require.Equal(t, []string{"cacheable-component"}, contracts[1].Types)
}
