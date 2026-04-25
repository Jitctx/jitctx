package planuc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/port/spec"
	"github.com/jitctx/jitctx/internal/domain/service"
	domplanuc "github.com/jitctx/jitctx/internal/domain/usecase/planuc"
	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
)

type Impl struct {
	finder  spec.FindSpecFilePort
	parser  spec.ParseSpecPort
	layerer service.DependencyLayerer
	mapper  service.ContractPathMapper
	logger  *slog.Logger
}

func New(
	f spec.FindSpecFilePort,
	p spec.ParseSpecPort,
	l service.DependencyLayerer,
	m service.ContractPathMapper,
	log *slog.Logger,
) *Impl {
	return &Impl{finder: f, parser: p, layerer: l, mapper: m, logger: log}
}

var _ domplanuc.UseCase = (*Impl)(nil)

func (u *Impl) Execute(ctx context.Context, in planvo.LayersInput) (planvo.LayersOutput, error) {
	// Step 1: ctx.Err() guard
	if err := ctx.Err(); err != nil {
		return planvo.LayersOutput{}, err
	}

	// Step 2: Validate exclusivity
	if in.Feature == "" && in.FilePath == "" {
		return planvo.LayersOutput{}, errors.New("either --feature or --file is required")
	}
	if in.Feature != "" && in.FilePath != "" {
		return planvo.LayersOutput{}, errors.New("--feature and --file are mutually exclusive")
	}

	// Step 3: Resolve + read
	var (
		path    string
		content []byte
		alts    []string
	)
	if in.FilePath != "" {
		var err error
		content, err = os.ReadFile(in.FilePath)
		if err != nil {
			return planvo.LayersOutput{}, fmt.Errorf("read spec file %s: %w", in.FilePath, err)
		}
		path = in.FilePath
	} else {
		var err error
		path, content, alts, err = u.finder.Find(ctx, in.Feature, in.BaseDir, in.PlansDir)
		if err != nil {
			var nf *domerr.SpecFileNotFoundError
			if errors.As(err, &nf) {
				return planvo.LayersOutput{}, err
			}
			return planvo.LayersOutput{}, fmt.Errorf("find spec file: %w", err)
		}
	}

	// Step 4: Parse
	parsed, warns, err := u.parser.ParseSpec(ctx, string(content))
	if err != nil {
		return planvo.LayersOutput{}, fmt.Errorf("parse spec %s: %w", path, err)
	}
	for _, w := range warns {
		u.logger.Warn("spec warning", slog.Int("line", w.Line), slog.String("msg", w.Message))
	}

	// Step 5: Log alts
	for _, alt := range alts {
		u.logger.Warn("spec found in additional location", slog.String("primary", path), slog.String("additional", alt))
	}

	// Step 6: Layer
	layers, externals, err := u.layerer.Layer(parsed.Contracts)
	if err != nil {
		return planvo.LayersOutput{}, fmt.Errorf("layering: %w", err)
	}

	// Step 7: Log externals — single-quoted so the rendered slog text contains `'ExternalThing'`
	for _, ex := range externals {
		u.logger.Warn("external reference", slog.String("name", "'"+ex+"'"))
	}

	// Step 8: Map paths
	for li := range layers {
		for ti := range layers[li].Targets {
			t := &layers[li].Targets[ti]
			p, err := u.mapper.Map(model.ContractType(t.Type), t.Name)
			if err != nil {
				return planvo.LayersOutput{}, err
			}
			t.TargetPath = p
		}
	}

	// Step 9: Assemble and return
	return planvo.LayersOutput{
		Feature:   parsed.Feature,
		Module:    parsed.Module,
		Layers:    layers,
		Externals: externals,
	}, nil
}
