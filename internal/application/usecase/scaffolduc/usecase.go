package scaffolduc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	spec "github.com/jitctx/jitctx/internal/domain/port/spec"
	"github.com/jitctx/jitctx/internal/domain/service"
	domscaffolduc "github.com/jitctx/jitctx/internal/domain/usecase/scaffolduc"
	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

var _ domscaffolduc.UseCase = (*Impl)(nil)

// Impl satisfies domscaffolduc.UseCase and orchestrates:
// spec resolution → parse → Package validation → per-contract view-model
// build → template render → atomic batch write.
type Impl struct {
	finder         spec.FindSpecFilePort
	parser         spec.ParseSpecPort
	mapper         service.ContractPathMapper
	testMapper     service.TestPathMapper
	importResolver service.JavaImportResolver
	endpointSynth  service.EndpointSynthesizer
	idUtils        service.JavaIdentifierUtils
	methodParser   service.MethodSignatureParser
	jpaAnnotator   service.JPAFieldAnnotator
	renderer       spec.RenderProductionTemplatePort
	testRenderer   spec.RenderTestTemplatePort
	writer         spec.WriteProductionFilesPort
	logger         *slog.Logger
}

// New constructs a scaffolduc.Impl. All parameters are required.
func New(
	finder spec.FindSpecFilePort,
	parser spec.ParseSpecPort,
	mapper service.ContractPathMapper,
	testMapper service.TestPathMapper,
	importResolver service.JavaImportResolver,
	endpointSynth service.EndpointSynthesizer,
	idUtils service.JavaIdentifierUtils,
	methodParser service.MethodSignatureParser,
	jpaAnnotator service.JPAFieldAnnotator,
	renderer spec.RenderProductionTemplatePort,
	testRenderer spec.RenderTestTemplatePort,
	writer spec.WriteProductionFilesPort,
	logger *slog.Logger,
) *Impl {
	return &Impl{
		finder:         finder,
		parser:         parser,
		mapper:         mapper,
		testMapper:     testMapper,
		importResolver: importResolver,
		endpointSynth:  endpointSynth,
		idUtils:        idUtils,
		methodParser:   methodParser,
		jpaAnnotator:   jpaAnnotator,
		renderer:       renderer,
		testRenderer:   testRenderer,
		writer:         writer,
		logger:         logger,
	}
}

// Execute generates production and test Java source files for every contract
// in the resolved feature spec and atomically writes them to disk in one
// merged batch (EP02RF-009).
func (u *Impl) Execute(ctx context.Context, in scaffoldvo.ScaffoldInput) (scaffoldvo.ScaffoldOutput, error) {
	// Step 1: ctx.Err() guard.
	if err := ctx.Err(); err != nil {
		return scaffoldvo.ScaffoldOutput{}, err
	}

	// Step 2: Validate exclusivity.
	if in.Feature == "" && in.FilePath == "" {
		return scaffoldvo.ScaffoldOutput{}, errors.New("either --feature or --file is required")
	}
	if in.Feature != "" && in.FilePath != "" {
		return scaffoldvo.ScaffoldOutput{}, errors.New("--feature and --file are mutually exclusive")
	}

	// Step 3: Resolve + read spec.
	var (
		path    string
		content []byte
		alts    []string
	)
	if in.FilePath != "" {
		var err error
		content, err = os.ReadFile(in.FilePath)
		if err != nil {
			return scaffoldvo.ScaffoldOutput{}, fmt.Errorf("read spec file %s: %w", in.FilePath, err)
		}
		path = in.FilePath
	} else {
		var err error
		path, content, alts, err = u.finder.Find(ctx, in.Feature, in.BaseDir, in.PlansDir)
		if err != nil {
			var nf *domerr.SpecFileNotFoundError
			if errors.As(err, &nf) {
				return scaffoldvo.ScaffoldOutput{}, err
			}
			return scaffoldvo.ScaffoldOutput{}, fmt.Errorf("find spec file: %w", err)
		}
	}

	// Step 4: Parse.
	parsed, warns, err := u.parser.ParseSpec(ctx, string(content))
	if err != nil {
		return scaffoldvo.ScaffoldOutput{}, fmt.Errorf("parse spec %s: %w", path, err)
	}
	for _, w := range warns {
		u.logger.Warn("spec warning", slog.Int("line", w.Line), slog.String("msg", w.Message))
	}
	for _, alt := range alts {
		u.logger.Warn("spec found in additional location", slog.String("primary", path), slog.String("additional", alt))
	}

	// Step 5: Validate Package.
	if parsed.Package == "" {
		return scaffoldvo.ScaffoldOutput{}, domerr.ErrSpecMissingPackage
	}

	// Step 6: Build production batch entries.
	// The batch holds the unified set of production + test files.
	var batch []scaffoldvo.ScaffoldFile
	productionCount := 0
	testCount := 0

	// Build a name → SpecContract index for Implements lookup and test import resolution.
	contractIndex := make(map[string]model.SpecContract, len(parsed.Contracts))
	for _, c := range parsed.Contracts {
		contractIndex[c.Name] = c
	}

	// Build production relPath index for test FQN resolution.
	productionRelPaths := make(map[string]string, len(parsed.Contracts))
	for _, c := range parsed.Contracts {
		relPath, err := u.mapper.Map(c.Type, c.Name)
		if err != nil {
			return scaffoldvo.ScaffoldOutput{}, err
		}
		if relPath != "" {
			productionRelPaths[c.Name] = relPath
		}
	}

	for _, c := range parsed.Contracts {
		// 6.1: Map contract to relative path.
		relPath, err := u.mapper.Map(c.Type, c.Name)
		if err != nil {
			// UnsupportedContractTypeError carries SupportedSorted — return as-is.
			return scaffoldvo.ScaffoldOutput{}, err
		}

		// 6.2: Compute absolute write path.
		packagePath := strings.ReplaceAll(parsed.Package, ".", "/")
		abs := filepath.Join(in.BaseDir, "src/main/java", packagePath, relPath)

		// 6.3: Build view model.

		// subPackage and fullPackage.
		subPackage := u.idUtils.PackageFromRelativePath(relPath)
		fullPackage := parsed.Package
		if subPackage != "" {
			fullPackage = parsed.Package + "." + subPackage
		}

		// Imports.
		imports, _ := u.importResolver.Resolve(parsed, c, parsed.Package)

		// classAnnotations: framework annotation first, then spec-declared annotations.
		var frameworkAnnotation string
		switch c.Type {
		case model.ContractService:
			frameworkAnnotation = annotationService
		case model.ContractRestAdapter:
			frameworkAnnotation = annotationRestController
		case model.ContractEntity, model.ContractAggregate:
			frameworkAnnotation = annotationEntity
		case model.ContractJPAAdapter:
			frameworkAnnotation = annotationRepository
		}

		var rawAnnotations []string
		if frameworkAnnotation != "" {
			rawAnnotations = append(rawAnnotations, frameworkAnnotation)
		}
		rawAnnotations = append(rawAnnotations, c.Annotations...)
		classAnnotations := dedupStrings(rawAnnotations)

		// implementsName: only for service.
		var implementsName string
		if c.Type == model.ContractService {
			implementsName = c.Implements
		}

		// Methods — branch on type.
		var methods []scaffoldvo.RenderedMethod
		switch c.Type {
		case model.ContractInputPort, model.ContractOutputPort:
			for _, m := range c.Methods {
				methods = append(methods, scaffoldvo.RenderedMethod{
					Signature: m,
					Override:  false,
					Body:      "",
				})
			}
		case model.ContractService, model.ContractJPAAdapter:
			implTarget, found := contractIndex[c.Implements]
			if found && c.Implements != "" {
				for _, m := range implTarget.Methods {
					pm, perr := u.methodParser.Parse(m)
					if perr != nil {
						return scaffoldvo.ScaffoldOutput{}, &domerr.ScaffoldRenderError{Contract: c.Name, Cause: perr}
					}
					methods = append(methods, scaffoldvo.RenderedMethod{
						Signature: m,
						Override:  true,
						Body:      buildMethodBody(c.Name, pm.Name, pm.ReturnType),
					})
				}
			} else {
				// Fallback: emit c.Methods with TODO body but WITHOUT @Override.
				u.logger.Warn("implements target not found",
					slog.String("name", c.Name),
					slog.String("target", c.Implements),
				)
				for _, m := range c.Methods {
					pm, perr := u.methodParser.Parse(m)
					if perr != nil {
						return scaffoldvo.ScaffoldOutput{}, &domerr.ScaffoldRenderError{Contract: c.Name, Cause: perr}
					}
					methods = append(methods, scaffoldvo.RenderedMethod{
						Signature: m,
						Override:  false,
						Body:      buildMethodBody(c.Name, pm.Name, pm.ReturnType),
					})
				}
			}
		case model.ContractEntity, model.ContractAggregate, model.ContractRestAdapter:
			// Empty methods: entities have no methods rendered; rest-adapter uses Endpoints.
			methods = nil
		}

		// Endpoints — only for rest-adapter.
		var endpoints []scaffoldvo.RenderedEndpoint
		if c.Type == model.ContractRestAdapter {
			for _, raw := range c.Endpoints {
				eb, err := u.endpointSynth.Parse(raw)
				if err != nil {
					return scaffoldvo.ScaffoldOutput{}, fmt.Errorf("parse endpoint %q: %w", raw, err)
				}
				endpoints = append(endpoints, scaffoldvo.RenderedEndpoint{
					Annotation: eb.Annotation,
					Method:     eb.MethodName,
					// restAdapter.tmpl renders endpoints with `public Object <method>()`,
					// never void, so the returnType is hard-coded to "Object" here so the
					// throw line is always emitted. Keeping the same helper gives format
					// consistency with service / jpa-adapter (US-001 acceptance scenario
					// "TODO format is consistent across types").
					Body: buildMethodBody(c.Name, eb.MethodName, "Object"),
				})
			}
		}

		// Dependencies: service / rest-adapter / jpa-adapter.
		var deps []scaffoldvo.ConstructorDep
		switch c.Type {
		case model.ContractService, model.ContractJPAAdapter:
			for _, n := range c.DependsOn {
				deps = append(deps, scaffoldvo.ConstructorDep{
					Type:      n,
					FieldName: u.idUtils.FieldNameFromType(n),
				})
			}
		case model.ContractRestAdapter:
			combined := dedupStrings(append(append([]string(nil), c.DependsOn...), c.Uses...))
			for _, n := range combined {
				deps = append(deps, scaffoldvo.ConstructorDep{
					Type:      n,
					FieldName: u.idUtils.FieldNameFromType(n),
				})
			}
		}

		// Assemble view model.
		vm := scaffoldvo.RenderInput{
			ContractType:     string(c.Type),
			Package:          fullPackage,
			ClassName:        c.Name,
			Imports:          imports,
			ClassAnnotations: classAnnotations,
			Implements:       implementsName,
			Fields:           u.jpaAnnotator.Annotate(c.Fields),
			Methods:          methods,
			Endpoints:        endpoints,
			Dependencies:     deps,
		}

		// 6.4: Render.
		body, err := u.renderer.Render(ctx, vm)
		if err != nil {
			return scaffoldvo.ScaffoldOutput{}, &domerr.ScaffoldRenderError{Contract: c.Name, Cause: err}
		}

		// 6.5: Append production file to batch (wrapped in ScaffoldFile).
		batch = append(batch, scaffoldvo.ScaffoldFile{
			Path:    abs,
			Content: body,
			Kind:    scaffoldvo.KindProduction,
		})
		productionCount++
	}

	// Step 6b: Build test batch entries.
	for _, c := range parsed.Contracts {
		// 6b.1: Map contract to test relative path.
		relPath, err := u.testMapper.Map(c.Type, c.Name)
		if err != nil {
			// Genuine unsupported contract type — return as-is.
			return scaffoldvo.ScaffoldOutput{}, err
		}
		if relPath == "" {
			// Intentionally non-testable (input-port, output-port, jpa-adapter).
			u.logger.Debug("skipping non-testable contract", slog.String("name", c.Name), slog.String("type", string(c.Type)))
			continue
		}

		// 6b.2: Compute test absolute path.
		packagePath := strings.ReplaceAll(parsed.Package, ".", "/")
		testAbs := filepath.Join(in.BaseDir, "src/test/java", packagePath, relPath)

		// 6b.3: Derive the production relPath to compute full package.
		prodRelPath := productionRelPaths[c.Name]
		subPackage := u.idUtils.PackageFromRelativePath(prodRelPath)
		fullPackage := parsed.Package
		if subPackage != "" {
			fullPackage = parsed.Package + "." + subPackage
		}

		// 6b.4: Build Mocks.
		var mocks []scaffoldvo.TestMockField
		switch c.Type {
		case model.ContractService:
			for _, n := range c.DependsOn {
				mocks = append(mocks, scaffoldvo.TestMockField{
					Type:      n,
					FieldName: u.idUtils.FieldNameFromType(n),
				})
			}
		case model.ContractRestAdapter:
			combined := dedupStrings(append(append([]string(nil), c.DependsOn...), c.Uses...))
			for _, n := range combined {
				mocks = append(mocks, scaffoldvo.TestMockField{
					Type:      n,
					FieldName: u.idUtils.FieldNameFromType(n),
				})
			}
		case model.ContractEntity, model.ContractAggregate:
			mocks = nil
		}

		// 6b.5: Build TestMethods.
		var testMethods []scaffoldvo.TestMethod
		switch c.Type {
		case model.ContractService:
			// For service: parse each c.Methods entry via methodParser.
			// If c.Methods is empty (implements target provides methods), try the implements target.
			methodSources := c.Methods
			if len(methodSources) == 0 && c.Implements != "" {
				if implTarget, found := contractIndex[c.Implements]; found {
					methodSources = implTarget.Methods
				}
			}
			for _, m := range methodSources {
				pm, err := u.methodParser.Parse(m)
				if err != nil {
					return scaffoldvo.ScaffoldOutput{}, &domerr.ScaffoldRenderError{Contract: c.Name, Cause: err}
				}
				testMethods = append(testMethods, scaffoldvo.TestMethod{
					Name: pm.Name + "_shouldDoSomething",
					Body: "// TODO: implement test",
				})
			}
		case model.ContractRestAdapter:
			// For rest-adapter: parse each c.Methods entry. When c.Methods is empty,
			// derive method names from endpoints via endpointSynth.
			if len(c.Methods) > 0 {
				for _, m := range c.Methods {
					pm, err := u.methodParser.Parse(m)
					if err != nil {
						return scaffoldvo.ScaffoldOutput{}, &domerr.ScaffoldRenderError{Contract: c.Name, Cause: err}
					}
					testMethods = append(testMethods, scaffoldvo.TestMethod{
						Name: pm.Name + "_shouldDoSomething",
						Body: "// TODO: implement test",
					})
				}
			} else {
				// Derive method names from endpoints.
				for _, raw := range c.Endpoints {
					eb, err := u.endpointSynth.Parse(raw)
					if err != nil {
						return scaffoldvo.ScaffoldOutput{}, fmt.Errorf("parse endpoint %q: %w", raw, err)
					}
					testMethods = append(testMethods, scaffoldvo.TestMethod{
						Name: eb.MethodName + "_shouldDoSomething",
						Body: "// TODO: implement test",
					})
				}
			}
		case model.ContractEntity, model.ContractAggregate:
			// One placeholder test method.
			testMethods = []scaffoldvo.TestMethod{
				{Name: "placeholder_shouldDoSomething", Body: "// TODO: implement test"},
			}
		}

		// 6b.6: Build Imports.
		testImports := buildTestImports(c.Type, mocks, contractIndex, productionRelPaths, parsed.Package, u.idUtils)

		// 6b.7: Assemble TestRenderInput.
		tvm := scaffoldvo.TestRenderInput{
			ContractType: string(c.Type),
			Package:      fullPackage,
			ClassName:    c.Name,
			Imports:      testImports,
			Mocks:        mocks,
			TestMethods:  testMethods,
		}

		// 6b.8: Render test.
		testBody, err := u.testRenderer.Render(ctx, tvm)
		if err != nil {
			return scaffoldvo.ScaffoldOutput{}, &domerr.ScaffoldRenderError{Contract: c.Name, Cause: err}
		}

		// 6b.9: Append test file to batch.
		batch = append(batch, scaffoldvo.ScaffoldFile{
			Path:    testAbs,
			Content: testBody,
			Kind:    scaffoldvo.KindTest,
		})
		testCount++
	}

	// Step 7: Single atomic write call — ONE merged batch (production + test).
	written, err := u.writer.WriteAll(ctx, batch)
	if err != nil {
		return scaffoldvo.ScaffoldOutput{}, err
	}

	// Step 8: Log + return with counters.
	u.logger.Info("scaffolded",
		slog.String("feature", parsed.Feature),
		slog.Int("production", productionCount),
		slog.Int("test", testCount),
		slog.Int("total", len(written)),
	)
	return scaffoldvo.ScaffoldOutput{
		Feature:         parsed.Feature,
		Module:          parsed.Module,
		Package:         parsed.Package,
		WrittenPaths:    written,
		ProductionCount: productionCount,
		TestCount:       testCount,
	}, nil
}

// buildTestImports constructs the sorted import list for a test class.
//
//   - Always includes org.junit.jupiter.api.Test.
//   - For service / rest-adapter: adds the four Mockito FQNs plus the FQN
//     of each mock type (if resolvable from the contract index).
//   - SUT FQN is OMITTED because the test shares the production package.
func buildTestImports(
	contractType model.ContractType,
	mocks []scaffoldvo.TestMockField,
	contractIndex map[string]model.SpecContract,
	productionRelPaths map[string]string,
	modulePackage string,
	idUtils service.JavaIdentifierUtils,
) []string {
	importSet := make(map[string]struct{})
	importSet["org.junit.jupiter.api.Test"] = struct{}{}

	switch contractType {
	case model.ContractService, model.ContractRestAdapter:
		importSet["org.junit.jupiter.api.extension.ExtendWith"] = struct{}{}
		importSet["org.mockito.InjectMocks"] = struct{}{}
		importSet["org.mockito.Mock"] = struct{}{}
		importSet[importTestRunnerExtensionFQN] = struct{}{}

		// For each mock, attempt FQN resolution via the contract index.
		for _, m := range mocks {
			if _, found := contractIndex[m.Type]; found {
				if relPath, ok := productionRelPaths[m.Type]; ok {
					fqn := idUtils.FQN(modulePackage, relPath, m.Type)
					importSet[fqn] = struct{}{}
				}
			}
			// External types (not in contract index) are skipped silently.
		}
	}

	// Build sorted slice.
	result := make([]string, 0, len(importSet))
	for imp := range importSet {
		result = append(result, imp)
	}
	sort.Strings(result)
	return result
}

// dedupStrings returns in with duplicates removed, preserving first occurrence order.
func dedupStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// buildMethodBody returns the body string for a class-method stub. The
// returned string is dropped verbatim into the {{.Body}} action of the
// service / jpa-adapter / rest-adapter template, which precedes it with
// 8 spaces of indentation. When the body is multi-line, every line AFTER
// the first must be prefixed with 8 spaces (the template only indents the
// first line) so that the rendered Java source has each statement at the
// expected method-body indent.
//
// Format:
//
//	"// TODO(jitctx): implement <ClassName>.<methodName>"
//	"        throw new UnsupportedOperationException(\"Not yet implemented\");"   // omitted when returnType == "void"
//
// Inputs:
//
//	className   — contract class name (e.g. "UserServiceImpl"). Required.
//	methodName  — Java method identifier (e.g. "execute"). Required.
//	returnType  — Java return-type token, lowercase-comparable. Pass "void"
//	              to omit the throw line. Pass "" or any other type to keep it.
//
// The function is total — never returns an error and never panics.
func buildMethodBody(className, methodName, returnType string) string {
	todo := "// TODO(jitctx): implement " + className + "." + methodName
	if returnType == "void" {
		return todo
	}
	return todo + "\n        throw new UnsupportedOperationException(\"Not yet implemented\");"
}
