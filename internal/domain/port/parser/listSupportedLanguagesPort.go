package parser

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/vo"
)

// ListSupportedLanguagesPort returns the language ids for which the binary
// embeds a query set, sorted alphabetically. Used by the bundle loader to
// produce the "available: …" suffix on LanguageUnsupportedError. A single
// adapter struct typically satisfies both this port and
// LoadLanguageQueriesPort (ISP composition by satisfaction, not
// inheritance).
type ListSupportedLanguagesPort interface {
	ListSupportedLanguages(ctx context.Context) ([]vo.Language, error)
}
