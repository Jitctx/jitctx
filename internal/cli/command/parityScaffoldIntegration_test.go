package command_test

import (
	"context"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	appscaffolduc "github.com/jitctx/jitctx/internal/application/usecase/scaffolduc"
	"github.com/jitctx/jitctx/internal/domain/model"
	specport "github.com/jitctx/jitctx/internal/domain/port/spec"
	"github.com/jitctx/jitctx/internal/domain/service"
	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
	"github.com/jitctx/jitctx/internal/infrastructure/fsscaffold"
	"github.com/jitctx/jitctx/internal/infrastructure/fsspec"
	"github.com/jitctx/jitctx/internal/infrastructure/mdspec"
)

// bundleAwareProdAdapter wraps *fsscaffold.TemplateRegistry so that
// every Render call routes through RenderWithBundle(ctx, bundle, in)
// instead of Render(ctx, in). This satisfies spec.RenderProductionTemplatePort
// and lets appscaffolduc.New consume bundle templates without changing the
// use-case's constructor signature (per §7.2 / T6-G5).
type bundleAwareProdAdapter struct {
	base   *fsscaffold.TemplateRegistry
	bundle *model.ProfileBundle
}

var _ specport.RenderProductionTemplatePort = (*bundleAwareProdAdapter)(nil)

func (a *bundleAwareProdAdapter) Render(ctx context.Context, in scaffoldvo.RenderInput) ([]byte, error) {
	return a.base.RenderWithBundle(ctx, a.bundle, in)
}

// bundleAwareTestAdapter wraps *fsscaffold.TestTemplateRegistry analogously.
type bundleAwareTestAdapter struct {
	base   *fsscaffold.TestTemplateRegistry
	bundle *model.ProfileBundle
}

var _ specport.RenderTestTemplatePort = (*bundleAwareTestAdapter)(nil)

func (a *bundleAwareTestAdapter) Render(ctx context.Context, in scaffoldvo.TestRenderInput) ([]byte, error) {
	return a.base.RenderWithBundleTest(ctx, a.bundle, in)
}

// TestParity_Scaffold_MatchesEP03Baseline composes appscaffolduc.New with
// bundle-aware adapters and SHA-256-compares every generated file against
// the 8 files captured under testdata/ep04us004/parity/scaffold/baseline/.
//
// The test proves that the bundled spring-boot-hexagonal templates deliver
// byte-identical output to the EP-03 embedded-template path (§7.5).
func TestParity_Scaffold_MatchesEP03Baseline(t *testing.T) {
	t.Parallel()

	// ── 1. Load the bundle from the bundled embed ────────────────────────────
	bundled := fsprofile.NewBundled()
	bundle, err := bundled.LoadBundled(t.Context(), "spring-boot-hexagonal")
	require.NoError(t, err, "LoadBundled spring-boot-hexagonal")
	require.NotNil(t, bundle)

	// ── 2. Compose the use case with bundle-aware renderers ──────────────────
	registry := fsscaffold.NewRegistry()
	testRegistry := fsscaffold.NewTestRegistry()

	prodAdapter := &bundleAwareProdAdapter{base: registry, bundle: bundle}
	testAdapter := &bundleAwareTestAdapter{base: testRegistry, bundle: bundle}

	specFinder := fsspec.NewFinder()
	parser := mdspec.New()
	mapper := service.NewContractPathMapper()
	testMapper := service.NewTestPathMapper()
	importResolver := service.NewJavaImportResolver(mapper)
	endpointSynth := service.NewEndpointSynthesizer()
	idUtils := service.NewJavaIdentifierUtils()
	methodParser := service.NewMethodSignatureParser()
	jpaAnnotator := service.NewJPAFieldAnnotator()
	writer := fsscaffold.NewMultiFileWriter()

	uc := appscaffolduc.New(
		specFinder,
		parser,
		mapper,
		testMapper,
		importResolver,
		endpointSynth,
		idUtils,
		methodParser,
		jpaAnnotator,
		prodAdapter,
		testAdapter,
		writer,
		discardLogger(),
	)

	// ── 3. Copy the spec fixture into a temp workDir ─────────────────────────
	tmpDir := t.TempDir()
	specSrc := fixtureDir(t, "ep04us004", "parity", "scaffold", "fixture", "specs", "createUser.md")
	specDst := filepath.Join(tmpDir, "jitctx-plans", "create-user.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(specDst), 0o755))
	specBytes, err := os.ReadFile(specSrc)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(specDst, specBytes, 0o644))

	// ── 4. Execute the scaffold use case ─────────────────────────────────────
	out, err := uc.Execute(t.Context(), scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: tmpDir,
	})
	require.NoError(t, err)
	require.Equal(t, 5, out.ProductionCount, "expected 5 production files")
	require.Equal(t, 3, out.TestCount, "expected 3 test files")

	// ── 5. SHA-256-compare every baseline file ────────────────────────────────
	baselineRoot := fixtureDir(t, "ep04us004", "parity", "scaffold", "baseline")
	mismatches := scaffoldParityCheckBaseline(t, baselineRoot, tmpDir)
	require.Empty(t, mismatches,
		"scaffold bundle output must be byte-identical to EP03 baseline; mismatches: %v", mismatches)
}

// scaffoldParityCheckBaseline walks every file under baselineRoot, computes its
// SHA-256, locates the corresponding generated file under generatedRoot (using
// the same relative path), and returns a slice of mismatch descriptions.
func scaffoldParityCheckBaseline(t *testing.T, baselineRoot, generatedRoot string) []string {
	t.Helper()

	var mismatches []string
	err := filepath.WalkDir(baselineRoot, func(path string, d os.DirEntry, walkErr error) error {
		require.NoError(t, walkErr)
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(baselineRoot, path)
		require.NoError(t, err)

		baselineHash := sha256FileOrFail(t, path)

		generated := filepath.Join(generatedRoot, rel)
		if _, statErr := os.Stat(generated); os.IsNotExist(statErr) {
			mismatches = append(mismatches, "MISSING generated file: "+rel)
			return nil
		}

		generatedHash := sha256FileOrFail(t, generated)
		if baselineHash != generatedHash {
			mismatches = append(mismatches,
				"SHA-256 mismatch for "+rel+
					": baseline="+hashHex(baselineHash)+
					" generated="+hashHex(generatedHash),
			)
		}
		return nil
	})
	require.NoError(t, err)
	return mismatches
}

// sha256FileOrFail reads a file and returns its SHA-256 as a fixed-size array.
func sha256FileOrFail(t *testing.T, path string) [sha256.Size]byte {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	require.NoError(t, err)

	var sum [sha256.Size]byte
	copy(sum[:], h.Sum(nil))
	return sum
}

// hashHex returns a short hex representation (first 8 bytes) for diagnostic messages.
func hashHex(h [sha256.Size]byte) string {
	const hextable = "0123456789abcdef"
	buf := make([]byte, 16)
	for i := 0; i < 8; i++ {
		buf[i*2] = hextable[h[i]>>4]
		buf[i*2+1] = hextable[h[i]&0x0f]
	}
	return string(buf)
}
