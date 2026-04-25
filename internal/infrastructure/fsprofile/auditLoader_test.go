package fsprofile_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
)

// minimalProfileWithAuditRules returns the bytes for a complete profile YAML
// that includes the supplied audit_rules block verbatim.
func minimalProfileWithAuditRules(auditRulesYAML string) []byte {
	base := `name: test-profile
languages: [java]
query_lang: java
detect:
  files: []
module_detection:
  strategy: hexagonal
  roots: []
  markers: []
rules: []
`
	if auditRulesYAML != "" {
		base += auditRulesYAML
	}
	return []byte(base)
}

// writeProfile writes content to <dir>/<name>.yaml.
func writeProfile(t *testing.T, dir, name string, content []byte) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name+".yaml"), content, 0o644))
}

// ----------------------------------------------------------------------------
// Scenario 1 – KnownFields strictness
// A top-level key not declared in profileDTO (e.g. the typo "audit_rule:")
// must return an error wrapping ErrProfileInvalid.
// ----------------------------------------------------------------------------

func TestAuditLoader_KnownFieldsStrictness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content []byte
	}{
		{
			name: "unknown-top-level-key-audit_rule-typo",
			content: []byte(`name: test-profile
languages: [java]
query_lang: java
detect:
  files: []
module_detection:
  strategy: hexagonal
  roots: []
  markers: []
rules: []
audit_rule:
  - id: r1
    kind: annotation_path_mismatch
    severity: ERROR
`),
		},
		{
			name: "unknown-field-inside-audit_rules-entry",
			content: []byte(`name: test-profile
languages: [java]
query_lang: java
detect:
  files: []
module_detection:
  strategy: hexagonal
  roots: []
  markers: []
rules: []
audit_rules:
  - id: r1
    kind: annotation_path_mismatch
    severity: ERROR
    unknown_extra_field: boom
`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			writeProfile(t, dir, "test-profile", tc.content)

			l := fsprofile.New(dir)
			_, err := l.LoadAuditRules(context.Background(), "test-profile")
			require.Error(t, err)
			require.ErrorIs(t, err, domerr.ErrProfileInvalid)
		})
	}
}

// ----------------------------------------------------------------------------
// Scenario 2 – Unknown kind is dropped silently
// A rule whose kind is unrecognised must be dropped from the returned slice.
// Other valid rules in the same profile must still be present.
// The load itself must succeed (no error).
// ----------------------------------------------------------------------------

func TestAuditLoader_UnknownKindIsDropped(t *testing.T) {
	t.Parallel()

	content := minimalProfileWithAuditRules(`audit_rules:
  - id: r-unknown
    kind: made_up_kind
    severity: WARNING
    description: will be dropped
    suggestion: ""
  - id: r-valid
    kind: annotation_path_mismatch
    severity: ERROR
    description: kept rule
    suggestion: move file
`)

	dir := t.TempDir()
	writeProfile(t, dir, "test-profile", content)

	l := fsprofile.New(dir)
	rules, err := l.LoadAuditRules(context.Background(), "test-profile")
	require.NoError(t, err)

	// The bad rule must be absent.
	for _, r := range rules {
		require.NotEqual(t, "r-unknown", r.ID, "unknown-kind rule must be dropped")
		require.NotEqual(t, model.AuditRuleKind("made_up_kind"), r.Kind)
	}

	// The valid rule must survive.
	require.Len(t, rules, 1)
	require.Equal(t, "r-valid", rules[0].ID)
	require.Equal(t, model.AuditKindAnnotationPathMismatch, rules[0].Kind)
	require.Equal(t, model.AuditSeverityError, rules[0].Severity)
}

// ----------------------------------------------------------------------------
// Scenario 3 – Unknown severity is fatal
// A rule with an unrecognised severity value must cause LoadAuditRules to
// return an error wrapping ErrProfileInvalid.
// ----------------------------------------------------------------------------

func TestAuditLoader_UnknownSeverityIsFatal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		severity string
	}{
		{"typo-with-dash", "fatal-typo"},
		{"lowercase-error", "error"},
		{"empty-severity", ""},
		{"unknown-word", "CRITICAL"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			yaml := minimalProfileWithAuditRules(`audit_rules:
  - id: r1
    kind: annotation_path_mismatch
    severity: ` + tc.severity + `
    description: bad severity rule
    suggestion: ""
`)
			dir := t.TempDir()
			writeProfile(t, dir, "test-profile", yaml)

			l := fsprofile.New(dir)
			_, err := l.LoadAuditRules(context.Background(), "test-profile")
			require.Error(t, err)
			require.ErrorIs(t, err, domerr.ErrProfileInvalid)
		})
	}
}

// ----------------------------------------------------------------------------
// Scenario 4 – Missing audit_rules key
// An old-style profile without an audit_rules: key loads cleanly and returns
// an empty (non-nil) slice with no error.
// ----------------------------------------------------------------------------

func TestAuditLoader_MissingAuditRulesKeyReturnsEmptySlice(t *testing.T) {
	t.Parallel()

	content := []byte(`name: legacy-profile
languages: [java]
query_lang: java
detect:
  files: []
module_detection:
  strategy: hexagonal
  roots: []
  markers: []
rules: []
`)

	dir := t.TempDir()
	writeProfile(t, dir, "legacy-profile", content)

	l := fsprofile.New(dir)
	rules, err := l.LoadAuditRules(context.Background(), "legacy-profile")
	require.NoError(t, err)
	require.NotNil(t, rules, "returned slice must not be nil")
	require.Empty(t, rules, "returned slice must be empty for profiles without audit_rules")
}
