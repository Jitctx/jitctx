package fsspec

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"sync"
	"text/template"

	"github.com/jitctx/jitctx/internal/domain/port/spec"
)

//go:embed templates/newFeatureSpec.tmpl
var rawTemplate string

// Compile-time assertion: TemplateRenderer satisfies RenderSpecTemplatePort.
var _ spec.RenderSpecTemplatePort = (*TemplateRenderer)(nil)

// TemplateRenderer renders the canonical "new feature" markdown template.
// It embeds the template source at compile time and parses it lazily on first
// use so that New is cheap (no I/O, no allocation).
type TemplateRenderer struct {
	once sync.Once
	tmpl *template.Template
	err  error
}

// New returns a new TemplateRenderer. Parsing of the embedded template is
// deferred to the first Render call.
func New() *TemplateRenderer {
	return &TemplateRenderer{}
}

// Render executes the embedded template with the given feature and module
// values and returns the rendered bytes. It honours ctx.Err() before execution
// and returns a wrapped error if the template fails to render.
func (r *TemplateRenderer) Render(ctx context.Context, feature, module string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.once.Do(func() {
		r.tmpl, r.err = template.New("newFeatureSpec").
			Option("missingkey=error").
			Parse(rawTemplate)
	})
	if r.err != nil {
		return nil, fmt.Errorf("parse spec template: %w", r.err)
	}

	data := struct {
		Feature string
		Module  string
	}{
		Feature: feature,
		Module:  module,
	}

	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute spec template: %w", err)
	}

	return buf.Bytes(), nil
}
