package profile

// ClassificationInput is the per-declaration input to
// profile.ClassifyDeclarativePort. Callers populate it from their own
// language-specific summary type (e.g., model.JavaDeclaration +
// model.JavaFileSummary.Path); the classifier itself is language-agnostic.
//
// Field semantics:
//
//	Kind        — Tree-sitter-style declaration kind. For Java today:
//	              "class_declaration" | "interface_declaration" |
//	              "enum_declaration" | "record_declaration". The classifier
//	              maps profile rule values "class" / "interface" against
//	              this string (see ClassificationRule docs).
//	Name        — simple class/interface name, e.g. "UserServiceImpl".
//	              Reserved for future rule fields (US-004 may add
//	              name_matches); currently unused by the engine.
//	Annotations — simple annotation names without the "@" prefix
//	              (e.g. ["Service", "Transactional"]).
//	Implements  — interface names as written in source — simple or
//	              qualified. Subset matching is name-equality based.
//	Path        — forward-slash path to the source file containing the
//	              declaration (e.g. "src/main/java/.../UserServiceImpl.java").
type ClassificationInput struct {
	Kind        string
	Name        string
	Annotations []string
	Implements  []string
	Path        string
}
