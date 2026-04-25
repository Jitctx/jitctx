package planuc_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/application/usecase/planuc"
	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
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

// ─── Helpers ─────────────────────────────────────────────────────────────────

func nopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func bufLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// fourContractSpec mirrors the .feature table at lines 117-121.
// Layer 0: CreateUserUseCase (input-port), UserRepository (output-port)
// Layer 1: UserServiceImpl (service, implements CreateUserUseCase, dependsOn UserRepository),
//
//	UserController (rest-adapter, uses CreateUserUseCase)
func fourContractSpec() model.FeatureSpec {
	return model.FeatureSpec{
		Feature: "f",
		Module:  "m",
		Contracts: []model.SpecContract{
			{
				Name: "CreateUserUseCase",
				Type: model.ContractInputPort,
			},
			{
				Name: "UserRepository",
				Type: model.ContractOutputPort,
			},
			{
				Name:       "UserServiceImpl",
				Type:       model.ContractService,
				Implements: "CreateUserUseCase",
				DependsOn:  []string{"UserRepository"},
			},
			{
				Name: "UserController",
				Type: model.ContractRestAdapter,
				Uses: []string{"CreateUserUseCase"},
			},
		},
	}
}

func newUC(finder fakeFinder, parser fakeParser, logger *slog.Logger) *planuc.Impl {
	return planuc.New(finder, parser, service.NewDependencyLayerer(), service.NewContractPathMapper(), logger)
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestPlanUseCase_executesEndToEnd_withFeature(t *testing.T) {
	t.Parallel()

	finder := fakeFinder{find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
		return "/some/path", []byte("ignored"), nil, nil
	}}
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return fourContractSpec(), nil, nil
	}}

	uc := newUC(finder, parser, nopLogger())
	out, err := uc.Execute(context.Background(), planvo.LayersInput{Feature: "create-user"})

	require.NoError(t, err)
	require.Equal(t, "f", out.Feature)
	require.Equal(t, "m", out.Module)
	require.Len(t, out.Layers, 2)
	require.Empty(t, out.Externals)

	layer0 := out.Layers[0]
	require.Len(t, layer0.Targets, 2)
	require.Equal(t, "CreateUserUseCase", layer0.Targets[0].Name)
	require.Equal(t, "UserRepository", layer0.Targets[1].Name)

	layer1 := out.Layers[1]
	require.Len(t, layer1.Targets, 2)
	require.Equal(t, "UserController", layer1.Targets[0].Name)
	require.Equal(t, "UserServiceImpl", layer1.Targets[1].Name)
}

func TestPlanUseCase_executesEndToEnd_withFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.md")
	require.NoError(t, os.WriteFile(specPath, []byte("# Feature: f\nModule: m\n"), 0o600))

	finder := fakeFinder{find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
		t.Errorf("finder must not be called when FilePath is set")
		return "", nil, nil, nil
	}}
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return fourContractSpec(), nil, nil
	}}

	uc := newUC(finder, parser, nopLogger())
	out, err := uc.Execute(context.Background(), planvo.LayersInput{FilePath: specPath})

	require.NoError(t, err)
	require.Equal(t, "f", out.Feature)
}

func TestPlanUseCase_mutuallyExclusive(t *testing.T) {
	t.Parallel()

	finder := fakeFinder{find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
		return "", nil, nil, nil
	}}
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return model.FeatureSpec{}, nil, nil
	}}

	uc := newUC(finder, parser, nopLogger())
	_, err := uc.Execute(context.Background(), planvo.LayersInput{
		Feature:  "some-feature",
		FilePath: "/some/path.md",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "mutually exclusive")
}

func TestPlanUseCase_noneSet(t *testing.T) {
	t.Parallel()

	finder := fakeFinder{find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
		return "", nil, nil, nil
	}}
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return model.FeatureSpec{}, nil, nil
	}}

	uc := newUC(finder, parser, nopLogger())
	_, err := uc.Execute(context.Background(), planvo.LayersInput{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "either --feature or --file is required")
}

func TestPlanUseCase_propagatesParserError(t *testing.T) {
	t.Parallel()

	finder := fakeFinder{find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
		return "/path/f.md", []byte("content"), nil, nil
	}}
	parseErr := &domerr.SpecParseError{Line: 5, Message: "oops"}
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return model.FeatureSpec{}, nil, parseErr
	}}

	uc := newUC(finder, parser, nopLogger())
	_, err := uc.Execute(context.Background(), planvo.LayersInput{Feature: "f"})

	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrSpecParse))
}

func TestPlanUseCase_propagatesCycleError(t *testing.T) {
	t.Parallel()

	// A depends on B, B depends on A — cycle.
	cycleSpec := model.FeatureSpec{
		Feature: "broken",
		Module:  "m",
		Contracts: []model.SpecContract{
			{Name: "A", Type: model.ContractInputPort, DependsOn: []string{"B"}},
			{Name: "B", Type: model.ContractOutputPort, DependsOn: []string{"A"}},
		},
	}
	finder := fakeFinder{find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
		return "/path/broken.md", []byte("content"), nil, nil
	}}
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return cycleSpec, nil, nil
	}}

	uc := newUC(finder, parser, nopLogger())
	_, err := uc.Execute(context.Background(), planvo.LayersInput{Feature: "broken"})

	require.Error(t, err)
	var cycleErr *service.CycleError
	require.True(t, errors.As(err, &cycleErr))
}

func TestPlanUseCase_unsupportedContractType(t *testing.T) {
	t.Parallel()

	weirdSpec := model.FeatureSpec{
		Feature: "weird",
		Module:  "m",
		Contracts: []model.SpecContract{
			{Name: "WhateverThing", Type: model.ContractType("weird")},
		},
	}
	finder := fakeFinder{find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
		return "/path/weird.md", []byte("content"), nil, nil
	}}
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return weirdSpec, nil, nil
	}}

	uc := newUC(finder, parser, nopLogger())
	_, err := uc.Execute(context.Background(), planvo.LayersInput{Feature: "weird"})

	require.Error(t, err)
	var uct *domerr.UnsupportedContractTypeError
	require.True(t, errors.As(err, &uct))
}

func TestPlanUseCase_externalsLogged(t *testing.T) {
	t.Parallel()

	extSpec := model.FeatureSpec{
		Feature: "ext",
		Module:  "m",
		Contracts: []model.SpecContract{
			{Name: "MyService", Type: model.ContractService, Uses: []string{"ExternalThing"}},
		},
	}
	finder := fakeFinder{find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
		return "/path/ext.md", []byte("content"), nil, nil
	}}
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return extSpec, nil, nil
	}}

	var buf bytes.Buffer
	uc := newUC(finder, parser, bufLogger(&buf))
	_, err := uc.Execute(context.Background(), planvo.LayersInput{Feature: "ext"})
	require.NoError(t, err)

	logOutput := buf.String()
	require.Contains(t, logOutput, "external reference")
	require.Contains(t, logOutput, "'ExternalThing'")
}

func TestPlanUseCase_altsLogged(t *testing.T) {
	t.Parallel()

	finder := fakeFinder{find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
		return "/primary/f.md", []byte("content"), []string{"/alt/x.md"}, nil
	}}
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return fourContractSpec(), nil, nil
	}}

	var buf bytes.Buffer
	uc := newUC(finder, parser, bufLogger(&buf))
	_, err := uc.Execute(context.Background(), planvo.LayersInput{Feature: "f"})
	require.NoError(t, err)

	logOutput := buf.String()
	require.Contains(t, logOutput, "additional")
	require.Contains(t, logOutput, "/alt/x.md")
}

func TestPlanUseCase_findSpecNotFound_passesThrough(t *testing.T) {
	t.Parallel()

	notFoundErr := &domerr.SpecFileNotFoundError{
		Feature:  "f",
		Searched: []string{"a", "b"},
	}
	finder := fakeFinder{find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
		return "", nil, nil, notFoundErr
	}}
	parser := fakeParser{parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
		return model.FeatureSpec{}, nil, nil
	}}

	uc := newUC(finder, parser, nopLogger())
	_, err := uc.Execute(context.Background(), planvo.LayersInput{Feature: "f"})

	require.Error(t, err)
	var sfnf *domerr.SpecFileNotFoundError
	require.True(t, errors.As(err, &sfnf))
}
