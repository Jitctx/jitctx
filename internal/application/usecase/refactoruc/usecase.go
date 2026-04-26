package refactoruc

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"sort"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	gitport "github.com/jitctx/jitctx/internal/domain/port/git"
	"github.com/jitctx/jitctx/internal/domain/port/manifest"
	"github.com/jitctx/jitctx/internal/domain/port/parser"
	"github.com/jitctx/jitctx/internal/domain/service"
	refactorvo "github.com/jitctx/jitctx/internal/domain/vo/refactor"
)

// Impl satisfies refactoruc.UseCase.
// Read-only by construction: zero writer ports (RNF-002).
type Impl struct {
	manifests    manifest.LoadManifestPort
	walker       parser.WalkJavaFilesPort
	comments     parser.ListJavaCommentsPort
	gitFileMTime gitport.FileLastModifiedTimePort
	gitLineMTime gitport.LineIntroducedTimePort
	markerParse  *service.MarkerParser
	logger       *slog.Logger
}

// New creates a new refactoruc.Impl with all required ports injected.
// The constructor accepts only read-shaped ports (RNF-002 read-only enforcement).
func New(
	manifests manifest.LoadManifestPort,
	walker parser.WalkJavaFilesPort,
	comments parser.ListJavaCommentsPort,
	gitFileMTime gitport.FileLastModifiedTimePort,
	gitLineMTime gitport.LineIntroducedTimePort,
	markerParse *service.MarkerParser,
	logger *slog.Logger,
) *Impl {
	return &Impl{
		manifests:    manifests,
		walker:       walker,
		comments:     comments,
		gitFileMTime: gitFileMTime,
		gitLineMTime: gitLineMTime,
		markerParse:  markerParse,
		logger:       logger,
	}
}

// Execute runs the refactor marker scan workflow.
//
// Orchestration:
//  1. ctx.Err() guard on entry.
//  2. Build fs.FS rooted at WorkDir.
//  3. Load manifest; tolerant of ErrManifestNotFound — proceeds with empty
//     module list and ManifestPresent=false. Other errors abort.
//  4. Walk Java files via WalkJavaFilesPort.
//  5. For each file, list comments via ListJavaCommentsPort. On ErrPartialParse
//     log a warning and continue with whatever comments came back.
//  6. For each comment, call MarkerParser.Parse. Unmatched comments are dropped.
//     Matched comments are enriched with a moduleID resolved via
//     service.ResolveModuleByPath.
//  7. Sort markers by (ModuleID, FilePath, Line, Type, Description) with
//     "<unmoduled>" last (RNF-003).
//     7a. Stale detection: probe git availability via the file-mtime port on
//     the first marker's path; if ErrGitUnavailable, set StaleSkipped=true
//     and skip per-marker loop. Otherwise iterate every marker and set
//     Stale=true when fileMTime is after lineMTime. Per-marker errors are
//     non-fatal (log debug, leave Stale=false).
//  8. Dedupe and sort UnknownTypes for deterministic stderr emission.
//  9. Return ScanRefactorsOutput.
func (u *Impl) Execute(ctx context.Context, in refactorvo.ScanRefactorsInput) (refactorvo.ScanRefactorsOutput, error) {
	// Step 1: honour context cancellation on entry.
	if err := ctx.Err(); err != nil {
		return refactorvo.ScanRefactorsOutput{}, err
	}

	// Step 2: build read-only fs.FS rooted at WorkDir.
	var fsys fs.FS
	if in.WorkDir == "" || in.WorkDir == "." {
		fsys = os.DirFS(".")
	} else {
		fsys = os.DirFS(in.WorkDir)
	}

	// Step 3: load manifest; tolerant of absence.
	manifestPresent := false

	state, err := u.manifests.Load(ctx)
	if err != nil {
		if errors.Is(err, domerr.ErrManifestNotFound) {
			u.logger.Info("manifest not found, markers will be grouped under <unmoduled>")
			// state remains nil; manifestPresent stays false.
		} else {
			return refactorvo.ScanRefactorsOutput{}, fmt.Errorf("scan refactors: load manifest: %w", err)
		}
	} else {
		manifestPresent = true
	}

	// Step 4: walk Java files.
	paths, err := u.walker.WalkJavaFiles(ctx, fsys)
	if err != nil {
		return refactorvo.ScanRefactorsOutput{}, fmt.Errorf("scan refactors: walk: %w", err)
	}

	var markers []refactorvo.RefactorMarker
	unknownTypeSet := make(map[string]struct{})

	// Step 5 & 6: per-file comment extraction and marker parsing.
	for _, p := range paths {
		// Cancellation guard inside the loop.
		if err := ctx.Err(); err != nil {
			return refactorvo.ScanRefactorsOutput{}, err
		}

		comments, commErr := u.comments.ListJavaComments(ctx, fsys, p)
		if commErr != nil {
			if errors.Is(commErr, domerr.ErrPartialParse) {
				u.logger.Warn("scan refactors: partial parse, using partial results", "path", p)
				// continue with whatever comments came back
			} else {
				return refactorvo.ScanRefactorsOutput{}, fmt.Errorf("scan refactors: list comments %s: %w", p, commErr)
			}
		}

		// Resolve moduleID for this file.
		var moduleID string
		if manifestPresent && state != nil {
			moduleID = service.ResolveModuleByPath(state.Modules, p)
		} else {
			moduleID = "<unmoduled>"
		}

		for _, c := range comments {
			result := u.markerParse.Parse(p, c.Line, c.Kind, c.Text)
			if !result.Matched {
				continue
			}
			result.Marker.ModuleID = moduleID
			markers = append(markers, result.Marker)

			if result.UnknownType != "" {
				unknownTypeSet[result.UnknownType] = struct{}{}
			}

			u.logger.Debug("scan refactors: marker detected",
				"path", p,
				"line", c.Line,
				"type", result.Marker.Type,
			)
		}
	}

	// Step 7: sort markers deterministically (RNF-003).
	sort.SliceStable(markers, func(i, j int) bool {
		return lessMarker(markers[i], markers[j])
	})

	// Step 7a: stale detection (EP03RF-009).
	// Resolve absolute repoRoot for git invocations.
	staleSkipped := false
	repoRoot := in.WorkDir
	if repoRoot == "" || repoRoot == "." {
		if cwd, err := os.Getwd(); err == nil {
			repoRoot = cwd
		}
	}

	if len(markers) > 0 {
		// Probe git availability using the first marker's file path.
		_, probeErr := u.gitFileMTime.Get(ctx, repoRoot, markers[0].FilePath)
		if errors.Is(probeErr, domerr.ErrGitUnavailable) {
			staleSkipped = true
			u.logger.Info("scan refactors: git not available; stale detection skipped")
		} else {
			// Per-marker stale detection; per-marker errors are non-fatal.
			for i := range markers {
				if err := ctx.Err(); err != nil {
					return refactorvo.ScanRefactorsOutput{}, err
				}
				fileMT, ferr := u.gitFileMTime.Get(ctx, repoRoot, markers[i].FilePath)
				if ferr != nil {
					u.logger.Debug("scan refactors: file mtime unavailable",
						"path", markers[i].FilePath, "err", ferr)
					continue
				}
				lineMT, lerr := u.gitLineMTime.Get(ctx, repoRoot, markers[i].FilePath, markers[i].Line)
				if lerr != nil {
					u.logger.Debug("scan refactors: line mtime unavailable",
						"path", markers[i].FilePath, "line", markers[i].Line, "err", lerr)
					continue
				}
				if fileMT.After(lineMT) {
					markers[i].Stale = true
				}
			}
		}
	}

	// Step 8: build deduped, sorted unknownTypes slice.
	unknownTypes := make([]string, 0, len(unknownTypeSet))
	for k := range unknownTypeSet {
		unknownTypes = append(unknownTypes, k)
	}
	sort.Strings(unknownTypes)

	return refactorvo.ScanRefactorsOutput{
		Markers:         markers,
		UnknownTypes:    unknownTypes,
		ManifestPresent: manifestPresent,
		StaleSkipped:    staleSkipped,
	}, nil
}

// lessMarker defines the deterministic sort order (RNF-003):
// (ModuleID, FilePath, Line, Type, Description) with "<unmoduled>" last.
func lessMarker(a, b refactorvo.RefactorMarker) bool {
	if a.ModuleID != b.ModuleID {
		if a.ModuleID == "<unmoduled>" {
			return false
		}
		if b.ModuleID == "<unmoduled>" {
			return true
		}
		return a.ModuleID < b.ModuleID
	}
	if a.FilePath != b.FilePath {
		return a.FilePath < b.FilePath
	}
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	if a.Type != b.Type {
		return string(a.Type) < string(b.Type)
	}
	return a.Description < b.Description
}
