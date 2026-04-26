package fsprofile

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	profileport "github.com/jitctx/jitctx/internal/domain/port/profile"
)

// Extractor implements profile.ExtractBundledProfilePort. It walks
// the embed.FS subtree at bundled/<name> and writes each file under
// <targetDir> using a tmp-rename atomic dance.
type Extractor struct{}

// NewExtractor returns an Extractor. Construction is cheap.
func NewExtractor() *Extractor { return &Extractor{} }

// Extract implements profile.ExtractBundledProfilePort.
//
// Algorithm:
//  1. fs.Sub(bundledFS, "bundled/"+name) → subFS.
//  2. fs.Stat(subFS, "profile.yaml") → if ErrNotExist, return
//     *UnknownBundledProfileError{Name: name, Available: list()}.
//  3. os.Stat(targetDir) → if exists, return *ProfileTargetExistsError.
//  4. Create a sibling tmp dir: <parent>/.jitctx-init-<name>-<rand>/.
//  5. defer os.RemoveAll(tmpDir) — only no-ops after a successful
//     rename because the rename-target is empty.
//  6. fs.WalkDir(subFS, ".") — for each entry: mkdir or copy bytes
//     verbatim with 0o644 perms (0o755 for directories).
//  7. os.Rename(tmpDir, targetDir).
//
// All filesystem syscalls wrap their error with fmt.Errorf("%s: %w",
// step, err). The function honours ctx.Err() before each major step.
func (e *Extractor) Extract(ctx context.Context, name string, targetDir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Step 1 — get sub-FS for the named bundled profile.
	subFS, err := fs.Sub(bundledFS, "bundled/"+name)
	if err != nil {
		available := bundledNames()
		return &domerr.UnknownBundledProfileError{Name: name, Available: available}
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// Step 2 — probe for profile.yaml to confirm the name is valid.
	if _, err := fs.Stat(subFS, "profile.yaml"); err != nil {
		available := bundledNames()
		return &domerr.UnknownBundledProfileError{Name: name, Available: available}
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// Step 3 — bail if the target already exists.
	if _, err := os.Stat(targetDir); err == nil {
		return &domerr.ProfileTargetExistsError{Target: targetDir}
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// Step 4 — create sibling tmp directory under the same parent to guarantee
	// a same-filesystem rename (avoids cross-device link errors).
	parent := filepath.Dir(targetDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}
	tmpDir, err := os.MkdirTemp(parent, ".jitctx-init-"+name+"-*")
	if err != nil {
		return fmt.Errorf("create tmp dir: %w", err)
	}

	// Step 5 — defer cleanup; no-op after a successful rename because tmpDir
	// no longer exists at that path.
	success := false
	defer func() {
		if !success {
			os.RemoveAll(tmpDir)
		}
	}()

	// Step 6 — walk and copy every file verbatim.
	if err := fs.WalkDir(subFS, ".", func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return fmt.Errorf("walk %s: %w", path, werr)
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		dest := filepath.Join(tmpDir, filepath.FromSlash(path))
		if d.IsDir() {
			if path == "." {
				// tmpDir is already created.
				return nil
			}
			if err := os.Mkdir(dest, 0o755); err != nil {
				return fmt.Errorf("mkdir %s: %w", dest, err)
			}
			return nil
		}

		data, err := fs.ReadFile(subFS, path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// Step 7 — atomic rename.
	if err := os.Rename(tmpDir, targetDir); err != nil {
		return fmt.Errorf("rename to target: %w", err)
	}
	success = true
	return nil
}

// bundledNames returns a sorted slice of names of all bundled profiles.
// Used to populate UnknownBundledProfileError.Available without
// depending on *Bundled.
func bundledNames() []string {
	entries, err := fs.ReadDir(bundledFS, "bundled")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// Compile-time assertion.
var _ profileport.ExtractBundledProfilePort = (*Extractor)(nil)
