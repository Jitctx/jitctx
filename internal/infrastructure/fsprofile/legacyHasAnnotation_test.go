package fsprofile

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
)

func TestTranslateLegacyHasAnnotation_Table(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		kind          string
		hasAnnotation string
		params        map[string]string

		wantKind       string
		wantParams     map[string]string
		wantTranslated bool
	}{
		{
			name:           "NoLegacyField_PassThrough",
			kind:           "required_annotations",
			hasAnnotation:  "",
			params:         map[string]string{"annotations": "Foo"},
			wantKind:       "required_annotations",
			wantParams:     map[string]string{"annotations": "Foo"},
			wantTranslated: false,
		},
		{
			name:           "NoLegacyField_NoKind_PassThrough",
			kind:           "",
			hasAnnotation:  "",
			params:         map[string]string{},
			wantKind:       "",
			wantParams:     map[string]string{},
			wantTranslated: false,
		},
		{
			name:          "LegacyOnly_TranslatesToRequiredAnnotations",
			kind:          "",
			hasAnnotation: "Service",
			params:        map[string]string{"path_scope": "application/usecase/"},
			wantKind:      "required_annotations",
			wantParams: map[string]string{
				"path_scope":  "application/usecase/",
				"annotations": "Service",
			},
			wantTranslated: true,
		},
		{
			name:           "LegacyOnly_NilParams_TranslatesAndAllocates",
			kind:           "",
			hasAnnotation:  "Service",
			params:         nil,
			wantKind:       "required_annotations",
			wantParams:     map[string]string{"annotations": "Service"},
			wantTranslated: true,
		},
		{
			name:           "BothKindAndHasAnnotation_KindWins",
			kind:           "forbidden_annotations",
			hasAnnotation:  "Service",
			params:         map[string]string{"annotations": "Autowired"},
			wantKind:       "forbidden_annotations",
			wantParams:     map[string]string{"annotations": "Autowired"},
			wantTranslated: false,
		},
		{
			name:           "LegacyAndExplicitAnnotationsParam_ParamsWins",
			kind:           "",
			hasAnnotation:  "Service",
			params:         map[string]string{"annotations": "Repository"},
			wantKind:       "required_annotations",
			wantParams:     map[string]string{"annotations": "Repository"},
			wantTranslated: true,
		},
		{
			name:           "LegacyAndEmptyAnnotationsParam_LegacyWins",
			kind:           "",
			hasAnnotation:  "Service",
			params:         map[string]string{"annotations": ""},
			wantKind:       "required_annotations",
			wantParams:     map[string]string{"annotations": "Service"},
			wantTranslated: true,
		},
		{
			name:          "ExplicitRequiredAnnotations_NoLegacy_PassThrough",
			kind:          "required_annotations",
			hasAnnotation: "",
			params: map[string]string{
				"path_scope":  "src/",
				"annotations": "Foo,Bar",
			},
			wantKind: "required_annotations",
			wantParams: map[string]string{
				"path_scope":  "src/",
				"annotations": "Foo,Bar",
			},
			wantTranslated: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotKind, gotParams, gotTranslated :=
				translateLegacyHasAnnotation(tc.kind, tc.hasAnnotation, tc.params)
			require.Equal(t, tc.wantKind, gotKind)
			require.Equal(t, tc.wantParams, gotParams)
			require.Equal(t, tc.wantTranslated, gotTranslated)
		})
	}
}

// TestTranslatedKindMatchesModelConstant locks the literal
// "required_annotations" emitted by translateLegacyHasAnnotation
// to model.AuditKindRequiredAnnotations. If a future maintainer
// renames the model constant, this test breaks.
func TestTranslatedKindMatchesModelConstant(t *testing.T) {
	t.Parallel()
	gotKind, _, gotTranslated := translateLegacyHasAnnotation(
		"", "Service", nil,
	)
	require.True(t, gotTranslated)
	require.Equal(t, string(model.AuditKindRequiredAnnotations), gotKind)
}

// TestTranslateLegacyHasAnnotation_ReturnsFreshMap locks the doc-comment
// guarantee that the returned effParams is never the caller's input map.
func TestTranslateLegacyHasAnnotation_ReturnsFreshMap(t *testing.T) {
	t.Parallel()
	in := map[string]string{"path_scope": "src/"}
	_, out, _ := translateLegacyHasAnnotation("", "Service", in)
	out["path_scope"] = "MUTATED"
	require.Equal(t, "src/", in["path_scope"],
		"translation must not alias the caller's params map")
}
