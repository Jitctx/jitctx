package fsspec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/port/spec"
)

// Compile-time check that Finder satisfies FindSpecFilePort.
var _ spec.FindSpecFilePort = (*Finder)(nil)

// Finder resolves a feature spec file from the conventional search locations
// defined by EP02RF-008.
type Finder struct{}

// NewFinder returns a stateless Finder.
func NewFinder() *Finder { return &Finder{} }

// Find searches for <feature>.md in the priority-ordered candidate list:
//  1. <baseDir>/jitctx-plans/<feature>.md
//  2. <baseDir>/.jitctx/plans/<feature>.md
//  3. <plansDir>/<feature>.md (if plansDir != ""; resolved relative to
//     baseDir when not absolute)
//
// Duplicate candidate paths are deduplicated while preserving order.
// Returns the path of the first match, its contents, any additional
// matched paths (alts), and an error. When no candidate exists,
// the error wraps *domerr.SpecFileNotFoundError.
func (f *Finder) Find(ctx context.Context, feature, baseDir, plansDir string) (string, []byte, []string, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, nil, err
	}

	// Build candidates in priority order.
	rawCandidates := []string{
		filepath.Join(baseDir, "jitctx-plans", feature+".md"),
		filepath.Join(baseDir, ".jitctx", "plans", feature+".md"),
	}
	if plansDir != "" {
		p := plansDir
		if !filepath.IsAbs(p) {
			p = filepath.Join(baseDir, p)
		}
		rawCandidates = append(rawCandidates, filepath.Join(p, feature+".md"))
	}

	// Deduplicate while preserving order.
	seen := make(map[string]bool, len(rawCandidates))
	candidates := make([]string, 0, len(rawCandidates))
	for _, c := range rawCandidates {
		if !seen[c] {
			seen[c] = true
			candidates = append(candidates, c)
		}
	}

	// Collect all candidates that exist.
	matched := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			matched = append(matched, c)
		}
	}

	if len(matched) == 0 {
		return "", nil, nil, &domerr.SpecFileNotFoundError{Feature: feature, Searched: candidates}
	}

	content, err := os.ReadFile(matched[0])
	if err != nil {
		return "", nil, nil, fmt.Errorf("read spec file %s: %w", matched[0], err)
	}

	return matched[0], content, matched[1:], nil
}
