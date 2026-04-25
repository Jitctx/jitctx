package service

import "strings"

// JavaIdentifierUtils centralises the small string-shape helpers used by
// the scaffold use case so the conversions are testable without touching
// the use case body.
type JavaIdentifierUtils struct{}

// NewJavaIdentifierUtils returns a stateless JavaIdentifierUtils.
func NewJavaIdentifierUtils() JavaIdentifierUtils { return JavaIdentifierUtils{} }

// FieldNameFromType lowercases the first rune of a PascalCase Java type
// name to produce a camelCase field name. Examples:
//
//	"UserRepository"  → "userRepository"
//	"URLBuilder"      → "uRLBuilder"  (intentionally simple — no acronym
//	                                    smarts; matches Spring/Lombok docs)
//	""                → ""            (caller validates non-empty upstream)
func (JavaIdentifierUtils) FieldNameFromType(typeName string) string {
	if typeName == "" {
		return ""
	}
	runes := []rune(typeName)
	runes[0] = []rune(strings.ToLower(string(runes[0])))[0]
	return string(runes)
}

// PackageFromRelativePath converts a slash-form relative path produced by
// ContractPathMapper (e.g., "port/in/Foo.java") into its dot-form package
// suffix ("port.in"). The basename and trailing ".java" are stripped.
func (JavaIdentifierUtils) PackageFromRelativePath(relPath string) string {
	idx := strings.LastIndex(relPath, "/")
	if idx < 0 {
		// No directory component — e.g. "Foo.java"
		return ""
	}
	dir := relPath[:idx]
	return strings.ReplaceAll(dir, "/", ".")
}

// FQN concatenates modulePackage + "." + PackageFromRelativePath(relPath)
// + "." + simpleTypeName. modulePackage and simpleTypeName are NOT validated.
func (u JavaIdentifierUtils) FQN(modulePackage, relPath, simpleTypeName string) string {
	pkg := u.PackageFromRelativePath(relPath)
	if pkg == "" {
		return modulePackage + "." + simpleTypeName
	}
	return modulePackage + "." + pkg + "." + simpleTypeName
}
