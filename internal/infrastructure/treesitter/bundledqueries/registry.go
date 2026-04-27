package bundledqueries

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"slices"
	"strings"
	"sync"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	parserport "github.com/jitctx/jitctx/internal/domain/port/parser"
	"github.com/jitctx/jitctx/internal/domain/vo"
)

// Registry implements parser.LoadLanguageQueriesPort and
// parser.ListSupportedLanguagesPort. It owns an in-memory cache of
// LanguageQuerySet aggregates keyed by language id; the cache is populated
// lazily on first request and subsequent requests for the same language
// return the SAME pointer. This pointer-equality invariant is the
// load-bearing assertion for the EP04US-005 "binary contains the queries
// only once" feature scenario — two profiles that both declare
// "language: java" share the same *model.LanguageQuerySet, with no
// duplication in the binary.
type Registry struct {
	logger    *slog.Logger
	supported []vo.Language // computed once at construction; sorted
	mu        sync.Mutex
	cache     map[vo.Language]*model.LanguageQuerySet
}

// NewRegistry constructs a Registry. The list of supported languages is
// computed eagerly by walking the embed root one level deep — every
// subdirectory is treated as a language id. Individual .scm files within
// each language directory are read lazily on first LoadLanguageQueries
// call. When logger is nil, slog.Default() is used.
func NewRegistry(logger *slog.Logger) *Registry {
	if logger == nil {
		logger = slog.Default()
	}
	r := &Registry{
		logger: logger,
		cache:  make(map[vo.Language]*model.LanguageQuerySet),
	}
	entries, err := fs.ReadDir(bundledFS, ".")
	if err != nil {
		// embed.FS at the package root cannot fail under go:embed; log and
		// fall through with an empty supported list so the constructor
		// never panics.
		logger.Warn("bundled queries: read embed root failed",
			slog.String("error", err.Error()))
		return r
	}
	supported := make([]vo.Language, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		supported = append(supported, vo.Language(entry.Name()))
	}
	slices.Sort(supported)
	r.supported = supported
	return r
}

// LoadLanguageQueries implements parser.LoadLanguageQueriesPort. It returns
// the cached query set for the requested language, reading the .scm files
// from the embedded filesystem on first request. Subsequent calls for the
// same language return the SAME pointer.
//
// When the requested language has no embedded directory (either entirely
// unknown or known to vo but not yet bundled) the returned error is a
// *domerr.LanguageUnsupportedError carrying the verbatim id and the
// alphabetically-sorted list of supported ids.
func (r *Registry) LoadLanguageQueries(ctx context.Context, lang vo.Language) (*model.LanguageQuerySet, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if cached, ok := r.cache[lang]; ok {
		return cached, nil
	}
	dir := string(lang)
	if dir == "" || strings.ContainsAny(dir, `/\`) || strings.Contains(dir, "..") {
		return nil, &domerr.LanguageUnsupportedError{
			Language:        dir,
			SupportedSorted: r.supportedAsStrings(),
		}
	}
	if !r.isSupported(lang) {
		return nil, &domerr.LanguageUnsupportedError{
			Language:        dir,
			SupportedSorted: r.supportedAsStrings(),
		}
	}
	entries, err := fs.ReadDir(bundledFS, dir)
	if err != nil {
		// Defensive: the supported list was computed from the embed root so
		// this should not happen in practice. Surface as unsupported for
		// safety so we never panic on a corrupt embed.
		return nil, &domerr.LanguageUnsupportedError{
			Language:        dir,
			SupportedSorted: r.supportedAsStrings(),
		}
	}
	queries := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".scm") {
			continue
		}
		body, readErr := fs.ReadFile(bundledFS, dir+"/"+name)
		if readErr != nil {
			return nil, fmt.Errorf("read bundled query %q for language %q: %w",
				name, lang, readErr)
		}
		queries[name] = body
	}
	set := &model.LanguageQuerySet{
		Language: lang,
		Queries:  queries,
	}
	r.cache[lang] = set
	return set, nil
}

// ListSupportedLanguages implements parser.ListSupportedLanguagesPort.
// Returns a fresh copy of the cached sorted list so callers cannot mutate
// internal state.
func (r *Registry) ListSupportedLanguages(ctx context.Context) ([]vo.Language, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]vo.Language, len(r.supported))
	copy(out, r.supported)
	return out, nil
}

func (r *Registry) isSupported(lang vo.Language) bool {
	return slices.Contains(r.supported, lang)
}

func (r *Registry) supportedAsStrings() []string {
	out := make([]string, len(r.supported))
	for i, s := range r.supported {
		out[i] = string(s)
	}
	return out
}

// Compile-time port assertions.
var (
	_ parserport.LoadLanguageQueriesPort    = (*Registry)(nil)
	_ parserport.ListSupportedLanguagesPort = (*Registry)(nil)
)
