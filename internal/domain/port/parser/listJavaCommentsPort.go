package parser

import (
	"context"
	"io/fs"
)

// JavaComment is a raw comment node extracted from a Java source file.
// Kind is the tree-sitter node type ("line_comment" or "block_comment");
// Line is 1-based and points to the start of the comment node.
type JavaComment struct {
	Kind string
	Line int
	Text string // verbatim source text including delimiters
}

// ListJavaCommentsPort yields every line_comment and block_comment node
// found anywhere in a Java file (file scope, class bodies, method bodies,
// nested types). The traversal is recursive — the marker scanner needs
// to find markers placed inside method bodies just as readily as at the
// file header. Implementations may share parser state with
// ParseJavaFilePort and ListJavaFieldsPort.
type ListJavaCommentsPort interface {
	ListJavaComments(ctx context.Context, fsys fs.FS, path string) ([]JavaComment, error)
}
