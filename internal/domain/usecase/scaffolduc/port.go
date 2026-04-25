package scaffolduc

import (
	"context"

	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

// UseCase generates production Java source files for every contract in a
// feature spec. Orchestrates: spec resolution (FindSpecFilePort) → markdown
// parse (ParseSpecPort) → Package validation → per-contract view-model
// build (JavaImportResolver, EndpointSynthesizer, JavaIdentifierUtils) →
// template render (RenderProductionTemplatePort) → atomic batch write
// (WriteProductionFilesPort).
type UseCase interface {
	Execute(ctx context.Context, input scaffoldvo.ScaffoldInput) (scaffoldvo.ScaffoldOutput, error)
}
