package parser

import (
	"context"
	"io/fs"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// ListJavaFieldsPort yields the field list for every top-level
// declaration in a Java file. The returned summary is a thin wrapper
// shaped like JavaFileSummary but populated only with Path, Package,
// Imports, and Declarations[i].Fields — the rest of JavaDeclaration is
// left zero so callers explicitly opt in to the field-only view.
//
// Implementations may share parser state with ParseJavaFilePort.
type ListJavaFieldsPort interface {
	ListJavaFields(ctx context.Context, fsys fs.FS, path string) (model.JavaFileSummary, error)
}
