package parser

import (
	"context"
	"io/fs"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// ParseJavaFilePort parses one Java file and returns a structured summary.
type ParseJavaFilePort interface {
	// ParseJavaFile parses one Java file and returns a structured summary.
	// Syntactic errors produce a non-nil summary AND a wrapped ErrPartialParse
	// that the caller may choose to skip (errors.Is(err, ErrPartialParse)).
	ParseJavaFile(ctx context.Context, fsys fs.FS, path string) (model.JavaFileSummary, error)
}
