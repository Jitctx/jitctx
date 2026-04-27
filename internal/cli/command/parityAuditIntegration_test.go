package command_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	appaudituc "github.com/jitctx/jitctx/internal/application/usecase/audituc"
	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/service"
	auditvo "github.com/jitctx/jitctx/internal/domain/vo/audit"
	"github.com/jitctx/jitctx/internal/infrastructure/fsconfig"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
	"github.com/jitctx/jitctx/internal/infrastructure/treesitter"
)

// buildBundleAuditUC constructs an audituc.Impl wired for the bundle path.
//
// The detector is backed by an empty profilesDir so the legacy single-file
// path returns ErrNoProfileMatch, forcing the use case into the resolver
// (bundle) branch. The BundleAuditRulesAdapter then supplies the audit rules
// from the bundled spring-boot-hexagonal profile.yaml.
func buildBundleAuditUC(t *testing.T, manifestPath string) *appaudituc.Impl {
	t.Helper()

	logger := discardLogger()

	// Legacy single-file adapters — backed by an empty dir so they do not match.
	profileDetector := fsprofile.NewDetectorWithLogger("", logger)
	auditRulesLoader := fsprofile.NewAuditRulesLoader("", logger)

	// Bundle path adapters (EP04US-004).
	bundleLoader := fsprofile.NewBundleLoader(logger, nil)
	bundled := fsprofile.NewBundled()
	resolver := fsprofile.NewResolver(bundleLoader, bundled, logger)
	bundleAuditRules := fsprofile.NewBundleAuditRulesAdapter()

	manifestStore := fsmanifest.New(manifestPath)
	tsParser := treesitter.New()
	tsWalker := treesitter.NewWalker()
	evaluator := service.NewAuditEvaluator()
	configLoader := fsconfig.New(logger)
	auditFilter := service.NewAuditRuleFilter()

	return appaudituc.New(
		manifestStore,
		profileDetector,
		auditRulesLoader,
		tsWalker,
		tsParser,
		tsParser,
		configLoader,
		auditFilter,
		evaluator,
		logger,
		bundleAuditRules, // profile.LoadBundleAuditRulesPort (EP04US-004)
		resolver,         // profile.ResolveProfilePort (EP04US-004)
		"",               // profilesDir: empty — no user profiles; routes to bundled embed
	)
}

// TestParity_Audit_CleanProject_MatchesEP03Baseline copies the auditClean
// parity fixture (no .jitctx/profiles/ directory) into a tmpdir, runs the
// audit use case via the bundle path, and asserts that the rendered stdout
// is byte-identical to the captured EP-03 baseline report.md.
//
// The fixture's absence of a single-file profile YAML forces the detector to
// return ErrNoProfileMatch, triggering the resolver → bundled embed fallback
// (per §5.2 selection precedence). The bundled audit_rules are byte-identical
// to the EP-03 spring-boot-hexagonal.yaml (per §4.4), so the output must match.
func TestParity_Audit_CleanProject_MatchesEP03Baseline(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "ep04us004", "parity", "audit", "auditClean", "fixture"), tmpDir)

	manifestPath := filepath.Join(tmpDir, "project-state.yaml")
	uc := buildBundleAuditUC(t, manifestPath)

	out, err := uc.Execute(t.Context(), auditvo.AuditProjectInput{
		WorkDir:      tmpDir,
		ManifestPath: manifestPath,
		ProfileName:  "spring-boot-hexagonal",
	})
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, format.WriteAuditReport(&buf, out))

	baselinePath := fixtureDir(t, "ep04us004", "parity", "audit", "auditClean", "baseline", "report.md")
	expected, err := os.ReadFile(baselinePath)
	require.NoError(t, err)

	require.Equal(t, string(expected), buf.String(),
		"audit report for clean project must byte-match EP-03 baseline (EP04US-004 parity)")
}

// TestParity_Audit_ViolationsProject_MatchesEP03Baseline copies the
// auditViolations parity fixture into a tmpdir, runs the audit use case via
// the bundle path, and asserts that the rendered stdout is byte-identical to
// the captured EP-03 baseline report.md containing five violations.
//
// The same bundle-path selection logic applies: no .jitctx/profiles/ in the
// fixture → ErrNoProfileMatch → resolver → bundled embed.
func TestParity_Audit_ViolationsProject_MatchesEP03Baseline(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "ep04us004", "parity", "audit", "auditViolations", "fixture"), tmpDir)

	manifestPath := filepath.Join(tmpDir, "project-state.yaml")
	uc := buildBundleAuditUC(t, manifestPath)

	out, err := uc.Execute(t.Context(), auditvo.AuditProjectInput{
		WorkDir:      tmpDir,
		ManifestPath: manifestPath,
		ProfileName:  "spring-boot-hexagonal",
	})
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, format.WriteAuditReport(&buf, out))

	baselinePath := fixtureDir(t, "ep04us004", "parity", "audit", "auditViolations", "baseline", "report.md")
	expected, err := os.ReadFile(baselinePath)
	require.NoError(t, err)

	require.Equal(t, string(expected), buf.String(),
		"audit report for violations project must byte-match EP-03 baseline (EP04US-004 parity)")
}
