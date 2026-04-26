package fsprofile

import (
	"fmt"
	"log/slog"

	"gopkg.in/yaml.v3"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
)

// toBundleDomain assembles a *model.ProfileBundle from a parsed bundleDTO and
// the eagerly-loaded templates map. It validates that every type entry with a
// non-empty template field references a template that was actually loaded.
// logger is used to emit WARN entries for unrecognised audit_rules kinds; when
// nil, slog.Default() is used.
//
// The caller is responsible for setting bundle.Dir and bundle.Profile.Source
// after this function returns.
func toBundleDomain(dto bundleDTO, templates map[string][]byte, logger *slog.Logger) (*model.ProfileBundle, error) {
	if logger == nil {
		logger = slog.Default()
	}
	// Determine effective name: prefer the singular "language" field for EP-04
	// schema; fall back to the first element of the legacy "languages" list.
	lang := dto.Language
	langs := dto.Languages
	if lang != "" && len(langs) == 0 {
		langs = []string{lang}
	}

	// Map legacy EP-03 classification rules when present.
	legacyRules := make([]model.ProfileRule, 0, len(dto.Rules))
	for _, r := range dto.Rules {
		if r.ClassifyAs == "" {
			continue
		}
		legacyRules = append(legacyRules, model.ProfileRule{
			Match: model.ProfileMatch{
				NodeType:      r.Match.NodeType,
				PathContains:  r.Match.PathContains,
				HasAnnotation: r.Match.HasAnnotation,
				Implements:    r.Match.Implements,
			},
			ClassifyAs: model.ContractType(r.ClassifyAs),
		})
	}

	legacyFiles := make([]model.ProfileFileMatcher, 0, len(dto.Detect.Files))
	for _, f := range dto.Detect.Files {
		legacyFiles = append(legacyFiles, model.ProfileFileMatcher{
			Name:     f.Name,
			Contains: f.Contains,
		})
	}

	legacyMarkers := make([]model.ModuleMarker, 0, len(dto.ModuleDetection.Markers))
	for _, m := range dto.ModuleDetection.Markers {
		legacyMarkers = append(legacyMarkers, model.ModuleMarker{Kind: m.Kind, Value: m.Value})
	}

	profile := &model.FrameworkProfile{
		Name:      dto.Name,
		Languages: langs,
		QueryLang: dto.QueryLang,
		Detect: model.ProfileDetect{
			Files: legacyFiles,
		},
		ModuleDetection: model.ModuleDetection{
			Strategy: dto.ModuleDetection.Strategy,
			Roots:    dto.ModuleDetection.Roots,
			Markers:  legacyMarkers,
		},
		Rules: legacyRules,
	}

	rawTypes := make([]model.ProfileTypeDeclaration, 0, len(dto.Types))
	for _, t := range dto.Types {
		if t.ID == "" {
			return nil, fmt.Errorf("profile %q: type entry missing required id field: %w",
				dto.Name, domerr.ErrProfileInvalid)
		}
		if t.Template != "" {
			if _, ok := templates[t.Template]; !ok {
				return nil, &domerr.TemplateMissingError{
					ProfileName: dto.Name,
					TypeID:      t.ID,
					Template:    t.Template,
				}
			}
		}
		classification := make([]model.ClassificationRule, 0, len(t.Classification))
		for _, c := range t.Classification {
			classification = append(classification, model.ClassificationRule{
				Kind:           c.Kind,
				ImplementsAll:  append([]string(nil), c.ImplementsAll...),
				ImplementsNone: append([]string(nil), c.ImplementsNone...),
				HasAnnotation:  c.HasAnnotation,
				PathContains:   c.PathContains,
			})
		}

		raw, err := yaml.Marshal(t)
		if err != nil {
			return nil, fmt.Errorf("re-marshal type %q: %w", t.ID, err)
		}

		rawTypes = append(rawTypes, model.ProfileTypeDeclaration{
			ID:             t.ID,
			Template:       t.Template,
			Description:    t.Description,
			Classification: classification,
			Raw:            raw,
		})
	}

	var rawPackaging []byte
	if dto.Packaging != nil {
		b, err := yaml.Marshal(dto.Packaging)
		if err != nil {
			return nil, fmt.Errorf("marshal packaging block: %w", err)
		}
		rawPackaging = b
	}

	// Map audit_rules — mirror auditLoader.go semantics: unknown kinds are
	// dropped with a WARN log; unknown severities are fatal.
	rawAuditRules := make([]model.AuditRule, 0, len(dto.AuditRules))
	for _, d := range dto.AuditRules {
		kind := model.AuditRuleKind(d.Kind)
		if !knownAuditRuleKinds[kind] {
			logger.Warn("unknown audit rule kind in bundle profile, dropping rule",
				slog.String("kind", d.Kind),
				slog.String("rule_id", d.ID),
				slog.String("profile", dto.Name),
			)
			continue
		}
		sev := model.AuditSeverity(d.Severity)
		if !knownAuditSeverities[sev] {
			return nil, fmt.Errorf("bundle profile %q: audit rule %q: unknown severity %q: %w",
				dto.Name, d.ID, d.Severity, domerr.ErrProfileInvalid)
		}
		rawAuditRules = append(rawAuditRules, model.AuditRule{
			ID:          d.ID,
			Kind:        kind,
			Severity:    sev,
			Description: d.Description,
			Suggestion:  d.Suggestion,
			Params:      d.Params,
		})
	}

	return &model.ProfileBundle{
		Profile:       profile,
		Templates:     templates,
		RawTypes:      rawTypes,
		RawPackaging:  rawPackaging,
		RawAuditRules: rawAuditRules,
	}, nil
}
