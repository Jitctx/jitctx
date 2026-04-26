package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// newInput is a concise factory for ClassificationInput used across test cases.
func newInput(kind, path string, impls, annots []string) profilevo.ClassificationInput {
	return profilevo.ClassificationInput{
		Kind:        kind,
		Path:        path,
		Implements:  impls,
		Annotations: annots,
	}
}

// typeWith builds a ProfileTypeDeclaration with a single classification rule.
func typeWith(id string, rules ...model.ClassificationRule) model.ProfileTypeDeclaration {
	return model.ProfileTypeDeclaration{
		ID:             id,
		Classification: rules,
	}
}

// rule is a convenience constructor for ClassificationRule.
func rule(kind, annotation, pathContains string, implAll, implNone []string) model.ClassificationRule {
	return model.ClassificationRule{
		Kind:           kind,
		HasAnnotation:  annotation,
		PathContains:   pathContains,
		ImplementsAll:  implAll,
		ImplementsNone: implNone,
	}
}

// --- Scenario 2: implements_all + path_contains AND match ---

// TestDeclarativeClassifier_AllAndPathMatches verifies that a type with
// kind=class, implements_all=[CreateUserUseCase], path_contains=application
// matches a class implementing CreateUserUseCase at an application path.
func TestDeclarativeClassifier_AllAndPathMatches(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("service", rule("class", "", "application", []string{"CreateUserUseCase"}, nil)),
	}
	input := newInput("class_declaration", "src/application/UserServiceImpl.java",
		[]string{"CreateUserUseCase"}, nil)

	got, err := c.ClassifyDeclarative(t.Context(), input, types)

	require.NoError(t, err)
	require.Equal(t, []string{"service"}, got)
}

// TestDeclarativeClassifier_PathMissesExcludesMatch is Scenario 6:
// class has @Service annotation but path is domain/X.java — path_contains
// fails so the AND rule does not fire.
func TestDeclarativeClassifier_PathMissesExcludesMatch(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("service",
			rule("class", "Service", "application", nil, nil),
		),
	}
	// @Service annotation present but path does not contain "application"
	input := newInput("class_declaration", "src/domain/X.java", nil, []string{"Service"})

	got, err := c.ClassifyDeclarative(t.Context(), input, types)

	require.NoError(t, err)
	require.Empty(t, got)
}

// --- Scenario 3: implements_none exclusion ---

// TestDeclarativeClassifier_ImplementsNoneExcludes verifies that a class
// implementing both Repository and Marker is NOT tagged when the rule has
// implements_none=[Marker].
func TestDeclarativeClassifier_ImplementsNoneExcludes(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("repo",
			rule("", "", "", []string{"Repository"}, []string{"Marker"}),
		),
	}
	input := newInput("class_declaration", "src/infra/Repo.java",
		[]string{"Repository", "Marker"}, nil)

	got, err := c.ClassifyDeclarative(t.Context(), input, types)

	require.NoError(t, err)
	require.Empty(t, got)
}

// TestDeclarativeClassifier_ImplementsNoneAllowsWhenAbsent verifies that the
// same class implementing only Repository (no Marker) IS tagged.
func TestDeclarativeClassifier_ImplementsNoneAllowsWhenAbsent(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("repo",
			rule("", "", "", []string{"Repository"}, []string{"Marker"}),
		),
	}
	input := newInput("class_declaration", "src/infra/Repo.java",
		[]string{"Repository"}, nil)

	got, err := c.ClassifyDeclarative(t.Context(), input, types)

	require.NoError(t, err)
	require.Equal(t, []string{"repo"}, got)
}

// --- Scenario 5: subset matching — extras tolerated ---

// TestDeclarativeClassifier_SubsetExtrasAllowed verifies that a class
// implementing [CreateUserUseCase, ChangeUserStatusUseCase, DeleteUserUseCase]
// matches a rule with implements_all=[CreateUserUseCase] (extras tolerated).
func TestDeclarativeClassifier_SubsetExtrasAllowed(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("service",
			rule("", "", "", []string{"CreateUserUseCase"}, nil),
		),
	}
	input := newInput("class_declaration", "src/application/UserService.java",
		[]string{"CreateUserUseCase", "ChangeUserStatusUseCase", "DeleteUserUseCase"}, nil)

	got, err := c.ClassifyDeclarative(t.Context(), input, types)

	require.NoError(t, err)
	require.Equal(t, []string{"service"}, got)
}

// --- Scenario 7: OR across entries ---

// TestDeclarativeClassifier_OrAcrossEntries verifies that a type with two
// classification entries is matched when only the second entry matches.
func TestDeclarativeClassifier_OrAcrossEntries(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("service",
			// First rule — requires @Service annotation + /service/ path; does NOT match
			rule("class", "Service", "/service/", nil, nil),
			// Second rule — requires implements CreateUserUseCase; DOES match
			rule("class", "", "", []string{"CreateUserUseCase"}, nil),
		),
	}
	// class with no annotation, implements CreateUserUseCase — only second rule fires
	input := newInput("class_declaration", "src/application/UserServiceImpl.java",
		[]string{"CreateUserUseCase"}, nil)

	got, err := c.ClassifyDeclarative(t.Context(), input, types)

	require.NoError(t, err)
	require.Equal(t, []string{"service"}, got)
}

// --- Annotation matching ---

// TestDeclarativeClassifier_AnnotationMatchesCaseInsensitive verifies that
// has_annotation match is case-insensitive.
func TestDeclarativeClassifier_AnnotationMatchesCaseInsensitive(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("svc", rule("class", "Service", "", nil, nil)),
	}
	// annotation written lowercase in source
	input := newInput("class_declaration", "src/svc/Foo.java", nil, []string{"service"})

	got, err := c.ClassifyDeclarative(t.Context(), input, types)

	require.NoError(t, err)
	require.Equal(t, []string{"svc"}, got)
}

// TestDeclarativeClassifier_AnnotationStripsAtPrefix verifies that a rule
// carrying "@Service" strips the "@" before matching.
func TestDeclarativeClassifier_AnnotationStripsAtPrefix(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("svc", rule("class", "@Service", "", nil, nil)),
	}
	input := newInput("class_declaration", "src/svc/Foo.java", nil, []string{"Service"})

	got, err := c.ClassifyDeclarative(t.Context(), input, types)

	require.NoError(t, err)
	require.Equal(t, []string{"svc"}, got)
}

// --- Kind mapping ---

// TestDeclarativeClassifier_KindClassMatchesRecord verifies that kind=class
// matches NodeType record_declaration (records are classes-with-data in Java).
func TestDeclarativeClassifier_KindClassMatchesRecord(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("record-type", rule("class", "", "", nil, nil)),
	}
	input := newInput("record_declaration", "src/domain/Point.java", nil, nil)

	got, err := c.ClassifyDeclarative(t.Context(), input, types)

	require.NoError(t, err)
	require.Equal(t, []string{"record-type"}, got)
}

// TestDeclarativeClassifier_KindUnknownNeverMatches verifies that an unknown
// kind value in the rule never matches any input.
func TestDeclarativeClassifier_KindUnknownNeverMatches(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("fn-type", rule("function", "", "", nil, nil)),
	}

	cases := []struct {
		name  string
		input profilevo.ClassificationInput
	}{
		{"class-declaration", newInput("class_declaration", "src/Foo.java", nil, nil)},
		{"interface-declaration", newInput("interface_declaration", "src/Bar.java", nil, nil)},
		{"record-declaration", newInput("record_declaration", "src/Rec.java", nil, nil)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := c.ClassifyDeclarative(t.Context(), tc.input, types)
			require.NoError(t, err)
			require.Empty(t, got)
		})
	}
}

// --- Empty classification ---

// TestDeclarativeClassifier_EmptyClassificationSkips verifies that a type
// with nil/empty Classification never appears in results.
func TestDeclarativeClassifier_EmptyClassificationSkips(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		{ID: "aggregate-root", Classification: nil},
		{ID: "no-rules", Classification: []model.ClassificationRule{}},
	}
	input := newInput("class_declaration", "src/domain/Root.java", nil, []string{"Entity"})

	got, err := c.ClassifyDeclarative(t.Context(), input, types)

	require.NoError(t, err)
	require.Empty(t, got)
}

// --- Declared order and deduplication ---

// TestDeclarativeClassifier_DeterministicDeclaredOrder verifies that when
// two types both match the result preserves declaration order.
func TestDeclarativeClassifier_DeterministicDeclaredOrder(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("first", rule("class", "", "", nil, nil)),
		typeWith("second", rule("class", "", "", nil, nil)),
	}
	input := newInput("class_declaration", "src/Foo.java", nil, nil)

	got, err := c.ClassifyDeclarative(t.Context(), input, types)

	require.NoError(t, err)
	require.Equal(t, []string{"first", "second"}, got)
}

// TestDeclarativeClassifier_NoDuplicateIDs verifies that when a type has two
// rules and both rules match, the type ID appears exactly once.
func TestDeclarativeClassifier_NoDuplicateIDs(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("svc",
			// Both rules match the input below — only one ID should appear.
			rule("class", "", "", nil, nil),
			rule("class", "Service", "", nil, nil),
		),
	}
	input := newInput("class_declaration", "src/Svc.java", nil, []string{"Service"})

	got, err := c.ClassifyDeclarative(t.Context(), input, types)

	require.NoError(t, err)
	require.Equal(t, []string{"svc"}, got)
}

// TestDeclarativeClassifier_DuplicateTypeEntriesDeduplicatedByID verifies
// that when the same type ID appears twice in the types slice (defensive),
// the ID is returned only once and in first-seen order.
func TestDeclarativeClassifier_DuplicateTypeEntriesDeduplicatedByID(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("svc", rule("class", "", "", nil, nil)),
		typeWith("other", rule("class", "", "", nil, nil)),
		typeWith("svc", rule("class", "", "", nil, nil)), // duplicate ID — should be skipped
	}
	input := newInput("class_declaration", "src/Foo.java", nil, nil)

	got, err := c.ClassifyDeclarative(t.Context(), input, types)

	require.NoError(t, err)
	require.Equal(t, []string{"svc", "other"}, got)
}

// --- Empty implements_all ---

// TestDeclarativeClassifier_EmptyImplementsAllNoConstraint verifies that an
// empty implements_all imposes no interface constraint — the rule matches
// solely on the other active legs.
func TestDeclarativeClassifier_EmptyImplementsAllNoConstraint(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("adapter",
			rule("class", "RestController", "", []string{}, nil),
		),
	}
	// class with @RestController but no interface — empty implements_all should still match
	input := newInput("class_declaration", "src/web/Controller.java", nil, []string{"RestController"})

	got, err := c.ClassifyDeclarative(t.Context(), input, types)

	require.NoError(t, err)
	require.Equal(t, []string{"adapter"}, got)
}

// --- kind mismatch ---

// TestDeclarativeClassifier_KindMismatchNotTagged verifies that when the
// input is an interface but the rule says kind=class the rule does not fire.
func TestDeclarativeClassifier_KindMismatchNotTagged(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("entity", rule("class", "", "", nil, nil)),
	}
	// input is an interface, rule expects a class
	input := newInput("interface_declaration", "src/domain/IRepo.java", nil, nil)

	got, err := c.ClassifyDeclarative(t.Context(), input, types)

	require.NoError(t, err)
	require.Empty(t, got)
}

// --- nil / empty types slice ---

// TestDeclarativeClassifier_NoTypesEmptyResult verifies that a nil types
// slice returns an empty result without error.
func TestDeclarativeClassifier_NoTypesEmptyResult(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	input := newInput("class_declaration", "src/Foo.java", nil, nil)

	got, err := c.ClassifyDeclarative(t.Context(), input, nil)

	require.NoError(t, err)
	require.Empty(t, got)
}

// --- Context cancellation ---

// TestDeclarativeClassifier_ContextCancelled verifies that a pre-cancelled
// context causes ClassifyDeclarative to return ctx.Err() immediately without
// iterating types.
func TestDeclarativeClassifier_ContextCancelled(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("svc", rule("class", "", "", nil, nil)),
	}
	input := newInput("class_declaration", "src/Foo.java", nil, nil)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	got, err := c.ClassifyDeclarative(ctx, input, types)

	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, got)
}

// --- Glob primitive table (§2.7) ---

// TestDeclarativeClassifier_GlobImplementsAll_LiteralStillMatches verifies
// backward-compatibility: a literal pattern (no wildcard) in implements_all
// still performs exact matching.
func TestDeclarativeClassifier_GlobImplementsAll_LiteralStillMatches(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("repo", rule("", "", "", []string{"UserRepository"}, nil)),
	}

	cases := []struct {
		name    string
		impls   []string
		wantTag bool
	}{
		{
			name:    "exact match",
			impls:   []string{"UserRepository", "Other"},
			wantTag: true,
		},
		{
			name:    "no match — different name",
			impls:   []string{"AdminRepository"},
			wantTag: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := newInput("class_declaration", "src/infra/Repo.java", tc.impls, nil)
			got, err := c.ClassifyDeclarative(t.Context(), input, types)
			require.NoError(t, err)
			if tc.wantTag {
				require.Equal(t, []string{"repo"}, got)
			} else {
				require.Empty(t, got)
			}
		})
	}
}

// TestDeclarativeClassifier_GlobImplementsAll_TrailingStar pins the EP-03
// service rule shape: implements_all: ["*UseCase"] matches a class that
// implements CreateUserUseCase (suffix glob) but does NOT match a class that
// implements UseCaseHelper (the suffix "UseCase" is not at the end).
func TestDeclarativeClassifier_GlobImplementsAll_TrailingStar(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("service", rule("", "", "", []string{"*UseCase"}, nil)),
	}

	cases := []struct {
		name    string
		impls   []string
		wantTag bool
	}{
		{
			name:    "suffix glob matches",
			impls:   []string{"CreateUserUseCase"},
			wantTag: true,
		},
		{
			name:    "prefix-only does not satisfy suffix glob",
			impls:   []string{"UseCaseHelper"},
			wantTag: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := newInput("class_declaration", "src/application/Svc.java", tc.impls, nil)
			got, err := c.ClassifyDeclarative(t.Context(), input, types)
			require.NoError(t, err)
			if tc.wantTag {
				require.Equal(t, []string{"service"}, got)
			} else {
				require.Empty(t, got)
			}
		})
	}
}

// TestDeclarativeClassifier_GlobImplementsAll_LeadingStar pins the prefix-glob
// behaviour: implements_all: ["User*"] matches UserRepository but not
// AdminRepository.
func TestDeclarativeClassifier_GlobImplementsAll_LeadingStar(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("repo", rule("", "", "", []string{"Use*"}, nil)),
	}

	cases := []struct {
		name    string
		impls   []string
		wantTag bool
	}{
		{
			name:    "prefix glob matches",
			impls:   []string{"UseCase"},
			wantTag: true,
		},
		{
			name:    "non-matching prefix not tagged",
			impls:   []string{"NotUse"},
			wantTag: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := newInput("class_declaration", "src/domain/Cls.java", tc.impls, nil)
			got, err := c.ClassifyDeclarative(t.Context(), input, types)
			require.NoError(t, err)
			if tc.wantTag {
				require.Equal(t, []string{"repo"}, got)
			} else {
				require.Empty(t, got)
			}
		})
	}
}

// TestDeclarativeClassifier_GlobImplementsAll_MultiplePatterns verifies AND
// semantics across the implements_all array: each pattern must be satisfied by
// at least one entry in the class's implements list, independently.
func TestDeclarativeClassifier_GlobImplementsAll_MultiplePatterns(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("svc", rule("", "", "", []string{"*UseCase", "Use*"}, nil)),
	}

	cases := []struct {
		name    string
		impls   []string
		wantTag bool
	}{
		{
			name:    "both patterns satisfied",
			impls:   []string{"CreateUserUseCase", "UsePort"},
			wantTag: true,
		},
		{
			name:    "only first pattern satisfied",
			impls:   []string{"CreateUserUseCase"},
			wantTag: false,
		},
		{
			name:    "only second pattern satisfied",
			impls:   []string{"UsePort"},
			wantTag: false,
		},
		{
			name:    "neither pattern satisfied",
			impls:   []string{"Repository"},
			wantTag: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := newInput("class_declaration", "src/app/Svc.java", tc.impls, nil)
			got, err := c.ClassifyDeclarative(t.Context(), input, types)
			require.NoError(t, err)
			if tc.wantTag {
				require.Equal(t, []string{"svc"}, got)
			} else {
				require.Empty(t, got)
			}
		})
	}
}

// TestDeclarativeClassifier_GlobImplementsNone_StarWildcard verifies that
// glob semantics apply to implements_none: a class implementing TestMarker
// is NOT tagged when the rule has implements_none: ["*Marker"].
func TestDeclarativeClassifier_GlobImplementsNone_StarWildcard(t *testing.T) {
	t.Parallel()

	c := service.NewDeclarativeClassifier()
	types := []model.ProfileTypeDeclaration{
		typeWith("svc", rule("", "", "", nil, []string{"*Marker"})),
	}

	cases := []struct {
		name    string
		impls   []string
		wantTag bool
	}{
		{
			name:    "implements_none glob excludes matching class",
			impls:   []string{"TestMarker"},
			wantTag: false,
		},
		{
			name:    "non-matching class is still tagged",
			impls:   []string{"Repository"},
			wantTag: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := newInput("class_declaration", "src/domain/Cls.java", tc.impls, nil)
			got, err := c.ClassifyDeclarative(t.Context(), input, types)
			require.NoError(t, err)
			if tc.wantTag {
				require.Equal(t, []string{"svc"}, got)
			} else {
				require.Empty(t, got)
			}
		})
	}
}
