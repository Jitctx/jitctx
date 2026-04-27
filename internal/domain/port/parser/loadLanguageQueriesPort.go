package parser

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/vo"
)

// LoadLanguageQueriesPort resolves a Language id to its bundled Tree-sitter
// query set. Returns errors.ErrLanguageUnsupported (wrapped in
// LanguageUnsupportedError) when no embedded query directory exists for
// the requested language. Implementations cache by language id so that
// repeated calls return the SAME *model.LanguageQuerySet pointer — this is
// the load-bearing invariant for EP04US-005 Scenario 3.
type LoadLanguageQueriesPort interface {
	LoadLanguageQueries(ctx context.Context, lang vo.Language) (*model.LanguageQuerySet, error)
}
