package fsprofile

import (
	"fmt"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
)

// knownAuditRuleKinds is the set of recognised AuditRuleKind values.
var knownAuditRuleKinds = map[model.AuditRuleKind]bool{
	model.AuditKindAnnotationPathMismatch:  true,
	model.AuditKindImplementsPathMismatch:  true,
	model.AuditKindInterfaceNaming:         true,
	model.AuditKindForbiddenImport:         true,
	model.AuditKindFieldTypeLayerViolation: true,
}

// knownAuditSeverities is the set of recognised AuditSeverity values.
var knownAuditSeverities = map[model.AuditSeverity]bool{
	model.AuditSeverityError:   true,
	model.AuditSeverityWarning: true,
	model.AuditSeverityInfo:    true,
}

// toDomain converts a profileDTO to a model.FrameworkProfile, validating required fields.
func toDomain(d profileDTO) (*model.FrameworkProfile, error) {
	if d.Name == "" {
		return nil, fmt.Errorf("profile name is empty: %w", domerr.ErrProfileInvalid)
	}

	rules := make([]model.ProfileRule, 0, len(d.Rules))
	for _, r := range d.Rules {
		if r.ClassifyAs == "" {
			return nil, fmt.Errorf("rule with empty classify_as: %w", domerr.ErrProfileInvalid)
		}
		ct := model.ContractType(r.ClassifyAs)
		if !isKnownContractType(ct) {
			return nil, fmt.Errorf("unknown classify_as %q: %w", r.ClassifyAs, domerr.ErrProfileInvalid)
		}
		rules = append(rules, model.ProfileRule{
			Match: model.ProfileMatch{
				NodeType:      r.Match.NodeType,
				PathContains:  r.Match.PathContains,
				HasAnnotation: r.Match.HasAnnotation,
				Implements:    r.Match.Implements,
			},
			ClassifyAs: ct,
		})
	}

	if d.ModuleDetection.Strategy != "" && d.ModuleDetection.Strategy != "hexagonal" {
		return nil, fmt.Errorf("unknown module_detection strategy %q: %w",
			d.ModuleDetection.Strategy, domerr.ErrProfileInvalid)
	}

	files := make([]model.ProfileFileMatcher, 0, len(d.Detect.Files))
	for _, f := range d.Detect.Files {
		files = append(files, model.ProfileFileMatcher{
			Name:     f.Name,
			Contains: f.Contains,
		})
	}

	markers := make([]model.ModuleMarker, 0, len(d.ModuleDetection.Markers))
	for _, m := range d.ModuleDetection.Markers {
		markers = append(markers, model.ModuleMarker{Kind: m.Kind, Value: m.Value})
	}

	return &model.FrameworkProfile{
		Name:      d.Name,
		Languages: d.Languages,
		QueryLang: d.QueryLang,
		Detect: model.ProfileDetect{
			Files: files,
		},
		ModuleDetection: model.ModuleDetection{
			Strategy: d.ModuleDetection.Strategy,
			Roots:    d.ModuleDetection.Roots,
			Markers:  markers,
		},
		Rules: rules,
	}, nil
}

func isKnownContractType(ct model.ContractType) bool {
	switch ct {
	case model.ContractInputPort, model.ContractOutputPort, model.ContractEntity,
		model.ContractAggregate, model.ContractService, model.ContractRestAdapter,
		model.ContractJPAAdapter:
		return true
	}
	return false
}
