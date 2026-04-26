package format

import "strings"

// formatTypeLabel renders a manifest-sourced []string of types as a
// single human-readable label. Empty/nil → "" (caller may render
// empty parens). Single → the string. Multi → "+"-joined.
func formatTypeLabel(types []string) string {
	if len(types) == 0 {
		return ""
	}
	if len(types) == 1 {
		return types[0]
	}
	return strings.Join(types, "+")
}

// normaliseStringSlice is a small util kept here so both yaml.go
// and contracts.go can use the same nil→[]string{} normalisation
// without round-tripping through the domain layer.
func normaliseStringSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
