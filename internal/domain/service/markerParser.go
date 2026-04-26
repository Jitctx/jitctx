package service

import (
	"regexp"
	"strings"

	refactorvo "github.com/jitctx/jitctx/internal/domain/vo/refactor"
)

// MarkerParser classifies a raw Java comment into a RefactorMarker (or
// reports that it is not a marker at all). Pure stateless service —
// safe to share, no allocations beyond the regex match results.
type MarkerParser struct{}

// NewMarkerParser constructs a MarkerParser. The struct carries no
// state; the constructor exists only for symmetry with other domain
// services and so wire.go can inject a typed value.
func NewMarkerParser() *MarkerParser {
	return &MarkerParser{}
}

// ParseResult bundles the outcome of parsing one comment.
//   - Matched   = false → comment is not a refactor marker; caller skips it.
//   - Marker    = the classified marker (Type may be Unparseable).
//   - UnknownType = non-empty when the parsed type is not in the recognised
//     RF-006 set; the marker is bucketed as MarkerTypeOther and the caller
//     records the raw type for stderr warnings.
type ParseResult struct {
	Matched     bool
	Marker      refactorvo.RefactorMarker
	UnknownType string
}

// MarkerPrefixPattern is the literal regex source the implementation must
// compile in init/at construction. Centralised here so unit tests can
// assert the pattern verbatim and the implementation cannot drift.
const MarkerPrefixPattern = `^TODO\(jitctx\):\s*(.*)$`

// markerPrefixRegexp is the compiled form. Package-level to avoid
// re-compilation on each call.
var markerPrefixRegexp = regexp.MustCompile(MarkerPrefixPattern)

// SeparatorLiteral is the required separator between type and description
// in a well-formed marker. Exported for unit-test coverage.
const SeparatorLiteral = " - "

// Parse classifies one raw comment.
//
// Algorithm:
//  1. Strip comment delimiters: "//" prefix, or surrounding "/*" "*/".
//     For block comments, also strip leading "*" on each internal line
//     when present (Javadoc style). Trim outer whitespace.
//  2. Match the prefix regex `^TODO\(jitctx\):\s*(.*)$`.
//     - No match → ParseResult{Matched: false}.
//  3. Within the captured tail, split on the first " - " (literal: space,
//     hyphen, space). When the separator is absent → MarkerTypeUnparseable
//     with OriginalText set to the comment's verbatim source (including
//     delimiters, exactly as it was in the file).
//  4. When the separator is present, the left side is the type token
//     (lowercased, surrounding whitespace trimmed) and the right side
//     is the description (whitespace trimmed, internal whitespace
//     preserved verbatim).
//  5. If type token matches a recognised RF-006 type → use it directly.
//     Otherwise → set Type = MarkerTypeOther, UnknownType = rawType.
//
// Inputs:
//   - filePath: forward-slash, relative; copied into Marker.FilePath.
//   - line: 1-based start line of the comment; copied into Marker.Line.
//   - kind: tree-sitter node type ("line_comment" | "block_comment").
//   - rawText: verbatim comment text including delimiters.
//
// The ModuleID field is left empty by Parse — the application layer
// populates it after calling the parser.
func (p *MarkerParser) Parse(filePath string, line int, kind, rawText string) ParseResult {
	stripped := stripCommentDelimiters(kind, rawText)

	// For block comments with multiple lines, match against the first
	// non-empty line (where the marker prefix must appear).
	textToMatch := stripped
	if kind == "block_comment" {
		lines := strings.Split(stripped, "\n")
		for _, l := range lines {
			trimmed := strings.TrimSpace(l)
			if trimmed != "" {
				textToMatch = trimmed
				break
			}
		}
	}

	m := markerPrefixRegexp.FindStringSubmatch(textToMatch)
	if m == nil {
		return ParseResult{Matched: false}
	}

	tail := strings.TrimSpace(m[1])

	// Step 3: check for separator.
	idx := strings.Index(tail, SeparatorLiteral)
	if idx < 0 {
		// Malformed: marker prefix present but no " - " separator.
		return ParseResult{
			Matched: true,
			Marker: refactorvo.RefactorMarker{
				FilePath:     filePath,
				Line:         line,
				Type:         refactorvo.MarkerTypeUnparseable,
				OriginalText: rawText,
			},
		}
	}

	// Step 4: split into type and description.
	typeStr := strings.ToLower(strings.TrimSpace(tail[:idx]))
	descStr := strings.TrimSpace(tail[idx+len(SeparatorLiteral):])

	// Step 5: classify the type.
	mt := refactorvo.MarkerType(typeStr)
	var unknownType string
	if !mt.IsKnownUserType() {
		unknownType = strings.TrimSpace(tail[:idx]) // preserve original case for warning
		mt = refactorvo.MarkerTypeOther
	}

	return ParseResult{
		Matched: true,
		Marker: refactorvo.RefactorMarker{
			FilePath:    filePath,
			Line:        line,
			Type:        mt,
			Description: descStr,
		},
		UnknownType: unknownType,
	}
}

// stripCommentDelimiters returns the inner text of a tree-sitter comment
// node with delimiters removed and whitespace trimmed.
//   - kind == "line_comment" → strip leading "//" and any single space.
//   - kind == "block_comment" → strip outer "/*" and "*/", then for
//     each internal line strip optional leading whitespace + "*" + optional space.
func stripCommentDelimiters(kind, rawText string) string {
	switch kind {
	case "line_comment":
		s := rawText
		s = strings.TrimPrefix(s, "//")
		// Strip at most one leading space (preserves intentional indentation).
		s = strings.TrimPrefix(s, " ")
		return strings.TrimSpace(s)

	case "block_comment":
		s := rawText
		// Strip outer delimiters.
		s = strings.TrimPrefix(s, "/*")
		s = strings.TrimSuffix(s, "*/")
		// For each line, strip optional leading whitespace + "*" + optional space
		// (Javadoc-style decoration).
		lines := strings.Split(s, "\n")
		cleaned := make([]string, 0, len(lines))
		for _, l := range lines {
			trimmed := strings.TrimLeft(l, " \t")
			trimmed = strings.TrimPrefix(trimmed, "*")
			trimmed = strings.TrimPrefix(trimmed, " ")
			cleaned = append(cleaned, trimmed)
		}
		return strings.TrimSpace(strings.Join(cleaned, "\n"))

	default:
		return strings.TrimSpace(rawText)
	}
}
