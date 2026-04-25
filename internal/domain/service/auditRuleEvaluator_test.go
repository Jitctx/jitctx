package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
	auditvo "github.com/jitctx/jitctx/internal/domain/vo/audit"
)

const testModuleID = "mod-user-management"

// newEvaluator returns a fresh AuditEvaluator for each test.
func newEvaluator() *service.AuditEvaluator { return service.NewAuditEvaluator() }

// ---------------------------------------------------------------------------
// annotation_path_mismatch
// ---------------------------------------------------------------------------

func TestAuditEvaluator_AnnotationPathMismatch(t *testing.T) {
	t.Parallel()

	rule := model.AuditRule{
		ID:          "AR-001",
		Kind:        model.AuditKindAnnotationPathMismatch,
		Severity:    model.AuditSeverityError,
		Description: "Entity class outside domain/",
		Suggestion:  "Move {name} to a domain/ directory",
		Params: map[string]string{
			"annotation":    "Entity",
			"path_required": "/domain/",
		},
	}

	cases := []struct {
		name           string
		filePath       string
		annotations    []string
		wantViolations int
	}{
		{
			name:           "violation-when-entity-outside-domain",
			filePath:       "src/main/java/com/app/infrastructure/UserEntity.java",
			annotations:    []string{"Entity"},
			wantViolations: 1,
		},
		{
			name:           "no-violation-when-entity-inside-domain",
			filePath:       "src/main/java/com/app/domain/UserEntity.java",
			annotations:    []string{"Entity"},
			wantViolations: 0,
		},
		{
			name:           "no-violation-when-no-matching-annotation",
			filePath:       "src/main/java/com/app/infrastructure/SomeService.java",
			annotations:    []string{"Service"},
			wantViolations: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			summary := model.JavaFileSummary{
				Path: tc.filePath,
				Declarations: []model.JavaDeclaration{
					{
						NodeType:    "class_declaration",
						Name:        "UserEntity",
						Annotations: tc.annotations,
					},
				},
			}

			got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

			require.Len(t, got, tc.wantViolations)
			if tc.wantViolations > 0 {
				v := got[0]
				require.Equal(t, rule.ID, v.RuleID)
				require.Equal(t, rule.Kind, v.Kind)
				require.Equal(t, rule.Severity, v.Severity)
				require.Equal(t, testModuleID, v.ModuleID)
				require.Equal(t, tc.filePath, v.FilePath)
				require.Equal(t, 0, v.Line) // Line is always 0 in current model
			}
		})
	}
}

// ---------------------------------------------------------------------------
// implements_path_mismatch
// ---------------------------------------------------------------------------

func TestAuditEvaluator_ImplementsPathMismatch(t *testing.T) {
	t.Parallel()

	rule := model.AuditRule{
		ID:          "AR-002",
		Kind:        model.AuditKindImplementsPathMismatch,
		Severity:    model.AuditSeverityError,
		Description: "UseCase implementer outside allowed path",
		Suggestion:  "Move {name} to application/ or service/ directory",
		Params: map[string]string{
			"implements_glob":   "*UseCase",
			"path_required_any": "/application/,/service/",
		},
	}

	cases := []struct {
		name           string
		filePath       string
		implements     []string
		wantViolations int
	}{
		{
			name:           "violation-when-usecase-impl-outside-allowed-paths",
			filePath:       "src/main/java/com/app/adapter/in/web/CreateUserController.java",
			implements:     []string{"CreateUserUseCase"},
			wantViolations: 1,
		},
		{
			name:           "no-violation-when-impl-inside-application",
			filePath:       "src/main/java/com/app/application/usecase/CreateUserService.java",
			implements:     []string{"CreateUserUseCase"},
			wantViolations: 0,
		},
		{
			name:           "no-violation-when-impl-inside-service",
			filePath:       "src/main/java/com/app/service/CreateUserService.java",
			implements:     []string{"CreateUserUseCase"},
			wantViolations: 0,
		},
		{
			name:           "no-violation-when-implements-non-matching-interface",
			filePath:       "src/main/java/com/app/adapter/in/web/SomeAdapter.java",
			implements:     []string{"SomeInterface"},
			wantViolations: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			summary := model.JavaFileSummary{
				Path: tc.filePath,
				Declarations: []model.JavaDeclaration{
					{
						NodeType:   "class_declaration",
						Name:       "SomeClass",
						Implements: tc.implements,
					},
				},
			}

			got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

			require.Len(t, got, tc.wantViolations)
			if tc.wantViolations > 0 {
				v := got[0]
				require.Equal(t, rule.ID, v.RuleID)
				require.Equal(t, testModuleID, v.ModuleID)
				require.Equal(t, 0, v.Line)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// interface_naming
// ---------------------------------------------------------------------------

func TestAuditEvaluator_InterfaceNaming(t *testing.T) {
	t.Parallel()

	rule := model.AuditRule{
		ID:          "AR-003",
		Kind:        model.AuditKindInterfaceNaming,
		Severity:    model.AuditSeverityWarning,
		Description: "Interface in port/in/ must end with UseCase",
		Suggestion:  "Rename {name} to end with UseCase",
		Params: map[string]string{
			"path_required": "/port/in/",
			"name_suffix":   "UseCase",
		},
	}

	cases := []struct {
		name           string
		filePath       string
		declName       string
		nodeType       string
		wantViolations int
		wantSeverity   model.AuditSeverity
	}{
		{
			name:           "warning-when-interface-name-does-not-end-with-use-case",
			filePath:       "src/main/java/com/app/port/in/CreateUser.java",
			declName:       "CreateUser",
			nodeType:       "interface_declaration",
			wantViolations: 1,
			wantSeverity:   model.AuditSeverityWarning,
		},
		{
			name:           "no-violation-when-interface-name-ends-with-use-case",
			filePath:       "src/main/java/com/app/port/in/CreateUserUseCase.java",
			declName:       "CreateUserUseCase",
			nodeType:       "interface_declaration",
			wantViolations: 0,
		},
		{
			name:           "no-violation-when-file-not-in-port-in",
			filePath:       "src/main/java/com/app/domain/CreateUser.java",
			declName:       "CreateUser",
			nodeType:       "interface_declaration",
			wantViolations: 0,
		},
		{
			name:           "no-violation-when-declaration-is-not-interface",
			filePath:       "src/main/java/com/app/port/in/CreateUser.java",
			declName:       "CreateUser",
			nodeType:       "class_declaration",
			wantViolations: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			summary := model.JavaFileSummary{
				Path: tc.filePath,
				Declarations: []model.JavaDeclaration{
					{
						NodeType: tc.nodeType,
						Name:     tc.declName,
					},
				},
			}

			got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

			require.Len(t, got, tc.wantViolations)
			if tc.wantViolations > 0 {
				v := got[0]
				require.Equal(t, rule.ID, v.RuleID)
				require.Equal(t, tc.wantSeverity, v.Severity)
				require.Equal(t, testModuleID, v.ModuleID)
				require.Equal(t, 0, v.Line)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// forbidden_import
// ---------------------------------------------------------------------------

func TestAuditEvaluator_ForbiddenImport(t *testing.T) {
	t.Parallel()

	rule := model.AuditRule{
		ID:          "AR-004",
		Kind:        model.AuditKindForbiddenImport,
		Severity:    model.AuditSeverityError,
		Description: "Domain file must not import org.springframework.*",
		Suggestion:  "Remove forbidden import {import} from {file}",
		Params: map[string]string{
			"path_scope":    "/domain/",
			"import_prefix": "org.springframework.",
		},
	}

	cases := []struct {
		name           string
		filePath       string
		imports        []string
		wantViolations int
	}{
		{
			name:           "violation-when-domain-file-imports-springframework",
			filePath:       "src/main/java/com/app/domain/UserService.java",
			imports:        []string{"org.springframework.stereotype.Service", "java.util.List"},
			wantViolations: 1,
		},
		{
			name:           "no-violation-when-path-scope-does-not-match",
			filePath:       "src/main/java/com/app/adapter/in/web/UserController.java",
			imports:        []string{"org.springframework.web.bind.annotation.RestController"},
			wantViolations: 0,
		},
		{
			name:           "no-violation-when-no-forbidden-import-present",
			filePath:       "src/main/java/com/app/domain/UserEntity.java",
			imports:        []string{"java.util.Objects", "java.util.List"},
			wantViolations: 0,
		},
		{
			name:           "multiple-violations-for-multiple-forbidden-imports",
			filePath:       "src/main/java/com/app/domain/UserService.java",
			imports:        []string{"org.springframework.stereotype.Service", "org.springframework.beans.factory.annotation.Autowired"},
			wantViolations: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			summary := model.JavaFileSummary{
				Path:    tc.filePath,
				Imports: tc.imports,
			}

			got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

			require.Len(t, got, tc.wantViolations)
			for _, v := range got {
				require.Equal(t, rule.ID, v.RuleID)
				require.Equal(t, testModuleID, v.ModuleID)
				require.Equal(t, tc.filePath, v.FilePath)
				require.Equal(t, 0, v.Line)
				assertViolationType(t, v, rule.Kind, rule.Severity)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// field_type_layer_violation
// ---------------------------------------------------------------------------

func TestAuditEvaluator_FieldTypeLayerViolation(t *testing.T) {
	t.Parallel()

	rule := model.AuditRule{
		ID:          "AR-005",
		Kind:        model.AuditKindFieldTypeLayerViolation,
		Severity:    model.AuditSeverityError,
		Description: "Service class must not inject JPA adapters directly",
		Suggestion:  "Replace field {field_name} of type {field_type} with a port interface",
		Params: map[string]string{
			"path_scope":            "/service/",
			"forbidden_type_suffix": "Jpa",
		},
	}

	cases := []struct {
		name           string
		filePath       string
		fields         []model.JavaField
		wantViolations int
	}{
		{
			name:     "violation-when-service-field-type-ends-with-jpa",
			filePath: "src/main/java/com/app/service/UserService.java",
			fields: []model.JavaField{
				{Name: "repository", Type: "UserRepositoryJpa"},
			},
			wantViolations: 1,
		},
		{
			name:     "no-violation-when-field-type-is-a-port-interface",
			filePath: "src/main/java/com/app/service/UserService.java",
			fields: []model.JavaField{
				{Name: "repository", Type: "LoadUserPort"},
			},
			wantViolations: 0,
		},
		{
			name:     "no-violation-when-file-not-in-service-path",
			filePath: "src/main/java/com/app/adapter/out/persistence/UserAdapter.java",
			fields: []model.JavaField{
				{Name: "repository", Type: "UserRepositoryJpa"},
			},
			wantViolations: 0,
		},
		{
			name:     "multiple-violations-for-multiple-jpa-fields",
			filePath: "src/main/java/com/app/service/OrderService.java",
			fields: []model.JavaField{
				{Name: "userRepo", Type: "UserRepositoryJpa"},
				{Name: "orderRepo", Type: "OrderRepositoryJpa"},
				{Name: "port", Type: "CreateOrderPort"},
			},
			wantViolations: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			summary := model.JavaFileSummary{
				Path: tc.filePath,
				Declarations: []model.JavaDeclaration{
					{
						NodeType: "class_declaration",
						Name:     "SomeService",
						Fields:   tc.fields,
					},
				},
			}

			got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

			require.Len(t, got, tc.wantViolations)
			for _, v := range got {
				require.Equal(t, rule.ID, v.RuleID)
				require.Equal(t, testModuleID, v.ModuleID)
				require.Equal(t, tc.filePath, v.FilePath)
				require.Equal(t, 0, v.Line)
				assertViolationType(t, v, rule.Kind, rule.Severity)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// assertViolationType verifies the violation has the expected kind and severity.
func assertViolationType(t *testing.T, v auditvo.AuditViolation, kind model.AuditRuleKind, severity model.AuditSeverity) {
	t.Helper()
	require.Equal(t, kind, v.Kind)
	require.Equal(t, severity, v.Severity)
}
