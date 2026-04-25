package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/service"
	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

func TestJPAFieldAnnotator_Annotate(t *testing.T) {
	t.Parallel()

	annotator := service.NewJPAFieldAnnotator()

	tests := []struct {
		name      string
		rawFields []string
		want      []scaffoldvo.EntityField
	}{
		{
			name:      "LongIdGetsIdAndGeneratedValue",
			rawFields: []string{"Long id"},
			want: []scaffoldvo.EntityField{
				{
					Type: "Long",
					Name: "id",
					Annotations: []string{
						"@Id",
						"@GeneratedValue(strategy = GenerationType.IDENTITY)",
					},
				},
			},
		},
		{
			name:      "UUIDIdGetsOnlyId",
			rawFields: []string{"UUID id"},
			want: []scaffoldvo.EntityField{
				{
					Type:        "UUID",
					Name:        "id",
					Annotations: []string{"@Id"},
				},
			},
		},
		{
			name:      "NonIdFieldGetsNoAnnotations",
			rawFields: []string{"String email"},
			want: []scaffoldvo.EntityField{
				{Type: "String", Name: "email", Annotations: nil},
			},
		},
		{
			name:      "TwoFieldsOnlyIdAnnotated",
			rawFields: []string{"Long id", "String email"},
			want: []scaffoldvo.EntityField{
				{
					Type: "Long",
					Name: "id",
					Annotations: []string{
						"@Id",
						"@GeneratedValue(strategy = GenerationType.IDENTITY)",
					},
				},
				{Type: "String", Name: "email", Annotations: nil},
			},
		},
		{
			name:      "CapitalINameCaseInsensitive",
			rawFields: []string{"Long Id"},
			want: []scaffoldvo.EntityField{
				{
					Type: "Long",
					Name: "Id",
					Annotations: []string{
						"@Id",
						"@GeneratedValue(strategy = GenerationType.IDENTITY)",
					},
				},
			},
		},
		{
			name:      "LowercaseTypeCaseInsensitive",
			rawFields: []string{"long id"},
			want: []scaffoldvo.EntityField{
				{
					Type: "long",
					Name: "id",
					Annotations: []string{
						"@Id",
						"@GeneratedValue(strategy = GenerationType.IDENTITY)",
					},
				},
			},
		},
		{
			name:      "GenericTypePreserved",
			rawFields: []string{"Optional<String> name"},
			want: []scaffoldvo.EntityField{
				{Type: "Optional<String>", Name: "name", Annotations: nil},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := annotator.Annotate(tc.rawFields)
			require.Equal(t, tc.want, got)
		})
	}

	t.Run("NilInputReturnsNil", func(t *testing.T) {
		t.Parallel()

		got := annotator.Annotate(nil)
		require.Nil(t, got)
	})

	t.Run("EmptySliceReturnsNil", func(t *testing.T) {
		t.Parallel()

		got := annotator.Annotate([]string{})
		require.Nil(t, got)
	})
}

func TestJPAFieldAnnotator_HasIDField(t *testing.T) {
	t.Parallel()

	annotator := service.NewJPAFieldAnnotator()

	tests := []struct {
		name      string
		rawFields []string
		want      bool
	}{
		{
			name:      "LongIdIsIDField",
			rawFields: []string{"Long id"},
			want:      true,
		},
		{
			name:      "UUIDIdIsIDField",
			rawFields: []string{"UUID id"},
			want:      true,
		},
		{
			name:      "NonIdFieldIsNotIDField",
			rawFields: []string{"String email"},
			want:      false,
		},
		{
			name:      "NilIsNotIDField",
			rawFields: nil,
			want:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := annotator.HasIDField(tc.rawFields)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestJPAFieldAnnotator_HasGeneratedValueField(t *testing.T) {
	t.Parallel()

	annotator := service.NewJPAFieldAnnotator()

	tests := []struct {
		name      string
		rawFields []string
		want      bool
	}{
		{
			name:      "LongIdHasGeneratedValue",
			rawFields: []string{"Long id"},
			want:      true,
		},
		{
			name:      "UUIDIdHasNoGeneratedValue",
			rawFields: []string{"UUID id"},
			want:      false,
		},
		{
			name:      "NonIdFieldHasNoGeneratedValue",
			rawFields: []string{"String email"},
			want:      false,
		},
		{
			name:      "NilHasNoGeneratedValue",
			rawFields: nil,
			want:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := annotator.HasGeneratedValueField(tc.rawFields)
			require.Equal(t, tc.want, got)
		})
	}
}
