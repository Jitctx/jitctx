package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
)

// loadSampleForTest loads the sample spring-boot-hexagonal profile from the
// testdata/fsprofile fixture so tests that verify rule ordering run against the
// actual YAML, not a synthetic copy.
func loadSampleForTest(t *testing.T) *model.FrameworkProfile {
	t.Helper()
	// Resolve testdata path relative to repo root.
	yamlPath := filepath.Join("..", "..", "..", "testdata", "fsprofile", "spring-boot-hexagonal.yaml")
	dir := t.TempDir()
	data, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "spring-boot-hexagonal.yaml"), data, 0o644))
	loader := fsprofile.New(dir)
	prof, err := loader.Load(context.Background(), "spring-boot-hexagonal")
	require.NoError(t, err)
	return prof
}

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

// TestClassifyDeclaration_SampleProfile runs two sub-cases against the real
// sample spring-boot-hexagonal YAML to lock rule ordering and coverage of
// EP01RF-012 §Business Rule (explicit @Repository mention).
func TestClassifyDeclaration_SampleProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		decl        model.JavaDeclaration
		path        string
		wantType    model.ContractType
		wantMatched bool
	}{
		{
			// EP01RF-012 §Business Rule — @Repository must classify as jpa-adapter.
			// No Gherkin scenario covers this today; this test makes the bundled rule
			// load-bearing (removing it from the YAML would cause this test to fail).
			name: "repository by annotation classifies as jpa-adapter",
			decl: model.JavaDeclaration{
				NodeType:    "class_declaration",
				Name:        "UserRepositoryImpl",
				Annotations: []string{"Repository"},
			},
			path:        "src/main/java/com/app/user/adapter/out/persistence/UserRepositoryImpl.java",
			wantType:    model.ContractJPAAdapter,
			wantMatched: true,
		},
		{
			// Sanity check on rule ordering: the @Entity rule appears before @Service
			// in the bundled YAML, so a class annotated with both must classify as
			// entity, not service (first-match-wins semantics).
			name: "entity rule beats service rule for class annotated with both Entity and Service",
			decl: model.JavaDeclaration{
				NodeType:    "class_declaration",
				Name:        "UserEntity",
				Annotations: []string{"Entity", "Service"},
			},
			path:        "src/main/java/com/app/user/domain/UserEntity.java",
			wantType:    model.ContractEntity,
			wantMatched: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bundled := loadSampleForTest(t)
			gotType, gotMatched := service.ClassifyDeclaration(tc.decl, tc.path, bundled)
			require.Equal(t, tc.wantMatched, gotMatched)
			require.Equal(t, tc.wantType, gotType)
		})
	}
}
