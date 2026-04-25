package scaffolduc_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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

// fakeTestRenderer implements spec.RenderTestTemplatePort. It captures the
// last TestRenderInput it received and returns a canned byte slice.
type fakeTestRenderer struct {
	render    func(ctx context.Context, input scaffoldvo.TestRenderInput) ([]byte, error)
	callCount int
	// captured stores every TestRenderInput received, in order.
	captured []scaffoldvo.TestRenderInput
}

func (f *fakeTestRenderer) Render(ctx context.Context, input scaffoldvo.TestRenderInput) ([]byte, error) {
	f.callCount++
	f.captured = append(f.captured, input)
	if f.render != nil {
		return f.render(ctx, input)
	}
	return []byte("// test stub"), nil
}

// fakeWriter implements spec.WriteProductionFilesPort. It tracks call count and
// the last batch of ScaffoldFile values received.
type fakeWriter struct {
	writeAll  func(ctx context.Context, files []scaffoldvo.ScaffoldFile) ([]string, error)
	callCount int
	// lastFiles is the batch passed to WriteAll on the most recent call.
	lastFiles []scaffoldvo.ScaffoldFile
}

func (f *fakeWriter) WriteAll(ctx context.Context, files []scaffoldvo.ScaffoldFile) ([]string, error) {
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

// testRendererAlwaysSucceeds returns a fakeTestRenderer returning "// test stub".
func testRendererAlwaysSucceeds() *fakeTestRenderer {
	return &fakeTestRenderer{}
}

// writerReturnsPaths returns a fakeWriter that echoes back the paths it was given.
func writerReturnsPaths() *fakeWriter {
	return &fakeWriter{
		writeAll: func(_ context.Context, files []scaffoldvo.ScaffoldFile) ([]string, error) {
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
	testRenderer *fakeTestRenderer,
	writer *fakeWriter,
	logger *slog.Logger,
) *scaffolduc.Impl {
	mapper := service.NewContractPathMapper()
	testMapper := service.NewTestPathMapper()
	importResolver := service.NewJavaImportResolver(mapper)
	endpointSynth := service.NewEndpointSynthesizer()
	idUtils := service.NewJavaIdentifierUtils()
	methodParser := service.NewMethodSignatureParser()
	return scaffolduc.New(finder, parser, mapper, testMapper, importResolver, endpointSynth, idUtils, methodParser, renderer, testRenderer, writer, logger)
}

// ─── Existing Tests ───────────────────────────────────────────────────────────

func TestScaffoldUseCase_HappyPath_FourContracts(t *testing.T) {
	t.Parallel()

	finder := finderReturnsPath()
	parser := parserReturns(canonicalSpec())
	renderer := rendererAlwaysSucceeds()
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finder, parser, renderer, testRenderer, writer, nopLogger())
	out, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})

	require.NoError(t, err)
	require.Equal(t, "create-user", out.Feature)
	require.Equal(t, "user-management", out.Module)
	require.Equal(t, "com.app.user", out.Package)
	// 4 production + 2 testable (service + rest-adapter + entity/aggregate) contracts.
	// canonicalSpec has: input-port (no test), output-port (no test), service (test), rest-adapter (test).
	// So 4 production + 2 test = 6 total.
	require.Len(t, out.WrittenPaths, 6)

	// Renderer must have been called once per contract.
	require.Equal(t, 4, renderer.callCount)

	// Writer must have been called exactly once with all files.
	require.Equal(t, 1, writer.callCount)
	require.Len(t, writer.lastFiles, 6)

	// Assert exact generated production paths for each contract.
	wantProductionPaths := []string{
		"/work/src/main/java/com/app/user/port/in/CreateUserUseCase.java",
		"/work/src/main/java/com/app/user/port/out/UserRepository.java",
		"/work/src/main/java/com/app/user/application/UserServiceImpl.java",
		"/work/src/main/java/com/app/user/adapter/in/web/UserController.java",
	}
	gotProductionPaths := make([]string, 0, 4)
	for _, f := range writer.lastFiles {
		if f.Kind == scaffoldvo.KindProduction {
			gotProductionPaths = append(gotProductionPaths, f.Path)
		}
	}
	require.Equal(t, wantProductionPaths, gotProductionPaths)
}

func TestScaffoldUseCase_MissingPackage_ErrSpecMissingPackage(t *testing.T) {
	t.Parallel()

	spec := canonicalSpec()
	spec.Package = ""

	renderer := rendererAlwaysSucceeds()
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(spec), renderer, testRenderer, writer, nopLogger())
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
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(weirdSpec), renderer, testRenderer, writer, nopLogger())
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
		writeAll: func(_ context.Context, _ []scaffoldvo.ScaffoldFile) ([]string, error) {
			return nil, conflictErr
		},
	}
	renderer := rendererAlwaysSucceeds()
	testRenderer := testRendererAlwaysSucceeds()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, testRenderer, writer, nopLogger())
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
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, testRenderer, writer, nopLogger())
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
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, testRenderer, writer, nopLogger())
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
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, testRenderer, writer, nopLogger())
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
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, testRenderer, writer, nopLogger())
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
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, testRenderer, writer, nopLogger())
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
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finder, parserReturns(canonicalSpec()), renderer, testRenderer, writer, nopLogger())
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
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finder, parserReturns(canonicalSpec()), renderer, testRenderer, writer, nopLogger())
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
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finder, parserReturns(canonicalSpec()), renderer, testRenderer, writer, nopLogger())
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
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	uc := newUC(finder, parserReturns(canonicalSpec()), renderer, testRenderer, writer, nopLogger())
	_, err := uc.Execute(ctx, scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})

	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 0, finder.callCount)
}

// ─── New Test Scenarios (T6-G5) ───────────────────────────────────────────────

// TestScaffoldUseCase_ServiceContract_ProducesTestFile asserts that a service
// contract produces a ScaffoldFile{Kind: KindTest} whose Path ends with the
// expected test file path AND whose TestRenderInput.Mocks lists UserRepository.
func TestScaffoldUseCase_ServiceContract_ProducesTestFile(t *testing.T) {
	t.Parallel()

	renderer := rendererAlwaysSucceeds()
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, testRenderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})
	require.NoError(t, err)

	// Find the test file for UserServiceImpl in the batch.
	var serviceTestFile *scaffoldvo.ScaffoldFile
	for i := range writer.lastFiles {
		f := &writer.lastFiles[i]
		if f.Kind == scaffoldvo.KindTest && strings.HasSuffix(f.Path, "UserServiceImplTest.java") {
			serviceTestFile = f
			break
		}
	}
	require.NotNil(t, serviceTestFile, "expected a KindTest ScaffoldFile for UserServiceImplTest.java")
	require.True(t, strings.Contains(serviceTestFile.Path, "src/test/java"),
		"test file path must contain src/test/java, got %s", serviceTestFile.Path)

	// Find the captured TestRenderInput for UserServiceImpl.
	var serviceTestInput *scaffoldvo.TestRenderInput
	for i := range testRenderer.captured {
		inp := &testRenderer.captured[i]
		if inp.ClassName == "UserServiceImpl" {
			serviceTestInput = inp
			break
		}
	}
	require.NotNil(t, serviceTestInput, "expected a captured TestRenderInput for UserServiceImpl")

	// Mocks must list UserRepository.
	require.Len(t, serviceTestInput.Mocks, 1)
	require.Equal(t, "UserRepository", serviceTestInput.Mocks[0].Type)
	require.Equal(t, "userRepository", serviceTestInput.Mocks[0].FieldName)
}

// TestScaffoldUseCase_RestAdapter_MocksIsDedup asserts that a rest-adapter
// contract's Mocks is the dedup of Uses + DependsOn.
func TestScaffoldUseCase_RestAdapter_MocksIsDedup(t *testing.T) {
	t.Parallel()

	// Build a spec with a rest-adapter that has both Uses and DependsOn,
	// with an intentional overlap to verify dedup.
	spec := model.FeatureSpec{
		Feature: "checkout",
		Module:  "order",
		Package: "com.app.order",
		Contracts: []model.SpecContract{
			{
				Name:      "OrderController",
				Type:      model.ContractRestAdapter,
				Uses:      []string{"PlaceOrderUseCase", "SharedService"},
				DependsOn: []string{"SharedService", "OrderMetrics"},
				Endpoints: []string{"POST /orders"},
			},
		},
	}

	renderer := rendererAlwaysSucceeds()
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(spec), renderer, testRenderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "checkout",
		BaseDir: "/work",
	})
	require.NoError(t, err)

	require.Len(t, testRenderer.captured, 1)
	inp := testRenderer.captured[0]
	require.Equal(t, "OrderController", inp.ClassName)

	// DependsOn=[SharedService, OrderMetrics], Uses=[PlaceOrderUseCase, SharedService]
	// dedup(DependsOn + Uses) = [SharedService, OrderMetrics, PlaceOrderUseCase] (first-occurrence order)
	// Wait — the usecase does: dedup(append(DependsOn..., Uses...))
	// So order is: DependsOn first: SharedService, OrderMetrics; then Uses: PlaceOrderUseCase, SharedService(dup)
	// dedup result: [SharedService, OrderMetrics, PlaceOrderUseCase]
	mockTypes := make([]string, len(inp.Mocks))
	for i, m := range inp.Mocks {
		mockTypes[i] = m.Type
	}
	// Verify no duplicates.
	seen := make(map[string]int)
	for _, mt := range mockTypes {
		seen[mt]++
	}
	for typ, count := range seen {
		require.Equal(t, 1, count, "mock type %q appears more than once", typ)
	}
	// Verify all three unique types are present.
	require.Contains(t, mockTypes, "PlaceOrderUseCase")
	require.Contains(t, mockTypes, "SharedService")
	require.Contains(t, mockTypes, "OrderMetrics")
}

// TestScaffoldUseCase_Entity_ExactlyOnePlaceholderTestMethod asserts that an
// entity contract produces exactly one TestMethod with the placeholder name.
func TestScaffoldUseCase_Entity_ExactlyOnePlaceholderTestMethod(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature: "order",
		Module:  "order",
		Package: "com.app.order",
		Contracts: []model.SpecContract{
			{
				Name: "Order",
				Type: model.ContractEntity,
			},
		},
	}

	renderer := rendererAlwaysSucceeds()
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(spec), renderer, testRenderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "order",
		BaseDir: "/work",
	})
	require.NoError(t, err)

	require.Len(t, testRenderer.captured, 1)
	inp := testRenderer.captured[0]
	require.Equal(t, "Order", inp.ClassName)
	require.Len(t, inp.TestMethods, 1)
	require.Equal(t, "placeholder_shouldDoSomething", inp.TestMethods[0].Name)
	require.Equal(t, "// TODO: implement test", inp.TestMethods[0].Body)
}

// TestScaffoldUseCase_AggregateRoot_ExactlyOnePlaceholderTestMethod asserts the
// same as the entity case but for aggregate-root.
func TestScaffoldUseCase_AggregateRoot_ExactlyOnePlaceholderTestMethod(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature: "product",
		Module:  "catalog",
		Package: "com.app.catalog",
		Contracts: []model.SpecContract{
			{
				Name: "Product",
				Type: model.ContractAggregate,
			},
		},
	}

	renderer := rendererAlwaysSucceeds()
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(spec), renderer, testRenderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "product",
		BaseDir: "/work",
	})
	require.NoError(t, err)

	require.Len(t, testRenderer.captured, 1)
	inp := testRenderer.captured[0]
	require.Equal(t, "Product", inp.ClassName)
	require.Len(t, inp.TestMethods, 1)
	require.Equal(t, "placeholder_shouldDoSomething", inp.TestMethods[0].Name)
}

// TestScaffoldUseCase_InterfaceContracts_NoTestFile asserts that input-port and
// output-port contracts produce NO test file in the batch handed to the writer.
func TestScaffoldUseCase_InterfaceContracts_NoTestFile(t *testing.T) {
	t.Parallel()

	spec := model.FeatureSpec{
		Feature: "create-user",
		Module:  "user",
		Package: "com.app.user",
		Contracts: []model.SpecContract{
			{
				Name:    "CreateUserUseCase",
				Type:    model.ContractInputPort,
				Methods: []string{"void execute(CreateUserCommand cmd)"},
			},
			{
				Name: "UserRepository",
				Type: model.ContractOutputPort,
				Methods: []string{
					"Optional<User> findByEmail(String email)",
				},
			},
		},
	}

	renderer := rendererAlwaysSucceeds()
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(spec), renderer, testRenderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})
	require.NoError(t, err)

	// No test files must appear in the batch.
	for _, f := range writer.lastFiles {
		require.NotEqual(t, scaffoldvo.KindTest, f.Kind,
			"unexpected test file in batch for interface contract: %s", f.Path)
	}

	// testRenderer must not have been called at all.
	require.Equal(t, 0, testRenderer.callCount)
}

// TestScaffoldUseCase_SingleWriteAllCall asserts that the merged batch is
// passed in exactly ONE call to writer.WriteAll per Execute invocation.
func TestScaffoldUseCase_SingleWriteAllCall(t *testing.T) {
	t.Parallel()

	renderer := rendererAlwaysSucceeds()
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, testRenderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})
	require.NoError(t, err)

	require.Equal(t, 1, writer.callCount, "writer.WriteAll must be called exactly once per Execute")
}

// TestScaffoldUseCase_OutputCountersPopulated asserts that ScaffoldOutput
// ProductionCount and TestCount are populated correctly.
func TestScaffoldUseCase_OutputCountersPopulated(t *testing.T) {
	t.Parallel()

	// canonicalSpec has:
	//   - CreateUserUseCase (input-port):  production=yes, test=no
	//   - UserRepository (output-port):    production=yes, test=no
	//   - UserServiceImpl (service):       production=yes, test=yes
	//   - UserController (rest-adapter):   production=yes, test=yes
	// Expected: ProductionCount=4, TestCount=2
	renderer := rendererAlwaysSucceeds()
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, testRenderer, writer, nopLogger())
	out, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})
	require.NoError(t, err)

	require.Equal(t, 4, out.ProductionCount)
	require.Equal(t, 2, out.TestCount)
}

// TestScaffoldUseCase_MethodNameFreeze asserts that for UserServiceImpl whose
// Implements points to CreateUserUseCase with method "execute(...)", the test
// method name emitted is "execute_shouldDoSomething".
func TestScaffoldUseCase_MethodNameFreeze(t *testing.T) {
	t.Parallel()

	renderer := rendererAlwaysSucceeds()
	testRenderer := testRendererAlwaysSucceeds()
	writer := writerReturnsPaths()

	uc := newUC(finderReturnsPath(), parserReturns(canonicalSpec()), renderer, testRenderer, writer, nopLogger())
	_, err := uc.Execute(context.Background(), scaffoldvo.ScaffoldInput{
		Feature: "create-user",
		BaseDir: "/work",
	})
	require.NoError(t, err)

	// Find the captured TestRenderInput for UserServiceImpl.
	var serviceTestInput *scaffoldvo.TestRenderInput
	for i := range testRenderer.captured {
		inp := &testRenderer.captured[i]
		if inp.ClassName == "UserServiceImpl" {
			serviceTestInput = inp
			break
		}
	}
	require.NotNil(t, serviceTestInput, "expected a captured TestRenderInput for UserServiceImpl")

	// CreateUserUseCase has one method: "UserResponse execute(CreateUserCommand cmd)".
	// The test method name must be "execute_shouldDoSomething".
	require.Len(t, serviceTestInput.TestMethods, 1)
	require.Equal(t, "execute_shouldDoSomething", serviceTestInput.TestMethods[0].Name)
	require.Equal(t, "// TODO: implement test", serviceTestInput.TestMethods[0].Body)
}
