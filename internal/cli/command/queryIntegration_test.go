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

// syntheticManifest is a minimal YAML-serialisable representation of
// project-state.yaml used only by query integration tests. It mirrors
// fsmanifest DTOs so yaml.Marshal produces valid input for fsmanifest.Store.
type syntheticManifest struct {
	GeneratedAt time.Time          `yaml:"generated_at"`
	Stack       syntheticStack     `yaml:"stack"`
	Modules     []syntheticModule  `yaml:"modules"`
	Contexts    []syntheticContext `yaml:"contexts"`
}

type syntheticStack struct {
	Languages  []string `yaml:"languages"`
	Frameworks []string `yaml:"frameworks"`
}

type syntheticModule struct {
	ID           string              `yaml:"id"`
	Path         string              `yaml:"path"`
	Tags         []string            `yaml:"tags"`
	Contracts    []syntheticContract `yaml:"contracts"`
	Dependencies []string            `yaml:"dependencies"`
}

type syntheticContract struct {
	Name    string            `yaml:"name"`
	Type    string            `yaml:"type"`
	Path    string            `yaml:"path"`
	Methods []syntheticMethod `yaml:"methods"`
}

type syntheticMethod struct {
	Signature string `yaml:"signature"`
}

type syntheticContext struct {
	ID            string   `yaml:"id"`
	Type          string   `yaml:"type"`
	AppliesTo     []string `yaml:"applies_to,omitempty"`
	Module        string   `yaml:"module,omitempty"`
	Tags          []string `yaml:"tags"`
	Path          string   `yaml:"path"`
	TokenEstimate int      `yaml:"token_estimate"`
}

// writeTestManifest serialises m into <dir>/project-state.yaml.
func writeTestManifest(t *testing.T, dir string, m syntheticManifest) {
	t.Helper()
	data, err := yaml.Marshal(m)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project-state.yaml"), data, 0o644))
}

func TestQueryCmd_Integration_ModuleHappyPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create context body files under .jitctx/guidelines/ and .jitctx/scenarios/.
	guidelinesDir := filepath.Join(tmpDir, ".jitctx", "guidelines")
	scenariosDir := filepath.Join(tmpDir, ".jitctx", "scenarios")
	require.NoError(t, os.MkdirAll(guidelinesDir, 0o755))
	require.NoError(t, os.MkdirAll(scenariosDir, 0o755))

	// java-conventions.md matches via applies_to: [java] overlapping with stack language java.
	javaConventionsBody := "# Java Conventions\n\nClasses use PascalCase. Methods use camelCase."
	require.NoError(t, os.WriteFile(
		filepath.Join(guidelinesDir, "java-conventions.md"),
		[]byte("---\ntags: [java, naming, hexagonal]\n---\n"+javaConventionsBody),
		0o644,
	))

	// user-scenarios.md matches via module: user-management.
	userScenariosBody := "# User Scenarios\n\nAs a user I want to create an account."
	require.NoError(t, os.WriteFile(
		filepath.Join(scenariosDir, "user-scenarios.md"),
		[]byte("---\nmodule: user-management\ntags: [user]\n---\n"+userScenariosBody),
		0o644,
	))

	// billing-scenarios.md must NOT appear in user-management query results.
	billingBody := "# Billing Scenarios\n\nAs an accountant I want to generate invoices."
	require.NoError(t, os.WriteFile(
		filepath.Join(scenariosDir, "billing-scenarios.md"),
		[]byte("---\nmodule: billing\ntags: [billing]\n---\n"+billingBody),
		0o644,
	))

	// Write a synthetic manifest that references all three contexts.
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
						Name: "CreateUserUseCase",
						Type: "input-port",
						Path: "src/main/java/com/app/user_management/port/in/CreateUserUseCase.java",
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

	// Wire real adapters and run the command.
	manifestPath := filepath.Join(tmpDir, "project-state.yaml")
	store := fsmanifest.New(manifestPath)
	reader := fscontext.New()
	estimator := token.NewHeuristicEstimator()
	uc := appqueryuc.New(store, reader, estimator, discardLogger())

	var stdout, stderr bytes.Buffer
	cmd := command.NewQueryCmd(uc, discardLogger())
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", tmpDir, "--module", "user-management"})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)

	output := stdout.String()

	// First line must match the header pattern with exactly 2 contexts loaded.
	firstLine := strings.SplitN(output, "\n", 2)[0]
	require.Regexp(t, `^<!-- jitctx: 2 contexts loaded`, firstLine,
		"first line of stdout must match the 2-contexts header")

	// Both matched context bodies must be present.
	require.Contains(t, output, javaConventionsBody,
		"stdout must contain java-conventions body")
	require.Contains(t, output, userScenariosBody,
		"stdout must contain user-scenarios body")

	// Billing context must be excluded.
	require.NotContains(t, output, billingBody,
		"stdout must NOT contain billing-scenarios body")

	// Contracts section must be present with the module id and method signature.
	require.Contains(t, output, "## Contracts — user-management",
		"stdout must contain the contracts section header")
	require.Contains(t, output, "User execute(String name, String email)",
		"stdout must contain the method signature from CreateUserUseCase")
}

func TestQueryCmd_Integration_ModuleNotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	writeTestManifest(t, tmpDir, syntheticManifest{
		GeneratedAt: time.Now(),
		Stack:       syntheticStack{Languages: []string{"java"}, Frameworks: []string{"spring-boot-hexagonal"}},
		Modules: []syntheticModule{
			{
				ID:           "user-management",
				Path:         "src/main/java/com/app/user_management",
				Tags:         []string{},
				Contracts:    []syntheticContract{},
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
		Contexts: []syntheticContext{},
	})

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
	cmd.SetArgs([]string{"--dir", tmpDir, "--module", "payments"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), `module "payments" not found`,
		"error must contain the queried module name")
	require.Contains(t, err.Error(), "available modules: billing, user-management",
		"error must list available modules in alphabetical order")
	require.Empty(t, stdout.String(), "stdout must be empty on error")
}

func TestQueryCmd_Integration_ManifestMissing(t *testing.T) {
	t.Parallel()

	// Fresh temp dir — no manifest written.
	tmpDir := t.TempDir()

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
	cmd.SetArgs([]string{"--dir", tmpDir, "--module", "anything"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "project-state.yaml not found",
		"error must mention the missing manifest file")
	require.Contains(t, err.Error(), "run 'jitctx scan' first",
		"error must suggest running jitctx scan")
	require.Empty(t, stdout.String(), "stdout must be empty on error")
}
