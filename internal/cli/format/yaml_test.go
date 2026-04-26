package format_test

import (
	"bytes"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/vo"
	queryvo "github.com/jitctx/jitctx/internal/domain/vo/query"
)

// twoContextOutput builds the canonical two-context fixture used by several tests.
func twoContextOutput() queryvo.QueryContextOutput {
	return queryvo.QueryContextOutput{
		Module: queryvo.ModuleSummary{
			ID: "user-management",
			Contracts: []queryvo.ContractSummary{
				{
					Name:    "CreateUserUseCase",
					Types:   []string{"input-port"},
					Methods: []string{"UserResponse execute(CreateUserCommand cmd)"},
				},
			},
		},
		Loaded: []queryvo.LoadedContext{
			{
				ID:            "ctx-java-conventions",
				Type:          vo.ArtifactType("guidelines"),
				Path:          ".jitctx/guidelines/java-conventions.md",
				Tags:          []string{"java", "naming"},
				Body:          "# Java Conventions\nUse camelCase for methods.",
				TokenEstimate: 300,
			},
			{
				ID:            "ctx-user-scenarios",
				Type:          vo.ArtifactType("scenarios"),
				Path:          ".jitctx/scenarios/user-scenarios.md",
				Tags:          []string{"user"},
				Body:          "# User Scenarios\nAs a user I want…",
				TokenEstimate: 200,
			},
		},
		TotalTokens: 500,
	}
}

func TestWriteQueryResult_YAMLHappyPath(t *testing.T) {
	t.Parallel()

	out := twoContextOutput()
	var buf bytes.Buffer
	err := format.WriteQueryResult(&buf, "yaml", out)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &doc))

	meta, ok := doc["metadata"].(map[string]any)
	require.True(t, ok, "metadata should be a map")
	require.Equal(t, "user-management", meta["module"])
	require.Equal(t, 2, meta["context_count"])
	require.Equal(t, 500, meta["token_total"])

	contexts, ok := doc["contexts"].([]any)
	require.True(t, ok, "contexts should be a sequence")
	require.Len(t, contexts, 2)

	expectedKeys := []string{"path", "type", "tags", "token_estimate", "content"}
	for i, raw := range contexts {
		item, ok := raw.(map[string]any)
		require.True(t, ok, "contexts[%d] should be a map", i)
		for _, k := range expectedKeys {
			_, exists := item[k]
			require.True(t, exists, "contexts[%d] missing key %q", i, k)
		}
	}

	// content of first item equals original Body
	firstItem := contexts[0].(map[string]any)
	require.Equal(t, out.Loaded[0].Body, firstItem["content"])
}

func TestWriteQueryResult_YAMLIncludesContracts(t *testing.T) {
	t.Parallel()

	out := twoContextOutput()
	var buf bytes.Buffer
	err := format.WriteQueryResult(&buf, "yaml", out)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &doc))

	meta := doc["metadata"].(map[string]any)
	contracts, ok := meta["contracts"].([]any)
	require.True(t, ok, "metadata.contracts should be a non-empty array")
	require.NotEmpty(t, contracts)

	first := contracts[0].(map[string]any)
	_, hasName := first["name"]
	_, hasTypes := first["types"]
	_, hasMethods := first["methods"]
	require.True(t, hasName, "contract missing 'name' key")
	require.True(t, hasTypes, "contract missing 'types' key")
	require.True(t, hasMethods, "contract missing 'methods' key")

	methods := first["methods"].([]any)
	require.NotEmpty(t, methods)
	require.Equal(t, "UserResponse execute(CreateUserCommand cmd)", methods[0])
}

func TestWriteQueryResult_YAMLOmitsContractsWhenEmpty(t *testing.T) {
	t.Parallel()

	out := queryvo.QueryContextOutput{
		Module: queryvo.ModuleSummary{
			ID:        "user-management",
			Contracts: nil,
		},
		Loaded: []queryvo.LoadedContext{
			{
				ID:            "ctx-1",
				Type:          vo.ArtifactType("guidelines"),
				Path:          "some/path.md",
				Tags:          []string{},
				Body:          "body text",
				TokenEstimate: 10,
			},
		},
		TotalTokens: 10,
	}

	var buf bytes.Buffer
	err := format.WriteQueryResult(&buf, "yaml", out)
	require.NoError(t, err)

	rawYAML := buf.String()

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(rawYAML), &doc))

	meta := doc["metadata"].(map[string]any)
	_, hasContracts := meta["contracts"]
	require.False(t, hasContracts, "metadata should not contain 'contracts' key when empty")

	// Belt-and-braces: raw text must not contain "contracts:" under metadata
	// (omitempty must suppress the key entirely)
	require.NotContains(t, rawYAML, "contracts:")
}

func TestWriteQueryResult_YAMLEmptyLoaded(t *testing.T) {
	t.Parallel()

	out := queryvo.QueryContextOutput{
		Module: queryvo.ModuleSummary{
			ID:        "user-management",
			Contracts: nil,
		},
		Loaded:      nil,
		TotalTokens: 0,
	}

	var buf bytes.Buffer
	err := format.WriteQueryResult(&buf, "yaml", out)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &doc), "output must be valid YAML even when Loaded is nil")

	meta := doc["metadata"].(map[string]any)
	require.Equal(t, 0, meta["context_count"])

	// contexts must be an empty sequence, not absent or null, and definitely
	// not a markdown no-results message
	rawYAML := buf.String()
	require.NotContains(t, rawYAML, "No contexts matched")

	ctxRaw, exists := doc["contexts"]
	require.True(t, exists, "contexts key must be present")
	if ctxRaw != nil {
		contexts, ok := ctxRaw.([]any)
		require.True(t, ok, "contexts must be a sequence when non-nil")
		require.Empty(t, contexts)
	}
}

func TestWriteQueryResult_YAMLTagsAsSequence(t *testing.T) {
	t.Parallel()

	out := queryvo.QueryContextOutput{
		Module: queryvo.ModuleSummary{ID: "m"},
		Loaded: []queryvo.LoadedContext{
			{
				ID:            "ctx-with-tags",
				Type:          vo.ArtifactType("guidelines"),
				Path:          "a.md",
				Tags:          []string{"java", "naming"},
				Body:          "content",
				TokenEstimate: 5,
			},
			{
				ID:            "ctx-nil-tags",
				Type:          vo.ArtifactType("scenarios"),
				Path:          "b.md",
				Tags:          nil, // nil must render as [] not null
				Body:          "content2",
				TokenEstimate: 5,
			},
		},
		TotalTokens: 10,
	}

	var buf bytes.Buffer
	err := format.WriteQueryResult(&buf, "yaml", out)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &doc))

	contexts := doc["contexts"].([]any)
	require.Len(t, contexts, 2)

	// First item: tags is a proper sequence with two elements
	first := contexts[0].(map[string]any)
	tags0, ok := first["tags"].([]any)
	require.True(t, ok, "tags should unmarshal as []any (YAML sequence), not a string")
	require.ElementsMatch(t, []any{"java", "naming"}, tags0)

	// Second item: nil Tags must render as empty sequence, not null
	second := contexts[1].(map[string]any)
	tagsRaw, exists := second["tags"]
	require.True(t, exists, "tags key must be present even for nil input")
	if tagsRaw != nil {
		tags1, ok := tagsRaw.([]any)
		require.True(t, ok, "nil Tags should render as a YAML sequence")
		require.Empty(t, tags1)
	}
}

func TestWriteQueryResult_YAMLNoMarkdownComment(t *testing.T) {
	t.Parallel()

	out := twoContextOutput()
	var buf bytes.Buffer
	err := format.WriteQueryResult(&buf, "yaml", out)
	require.NoError(t, err)

	rawYAML := buf.String()
	require.False(t, strings.HasPrefix(rawYAML, "<!--"),
		"YAML output must not start with an HTML comment (markdown header leak)")
}

// TestWriteQueryYAML_EmitsTypesSequence asserts that the YAML output emits
// `types:` as a YAML sequence, not a scalar `type:` field (EP04US-003).
func TestWriteQueryYAML_EmitsTypesSequence(t *testing.T) {
	t.Parallel()

	out := queryvo.QueryContextOutput{
		Module: queryvo.ModuleSummary{
			ID: "payments",
			Contracts: []queryvo.ContractSummary{
				{
					Name:    "PaymentGateway",
					Types:   []string{"service"},
					Methods: []string{"void charge(Amount a)"},
				},
			},
		},
		Loaded:      nil,
		TotalTokens: 0,
	}

	var buf bytes.Buffer
	err := format.WriteQueryResult(&buf, "yaml", out)
	require.NoError(t, err)

	rawYAML := buf.String()

	// Must use `types:` (plural) not `type:` (singular).
	require.Contains(t, rawYAML, "types:")
	require.NotContains(t, rawYAML, "\n      type: ")

	// The value must appear as a sequence item.
	require.Contains(t, rawYAML, "- service")
}
