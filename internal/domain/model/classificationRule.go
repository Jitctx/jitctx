package model

// ClassificationRule is a single AND-internal classification entry on a
// ProfileTypeDeclaration. A type matches a code element when ANY rule in
// its Classification slice matches.
//
// Field semantics (each non-zero field is an AND constraint inside one rule):
//
//	Kind            — "class" | "interface" (extensible; unknown values never
//	                  match in US-002). Empty string = no constraint.
//	ImplementsAll   — every name in this list must appear in the declaration's
//	                  implemented-interfaces list (subset match — extras allowed).
//	                  Each entry is a single-`*` glob (anchored at start AND
//	                  end of the implemented-interface name): "*UseCase" matches
//	                  any interface ending in "UseCase"; "Foo*" matches any
//	                  starting with "Foo"; "Foo" requires literal equality.
//	                  Java identifiers cannot contain `*`, so existing literal
//	                  entries written before EP04US-004 keep matching exactly
//	                  the same set of declarations.
//	                  nil/empty = no constraint.
//	ImplementsNone  — same glob semantics as ImplementsAll. If ANY name in
//	                  the declaration's implemented-interfaces list matches
//	                  ANY pattern in this slice, the rule fails.
//	                  nil/empty = no constraint.
//	HasAnnotation   — single annotation name, case-insensitive, "@" prefix
//	                  tolerated. Empty string = no constraint.
//	PathContains    — substring match against the forward-slash file path.
//	                  Empty string = no constraint.
type ClassificationRule struct {
	Kind           string
	ImplementsAll  []string
	ImplementsNone []string
	HasAnnotation  string
	PathContains   string
}
