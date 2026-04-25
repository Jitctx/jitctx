package plannewuc

import (
	"context"

	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
)

// UseCase generates a new feature spec markdown file from the canonical
// template. Its single method composes RenderSpecTemplatePort and
// WriteSpecTemplatePort, using SpecPathResolver to compute the target
// path from input.Feature + input.BaseDir.
type UseCase interface {
	Execute(ctx context.Context, input planvo.NewTemplateInput) (planvo.NewTemplateOutput, error)
}
