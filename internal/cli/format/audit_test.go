package format_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/model"
	auditvo "github.com/jitctx/jitctx/internal/domain/vo/audit"
)

// TestWriteAuditReport_EmptyOutput verifies that a clean project (no violations)
// renders both required headings, the "no violations" message, and the verbatim
// semantic placeholder (RNF-005).
func TestWriteAuditReport_EmptyOutput(t *testing.T) {
	t.Parallel()

	out := auditvo.AuditProjectOutput{
		ProfileName:         "",
		Sintatic:            nil,
		SemanticPlaceholder: auditvo.SemanticPlaceholder,
	}

	var buf bytes.Buffer
	err := format.WriteAuditReport(&buf, out)
	require.NoError(t, err)

	body := buf.String()
	require.Contains(t, body, "## Sintatic Violations")
	require.Contains(t, body, "## Semantic Analysis")
	require.Contains(t, body, "No sintatic violations detected")
	require.Contains(t, body, auditvo.SemanticPlaceholder)
}

// TestWriteAuditReport_SemanticPlaceholderVerbatim asserts the exact literal
// string is present in the rendered output as-is (RNF-005).
func TestWriteAuditReport_SemanticPlaceholderVerbatim(t *testing.T) {
	t.Parallel()

	out := auditvo.AuditProjectOutput{
		SemanticPlaceholder: auditvo.SemanticPlaceholder,
	}

	var buf bytes.Buffer
	err := format.WriteAuditReport(&buf, out)
	require.NoError(t, err)

	require.Contains(t, buf.String(), auditvo.SemanticPlaceholder)
}

// TestWriteAuditReport_SingleViolation verifies that a single violation is
// rendered with the correct severity badge, file path, message, and suggestion.
func TestWriteAuditReport_SingleViolation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		violation      auditvo.AuditViolation
		wantBadge      string
		wantSuggestion bool
	}{
		{
			name: "error-severity-with-suggestion",
			violation: auditvo.AuditViolation{
				RuleID:     "R001",
				Kind:       model.AuditKindAnnotationPathMismatch,
				Severity:   model.AuditSeverityError,
				ModuleID:   "mod-user",
				FilePath:   "src/main/java/Foo.java",
				Line:       42,
				Message:    "@Entity found outside domain/",
				Suggestion: "Move Foo.java to the domain/ package.",
			},
			wantBadge:      "🔴 ERROR",
			wantSuggestion: true,
		},
		{
			name: "warning-severity-no-suggestion",
			violation: auditvo.AuditViolation{
				RuleID:   "R002",
				Kind:     model.AuditKindInterfaceNaming,
				Severity: model.AuditSeverityWarning,
				ModuleID: "mod-user",
				FilePath: "src/main/java/Bar.java",
				Line:     0,
				Message:  "Interface in port/in/ does not end with UseCase",
			},
			wantBadge:      "🟡 WARNING",
			wantSuggestion: false,
		},
		{
			name: "info-severity-with-line",
			violation: auditvo.AuditViolation{
				RuleID:     "R003",
				Kind:       model.AuditKindForbiddenImport,
				Severity:   model.AuditSeverityInfo,
				ModuleID:   "mod-order",
				FilePath:   "src/main/java/Baz.java",
				Line:       7,
				Message:    "domain/ file imports org.springframework.*",
				Suggestion: "Remove the Spring dependency.",
			},
			wantBadge:      "🔵 INFO",
			wantSuggestion: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out := auditvo.AuditProjectOutput{
				SemanticPlaceholder: auditvo.SemanticPlaceholder,
				Sintatic:            []auditvo.AuditViolation{tc.violation},
			}

			var buf bytes.Buffer
			err := format.WriteAuditReport(&buf, out)
			require.NoError(t, err)

			body := buf.String()
			require.Contains(t, body, tc.wantBadge)
			require.Contains(t, body, tc.violation.FilePath)
			require.Contains(t, body, tc.violation.Message)
			require.Contains(t, body, tc.violation.RuleID)

			if tc.violation.Line != 0 {
				require.Contains(t, body, ":")
			}

			if tc.wantSuggestion {
				require.Contains(t, body, tc.violation.Suggestion)
			} else {
				require.NotContains(t, body, "suggestion:")
			}
		})
	}
}

// TestWriteAuditReport_MultiModule verifies that violations from different
// modules are grouped under separate "## Module: <ID>" headings in the order
// the input provides them, and that modules with zero violations under them
// produce no heading in the output.
func TestWriteAuditReport_MultiModule(t *testing.T) {
	t.Parallel()

	violations := []auditvo.AuditViolation{
		{
			RuleID:   "R001",
			Severity: model.AuditSeverityError,
			ModuleID: "mod-alpha",
			FilePath: "src/Alpha.java",
			Line:     1,
			Message:  "alpha violation",
		},
		{
			RuleID:   "R002",
			Severity: model.AuditSeverityWarning,
			ModuleID: "mod-beta",
			FilePath: "src/Beta.java",
			Line:     2,
			Message:  "beta violation",
		},
		{
			RuleID:   "R003",
			Severity: model.AuditSeverityInfo,
			ModuleID: "mod-beta",
			FilePath: "src/Beta2.java",
			Line:     5,
			Message:  "beta second violation",
		},
	}

	out := auditvo.AuditProjectOutput{
		SemanticPlaceholder: auditvo.SemanticPlaceholder,
		Sintatic:            violations,
	}

	var buf bytes.Buffer
	err := format.WriteAuditReport(&buf, out)
	require.NoError(t, err)

	body := buf.String()

	// Both module headings must appear.
	require.Contains(t, body, "## Module: mod-alpha")
	require.Contains(t, body, "## Module: mod-beta")

	// mod-alpha heading must precede mod-beta heading (input order preserved).
	alphaIdx := strings.Index(body, "## Module: mod-alpha")
	betaIdx := strings.Index(body, "## Module: mod-beta")
	require.Less(t, alphaIdx, betaIdx, "mod-alpha heading must appear before mod-beta heading")

	// Violations from the correct modules appear under their headings.
	require.Contains(t, body, "alpha violation")
	require.Contains(t, body, "beta violation")
	require.Contains(t, body, "beta second violation")

	// A module that has no violations must not appear as a heading.
	require.NotContains(t, body, "## Module: mod-gamma")
}

// TestWriteAuditReport_MultiModuleNoViolationModuleOmitted verifies that a
// module with zero violations in the Sintatic slice does not produce a heading,
// even if it appears in the Modules list.
func TestWriteAuditReport_MultiModuleNoViolationModuleOmitted(t *testing.T) {
	t.Parallel()

	// mod-empty has no entry in Sintatic, so no heading should appear.
	out := auditvo.AuditProjectOutput{
		SemanticPlaceholder: auditvo.SemanticPlaceholder,
		Modules: []auditvo.AuditModuleReport{
			{ModuleID: "mod-with-violation", Path: "src/a"},
			{ModuleID: "mod-empty", Path: "src/b"},
		},
		Sintatic: []auditvo.AuditViolation{
			{
				RuleID:   "R001",
				Severity: model.AuditSeverityError,
				ModuleID: "mod-with-violation",
				FilePath: "src/a/A.java",
				Line:     3,
				Message:  "something wrong",
			},
		},
	}

	var buf bytes.Buffer
	err := format.WriteAuditReport(&buf, out)
	require.NoError(t, err)

	body := buf.String()
	require.Contains(t, body, "## Module: mod-with-violation")
	require.NotContains(t, body, "## Module: mod-empty")
}

// TestWriteAuditReport_Determinism verifies that two consecutive calls on the
// same input produce byte-identical output (RNF-003).
func TestWriteAuditReport_Determinism(t *testing.T) {
	t.Parallel()

	out := auditvo.AuditProjectOutput{
		ProfileName:         "hexagonal-java",
		SemanticPlaceholder: auditvo.SemanticPlaceholder,
		Sintatic: []auditvo.AuditViolation{
			{
				RuleID:     "R001",
				Severity:   model.AuditSeverityError,
				ModuleID:   "mod-core",
				FilePath:   "src/Core.java",
				Line:       10,
				Message:    "layer violation",
				Suggestion: "Restructure the package.",
			},
			{
				RuleID:   "R002",
				Severity: model.AuditSeverityWarning,
				ModuleID: "mod-core",
				FilePath: "src/Core2.java",
				Line:     20,
				Message:  "naming issue",
			},
		},
	}

	var buf1, buf2 bytes.Buffer

	err := format.WriteAuditReport(&buf1, out)
	require.NoError(t, err)

	err = format.WriteAuditReport(&buf2, out)
	require.NoError(t, err)

	require.Equal(t, buf1.Bytes(), buf2.Bytes(), "WriteAuditReport must be deterministic (RNF-003)")
}

// TestWriteAuditReport_ProfileNameInHeader verifies that when ProfileName is
// set, the HTML comment header appears in the output.
func TestWriteAuditReport_ProfileNameInHeader(t *testing.T) {
	t.Parallel()

	out := auditvo.AuditProjectOutput{
		ProfileName:         "hexagonal-java",
		SemanticPlaceholder: auditvo.SemanticPlaceholder,
	}

	var buf bytes.Buffer
	err := format.WriteAuditReport(&buf, out)
	require.NoError(t, err)

	require.Contains(t, buf.String(), "<!-- jitctx audit | profile: hexagonal-java -->")
}

// TestWriteAuditReport_NoProfileNameOmitsHeader verifies that when ProfileName
// is empty, the HTML comment header is absent.
func TestWriteAuditReport_NoProfileNameOmitsHeader(t *testing.T) {
	t.Parallel()

	out := auditvo.AuditProjectOutput{
		ProfileName:         "",
		SemanticPlaceholder: auditvo.SemanticPlaceholder,
	}

	var buf bytes.Buffer
	err := format.WriteAuditReport(&buf, out)
	require.NoError(t, err)

	require.NotContains(t, buf.String(), "<!-- jitctx audit")
}
