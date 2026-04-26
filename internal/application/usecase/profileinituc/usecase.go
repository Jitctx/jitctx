package profileinituc

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	profileport "github.com/jitctx/jitctx/internal/domain/port/profile"
	profileinitucport "github.com/jitctx/jitctx/internal/domain/usecase/profileinituc"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// Impl orchestrates "jitctx profile init <name>" per the contract in
// internal/domain/usecase/profileinituc/port.go.
type Impl struct {
	bundled   profileport.ListBundledProfilesPort
	extractor profileport.ExtractBundledProfilePort
	logger    *slog.Logger
}

// New constructs a profileinituc.Impl. All parameters are required except
// logger — a nil logger falls back to slog.Default().
func New(
	bundled profileport.ListBundledProfilesPort,
	extractor profileport.ExtractBundledProfilePort,
	logger *slog.Logger,
) *Impl {
	if logger == nil {
		logger = slog.Default()
	}
	return &Impl{bundled: bundled, extractor: extractor, logger: logger}
}

// Execute implements profileinituc.UseCase.
func (u *Impl) Execute(ctx context.Context, in profilevo.ProfileInitInput) (profilevo.ProfileInitOutput, error) {
	if err := ctx.Err(); err != nil {
		return profilevo.ProfileInitOutput{}, err
	}

	// Step 1 — validate name.
	if in.Name == "" {
		return profilevo.ProfileInitOutput{}, fmt.Errorf(
			"profile init: name is required: %w", domerr.ErrProfileInvalid)
	}
	if strings.ContainsAny(in.Name, `/\`) || strings.Contains(in.Name, "..") {
		return profilevo.ProfileInitOutput{}, fmt.Errorf(
			"profile init: invalid name %q: %w", in.Name, domerr.ErrProfileInvalid)
	}

	// Step 2 — verify bundled name exists.
	available, err := u.bundled.ListBundled(ctx)
	if err != nil {
		return profilevo.ProfileInitOutput{}, fmt.Errorf("profile init: list bundled: %w", err)
	}
	if !contains(available, in.Name) {
		sort.Strings(available)
		return profilevo.ProfileInitOutput{}, &domerr.UnknownBundledProfileError{
			Name:      in.Name,
			Available: available,
		}
	}

	// Step 3 — resolve target dir.
	workDir := in.WorkDir
	if workDir == "" {
		workDir = "."
	}
	profilesDir := in.ProfilesDir
	if profilesDir == "" {
		profilesDir = ".jitctx/profiles"
	}
	target := filepath.Join(workDir, profilesDir, in.Name)
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return profilevo.ProfileInitOutput{}, fmt.Errorf("profile init: resolve target: %w", err)
	}

	// Step 4 — extract (idempotent existence check is inside Extractor).
	if err := u.extractor.Extract(ctx, in.Name, absTarget); err != nil {
		return profilevo.ProfileInitOutput{}, err
	}

	// Step 5 — count files written under target for the success message.
	n, err := countFiles(absTarget)
	if err != nil {
		// Non-fatal — extraction succeeded; we just cannot count.
		u.logger.Warn("profile init: count files failed", "target", absTarget, "reason", err)
		n = 0
	}

	u.logger.Info("profile initialised",
		"name", in.Name,
		"target", absTarget,
		"files_written", n,
	)

	return profilevo.ProfileInitOutput{
		Name:         in.Name,
		TargetDir:    absTarget,
		FilesWritten: n,
	}, nil
}

// contains is a tiny local helper kept private to the package.
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// countFiles walks dir and returns the count of regular files (excluding
// directories). Non-fatal — callers log at WARN on error.
func countFiles(dir string) (int, error) {
	var n int
	err := filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			n++
		}
		return nil
	})
	return n, err
}

// Compile-time assertion: *Impl must satisfy the domain UseCase interface.
var _ profileinitucport.UseCase = (*Impl)(nil)
