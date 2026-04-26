package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/service"
	refactorvo "github.com/jitctx/jitctx/internal/domain/vo/refactor"
)

// TestNewMarkerParser_ReturnsNonNil verifies the constructor returns a usable
// value (not nil) — required by wire.go's injection pattern.
func TestNewMarkerParser_ReturnsNonNil(t *testing.T) {
	t.Parallel()
	p := service.NewMarkerParser()
	require.NotNil(t, p)
}

// TestMarkerParser_Parse_LineComment covers the happy-path line-comment case.
func TestMarkerParser_Parse_LineComment(t *testing.T) {
	t.Parallel()
	p := service.NewMarkerParser()
	raw := "// TODO(jitctx): rename - rename foo to handleFoo"
	result := p.Parse("com/example/Foo.java", 10, "line_comment", raw)
	require.True(t, result.Matched)
	require.Equal(t, refactorvo.MarkerTypeRename, result.Marker.Type)
	require.Equal(t, "rename foo to handleFoo", result.Marker.Description)
	require.Empty(t, result.UnknownType)
	require.Equal(t, "com/example/Foo.java", result.Marker.FilePath)
	require.Equal(t, 10, result.Marker.Line)
}

// TestMarkerParser_Parse_BlockCommentSingleLine covers a single-line block comment.
func TestMarkerParser_Parse_BlockCommentSingleLine(t *testing.T) {
	t.Parallel()
	p := service.NewMarkerParser()
	raw := "/* TODO(jitctx): rename - bad name */"
	result := p.Parse("com/example/Bar.java", 5, "block_comment", raw)
	require.True(t, result.Matched)
	require.Equal(t, refactorvo.MarkerTypeRename, result.Marker.Type)
	require.Equal(t, "bad name", result.Marker.Description)
	require.Empty(t, result.UnknownType)
}

// TestMarkerParser_Parse_BlockCommentMultiLine covers Javadoc-style block
// comments where internal lines are prefixed with " * ".
func TestMarkerParser_Parse_BlockCommentMultiLine(t *testing.T) {
	t.Parallel()
	p := service.NewMarkerParser()
	raw := "/*\n * TODO(jitctx): extract-method - validateEmail\n */"
	result := p.Parse("com/example/Baz.java", 20, "block_comment", raw)
	require.True(t, result.Matched)
	require.Equal(t, refactorvo.MarkerTypeExtractMethod, result.Marker.Type)
	require.Equal(t, "validateEmail", result.Marker.Description)
	require.Empty(t, result.UnknownType)
}

// TestMarkerParser_Parse_MissingSeparator verifies that a marker prefix
// without the required " - " separator produces an Unparseable result.
func TestMarkerParser_Parse_MissingSeparator(t *testing.T) {
	t.Parallel()
	p := service.NewMarkerParser()
	raw := "// TODO(jitctx): missing format here"
	result := p.Parse("com/example/Qux.java", 7, "line_comment", raw)
	require.True(t, result.Matched)
	require.Equal(t, refactorvo.MarkerTypeUnparseable, result.Marker.Type)
	// OriginalText must be the verbatim raw comment (including delimiters).
	require.Equal(t, raw, result.Marker.OriginalText)
	require.Empty(t, result.Marker.Description)
}

// TestMarkerParser_Parse_UnknownType verifies that an unrecognised type token
// produces MarkerTypeOther and populates UnknownType with the original case.
func TestMarkerParser_Parse_UnknownType(t *testing.T) {
	t.Parallel()
	p := service.NewMarkerParser()
	raw := "// TODO(jitctx): weird-thing - some text"
	result := p.Parse("com/example/Foo.java", 3, "line_comment", raw)
	require.True(t, result.Matched)
	require.Equal(t, refactorvo.MarkerTypeOther, result.Marker.Type)
	require.Equal(t, "some text", result.Marker.Description)
	require.Equal(t, "weird-thing", result.UnknownType)
}

// TestMarkerParser_Parse_NonMarkerComments verifies that comments that do not
// contain the TODO(jitctx): prefix return Matched=false.
func TestMarkerParser_Parse_NonMarkerComments(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		kind string
		raw  string
	}{
		{
			name: "plain-todo",
			kind: "line_comment",
			raw:  "// TODO: improve this",
		},
		{
			name: "fixme-other-owner",
			kind: "line_comment",
			raw:  "// FIXME(someone): broken",
		},
		{
			name: "plain-comment",
			kind: "line_comment",
			raw:  "// just a comment",
		},
	}

	p := service.NewMarkerParser()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := p.Parse("com/example/Foo.java", 1, tc.kind, tc.raw)
			require.False(t, result.Matched, "expected Matched=false for: %s", tc.raw)
		})
	}
}

// TestMarkerParser_Parse_CaseInsensitiveType verifies case-insensitive type
// matching: the parser lowercases the type token before classifying, so
// "RENAME" resolves to MarkerTypeRename.
func TestMarkerParser_Parse_CaseInsensitiveType(t *testing.T) {
	t.Parallel()
	p := service.NewMarkerParser()
	raw := "// TODO(jitctx): RENAME - foo"
	result := p.Parse("com/example/Foo.java", 1, "line_comment", raw)
	require.True(t, result.Matched)
	// The parser lowercases the type token (step 4), so RENAME → rename → MarkerTypeRename.
	require.Equal(t, refactorvo.MarkerTypeRename, result.Marker.Type)
	require.Empty(t, result.UnknownType)
}

// TestMarkerParser_Parse_DescriptionWhitespace verifies that leading and
// trailing whitespace around the description is trimmed, but internal spacing
// is preserved verbatim (per §2 spec / step 4 of the algorithm).
func TestMarkerParser_Parse_DescriptionWhitespace(t *testing.T) {
	t.Parallel()
	p := service.NewMarkerParser()
	// Leading and trailing spaces after the separator.
	raw := "// TODO(jitctx): rename -   spaces preserved   "
	result := p.Parse("com/example/Foo.java", 1, "line_comment", raw)
	require.True(t, result.Matched)
	require.Equal(t, refactorvo.MarkerTypeRename, result.Marker.Type)
	// The implementation trims outer whitespace (strings.TrimSpace) from
	// the description, so leading/trailing spaces are removed but internal
	// spaces within "spaces preserved" are kept.
	require.Equal(t, "spaces preserved", result.Marker.Description)
}

// TestMarkerParser_Parse_AllKnownTypes exercises a happy-path subtest for every
// RF-006 recognised type to ensure none are accidentally classified as "other".
func TestMarkerParser_Parse_AllKnownTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		typeToken string
		wantType  refactorvo.MarkerType
	}{
		{
			name:      "rename",
			typeToken: "rename",
			wantType:  refactorvo.MarkerTypeRename,
		},
		{
			name:      "extract-method",
			typeToken: "extract-method",
			wantType:  refactorvo.MarkerTypeExtractMethod,
		},
		{
			name:      "move",
			typeToken: "move",
			wantType:  refactorvo.MarkerTypeMove,
		},
		{
			name:      "inline",
			typeToken: "inline",
			wantType:  refactorvo.MarkerTypeInline,
		},
		{
			name:      "simplify",
			typeToken: "simplify",
			wantType:  refactorvo.MarkerTypeSimplify,
		},
	}

	p := service.NewMarkerParser()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			raw := "// TODO(jitctx): " + tc.typeToken + " - some description"
			result := p.Parse("com/example/Foo.java", 1, "line_comment", raw)
			require.True(t, result.Matched)
			require.Equal(t, tc.wantType, result.Marker.Type)
			require.Equal(t, "some description", result.Marker.Description)
			require.Empty(t, result.UnknownType)
		})
	}
}
