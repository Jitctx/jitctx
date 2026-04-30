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

// ---------------------------------------------------------------------------
// required_annotations  (PC01US-001 / PC01RF-001 / PC01RF-009)
// ---------------------------------------------------------------------------

// requiredAnnotationsRule is the canonical rule fixture for PC01US-001:
// "Domain model classes must declare @Getter, @Builder, @NoArgsConstructor,
// @AllArgsConstructor together". The Description embeds the {missing}
// placeholder so the substituted Message carries the evidence that
// PC01US-001 Scenario 2 asserts on.
func requiredAnnotationsRule() model.AuditRule {
	return model.AuditRule{
		ID:          "domain-model-lombok",
		Kind:        model.AuditKindRequiredAnnotations,
		Severity:    model.AuditSeverityError,
		Description: "Domain model {name} must declare all of [{required}]; missing={missing}",
		Suggestion:  "Add the missing annotation(s) to {name}: {missing}",
		Params: map[string]string{
			"path_scope":  "/domain/model/",
			"annotations": "Getter,Builder,NoArgsConstructor,AllArgsConstructor",
		},
	}
}

func TestAuditEvaluator_RequiredAnnotations_DomainModelWithAllAnnotationsPasses(t *testing.T) {
	t.Parallel()
	// PC01US-001 Scenario 1: a domain model declaring all four required
	// annotations produces zero violations.

	rule := requiredAnnotationsRule()
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/domain/model/Order.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "Order",
				Annotations: []string{
					"Getter", "Builder", "NoArgsConstructor", "AllArgsConstructor",
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Empty(t, got, "no violation expected when all required annotations are present")
}

func TestAuditEvaluator_RequiredAnnotations_DomainModelMissingOneFailsWithEvidence(t *testing.T) {
	t.Parallel()
	// PC01US-001 Scenario 2: a domain model missing @AllArgsConstructor
	// produces exactly one violation whose evidence contains
	// "missing=[AllArgsConstructor]".

	rule := requiredAnnotationsRule()
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/domain/model/Order.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "Order",
				Annotations: []string{
					"Getter", "Builder", "NoArgsConstructor",
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1, "exactly one violation expected when one annotation is missing")
	v := got[0]
	require.Equal(t, rule.ID, v.RuleID)
	require.Equal(t, model.AuditKindRequiredAnnotations, v.Kind)
	require.Equal(t, model.AuditSeverityError, v.Severity)
	require.Equal(t, summary.Path, v.FilePath)
	require.Contains(t, v.Message, "missing=[AllArgsConstructor]",
		"PC01US-001 Scenario 2: violation evidence must surface the missing-annotation list")
}

func TestAuditEvaluator_RequiredAnnotations_OutsideScopeIsIgnored(t *testing.T) {
	t.Parallel()
	// Defensive: a class outside path_scope is ignored even when missing
	// annotations. Ensures the rule does not produce false positives in
	// unrelated layers.

	rule := requiredAnnotationsRule()
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/application/usecase/FindOrder.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType:    "class_declaration",
				Name:        "FindOrder",
				Annotations: []string{"Service"},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Empty(t, got, "rule must not fire outside path_scope")
}

func TestAuditEvaluator_RequiredAnnotations_MissingMultipleSurfacesAllInOrder(t *testing.T) {
	t.Parallel()
	// PC01RF-009: evidence-rich violation messages. When several
	// annotations are missing, all of them must be enumerated under
	// {missing} preserving the order declared in params.annotations.

	rule := requiredAnnotationsRule()
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/domain/model/Customer.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType:    "class_declaration",
				Name:        "Customer",
				Annotations: []string{"Getter"}, // missing Builder, NoArgsConstructor, AllArgsConstructor
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1)
	require.Contains(t, got[0].Message,
		"missing=[Builder,NoArgsConstructor,AllArgsConstructor]",
		"missing list must preserve the order of params.annotations")
}

func TestAuditEvaluator_RequiredAnnotations_MalformedRuleEmitsNothing(t *testing.T) {
	t.Parallel()
	// Defensive: an empty annotations param is rejected by the loader at
	// profile-load time (PC01RF-011); the runtime evaluator must remain
	// silent rather than emit spurious violations.

	cases := []struct {
		name   string
		params map[string]string
	}{
		{"empty-annotations", map[string]string{"path_scope": "/domain/model/", "annotations": ""}},
		{"missing-path-scope", map[string]string{"path_scope": "", "annotations": "Getter"}},
	}

	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/domain/model/Order.java",
		Declarations: []model.JavaDeclaration{
			{NodeType: "class_declaration", Name: "Order", Annotations: nil},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rule := model.AuditRule{
				ID:       "broken",
				Kind:     model.AuditKindRequiredAnnotations,
				Severity: model.AuditSeverityError,
				Params:   tc.params,
			}
			got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})
			require.Empty(t, got)
		})
	}
}

func TestAuditEvaluator_RequiredAnnotations_NonClassDeclarationIgnoredByDefault(t *testing.T) {
	t.Parallel()
	// Default node_types is "class_declaration"; an interface in the same
	// scope MUST NOT trigger the rule unless node_types is widened.

	rule := requiredAnnotationsRule()
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/domain/model/OrderRepo.java",
		Declarations: []model.JavaDeclaration{
			{NodeType: "interface_declaration", Name: "OrderRepo", Annotations: nil},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Empty(t, got, "interfaces must be skipped by default node_types filter")
}

// ---------------------------------------------------------------------------
// forbidden_annotations  (PC01US-004 / PC01RF-002 / PC01RF-003 / PC01RF-008 / PC01RF-009)
// ---------------------------------------------------------------------------

// forbiddenAnnotationsFieldRule returns the canonical field-scope rule fixture
// for PC01US-004: "Production classes must not inject dependencies via
// @Autowired field injection".
func forbiddenAnnotationsFieldRule() model.AuditRule {
	return model.AuditRule{
		ID:          "no-field-injection",
		Kind:        model.AuditKindForbiddenAnnotations,
		Severity:    model.AuditSeverityError,
		Description: "Field {name} carries a forbidden annotation; found={found}",
		Suggestion:  "Replace field injection with constructor injection for {name}",
		Params: map[string]string{
			"path_scope":   "src/main/java/",
			"annotations":  "Autowired",
			"target":       "field",
			"node_types":   "class_declaration",
			"exempt_paths": "**/testsupport/**",
		},
	}
}

func TestAuditEvaluator_ForbiddenAnnotations_FieldScope_FlagsAutowired(t *testing.T) {
	t.Parallel()
	// PC01US-004 Scenario 1: a field carrying @Autowired inside a production
	// file produces exactly one violation whose Line matches the field's line
	// and whose evidence contains "found=[Autowired]".

	rule := forbiddenAnnotationsFieldRule()
	field := model.JavaField{
		Name:        "userRepository",
		Type:        "UserRepository",
		Annotations: []string{"Autowired"},
		Line:        12,
	}
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/service/UserService.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "UserService",
				Fields:   []model.JavaField{field},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1, "exactly one violation expected for the Autowired field")
	v := got[0]
	require.Equal(t, rule.ID, v.RuleID)
	require.Equal(t, model.AuditKindForbiddenAnnotations, v.Kind)
	require.Equal(t, model.AuditSeverityError, v.Severity)
	require.Equal(t, testModuleID, v.ModuleID)
	require.Equal(t, summary.Path, v.FilePath)
	require.Equal(t, field.Line, v.Line,
		"PC01US-004 Scenario 1: violation Line must equal the field's line")
	require.Contains(t, v.Message, "found=[Autowired]",
		"PC01RF-009: evidence must surface the forbidden annotations actually found")
}

func TestAuditEvaluator_ForbiddenAnnotations_FieldScope_NoFlagWhenAnnotationAbsent(t *testing.T) {
	t.Parallel()
	// A field annotated with @Inject (not @Autowired) must not trigger the rule.

	rule := forbiddenAnnotationsFieldRule()
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/service/UserService.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "UserService",
				Fields: []model.JavaField{
					{Name: "userRepository", Type: "UserRepository", Annotations: []string{"Inject"}, Line: 12},
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Empty(t, got, "rule must not fire when field carries only non-forbidden annotations")
}

func TestAuditEvaluator_ForbiddenAnnotations_FieldScope_RespectsExemptPaths(t *testing.T) {
	t.Parallel()
	// PC01RF-008: a file under **/testsupport/** is exempt from the rule
	// even when the field carries the forbidden annotation.

	rule := forbiddenAnnotationsFieldRule()
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/testsupport/Helper.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "Helper",
				Fields: []model.JavaField{
					{Name: "svc", Type: "UserService", Annotations: []string{"Autowired"}, Line: 8},
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Empty(t, got, "PC01RF-008: exempt_paths must suppress violations for testsupport files")
}

func TestAuditEvaluator_ForbiddenAnnotations_OutsidePathScopeIsIgnored(t *testing.T) {
	t.Parallel()
	// A file whose path does not contain path_scope must produce zero violations
	// even when its fields carry forbidden annotations.

	rule := forbiddenAnnotationsFieldRule()
	summary := model.JavaFileSummary{
		// path_scope is "/src/main/java/" — test sources are outside scope
		Path: "src/test/java/com/acme/service/UserServiceTest.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "UserServiceTest",
				Fields: []model.JavaField{
					{Name: "userRepository", Type: "UserRepository", Annotations: []string{"Autowired"}, Line: 15},
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Empty(t, got, "rule must not fire for files outside path_scope")
}

func TestAuditEvaluator_ForbiddenAnnotations_ClassScope_FlagsClassAnnotation(t *testing.T) {
	t.Parallel()
	// target=class: a declaration carrying @Deprecated must produce one
	// violation with Line==0 and the declaration's simple name.

	rule := model.AuditRule{
		ID:          "no-deprecated-classes",
		Kind:        model.AuditKindForbiddenAnnotations,
		Severity:    model.AuditSeverityWarning,
		Description: "Class {name} is annotated with a forbidden annotation; found={found}",
		Suggestion:  "Remove the forbidden annotation from {name}",
		Params: map[string]string{
			"path_scope":  "src/main/java/",
			"annotations": "Deprecated",
			"target":      "class",
			"node_types":  "class_declaration",
		},
	}
	decl := model.JavaDeclaration{
		NodeType:    "class_declaration",
		Name:        "LegacyAdapter",
		Annotations: []string{"Deprecated"},
	}
	summary := model.JavaFileSummary{
		Path:         "src/main/java/com/acme/adapter/LegacyAdapter.java",
		Declarations: []model.JavaDeclaration{decl},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1, "exactly one violation expected for the Deprecated class")
	v := got[0]
	require.Equal(t, rule.ID, v.RuleID)
	require.Equal(t, 0, v.Line,
		"target=class violations always carry Line==0 (class line not captured)")
	require.Contains(t, v.Message, decl.Name,
		"violation message must reference the declaration's simple name")
	require.Contains(t, v.Message, "found=[Deprecated]",
		"PC01RF-009: found evidence must list the forbidden annotation")
}

func TestAuditEvaluator_ForbiddenAnnotations_MalformedRuleEmitsNothing(t *testing.T) {
	t.Parallel()
	// Defensive: empty annotations, missing path_scope, or unknown target must
	// each produce zero violations rather than panic or spurious output.

	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/service/UserService.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType:    "class_declaration",
				Name:        "UserService",
				Annotations: []string{"Autowired"},
				Fields: []model.JavaField{
					{Name: "repo", Type: "UserRepo", Annotations: []string{"Autowired"}, Line: 5},
				},
			},
		},
	}

	cases := []struct {
		name   string
		params map[string]string
	}{
		{
			"empty-annotations",
			map[string]string{
				"path_scope":  "/src/main/java/",
				"annotations": "",
				"target":      "field",
			},
		},
		{
			"missing-path-scope",
			map[string]string{
				"path_scope":  "",
				"annotations": "Autowired",
				"target":      "field",
			},
		},
		{
			"unknown-target",
			map[string]string{
				"path_scope":  "/src/main/java/",
				"annotations": "Autowired",
				"target":      "method",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rule := model.AuditRule{
				ID:       "broken",
				Kind:     model.AuditKindForbiddenAnnotations,
				Severity: model.AuditSeverityError,
				Params:   tc.params,
			}
			got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})
			require.Empty(t, got, "malformed rule must produce no violations; case: %s", tc.name)
		})
	}
}

func TestAuditEvaluator_ForbiddenAnnotations_MultipleFieldsOneOffending(t *testing.T) {
	t.Parallel()
	// When a class has two fields but only one carries the forbidden annotation,
	// exactly one violation must be produced pointing to the offending field.

	rule := forbiddenAnnotationsFieldRule()
	offendingField := model.JavaField{
		Name:        "userRepository",
		Type:        "UserRepository",
		Annotations: []string{"Autowired"},
		Line:        20,
	}
	cleanField := model.JavaField{
		Name:        "logger",
		Type:        "Logger",
		Annotations: nil,
		Line:        18,
	}
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/service/OrderService.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "OrderService",
				Fields:   []model.JavaField{cleanField, offendingField},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1, "exactly one violation expected — only the Autowired field is offending")
	v := got[0]
	require.Equal(t, offendingField.Line, v.Line,
		"violation must point to the offending field's line, not the clean field")
	require.Contains(t, v.Message, offendingField.Name,
		"violation message must name the offending field")
}

// ---------------------------------------------------------------------------
// method_naming  (PC01US-005 / PC01RF-004 / PC01RF-009)
// ---------------------------------------------------------------------------

// methodNamingRule is the canonical rule fixture for PC01US-005:
// "Test methods must follow the shouldX_whenY naming convention".
func methodNamingRule() model.AuditRule {
	return model.AuditRule{
		ID:          "test-naming",
		Kind:        model.AuditKindMethodNaming,
		Severity:    model.AuditSeverityError,
		Description: "Test method violates naming convention; name={name}, expected_pattern={expected_pattern}",
		Suggestion:  "Rename {name} to match {expected_pattern}",
		Params: map[string]string{
			"path_scope":   "src/test/java/",
			"triggered_by": "Test",
			"name_pattern": "^should[A-Z].*_when[A-Z].*$",
			"node_types":   "class_declaration",
		},
	}
}

func TestAuditEvaluator_MethodNaming_Compliant_NoViolation(t *testing.T) {
	t.Parallel()
	// AC1: a @Test method whose name matches the pattern must produce zero violations.

	rule := methodNamingRule()
	summary := model.JavaFileSummary{
		Path: "src/test/java/com/acme/UserServiceTest.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "UserServiceTest",
				Methods: []model.JavaMethod{
					{Name: "shouldReturnUser_whenIdExists", Annotations: []string{"Test"}, Line: 10},
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Empty(t, got, "AC1: compliant @Test method must produce zero violations")
}

func TestAuditEvaluator_MethodNaming_NonCompliant_FlagsWithEvidence(t *testing.T) {
	t.Parallel()
	// AC2: a @Test method whose name does NOT match the pattern must produce
	// exactly one violation; the message must contain the literal evidence
	// "name=testFindUser, expected_pattern=^should[A-Z].*_when[A-Z].*$".

	rule := methodNamingRule()
	summary := model.JavaFileSummary{
		Path: "src/test/java/com/acme/UserServiceTest.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "UserServiceTest",
				Methods: []model.JavaMethod{
					{Name: "testFindUser", Annotations: []string{"Test"}, Line: 10},
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1, "AC2: exactly one violation expected for non-compliant @Test method")
	v := got[0]
	require.Equal(t, rule.ID, v.RuleID)
	require.Equal(t, model.AuditKindMethodNaming, v.Kind)
	require.Equal(t, model.AuditSeverityError, v.Severity)
	require.Equal(t, testModuleID, v.ModuleID)
	require.Equal(t, summary.Path, v.FilePath)
	require.Contains(t, v.Message, "name=testFindUser, expected_pattern=^should[A-Z].*_when[A-Z].*$",
		"AC2: violation evidence must contain the method name and expected pattern verbatim")
}

func TestAuditEvaluator_MethodNaming_UntriggeredMethodIgnored(t *testing.T) {
	t.Parallel()
	// A method that does NOT carry the trigger annotation (@Test) must be
	// ignored even when its name is non-compliant.

	rule := methodNamingRule()
	summary := model.JavaFileSummary{
		Path: "src/test/java/com/acme/UserServiceTest.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "UserServiceTest",
				Methods: []model.JavaMethod{
					// no "Test" annotation → must be skipped
					{Name: "testFindUser", Annotations: []string{"Override"}, Line: 5},
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Empty(t, got, "method without trigger annotation must not produce violations")
}

func TestAuditEvaluator_MethodNaming_OutsidePathScopeIgnored(t *testing.T) {
	t.Parallel()
	// A file whose path does not contain path_scope must produce zero violations
	// even when a @Test method is non-compliant.

	rule := methodNamingRule()
	summary := model.JavaFileSummary{
		// path_scope is "src/test/java/" — main sources are outside scope
		Path: "src/main/java/com/acme/Foo.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "Foo",
				Methods: []model.JavaMethod{
					{Name: "testFindUser", Annotations: []string{"Test"}, Line: 5},
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Empty(t, got, "rule must not fire for files outside path_scope")
}

func TestAuditEvaluator_MethodNaming_RespectsExemptPaths(t *testing.T) {
	t.Parallel()
	// PC01RF-008: a file under **/legacy/** is exempt from the rule via
	// exempt_paths even when the @Test method is non-compliant.

	rule := methodNamingRule()
	rule.Params["exempt_paths"] = "**/legacy/**"

	summary := model.JavaFileSummary{
		Path: "src/test/java/com/acme/legacy/OldTest.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "OldTest",
				Methods: []model.JavaMethod{
					{Name: "testFindUser", Annotations: []string{"Test"}, Line: 8},
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Empty(t, got, "PC01RF-008: exempt_paths must suppress violations for legacy files")
}

func TestAuditEvaluator_MethodNaming_LinePropagation(t *testing.T) {
	t.Parallel()
	// The violation's Line must equal the method_declaration node line.

	rule := methodNamingRule()
	summary := model.JavaFileSummary{
		Path: "src/test/java/com/acme/UserServiceTest.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "UserServiceTest",
				Methods: []model.JavaMethod{
					{Name: "testFindUser", Annotations: []string{"Test"}, Line: 17},
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1, "exactly one violation expected")
	require.Equal(t, 17, got[0].Line, "violation Line must equal the method's line (17)")
}

func TestAuditEvaluator_MethodNaming_MalformedRuleEmitsNothing(t *testing.T) {
	t.Parallel()
	// Defensive: empty triggered_by, empty name_pattern, malformed regex, or
	// missing path_scope each produce zero violations rather than panic.

	summary := model.JavaFileSummary{
		Path: "src/test/java/com/acme/UserServiceTest.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "UserServiceTest",
				Methods: []model.JavaMethod{
					{Name: "testFindUser", Annotations: []string{"Test"}, Line: 5},
				},
			},
		},
	}

	cases := []struct {
		name   string
		params map[string]string
	}{
		{
			"empty-triggered-by",
			map[string]string{
				"path_scope":   "src/test/java/",
				"triggered_by": "",
				"name_pattern": "^should[A-Z].*_when[A-Z].*$",
				"node_types":   "class_declaration",
			},
		},
		{
			"empty-name-pattern",
			map[string]string{
				"path_scope":   "src/test/java/",
				"triggered_by": "Test",
				"name_pattern": "",
				"node_types":   "class_declaration",
			},
		},
		{
			"malformed-regex",
			map[string]string{
				"path_scope":   "src/test/java/",
				"triggered_by": "Test",
				"name_pattern": "[unclosed",
				"node_types":   "class_declaration",
			},
		},
		{
			"missing-path-scope",
			map[string]string{
				"path_scope":   "",
				"triggered_by": "Test",
				"name_pattern": "^should[A-Z].*_when[A-Z].*$",
				"node_types":   "class_declaration",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rule := model.AuditRule{
				ID:       "broken",
				Kind:     model.AuditKindMethodNaming,
				Severity: model.AuditSeverityError,
				Params:   tc.params,
			}
			got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})
			require.Empty(t, got, "malformed rule must produce no violations; case: %s", tc.name)
		})
	}
}

func TestAuditEvaluator_MethodNaming_MultipleMethodsMixed(t *testing.T) {
	t.Parallel()
	// Three methods on one class:
	//   1. shouldX_whenY + @Test   → compliant, no violation
	//   2. testFoo     + @Test     → non-compliant, one violation
	//   3. testBar     (no @Test)  → no trigger annotation, no violation
	// Exactly one violation must be produced, pointing to "testFoo".

	rule := methodNamingRule()
	compliantMethod := model.JavaMethod{
		Name: "shouldReturnUser_whenIdExists", Annotations: []string{"Test"}, Line: 10,
	}
	offendingMethod := model.JavaMethod{
		Name: "testFoo", Annotations: []string{"Test"}, Line: 20,
	}
	noTriggerMethod := model.JavaMethod{
		Name: "testBar", Annotations: nil, Line: 30,
	}
	summary := model.JavaFileSummary{
		Path: "src/test/java/com/acme/UserServiceTest.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "UserServiceTest",
				Methods:  []model.JavaMethod{compliantMethod, offendingMethod, noTriggerMethod},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1, "exactly one violation expected — only testFoo is offending")
	v := got[0]
	require.Equal(t, offendingMethod.Line, v.Line,
		"violation must point to testFoo's line")
	require.Contains(t, v.Message, "testFoo",
		"violation message must name the offending method")
}

// ---------------------------------------------------------------------------
// required_annotations — PC01US-006 / PC01RF-007 / PC01RF-009 (T6-G2)
// ---------------------------------------------------------------------------

// unitTestClassContractRule returns the canonical rule fixture for PC01US-006:
// "Unit-test classes must declare @ExtendWith(MockitoExtension.class) and
// @DisplayName". The Description embeds the {evidence} placeholder so the
// substituted Message carries the evidence asserted by each AC scenario.
func unitTestClassContractRule() model.AuditRule {
	return model.AuditRule{
		ID:          "unit-test-class-contract",
		Kind:        model.AuditKindRequiredAnnotations,
		Severity:    model.AuditSeverityError,
		Description: "{name}: {evidence}",
		Suggestion:  "Apply the contract to {name}",
		Params: map[string]string{
			"path_scope":      "src/main/java/",
			"annotations":     "ExtendWith,DisplayName",
			"expected_values": "ExtendWith=MockitoExtension.class",
			"node_types":      "class_declaration",
		},
	}
}

func TestAuditEvaluator_RequiredAnnotations_UnitTestClassWithBothAnnotationsAndCorrectArgPasses(t *testing.T) {
	t.Parallel()
	// AC1: a class with both @ExtendWith(MockitoExtension.class) and
	// @DisplayName inside path_scope must produce zero violations.

	rule := unitTestClassContractRule()
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/UserServiceTest.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType:    "class_declaration",
				Name:        "UserServiceTest",
				Annotations: []string{"ExtendWith", "DisplayName"},
				AnnotationArgs: map[string]string{
					"ExtendWith":  "MockitoExtension.class",
					"DisplayName": `"User service tests"`,
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Empty(t, got, "AC1: no violations expected when all annotations are present with correct arguments")
}

func TestAuditEvaluator_RequiredAnnotations_UnitTestClassWrongExtensionArgFlagsViolation(t *testing.T) {
	t.Parallel()
	// AC2: a class with @ExtendWith(SpringExtension.class) instead of
	// MockitoExtension.class must produce exactly one violation whose message
	// contains the literal evidence substring mandated by AC2.

	rule := unitTestClassContractRule()
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/UserServiceTest.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType:    "class_declaration",
				Name:        "UserServiceTest",
				Annotations: []string{"ExtendWith", "DisplayName"},
				AnnotationArgs: map[string]string{
					"ExtendWith":  "SpringExtension.class",
					"DisplayName": `"User service tests"`,
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1, "AC2: exactly one violation expected for wrong ExtendWith argument")
	require.Contains(t, got[0].Message,
		"annotation=ExtendWith, expected_value=MockitoExtension.class, actual=SpringExtension.class",
		"AC2: violation evidence must carry the annotation mismatch verbatim")
}

func TestAuditEvaluator_RequiredAnnotations_UnitTestClassMissingDisplayNameFlagsViolation(t *testing.T) {
	t.Parallel()
	// AC3: a class that has @ExtendWith(MockitoExtension.class) but is missing
	// @DisplayName must produce exactly one violation whose message contains
	// the literal missing-set evidence mandated by AC3.

	rule := unitTestClassContractRule()
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/UserServiceTest.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType:    "class_declaration",
				Name:        "UserServiceTest",
				Annotations: []string{"ExtendWith"},
				AnnotationArgs: map[string]string{
					"ExtendWith": "MockitoExtension.class",
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1, "AC3: exactly one violation expected for missing @DisplayName")
	require.Contains(t, got[0].Message, "missing=[DisplayName]",
		"AC3: violation evidence must surface the missing-annotation list")
}

func TestAuditEvaluator_RequiredAnnotations_BothMissingAndMismatchEmitTwoOrderedViolations(t *testing.T) {
	t.Parallel()
	// PC01RNF-003: when a class both misses @DisplayName AND has a wrong
	// @ExtendWith argument, two violations must be produced in deterministic
	// order: missing-violation FIRST, then arg-mismatch in expected_values order.

	rule := unitTestClassContractRule()
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/UserServiceTest.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType:    "class_declaration",
				Name:        "UserServiceTest",
				Annotations: []string{"ExtendWith"},
				AnnotationArgs: map[string]string{
					"ExtendWith": "SpringExtension.class",
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 2, "two violations expected: one for missing annotation, one for arg mismatch")
	require.Contains(t, got[0].Message, "missing=[DisplayName]",
		"PC01RNF-003: missing-violation must be emitted first")
	require.Contains(t, got[1].Message,
		"annotation=ExtendWith, expected_value=MockitoExtension.class, actual=SpringExtension.class",
		"PC01RNF-003: arg-mismatch violation must follow the missing-violation")
}

func TestAuditEvaluator_RequiredAnnotations_ExpectedValuesParsing_DuplicateKeyLastWins(t *testing.T) {
	t.Parallel()
	// Duplicate key in expected_values: last occurrence wins. When
	// expected_values="Foo=A,Foo=B" the effective constraint is Foo=B.
	// Tested via the public path: construct a rule, evaluate, assert the
	// mismatch evidence shows expected_value=B.

	rule := model.AuditRule{
		ID:          "dup-key-test",
		Kind:        model.AuditKindRequiredAnnotations,
		Severity:    model.AuditSeverityError,
		Description: "{name}: {evidence}",
		Suggestion:  "Fix {name}",
		Params: map[string]string{
			"path_scope":      "src/main/java/",
			"annotations":     "Foo",
			"expected_values": "Foo=A,Foo=B",
			"node_types":      "class_declaration",
		},
	}
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/SomeClass.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType:       "class_declaration",
				Name:           "SomeClass",
				Annotations:    []string{"Foo"},
				AnnotationArgs: map[string]string{"Foo": "A"},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1, "exactly one mismatch violation expected")
	require.Contains(t, got[0].Message, "expected_value=B, actual=A",
		"duplicate key: last occurrence (B) must win as the effective expected value")
}

func TestAuditEvaluator_RequiredAnnotations_ExpectedValuesIgnoresMalformedPiece(t *testing.T) {
	t.Parallel()
	// A malformed piece in expected_values (no "=") is silently dropped.
	// expected_values="Foo,Bar=B": "Foo" has no "=" and is ignored; only
	// Bar=B is enforced. A class with Bar="X" must produce exactly one mismatch
	// for Bar and no extra violation for Foo.

	rule := model.AuditRule{
		ID:          "malformed-piece-test",
		Kind:        model.AuditKindRequiredAnnotations,
		Severity:    model.AuditSeverityError,
		Description: "{name}: {evidence}",
		Suggestion:  "Fix {name}",
		Params: map[string]string{
			"path_scope":      "src/main/java/",
			"annotations":     "Bar",
			"expected_values": "Foo,Bar=B",
			"node_types":      "class_declaration",
		},
	}
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/SomeClass.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType:       "class_declaration",
				Name:           "SomeClass",
				Annotations:    []string{"Bar"},
				AnnotationArgs: map[string]string{"Bar": "X"},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1, "exactly one violation expected — only Bar=B mismatch fires; malformed Foo piece is dropped")
	require.Contains(t, got[0].Message, "expected_value=B, actual=X",
		"mismatch evidence must reference Bar's expected value B and actual value X")
}

func TestAuditEvaluator_RequiredAnnotations_NoExpectedValues_BehavesLikeBefore(t *testing.T) {
	t.Parallel()
	// Backward-compat guard (PC01US-001 / PC01RF-001): a rule with no
	// expected_values param must continue to emit missing-set violations using
	// the same {evidence} / {missing} format as before PC01US-006.

	rule := requiredAnnotationsRule()
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/domain/model/Order.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType:    "class_declaration",
				Name:        "Order",
				Annotations: []string{"Getter", "Builder"},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1, "one violation expected: NoArgsConstructor and AllArgsConstructor are missing")
	require.Contains(t, got[0].Message, "missing=[NoArgsConstructor,AllArgsConstructor]",
		"backward compat: missing-set evidence must still use the [A,B] format")
}

// TestAuditEvaluator_MatchPathGlob verifies the matchPathGlob semantics
// end-to-end through evalForbiddenAnnotations / pathExempt. The table covers
// every case specified in plan §7.2.
//
// Test strategy: build a summary whose field carries the forbidden annotation
// and set exempt_paths to the pattern under test. When the pattern matches the
// file path the evaluator returns zero violations (exempted); when the pattern
// does NOT match the evaluator returns one violation. This exercises the full
// matchPathGlob / matchSegments logic without needing direct access to the
// unexported function.
// ---------------------------------------------------------------------------
// forbidden_field_type_pattern  (PC01US-007 / PC01RF-005 / PC01RF-009)
// ---------------------------------------------------------------------------

// domainNoEntityCollectionRule returns the canonical rule fixture for PC01US-007:
// "Domain model field {field_name} carries forbidden collection".
func domainNoEntityCollectionRule() model.AuditRule {
	return model.AuditRule{
		ID:          "domain-no-entity-collection",
		Kind:        model.AuditKindForbiddenFieldTypePattern,
		Severity:    model.AuditSeverityError,
		Description: "Domain model field {field_name} carries forbidden collection: type={type}, matched_pattern={matched_pattern}",
		Suggestion:  "Replace {type} with a non-entity collection or a domain VO",
		Params: map[string]string{
			"path_scope":              "src/main/java/",
			"forbidden_type_patterns": "List<*Entity>,Set<*Entity>",
			"node_types":              "class_declaration",
		},
	}
}

func TestAuditEvaluator_ForbiddenFieldTypePattern_NonEntityCollection_NoViolation(t *testing.T) {
	t.Parallel()
	// AC1: a field of type List<String> does NOT match List<*Entity> because
	// "String" does not end with "Entity". Zero violations expected.

	rule := domainNoEntityCollectionRule()
	summary := model.JavaFileSummary{
		Path:    "src/main/java/com/acme/domain/Tag.java",
		Imports: []string{"java.util.List"},
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "Tag",
				Fields: []model.JavaField{
					{Name: "tags", Type: "List<String>", Line: 7},
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Empty(t, got, "AC1: List<String> must not match List<*Entity>")
}

func TestAuditEvaluator_ForbiddenFieldTypePattern_ListOfEntity_FlagsWithFqnAndPatternEvidence(t *testing.T) {
	t.Parallel()
	// AC2: a field of type List<OrderEntity> WITH import java.util.List must
	// produce exactly one violation. The message must contain the literal
	// AC2 substring "type=java.util.List<OrderEntity>, matched_pattern=List<*Entity>".

	rule := domainNoEntityCollectionRule()
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/domain/Order.java",
		Imports: []string{
			"java.util.List",
			"com.acme.domain.OrderEntity",
		},
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "Order",
				Fields: []model.JavaField{
					{Name: "orders", Type: "List<OrderEntity>", Line: 9},
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1, "AC2: exactly one violation expected for List<OrderEntity>")
	require.Equal(t, 9, got[0].Line, "AC2: violation must be on field line 9")
	require.Contains(t, got[0].Message,
		"type=java.util.List<OrderEntity>, matched_pattern=List<*Entity>",
		"AC2: violation evidence must surface FQN type and matched pattern verbatim")
}

func TestAuditEvaluator_ForbiddenFieldTypePattern_SetOfEntity_ReportsFieldLine(t *testing.T) {
	t.Parallel()
	// AC3: a field of type Set<UserEntity> must produce exactly one violation
	// whose Line equals the field's line number (11). The message must contain
	// evidence for both the type and the matched pattern.

	rule := domainNoEntityCollectionRule()
	summary := model.JavaFileSummary{
		Path:    "src/main/java/com/acme/domain/User.java",
		Imports: []string{"java.util.Set"},
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "User",
				Fields: []model.JavaField{
					{Name: "users", Type: "Set<UserEntity>", Line: 11},
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1, "AC3: exactly one violation expected for Set<UserEntity>")
	require.Equal(t, 11, got[0].Line,
		"AC3: violation Line must equal the field's line (11)")
	require.Contains(t, got[0].Message, "type=java.util.Set<UserEntity>",
		"AC3: violation evidence must surface the resolved FQN type")
	require.Contains(t, got[0].Message, "matched_pattern=Set<*Entity>",
		"AC3: violation evidence must surface the matched pattern")
}

func TestAuditEvaluator_ForbiddenFieldTypePattern_NonParameterizedType_Skipped(t *testing.T) {
	t.Parallel()
	// Non-parameterized field types (no angle brackets) must be silently skipped.
	// A field of type "String" against pattern "List<*Entity>" must produce zero violations.

	rule := domainNoEntityCollectionRule()
	summary := model.JavaFileSummary{
		Path: "src/main/java/com/acme/domain/Foo.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "Foo",
				Fields: []model.JavaField{
					{Name: "x", Type: "String", Line: 5},
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Empty(t, got, "non-parameterized field type must not produce violations")
}

func TestAuditEvaluator_ForbiddenFieldTypePattern_OutsidePathScope_Skipped(t *testing.T) {
	t.Parallel()
	// A file whose path does not contain path_scope ("src/main/java/") must
	// produce zero violations even when its field type matches the pattern.

	rule := domainNoEntityCollectionRule()
	summary := model.JavaFileSummary{
		Path: "src/test/java/com/acme/domain/Foo.java",
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "Foo",
				Fields: []model.JavaField{
					{Name: "orders", Type: "List<OrderEntity>", Line: 3},
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Empty(t, got, "rule must not fire for files outside path_scope")
}

func TestAuditEvaluator_ForbiddenFieldTypePattern_ImportNotFound_FallsBackToSimpleName(t *testing.T) {
	t.Parallel()
	// When no import resolves the outer type, resolveFQN falls back to the
	// simple name. The violation must still fire and the message must contain
	// "type=List<OrderEntity>" (no FQN prefix), proving the fallback behaviour.

	rule := domainNoEntityCollectionRule()
	summary := model.JavaFileSummary{
		Path:    "src/main/java/com/acme/domain/Order.java",
		Imports: []string{"com.acme.domain.OrderEntity"}, // java.util.List is intentionally absent
		Declarations: []model.JavaDeclaration{
			{
				NodeType: "class_declaration",
				Name:     "Order",
				Fields: []model.JavaField{
					{Name: "orders", Type: "List<OrderEntity>", Line: 6},
				},
			},
		},
	}

	got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})

	require.Len(t, got, 1, "violation must fire even when the import for the outer type is absent")
	require.Contains(t, got[0].Message, "type=List<OrderEntity>",
		"when no import resolves, message must carry simple name (no FQN prefix)")
}

func TestAuditEvaluator_MatchPathGlob(t *testing.T) {
	t.Parallel()

	// baseRule is the field-scope forbidden_annotations rule reused across
	// all subtests; only exempt_paths changes per case.
	baseRule := func(exemptPattern string) model.AuditRule {
		return model.AuditRule{
			ID:          "no-field-injection",
			Kind:        model.AuditKindForbiddenAnnotations,
			Severity:    model.AuditSeverityError,
			Description: "found={found}",
			Suggestion:  "fix {name}",
			Params: map[string]string{
				"path_scope":   "/",
				"annotations":  "Autowired",
				"target":       "field",
				"node_types":   "class_declaration",
				"exempt_paths": exemptPattern,
			},
		}
	}

	makeSummary := func(path string) model.JavaFileSummary {
		return model.JavaFileSummary{
			Path: path,
			Declarations: []model.JavaDeclaration{
				{
					NodeType: "class_declaration",
					Name:     "SomeClass",
					Fields: []model.JavaField{
						{Name: "dep", Type: "Dep", Annotations: []string{"Autowired"}, Line: 5},
					},
				},
			},
		}
	}

	cases := []struct {
		pattern string
		path    string
		match   bool // true → exempt (zero violations); false → not exempt (one violation)
	}{
		// ** matches zero or more path segments
		{"**/testsupport/**", "src/test/java/com/acme/testsupport/Helper.java", true},
		{"**/testsupport/**", "src/main/java/com/acme/Foo.java", false},
		// ** at end anchors to the directory itself
		{"**/testsupport", "src/test/java/com/acme/testsupport", true},
		{"**/testsupport", "src/test/java/com/acme/testsupport/Helper.java", false},
		// ** before filename
		{"**/Helper.java", "src/test/java/com/acme/testsupport/Helper.java", true},
		// **/foo/** where path ends at the directory (no trailing file)
		{"**/foo/**", "src/test/java/com/acme/foo", true},
		{"**/foo/**", "src/main/java/foo.txt", false},
	}

	for _, tc := range cases {
		t.Run(tc.pattern+"|"+tc.path, func(t *testing.T) {
			t.Parallel()
			rule := baseRule(tc.pattern)
			summary := makeSummary(tc.path)
			got := newEvaluator().EvaluateFile(testModuleID, summary, []model.AuditRule{rule})
			if tc.match {
				require.Empty(t, got,
					"pattern %q should match %q (file exempted, zero violations)", tc.pattern, tc.path)
			} else {
				require.Len(t, got, 1,
					"pattern %q should NOT match %q (file not exempted, one violation)", tc.pattern, tc.path)
			}
		})
	}
}
