package model

import "github.com/jitctx/jitctx/internal/domain/vo"

// LanguageQuerySet is a frozen, in-memory view of every Tree-sitter query
// (.scm) file the binary embeds for one language. Returned by
// parser.LoadLanguageQueriesPort. The returned pointer is the SAME pointer
// across repeated calls for the same Language — the registry caches by
// language id, so two profiles declaring the same language share one
// LanguageQuerySet without binary duplication. Tests rely on this
// pointer-equality invariant (see EP04US-005 Scenario 3).
type LanguageQuerySet struct {
	// Language is the canonical id this set was loaded for. Set by the
	// registry; callers MUST NOT mutate.
	Language vo.Language

	// Queries is keyed by .scm file basename relative to the language
	// directory (e.g. "declarations.scm"). Values are the raw bytes —
	// the parser side parses them via tree-sitter's QueryNew when it
	// starts to consume them in a future US. Iteration order is
	// undefined; callers that need deterministic iteration sort the
	// keys themselves.
	Queries map[string][]byte
}
