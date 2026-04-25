package scaffolduc_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/application/usecase/scaffolduc"
	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

// ─── Fakes ───────────────────────────────────────────────────────────────────

type fakeFinder struct {
	find func(ctx context.Context, feature, baseDir, plansDir string) (string, []byte, []string, error)
	// callCount is incremented on each Find call.
	callCount int
}

func (f *fakeFinder) Find(ctx context.Context, feature, baseDir, plansDir string) (string, []byte, []string, error) {
	f.callCount++
	return f.find(ctx, feature, baseDir, plansDir)
}

type fakeParser struct {
	parse func(ctx context.Context, content string) (model.FeatureSpec, []domerr.SpecParseWarning, error)
}

func (f *fakeParser) ParseSpec(ctx context.Context, content string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
	return f.parse(ctx, content)
}

type fakeRenderer struct {
	render    func(ctx context.Context, input scaffoldvo.RenderInput) ([]byte, error)
	callCount int
	// captured stores every RenderInput received, in order.
	captured []scaffoldvo.RenderInput
}

func (f *fakeRenderer) Render(ctx context.Context, input scaffoldvo.RenderInput) ([]byte, error) {
	f.callCount++
	f.captured = append(f.captured, input)
	return f.render(ctx, input)
}

type fakeWriter struct {
	writeAll  func(ctx context.Context, files []scaffoldvo.ProductionFile) ([]string, error)
	callCount int
	// lastFiles is the batch passed to WriteAll on the most recent call.
	lastFiles []scaffoldvo.ProductionFile
}

func (f *fakeWriter) WriteAll(ctx context.Context, files []scaffoldvo.ProductionFile) ([]string, error) {
	f.callCount++
	f.lastFiles = files
	return f.writeAll(ctx, files)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func nopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// canonicalSpec is a 4-contract spec with Package set, mirroring the plan fixture.
func canonicalSpec() model.FeatureSpec {
	return model.FeatureSpec{
		Feature: "create-user",
		Module:  "user-management",
		Package: "com.app.user",
		Contracts: []model.SpecContract{
			{
				Name:    "CreateUserUseCase",
				Type:    model.ContractInputPort,
				Methods: []string{"UserResponse execute(CreateUserCommand cmd)"},
			},
			{
				Name: "UserRepository",
				Type: model.ContractOutputPort,
				Methods: []string{
					"Optional<User> findByEmail(String email)",
					"User save(User user)",
				},
			},
			{
				Name:       "UserServiceImpl",
				Type:       model.ContractService,
				Implements: "CreateUserUseCase",
				DependsOn:  []string{"UserRepository"},
			},
			{
				Name:      "UserController",
				Type:      model.ContractRestAdapter,
				Uses:      []string{"CreateUserUseCase"},
				Endpoints: []string{"POST /users"},
			},
		},
	}
}

// finderReturnsPath is a simple fakeFinder that returns a fixed path+content.
func finderReturnsPath() *fakeFinder {
	return &fakeFinder{
		find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
			return "/x", []byte("ignored"), nil, nil
		},
	}
}

// parserReturns builds a fakeParser that returns the given spec.
func parserReturns(spec model.FeatureSpec) *fakeParser {
	return &fakeParser{
		parse: func(_ context.Context, _ string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
			return spec, nil, nil
		},
	}
}

// rendererAlwaysSucceeds returns a fakeRenderer that produces "// rendered <ClassName>".
func rendererAlwaysSucceeds() *fakeRenderer {
	return &fakeRenderer{
		render: func(_ context.Context, in scaffoldvo.RenderInput) ([]byte, error) {
			return []byte("// rendered " + in.ClassName), nil
		},
	}
}

// writerReturnsPaths returns a fakeWriter that echoes back the paths it was given.
func writerReturnsPaths() *fakeWriter {
	return &fakeWriter{
		writeAll: func(_ context.Context, files []scaffoldvo.ProductionFile) ([]string, error) {
			paths := make([]string, len(files))
			for i, f := range files {
				paths[i] = f.Path
			}
			return paths, nil
		},
	}
}

// newUC builds a scaffolduc.Impl with real domain services.
func newUC(
	finder *fakeFinder,
	parser *fakeParser,
	renderer *fakeRenderer,
	writer *fakeWriter,
	logger *slog.Logger,
) *scaffolduc.Impl {
	mapper := service.NewContractPathMapper()
	importResolver := service.NewJavaImportResolver(mapper)
	endpointSynth := service.NewEndpointSynthesizer()
	idUtils := service.NewJavaIdentifierUtils()
	return scaffolduc.New(finder, parser, mapper, importResolver, endpointSynth, idUtils, renderer, writer, logger)
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestScaffoldUseCase_HappyPath_FourContracts(t *testing.T) {
	t.Parallel()

	finder := finderReturnsPath()
	parser := parserReturns(canonicalSpec())
	renderer := rendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finder, parser, renderer, writer, nopLogger())
	out, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})

	require.NoError(t, err)
	require.Equal(t, "create-user", out.Feature)
	require.Equal(t, "user-management", out.Module)
	require.Equal(t, "com.app.user", out.Package)
	require.Len(t, out.WrittenPaths, 4)

	// Renderer must have been called once per contract.
	require.Equal(t, 4, renderer.callCount)

	// Writer must have been called exactly once with all 4 files.
	require.Equal(t, 1, writer.callCount)
	require.Len(t, writer.lastFiles, 4)

	// Assert exact generated paths for each contract.
	wantPaths := []string{
		"/work/src/main/java/com/app/user/port/in/CreateUserUseCase.java",
		"/work/src/main/java/com/app/user/port/out/UserRepository.java",
		"/work/src/main/java/com/app/user/application/UserServiceImpl.java",
		"/work/src/main/java/com/app/user/adapter/in/web/UserController.java",
	}
	// WrittenPaths is what the writer returned (which echoes the paths we sent).
	// The batch order matches declaration order in the spec.
	gotPaths := make([]string, len(writer.lastFiles))
	for i, f := range writer.lastFiles {
		gotPaths[i] = f.Path
	}
	require.Equal(t, wantPaths, gotPaths)
}

func TestScaffoldUseCase_MissingPackage_ErrSpecMissingPackage(t *testing.T) {
	t.Parallel()

	spec := canonicalSpec()
	spec.Package = ""

	renderer := rendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(spec), renderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})

	require.ErrorIs(t, err, domerr.ErrSpecMissingPackage)
	// Renderer and writer must never be called.
	require.Equal(t, 0, renderer.callCount)
	require.Equal(t, 0, writer.callCount)
}

func TestScaffoldUseCase_UnsupportedContractType_ShortCircuits(t *testing.T) {
	t.Parallel()

	weirdSpec := model.FeatureSpec{
		Feature: "weird",
		Module:  "m",
		Package: "com.app.weird",
		Contracts: []model.SpecContract{
			{Name: "WhateverThing", Type: model.ContractType("weird-thing")},
		},
	}

	renderer := rendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(weirdSpec), renderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "weird",
		BaseDir: "/work",
	})

	require.Error(t, err)
	var uct *domerr.UnsupportedContractTypeError
	require.ErrorAs(t, err, &uct)

	// Renderer and writer must not have been called.
	require.Equal(t, 0, renderer.callCount)
	require.Equal(t, 0, writer.callCount)
}

func TestScaffoldUseCase_ConflictPropagated(t *testing.T) {
	t.Parallel()

	conflictErr := &domerr.ScaffoldConflictError{Conflicts: []string{"/x.java"}}
	writer := &fakeWriter{
		writeAll: func(_ context.Context, _ []scaffoldvo.ProductionFile) ([]string, error) {
			return nil, conflictErr
		},
	}
	renderer := rendererAlwaysSucceeds()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})

	require.Error(t, err)
	var sce *domerr.ScaffoldConflictError
	require.ErrorAs(t, err, &sce)
	require.Equal(t, []string{"/x.java"}, sce.Conflicts)
}

func TestScaffoldUseCase_RenderFailurePropagated(t *testing.T) {
	t.Parallel()

	renderBoom := errors.New("template boom")
	renderer := &fakeRenderer{
		render: func(_ context.Context, in scaffoldvo.RenderInput) ([]byte, error) {
			// Fail only on the first call (first contract).
			return nil, renderBoom
		},
	}
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})

	require.Error(t, err)
	var sre *domerr.ScaffoldRenderError
	require.ErrorAs(t, err, &sre)
	// First contract in canonicalSpec is CreateUserUseCase.
	require.Equal(t, "CreateUserUseCase", sre.Contract)
	require.Contains(t, sre.Cause.Error(), "template boom")

	// Writer must not have been called.
	require.Equal(t, 0, writer.callCount)
}

func TestScaffoldUseCase_ImportsInRenderInput(t *testing.T) {
	t.Parallel()

	renderer := rendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})
	require.NoError(t, err)

	// Find the captured RenderInput for UserController (index 3 in canonicalSpec).
	require.Len(t, renderer.captured, 4)
	controllerInput := renderer.captured[3]
	require.Equal(t, "UserController", controllerInput.ClassName)

	// UserController uses CreateUserUseCase → must appear in imports.
	require.Contains(t, controllerInput.Imports, "com.app.user.port.in.CreateUserUseCase")

	// Find the captured RenderInput for UserServiceImpl (index 2 in canonicalSpec).
	serviceInput := renderer.captured[2]
	require.Equal(t, "UserServiceImpl", serviceInput.ClassName)

	// UserServiceImpl dependsOn UserRepository → must appear in imports.
	require.Contains(t, serviceInput.Imports, "com.app.user.port.out.UserRepository")
}

func TestScaffoldUseCase_OverrideMethodsForServiceFromImplementsTarget(t *testing.T) {
	t.Parallel()

	renderer := rendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})
	require.NoError(t, err)

	require.Len(t, renderer.captured, 4)
	// UserServiceImpl is captured at index 2.
	serviceInput := renderer.captured[2]
	require.Equal(t, "UserServiceImpl", serviceInput.ClassName)

	// Methods should come from CreateUserUseCase (its Implements target).
	// CreateUserUseCase has one method: "UserResponse execute(CreateUserCommand cmd)".
	require.Len(t, serviceInput.Methods, 1)
	m := serviceInput.Methods[0]
	require.Equal(t, "UserResponse execute(CreateUserCommand cmd)", m.Signature)
	require.True(t, m.Override)
	require.Contains(t, m.Body, "throw new UnsupportedOperationException")
}

func TestScaffoldUseCase_EndpointsForRestAdapter(t *testing.T) {
	t.Parallel()

	renderer := rendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})
	require.NoError(t, err)

	require.Len(t, renderer.captured, 4)
	// UserController is captured at index 3.
	controllerInput := renderer.captured[3]
	require.Equal(t, "UserController", controllerInput.ClassName)
	require.Len(t, controllerInput.Endpoints, 1)

	ep := controllerInput.Endpoints[0]
	require.Equal(t, `@PostMapping("/users")`, ep.Annotation)
	require.Equal(t, "postUsers", ep.Method)
}

func TestScaffoldUseCase_DependenciesGenerated(t *testing.T) {
	t.Parallel()

	renderer := rendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})
	require.NoError(t, err)

	require.Len(t, renderer.captured, 4)
	// UserServiceImpl is at index 2.
	serviceInput := renderer.captured[2]
	require.Equal(t, "UserServiceImpl", serviceInput.ClassName)

	// DependsOn: ["UserRepository"] → one ConstructorDep.
	require.Len(t, serviceInput.Dependencies, 1)
	dep := serviceInput.Dependencies[0]
	require.Equal(t, "UserRepository", dep.Type)
	require.Equal(t, "userRepository", dep.FieldName)
}

func TestScaffoldUseCase_MutuallyExclusive_FeatureAndFile(t *testing.T) {
	t.Parallel()

	finder := &fakeFinder{
		find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
			t.Errorf("finder must not be called")
			return "", nil, nil, nil
		},
	}
	renderer := rendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finder, parserReturns(canonicalSpec()), renderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature:  "create-user",
		FilePath: "/some/path.md",
		BaseDir:  "/work",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "mutually exclusive")
	require.Equal(t, 0, finder.callCount)
}

func TestScaffoldUseCase_Validation_NoneSet(t *testing.T) {
	t.Parallel()

	finder := &fakeFinder{
		find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
			t.Errorf("finder must not be called")
			return "", nil, nil, nil
		},
	}
	renderer := rendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finder, parserReturns(canonicalSpec()), renderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		BaseDir: "/work",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "either --feature or --file is required")
	require.Equal(t, 0, finder.callCount)
}

func TestScaffoldUseCase_WithFile_BypassesFinder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.md")
	require.NoError(t, os.WriteFile(specPath, []byte("# Feature: create-user\nModule: user-management\n"), 0o600))

	finder := &fakeFinder{
		find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
			t.Errorf("finder must not be called when FilePath is set")
			return "", nil, nil, nil
		},
	}
	renderer := rendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finder, parserReturns(canonicalSpec()), renderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		FilePath: specPath,
		BaseDir:  "/work",
	})

	require.NoError(t, err)
	// The finder must not have been called at all.
	require.Equal(t, 0, finder.callCount)
}

func TestScaffoldUseCase_CtxCancelled(t *testing.T) {
	t.Parallel()

	finder := &fakeFinder{
		find: func(_ context.Context, _, _, _ string) (string, []byte, []string, error) {
			t.Errorf("finder must not be called")
			return "", nil, nil, nil
		},
	}
	renderer := rendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	uc := newUC(finder, parserReturns(canonicalSpec()), renderer, writer, nopLogger())
	_, err := uc.Execute(ctx, scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})

	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 0, finder.callCount)
}
