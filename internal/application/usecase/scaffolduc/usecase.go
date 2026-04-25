package scaffolduc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
	importResolver service.JavaImportResolver
	endpointSynth  service.EndpointSynthesizer
	idUtils        service.JavaIdentifierUtils
	renderer       spec.RenderProductionTemplatePort
	writer         spec.WriteProductionFilesPort
	logger         *slog.Logger
}

// New constructs a scaffolduc.Impl. All parameters are required.
func New(
	finder spec.FindSpecFilePort,
	parser spec.ParseSpecPort,
	mapper service.ContractPathMapper,
	importResolver service.JavaImportResolver,
	endpointSynth service.EndpointSynthesizer,
	idUtils service.JavaIdentifierUtils,
	renderer spec.RenderProductionTemplatePort,
	writer spec.WriteProductionFilesPort,
	logger *slog.Logger,
) *Impl {
	return &Impl{
		finder:         finder,
		parser:         parser,
		mapper:         mapper,
		importResolver: importResolver,
		endpointSynth:  endpointSynth,
		idUtils:        idUtils,
		renderer:       renderer,
		writer:         writer,
		logger:         logger,
	}
}

// Execute generates production Java source files for every contract in the
// resolved feature spec and atomically writes them to disk.
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

	// Step 6: Build batch.
	var batch []scaffoldvo.ProductionFile

	// Build a name → SpecContract index for Implements lookup.
	contractIndex := make(map[string]model.SpecContract, len(parsed.Contracts))
	for _, c := range parsed.Contracts {
		contractIndex[c.Name] = c
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
			frameworkAnnotation = "@Service"
		case model.ContractRestAdapter:
			frameworkAnnotation = "@RestController"
		case model.ContractEntity, model.ContractAggregate:
			frameworkAnnotation = "@Entity"
		case model.ContractJPAAdapter:
			frameworkAnnotation = "@Repository"
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
					methods = append(methods, scaffoldvo.RenderedMethod{
						Signature: m,
						Override:  true,
						Body:      `throw new UnsupportedOperationException("Not yet implemented");`,
					})
				}
			} else {
				// Fallback: emit c.Methods with throw body but WITHOUT @Override.
				u.logger.Warn("implements target not found",
					slog.String("name", c.Name),
					slog.String("target", c.Implements),
				)
				for _, m := range c.Methods {
					methods = append(methods, scaffoldvo.RenderedMethod{
						Signature: m,
						Override:  false,
						Body:      `throw new UnsupportedOperationException("Not yet implemented");`,
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
					Body:       `throw new UnsupportedOperationException("Not yet implemented");`,
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
			Fields:           append([]string(nil), c.Fields...),
			Methods:          methods,
			Endpoints:        endpoints,
			Dependencies:     deps,
		}

		// 6.4: Render.
		body, err := u.renderer.Render(ctx, vm)
		if err != nil {
			return scaffoldvo.ScaffoldOutput{}, &domerr.ScaffoldRenderError{Contract: c.Name, Cause: err}
		}

		// 6.5: Append to batch.
		batch = append(batch, scaffoldvo.ProductionFile{Path: abs, Content: body})
	}

	// Step 7: Single atomic write call.
	written, err := u.writer.WriteAll(ctx, batch)
	if err != nil {
		return scaffoldvo.ScaffoldOutput{}, err
	}

	// Step 8: Log + return.
	u.logger.Info("scaffolded", slog.String("feature", parsed.Feature), slog.Int("count", len(written)))
	return scaffoldvo.ScaffoldOutput{
		Feature:      parsed.Feature,
		Module:       parsed.Module,
		Package:      parsed.Package,
		WrittenPaths: written,
	}, nil
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
