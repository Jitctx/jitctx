package query

import "github.com/jitctx/jitctx/internal/domain/vo"

type QueryContextInput struct {
	Module   string
	Tags     []string
	Types    []vo.ArtifactType
	FilePath string
	Budget   vo.TokenBudget
}

type QueryContextOutput struct {
	Loaded      []LoadedContext
	Trimmed     []LoadedContext
	TotalTokens int
}

type LoadedContext struct {
	ID            string
	Type          vo.ArtifactType
	Path          string
	Tags          []string
	Body          string
	TokenEstimate int
}
