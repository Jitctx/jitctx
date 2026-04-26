package config

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// LoadJitctxConfigPort returns the JitctxConfig view rooted at workDir.
//
// Contract:
//   - workDir is the project root the use case is operating on; the loader
//     resolves "<workDir>/.jitctx/config.yaml" internally.
//   - When the config file is ABSENT, the loader returns
//     (model.JitctxConfig{}, nil) — a missing file is the documented default.
//   - When the file is present but malformed (invalid YAML, unknown
//     top-level key, wrong type), the loader returns a wrapped error.
//   - When `audit.disabled_rules` is null, missing, or `[]`, DisabledRules
//     is the nil slice.
//
// ISP: exactly one method, per domain-layer guideline.
type LoadJitctxConfigPort interface {
	LoadJitctxConfig(ctx context.Context, workDir string) (model.JitctxConfig, error)
}
