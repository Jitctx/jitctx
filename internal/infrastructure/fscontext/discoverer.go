package fscontext

import (
	"context"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// contextSubdirs are the .jitctx subdirectories we discover.
var contextSubdirs = []string{"guidelines", "requirements", "scenarios", "contracts"}

// Discoverer implements DiscoverContextsPort and ReadContextBodyPort.
type Discoverer struct{}

// New creates a new Discoverer.
func New() *Discoverer {
	return &Discoverer{}
}

// DiscoverContexts walks .jitctx/{guidelines,requirements,scenarios,contracts}/**/*.md
// and returns model.Context entries without token_estimate populated.
func (d *Discoverer) DiscoverContexts(ctx context.Context, fsys fs.FS) ([]model.Context, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var contexts []model.Context

	for _, subdir := range contextSubdirs {
		root := filepath.ToSlash(filepath.Join(".jitctx", subdir))

		// Check if directory exists.
		if _, err := fs.Stat(fsys, root); err != nil {
			continue
		}

		// Collect and sort entries for determinism.
		var mdFiles []string
		err := fs.WalkDir(fsys, root, func(path string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				mdFiles = append(mdFiles, filepath.ToSlash(path))
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		sort.Strings(mdFiles)

		for _, path := range mdFiles {
			data, err := fs.ReadFile(fsys, path)
			if err != nil {
				continue
			}
			fm, _, hasFM, _ := parseFrontMatter(data)
			c := mapToContext(path, subdir, fm, hasFM)
			contexts = append(contexts, c)
		}
	}

	return contexts, nil
}

// ReadContextBody returns the markdown body of the file at path with front matter stripped.
func (d *Discoverer) ReadContextBody(ctx context.Context, fsys fs.FS, path string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return "", err
	}

	_, body, _, _ := parseFrontMatter(data)
	return body, nil
}
