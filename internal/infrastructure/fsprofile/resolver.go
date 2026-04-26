package fsprofile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	profileport "github.com/jitctx/jitctx/internal/domain/port/profile"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// Resolver implements profile.ResolveProfilePort. It composes a BundleLoader
// (for user-dir loads) and a Bundled adapter (for embed loads) to implement
// the EP04RF-012 resolution cascade.
type Resolver struct {
	loader  *BundleLoader
	bundled *Bundled
	logger  *slog.Logger
}

// NewResolver constructs a Resolver. When logger is nil, slog.Default() is used.
func NewResolver(loader *BundleLoader, bundled *Bundled, logger *slog.Logger) *Resolver {
	if logger == nil {
		logger = slog.Default()
	}
	return &Resolver{loader: loader, bundled: bundled, logger: logger}
}

// Resolve implements profile.ResolveProfilePort.
//
// Cascade:
//   - Input.Name != "":
//     try <WorkDir>/<ProfilesDir>/<Name>/ via loader.LoadBundle(Dir);
//     on success, return (Source == ProfileSourceCustom).
//     try bundled.LoadBundled(Name);
//     on success, return (Source == ProfileSourceBundled).
//     return *UnknownBundledProfileError{Name, Available: bundled.ListBundled()}.
//   - Input.Name == "":
//     walk <WorkDir>/<ProfilesDir>/ for child directories;
//     for each, try loader.LoadBundle(Dir) — return the first that succeeds
//     (Source == ProfileSourceCustom).
//     walk bundled.ListBundled() — try each; return the first that loads
//     (Source == ProfileSourceBundled).
//     return ErrNoProfileMatch.
//
// A user-dir that fails to load in the AUTO branch is logged at WARN and
// skipped. In the EXPLICIT branch (Name != "") a load failure is returned
// directly to the caller.
func (r *Resolver) Resolve(ctx context.Context, in profilevo.ResolveProfileInput) (*model.ProfileBundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	workDir := in.WorkDir
	if workDir == "" {
		workDir = "."
	}

	profilesDir := filepath.Join(workDir, in.ProfilesDir)
	absProfilesDir, err := filepath.Abs(profilesDir)
	if err != nil {
		return nil, fmt.Errorf("resolve profiles dir: %w", err)
	}

	if in.Name != "" {
		return r.resolveExplicit(ctx, in.Name, absProfilesDir)
	}
	return r.resolveAuto(ctx, absProfilesDir)
}

// resolveExplicit handles the Input.Name != "" branch.
func (r *Resolver) resolveExplicit(ctx context.Context, name, absProfilesDir string) (*model.ProfileBundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// 1. Try the user directory.
	userDir := filepath.Join(absProfilesDir, name)
	if _, err := os.Stat(userDir); err == nil {
		bundle, err := r.loader.LoadBundle(ctx, profilevo.LoadProfileBundleInput{Dir: userDir})
		if err != nil {
			// Explicit request: surface the load error — the user asked for this profile.
			return nil, fmt.Errorf("load user profile %q from %s: %w", name, userDir, err)
		}
		return bundle, nil
	}

	// 2. Try the bundled embed.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bundle, err := r.bundled.LoadBundled(ctx, name)
	if err == nil {
		return bundle, nil
	}
	if !errors.Is(err, domerr.ErrBundledProfileNotFound) {
		return nil, fmt.Errorf("load bundled profile %q: %w", name, err)
	}

	// 3. Neither source found — build the typed error with available names.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	available, listErr := r.bundled.ListBundled(ctx)
	if listErr != nil {
		return nil, fmt.Errorf("list bundled profiles: %w", listErr)
	}
	sort.Strings(available)
	return nil, &domerr.UnknownBundledProfileError{
		Name:      name,
		Available: available,
	}
}

// resolveAuto handles the Input.Name == "" branch.
func (r *Resolver) resolveAuto(ctx context.Context, absProfilesDir string) (*model.ProfileBundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// 1. Walk user directories alphabetically; return the first clean load.
	entries, err := os.ReadDir(absProfilesDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read profiles dir %s: %w", absProfilesDir, err)
	}

	// Build sorted list of sub-directory names.
	var dirNames []string
	for _, e := range entries {
		if e.IsDir() {
			dirNames = append(dirNames, e.Name())
		}
	}
	sort.Strings(dirNames)

	for _, name := range dirNames {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		userDir := filepath.Join(absProfilesDir, name)
		bundle, loadErr := r.loader.LoadBundle(ctx, profilevo.LoadProfileBundleInput{Dir: userDir})
		if loadErr != nil {
			r.logger.Warn("resolver: skipping malformed user-dir profile",
				"dir", userDir,
				"reason", loadErr,
			)
			continue
		}
		return bundle, nil
	}

	// 2. Walk bundled profiles alphabetically; return the first clean load.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bundledNames, err := r.bundled.ListBundled(ctx)
	if err != nil {
		return nil, fmt.Errorf("list bundled profiles: %w", err)
	}
	// ListBundled already returns sorted names, but sort again for safety.
	sort.Strings(bundledNames)

	for _, name := range bundledNames {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		bundle, loadErr := r.bundled.LoadBundled(ctx, name)
		if loadErr != nil {
			r.logger.Warn("resolver: skipping malformed bundled profile",
				"name", name,
				"reason", loadErr,
			)
			continue
		}
		return bundle, nil
	}

	// 3. Nothing found.
	return nil, domerr.ErrNoProfileMatch
}

// Compile-time assertion.
var _ profileport.ResolveProfilePort = (*Resolver)(nil)
