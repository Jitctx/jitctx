package format

import (
	"fmt"
	"io"

	refactorvo "github.com/jitctx/jitctx/internal/domain/vo/refactor"
)

// WriteRefactorMarkersReport renders a ScanRefactorsOutput as
// deterministic markdown. Markdown is the only supported output format
// per EP03RF-007. Same input → byte-identical output (RNF-003).
//
// Structure:
//
//	<!-- jitctx scan --refactors -->
//
//	## Refactor Markers
//
//	## Module: user-management
//	- rename — `src/main/java/.../A.java:15` — rename foo to handleFoo
//	- extract-method — `src/main/java/.../V.java:42` — validateEmail
//
//	## Module: billing
//	- other — `.../X.java:7` — some text
//	- unparseable — `.../Y.java:12` — // TODO(jitctx): missing format here
//
// Empty Markers slice → emit the verbatim line (and only that line):
//
//	No refactor markers found.
func WriteRefactorMarkersReport(w io.Writer, out refactorvo.ScanRefactorsOutput) error {
	// Empty case: emit verbatim line and stop.
	if len(out.Markers) == 0 {
		if _, err := fmt.Fprintln(w, "No refactor markers found."); err != nil {
			return err
		}
		return nil
	}

	// Header comment.
	if _, err := fmt.Fprintf(w, "<!-- jitctx scan --refactors -->\n\n"); err != nil {
		return err
	}

	// Top-level heading.
	if _, err := fmt.Fprintf(w, "## Refactor Markers\n\n"); err != nil {
		return err
	}

	// Module grouping: markers are already sorted by (ModuleID, FilePath, Line, ...)
	// with "<unmoduled>" last. A single linear pass emits one heading per module group.
	lastModuleID := ""
	for _, m := range out.Markers {
		if m.ModuleID != lastModuleID {
			// Emit blank line between module groups (not before the very first).
			if lastModuleID != "" {
				if _, err := fmt.Fprintln(w); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(w, "## Module: %s\n\n", m.ModuleID); err != nil {
				return err
			}
			lastModuleID = m.ModuleID
		}

		// Choose trailing text: OriginalText for unparseable, Description otherwise.
		trailing := m.Description
		if m.Type == refactorvo.MarkerTypeUnparseable {
			trailing = m.OriginalText
		}

		// Marker bullet: - <type> — `<filepath>:<line>` — <description-or-original>
		if _, err := fmt.Fprintf(w, "- %s — `%s:%d` — %s\n",
			string(m.Type), m.FilePath, m.Line, trailing); err != nil {
			return err
		}
	}

	// Trailing newline after the last module group.
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	return nil
}
