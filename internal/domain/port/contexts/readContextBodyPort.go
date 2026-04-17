package contexts

import (
	"context"
	"io/fs"
)

// ReadContextBodyPort returns the raw markdown body of a context file
// with any YAML front matter already stripped.
type ReadContextBodyPort interface {
	// ReadContextBody returns the raw markdown body of the context file at
	// path (with any YAML front matter already stripped).
	ReadContextBody(ctx context.Context, fsys fs.FS, path string) (string, error)
}
