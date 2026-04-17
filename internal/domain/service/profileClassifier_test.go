package service_test

import (
	"testing"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
)

func TestClassifyDeclaration(t *testing.T) {
	t.Parallel()

	prof := &model.FrameworkProfile{
		Rules: []model.ProfileRule{
			{Match: model.ProfileMatch{NodeType: "interface_declaration", PathContains: "/port/in/"}, ClassifyAs: model.ContractInputPort},
			{Match: model.ProfileMatch{NodeType: "interface_declaration", PathContains: "/port/out/"}, ClassifyAs: model.ContractOutputPort},
			{Match: model.ProfileMatch{NodeType: "class_declaration", HasAnnotation: "Entity"}, ClassifyAs: model.ContractEntity},
			{Match: model.ProfileMatch{NodeType: "class_declaration", HasAnnotation: "RestController"}, ClassifyAs: model.ContractRestAdapter},
			{Match: model.ProfileMatch{NodeType: "class_declaration", Implements: "*UseCase", PathContains: "/service/"}, ClassifyAs: model.ContractService},
		},
	}

	tests := []struct {
		name        string
		decl        model.JavaDeclaration
		path        string
		wantType    model.ContractType
		wantMatched bool
	}{
		{
			name:        "input port by path and node type",
			decl:        model.JavaDeclaration{NodeType: "interface_declaration", Name: "CreateUserUseCase"},
			path:        "src/main/java/com/app/user/port/in/CreateUserUseCase.java",
			wantType:    model.ContractInputPort,
			wantMatched: true,
		},
		{
			name:        "output port by path",
			decl:        model.JavaDeclaration{NodeType: "interface_declaration", Name: "UserRepository"},
			path:        "src/main/java/com/app/user/port/out/UserRepository.java",
			wantType:    model.ContractOutputPort,
			wantMatched: true,
		},
		{
			name:        "entity by annotation",
			decl:        model.JavaDeclaration{NodeType: "class_declaration", Name: "User", Annotations: []string{"Entity"}},
			path:        "src/main/java/com/app/user/domain/User.java",
			wantType:    model.ContractEntity,
			wantMatched: true,
		},
		{
			name:        "rest adapter by annotation",
			decl:        model.JavaDeclaration{NodeType: "class_declaration", Name: "UserController", Annotations: []string{"RestController"}},
			path:        "src/main/java/com/app/user/adapter/in/web/UserController.java",
			wantType:    model.ContractRestAdapter,
			wantMatched: true,
		},
		{
			name:        "service by implements and path",
			decl:        model.JavaDeclaration{NodeType: "class_declaration", Name: "UserServiceImpl", Implements: []string{"CreateUserUseCase"}},
			path:        "src/main/java/com/app/user/service/UserServiceImpl.java",
			wantType:    model.ContractService,
			wantMatched: true,
		},
		{
			name:        "no match",
			decl:        model.JavaDeclaration{NodeType: "class_declaration", Name: "SomeHelper"},
			path:        "src/main/java/com/app/user/util/SomeHelper.java",
			wantType:    "",
			wantMatched: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotType, gotMatched := service.ClassifyDeclaration(tc.decl, tc.path, prof)
			if gotMatched != tc.wantMatched {
				t.Errorf("ClassifyDeclaration matched=%v, want %v", gotMatched, tc.wantMatched)
			}
			if gotType != tc.wantType {
				t.Errorf("ClassifyDeclaration type=%v, want %v", gotType, tc.wantType)
			}
		})
	}
}
