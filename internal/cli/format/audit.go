package format

import (
	"fmt"
	"io"

	auditvo "github.com/jitctx/jitctx/internal/domain/vo/audit"
)

// WriteAuditReport renders the AuditProjectOutput as deterministic markdown.
// Only one output format is supported — markdown — per EP03RF-007.
// The renderer is pure: same input => byte-identical output (RNF-003).
//
// Structure (both headings always present per RNF-005):
//
//	## Sintatic Violations
//	<per-module blocks or clean message>
//
//	## Semantic Analysis
//	<SemanticPlaceholder literal from the use case>
func WriteAuditReport(w io.Writer, out auditvo.AuditProjectOutput) error {
	// 1. Optional header comment with profile name.
	if out.ProfileName != "" {
		if _, err := fmt.Fprintf(w, "<!-- jitctx audit | profile: %s -->\n\n", out.ProfileName); err != nil {
			return err
		}
	}

	// 2. Sintatic Violations heading — ALWAYS present (RNF-005).
	if _, err := fmt.Fprintln(w, "## Sintatic Violations"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	if len(out.Sintatic) == 0 {
		// Clean project: emit the exact verbatim line required by the .feature scenario.
		if _, err := fmt.Fprintln(w, "No sintatic violations detected"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	} else {
		// Group violations by ModuleID. The use case guarantees the slice is
		// already sorted by (ModuleID, FilePath, Line, RuleID), so a single
		// linear pass is enough to emit one "## Module: <id>" heading per group.
		lastModuleID := ""
		for _, v := range out.Sintatic {
			if v.ModuleID != lastModuleID {
				// New module group: emit sub-heading.
				if _, err := fmt.Fprintf(w, "## Module: %s\n\n", v.ModuleID); err != nil {
					return err
				}
				lastModuleID = v.ModuleID
			}

			// Severity badge per RF-007.
			badge := auditvo.SeverityBadge(v.Severity)

			// File path with optional line suffix (omit :line when Line == 0).
			location := v.FilePath
			if v.Line != 0 {
				location = fmt.Sprintf("%s:%d", v.FilePath, v.Line)
			}

			// Violation block.
			if _, err := fmt.Fprintf(w, "- %s [%s] %s — %s\n", badge, v.RuleID, location, v.Message); err != nil {
				return err
			}

			// Suggestion is optional.
			if v.Suggestion != "" {
				if _, err := fmt.Fprintf(w, "  suggestion: %s\n", v.Suggestion); err != nil {
					return err
				}
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	// 3. Semantic Analysis heading — ALWAYS present (RNF-005).
	if _, err := fmt.Fprintln(w, "## Semantic Analysis"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	// Emit the canonical literal owned by the use case — never hardcode here.
	if _, err := fmt.Fprintln(w, out.SemanticPlaceholder); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	return nil
}
