package service

import (
	"errors"
	"strings"
	"unicode"
)

// EndpointSynthesizer parses a raw "VERB /path" string from
// SpecContract.Endpoints into the data needed to render one rest-adapter
// method.
type EndpointSynthesizer struct{}

// NewEndpointSynthesizer returns a stateless EndpointSynthesizer.
func NewEndpointSynthesizer() EndpointSynthesizer { return EndpointSynthesizer{} }

// EndpointBinding is the parsed projection.
//
//	Verb:         uppercased HTTP verb ("GET", "POST", …)
//	Path:         path literal as written ("/users/{id}")
//	Annotation:   `@<Verb>Mapping("<path>")` already formatted
//	MethodName:   synthesised Java identifier (camelCase from verb+path)
type EndpointBinding struct {
	Verb       string
	Path       string
	Annotation string
	MethodName string
}

// errMalformedEndpoint is the sentinel for malformed endpoint strings.
var errMalformedEndpoint = errors.New("endpoint must be 'VERB /path'")

// validVerbs is the set of accepted HTTP verbs.
var validVerbs = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"PATCH":   true,
	"HEAD":    true,
	"OPTIONS": true,
}

// Parse splits raw on whitespace. Must yield exactly 2 fields, else
// errMalformedEndpoint. Verb is uppercased and validated; Path is used
// verbatim to build the annotation literal and the synthesised method name.
func (EndpointSynthesizer) Parse(raw string) (EndpointBinding, error) {
	parts := strings.Fields(raw)
	if len(parts) != 2 {
		return EndpointBinding{}, errMalformedEndpoint
	}

	verb := strings.ToUpper(parts[0])
	path := parts[1]

	if !validVerbs[verb] {
		return EndpointBinding{}, errMalformedEndpoint
	}

	// Title-case the verb for the annotation name: "GET" → "Get"
	annotationVerb := strings.ToUpper(verb[:1]) + strings.ToLower(verb[1:])
	annotation := "@" + annotationVerb + `Mapping("` + path + `")`

	// Synthesise the Java method name: lowercase(verb) + capitalised path segments.
	// Walk runes of path: skip non-letter/digit; capitalise first letter after separator.
	methodName := synthesiseMethodName(verb, path)

	return EndpointBinding{
		Verb:       verb,
		Path:       path,
		Annotation: annotation,
		MethodName: methodName,
	}, nil
}

// synthesiseMethodName builds a camelCase Java method name from the HTTP
// verb and URL path. E.g. "GET /users/{id}" → "getUsersId".
func synthesiseMethodName(verb, path string) string {
	var b strings.Builder
	// Start with lowercase verb
	b.WriteString(strings.ToLower(verb))

	// Walk the path; capitalise first alphanum after any separator run
	capitaliseNext := false
	for _, r := range path {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if capitaliseNext {
				b.WriteRune(unicode.ToUpper(r))
				capitaliseNext = false
			} else {
				b.WriteRune(r)
			}
		} else {
			// Non-alphanum: trigger capitalisation of the next letter/digit
			if b.Len() > len(strings.ToLower(verb)) {
				// Only set capitaliseNext when we've already emitted some path rune
				capitaliseNext = true
			} else {
				// Still in the prefix after verb, treat separator as "capitalise next"
				capitaliseNext = true
			}
		}
	}

	return b.String()
}
