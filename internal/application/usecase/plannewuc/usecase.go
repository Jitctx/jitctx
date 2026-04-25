package plannewuc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/port/spec"
	"github.com/jitctx/jitctx/internal/domain/service"
	domplannewuc "github.com/jitctx/jitctx/internal/domain/usecase/plannewuc"
	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
)

// Compile-time check that Impl satisfies the domain interface.
var _ domplannewuc.UseCase = (*Impl)(nil)

// Impl orchestrates RenderSpecTemplatePort + WriteSpecTemplatePort +
// SpecPathResolver to generate a new feature spec markdown file.
type Impl struct {
	renderer spec.RenderSpecTemplatePort
	writer   spec.WriteSpecTemplatePort
	resolver service.SpecPathResolver
	logger   *slog.Logger
}

// New constructs an Impl with the given collaborators.
func New(
	r spec.RenderSpecTemplatePort,
	w spec.WriteSpecTemplatePort,
	res service.SpecPathResolver,
	l *slog.Logger,
) *Impl {
	return &Impl{renderer: r, writer: w, resolver: res, logger: l}
}

// Execute generates a new feature spec markdown file from the canonical
// template. It resolves the target path, renders the template, and writes
// the result atomically to disk.
func (u *Impl) Execute(ctx context.Context, in planvo.NewTemplateInput) (planvo.NewTemplateOutput, error) {
	if err := ctx.Err(); err != nil {
		return planvo.NewTemplateOutput{}, err
	}

	target, err := u.resolver.Resolve(in.Feature, in.BaseDir)
	if err != nil {
		return planvo.NewTemplateOutput{}, fmt.Errorf("resolve spec path: %w", err)
	}

	body, err := u.renderer.Render(ctx, in.Feature, in.Module)
	if err != nil {
		return planvo.NewTemplateOutput{}, fmt.Errorf("render template: %w", err)
	}

	written, err := u.writer.Write(ctx, target, body)
	if err != nil {
		// Pass typed SpecFileExistsError through unchanged so errors.As /
		// errors.Is works in format.TranslateError; otherwise wrap with context.
		var existsErr *domerr.SpecFileExistsError
		if errors.As(err, &existsErr) {
			return planvo.NewTemplateOutput{}, err
		}
		return planvo.NewTemplateOutput{}, fmt.Errorf("write template: %w", err)
	}

	u.logger.Info("created spec template",
		slog.String("feature", in.Feature),
		slog.String("module", in.Module),
		slog.String("path", written))

	return planvo.NewTemplateOutput{Path: written}, nil
}
