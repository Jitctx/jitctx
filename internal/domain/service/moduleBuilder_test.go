package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
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

	modules := service.BuildModules(summaries, prof)
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

	modules := service.BuildModules(summaries, prof)
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

	modules := service.BuildModules(summaries, prof)
	require.Len(t, modules, 1)
	// user_management → user-management
	require.Equal(t, "user-management", modules[0].ID)
}
