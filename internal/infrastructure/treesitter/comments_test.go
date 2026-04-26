package treesitter_test

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/port/parser"
	"github.com/jitctx/jitctx/internal/infrastructure/treesitter"
)

// TestListJavaComments_LineComment verifies that a file containing a single
// line comment (`// ...`) returns exactly one JavaComment with kind=line_comment
// and a 1-based line number matching the position in the source.
func TestListJavaComments_LineComment(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app;

// just a line comment
public class Foo {
}`
	fsys := fstest.MapFS{
		"Foo.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	comments, err := p.ListJavaComments(context.Background(), fsys, "Foo.java")
	require.NoError(t, err)
	require.Len(t, comments, 1)

	c := comments[0]
	require.Equal(t, "line_comment", c.Kind)
	require.Equal(t, 3, c.Line) // line 3 in the source (1-based)
	require.Equal(t, "// just a line comment", c.Text)
}

// TestListJavaComments_BlockComment verifies that a file with a single-line
// block comment (`/* ... */`) returns one JavaComment with kind=block_comment
// and the correct 1-based line number.
func TestListJavaComments_BlockComment(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app;

/* a block */
public class Bar {
}`
	fsys := fstest.MapFS{
		"Bar.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	comments, err := p.ListJavaComments(context.Background(), fsys, "Bar.java")
	require.NoError(t, err)
	require.Len(t, comments, 1)

	c := comments[0]
	require.Equal(t, "block_comment", c.Kind)
	require.Equal(t, 3, c.Line) // opening line is line 3 (1-based)
	require.Equal(t, "/* a block */", c.Text)
}

// TestListJavaComments_MultiLineBlockComment verifies that a multi-line block
// comment is returned as a single JavaComment whose Line points to the opening
// `/*` line (1-based), not to subsequent lines.
func TestListJavaComments_MultiLineBlockComment(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app;

/*
 * multi-line
 * comment
 */
public class Baz {
}`
	fsys := fstest.MapFS{
		"Baz.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	comments, err := p.ListJavaComments(context.Background(), fsys, "Baz.java")
	require.NoError(t, err)
	require.Len(t, comments, 1)

	c := comments[0]
	require.Equal(t, "block_comment", c.Kind)
	require.Equal(t, 3, c.Line) // line of opening "/*" (1-based)
}

// TestListJavaComments_NestedMethodBodyComment verifies that the recursive
// tree walk descends into method bodies, so a comment placed inside a method
// is also extracted.
func TestListJavaComments_NestedMethodBodyComment(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app;

public class Service {
    public void doWork() {
        // comment inside method body
        int x = 1;
    }
}`
	fsys := fstest.MapFS{
		"Service.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	comments, err := p.ListJavaComments(context.Background(), fsys, "Service.java")
	require.NoError(t, err)
	require.Len(t, comments, 1)

	c := comments[0]
	require.Equal(t, "line_comment", c.Kind)
	require.Equal(t, 5, c.Line) // comment is on line 5 (1-based)
	require.Equal(t, "// comment inside method body", c.Text)
}

// TestListJavaComments_CommentFreeFile verifies that a syntactically valid
// Java file with no comments returns an empty slice and no error.
func TestListJavaComments_CommentFreeFile(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app;

public class Empty {
    private int value;
}`
	fsys := fstest.MapFS{
		"Empty.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	comments, err := p.ListJavaComments(context.Background(), fsys, "Empty.java")
	require.NoError(t, err)
	require.Empty(t, comments)
}

// TestListJavaComments_MultipleCommentsAtVaryingNestingLevels verifies that
// all comments are returned regardless of nesting depth: top-of-file, between
// methods, inside a method body, and inside a nested class.
func TestListJavaComments_MultipleCommentsAtVaryingNestingLevels(t *testing.T) {
	t.Parallel()

	// line 1: package
	// line 2: blank
	// line 3: top-of-file comment
	// line 4: class opens
	// line 5: between-methods comment
	// line 6: method opens
	// line 7: inside-method comment
	// line 8: method closes
	// line 9: nested class opens
	// line 10: inside-nested-class comment
	// line 11: nested class closes
	// line 12: outer class closes
	javaCode := `package com.app;

// top-of-file
public class Outer {
    // between methods
    public void method() {
        // inside method
    }
    public static class Inner {
        // inside nested class
    }
}`
	fsys := fstest.MapFS{
		"Outer.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	comments, err := p.ListJavaComments(context.Background(), fsys, "Outer.java")
	require.NoError(t, err)
	require.Len(t, comments, 4)

	// Verify order and line numbers (tree-walk order = source order).
	require.Equal(t, 3, comments[0].Line)  // top-of-file
	require.Equal(t, 5, comments[1].Line)  // between methods
	require.Equal(t, 7, comments[2].Line)  // inside method
	require.Equal(t, 10, comments[3].Line) // inside nested class

	// All are line comments.
	for _, c := range comments {
		require.Equal(t, "line_comment", c.Kind)
	}
}

// TestListJavaComments_PartialParseReturnsCommentsAndErrPartialParse verifies
// that a syntactically broken Java file still yields all comments found before
// the error node, and that the returned error wraps domerr.ErrPartialParse.
func TestListJavaComments_PartialParseReturnsCommentsAndErrPartialParse(t *testing.T) {
	t.Parallel()

	// The syntax error (missing closing brace / garbled token) forces tree-sitter
	// to produce an ERROR node, which triggers the ErrPartialParse path.
	javaCode := `package com.app;

// comment before error
public class Broken {
    @@@@@@invalid_token
`
	fsys := fstest.MapFS{
		"Broken.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	comments, err := p.ListJavaComments(context.Background(), fsys, "Broken.java")

	// Must wrap ErrPartialParse.
	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrPartialParse), "expected ErrPartialParse, got: %v", err)

	// The comment before the error node must still be present.
	require.NotEmpty(t, comments)
	found := false
	for _, c := range comments {
		if c.Kind == "line_comment" && c.Line == 3 {
			require.Equal(t, "// comment before error", c.Text)
			found = true
		}
	}
	require.True(t, found, "expected the comment on line 3 to be returned despite the parse error")
}

// TestListJavaComments_LineNumbersAre1Based verifies the 1-based line number
// contract for several comments at known positions in a source string.
func TestListJavaComments_LineNumbersAre1Based(t *testing.T) {
	t.Parallel()

	// Line 1: package
	// Line 2: blank
	// Line 3: first comment
	// Line 4: class opens
	// Line 5: second comment
	// Line 6: class closes
	javaCode := `package com.app;

// first comment
public class Num {
    /* second comment */
}`
	fsys := fstest.MapFS{
		"Num.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	comments, err := p.ListJavaComments(context.Background(), fsys, "Num.java")
	require.NoError(t, err)
	require.Len(t, comments, 2)

	// Verify exact 1-based line numbers.
	var lineNums []int
	for _, c := range comments {
		lineNums = append(lineNums, c.Line)
	}

	wantLines := []int{3, 5}
	require.Equal(t, wantLines, lineNums)

	// Confirm kinds as well.
	require.Equal(t, "line_comment", comments[0].Kind)
	require.Equal(t, "block_comment", comments[1].Kind)
}

// TestListJavaComments_satisfiesListJavaCommentsPort is a compile-time
// assertion that *Parser implements parser.ListJavaCommentsPort.
// If the interface changes, this test file will fail to compile.
func TestListJavaComments_satisfiesListJavaCommentsPort(t *testing.T) {
	t.Parallel()
	var _ parser.ListJavaCommentsPort = treesitter.New()
}
