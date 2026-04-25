package spec

import (
	"context"

	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

// RenderTestTemplatePort renders a single TestRenderInput into JUnit 5
// Java source bytes. The implementation selects which test template to
// execute based on input.ContractType. An unrecognised ContractType MUST
// return *domerr.UnsupportedContractTypeError so the use case can short-
// circuit before any disk I/O.
//
// This is a SECOND port (not a method on RenderProductionTemplatePort)
// because each port is one method (ISP) and the input VO type differs.
type RenderTestTemplatePort interface {
	Render(ctx context.Context, input scaffoldvo.TestRenderInput) ([]byte, error)
}
