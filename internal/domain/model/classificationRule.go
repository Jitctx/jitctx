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
//	                  nil/empty = no constraint.
//	ImplementsNone  — if ANY name in this list appears in the declaration's
//	                  implemented-interfaces list, the rule fails.
//	                  nil/empty = no constraint.
//	HasAnnotation   — single annotation name, case-insensitive, "@" prefix
//	                  tolerated. Empty string = no constraint.
//	PathContains    — substring match against the forward-slash file path.
//	                  Empty string = no constraint.
//
// Future fields (path_matches, has_annotations_any, extends) are deferred
// to EP04US-004; YAML decoding is lenient so authoring them now is harmless.
type ClassificationRule struct {
	Kind           string
	ImplementsAll  []string
	ImplementsNone []string
	HasAnnotation  string
	PathContains   string
}
