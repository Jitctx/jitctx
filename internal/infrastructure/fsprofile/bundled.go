package fsprofile

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	profileport "github.com/jitctx/jitctx/internal/domain/port/profile"
)

//go:embed all:bundled
var bundledFS embed.FS

// Bundled satisfies profile.LoadBundledProfilePort and
// profile.ListBundledProfilesPort. It delegates the actual decoding to
// loadFromFS, passing an embed-backed sub-filesystem so the loader has no
// direct dependency on go:embed.
type Bundled struct{}

// NewBundled returns a Bundled adapter. Construction is cheap — no I/O
// happens until LoadBundled or ListBundled is called.
func NewBundled() *Bundled { return &Bundled{} }

// LoadBundled implements profile.LoadBundledProfilePort.
func (b *Bundled) LoadBundled(ctx context.Context, name string) (*model.ProfileBundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return nil, fmt.Errorf("bundled profile %q: invalid name: %w", name, domerr.ErrProfileInvalid)
	}
	sub, err := fs.Sub(bundledFS, "bundled/"+name)
	if err != nil {
		return nil, fmt.Errorf("bundled %q: %w", name, domerr.ErrBundledProfileNotFound)
	}
	// fs.Sub does not error when the subdirectory is absent; probe with Stat.
	if _, err := fs.Stat(sub, "profile.yaml"); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("bundled %q: %w", name, domerr.ErrBundledProfileNotFound)
		}
		return nil, fmt.Errorf("bundled %q: stat profile.yaml: %w", name, err)
	}
	bundle, err := loadFromFS(sub)
	if err != nil {
		return nil, fmt.Errorf("bundled %q: %w", name, err)
	}
	bundle.Profile.Source = model.ProfileSourceBundled
	bundle.Dir = "" // bundled profiles have no on-disk path
	return bundle, nil
}

// ListBundled implements profile.ListBundledProfilesPort.
func (b *Bundled) ListBundled(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := fs.ReadDir(bundledFS, "bundled")
	if err != nil {
		return nil, fmt.Errorf("read bundled root: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// Compile-time port assertions.
var (
	_ profileport.LoadBundledProfilePort  = (*Bundled)(nil)
	_ profileport.ListBundledProfilesPort = (*Bundled)(nil)
)
