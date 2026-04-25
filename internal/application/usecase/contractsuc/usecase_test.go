package contractsuc_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/application/usecase/contractsuc"
	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
	contractsvo "github.com/jitctx/jitctx/internal/domain/vo/contracts"
)

// ─── Fakes ───────────────────────────────────────────────────────────────────

type fakeFinder struct {
	find func(ctx context.Context, feature, baseDir, plansDir string) (string, []byte, []string, error)
}

func (f fakeFinder) Find(ctx context.Context, feature, baseDir, plansDir string) (string, []byte, []string, error) {
	return f.find(ctx, feature, baseDir, plansDir)
}

type fakeParser struct {
	parse func(ctx context.Context, content string) (model.FeatureSpec, []domerr.SpecParseWarning, error)
}

func (f fakeParser) ParseSpec(ctx context.Context, content string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
	return f.parse(ctx, content)
}

type fakeManifest struct {
	load func(ctx context.Context) (*model.ProjectState, error)
}

func (f fakeManifest) Load(ctx context.Context) (*model.ProjectState, error) {
	return f.load(ctx)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func nopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func bufLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func newUC(finder fakeFinder, parser fakeParser, manifest fakeManifest, logger *slog.Logger) *contractsuc.Impl {
	return contractsuc.New(
		finder,
		parser,
		service.NewContractPathMapper(),
		service.NewContractRoleDescriber(),
		service.NewContractTargetResolver(),
		manifest,
		logger,
	)
}

// noopFinder returns a finder that panics if invoked — used to assert a port is never called.
func mustNotCallFinder(t *testing.T) fakeFinder {
	t.Helper()
	return fakeFinder{find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
		t.Errorf("finder.Find must not be called")
		return "", nil, nil, nil
	}}
}

// mustNotCallParser returns a parser that fails the test if invoked.
func mustNotCallParser(t *testing.T) fakeParser {
	t.Helper()
	return fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		t.Errorf("parser.ParseSpec must not be called")
		return model.FeatureSpec{}, nil, nil
	}}
}

// happyFinder returns a finder that always succeeds with minimal non-empty content.
func happyFinder(alts []string) fakeFinder {
	return fakeFinder{find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
		return "/x", []byte("ignored"), alts, nil
	}}
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestContractsUseCase_HappyPath_Spec_Service(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature: "create-user",
		Module:  "user-management",
		Contracts: []model.SpecContract{
			{Name: "CreateUserUseCase", Type: model.ContractInputPort},
			{Name: "UserRepository", Type: model.ContractOutputPort},
			{
				Name:       "UserServiceImpl",
				Type:       model.ContractService,
				Implements: "CreateUserUseCase",
				DependsOn:  []string{"UserRepository"},
			},
		},
	}

	finder := happyFinder(nil)
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return spec, nil, nil
	}}
	manifest := fakeManifest{load: func(_ context.Context) (*model.ProjectState, error) {
		t.Errorf("manifest.Load must not be called when spec search succeeds")
		return nil, nil
	}}

	uc := newUC(finder, parser, manifest, nopLogger())
	out, err := uc.Execute(context.Background(), contractsvo.ExtractContractsInput{
		TargetFile: "src/main/java/com/app/UserServiceImpl.java",
		Feature:    "create-user",
	})

	require.NoError(t, err)
	require.Equal(t, "spec", out.Source)
	require.Equal(t, "UserServiceImpl", out.Target.Name)
	require.Equal(t, "application/UserServiceImpl.java", out.Target.Path)
	require.Len(t, out.Related, 2)
	// Related must be alphabetically sorted.
	require.Equal(t, "CreateUserUseCase", out.Related[0].Name)
	require.Equal(t, "UserRepository", out.Related[1].Name)
}

func TestContractsUseCase_HappyPath_Spec_RestAdapter(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature: "create-user",
		Module:  "user-management",
		Contracts: []model.SpecContract{
			{Name: "CreateUserUseCase", Type: model.ContractInputPort},
			{
				Name: "UserController",
				Type: model.ContractRestAdapter,
				Uses: []string{"CreateUserUseCase"},
			},
			{
				Name:       "UserServiceImpl",
				Type:       model.ContractService,
				Implements: "CreateUserUseCase",
			},
		},
	}

	finder := happyFinder(nil)
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return spec, nil, nil
	}}
	manifest := fakeManifest{load: func(_ context.Context) (*model.ProjectState, error) {
		t.Errorf("manifest.Load must not be called when spec search succeeds")
		return nil, nil
	}}

	uc := newUC(finder, parser, manifest, nopLogger())
	out, err := uc.Execute(context.Background(), contractsvo.ExtractContractsInput{
		TargetFile: "UserController.java",
		Feature:    "create-user",
	})

	require.NoError(t, err)
	require.Equal(t, "spec", out.Source)
	require.Equal(t, "UserController", out.Target.Name)
	// Only CreateUserUseCase is in Uses; UserServiceImpl is unrelated to UserController.
	require.Len(t, out.Related, 1)
	require.Equal(t, "CreateUserUseCase", out.Related[0].Name)
}

func TestContractsUseCase_HappyPath_Spec_InputPortStandalone(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature: "create-user",
		Module:  "user-management",
		Contracts: []model.SpecContract{
			{Name: "BasicInput", Type: model.ContractInputPort},
		},
	}

	finder := happyFinder(nil)
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return spec, nil, nil
	}}
	manifest := fakeManifest{load: func(_ context.Context) (*model.ProjectState, error) {
		t.Errorf("manifest.Load must not be called when spec search succeeds")
		return nil, nil
	}}

	uc := newUC(finder, parser, manifest, nopLogger())
	out, err := uc.Execute(context.Background(), contractsvo.ExtractContractsInput{
		TargetFile: "BasicInput.java",
		Feature:    "create-user",
	})

	require.NoError(t, err)
	require.Equal(t, "spec", out.Source)
	require.Equal(t, "BasicInput", out.Target.Name)
	require.Empty(t, out.Related)
}

func TestContractsUseCase_ManifestFallback_WhenNotInSpec(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature: "create-user",
		Module:  "user-management",
		Contracts: []model.SpecContract{
			{Name: "SomeOtherContract", Type: model.ContractInputPort},
		},
	}

	state := &model.ProjectState{
		Modules: []model.Module{
			{
				ID:   "user-management",
				Path: "src/main/java/com/app",
				Contracts: []model.Contract{
					{
						Name: "Whatever",
						Type: model.ContractService,
						Path: "application/Whatever.java",
						Methods: []model.Method{
							{Signature: "void doIt()"},
						},
					},
				},
			},
		},
	}

	finder := happyFinder(nil)
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return spec, nil, nil
	}}
	manifestStore := fakeManifest{load: func(_ context.Context) (*model.ProjectState, error) {
		return state, nil
	}}

	uc := newUC(finder, parser, manifestStore, nopLogger())
	out, err := uc.Execute(context.Background(), contractsvo.ExtractContractsInput{
		TargetFile: "Whatever.java",
		Feature:    "create-user",
	})

	require.NoError(t, err)
	require.Equal(t, "manifest", out.Source)
	require.Equal(t, "Whatever", out.Target.Name)
	require.Equal(t, "application/Whatever.java", out.Target.Path)
	require.Equal(t, []string{"void doIt()"}, out.Target.Methods)
	require.Empty(t, out.Related)
}

func TestContractsUseCase_ManifestFallback_WhenManifestOnlyMode(t *testing.T) {
	t.Parallel()

	state := &model.ProjectState{
		Modules: []model.Module{
			{
				ID:   "user-management",
				Path: "src/main/java",
				Contracts: []model.Contract{
					{
						Name: "Whatever",
						Type: model.ContractService,
						Path: "application/Whatever.java",
					},
				},
			},
		},
	}

	finder := mustNotCallFinder(t)
	parser := mustNotCallParser(t)
	manifestStore := fakeManifest{load: func(_ context.Context) (*model.ProjectState, error) {
		return state, nil
	}}

	uc := newUC(finder, parser, manifestStore, nopLogger())
	// Feature="" and FilePath="" → manifest-only mode.
	out, err := uc.Execute(context.Background(), contractsvo.ExtractContractsInput{
		TargetFile: "Whatever.java",
	})

	require.NoError(t, err)
	require.Equal(t, "manifest", out.Source)
	require.Equal(t, "Whatever", out.Target.Name)
}

func TestContractsUseCase_BothMiss_ReturnsTypedNotFound(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature: "create-user",
		Module:  "user-management",
		Contracts: []model.SpecContract{
			{Name: "SomeOtherContract", Type: model.ContractInputPort},
		},
	}
	state := &model.ProjectState{
		Modules: []model.Module{
			{ID: "user-management", Contracts: []model.Contract{}},
		},
	}

	finder := happyFinder(nil)
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return spec, nil, nil
	}}
	manifestStore := fakeManifest{load: func(_ context.Context) (*model.ProjectState, error) {
		return state, nil
	}}

	uc := newUC(finder, parser, manifestStore, nopLogger())
	_, err := uc.Execute(context.Background(), contractsvo.ExtractContractsInput{
		TargetFile: "Unknown.java",
		Feature:    "create-user",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrContractTargetNotFound))

	var typedErr *domerr.ContractTargetNotFoundError
	require.True(t, errors.As(err, &typedErr))
	require.True(t, typedErr.SearchedSpec)
	require.True(t, typedErr.SearchedManifest)
}

func TestContractsUseCase_BothMiss_ManifestNotFound_StillTypedNotFound(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature:   "create-user",
		Module:    "user-management",
		Contracts: []model.SpecContract{},
	}

	finder := happyFinder(nil)
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return spec, nil, nil
	}}
	manifestStore := fakeManifest{load: func(_ context.Context) (*model.ProjectState, error) {
		return nil, domerr.ErrManifestNotFound
	}}

	uc := newUC(finder, parser, manifestStore, nopLogger())
	_, err := uc.Execute(context.Background(), contractsvo.ExtractContractsInput{
		TargetFile: "Unknown.java",
		Feature:    "create-user",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrContractTargetNotFound))

	var typedErr *domerr.ContractTargetNotFoundError
	require.True(t, errors.As(err, &typedErr))
	require.True(t, typedErr.SearchedSpec)
	require.False(t, typedErr.SearchedManifest)
}

func TestContractsUseCase_Validation_EmptyTargetFile(t *testing.T) {
	t.Parallel()

	finder := mustNotCallFinder(t)
	parser := mustNotCallParser(t)
	manifestStore := fakeManifest{load: func(_ context.Context) (*model.ProjectState, error) {
		t.Errorf("manifest.Load must not be called on validation error")
		return nil, nil
	}}

	uc := newUC(finder, parser, manifestStore, nopLogger())
	_, err := uc.Execute(context.Background(), contractsvo.ExtractContractsInput{
		TargetFile: "",
		Feature:    "create-user",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "target file path must not be empty")
}

func TestContractsUseCase_Validation_FeatureAndFile(t *testing.T) {
	t.Parallel()

	finder := mustNotCallFinder(t)
	parser := mustNotCallParser(t)
	manifestStore := fakeManifest{load: func(_ context.Context) (*model.ProjectState, error) {
		t.Errorf("manifest.Load must not be called on validation error")
		return nil, nil
	}}

	uc := newUC(finder, parser, manifestStore, nopLogger())
	_, err := uc.Execute(context.Background(), contractsvo.ExtractContractsInput{
		TargetFile: "Foo.java",
		Feature:    "some-feature",
		FilePath:   "/some/path.md",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "mutually exclusive")
}

func TestContractsUseCase_ParserError_PassThrough(t *testing.T) {
	t.Parallel()

	parseErr := &domerr.SpecParseError{Line: 5, Message: "oops"}
	finder := happyFinder(nil)
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return model.FeatureSpec{}, nil, parseErr
	}}
	manifestStore := fakeManifest{load: func(_ context.Context) (*model.ProjectState, error) {
		t.Errorf("manifest.Load must not be called when parser fails")
		return nil, nil
	}}

	uc := newUC(finder, parser, manifestStore, nopLogger())
	_, err := uc.Execute(context.Background(), contractsvo.ExtractContractsInput{
		TargetFile: "Foo.java",
		Feature:    "create-user",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrSpecParse))
	require.Contains(t, err.Error(), "parse spec")
}

func TestContractsUseCase_ExternalRefWarning(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature: "create-user",
		Module:  "user-management",
		Contracts: []model.SpecContract{
			{
				Name: "MyService",
				Type: model.ContractService,
				Uses: []string{"Unknown"},
			},
		},
	}

	finder := happyFinder(nil)
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return spec, nil, nil
	}}
	manifestStore := fakeManifest{load: func(_ context.Context) (*model.ProjectState, error) {
		t.Errorf("manifest.Load must not be called when spec search succeeds")
		return nil, nil
	}}

	var buf bytes.Buffer
	uc := newUC(finder, parser, manifestStore, bufLogger(&buf))
	out, err := uc.Execute(context.Background(), contractsvo.ExtractContractsInput{
		TargetFile: "MyService.java",
		Feature:    "create-user",
	})

	require.NoError(t, err)
	require.Equal(t, "MyService", out.Target.Name)
	require.Empty(t, out.Related)

	logOutput := buf.String()
	require.Contains(t, logOutput, "external reference")
	require.Contains(t, logOutput, "'Unknown'")
}

func TestContractsUseCase_AltsLogged(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature: "create-user",
		Module:  "user-management",
		Contracts: []model.SpecContract{
			{Name: "SomeContract", Type: model.ContractInputPort},
		},
	}

	finder := happyFinder([]string{"other.md"})
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return spec, nil, nil
	}}
	manifestStore := fakeManifest{load: func(_ context.Context) (*model.ProjectState, error) {
		t.Errorf("manifest.Load must not be called when spec search succeeds")
		return nil, nil
	}}

	var buf bytes.Buffer
	uc := newUC(finder, parser, manifestStore, bufLogger(&buf))
	_, err := uc.Execute(context.Background(), contractsvo.ExtractContractsInput{
		TargetFile: "SomeContract.java",
		Feature:    "create-user",
	})

	require.NoError(t, err)

	logOutput := buf.String()
	require.Contains(t, logOutput, "spec found in additional location")
	require.Contains(t, logOutput, "other.md")
}

func TestContractsUseCase_ContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	finder := mustNotCallFinder(t)
	parser := mustNotCallParser(t)
	manifestStore := fakeManifest{load: func(_ context.Context) (*model.ProjectState, error) {
		t.Errorf("manifest.Load must not be called on cancelled context")
		return nil, nil
	}}

	uc := newUC(finder, parser, manifestStore, nopLogger())
	_, err := uc.Execute(ctx, contractsvo.ExtractContractsInput{
		TargetFile: "Foo.java",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled))
}
