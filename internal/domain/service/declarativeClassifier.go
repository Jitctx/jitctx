package service

import (
	"context"
	"strings"

	"github.com/jitctx/jitctx/internal/domain/model"
	profileport "github.com/jitctx/jitctx/internal/domain/port/profile"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// DeclarativeClassifier is the stateless engine that backs
// profile.ClassifyDeclarativePort. It is exposed as a struct (zero-state)
// so wire.go can inject it as a port-satisfying value while keeping the
// underlying functions usable in tests without construction noise.
type DeclarativeClassifier struct{}

// NewDeclarativeClassifier returns a zero-state classifier. Construction
// is cheap; tests construct one per case for clarity.
func NewDeclarativeClassifier() *DeclarativeClassifier {
	return &DeclarativeClassifier{}
}

// ClassifyDeclarative implements profile.ClassifyDeclarativePort.
// Iterates types in declared order; for each type, returns the type ID if
// ANY rule in the type's Classification slice matches the input. Skips
// types with empty Classification (no rules ⇒ no match).
func (DeclarativeClassifier) ClassifyDeclarative(
	ctx context.Context,
	input profilevo.ClassificationInput,
	types []model.ProfileTypeDeclaration,
) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	matched := make([]string, 0, len(types))
	seen := make(map[string]struct{}, len(types))
	for _, td := range types {
		if _, dup := seen[td.ID]; dup {
			continue
		}
		if len(td.Classification) == 0 {
			continue
		}
		for _, rule := range td.Classification {
			if matchClassificationRule(rule, input) {
				matched = append(matched, td.ID)
				seen[td.ID] = struct{}{}
				break // OR-across-entries — first match settles the type
			}
		}
	}
	return matched, nil
}

// matchClassificationRule returns true when every non-zero field of the
// rule is satisfied by the input (AND semantics within one rule).
func matchClassificationRule(rule model.ClassificationRule, in profilevo.ClassificationInput) bool {
	if rule.Kind != "" && !kindMatches(rule.Kind, in.Kind) {
		return false
	}
	if rule.HasAnnotation != "" && !annotationMatches(rule.HasAnnotation, in.Annotations) {
		return false
	}
	if rule.PathContains != "" && !strings.Contains(in.Path, rule.PathContains) {
		return false
	}
	if len(rule.ImplementsAll) > 0 && !implementsAllOf(rule.ImplementsAll, in.Implements) {
		return false
	}
	if len(rule.ImplementsNone) > 0 && implementsAnyOf(rule.ImplementsNone, in.Implements) {
		return false
	}
	return true
}

// kindMatches handles the "class" / "interface" abbreviations used in
// profile.yaml. Maps to Tree-sitter NodeType values:
//
//	"class"     -> "class_declaration" or "record_declaration"
//	"interface" -> "interface_declaration"
//
// Unknown values never match (forward-compatible with US-007 warnings).
func kindMatches(profileKind, inputKind string) bool {
	switch profileKind {
	case "class":
		return inputKind == "class_declaration" || inputKind == "record_declaration"
	case "interface":
		return inputKind == "interface_declaration"
	default:
		return false
	}
}

// annotationMatches case-insensitively compares an annotation name
// (with or without leading "@") against the input list.
func annotationMatches(target string, annotations []string) bool {
	target = strings.TrimPrefix(target, "@")
	for _, a := range annotations {
		if strings.EqualFold(a, target) {
			return true
		}
	}
	return false
}

// implementsAllOf returns true when every name in want appears in have
// (subset match — have may carry extras).
func implementsAllOf(want, have []string) bool {
	for _, w := range want {
		found := false
		for _, h := range have {
			if h == w {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// implementsAnyOf returns true when any name in candidates appears in have.
func implementsAnyOf(candidates, have []string) bool {
	for _, c := range candidates {
		for _, h := range have {
			if h == c {
				return true
			}
		}
	}
	return false
}

// Compile-time assertion that DeclarativeClassifier satisfies ClassifyDeclarativePort.
var _ profileport.ClassifyDeclarativePort = (*DeclarativeClassifier)(nil)
