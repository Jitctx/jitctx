package service

import (
	"strings"

	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

// JPAFieldAnnotator converts the raw "<Type> <name>" field strings carried
// on a SpecContract into typed EntityField values whose Annotations slice
// carries the per-field JPA annotations the template should emit.
//
// Stateless and side-effect free — mirrors EndpointSynthesizer /
// MethodSignatureParser. Constructed once and reused.
//
// Rules (frozen — see §8 Q1, Q2, Q7):
//
//  1. Annotation match is case-insensitive on the FIELD NAME only:
//     a field named "id", "Id", or "ID" qualifies.
//  2. When the type token (lower-cased) equals "long" the annotator emits
//     BOTH "@Id" AND
//     "@GeneratedValue(strategy = GenerationType.IDENTITY)".
//  3. For any other type on an id-named field (UUID, String, ...) the
//     annotator emits ONLY "@Id" (no @GeneratedValue — UUIDs and string
//     keys are caller-assigned).
//  4. Non-id fields receive an empty Annotations slice (nil). The template
//     still iterates them and renders the bare "private <Type> <name>;"
//     line.
//  5. Whitespace inside a raw field string is collapsed: the annotator
//     splits on the LAST single space — types containing generics like
//     "Optional<String> name" still parse to Type="Optional<String>",
//     Name="name". A field string with NO space is a malformed input and
//     yields EntityField{Type: raw, Name: "", Annotations: nil}; the use
//     case is responsible for surfacing parse errors upstream (the spec
//     parser already enforces this — see US-001).
type JPAFieldAnnotator struct{}

// NewJPAFieldAnnotator returns a stateless annotator.
func NewJPAFieldAnnotator() JPAFieldAnnotator { return JPAFieldAnnotator{} }

// Annotate converts the raw "<Type> <name>" strings (as found in
// SpecContract.Fields) into typed EntityField values. Order is preserved.
// rawFields == nil yields a nil result.
func (JPAFieldAnnotator) Annotate(rawFields []string) []scaffoldvo.EntityField {
	if len(rawFields) == 0 {
		return nil
	}
	out := make([]scaffoldvo.EntityField, 0, len(rawFields))
	for _, raw := range rawFields {
		out = append(out, parseAndAnnotate(raw))
	}
	return out
}

// HasIDField reports whether any of the raw fields is an id-named field.
// Used by JavaImportResolver to decide whether to add jakarta.persistence.Id.
func (JPAFieldAnnotator) HasIDField(rawFields []string) bool {
	for _, raw := range rawFields {
		_, name := splitTypeAndName(raw)
		if isIDName(name) {
			return true
		}
	}
	return false
}

// HasGeneratedValueField reports whether any of the raw fields is an
// id-named field with type Long. Used by JavaImportResolver to decide
// whether to add jakarta.persistence.GeneratedValue and
// jakarta.persistence.GenerationType.
func (JPAFieldAnnotator) HasGeneratedValueField(rawFields []string) bool {
	for _, raw := range rawFields {
		typ, name := splitTypeAndName(raw)
		if isIDName(name) && isLongType(typ) {
			return true
		}
	}
	return false
}

// parseAndAnnotate is the unexported helper: split on last space, lower-case
// compare on name, emit annotations per the rule table above.
func parseAndAnnotate(raw string) scaffoldvo.EntityField {
	typ, name := splitTypeAndName(raw)
	if name == "" {
		// Malformed: no space found.
		return scaffoldvo.EntityField{Type: raw, Name: "", Annotations: nil}
	}
	if !isIDName(name) {
		return scaffoldvo.EntityField{Type: typ, Name: name, Annotations: nil}
	}
	// id-named field.
	if isLongType(typ) {
		return scaffoldvo.EntityField{
			Type: typ,
			Name: name,
			Annotations: []string{
				"@Id",
				"@GeneratedValue(strategy = GenerationType.IDENTITY)",
			},
		}
	}
	return scaffoldvo.EntityField{
		Type:        typ,
		Name:        name,
		Annotations: []string{"@Id"},
	}
}

// splitTypeAndName splits a raw field string on the LAST space.
// Returns (type, name). If no space is found, returns (raw, "").
func splitTypeAndName(raw string) (typ, name string) {
	raw = strings.TrimSpace(raw)
	idx := strings.LastIndexByte(raw, ' ')
	if idx < 0 {
		return raw, ""
	}
	return strings.TrimSpace(raw[:idx]), strings.TrimSpace(raw[idx+1:])
}

// isIDName reports whether the field name is "id" (case-insensitive).
func isIDName(name string) bool {
	return strings.EqualFold(name, "id")
}

// isLongType reports whether the type token is "long" (case-insensitive).
func isLongType(typ string) bool {
	return strings.EqualFold(typ, "long")
}
