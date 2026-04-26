package service

import "strings"

// SignatureNormalizer normalises a Java method signature for whitespace-
// insensitive equality comparison. Pure, stateless. Reused by
// ContractDiffer to detect signature divergence per Gherkin scenario 6.
//
// Algorithm:
//  1. Trim outer whitespace and a trailing ';' if present.
//  2. Collapse runs of ASCII whitespace to a single space outside of
//     '(' … ')'.
//  3. Inside '(' … ')', remove ALL whitespace adjacent to '(' and ')'
//     and collapse internal whitespace to a single space between tokens.
//
// Examples:
//
//	"User save( User user )"   -> "User save(User user)"
//	"User  save(User  user)"   -> "User save(User user)"
//	"User save(User user) ;"   -> "User save(User user)"
type SignatureNormalizer struct{}

// NewSignatureNormalizer returns a stateless normaliser.
func NewSignatureNormalizer() SignatureNormalizer { return SignatureNormalizer{} }

// Normalize returns the canonical form of sig per the algorithm above.
// On unparseable input (e.g. unmatched parens) returns the trimmed
// original — the differ falls back to plain string equality, which is
// safe (worst case: a false MODIFY that the user can clear by aligning
// formatting).
func (SignatureNormalizer) Normalize(sig string) string {
	// Step 1: trim outer whitespace and trailing ';'.
	s := strings.TrimSpace(sig)
	s = strings.TrimRight(s, ";")
	s = strings.TrimSpace(s)

	if s == "" {
		return s
	}

	// Find the opening '(' to split the signature into pre-paren and in-paren parts.
	openIdx := strings.IndexByte(s, '(')
	if openIdx < 0 {
		// No parens — just collapse whitespace and return.
		return collapseWhitespace(s)
	}

	// Find matching closing ')'.
	closeIdx := findMatchingCloseParen(s, openIdx)
	if closeIdx < 0 {
		// Unmatched parens — fall back to trimmed original.
		return s
	}

	// Step 2: normalise the part before '('.
	prefix := collapseWhitespace(strings.TrimSpace(s[:openIdx]))

	// Step 3: normalise the parameter list inside '(' … ')'.
	inner := s[openIdx+1 : closeIdx]
	normInner := normalizeParamList(inner)

	// Reconstruct.
	return prefix + "(" + normInner + ")"
}

// collapseWhitespace replaces any run of ASCII whitespace with a single space.
func collapseWhitespace(s string) string {
	var b strings.Builder
	inSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
		} else {
			b.WriteRune(r)
			inSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}

// normalizeParamList normalises the content between '(' and ')':
// trims leading/trailing whitespace, then collapses all internal
// whitespace runs to a single space between tokens.
func normalizeParamList(inner string) string {
	trimmed := strings.TrimSpace(inner)
	if trimmed == "" {
		return ""
	}
	return collapseWhitespace(trimmed)
}

// findMatchingCloseParen returns the index of ')' that closes the '(' at
// openIdx. Returns -1 if not found.
func findMatchingCloseParen(s string, openIdx int) int {
	depth := 0
	for i := openIdx; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
