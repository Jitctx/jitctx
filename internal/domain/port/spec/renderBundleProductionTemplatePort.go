package spec

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

// RenderBundleProductionTemplatePort renders a single contract's view
// model into Java source bytes, sourcing the .tmpl text from the
// active ProfileBundle (when non-nil) and falling back to embedded
// templates otherwise. Selection by ContractType is unchanged from
// RenderProductionTemplatePort; only the lookup precedence differs.
//
// Bundle key convention: bundle.Templates is keyed by basename WITH
// the .tmpl suffix (e.g. "service.tmpl"). The scaffold renderer
// derives the lookup key by mapping ContractType through the same
// templateNameByType table the embed adapter uses, then appending
// ".tmpl".
type RenderBundleProductionTemplatePort interface {
	RenderWithBundle(
		ctx context.Context,
		bundle *model.ProfileBundle,
		input scaffoldvo.RenderInput,
	) ([]byte, error)
}
