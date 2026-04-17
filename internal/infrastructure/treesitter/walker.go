package treesitter

import (
	"context"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Walker implements WalkJavaFilesPort.
type Walker struct{}

// NewWalker creates a new Walker.
func NewWalker() *Walker {
	return &Walker{}
}

// WalkJavaFiles returns all *.java files under src/main/java/ in sorted order.
// Skips src/test/, hidden directories, and target/ / build/ directories.
func (w *Walker) WalkJavaFiles(ctx context.Context, fsys fs.FS) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	const javaRoot = "src/main/java"

	// Check if the java root exists.
	_, err := fs.Stat(fsys, javaRoot)
	if err != nil {
		// If javaRoot doesn't exist, return empty list (not an error).
		return nil, nil
	}

	var files []string
	err = fs.WalkDir(fsys, javaRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		name := d.Name()

		// Skip hidden directories.
		if d.IsDir() && strings.HasPrefix(name, ".") {
			return fs.SkipDir
		}
		// Skip build artifact directories.
		if d.IsDir() && (name == "target" || name == "build") {
			return fs.SkipDir
		}

		if !d.IsDir() && strings.HasSuffix(name, ".java") {
			// Normalize to forward slashes.
			files = append(files, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}
