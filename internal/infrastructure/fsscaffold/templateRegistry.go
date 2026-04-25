package fsscaffold

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"
	"sync"
	"text/template"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/port/spec"
	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

//go:embed templates/*.tmpl
var rawTemplates embed.FS

// TemplateRegistry is an infrastructure adapter that satisfies
// spec.RenderProductionTemplatePort. It lazily parses all embedded .tmpl
// files on first use (sync.Once) and dispatches Render calls by ContractType.
type TemplateRegistry struct {
	once sync.Once
	tmpl *template.Template
	err  error
}

// NewRegistry returns a zero-value TemplateRegistry ready for use.
func NewRegistry() *TemplateRegistry { return &TemplateRegistry{} }

// compile-time check that TemplateRegistry satisfies the port.
var _ spec.RenderProductionTemplatePort = (*TemplateRegistry)(nil)

// templateNameByType maps the SpecContract.Type (string) to the template
// basename (without .tmpl).
var templateNameByType = map[string]string{
	"input-port":     "inputPort",
	"output-port":    "outputPort",
	"entity":         "entity",
	"aggregate-root": "aggregateRoot",
	"service":        "service",
	"rest-adapter":   "restAdapter",
	"jpa-adapter":    "jpaAdapter",
}

// Render selects the template for in.ContractType, executes it against in,
// and returns the resulting Java source bytes. An unrecognised ContractType
// returns *domerr.UnsupportedContractTypeError.
func (r *TemplateRegistry) Render(ctx context.Context, in scaffoldvo.RenderInput) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.once.Do(func() {
		tmpl := template.New("scaffold").Option("missingkey=error").Funcs(template.FuncMap{
			"join": strings.Join,
		})
		entries, err := rawTemplates.ReadDir("templates")
		if err != nil {
			r.err = fmt.Errorf("read embedded templates: %w", err)
			return
		}
		for _, e := range entries {
			content, rerr := rawTemplates.ReadFile("templates/" + e.Name())
			if rerr != nil {
				r.err = fmt.Errorf("read %s: %w", e.Name(), rerr)
				return
			}
			name := strings.TrimSuffix(e.Name(), ".tmpl")
			if _, perr := tmpl.New(name).Parse(string(content)); perr != nil {
				r.err = fmt.Errorf("parse %s: %w", e.Name(), perr)
				return
			}
		}
		r.tmpl = tmpl
	})
	if r.err != nil {
		return nil, r.err
	}

	name, ok := templateNameByType[in.ContractType]
	if !ok {
		supported := make([]string, 0, len(templateNameByType))
		for k := range templateNameByType {
			supported = append(supported, k)
		}
		sort.Strings(supported)
		return nil, &domerr.UnsupportedContractTypeError{Type: in.ContractType, SupportedSorted: supported}
	}

	var buf bytes.Buffer
	if err := r.tmpl.ExecuteTemplate(&buf, name, in); err != nil {
		return nil, fmt.Errorf("execute template %s: %w", name, err)
	}
	return buf.Bytes(), nil
}
