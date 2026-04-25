package spec

import "context"

// RenderSpecTemplatePort renders the canonical "new feature" markdown
// template, substituting feature and module placeholders. The renderer
// is deterministic (same inputs → byte-identical output) per EP02RNF-002.
//
// Returned bytes are ready to write to disk verbatim — they include the
// trailing newline and the placeholder contract blocks for input-port,
// service, and rest-adapter, with `<Name>` and `<TODO>` markers.
type RenderSpecTemplatePort interface {
	Render(ctx context.Context, feature, module string) ([]byte, error)
}
