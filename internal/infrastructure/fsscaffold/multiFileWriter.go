package fsscaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/port/spec"
	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

// MultiFileWriter implements spec.WriteProductionFilesPort by writing a batch
// of rendered production files atomically per EP02RF-009.
type MultiFileWriter struct{}

func NewMultiFileWriter() *MultiFileWriter { return &MultiFileWriter{} }

var _ spec.WriteProductionFilesPort = (*MultiFileWriter)(nil)

func (w *MultiFileWriter) WriteAll(ctx context.Context, files []scaffoldvo.ProductionFile) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return []string{}, nil
	}

	// Phase 1: clean paths + conflict check.
	cleaned := make([]string, len(files))
	var conflicts []string
	for i, f := range files {
		cleaned[i] = filepath.Clean(f.Path)
		if _, err := os.Stat(cleaned[i]); err == nil {
			conflicts = append(conflicts, cleaned[i])
		}
	}
	if len(conflicts) > 0 {
		sort.Strings(conflicts)
		return nil, &domerr.ScaffoldConflictError{Conflicts: conflicts}
	}

	// Phase 2: mkdir -p parent of every Path; write each to a tmp file.
	type pair struct {
		tmp   string
		final string
		idx   int
	}
	pairs := make([]pair, 0, len(files))
	cleanup := func() {
		for _, p := range pairs {
			_ = os.Remove(p.tmp)
		}
	}
	for i, f := range files {
		final := cleaned[i]
		dir := filepath.Dir(final)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			cleanup()
			return nil, fmt.Errorf("scaffold: mkdir %s: %w", dir, domerr.ErrSpecWriteFailed)
		}
		base := filepath.Base(final)
		tf, err := os.CreateTemp(dir, base+".*.tmp")
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("scaffold: create temp for %s: %w", final, domerr.ErrSpecWriteFailed)
		}
		if _, werr := tf.Write(f.Content); werr != nil {
			tf.Close()
			_ = os.Remove(tf.Name())
			cleanup()
			return nil, fmt.Errorf("scaffold: write temp for %s: %w", final, domerr.ErrSpecWriteFailed)
		}
		if cerr := tf.Close(); cerr != nil {
			_ = os.Remove(tf.Name())
			cleanup()
			return nil, fmt.Errorf("scaffold: close temp for %s: %w", final, domerr.ErrSpecWriteFailed)
		}
		pairs = append(pairs, pair{tmp: tf.Name(), final: final, idx: i})
	}

	// Phase 3: rename each tmp → final. On rename failure, best-effort
	// remove already-renamed targets AND remaining temps.
	written := make([]string, 0, len(pairs))
	for i, p := range pairs {
		if err := os.Rename(p.tmp, p.final); err != nil {
			// cleanup successful renames
			for _, w := range written {
				_ = os.Remove(w)
			}
			// cleanup remaining temps
			for _, rest := range pairs[i:] {
				_ = os.Remove(rest.tmp)
			}
			return nil, fmt.Errorf("scaffold: rename %s: %w", p.final, domerr.ErrSpecWriteFailed)
		}
		written = append(written, p.final)
	}

	sort.Strings(written)
	return written, nil
}
