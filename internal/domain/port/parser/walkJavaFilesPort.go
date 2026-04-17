package parser

import (
	"context"
	"io/fs"
)

// WalkJavaFilesPort enumerates Java files under src/main/java/** in an fs.FS.
type WalkJavaFilesPort interface {
	// WalkJavaFiles yields *.java paths under src/main/java/** relative to
	// the root of fsys. Returns an error only for fatal I/O failures; per-file
	// errors from the underlying fs are wrapped and returned.
	WalkJavaFiles(ctx context.Context, fsys fs.FS) ([]string, error)
}
