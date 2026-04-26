package fsconfig

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// Loader implements LoadJitctxConfigPort. Stateless; the workDir is
// supplied per-call so the same instance can be wired once and reused
// across commands that operate on different roots.
type Loader struct {
	logger *slog.Logger
}

// New returns a Loader. logger may be nil — the loader treats nil as
// slog.Default().
func New(logger *slog.Logger) *Loader {
	if logger == nil {
		logger = slog.Default()
	}
	return &Loader{logger: logger}
}

// LoadJitctxConfig reads <workDir>/.jitctx/config.yaml.
//
// Contract:
//   - missing file (os.IsNotExist) → (model.JitctxConfig{}, nil)
//   - empty/comment-only file (io.EOF on Decode) → (model.JitctxConfig{}, nil)
//   - malformed YAML or unknown top-level key → wrapped error
//   - audit.disabled_rules null|missing|[] → empty DisabledRules
//
// SEC-001: workDir governs the resolution; the helper joins
// ".jitctx/config.yaml" and never traverses outside.
func (l *Loader) LoadJitctxConfig(ctx context.Context, workDir string) (model.JitctxConfig, error) {
	if err := ctx.Err(); err != nil {
		return model.JitctxConfig{}, err
	}
	if workDir == "" {
		workDir = "."
	}
	path := filepath.Join(workDir, ".jitctx", "config.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return model.JitctxConfig{}, nil
		}
		return model.JitctxConfig{}, fmt.Errorf("read %s: %w", path, err)
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var dto configFileDTO
	if err := dec.Decode(&dto); err != nil {
		if errors.Is(err, io.EOF) {
			// Empty file (zero bytes or comments only).
			return model.JitctxConfig{}, nil
		}
		return model.JitctxConfig{}, fmt.Errorf("read %s: %w", path, err)
	}
	return toDomain(dto), nil
}
