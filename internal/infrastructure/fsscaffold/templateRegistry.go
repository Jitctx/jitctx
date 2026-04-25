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
	"unicode"
	"unicode/utf8"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/port/spec"
	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

//go:embed templates/*.tmpl
var rawTemplates embed.FS

// sharedOnce and sharedTmpl ensure the template parse pass (including FuncMap
// registration) runs exactly once per process, regardless of whether
// TemplateRegistry or TestTemplateRegistry triggers the first Render call.
// This eliminates any sequencing risk (§8 Q7).
var (
	sharedOnce sync.Once
	sharedTmpl *template.Template
	sharedErr  error
)

// lcFirst lowercases the first rune of s. Returns s unchanged when s is empty.
func lcFirst(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[size:]
}

func loadSharedTemplates() {
	sharedOnce.Do(func() {
		tmpl := template.New("scaffold").Option("missingkey=error").Funcs(template.FuncMap{
			"join":    strings.Join,
			"lcFirst": lcFirst,
		})
		entries, err := rawTemplates.ReadDir("templates")
		if err != nil {
			sharedErr = fmt.Errorf("read embedded templates: %w", err)
			return
		}
		for _, e := range entries {
			content, rerr := rawTemplates.ReadFile("templates/" + e.Name())
			if rerr != nil {
				sharedErr = fmt.Errorf("read %s: %w", e.Name(), rerr)
				return
			}
			name := strings.TrimSuffix(e.Name(), ".tmpl")
			if _, perr := tmpl.New(name).Parse(string(content)); perr != nil {
				sharedErr = fmt.Errorf("parse %s: %w", e.Name(), perr)
				return
			}
		}
		sharedTmpl = tmpl
	})
}

// TemplateRegistry is an infrastructure adapter that satisfies
// spec.RenderProductionTemplatePort. It lazily parses all embedded .tmpl
// files on first use (shared sync.Once) and dispatches Render calls by ContractType.
type TemplateRegistry struct{}

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

	loadSharedTemplates()
	if sharedErr != nil {
		return nil, sharedErr
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
	if err := sharedTmpl.ExecuteTemplate(&buf, name, in); err != nil {
		return nil, fmt.Errorf("execute template %s: %w", name, err)
	}
	return buf.Bytes(), nil
}

// TestTemplateRegistry is an infrastructure adapter that satisfies
// spec.RenderTestTemplatePort. It shares the same parsed *template.Template
// as TemplateRegistry via the package-level sharedOnce (§8 Q7).
type TestTemplateRegistry struct{}

// NewTestRegistry returns a zero-value TestTemplateRegistry ready for use.
func NewTestRegistry() *TestTemplateRegistry { return &TestTemplateRegistry{} }

// compile-time check that TestTemplateRegistry satisfies the port.
var _ spec.RenderTestTemplatePort = (*TestTemplateRegistry)(nil)

// testTemplateNameByType maps SpecContract.Type to the test template basename
// (without .tmpl). Only the four non-interface contract types are testable by
// this registry.
var testTemplateNameByType = map[string]string{
	"service":        "serviceTest",
	"rest-adapter":   "restAdapterTest",
	"entity":         "entityTest",
	"aggregate-root": "aggregateRootTest",
}

// Render selects the test template for input.ContractType, executes it against
// input, and returns the resulting JUnit 5 Java source bytes. An unrecognised
// ContractType returns *domerr.UnsupportedContractTypeError listing the four
// test-supported types in sorted order.
func (r *TestTemplateRegistry) Render(ctx context.Context, input scaffoldvo.TestRenderInput) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	loadSharedTemplates()
	if sharedErr != nil {
		return nil, sharedErr
	}

	name, ok := testTemplateNameByType[input.ContractType]
	if !ok {
		supported := make([]string, 0, len(testTemplateNameByType))
		for k := range testTemplateNameByType {
			supported = append(supported, k)
		}
		sort.Strings(supported)
		return nil, &domerr.UnsupportedContractTypeError{Type: input.ContractType, SupportedSorted: supported}
	}

	var buf bytes.Buffer
	if err := sharedTmpl.ExecuteTemplate(&buf, name, input); err != nil {
		return nil, fmt.Errorf("execute test template %s: %w", name, err)
	}
	return buf.Bytes(), nil
}
