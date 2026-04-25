package fsprofile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
)

// NewAuditRulesLoader returns the *Loader as a LoadAuditRulesPort. The same
// instance can be passed to both LoadProfilePort and LoadAuditRulesPort in
// wire.go, satisfying both ISP ports from one struct.
func NewAuditRulesLoader(userDir string, logger *slog.Logger) *Loader {
	return NewWithLogger(userDir, logger)
}

// LoadAuditRules loads the audit_rules section from the named profile.
// If the profile file has no audit_rules: key (or an empty list) an empty
// slice is returned — that is not an error (clean-state profiles are valid).
// Unknown rule kinds are dropped and logged as warnings (slog.Warn).
// Unknown severity values are fatal and return a wrapped ErrProfileInvalid.
func (l *Loader) LoadAuditRules(ctx context.Context, profileName string) ([]model.AuditRule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// SEC-001: reject names that attempt path traversal.
	if strings.ContainsAny(profileName, `/\`) || strings.Contains(profileName, "..") {
		return nil, fmt.Errorf("profile %q: invalid name: %w", profileName, domerr.ErrProfileInvalid)
	}

	data, err := l.readProfileData(profileName)
	if err != nil {
		return nil, err
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var dto profileDTO
	if err := dec.Decode(&dto); err != nil {
		return nil, fmt.Errorf("profile %q: decode: %w: %w", profileName, err, domerr.ErrProfileInvalid)
	}

	if len(dto.AuditRules) == 0 {
		return []model.AuditRule{}, nil
	}

	var rules []model.AuditRule
	for _, d := range dto.AuditRules {
		kind := model.AuditRuleKind(d.Kind)
		if !knownAuditRuleKinds[kind] {
			l.logger.Warn("unknown audit rule kind in profile, dropping rule",
				slog.String("kind", d.Kind),
				slog.String("rule_id", d.ID),
				slog.String("profile", profileName),
			)
			continue
		}
		sev := model.AuditSeverity(d.Severity)
		if !knownAuditSeverities[sev] {
			return nil, fmt.Errorf("profile %q: audit rule %q: unknown severity %q: %w",
				profileName, d.ID, d.Severity, domerr.ErrProfileInvalid)
		}
		rules = append(rules, model.AuditRule{
			ID:          d.ID,
			Kind:        kind,
			Severity:    sev,
			Description: d.Description,
			Suggestion:  d.Suggestion,
			Params:      d.Params,
		})
	}
	if rules == nil {
		rules = []model.AuditRule{}
	}
	return rules, nil
}

// readProfileData locates and reads the raw YAML bytes for the named profile.
func (l *Loader) readProfileData(profileName string) ([]byte, error) {
	rootAbs, err := filepath.Abs(l.userDir)
	if err != nil {
		return nil, fmt.Errorf("profile %q: resolve profiles dir: %w", profileName, err)
	}
	for _, ext := range []string{".yaml", ".yml"} {
		candidate := filepath.Clean(filepath.Join(rootAbs, profileName+ext))
		if !strings.HasPrefix(candidate, rootAbs+string(filepath.Separator)) {
			return nil, fmt.Errorf("profile %q: escapes profiles dir: %w", profileName, domerr.ErrProfileInvalid)
		}
		data, err := os.ReadFile(candidate)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read profile %q: %w", candidate, err)
		}
		return data, nil
	}
	return nil, fmt.Errorf("profile %q not found: %w", profileName, domerr.ErrProfileInvalid)
}
