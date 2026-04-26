package refactoruc_test

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apprefactoruc "github.com/jitctx/jitctx/internal/application/usecase/refactoruc"
	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/port/parser"
	"github.com/jitctx/jitctx/internal/domain/service"
	refactorvo "github.com/jitctx/jitctx/internal/domain/vo/refactor"
)

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

type fakeLoadManifestPort struct {
	load func(ctx context.Context) (*model.ProjectState, error)
}

func (f *fakeLoadManifestPort) Load(ctx context.Context) (*model.ProjectState, error) {
	return f.load(ctx)
}

type fakeWalkJavaFilesPort struct {
	walk func(ctx context.Context, fsys fs.FS) ([]string, error)
}

func (f *fakeWalkJavaFilesPort) WalkJavaFiles(ctx context.Context, fsys fs.FS) ([]string, error) {
	return f.walk(ctx, fsys)
}

type fakeListJavaCommentsPort struct {
	list func(ctx context.Context, fsys fs.FS, path string) ([]parser.JavaComment, error)
}

func (f *fakeListJavaCommentsPort) ListJavaComments(ctx context.Context, fsys fs.FS, path string) ([]parser.JavaComment, error) {
	return f.list(ctx, fsys, path)
}

// fakeFileMTimePort is a fake for gitport.FileLastModifiedTimePort.
// The get field is called on every invocation; if nil, returns ErrGitUnavailable.
type fakeFileMTimePort struct {
	get func(ctx context.Context, repoRoot, filePath string) (time.Time, error)
}

func (f *fakeFileMTimePort) Get(ctx context.Context, repoRoot, filePath string) (time.Time, error) {
	if f.get == nil {
		return time.Time{}, domerr.ErrGitUnavailable
	}
	return f.get(ctx, repoRoot, filePath)
}

// fakeLineMTimePort is a fake for gitport.LineIntroducedTimePort.
// The get field is called on every invocation; if nil, returns ErrGitUnavailable.
type fakeLineMTimePort struct {
	get func(ctx context.Context, repoRoot, filePath string, line int) (time.Time, error)
}

func (f *fakeLineMTimePort) Get(ctx context.Context, repoRoot, filePath string, line int) (time.Time, error) {
	if f.get == nil {
		return time.Time{}, domerr.ErrGitUnavailable
	}
	return f.get(ctx, repoRoot, filePath, line)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func nopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// defaultGitUnavailableFakes returns fakes that always report git as unavailable.
// Used by buildUC so that all pre-existing tests see StaleSkipped=true and
// Stale=false on every marker — no behavioural regression.
func defaultGitUnavailableFakes() (*fakeFileMTimePort, *fakeLineMTimePort) {
	return &fakeFileMTimePort{}, &fakeLineMTimePort{}
}

func buildUC(
	loadFn func(context.Context) (*model.ProjectState, error),
	walkFn func(context.Context, fs.FS) ([]string, error),
	listFn func(context.Context, fs.FS, string) ([]parser.JavaComment, error),
) *apprefactoruc.Impl {
	fileMTime, lineMTime := defaultGitUnavailableFakes()
	return apprefactoruc.New(
		&fakeLoadManifestPort{load: loadFn},
		&fakeWalkJavaFilesPort{walk: walkFn},
		&fakeListJavaCommentsPort{list: listFn},
		fileMTime,
		lineMTime,
		service.NewMarkerParser(),
		nopLogger(),
	)
}

// buildUCWithGit is like buildUC but accepts explicit git port fakes.
// Used by tests that exercise stale detection behaviour.
func buildUCWithGit(
	loadFn func(context.Context) (*model.ProjectState, error),
	walkFn func(context.Context, fs.FS) ([]string, error),
	listFn func(context.Context, fs.FS, string) ([]parser.JavaComment, error),
	fileMTime *fakeFileMTimePort,
	lineMTime *fakeLineMTimePort,
) *apprefactoruc.Impl {
	return apprefactoruc.New(
		&fakeLoadManifestPort{load: loadFn},
		&fakeWalkJavaFilesPort{walk: walkFn},
		&fakeListJavaCommentsPort{list: listFn},
		fileMTime,
		lineMTime,
		service.NewMarkerParser(),
		nopLogger(),
	)
}

// stateWithModule returns a ProjectState with one module rooted at the given path.
func stateWithModule(moduleID, modulePath string) *model.ProjectState {
	return &model.ProjectState{
		Modules: []model.Module{
			{ID: moduleID, Path: modulePath},
		},
	}
}

// lineComment builds a JavaComment of kind line_comment.
func lineComment(line int, text string) parser.JavaComment {
	return parser.JavaComment{Kind: "line_comment", Line: line, Text: text}
}

// defaultInput is a minimal ScanRefactorsInput that points at the temp dir.
// The use case opens os.DirFS(WorkDir) — the fakes never touch the filesystem,
// so any non-empty string is fine.
func defaultInput(t *testing.T) refactorvo.ScanRefactorsInput {
	t.Helper()
	return refactorvo.ScanRefactorsInput{
		WorkDir:      t.TempDir(),
		ManifestPath: "project-state.yaml",
	}
}

// ---------------------------------------------------------------------------
// Pre-existing tests (updated to pass default-unavailable git fakes via buildUC)
// ---------------------------------------------------------------------------

// TestRefactorUC_HappyPath: manifest with one module, walker returns one Java
// file, comments port returns one well-formed marker comment → output has one
// marker bucketed under the right module.
func TestRefactorUC_HappyPath(t *testing.T) {
	t.Parallel()

	const moduleID = "mod-user"
	const modulePath = "src/main/java/com/app/user"
	const filePath = "src/main/java/com/app/user/UserService.java"

	uc := buildUC(
		func(_ context.Context) (*model.ProjectState, error) {
			return stateWithModule(moduleID, modulePath), nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{filePath}, nil
		},
		func(_ context.Context, _ fs.FS, _ string) ([]parser.JavaComment, error) {
			return []parser.JavaComment{
				lineComment(10, "// TODO(jitctx): extract-method - extract payment logic"),
			}, nil
		},
	)

	out, err := uc.Execute(context.Background(), defaultInput(t))
	require.NoError(t, err)

	require.True(t, out.ManifestPresent, "ManifestPresent must be true when manifest loads")
	require.Len(t, out.Markers, 1)

	m := out.Markers[0]
	require.Equal(t, moduleID, m.ModuleID)
	require.Equal(t, filePath, m.FilePath)
	require.Equal(t, 10, m.Line)
	require.Equal(t, refactorvo.MarkerTypeExtractMethod, m.Type)
	require.Equal(t, "extract payment logic", m.Description)
	require.Empty(t, out.UnknownTypes)
}

// TestRefactorUC_ManifestAbsent: LoadManifestPort returns ErrManifestNotFound →
// use case proceeds, marker is bucketed under "<unmoduled>", ManifestPresent=false.
func TestRefactorUC_ManifestAbsent(t *testing.T) {
	t.Parallel()

	const filePath = "src/main/java/com/app/SomeService.java"

	uc := buildUC(
		func(_ context.Context) (*model.ProjectState, error) {
			return nil, domerr.ErrManifestNotFound
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{filePath}, nil
		},
		func(_ context.Context, _ fs.FS, _ string) ([]parser.JavaComment, error) {
			return []parser.JavaComment{
				lineComment(5, "// TODO(jitctx): rename - use domain naming"),
			}, nil
		},
	)

	out, err := uc.Execute(context.Background(), defaultInput(t))
	require.NoError(t, err, "ErrManifestNotFound must NOT abort the use case")

	require.False(t, out.ManifestPresent, "ManifestPresent must be false when manifest absent")
	require.Len(t, out.Markers, 1)
	require.Equal(t, "<unmoduled>", out.Markers[0].ModuleID,
		"marker must be bucketed under <unmoduled> when manifest is missing")
}

// TestRefactorUC_PartialParseTolerance: comments port returns (comments,
// ErrPartialParse) → use case STILL emits markers from partial results, does
// NOT return an error.
func TestRefactorUC_PartialParseTolerance(t *testing.T) {
	t.Parallel()

	const filePath = "src/main/java/com/app/BrokenSyntax.java"

	uc := buildUC(
		func(_ context.Context) (*model.ProjectState, error) {
			return stateWithModule("mod-app", "src/main/java/com/app"), nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{filePath}, nil
		},
		func(_ context.Context, _ fs.FS, _ string) ([]parser.JavaComment, error) {
			// Return partial results alongside ErrPartialParse.
			return []parser.JavaComment{
				lineComment(7, "// TODO(jitctx): simplify - reduce cyclomatic complexity"),
			}, domerr.ErrPartialParse
		},
	)

	out, err := uc.Execute(context.Background(), defaultInput(t))
	require.NoError(t, err, "ErrPartialParse must not abort the use case")
	require.Len(t, out.Markers, 1,
		"markers from partial parse must still appear in output")
	require.Equal(t, refactorvo.MarkerTypeSimplify, out.Markers[0].Type)
}

// TestRefactorUC_UnknownTypesDedupeSorted: three comments with unknown types
// "c-rule", "a-rule", "c-rule" → UnknownTypes is ["a-rule", "c-rule"] (sorted,
// deduped).
func TestRefactorUC_UnknownTypesDedupeSorted(t *testing.T) {
	t.Parallel()

	const filePath = "src/main/java/com/app/Mixed.java"

	uc := buildUC(
		func(_ context.Context) (*model.ProjectState, error) {
			return &model.ProjectState{}, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{filePath}, nil
		},
		func(_ context.Context, _ fs.FS, _ string) ([]parser.JavaComment, error) {
			return []parser.JavaComment{
				lineComment(1, "// TODO(jitctx): c-rule - some desc"),
				lineComment(2, "// TODO(jitctx): a-rule - another desc"),
				lineComment(3, "// TODO(jitctx): c-rule - third desc"),
			}, nil
		},
	)

	out, err := uc.Execute(context.Background(), defaultInput(t))
	require.NoError(t, err)

	require.Equal(t, []string{"a-rule", "c-rule"}, out.UnknownTypes,
		"UnknownTypes must be sorted and deduped")

	// Each unknown marker must be bucketed as "other".
	for _, m := range out.Markers {
		require.Equal(t, refactorvo.MarkerTypeOther, m.Type,
			"unknown marker type must be bucketed as 'other'")
	}
}

// TestRefactorUC_SortOrder: markers across multiple files/modules → output is
// sorted by (ModuleID, FilePath, Line) with "<unmoduled>" last.
func TestRefactorUC_SortOrder(t *testing.T) {
	t.Parallel()

	// Two modules: mod-alpha covers alpha/ and mod-beta covers beta/.
	// A third file has no module match → <unmoduled>.
	fileAlpha := "src/main/java/com/app/alpha/Alpha.java"
	fileBeta := "src/main/java/com/app/beta/Beta.java"
	fileOrphan := "src/main/java/com/app/orphan/Orphan.java"

	state := &model.ProjectState{
		Modules: []model.Module{
			{ID: "mod-alpha", Path: "src/main/java/com/app/alpha"},
			{ID: "mod-beta", Path: "src/main/java/com/app/beta"},
		},
	}

	uc := buildUC(
		func(_ context.Context) (*model.ProjectState, error) {
			return state, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			// Return files in reverse "desired" order to stress the sort.
			return []string{fileOrphan, fileBeta, fileAlpha}, nil
		},
		func(_ context.Context, _ fs.FS, path string) ([]parser.JavaComment, error) {
			switch path {
			case fileAlpha:
				return []parser.JavaComment{
					lineComment(20, "// TODO(jitctx): rename - alpha rename"),
					lineComment(5, "// TODO(jitctx): move - alpha move"),
				}, nil
			case fileBeta:
				return []parser.JavaComment{
					lineComment(3, "// TODO(jitctx): inline - beta inline"),
				}, nil
			case fileOrphan:
				return []parser.JavaComment{
					lineComment(1, "// TODO(jitctx): simplify - orphan simplify"),
				}, nil
			}
			return nil, nil
		},
	)

	out, err := uc.Execute(context.Background(), defaultInput(t))
	require.NoError(t, err)
	require.Len(t, out.Markers, 4)

	// Verify sorted order: mod-alpha (line 5 before 20), mod-beta, <unmoduled> last.
	require.Equal(t, "mod-alpha", out.Markers[0].ModuleID)
	require.Equal(t, fileAlpha, out.Markers[0].FilePath)
	require.Equal(t, 5, out.Markers[0].Line)

	require.Equal(t, "mod-alpha", out.Markers[1].ModuleID)
	require.Equal(t, 20, out.Markers[1].Line)

	require.Equal(t, "mod-beta", out.Markers[2].ModuleID)

	require.Equal(t, "<unmoduled>", out.Markers[3].ModuleID,
		"<unmoduled> must sort last")
}

// TestRefactorUC_CtxCancellationOnEntry: cancelled context at entry → returns
// ctx.Err() immediately.
func TestRefactorUC_CtxCancellationOnEntry(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Execute is called

	uc := buildUC(
		func(_ context.Context) (*model.ProjectState, error) {
			return &model.ProjectState{}, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{"src/main/java/com/app/Foo.java"}, nil
		},
		func(_ context.Context, _ fs.FS, _ string) ([]parser.JavaComment, error) {
			return nil, nil
		},
	)

	_, err := uc.Execute(ctx, defaultInput(t))
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled),
		"cancelled context must propagate as context.Canceled")
}

// TestRefactorUC_CtxCancellationMidLoop: context cancelled during file loop →
// Execute returns ctx.Err() before completing the walk.
func TestRefactorUC_CtxCancellationMidLoop(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	uc := buildUC(
		func(_ context.Context) (*model.ProjectState, error) {
			return &model.ProjectState{}, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{
				"src/main/java/com/app/First.java",
				"src/main/java/com/app/Second.java",
			}, nil
		},
		func(_ context.Context, _ fs.FS, _ string) ([]parser.JavaComment, error) {
			callCount++
			// Cancel after the first file is processed.
			if callCount == 1 {
				cancel()
			}
			return nil, nil
		},
	)

	_, err := uc.Execute(ctx, defaultInput(t))
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled),
		"mid-loop cancellation must propagate as context.Canceled")
}

// TestRefactorUC_ReadOnlyGuardrail: reflectively enumerate Impl fields and
// assert no field type name ends with a write-port suffix (RNF-002).
func TestRefactorUC_ReadOnlyGuardrail(t *testing.T) {
	t.Parallel()

	forbiddenSuffixes := []string{
		"SavePort",
		"WritePort",
		"WriterPort",
		"SaverPort",
		"PersistPort",
	}

	implType := reflect.TypeOf(apprefactoruc.Impl{})
	for i := range implType.NumField() {
		field := implType.Field(i)
		typeName := field.Type.Name()
		if field.Type.Kind() == reflect.Interface {
			typeName = field.Type.Name()
		}
		for _, suffix := range forbiddenSuffixes {
			require.False(t,
				strings.HasSuffix(typeName, suffix),
				"Impl field %q has a write-port type %q (suffix %q) — violates RNF-002",
				field.Name, typeName, suffix,
			)
		}
	}
}

// TestRefactorUC_Determinism: two consecutive runs with identical fakes return
// outputs that compare deeply equal.
func TestRefactorUC_Determinism(t *testing.T) {
	t.Parallel()

	state := stateWithModule("mod-payments", "src/main/java/com/app/payments")
	files := []string{
		"src/main/java/com/app/payments/PaymentService.java",
		"src/main/java/com/app/payments/InvoiceService.java",
	}

	makeUC := func() *apprefactoruc.Impl {
		return buildUC(
			func(_ context.Context) (*model.ProjectState, error) {
				return state, nil
			},
			func(_ context.Context, _ fs.FS) ([]string, error) {
				return []string{files[0], files[1]}, nil
			},
			func(_ context.Context, _ fs.FS, path string) ([]parser.JavaComment, error) {
				switch path {
				case files[0]:
					return []parser.JavaComment{
						lineComment(3, "// TODO(jitctx): extract-method - extract charge logic"),
						lineComment(15, "// TODO(jitctx): rename - rename processPayment"),
					}, nil
				case files[1]:
					return []parser.JavaComment{
						lineComment(8, "// TODO(jitctx): move - move to billing module"),
					}, nil
				}
				return nil, nil
			},
		)
	}

	in := defaultInput(t)

	out1, err1 := makeUC().Execute(context.Background(), in)
	require.NoError(t, err1)

	out2, err2 := makeUC().Execute(context.Background(), in)
	require.NoError(t, err2)

	require.Equal(t, out1, out2,
		"two consecutive Execute calls with identical fakes must return deeply equal output")
}

// ---------------------------------------------------------------------------
// New stale-detection tests (EP03RF-009 / EP03US-006)
// ---------------------------------------------------------------------------

// TestRefactorUC_StaleSkipped: both git ports return ErrGitUnavailable →
// StaleSkipped is true and every marker has Stale=false.
func TestRefactorUC_StaleSkipped(t *testing.T) {
	t.Parallel()

	const filePath = "src/main/java/com/app/BillingService.java"

	fileMTime := &fakeFileMTimePort{} // nil get → ErrGitUnavailable
	lineMTime := &fakeLineMTimePort{} // nil get → ErrGitUnavailable

	uc := buildUCWithGit(
		func(_ context.Context) (*model.ProjectState, error) {
			return &model.ProjectState{}, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{filePath}, nil
		},
		func(_ context.Context, _ fs.FS, _ string) ([]parser.JavaComment, error) {
			return []parser.JavaComment{
				lineComment(5, "// TODO(jitctx): rename - rename processPayment"),
			}, nil
		},
		fileMTime,
		lineMTime,
	)

	out, err := uc.Execute(context.Background(), defaultInput(t))
	require.NoError(t, err)

	require.True(t, out.StaleSkipped, "StaleSkipped must be true when git is unavailable")
	require.Len(t, out.Markers, 1)
	require.False(t, out.Markers[0].Stale, "Stale must be false when git is unavailable")
}

// TestRefactorUC_MarkerStale: file was modified after the marker line was
// introduced → marker.Stale=true, StaleSkipped=false.
// file mtime = 5 days ago (recent); line mtime = 30 days ago (old).
func TestRefactorUC_MarkerStale(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	fileMTimeVal := now.Add(-5 * 24 * time.Hour)  // 5 days ago — file was recently modified
	lineMTimeVal := now.Add(-30 * 24 * time.Hour) // 30 days ago — marker was written earlier

	const filePath = "src/main/java/com/app/BillingService.java"

	fileMTime := &fakeFileMTimePort{
		get: func(_ context.Context, _, _ string) (time.Time, error) {
			return fileMTimeVal, nil
		},
	}
	lineMTime := &fakeLineMTimePort{
		get: func(_ context.Context, _, _ string, _ int) (time.Time, error) {
			return lineMTimeVal, nil
		},
	}

	uc := buildUCWithGit(
		func(_ context.Context) (*model.ProjectState, error) {
			return &model.ProjectState{}, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{filePath}, nil
		},
		func(_ context.Context, _ fs.FS, _ string) ([]parser.JavaComment, error) {
			return []parser.JavaComment{
				lineComment(10, "// TODO(jitctx): extract-method - extract payment logic"),
			}, nil
		},
		fileMTime,
		lineMTime,
	)

	out, err := uc.Execute(context.Background(), defaultInput(t))
	require.NoError(t, err)

	require.False(t, out.StaleSkipped, "StaleSkipped must be false when git is available")
	require.Len(t, out.Markers, 1)
	require.True(t, out.Markers[0].Stale,
		"marker must be Stale when the file was modified after the marker line was introduced")
}

// TestRefactorUC_NotStale: file mtime equals line mtime → file was not modified
// after the marker was added → marker.Stale=false, StaleSkipped=false.
func TestRefactorUC_NotStale(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	sharedMTime := now.Add(-30 * 24 * time.Hour) // both file and line have the same mtime

	const filePath = "src/main/java/com/app/UserService.java"

	fileMTime := &fakeFileMTimePort{
		get: func(_ context.Context, _, _ string) (time.Time, error) {
			return sharedMTime, nil
		},
	}
	lineMTime := &fakeLineMTimePort{
		get: func(_ context.Context, _, _ string, _ int) (time.Time, error) {
			return sharedMTime, nil
		},
	}

	uc := buildUCWithGit(
		func(_ context.Context) (*model.ProjectState, error) {
			return &model.ProjectState{}, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{filePath}, nil
		},
		func(_ context.Context, _ fs.FS, _ string) ([]parser.JavaComment, error) {
			return []parser.JavaComment{
				lineComment(7, "// TODO(jitctx): simplify - reduce complexity"),
			}, nil
		},
		fileMTime,
		lineMTime,
	)

	out, err := uc.Execute(context.Background(), defaultInput(t))
	require.NoError(t, err)

	require.False(t, out.StaleSkipped, "StaleSkipped must be false when git is available")
	require.Len(t, out.Markers, 1)
	require.False(t, out.Markers[0].Stale,
		"marker must not be Stale when the file mtime equals the line mtime")
}

// TestRefactorUC_PerMarkerErrorTolerated: the line-mtime port returns a
// non-ErrGitUnavailable error for one marker → use case logs at debug, leaves
// Stale=false for that marker, does not return an error, and StaleSkipped=false.
func TestRefactorUC_PerMarkerErrorTolerated(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	fileMTimeVal := now.Add(-5 * 24 * time.Hour)

	const filePath = "src/main/java/com/app/InvoiceService.java"
	perMarkerErr := errors.New("file not tracked")

	fileMTime := &fakeFileMTimePort{
		get: func(_ context.Context, _, _ string) (time.Time, error) {
			return fileMTimeVal, nil
		},
	}
	// line port returns a non-unavailability error — per-marker tolerated.
	lineMTime := &fakeLineMTimePort{
		get: func(_ context.Context, _, _ string, _ int) (time.Time, error) {
			return time.Time{}, perMarkerErr
		},
	}

	uc := buildUCWithGit(
		func(_ context.Context) (*model.ProjectState, error) {
			return &model.ProjectState{}, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{filePath}, nil
		},
		func(_ context.Context, _ fs.FS, _ string) ([]parser.JavaComment, error) {
			return []parser.JavaComment{
				lineComment(12, "// TODO(jitctx): move - move to payments module"),
			}, nil
		},
		fileMTime,
		lineMTime,
	)

	out, err := uc.Execute(context.Background(), defaultInput(t))
	require.NoError(t, err, "per-marker error must not abort the use case")

	require.False(t, out.StaleSkipped, "StaleSkipped must be false — git IS available, only this marker failed")
	require.Len(t, out.Markers, 1)
	require.False(t, out.Markers[0].Stale,
		"Stale must be false when the line-mtime query fails for that marker")
}

// TestRefactorUC_NoMarkersNoProbe: walker returns no files → zero markers →
// git probe is skipped entirely, StaleSkipped remains false.
func TestRefactorUC_NoMarkersNoProbe(t *testing.T) {
	t.Parallel()

	probeCallCount := 0

	fileMTime := &fakeFileMTimePort{
		get: func(_ context.Context, _, _ string) (time.Time, error) {
			probeCallCount++
			return time.Time{}, domerr.ErrGitUnavailable
		},
	}
	lineMTime := &fakeLineMTimePort{
		get: func(_ context.Context, _, _ string, _ int) (time.Time, error) {
			probeCallCount++
			return time.Time{}, domerr.ErrGitUnavailable
		},
	}

	uc := buildUCWithGit(
		func(_ context.Context) (*model.ProjectState, error) {
			return &model.ProjectState{}, nil
		},
		// walker returns no files → no markers → no probe.
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{}, nil
		},
		func(_ context.Context, _ fs.FS, _ string) ([]parser.JavaComment, error) {
			return nil, nil
		},
		fileMTime,
		lineMTime,
	)

	out, err := uc.Execute(context.Background(), defaultInput(t))
	require.NoError(t, err)

	require.Empty(t, out.Markers, "no markers expected when walker returns no files")
	require.False(t, out.StaleSkipped,
		"StaleSkipped must be false when there are no markers (probe is skipped, per plan Q10)")
	require.Equal(t, 0, probeCallCount,
		"git ports must NOT be called when there are zero markers")
}
