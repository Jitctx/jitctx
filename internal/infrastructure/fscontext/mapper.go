package fscontext

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/vo"
)

var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9]+`)

// sanitizeID converts a string to a lowercase kebab-case ID.
func sanitizeID(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumRe.ReplaceAllString(s, "-")
	// Collapse runs of dashes.
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

// contextTypeFromDir derives the ArtifactType from the .jitctx subdirectory name.
func contextTypeFromDir(subdir string) vo.ArtifactType {
	switch subdir {
	case "guidelines":
		return vo.ArtifactGuidelines
	case "requirements":
		return vo.ArtifactRequirements
	case "scenarios":
		return vo.ArtifactScenarios
	case "contracts":
		return vo.ArtifactContracts
	}
	return vo.ArtifactGuidelines
}

// mapToContext converts file metadata to a model.Context (without token_estimate).
func mapToContext(filePath string, contextSubdir string, fm frontMatterResult, hasFrontMatter bool) model.Context {
	stem := strings.TrimSuffix(filepath.Base(filePath), ".md")
	parentDir := filepath.Base(filepath.Dir(filePath))

	// Derive ID: from front matter if present, else from filename.
	id := fm.ID
	if id == "" {
		id = sanitizeID(stem)
	}

	// Derive tags.
	var tags []string
	if hasFrontMatter && len(fm.Tags) > 0 {
		tags = append(tags, fm.Tags...)
	} else {
		// Infer tags from path: [parent_dir_basename, sanitized_filename_stem].
		t1 := sanitizeID(parentDir)
		t2 := sanitizeID(stem)
		if t1 != "" && t1 != id {
			tags = append(tags, t1)
		}
		if t2 != "" {
			tags = append(tags, t2)
		}
	}

	// AppliesTo.
	var appliesTo []string
	if hasFrontMatter {
		appliesTo = fm.AppliesTo
	}

	// Module.
	module := ""
	if hasFrontMatter {
		module = fm.Module
	}

	return model.Context{
		ID:        id,
		Type:      contextTypeFromDir(contextSubdir),
		AppliesTo: appliesTo,
		Module:    module,
		Tags:      tags,
		Path:      filepath.ToSlash(filePath),
	}
}
