package fsprofile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	parserport "github.com/jitctx/jitctx/internal/domain/port/parser"
	profileport "github.com/jitctx/jitctx/internal/domain/port/profile"
	"github.com/jitctx/jitctx/internal/domain/vo"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// BundleLoader implements profile.LoadProfileBundlePort. It loads a profile
// from a directory layout (profile.yaml + templates/ subdirectory) into a
// *model.ProfileBundle. For bundled-embed loads it delegates to a sub-FS
// rooted at the named bundled directory inside fsprofile/bundled.go.
//
// EP04US-005: when a profile.yaml declares a non-empty `language:` value,
// the loader resolves it through the injected LoadLanguageQueriesPort and
// attaches the resulting *model.LanguageQuerySet to ProfileBundle.LanguageQueries.
// The languageQueries dependency is nil-tolerant — when nil the loader skips
// query resolution. Production wiring always passes a non-nil registry; tests
// that don't care about queries pass nil.
type BundleLoader struct {
	logger          *slog.Logger
	languageQueries parserport.LoadLanguageQueriesPort
}

// NewBundleLoader returns a BundleLoader. When logger is nil, slog.Default()
// is used. When languageQueries is nil, the loader skips bundled-query
// resolution (legacy code paths and tests that don't care about queries
// continue to work).
func NewBundleLoader(logger *slog.Logger, languageQueries parserport.LoadLanguageQueriesPort) *BundleLoader {
	if logger == nil {
		logger = slog.Default()
	}
	return &BundleLoader{logger: logger, languageQueries: languageQueries}
}

// LoadBundle implements profile.LoadProfileBundlePort.
func (l *BundleLoader) LoadBundle(ctx context.Context, in profilevo.LoadProfileBundleInput) (*model.ProfileBundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if in.Dir == "" && in.BundledName == "" {
		return nil, fmt.Errorf("LoadBundle: input has neither Dir nor BundledName: %w", domerr.ErrProfileInvalid)
	}
	if in.Dir != "" {
		return l.loadFromOS(ctx, in.Dir)
	}
	return l.loadFromBundled(ctx, in.BundledName)
}

// loadFromOS opens dir as a normal os.DirFS and delegates to loadFromFS.
// Source is set to ProfileSourceCustom.
func (l *BundleLoader) loadFromOS(ctx context.Context, dir string) (*model.ProfileBundle, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve profile dir %q: %w", dir, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat profile dir %q: %w", abs, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("profile path %q is not a directory: %w", abs, domerr.ErrProfileInvalid)
	}
	bundle, dto, err := loadFromFS(os.DirFS(abs), l.logger)
	if err != nil {
		return nil, fmt.Errorf("profile bundle %q: %w", abs, err)
	}
	bundle.Dir = abs
	bundle.Profile.Source = model.ProfileSourceCustom
	if err := l.attachLanguageQueries(ctx, dto, bundle); err != nil {
		return nil, fmt.Errorf("profile bundle %q: %w", abs, err)
	}
	return bundle, nil
}

// loadFromBundled is the embed-backed branch — it mirrors Bundled.LoadBundled
// but threads the bundleDTO through so attachLanguageQueries can use the
// verbatim profile.yaml `language:` value.
func (l *BundleLoader) loadFromBundled(ctx context.Context, name string) (*model.ProfileBundle, error) {
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return nil, fmt.Errorf("bundled profile %q: invalid name: %w", name, domerr.ErrProfileInvalid)
	}
	sub, err := fs.Sub(bundledFS, "bundled/"+name)
	if err != nil {
		return nil, fmt.Errorf("bundled %q: %w", name, domerr.ErrBundledProfileNotFound)
	}
	if _, err := fs.Stat(sub, "profile.yaml"); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("bundled %q: %w", name, domerr.ErrBundledProfileNotFound)
		}
		return nil, fmt.Errorf("bundled %q: stat profile.yaml: %w", name, err)
	}
	bundle, dto, err := loadFromFS(sub, l.logger)
	if err != nil {
		return nil, fmt.Errorf("bundled %q: %w", name, err)
	}
	bundle.Profile.Source = model.ProfileSourceBundled
	bundle.Dir = ""
	if err := l.attachLanguageQueries(ctx, dto, bundle); err != nil {
		return nil, fmt.Errorf("bundled %q: %w", name, err)
	}
	return bundle, nil
}

// attachLanguageQueries wires bundle.LanguageQueries when:
//   - the loader has a non-nil registry (production wiring),
//   - and the profile.yaml declared a non-empty `language:` value.
//
// On registry failure (typically a *domerr.LanguageUnsupportedError) the
// error propagates verbatim to the caller. On success the resolved set is
// attached and bundle.Profile.Language is overwritten with the canonical
// (lowercase, vo-validated) id.
func (l *BundleLoader) attachLanguageQueries(ctx context.Context, dto bundleDTO, bundle *model.ProfileBundle) error {
	if l.languageQueries == nil || dto.Language == "" {
		return nil
	}
	set, err := l.languageQueries.LoadLanguageQueries(ctx, vo.Language(dto.Language))
	if err != nil {
		return err
	}
	bundle.LanguageQueries = set
	bundle.Profile.Language = set.Language
	return nil
}

// loadFromFS is the shared core used by both BundleLoader (OS-backed fs.FS)
// and the bundled embed sub-filesystem path. The caller sets Dir and Source
// after this returns. logger is forwarded to toBundleDomain for WARN entries;
// when nil, slog.Default() is used.
//
// Returns (bundle, dto, err) so callers can drive the EP04US-005 language
// query attachment (the dto carries the verbatim `language:` string).
func loadFromFS(fsys fs.FS, logger *slog.Logger) (*model.ProfileBundle, bundleDTO, error) {
	// Step 1: read profile.yaml.
	data, err := fs.ReadFile(fsys, "profile.yaml")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, bundleDTO{}, domerr.ErrProfileYamlMissing
		}
		return nil, bundleDTO{}, fmt.Errorf("read profile.yaml: %w", err)
	}

	// Step 2: decode into bundleDTO with KnownFields(false) — intentional
	// relaxed mode for US-001 (future fields present in bundled stubs).
	var dto bundleDTO
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(false)
	if err := dec.Decode(&dto); err != nil {
		return nil, bundleDTO{}, fmt.Errorf("decode profile.yaml: %w", domerr.ErrProfileInvalid)
	}

	// Step 3: eagerly load templates.
	templates := make(map[string][]byte)
	entries, err := fs.ReadDir(fsys, "templates")
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, bundleDTO{}, fmt.Errorf("read templates dir: %w", err)
		}
		// Missing templates/ directory is tolerated (empty map).
	} else {
		// Sorted iteration for deterministic behaviour.
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			names = append(names, entry.Name())
		}
		sort.Strings(names)
		for _, name := range names {
			body, readErr := fs.ReadFile(fsys, "templates/"+name)
			if readErr != nil {
				return nil, bundleDTO{}, fmt.Errorf("read template %q: %w", name, readErr)
			}
			templates[name] = body
		}
	}

	// Step 4: assemble the bundle via the pure mapper (validates types, packaging).
	bundle, err := toBundleDomain(dto, templates, logger)
	if err != nil {
		return nil, bundleDTO{}, err
	}

	return bundle, dto, nil
}

// Compile-time assertions — kept here alongside the implementation they verify.
var (
	_ profileport.LoadProfileBundlePort = (*BundleLoader)(nil)
)
