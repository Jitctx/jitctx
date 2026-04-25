package spec

import (
	"context"

	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

// RenderProductionTemplatePort renders a single contract's view model
// into Java source bytes. The implementation selects which template to
// execute based on input.ContractType. An unrecognised ContractType MUST
// return *domerr.UnsupportedContractTypeError so the use case can short-
// circuit before any disk I/O.
type RenderProductionTemplatePort interface {
	Render(ctx context.Context, input scaffoldvo.RenderInput) ([]byte, error)
}
