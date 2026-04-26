package fsprofile

import (
	"errors"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
)

// TestToBundleDomain exercises the pure toBundleDomain mapper without any I/O.

func TestToBundleDomain_HappyPath(t *testing.T) {
	t.Parallel()

	dto := bundleDTO{
		Name:     "spring-boot-hexagonal",
		Language: "java",
		Types: []bundleTypeDTO{
			{ID: "service", Template: "service.java.tmpl"},
		},
	}
	templates := map[string][]byte{
		"service.java.tmpl": []byte("// stub"),
	}

	bundle, err := toBundleDomain(dto, templates)

	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.Len(t, bundle.RawTypes, 1)
	require.Equal(t, "service", bundle.RawTypes[0].ID)
	require.Equal(t, "service.java.tmpl", bundle.RawTypes[0].Template)

	got, ok := bundle.GetTemplate("service.java.tmpl")
	require.True(t, ok)
	require.Equal(t, []byte("// stub"), got)
}

func TestToBundleDomain_MissingTemplate(t *testing.T) {
	t.Parallel()

	dto := bundleDTO{
		Name:     "spring-boot-hexagonal",
		Language: "java",
		Types: []bundleTypeDTO{
			{ID: "service", Template: "service.java.tmpl"},
		},
	}
	templates := map[string][]byte{} // empty — template not loaded

	_, err := toBundleDomain(dto, templates)

	require.Error(t, err)

	var tme *domerr.TemplateMissingError
	require.ErrorAs(t, err, &tme)
	require.True(t, errors.Is(err, domerr.ErrTemplateMissing))
	require.True(t, errors.Is(err, domerr.ErrProfileInvalid))
	require.Contains(t, err.Error(), "service.java.tmpl")
	require.Contains(t, err.Error(), "service")
}

func TestToBundleDomain_TypeWithoutTemplate(t *testing.T) {
	t.Parallel()

	dto := bundleDTO{
		Name:     "my-profile",
		Language: "java",
		Types: []bundleTypeDTO{
			{ID: "abstract-base", Template: ""}, // no template — must be allowed
		},
	}
	// Even an empty templates map should not trigger an error when Template == "".
	templates := map[string][]byte{}

	bundle, err := toBundleDomain(dto, templates)

	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.Len(t, bundle.RawTypes, 1)
	require.Equal(t, "abstract-base", bundle.RawTypes[0].ID)
	require.Equal(t, "", bundle.RawTypes[0].Template)
}

func TestToBundleDomain_EmptyTypesList(t *testing.T) {
	t.Parallel()

	dto := bundleDTO{
		Name:     "bare-profile",
		Language: "java",
		Types:    nil,
	}

	bundle, err := toBundleDomain(dto, map[string][]byte{})

	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.Empty(t, bundle.RawTypes)
}

func TestToBundleDomain_MultipleTypes(t *testing.T) {
	t.Parallel()

	dto := bundleDTO{
		Name:     "full-profile",
		Language: "java",
		Types: []bundleTypeDTO{
			{ID: "service", Template: "service.java.tmpl"},
			{ID: "repository", Template: "repository.java.tmpl"},
			{ID: "dto", Template: ""}, // no template
		},
	}
	templates := map[string][]byte{
		"service.java.tmpl":    []byte("// service stub"),
		"repository.java.tmpl": []byte("// repository stub"),
	}

	bundle, err := toBundleDomain(dto, templates)

	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.Len(t, bundle.RawTypes, 3)
	require.Equal(t, "service", bundle.RawTypes[0].ID)
	require.Equal(t, "repository", bundle.RawTypes[1].ID)
	require.Equal(t, "dto", bundle.RawTypes[2].ID)

	svc, ok := bundle.GetTemplate("service.java.tmpl")
	require.True(t, ok)
	require.Equal(t, []byte("// service stub"), svc)

	repo, ok := bundle.GetTemplate("repository.java.tmpl")
	require.True(t, ok)
	require.Equal(t, []byte("// repository stub"), repo)
}

func TestToBundleDomain_LegacyLanguagesField(t *testing.T) {
	t.Parallel()

	// When the singular "language" field is empty and "languages" is set,
	// the profile.Languages slice should reflect the legacy list.
	dto := bundleDTO{
		Name:      "legacy-profile",
		Languages: []string{"java", "kotlin"},
		Types:     nil,
	}

	bundle, err := toBundleDomain(dto, map[string][]byte{})

	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.Equal(t, []string{"java", "kotlin"}, bundle.Profile.Languages)
}

func TestToBundleDomain_SingularLanguagePromotedToSlice(t *testing.T) {
	t.Parallel()

	// When "language" is set and "languages" is empty, Languages should be
	// synthesised as a one-element slice.
	dto := bundleDTO{
		Name:     "single-lang-profile",
		Language: "java",
		Types:    nil,
	}

	bundle, err := toBundleDomain(dto, map[string][]byte{})

	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.Equal(t, []string{"java"}, bundle.Profile.Languages)
}

func TestToBundleDomain_PackagingBlockForwarded(t *testing.T) {
	t.Parallel()

	// When the DTO carries a non-nil Packaging node the mapper should
	// marshal it into RawPackaging bytes (non-empty).
	//
	// We use table-driven sub-cases: one with packaging, one without.
	cases := []struct {
		name       string
		dto        bundleDTO
		wantRawNil bool
	}{
		{
			name: "with-packaging",
			dto: bundleDTO{
				Name:      "pkg-profile",
				Language:  "java",
				Packaging: mustParseYAMLNode(t, "layout: maven\n"),
			},
			wantRawNil: false,
		},
		{
			name: "without-packaging",
			dto: bundleDTO{
				Name:     "no-pkg-profile",
				Language: "java",
			},
			wantRawNil: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bundle, err := toBundleDomain(tc.dto, map[string][]byte{})

			require.NoError(t, err)
			require.NotNil(t, bundle)
			if tc.wantRawNil {
				require.Nil(t, bundle.RawPackaging)
			} else {
				require.NotEmpty(t, bundle.RawPackaging)
			}
		})
	}
}

// mustParseYAMLNode decodes a YAML string into a *yaml.Node for use in test
// fixtures that require a non-nil Packaging field.
func mustParseYAMLNode(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	if err := yaml.NewDecoder(strings.NewReader(src)).Decode(&doc); err != nil {
		t.Fatalf("mustParseYAMLNode: %v", err)
	}
	// yaml.Decode wraps in a DocumentNode; unwrap to the mapping node.
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}
	return &doc
}

func TestToBundleDomain_MissingIDReturnsError(t *testing.T) {
	t.Parallel()

	dto := bundleDTO{
		Name:     "bad-profile",
		Language: "java",
		Types: []bundleTypeDTO{
			{ID: "", Template: "service.java.tmpl"}, // missing id
		},
	}

	_, err := toBundleDomain(dto, map[string][]byte{"service.java.tmpl": []byte("stub")})

	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrProfileInvalid))
}

// --- EP04US-002 tests (T6-G2) ---

func TestToBundleDomain_PreservesClassificationRules(t *testing.T) {
	t.Parallel()

	dto := bundleDTO{
		Name:     "classified-profile",
		Language: "java",
		Types: []bundleTypeDTO{
			{
				ID: "application-service",
				Classification: []bundleClassificationDTO{
					{
						Kind:           "class",
						ImplementsAll:  []string{"A", "B"},
						ImplementsNone: []string{"C"},
						HasAnnotation:  "Service",
						PathContains:   "application",
					},
				},
			},
		},
	}

	bundle, err := toBundleDomain(dto, map[string][]byte{})

	require.NoError(t, err)
	require.Len(t, bundle.RawTypes, 1)

	rules := bundle.RawTypes[0].Classification
	require.Len(t, rules, 1)
	require.Equal(t, "class", rules[0].Kind)
	require.Equal(t, []string{"A", "B"}, rules[0].ImplementsAll)
	require.Equal(t, []string{"C"}, rules[0].ImplementsNone)
	require.Equal(t, "Service", rules[0].HasAnnotation)
	require.Equal(t, "application", rules[0].PathContains)
}

func TestToBundleDomain_PreservesDescription(t *testing.T) {
	t.Parallel()

	const wantDesc = "Application service implementing input ports"

	dto := bundleDTO{
		Name:     "described-profile",
		Language: "java",
		Types: []bundleTypeDTO{
			{
				ID:          "application-service",
				Description: wantDesc,
			},
		},
	}

	bundle, err := toBundleDomain(dto, map[string][]byte{})

	require.NoError(t, err)
	require.Len(t, bundle.RawTypes, 1)
	require.Equal(t, wantDesc, bundle.RawTypes[0].Description)
}

func TestToBundleDomain_RawTypeBytesAreRoundTripped(t *testing.T) {
	t.Parallel()

	dto := bundleDTO{
		Name:     "raw-profile",
		Language: "java",
		Types: []bundleTypeDTO{
			{
				ID: "application-service",
				Classification: []bundleClassificationDTO{
					{
						Kind:          "class",
						ImplementsAll: []string{"InputPort"},
					},
				},
			},
		},
	}

	bundle, err := toBundleDomain(dto, map[string][]byte{})

	require.NoError(t, err)
	require.Len(t, bundle.RawTypes, 1)

	rawType := bundle.RawTypes[0].Raw
	require.Greater(t, len(rawType), 0)

	// Unmarshal the raw bytes back into a generic node and confirm both
	// "id" and "classification" keys are present.
	var node map[string]any
	require.NoError(t, yaml.Unmarshal(rawType, &node))
	_, hasID := node["id"]
	_, hasClassification := node["classification"]
	require.True(t, hasID, "raw bytes must contain 'id' key")
	require.True(t, hasClassification, "raw bytes must contain 'classification' key")
}

func TestToBundleDomain_SliceCopiesNotAliased(t *testing.T) {
	t.Parallel()

	dto := bundleDTO{
		Name:     "aliasing-profile",
		Language: "java",
		Types: []bundleTypeDTO{
			{
				ID: "application-service",
				Classification: []bundleClassificationDTO{
					{
						ImplementsAll: []string{"X", "Y"},
					},
				},
			},
		},
	}

	bundle, err := toBundleDomain(dto, map[string][]byte{})
	require.NoError(t, err)

	// Mutate the source DTO slice AFTER the mapper has run.
	dto.Types[0].Classification[0].ImplementsAll[0] = "MUTATED"

	// The produced model slice must not reflect the mutation.
	require.Equal(t, "X", bundle.RawTypes[0].Classification[0].ImplementsAll[0])
}
