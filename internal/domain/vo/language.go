package vo

// Language identifies a programming language by its canonical, lowercase id.
// Used by both the framework-profile metadata block and the bundled
// Tree-sitter query registry.
type Language string

const (
	LanguageGo         Language = "go"
	LanguageJava       Language = "java"
	LanguageTypeScript Language = "typescript"
	LanguagePython     Language = "python"
)

// String returns the canonical id as a plain string. Implements fmt.Stringer
// so that error messages can interpolate language values without repeated
// type conversions.
func (l Language) String() string { return string(l) }

// ParseLanguage returns the Language matching the given canonical id.
// Returns the zero value ("") and false when the id is empty or unknown.
// The set of recognised ids is the const block above; the *supported* set
// (i.e. those for which a bundled query directory exists) is owned by the
// infrastructure registry, not by this VO.
func ParseLanguage(id string) (Language, bool) {
	switch Language(id) {
	case LanguageGo, LanguageJava, LanguageTypeScript, LanguagePython:
		return Language(id), true
	default:
		return Language(""), false
	}
}
