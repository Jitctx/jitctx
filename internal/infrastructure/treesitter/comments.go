package treesitter

import (
	"context"
	"fmt"
	"io/fs"

	sitter "github.com/smacker/go-tree-sitter"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/port/parser"
)

// ListJavaComments satisfies ListJavaCommentsPort. It parses the given Java
// file and returns every line_comment and block_comment node found anywhere
// in the syntax tree — at file scope, inside class bodies, inside method
// bodies, and inside nested types.
//
// The same *Parser struct satisfies ParseJavaFilePort, ListJavaFieldsPort,
// and ListJavaCommentsPort (one collaborator, multiple ISP ports) without
// any shared mutable state.
//
// Partial parses: when the tree contains ERROR or MISSING nodes the method
// still returns all comments collected up to that point, wrapped together
// with domerr.ErrPartialParse — mirroring the behaviour of ParseJavaFile.
func (p *Parser) ListJavaComments(ctx context.Context, fsys fs.FS, path string) ([]parser.JavaComment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, domerr.ErrParseFailure)
	}

	tsParser := sitter.NewParser()
	tsParser.SetLanguage(JavaLanguage())
	tree, parseErr := tsParser.ParseCtx(ctx, nil, data)
	if parseErr != nil {
		return nil, fmt.Errorf("parse tree error for %s: %w", path, domerr.ErrParseFailure)
	}
	if tree == nil {
		return nil, fmt.Errorf("parse tree is nil for %s: %w", path, domerr.ErrParseFailure)
	}
	defer tree.Close()

	root := tree.RootNode()
	hasErrors := containsErrors(root)

	comments := collectComments(root, data, path)

	if hasErrors {
		return comments, fmt.Errorf("partial parse %s: %w", path, domerr.ErrPartialParse)
	}
	return comments, nil
}

// collectComments performs a recursive pre-order walk of the syntax tree,
// collecting every line_comment and block_comment node it encounters.
// Walking the full tree (rather than restricting to class_body direct
// children) ensures markers inside method bodies and nested types are found.
func collectComments(node *sitter.Node, src []byte, filePath string) []parser.JavaComment {
	var comments []parser.JavaComment
	collectCommentsInto(node, src, filePath, &comments)
	return comments
}

// collectCommentsInto is the recursive helper for collectComments.
func collectCommentsInto(node *sitter.Node, src []byte, filePath string, out *[]parser.JavaComment) {
	if node == nil {
		return
	}

	kind := node.Type()
	if kind == nodeLineComment || kind == nodeBlockComment {
		// StartPoint().Row is 0-based; add 1 for the 1-based line number
		// required by the JavaComment contract. For block comments this
		// is the line of the opening "/*" delimiter.
		line := int(node.StartPoint().Row) + 1
		*out = append(*out, parser.JavaComment{
			Kind: kind,
			Line: line,
			Text: nodeText(node, src),
		})
		// Comment nodes in tree-sitter have no meaningful children;
		// no need to recurse into them.
		return
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		collectCommentsInto(node.Child(i), src, filePath, out)
	}
}
