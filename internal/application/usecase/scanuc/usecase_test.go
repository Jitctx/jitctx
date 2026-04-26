package scanuc_test

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	appscanuc "github.com/jitctx/jitctx/internal/application/usecase/scanuc"
	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
	scanvo "github.com/jitctx/jitctx/internal/domain/vo/scan"
)

// --- Fakes ---

type fakeClassifyDeclarativePort struct {
	classify func(ctx context.Context, input profilevo.ClassificationInput, types []model.ProfileTypeDeclaration) ([]string, error)
}

func (f *fakeClassifyDeclarativePort) ClassifyDeclarative(
	ctx context.Context,
	input profilevo.ClassificationInput,
	types []model.ProfileTypeDeclaration,
) ([]string, error) {
	if f.classify != nil {
		return f.classify(ctx, input, types)
	}
	return nil, nil
}

type fakeDetectPort struct {
	detect func(ctx context.Context, fsys fs.FS) (*model.FrameworkProfile, error)
}

func (f *fakeDetectPort) Detect(ctx context.Context, fsys fs.FS) (*model.FrameworkProfile, error) {
	return f.detect(ctx, fsys)
}

type fakeWalkPort struct {
	walk func(ctx context.Context, fsys fs.FS) ([]string, error)
}

func (f *fakeWalkPort) WalkJavaFiles(ctx context.Context, fsys fs.FS) ([]string, error) {
	return f.walk(ctx, fsys)
}

type fakeParsePort struct {
	parse func(ctx context.Context, fsys fs.FS, path string) (model.JavaFileSummary, error)
}

func (f *fakeParsePort) ParseJavaFile(ctx context.Context, fsys fs.FS, path string) (model.JavaFileSummary, error) {
	return f.parse(ctx, fsys, path)
}

type fakeDiscoverPort struct {
	discover func(ctx context.Context, fsys fs.FS) ([]model.Context, error)
}

func (f *fakeDiscoverPort) DiscoverContexts(ctx context.Context, fsys fs.FS) ([]model.Context, error) {
	return f.discover(ctx, fsys)
}

type fakeReadBodyPort struct {
	read func(ctx context.Context, fsys fs.FS, path string) (string, error)
}

func (f *fakeReadBodyPort) ReadContextBody(ctx context.Context, fsys fs.FS, path string) (string, error) {
	return f.read(ctx, fsys, path)
}

type fakeEstimatePort struct {
	estimate func(ctx context.Context, text string) (int, error)
}

func (f *fakeEstimatePort) Estimate(ctx context.Context, text string) (int, error) {
	return f.estimate(ctx, text)
}

type fakeSavePort struct {
	save func(ctx context.Context, state *model.ProjectState) error
}

func (f *fakeSavePort) Save(ctx context.Context, state *model.ProjectState) error {
	return f.save(ctx, state)
}

// --- Test helpers ---

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, &slog.HandlerOptions{Level: slog.LevelError}))
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func buildMinimalUseCase(
	detectFn func(context.Context, fs.FS) (*model.FrameworkProfile, error),
	walkFn func(context.Context, fs.FS) ([]string, error),
	parseFn func(context.Context, fs.FS, string) (model.JavaFileSummary, error),
	saveFn func(context.Context, *model.ProjectState) error,
) *appscanuc.Impl {
	return appscanuc.New(
		&fakeDetectPort{detect: detectFn},
		&fakeClassifyDeclarativePort{},
		&fakeWalkPort{walk: walkFn},
		&fakeParsePort{parse: parseFn},
		&fakeDiscoverPort{discover: func(_ context.Context, _ fs.FS) ([]model.Context, error) {
			return nil, nil
		}},
		&fakeReadBodyPort{read: func(_ context.Context, _ fs.FS, _ string) (string, error) {
			return "", nil
		}},
		&fakeEstimatePort{estimate: func(_ context.Context, _ string) (int, error) {
			return 0, nil
		}},
		&fakeSavePort{save: saveFn},
		noopLogger(),
	)
}

func minimalProfile() *model.FrameworkProfile {
	return &model.FrameworkProfile{
		Name:      "spring-boot-hexagonal",
		Languages: []string{"java"},
		Rules: []model.ProfileRule{
			{Match: model.ProfileMatch{NodeType: "interface_declaration", PathContains: "/port/in/"}, ClassifyAs: model.ContractInputPort},
		},
	}
}

// --- Tests ---

func TestScanUC_HappyPath(t *testing.T) {
	t.Parallel()

	saved := false
	uc := buildMinimalUseCase(
		func(_ context.Context, _ fs.FS) (*model.FrameworkProfile, error) { return minimalProfile(), nil },
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{"src/main/java/com/app/user/port/in/CreateUserUseCase.java"}, nil
		},
		func(_ context.Context, _ fs.FS, path string) (model.JavaFileSummary, error) {
			return model.JavaFileSummary{
				Path:    path,
				Package: "com.app.user.port.in",
				Declarations: []model.JavaDeclaration{
					{NodeType: "interface_declaration", Name: "CreateUserUseCase"},
				},
			}, nil
		},
		func(_ context.Context, _ *model.ProjectState) error {
			saved = true
			return nil
		},
	)

	fsys := fstest.MapFS{}
	_ = fsys

	out, err := uc.Execute(context.Background(), scanvo.ScanProjectInput{
		WorkDir:      t.TempDir(),
		ManifestPath: "project-state.yaml",
	})
	require.NoError(t, err)
	require.True(t, saved)
	require.Equal(t, 1, out.ModuleCount)
	require.Equal(t, "project-state.yaml", out.ManifestPath)
}

func TestScanUC_ErrNoProfileMatch(t *testing.T) {
	t.Parallel()

	uc := buildMinimalUseCase(
		func(_ context.Context, _ fs.FS) (*model.FrameworkProfile, error) {
			return nil, domerr.ErrNoProfileMatch
		},
		func(_ context.Context, _ fs.FS) ([]string, error) { return nil, nil },
		func(_ context.Context, _ fs.FS, _ string) (model.JavaFileSummary, error) {
			return model.JavaFileSummary{}, nil
		},
		func(_ context.Context, _ *model.ProjectState) error { return nil },
	)

	_, err := uc.Execute(context.Background(), scanvo.ScanProjectInput{WorkDir: t.TempDir()})
	require.True(t, errors.Is(err, domerr.ErrNoProfileMatch))
}

func TestScanUC_PartialParseSkipped(t *testing.T) {
	t.Parallel()

	uc := buildMinimalUseCase(
		func(_ context.Context, _ fs.FS) (*model.FrameworkProfile, error) { return minimalProfile(), nil },
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{"Broken.java"}, nil
		},
		func(_ context.Context, _ fs.FS, _ string) (model.JavaFileSummary, error) {
			return model.JavaFileSummary{HasErrors: true}, domerr.ErrPartialParse
		},
		func(_ context.Context, _ *model.ProjectState) error { return nil },
	)

	out, err := uc.Execute(context.Background(), scanvo.ScanProjectInput{WorkDir: t.TempDir()})
	require.NoError(t, err)
	require.Len(t, out.SkippedFiles, 1)
	require.Contains(t, out.SkippedFiles[0], "Broken.java")
}

func TestScanUC_Cancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	uc := buildMinimalUseCase(
		func(_ context.Context, _ fs.FS) (*model.FrameworkProfile, error) { return minimalProfile(), nil },
		func(_ context.Context, _ fs.FS) ([]string, error) { return nil, nil },
		func(_ context.Context, _ fs.FS, _ string) (model.JavaFileSummary, error) {
			return model.JavaFileSummary{}, nil
		},
		func(_ context.Context, _ *model.ProjectState) error { return nil },
	)

	_, err := uc.Execute(ctx, scanvo.ScanProjectInput{WorkDir: t.TempDir()})
	require.True(t, errors.Is(err, context.Canceled))
}

func TestScanUC_ProfileNameMismatch(t *testing.T) {
	t.Parallel()

	uc := buildMinimalUseCase(
		func(_ context.Context, _ fs.FS) (*model.FrameworkProfile, error) { return minimalProfile(), nil },
		func(_ context.Context, _ fs.FS) ([]string, error) { return nil, nil },
		func(_ context.Context, _ fs.FS, _ string) (model.JavaFileSummary, error) {
			return model.JavaFileSummary{}, nil
		},
		func(_ context.Context, _ *model.ProjectState) error { return nil },
	)

	_, err := uc.Execute(context.Background(), scanvo.ScanProjectInput{
		WorkDir:     t.TempDir(),
		ProfileName: "other-profile",
	})
	require.True(t, errors.Is(err, domerr.ErrNoProfileMatch))
}
