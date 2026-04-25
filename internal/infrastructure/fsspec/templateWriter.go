package fsspec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/port/spec"
)

// Compile-time check that TemplateWriter satisfies the domain port.
var _ spec.WriteSpecTemplatePort = (*TemplateWriter)(nil)

// TemplateWriter writes a fully-rendered spec template to disk using a
// temp-file + atomic rename strategy (mirrors fsmanifest.Store.Save).
type TemplateWriter struct{}

// NewWriter returns a ready-to-use *TemplateWriter.
func NewWriter() *TemplateWriter {
	return &TemplateWriter{}
}

// Write writes content to path atomically:
//  1. ctx.Err() guard.
//  2. filepath.Clean(path); reject if final basename ends in ".tmp" (defensive).
//  3. Stat the final path; if it exists, return *domerr.SpecFileExistsError.
//  4. os.MkdirAll(filepath.Dir(cleaned), 0o755).
//  5. Create a temp file in the same directory via os.CreateTemp so the rename
//     is intra-volume.
//  6. Write content, close, then os.Rename. On rename failure remove the temp
//     file and return a wrapped domerr.ErrSpecWriteFailed.
//  7. Return cleaned, nil.
func (w *TemplateWriter) Write(ctx context.Context, path string, content []byte) (string, error) {
	// 1. ctx guard
	if err := ctx.Err(); err != nil {
		return "", err
	}

	// 2. Clean and reject .tmp basenames
	cleaned := filepath.Clean(path)
	if strings.HasSuffix(filepath.Base(cleaned), ".tmp") {
		return "", fmt.Errorf("rename spec template: %w", domerr.ErrSpecWriteFailed)
	}

	// 3. Conflict check — fail fast when the target already exists
	if _, err := os.Stat(cleaned); err == nil {
		return "", &domerr.SpecFileExistsError{Path: cleaned}
	}

	// 4. Ensure parent directory exists
	dir := filepath.Dir(cleaned)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir spec dir: %w", err)
	}

	// 5. Create temp file in the same directory (intra-volume rename)
	base := filepath.Base(cleaned)
	tmp, err := os.CreateTemp(dir, base+".*.tmp")
	if err != nil {
		return "", fmt.Errorf("tempfile: %w", err)
	}
	tmpPath := tmp.Name()

	// Write content and close before rename
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("write spec template: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("close tempfile: %w", err)
	}

	// 6. Atomic rename
	if err := os.Rename(tmpPath, cleaned); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("rename spec template: %w", domerr.ErrSpecWriteFailed)
	}

	// 7. Return the canonicalised path
	return cleaned, nil
}
