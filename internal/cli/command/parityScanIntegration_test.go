package command_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	appscanuc "github.com/jitctx/jitctx/internal/application/usecase/scanuc"
	domspecsvc "github.com/jitctx/jitctx/internal/domain/service"
	"github.com/jitctx/jitctx/internal/infrastructure/fscontext"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
	"github.com/jitctx/jitctx/internal/infrastructure/token"
	"github.com/jitctx/jitctx/internal/infrastructure/treesitter"

	scanvo "github.com/jitctx/jitctx/internal/domain/vo/scan"
)

// TestParity_Scan_MatchesEP03Baseline verifies that the EP-04 declarative profile
// architecture (bundled embed resolver path) produces a manifest byte-identical to
// the EP-03 baseline (modulo generated_at). Routes exclusively through the bundled
// embed — the fixture contains no .jitctx/profiles/ directory.
func TestParity_Scan_MatchesEP03Baseline(t *testing.T) {
	t.Parallel()

	// 1. Set up: copy fixture into a tmp dir.
	tmpDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "ep04us004", "parity", "scan", "fixture"), tmpDir)

	// 2. Compose use case manually with the bundled-resolver-backed adapter.
	//    The fixture has no .jitctx/profiles/ directory so the resolver falls
	//    through to the bundled embed (ProfileSourceBundled).
	logger := discardLogger()
	bundled := fsprofile.NewBundled()
	bundleLoader := fsprofile.NewBundleLoader(logger)
	resolver := fsprofile.NewResolver(bundleLoader, bundled, logger)
	declarativeClassifier := domspecsvc.NewDeclarativeClassifier()
	walker := treesitter.NewWalker()
	parser := treesitter.New()
	discoverer := fscontext.New()
	estimator := token.NewHeuristicEstimator()

	manifestPath := filepath.Join(tmpDir, "project-state.yaml")
	store := fsmanifest.New(manifestPath)

	// profilesDir is empty — no user profiles; resolver must use bundled embed.
	uc := appscanuc.New(
		resolver,
		declarativeClassifier,
		walker,
		parser,
		discoverer,
		discoverer,
		estimator,
		store,
		".jitctx/profiles",
		logger,
	)

	// 3. Run scan and assert no error.
	scanIn := scanvo.ScanProjectInput{
		WorkDir:      tmpDir,
		ProfileName:  "spring-boot-hexagonal",
		ManifestPath: manifestPath,
	}
	_, err := uc.Execute(t.Context(), scanIn)
	require.NoError(t, err)

	// 4. Read the generated manifest.
	generated, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	// 5. Read the baseline.
	baselinePath := fixtureDir(t, "ep04us004", "parity", "scan", "baseline", "project-state.yaml")
	baseline, err := os.ReadFile(baselinePath)
	require.NoError(t, err)

	// 6. Strip generated_at from both.
	generatedStripped := normalizeYAML(stripGeneratedAt(string(generated)))
	baselineStripped := normalizeYAML(stripGeneratedAt(string(baseline)))

	// 7. Byte-compare.
	require.Equal(t, baselineStripped, generatedStripped,
		"scan output must byte-match EP-03 baseline (modulo generated_at)")
}
