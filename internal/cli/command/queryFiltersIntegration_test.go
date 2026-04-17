package command_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appqueryuc "github.com/jitctx/jitctx/internal/application/usecase/queryuc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/infrastructure/fscontext"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
	"github.com/jitctx/jitctx/internal/infrastructure/token"
)

// filterTestContext describes a synthetic context entry used by the filter
// integration tests.
type filterTestContext struct {
	id      string
	ctxType string
	module  string
	tags    []string
	body    string
}

// buildFilterFixture creates the directory layout and project-state.yaml for a
// filter integration test. It returns the temp directory path. Each context's
// body file is written under <tmpDir>/.jitctx/<type>/<id>.md.
func buildFilterFixture(t *testing.T, modules []syntheticModule, contexts []filterTestContext) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Write body files for every context.
	for _, ctx := range contexts {
		dir := filepath.Join(tmpDir, ".jitctx", ctx.ctxType)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, ctx.id+".md"),
			[]byte(ctx.body),
			0o644,
		))
	}

	// Build synthetic manifest contexts.
	synthContexts := make([]syntheticContext, 0, len(contexts))
	for _, ctx := range contexts {
		sc := syntheticContext{
			ID:            ctx.id,
			Type:          ctx.ctxType,
			Tags:          ctx.tags,
			Path:          ".jitctx/" + ctx.ctxType + "/" + ctx.id + ".md",
			TokenEstimate: 10,
		}
		if ctx.module != "" {
			sc.Module = ctx.module
		}
		synthContexts = append(synthContexts, sc)
	}

	writeTestManifest(t, tmpDir, syntheticManifest{
		GeneratedAt: time.Now(),
		Stack: syntheticStack{
			Languages:  []string{"java"},
			Frameworks: []string{"spring-boot-hexagonal"},
		},
		Modules:  modules,
		Contexts: synthContexts,
	})

	return tmpDir
}

// runQueryCmd wires real adapters and executes the query command. It returns
// the combined stdout and stderr output.
func runQueryCmd(t *testing.T, tmpDir string, args []string, logger *slog.Logger) (stdout, stderr string) {
	t.Helper()
	manifestPath := filepath.Join(tmpDir, "project-state.yaml")
	store := fsmanifest.New(manifestPath)
	reader := fscontext.New()
	estimator := token.NewHeuristicEstimator()
	uc := appqueryuc.New(store, reader, estimator, discardLogger())

	var outBuf, errBuf bytes.Buffer
	cmd := command.NewQueryCmd(uc, logger)
	cmd.SilenceUsage = true
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(append([]string{"--dir", tmpDir}, args...))

	require.NoError(t, cmd.ExecuteContext(context.Background()))
	return outBuf.String(), errBuf.String()
}

// userManagementModules is the canonical module list used across filter tests.
var userManagementModules = []syntheticModule{
	{
		ID:           "user-management",
		Path:         "src/main/java/com/app/user",
		Tags:         []string{},
		Contracts:    []syntheticContract{},
		Dependencies: []string{},
	},
}

// TestQueryCmd_Integration_FilterBySingleType corresponds to Gherkin lines 96-104:
// Given module "user-management" with three contexts of different types,
// when --type guidelines is applied, only the guidelines context appears.
func TestQueryCmd_Integration_FilterBySingleType(t *testing.T) {
	t.Parallel()

	contexts := []filterTestContext{
		{id: "java-conventions", ctxType: "guidelines", module: "user-management", tags: []string{"java"}, body: "# Java Conventions\n\nJava style guide."},
		{id: "user-scenarios", ctxType: "scenarios", module: "user-management", tags: []string{"user"}, body: "# User Scenarios\n\nUser stories."},
		{id: "user-requirements", ctxType: "requirements", module: "user-management", tags: []string{"user"}, body: "# User Requirements\n\nFunctional requirements."},
	}

	tmpDir := buildFilterFixture(t, userManagementModules, contexts)
	stdout, _ := runQueryCmd(t, tmpDir, []string{"--module", "user-management", "--type", "guidelines"}, discardLogger())

	require.Contains(t, stdout, "Java Conventions", "stdout must contain guidelines context")
	require.NotContains(t, stdout, "User Scenarios", "stdout must NOT contain scenarios context")
	require.NotContains(t, stdout, "User Requirements", "stdout must NOT contain requirements context")
}

// TestQueryCmd_Integration_FilterByMultipleTypes corresponds to Gherkin lines 106-111:
// --type guidelines,scenarios includes both but excludes requirements.
func TestQueryCmd_Integration_FilterByMultipleTypes(t *testing.T) {
	t.Parallel()

	contexts := []filterTestContext{
		{id: "java-conventions", ctxType: "guidelines", module: "user-management", tags: []string{"java"}, body: "# Java Conventions\n\nJava style guide."},
		{id: "user-scenarios", ctxType: "scenarios", module: "user-management", tags: []string{"user"}, body: "# User Scenarios\n\nUser stories."},
		{id: "user-requirements", ctxType: "requirements", module: "user-management", tags: []string{"user"}, body: "# User Requirements\n\nFunctional requirements."},
	}

	tmpDir := buildFilterFixture(t, userManagementModules, contexts)
	stdout, _ := runQueryCmd(t, tmpDir, []string{"--module", "user-management", "--type", "guidelines,scenarios"}, discardLogger())

	require.Contains(t, stdout, "Java Conventions", "stdout must contain guidelines context")
	require.Contains(t, stdout, "User Scenarios", "stdout must contain scenarios context")
	require.NotContains(t, stdout, "User Requirements", "stdout must NOT contain requirements context")
}

// TestQueryCmd_Integration_FilterByTag corresponds to Gherkin lines 113-121:
// Three contexts with distinct tags; --tags security returns only the security one.
func TestQueryCmd_Integration_FilterByTag(t *testing.T) {
	t.Parallel()

	contexts := []filterTestContext{
		{id: "java-conventions", ctxType: "guidelines", module: "user-management", tags: []string{"java", "naming"}, body: "# Java Conventions\n\nJava style guide."},
		{id: "security-guidelines", ctxType: "guidelines", module: "user-management", tags: []string{"security", "auth"}, body: "# Security Guidelines\n\nSecurity rules."},
		{id: "test-conventions", ctxType: "guidelines", module: "user-management", tags: []string{"java", "testing"}, body: "# Test Conventions\n\nTesting guide."},
	}

	tmpDir := buildFilterFixture(t, userManagementModules, contexts)
	stdout, _ := runQueryCmd(t, tmpDir, []string{"--module", "user-management", "--tags", "security"}, discardLogger())

	require.Contains(t, stdout, "Security Guidelines", "stdout must contain security-guidelines context")
	require.NotContains(t, stdout, "Java Conventions", "stdout must NOT contain java-conventions context")
	require.NotContains(t, stdout, "Test Conventions", "stdout must NOT contain test-conventions context")
}

// TestQueryCmd_Integration_FilterByMultipleTagsOR corresponds to Gherkin lines 123-131:
// --tags security,auth uses OR logic: contexts with either tag are included.
func TestQueryCmd_Integration_FilterByMultipleTagsOR(t *testing.T) {
	t.Parallel()

	contexts := []filterTestContext{
		{id: "security-guidelines", ctxType: "guidelines", module: "user-management", tags: []string{"security"}, body: "# Security Guidelines\n\nSecurity rules."},
		{id: "auth-guide", ctxType: "guidelines", module: "user-management", tags: []string{"auth"}, body: "# Auth Guide\n\nAuthentication guide."},
		{id: "naming-guide", ctxType: "guidelines", module: "user-management", tags: []string{"naming"}, body: "# Naming Guide\n\nNaming conventions."},
	}

	tmpDir := buildFilterFixture(t, userManagementModules, contexts)
	stdout, _ := runQueryCmd(t, tmpDir, []string{"--module", "user-management", "--tags", "security,auth"}, discardLogger())

	require.Contains(t, stdout, "Security Guidelines", "stdout must contain security-guidelines (has tag 'security')")
	require.Contains(t, stdout, "Auth Guide", "stdout must contain auth-guide (has tag 'auth')")
	require.NotContains(t, stdout, "Naming Guide", "stdout must NOT contain naming-guide (only has 'naming')")
}

// TestQueryCmd_Integration_CombinedFilters corresponds to Gherkin lines 133-141:
// --module + --type + --tags all apply with AND semantics; only the intersection is returned.
func TestQueryCmd_Integration_CombinedFilters(t *testing.T) {
	t.Parallel()

	billingModules := append(userManagementModules, syntheticModule{
		ID:           "billing",
		Path:         "src/main/java/com/app/billing",
		Tags:         []string{},
		Contracts:    []syntheticContract{},
		Dependencies: []string{},
	})

	contexts := []filterTestContext{
		{id: "security-guidelines", ctxType: "guidelines", module: "user-management", tags: []string{"security"}, body: "# Security Guidelines\n\nSecurity rules."},
		{id: "security-scenarios", ctxType: "scenarios", module: "user-management", tags: []string{"security"}, body: "# Security Scenarios\n\nSecurity scenarios."},
		{id: "billing-security", ctxType: "guidelines", module: "billing", tags: []string{"security"}, body: "# Billing Security\n\nBilling security rules."},
	}

	tmpDir := buildFilterFixture(t, billingModules, contexts)
	stdout, _ := runQueryCmd(t, tmpDir, []string{
		"--module", "user-management",
		"--type", "guidelines",
		"--tags", "security",
	}, discardLogger())

	require.Contains(t, stdout, "Security Guidelines",
		"stdout must contain security-guidelines (module=user-management, type=guidelines, tag=security)")
	require.NotContains(t, stdout, "Security Scenarios",
		"stdout must NOT contain security-scenarios (type=scenarios, not guidelines)")
	require.NotContains(t, stdout, "Billing Security",
		"stdout must NOT contain billing-security (module=billing, not user-management)")
}

// TestQueryCmd_Integration_NoResults corresponds to Gherkin lines 143-148:
// When no contexts match all combined filters, a helpful message is shown on stdout.
func TestQueryCmd_Integration_NoResults(t *testing.T) {
	t.Parallel()

	billingModules := []syntheticModule{
		{
			ID:           "billing",
			Path:         "src/main/java/com/app/billing",
			Tags:         []string{},
			Contracts:    []syntheticContract{},
			Dependencies: []string{},
		},
	}

	contexts := []filterTestContext{
		{id: "billing-guide", ctxType: "guidelines", module: "billing", tags: []string{"rest"}, body: "# Billing Guide\n\nBilling guidelines."},
	}

	tmpDir := buildFilterFixture(t, billingModules, contexts)
	stdout, _ := runQueryCmd(t, tmpDir, []string{
		"--module", "billing",
		"--type", "scenarios",
		"--tags", "graphql",
	}, discardLogger())

	// Header must still be emitted with 0 contexts.
	firstLine := strings.SplitN(stdout, "\n", 2)[0]
	require.Regexp(t, `^<!-- jitctx: 0 contexts loaded`, firstLine,
		"first line of stdout must be the header comment with 0 contexts loaded")

	// No-match message and hint must appear.
	require.Contains(t, stdout, "No contexts matched the given filters",
		"stdout must contain the no-match message")
	require.Contains(t, stdout, "try broader filters",
		"stdout must contain the hint about broader filters")
}

// TestQueryCmd_Integration_UnknownTypeWarnsAndIgnored locks EP01RF-008:
// an unknown --type value is warned on stderr and dropped; known types still match.
func TestQueryCmd_Integration_UnknownTypeWarnsAndIgnored(t *testing.T) {
	t.Parallel()

	contexts := []filterTestContext{
		{id: "java-conventions", ctxType: "guidelines", module: "user-management", tags: []string{"java"}, body: "# Java Conventions\n\nJava style guide."},
		{id: "user-scenarios", ctxType: "scenarios", module: "user-management", tags: []string{"user"}, body: "# User Scenarios\n\nUser stories."},
	}

	tmpDir := buildFilterFixture(t, userManagementModules, contexts)

	// Use a real slog handler backed by a buffer so we can inspect warnings.
	var warnBuf bytes.Buffer
	warnLogger := slog.New(slog.NewTextHandler(&warnBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	manifestPath := filepath.Join(tmpDir, "project-state.yaml")
	store := fsmanifest.New(manifestPath)
	reader := fscontext.New()
	estimator := token.NewHeuristicEstimator()
	uc := appqueryuc.New(store, reader, estimator, discardLogger())

	var outBuf, errBuf bytes.Buffer
	cmd := command.NewQueryCmd(uc, warnLogger)
	cmd.SilenceUsage = true
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"--dir", tmpDir, "--module", "user-management", "--type", "guidelines,junk"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	stdout := outBuf.String()

	// The known type (guidelines) still matches.
	require.Contains(t, stdout, "Java Conventions",
		"stdout must contain the guidelines context despite the unknown type")

	// The unknown type must not cause a match (scenarios body absent).
	require.NotContains(t, stdout, "User Scenarios",
		"stdout must NOT contain scenarios body (only guidelines was requested and valid)")

	// The warn logger must have captured the EP01RF-008 warning.
	warnOutput := warnBuf.String()
	require.Contains(t, warnOutput, "ignoring unknown --type value",
		"stderr warn must mention 'ignoring unknown --type value'")
	require.Contains(t, warnOutput, "junk",
		"stderr warn must include the offending value 'junk'")
}
