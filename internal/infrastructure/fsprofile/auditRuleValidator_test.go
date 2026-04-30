package fsprofile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAuditRuleParams_Table(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		in      auditRuleSchema
		wantErr bool
		wantMsg string // substring match when wantErr is true
	}{
		// M1 — required_annotations / empty annotations.
		{
			name: "M1_RequiredAnnotations_EmptyAnnotations",
			in: auditRuleSchema{
				ID: "X", Kind: "required_annotations",
				Params: map[string]string{"annotations": ""},
			},
			wantErr: true,
			wantMsg: "rule 'X': required_annotations must declare at least one annotation",
		},
		{
			name: "M1_RequiredAnnotations_AbsentAnnotations",
			in: auditRuleSchema{
				ID: "X", Kind: "required_annotations",
				Params: map[string]string{},
			},
			wantErr: true,
			wantMsg: "rule 'X': required_annotations must declare at least one annotation",
		},
		{
			name: "M1_RequiredAnnotations_WhitespaceAnnotations",
			in: auditRuleSchema{
				ID: "X", Kind: "required_annotations",
				Params: map[string]string{"annotations": "  ,  "},
			},
			wantErr: true,
			wantMsg: "rule 'X': required_annotations must declare at least one annotation",
		},
		{
			name: "M1_RequiredAnnotations_NonEmpty_Pass",
			in: auditRuleSchema{
				ID: "ok", Kind: "required_annotations",
				Params: map[string]string{"annotations": "A"},
			},
			wantErr: false,
		},
		// M2 — target enum.
		{
			name: "M2_UnknownTarget_Foo",
			in: auditRuleSchema{
				ID: "X", Kind: "forbidden_annotations",
				Params: map[string]string{
					"annotations": "Some",
					"target":      "foo",
				},
			},
			wantErr: true,
			wantMsg: "rule 'X': target must be one of [class, field, method, supertype]",
		},
		{
			name: "M2_UnknownTarget_FiresBeforeKindCheck",
			// Even with a kind-specific failure pending, target check
			// runs first by §4.1 ordering.
			in: auditRuleSchema{
				ID: "X", Kind: "required_annotations",
				Params: map[string]string{
					"annotations": "",
					"target":      "bogus",
				},
			},
			wantErr: true,
			wantMsg: "rule 'X': target must be one of [class, field, method, supertype]",
		},
		{
			name: "M2_KnownTarget_Class_Pass",
			in: auditRuleSchema{
				ID: "ok", Kind: "forbidden_annotations",
				Params: map[string]string{
					"annotations": "Some",
					"target":      "class",
				},
			},
			wantErr: false,
		},
		{
			name: "M2_KnownTarget_Field_Pass",
			in: auditRuleSchema{
				ID: "ok", Kind: "forbidden_annotations",
				Params: map[string]string{
					"annotations": "Some",
					"target":      "field",
				},
			},
			wantErr: false,
		},
		{
			name: "M2_KnownTarget_Method_Pass",
			in: auditRuleSchema{
				ID: "ok", Kind: "method_naming",
				Params: map[string]string{
					"triggered_by": "Test",
					"name_pattern": "^should.*",
					"target":       "method",
				},
			},
			wantErr: false,
		},
		{
			name: "M2_KnownTarget_Supertype_Pass",
			in: auditRuleSchema{
				ID: "ok", Kind: "required_parameterized_supertype",
				Params: map[string]string{
					"implements_glob": "*UseCase",
					"args":            "*,*",
					"target":          "supertype",
				},
			},
			wantErr: false,
		},
		{
			name: "M2_TargetAbsent_Pass",
			in: auditRuleSchema{
				ID: "ok", Kind: "required_annotations",
				Params: map[string]string{"annotations": "A"},
			},
			wantErr: false,
		},
		// M3 — forbidden_annotations / empty annotations.
		{
			name: "M3_ForbiddenAnnotations_EmptyAnnotations",
			in: auditRuleSchema{
				ID: "X", Kind: "forbidden_annotations",
				Params: map[string]string{"annotations": ""},
			},
			wantErr: true,
			wantMsg: "rule 'X': forbidden_annotations must declare at least one annotation",
		},
		// M4 — forbidden_field_type_pattern / empty patterns.
		{
			name: "M4_ForbiddenFieldTypePattern_EmptyPatterns",
			in: auditRuleSchema{
				ID: "X", Kind: "forbidden_field_type_pattern",
				Params: map[string]string{"forbidden_type_patterns": ""},
			},
			wantErr: true,
			wantMsg: "rule 'X': forbidden_field_type_pattern must declare at least one pattern",
		},
		// Other kinds — currently NO PC01US-011 constraints.
		{
			name: "OtherKind_MethodNaming_NoConstraints_Pass",
			in: auditRuleSchema{
				ID: "ok", Kind: "method_naming",
				Params: map[string]string{
					"triggered_by": "Test",
					"name_pattern": "^should.*",
				},
			},
			wantErr: false,
		},
		{
			name: "OtherKind_RequiredParameterizedSupertype_NoConstraints_Pass",
			in: auditRuleSchema{
				ID: "ok", Kind: "required_parameterized_supertype",
				Params: map[string]string{"implements_glob": "*UseCase"},
			},
			wantErr: false,
		},
		{
			name: "OtherKind_InterfaceNaming_NoConstraints_Pass",
			in: auditRuleSchema{
				ID:     "ok",
				Kind:   "interface_naming",
				Params: map[string]string{},
			},
			wantErr: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateAuditRuleParams(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
