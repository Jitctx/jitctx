package service

import (
	"errors"
	"strings"
)

// MethodSignatureParser splits a Java method signature string (as found in
// SpecContract.Methods) into the parts the test template needs to emit a
// `<methodName>_shouldDoSomething()` test method.
//
// Stateless and side-effect free. Mirrors the style of EndpointSynthesizer.
type MethodSignatureParser struct{}

// NewMethodSignatureParser returns a stateless parser.
func NewMethodSignatureParser() MethodSignatureParser { return MethodSignatureParser{} }

// ParsedMethod is the projection consumed by serviceTest.tmpl /
// restAdapterTest.tmpl. Only the Name field is currently consumed, but
// ReturnType and Params are kept so the renderer (or a future US) can
// build richer Mockito stubs (e.g., when arguments are present).
//
//	ReturnType: literal return-type token, e.g. "UserResponse", "void", "Optional<User>".
//	Name:       Java method identifier, e.g. "execute", "findByEmail".
//	Params:     positional parameter projection; empty slice when none.
type ParsedMethod struct {
	ReturnType string
	Name       string
	Params     []ParsedParam
}

// ParsedParam is one method parameter.
//
//	Type: literal type token (generics preserved as written).
//	Name: parameter identifier as declared.
type ParsedParam struct {
	Type string
	Name string
}

// errMalformedSignature is the sentinel for unparseable signatures.
var errMalformedSignature = errors.New("method signature must be '<ReturnType> <name>(<params>)'")

// Parse extracts the method projection from a raw signature string.
// Algorithm:
//  1. Trim whitespace and a trailing ';' if present.
//  2. Find the first '(' — left side splits into ReturnType + Name on
//     the LAST whitespace; right side up to matching ')' is the param list.
//  3. Param list is split on top-level ',' (depth tracked across '<','>').
//     Each param splits into Type + Name on the LAST whitespace.
//
// Returns errMalformedSignature on any structural failure. Never panics.
func (MethodSignatureParser) Parse(raw string) (ParsedMethod, error) {
	// Step 1: trim whitespace and trailing semicolon.
	s := strings.TrimSpace(raw)
	s = strings.TrimRight(s, ";")
	s = strings.TrimSpace(s)

	if s == "" {
		return ParsedMethod{}, errMalformedSignature
	}

	// Step 2: find first '('.
	parenIdx := strings.IndexByte(s, '(')
	if parenIdx < 0 {
		return ParsedMethod{}, errMalformedSignature
	}

	// Find matching closing ')'.
	closeIdx := findClosingParen(s, parenIdx)
	if closeIdx < 0 {
		return ParsedMethod{}, errMalformedSignature
	}

	// Left of '(' → "ReturnType Name"
	left := strings.TrimSpace(s[:parenIdx])
	if left == "" {
		return ParsedMethod{}, errMalformedSignature
	}

	// Split on LAST whitespace to get ReturnType and Name.
	lastSpace := strings.LastIndexAny(left, " \t")
	if lastSpace < 0 {
		return ParsedMethod{}, errMalformedSignature
	}
	returnType := strings.TrimSpace(left[:lastSpace])
	methodName := strings.TrimSpace(left[lastSpace+1:])

	if returnType == "" || methodName == "" {
		return ParsedMethod{}, errMalformedSignature
	}

	// Step 3: parse param list between '(' and ')'.
	paramStr := s[parenIdx+1 : closeIdx]
	params, err := parseParams(paramStr)
	if err != nil {
		return ParsedMethod{}, err
	}

	return ParsedMethod{
		ReturnType: returnType,
		Name:       methodName,
		Params:     params,
	}, nil
}

// findClosingParen returns the index of the ')' that closes the '(' at
// openIdx. Returns -1 if not found.
func findClosingParen(s string, openIdx int) int {
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

// parseParams splits a comma-separated Java parameter list, tracking
// generic depth so commas inside '<...>' are not treated as delimiters.
func parseParams(paramStr string) ([]ParsedParam, error) {
	trimmed := strings.TrimSpace(paramStr)
	if trimmed == "" {
		return nil, nil
	}

	segments := splitTopLevelCommas(trimmed)
	params := make([]ParsedParam, 0, len(segments))
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			return nil, errMalformedSignature
		}
		// Split on LAST whitespace to separate type from identifier.
		lastSpace := strings.LastIndexAny(seg, " \t")
		if lastSpace < 0 {
			return nil, errMalformedSignature
		}
		pType := strings.TrimSpace(seg[:lastSpace])
		pName := strings.TrimSpace(seg[lastSpace+1:])
		if pType == "" || pName == "" {
			return nil, errMalformedSignature
		}
		params = append(params, ParsedParam{Type: pType, Name: pName})
	}
	return params, nil
}

// splitTopLevelCommas splits s on ',' characters that are not inside
// angle brackets (generics depth tracking).
func splitTopLevelCommas(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			depth++
		case '>':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}
