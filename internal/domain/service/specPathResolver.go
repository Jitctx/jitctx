package service

import (
	"errors"
	"path/filepath"
	"strings"
)

// SpecPathResolver computes the on-disk target path for a feature spec
// given the feature name and the base directory. It is a pure function
// (no I/O, no deps) so it can be unit-tested without touching the
// filesystem and reused across read/write paths in later epics.
type SpecPathResolver struct{}

// NewSpecPathResolver returns a new SpecPathResolver value.
func NewSpecPathResolver() SpecPathResolver { return SpecPathResolver{} }

// Resolve validates feature (non-empty, no path separators, not "." or "..") and
// returns filepath.Join(baseDir, feature+".md"). Returns an error wrapping
// the underlying validation message; the caller wraps with context.
func (SpecPathResolver) Resolve(feature, baseDir string) (string, error) {
	f := strings.TrimSpace(feature)
	if f == "" {
		return "", errors.New("feature name must not be empty")
	}
	if strings.ContainsAny(f, `/\`) || f == "." || f == ".." {
		return "", errors.New("feature name must not contain path separators")
	}
	if baseDir == "" {
		return "", errors.New("base dir must not be empty")
	}
	return filepath.Join(baseDir, f+".md"), nil
}
