package model

// SupertypeKind enumerates the two possible parameterized-supertype origins.
// The string values are stable and may be referenced by profile authors via
// the rule's "supertype_kind" param ("extends" | "implements" | "" meaning either).
type SupertypeKind string

const (
	SupertypeKindExtends    SupertypeKind = "extends"
	SupertypeKindImplements SupertypeKind = "implements"
)

// ParameterizedSupertype represents one parameterized supertype declared on a
// class (either via `extends X<...>` or `implements Y<...>`). The outer name
// is captured verbatim as it appears in source; type arguments are captured as
// the comma-separated raw text tokens between the outermost angle brackets,
// with surrounding whitespace trimmed but their inner structure preserved.
//
// The slice ONLY carries entries from declarations whose source form contains
// `<...>` brackets. Bare-name supertypes (e.g. `implements Runnable`) continue
// to populate the Implements and Extends fields exactly as today (backward
// compatibility). TypeArgs are split on TOP-LEVEL commas (depth-zero) — no
// structural parsing of the inner tokens is performed by the domain model.
//
// PC01RF-006, PC01RF-010.
type ParameterizedSupertype struct {
	Kind     SupertypeKind // "extends" | "implements"
	Outer    string        // outer type name verbatim (simple or scoped)
	TypeArgs []string      // raw type-argument tokens, trimmed; len == arity
}

// JavaFileSummary is the structured output produced by parsing a single Java file.
type JavaFileSummary struct {
	Path         string   // forward-slash path
	Package      string   // "com.app.user.port.in"
	Imports      []string // fully-qualified types
	Declarations []JavaDeclaration
	HasErrors    bool // true if Tree-sitter reported ERROR nodes
}

// JavaField represents one field declared in a class body. Used by audit
// rule kinds field_type_layer_violation and forbidden_annotations. Empty
// slice when the declaration is not a class or has no fields.
type JavaField struct {
	Name        string   // field identifier, e.g. "repository"
	Type        string   // raw type token as it appears in source, e.g. "UserRecord" or "List<User>"
	Annotations []string // simple names, no leading @ (e.g. ["Marker"]). Empty when not extracted.
	Line        int      // 1-based line of the field_declaration node. 0 if unknown.
}

// JavaDeclaration represents a top-level type declaration in a Java file.
type JavaDeclaration struct {
	NodeType             string       // "class_declaration" | "interface_declaration" | "enum_declaration" | "record_declaration"
	Name                 string       // simple name
	Annotations          []string     // simple names, no leading @ (e.g. ["Entity", "Table"])
	QualifiedAnnotations []string     // same length and order as Annotations; entries are fully-qualified when the source wrote @a.b.C, else equal to the simple name
	Implements           []string     // simple and/or qualified interface names
	Extends              []string     // superclass simple or qualified name (length 0 or 1)
	Methods              []JavaMethod // class/interface-owned methods
	Fields               []JavaField  // empty for non-class declarations and for classes with no fields

	// ParameterizedSupertypes carries every parameterized supertype
	// declared on this class via `extends X<...>` or `implements Y<...>`.
	// Empty for non-class declarations and for classes whose supertypes are
	// all non-parameterized. Order preserves source order: the Extends entry
	// first (if any), then Implements entries in their declaration order.
	// Populated by the language adapter; evaluators must treat nil and empty
	// slice identically (no parameterized supertypes detected).
	// PC01RF-006, PC01RF-010.
	ParameterizedSupertypes []ParameterizedSupertype

	// AnnotationArgs maps simple annotation name → text of the first
	// positional argument as it appears in source (including quotes for
	// string literals and ".class" suffixes for class literals). Empty
	// string when the annotation is a marker (no argument list). Keys are
	// a subset of Annotations[].
	//
	// The map is keyed by SIMPLE annotation name (matching entries in
	// Annotations[]). When the same simple annotation appears more than
	// once on the same declaration (rare) the LAST occurrence wins.
	//
	// Empty map (or nil) when the language adapter did not extract
	// arguments — evaluators must treat both nil and "" entries as
	// "no argument captured".
	//
	// PC01RF-007.
	AnnotationArgs map[string]string
}

// JavaMethod represents a method extracted from a Java declaration.
type JavaMethod struct {
	Signature   string   // as defined in contract §3.4
	Name        string   // simple method name (identifier), e.g. "testFindUser"
	Annotations []string // simple annotation names, no leading @ (e.g. ["Override"]). Empty when not extracted.
	Line        int      // 1-based line of the method_declaration node. 0 if unknown.
}
