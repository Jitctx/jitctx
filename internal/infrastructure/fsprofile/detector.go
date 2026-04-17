package fsprofile

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
)

// Detector implements DetectProfilePort.
type Detector struct {
	userDir string
	logger  *slog.Logger
}

// NewDetector creates a Detector that looks for custom profiles in userDir.
func NewDetector(userDir string) *Detector {
	return &Detector{userDir: userDir, logger: slog.Default()}
}

// NewDetectorWithLogger creates a Detector with a custom logger.
func NewDetectorWithLogger(userDir string, logger *slog.Logger) *Detector {
	return &Detector{userDir: userDir, logger: logger}
}

// Detect picks the first matching profile (custom profiles first, then bundled).
func (d *Detector) Detect(ctx context.Context, fsys fs.FS) (*model.FrameworkProfile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Load and evaluate custom profiles first.
	customProfiles := d.loadCustomProfiles()
	for _, prof := range customProfiles {
		if profileMatches(prof, fsys) {
			return prof, nil
		}
	}

	// Evaluate bundled profiles.
	bundledProfiles := d.loadBundledProfiles()
	for _, prof := range bundledProfiles {
		if profileMatches(prof, fsys) {
			return prof, nil
		}
	}

	return nil, domerr.ErrNoProfileMatch
}

func (d *Detector) loadCustomProfiles() []*model.FrameworkProfile {
	var profiles []*model.FrameworkProfile
	entries, err := os.ReadDir(d.userDir)
	if err != nil {
		return nil // directory may not exist
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		path := filepath.Join(d.userDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			d.logger.Warn("cannot read custom profile", "file", path, "reason", err)
			continue
		}
		prof, err := decodeProfile(data, true)
		if err != nil {
			d.logger.Warn("custom profile parse error", "file", path, "reason", err)
			continue
		}
		profiles = append(profiles, prof)
	}
	return profiles
}

func (d *Detector) loadBundledProfiles() []*model.FrameworkProfile {
	var profiles []*model.FrameworkProfile
	entries, err := fs.ReadDir(embeddedProfiles, "bundled")
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := embeddedProfiles.ReadFile("bundled/" + e.Name())
		if err != nil {
			continue
		}
		prof, err := decodeProfile(data, true)
		if err != nil {
			continue
		}
		profiles = append(profiles, prof)
	}
	return profiles
}

// profileMatches returns true if any of the profile's file matchers match the fsys.
func profileMatches(prof *model.FrameworkProfile, fsys fs.FS) bool {
	for _, matcher := range prof.Detect.Files {
		data, err := fs.ReadFile(fsys, matcher.Name)
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(string(data)), strings.ToLower(matcher.Contains)) {
			return true
		}
	}
	return false
}
