package model

// JavaFileSummary is the structured output produced by parsing a single Java file.
type JavaFileSummary struct {
	Path         string   // forward-slash path
	Package      string   // "com.app.user.port.in"
	Imports      []string // fully-qualified types
	Declarations []JavaDeclaration
	HasErrors    bool // true if Tree-sitter reported ERROR nodes
}

// JavaDeclaration represents a top-level type declaration in a Java file.
type JavaDeclaration struct {
	NodeType    string       // "class_declaration" | "interface_declaration" | ...
	Name        string       // simple name
	Annotations []string     // simple names, no leading @
	Implements  []string     // simple and/or qualified interface names
	Extends     []string     // superclass simple or qualified name (length 0 or 1)
	Methods     []JavaMethod // class/interface-owned methods
}

// JavaMethod represents a method extracted from a Java declaration.
type JavaMethod struct {
	Signature string // as defined in contract §3.4
}
