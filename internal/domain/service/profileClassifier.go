package service

import (
	"path/filepath"
	"strings"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// ClassifyDeclaration applies the profile rules to a JavaDeclaration and returns
// the first matching ContractType. Returns ("", false) if no rule matches.
func ClassifyDeclaration(
	d model.JavaDeclaration,
	filePath string,
	prof *model.FrameworkProfile,
) (model.ContractType, bool) {
	// Normalize path separators.
	normalPath := filepath.ToSlash(filePath)
	for _, rule := range prof.Rules {
		if matchRule(rule.Match, d, normalPath) {
			return rule.ClassifyAs, true
		}
	}
	return "", false
}

// matchRule returns true when all non-empty fields in the match block satisfy the declaration.
func matchRule(m model.ProfileMatch, d model.JavaDeclaration, filePath string) bool {
	if m.NodeType != "" && m.NodeType != d.NodeType {
		return false
	}
	if m.PathContains != "" && !strings.Contains(filePath, m.PathContains) {
		return false
	}
	if m.HasAnnotation != "" && !hasAnnotation(d.Annotations, m.HasAnnotation) {
		return false
	}
	if m.Implements != "" && !implementsGlob(d.Implements, m.Implements) {
		return false
	}
	return true
}

func hasAnnotation(annotations []string, target string) bool {
	target = strings.TrimPrefix(target, "@")
	for _, a := range annotations {
		if strings.EqualFold(a, target) {
			return true
		}
	}
	return false
}

// implementsGlob supports a single leading or trailing wildcard (*Foo or Foo*).
func implementsGlob(implements []string, pattern string) bool {
	for _, iface := range implements {
		if globMatch(pattern, iface) {
			return true
		}
	}
	return false
}

// globMatch supports a single '*' wildcard.
func globMatch(pattern, s string) bool {
	prefix, suffix, found := strings.Cut(pattern, "*")
	if !found {
		return pattern == s
	}
	return strings.HasPrefix(s, prefix) && strings.HasSuffix(s, suffix)
}
