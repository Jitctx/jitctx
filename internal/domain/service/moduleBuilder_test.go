package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

func TestBuildModules_SingleFile(t *testing.T) {
	t.Parallel()

	prof := &model.FrameworkProfile{
		Rules: []model.ProfileRule{
			{Match: model.ProfileMatch{NodeType: "interface_declaration", PathContains: "/port/in/"}, ClassifyAs: model.ContractInputPort},
		},
	}

	summaries := []model.JavaFileSummary{
		{
			Path:    "src/main/java/com/app/user_management/port/in/CreateUserUseCase.java",
			Package: "com.app.user_management.port.in",
			Declarations: []model.JavaDeclaration{
				{NodeType: "interface_declaration", Name: "CreateUserUseCase"},
			},
		},
	}

	modules, err := service.BuildModules(context.Background(), nil, summaries, prof, nil)
	require.NoError(t, err)
	require.Len(t, modules, 1)
	require.Equal(t, "user-management", modules[0].ID)
	require.Len(t, modules[0].Contracts, 1)
	require.Equal(t, "CreateUserUseCase", modules[0].Contracts[0].Name)
}

func TestBuildModules_UnclassifiedDropped(t *testing.T) {
	t.Parallel()

	prof := &model.FrameworkProfile{
		Rules: []model.ProfileRule{
			{Match: model.ProfileMatch{NodeType: "interface_declaration", PathContains: "/port/in/"}, ClassifyAs: model.ContractInputPort},
		},
	}

	summaries := []model.JavaFileSummary{
		{
			Path:    "src/main/java/com/app/user_management/util/Helper.java",
			Package: "com.app.user_management.util",
			Declarations: []model.JavaDeclaration{
				{NodeType: "class_declaration", Name: "Helper"},
			},
		},
	}

	modules, err := service.BuildModules(context.Background(), nil, summaries, prof, nil)
	require.NoError(t, err)
	require.Len(t, modules, 0)
}

func TestBuildModules_IDNormalization(t *testing.T) {
	t.Parallel()

	prof := &model.FrameworkProfile{
		Rules: []model.ProfileRule{
			{Match: model.ProfileMatch{NodeType: "interface_declaration", PathContains: "/port/in/"}, ClassifyAs: model.ContractInputPort},
		},
	}

	summaries := []model.JavaFileSummary{
		{
			Path:    "src/main/java/com/app/user_management/port/in/CreateUserUseCase.java",
			Package: "com.app.user_management.port.in",
			Declarations: []model.JavaDeclaration{
				{NodeType: "interface_declaration", Name: "CreateUserUseCase"},
			},
		},
	}

	modules, err := service.BuildModules(context.Background(), nil, summaries, prof, nil)
	require.NoError(t, err)
	require.Len(t, modules, 1)
	// user_management → user-management
	require.Equal(t, "user-management", modules[0].ID)
}

func TestBuildModules_DeclarativePath_TypesPopulated(t *testing.T) {
	t.Parallel()

	// When typesDecl is non-nil, BuildModules uses the declarative path and
	// the resulting contracts have Types populated from the classifier's response.
	typesDecl := []model.ProfileTypeDeclaration{
		{ID: "output-adapter"},
		{ID: "cacheable-component"},
	}

	summaries := []model.JavaFileSummary{
		{
			Path:    "src/main/java/com/app/cache/adapter/out/UserCacheAdapter.java",
			Package: "com.app.cache.adapter.out",
			Declarations: []model.JavaDeclaration{
				{
					NodeType:   "class_declaration",
					Name:       "UserCacheAdapter",
					Implements: []string{"Repository", "Cacheable"},
				},
			},
		},
	}

	stub := &fakeClassifyDeclarativePort{
		fn: func(_ context.Context, input profilevo.ClassificationInput, _ []model.ProfileTypeDeclaration) ([]string, error) {
			if input.Name == "UserCacheAdapter" {
				return []string{"output-adapter", "cacheable-component"}, nil
			}
			return []string{}, nil
		},
	}

	modules, err := service.BuildModules(t.Context(), stub, summaries, nil, typesDecl)
	require.NoError(t, err)
	require.Len(t, modules, 1)
	require.Equal(t, "cache", modules[0].ID)
	require.Len(t, modules[0].Contracts, 1)
	require.Equal(t, "UserCacheAdapter", modules[0].Contracts[0].Name)
	require.Equal(t, []string{"output-adapter", "cacheable-component"}, modules[0].Contracts[0].Types)
}
