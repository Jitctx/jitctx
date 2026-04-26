package spec

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

// RenderBundleTestTemplatePort is the test-side sibling of
// RenderBundleProductionTemplatePort. Same bundle-first / embedded-fallback
// semantics. Its testTemplateNameByType table mirrors the embed adapter's.
type RenderBundleTestTemplatePort interface {
	RenderWithBundleTest(
		ctx context.Context,
		bundle *model.ProfileBundle,
		input scaffoldvo.TestRenderInput,
	) ([]byte, error)
}
