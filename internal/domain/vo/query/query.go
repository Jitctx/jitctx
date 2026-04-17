package query

import (
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/vo"
)

type QueryContextInput struct {
	WorkDir  string            // project root used to open the fs.FS for reading context bodies
	Module   string            // required for US-002 (validated at the cobra layer)
	Tags     []string          // reserved for US-003
	Types    []vo.ArtifactType // reserved for US-003
	FilePath string            // reserved for US-005 (`--file` inference)
	Budget   vo.TokenBudget    // reserved for future budget story (already wired)
}

type QueryContextOutput struct {
	Module      ModuleSummary // summary of the queried module (for the contracts section)
	Loaded      []LoadedContext
	Trimmed     []LoadedContext
	TotalTokens int
}

// ModuleSummary projects only the fields the Markdown formatter needs,
// avoiding a hard dependency on model.Module in the presentation layer.
type ModuleSummary struct {
	ID        string
	Contracts []ContractSummary
}

type ContractSummary struct {
	Name    string
	Type    string   // string form of model.ContractType
	Methods []string // method signatures in order
}

type LoadedContext struct {
	ID            string
	Type          vo.ArtifactType
	Path          string
	Tags          []string
	Body          string // populated by the use case via ReadContextBodyPort
	TokenEstimate int
}

// newLoadedContextFromModel is an unexported helper living alongside the
// VO (permitted — it does not import infra/presentation).
func newLoadedContextFromModel(c model.Context, body string) LoadedContext {
	return LoadedContext{
		ID:            c.ID,
		Type:          c.Type,
		Path:          c.Path,
		Tags:          append([]string(nil), c.Tags...),
		Body:          body,
		TokenEstimate: c.TokenEstimate,
	}
}
