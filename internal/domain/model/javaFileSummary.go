package model

// JavaFileSummary is the structured output produced by parsing a single Java file.
type JavaFileSummary struct {
	Path         string   // forward-slash path
	Package      string   // "com.app.user.port.in"
	Imports      []string // fully-qualified types
	Declarations []JavaDeclaration
	HasErrors    bool // true if Tree-sitter reported ERROR nodes
}

// JavaField represents one field declared in a class body. Used by audit
// rule kind field_type_layer_violation. Empty slice when the declaration
// is not a class or has no fields.
type JavaField struct {
	Name string // field identifier, e.g. "repository"
	Type string // raw type token as it appears in source, e.g. "UserRepositoryJpa" or "List<User>"
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
	Fields               []JavaField  // NEW — empty for non-class declarations and for classes with no fields
}

// JavaMethod represents a method extracted from a Java declaration.
type JavaMethod struct {
	Signature string // as defined in contract §3.4
}
