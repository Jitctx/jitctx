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

	"gopkg.in/yaml.v3"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// BundleLoader implements profile.LoadProfileBundlePort. It loads a profile
// from a directory layout (profile.yaml + templates/ subdirectory) into a
// *model.ProfileBundle. For bundled-embed loads it delegates to Bundled.
type BundleLoader struct {
	logger *slog.Logger
}

// NewBundleLoader returns a BundleLoader. When logger is nil, slog.Default() is used.
func NewBundleLoader(logger *slog.Logger) *BundleLoader {
	if logger == nil {
		logger = slog.Default()
	}
	return &BundleLoader{logger: logger}
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
	// Fallback to bundled embed via Bundled adapter.
	return NewBundled().LoadBundled(ctx, in.BundledName)
}

// loadFromOS opens dir as a normal os.DirFS and delegates to loadFromFS.
// Source is set to ProfileSourceCustom. ctx is reserved for future
// cancellation propagation into the eager template walk.
func (l *BundleLoader) loadFromOS(_ context.Context, dir string) (*model.ProfileBundle, error) {
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
	bundle, err := loadFromFS(os.DirFS(abs))
	if err != nil {
		return nil, fmt.Errorf("profile bundle %q: %w", abs, err)
	}
	bundle.Dir = abs
	bundle.Profile.Source = model.ProfileSourceCustom
	return bundle, nil
}

// loadFromFS is the shared core used by both BundleLoader (OS-backed fs.FS)
// and Bundled (embed.FS sub-filesystem). The caller sets Dir and Source after
// this returns.
func loadFromFS(fsys fs.FS) (*model.ProfileBundle, error) {
	// Step 1: read profile.yaml.
	data, err := fs.ReadFile(fsys, "profile.yaml")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, domerr.ErrProfileYamlMissing
		}
		return nil, fmt.Errorf("read profile.yaml: %w", err)
	}

	// Step 2: decode into bundleDTO with KnownFields(false) — intentional
	// relaxed mode for US-001 (future fields present in bundled stubs).
	var dto bundleDTO
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(false)
	if err := dec.Decode(&dto); err != nil {
		return nil, fmt.Errorf("decode profile.yaml: %w", domerr.ErrProfileInvalid)
	}

	// Step 3: eagerly load templates.
	templates := make(map[string][]byte)
	entries, err := fs.ReadDir(fsys, "templates")
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("read templates dir: %w", err)
		}
		// Missing templates/ directory is tolerated (empty map).
	} else {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			body, readErr := fs.ReadFile(fsys, "templates/"+name)
			if readErr != nil {
				return nil, fmt.Errorf("read template %q: %w", name, readErr)
			}
			templates[name] = body
		}
	}

	// Step 4: assemble the bundle via the pure mapper (validates types, packaging).
	bundle, err := toBundleDomain(dto, templates)
	if err != nil {
		return nil, err
	}

	return bundle, nil
}

// Compile-time assertions — kept here alongside the implementation they verify.
var (
	_ interface {
		LoadBundle(ctx context.Context, input profilevo.LoadProfileBundleInput) (*model.ProfileBundle, error)
	} = (*BundleLoader)(nil)
)
