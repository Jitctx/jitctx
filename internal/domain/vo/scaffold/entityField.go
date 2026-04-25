package scaffold

// EntityField is the typed projection of one declared field on an entity
// or aggregate-root contract. The use case computes EntityField slices via
// service.JPAFieldAnnotator and hands them to the renderer; the template
// stays "dumb" and just iterates.
//
// Field semantics:
//
//	Type:        Java type token as written in the spec ("UUID", "Long",
//	             "String", "Optional<String>", ...). Generics preserved.
//	Name:        identifier as written in the spec ("id", "email", ...).
//	             Case is preserved; the annotator matches case-insensitively.
//	Annotations: zero or more fully-formed annotation tokens, each already
//	             prefixed with '@' (e.g. "@Id",
//	             "@GeneratedValue(strategy = GenerationType.IDENTITY)").
//	             Order is annotator-defined and stable so the template
//	             output is deterministic (EP02RNF-002).
type EntityField struct {
	Type        string
	Name        string
	Annotations []string
}
