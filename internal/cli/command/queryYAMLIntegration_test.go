package command_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	appqueryuc "github.com/jitctx/jitctx/internal/application/usecase/queryuc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/infrastructure/fscontext"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
	"github.com/jitctx/jitctx/internal/infrastructure/token"
)

// buildYAMLTestFixture creates the directory layout and project-state.yaml for
// the YAML output integration tests. It returns the temp directory path.
//
// The fixture contains:
//   - java-conventions.md — matches via applies_to: [java] (stack language java)
//   - user-scenarios.md   — matches via module: user-management
//   - billing-scenarios.md — does NOT match user-management queries
func buildYAMLTestFixture(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	guidelinesDir := filepath.Join(tmpDir, ".jitctx", "guidelines")
	scenariosDir := filepath.Join(tmpDir, ".jitctx", "scenarios")
	require.NoError(t, os.MkdirAll(guidelinesDir, 0o755))
	require.NoError(t, os.MkdirAll(scenariosDir, 0o755))

	// java-conventions.md: matches via applies_to: [java] ∩ stack.languages = [java]
	require.NoError(t, os.WriteFile(
		filepath.Join(guidelinesDir, "java-conventions.md"),
		[]byte("---\ntags: [java, naming, hexagonal]\n---\n# Java Conventions\n\nClasses use PascalCase."),
		0o644,
	))

	// user-scenarios.md: matches via module: user-management
	require.NoError(t, os.WriteFile(
		filepath.Join(scenariosDir, "user-scenarios.md"),
		[]byte("---\nmodule: user-management\ntags: [user]\n---\n# User Scenarios\n\nAs a user I want to create an account."),
		0o644,
	))

	// billing-scenarios.md: must NOT appear in user-management query results
	require.NoError(t, os.WriteFile(
		filepath.Join(scenariosDir, "billing-scenarios.md"),
		[]byte("---\nmodule: billing\ntags: [billing]\n---\n# Billing Scenarios\n\nAs an accountant I want to generate invoices."),
		0o644,
	))

	writeTestManifest(t, tmpDir, syntheticManifest{
		GeneratedAt: time.Now(),
		Stack: syntheticStack{
			Languages:  []string{"java"},
			Frameworks: []string{"spring-boot-hexagonal"},
		},
		Modules: []syntheticModule{
			{
				ID:   "user-management",
				Path: "src/main/java/com/app/user_management",
				Tags: []string{},
				Contracts: []syntheticContract{
					{
						Name:  "CreateUserUseCase",
						Types: []string{"input-port"},
						Path:  "src/main/java/com/app/user_management/port/in/CreateUserUseCase.java",
						Methods: []syntheticMethod{
							{Signature: "User execute(String name, String email)"},
						},
					},
				},
				Dependencies: []string{},
			},
			{
				ID:           "billing",
				Path:         "src/main/java/com/app/billing",
				Tags:         []string{},
				Contracts:    []syntheticContract{},
				Dependencies: []string{},
			},
		},
		Contexts: []syntheticContext{
			{
				ID:            "java-conventions",
				Type:          "guidelines",
				AppliesTo:     []string{"java"},
				Tags:          []string{"java", "naming", "hexagonal"},
				Path:          ".jitctx/guidelines/java-conventions.md",
				TokenEstimate: 15,
			},
			{
				ID:            "user-scenarios",
				Type:          "scenarios",
				Module:        "user-management",
				Tags:          []string{"user"},
				Path:          ".jitctx/scenarios/user-scenarios.md",
				TokenEstimate: 12,
			},
			{
				ID:            "billing-scenarios",
				Type:          "scenarios",
				Module:        "billing",
				Tags:          []string{"billing"},
				Path:          ".jitctx/scenarios/billing-scenarios.md",
				TokenEstimate: 12,
			},
		},
	})

	return tmpDir
}

// expectedContextKeys are the five keys every context item in the YAML output
// must carry (feature lines 159-160).
var expectedContextKeys = []string{"path", "type", "tags", "token_estimate", "content"}

// TestQueryCmd_Integration_YAMLOutput covers feature lines 153-160.
// It runs "query --dir <tmp> --module user-management --format yaml" and
// asserts that the output is valid YAML with the expected metadata and
// contexts structure.
func TestQueryCmd_Integration_YAMLOutput(t *testing.T) {
	t.Parallel()

	tmpDir := buildYAMLTestFixture(t)

	manifestPath := filepath.Join(tmpDir, "project-state.yaml")
	store := fsmanifest.New(manifestPath)
	reader := fscontext.New()
	estimator := token.NewHeuristicEstimator()
	uc := appqueryuc.New(store, reader, estimator, discardLogger())

	var stdout, stderr bytes.Buffer
	cmd := command.NewQueryCmd(uc, discardLogger())
	cmd.SilenceUsage = true
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", tmpDir, "--module", "user-management", "--format", "yaml"})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err, "query --format yaml must succeed with exit code 0")

	// --- (1) stdout is valid YAML ---
	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(stdout.Bytes(), &doc),
		"stdout must be valid YAML")

	// --- (2) metadata.module == "user-management" ---
	meta, ok := doc["metadata"].(map[string]any)
	require.True(t, ok, "metadata must be a YAML mapping")
	require.Equal(t, "user-management", meta["module"],
		"metadata.module must equal 'user-management'")

	// --- (3) metadata.context_count == 2 ---
	ctxCount, ok := meta["context_count"].(int)
	require.True(t, ok, "metadata.context_count must be an integer")
	require.Equal(t, 2, ctxCount,
		"metadata.context_count must be 2 (java-conventions + user-scenarios)")

	// --- (4) contexts is a 2-element array ---
	contexts, ok := doc["contexts"].([]any)
	require.True(t, ok, "contexts must be a YAML sequence")
	require.Len(t, contexts, 2,
		"contexts must have exactly 2 elements")

	// --- (5) each context item has the five required keys ---
	for i, item := range contexts {
		ctxMap, ok := item.(map[string]any)
		require.True(t, ok, "contexts[%d] must be a YAML mapping", i)
		for _, key := range expectedContextKeys {
			_, present := ctxMap[key]
			require.True(t, present,
				"contexts[%d] must have key %q", i, key)
		}
	}

	// --- (6) billing body must not appear in any content field ---
	billingBody := "As an accountant I want to generate invoices"
	for i, item := range contexts {
		ctxMap := item.(map[string]any)
		content, _ := ctxMap["content"].(string)
		require.NotContains(t, content, billingBody,
			"contexts[%d].content must not contain billing body", i)
	}

	// --- (7) stderr is empty ---
	require.Empty(t, stderr.String(), "stderr must be empty — no warnings leaked")
}

// TestQueryCmd_Integration_DefaultFormatIsMarkdown covers feature lines 162-166.
// It runs "query --dir <tmp> --module user-management" (no --format flag) and
// asserts that the output is Markdown, not YAML.
func TestQueryCmd_Integration_DefaultFormatIsMarkdown(t *testing.T) {
	t.Parallel()

	tmpDir := buildYAMLTestFixture(t)

	manifestPath := filepath.Join(tmpDir, "project-state.yaml")
	store := fsmanifest.New(manifestPath)
	reader := fscontext.New()
	estimator := token.NewHeuristicEstimator()
	uc := appqueryuc.New(store, reader, estimator, discardLogger())

	var stdout, stderr bytes.Buffer
	cmd := command.NewQueryCmd(uc, discardLogger())
	cmd.SilenceUsage = true
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", tmpDir, "--module", "user-management"})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err, "query without --format must succeed")

	output := stdout.String()

	// --- (1) stdout starts with the HTML comment header ---
	firstNonEmpty := ""
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) != "" {
			firstNonEmpty = line
			break
		}
	}
	require.True(t, strings.HasPrefix(firstNonEmpty, "<!--"),
		"first non-empty line must start with '<!--'; got: %q", firstNonEmpty)

	// --- (2) stdout contains the "---" context separator ---
	require.Contains(t, output, "---",
		"stdout must contain the '---' context separator")

	// --- (3) output is NOT valid YAML at the document root ---
	// The markdown header comment makes the document structurally non-YAML.
	var probe map[string]any
	unmarshalErr := yaml.Unmarshal(stdout.Bytes(), &probe)
	// yaml.v3 may succeed for some edge cases but produce a string, not a map.
	// Provide belt-and-braces: either an error OR a nil/non-map top level.
	if unmarshalErr == nil {
		require.Nil(t, probe,
			"if yaml.Unmarshal does not error on markdown output, the root value must be nil (not a map)")
	}
	// Regardless of yaml parse result, the definitive check is the HTML prefix.
	require.True(t, strings.HasPrefix(firstNonEmpty, "<!--"),
		"output is markdown (HTML comment header), not YAML")

	// stderr must be clean.
	require.Empty(t, stderr.String(), "stderr must be empty")
}
