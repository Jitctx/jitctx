package format

import (
	"fmt"
	"io"
	"strings"

	diffvo "github.com/jitctx/jitctx/internal/domain/vo/diff"
)

// WriteDiffPlanReport renders a DiffPlanOutput as deterministic markdown.
// Markdown is the only supported output format per EP03RF-007.
//
// Structure:
//
//	<!-- jitctx plan --diff | feature: <name> -->
//
//	## Diff Plan
//
//	### Layer 0
//	- 🔴 ERROR  CREATE: ChangeUserStatusUseCase (input-port)
//	- 🟡 WARNING MODIFY: UserRepository
//	  - added: save
//	  - removed: persist
//
//	### Layer 1
//	- …
//
//	### Extras (manifest contracts not in spec)
//	- 🔵 INFO EXTRA: DeprecatedHelper (service)
//
// Clean diff (Actions empty) emits the verbatim line:
//
//	No diff detected. Current state matches spec.
func WriteDiffPlanReport(w io.Writer, out diffvo.DiffPlanOutput) error {
	// 1. Clean diff: no actions at all → single verbatim line and stop.
	if len(out.Actions) == 0 {
		if _, err := fmt.Fprintln(w, "No diff detected. Current state matches spec."); err != nil {
			return err
		}
		return nil
	}

	// 2. Header comment (omit when Feature is empty).
	if out.Feature != "" {
		if _, err := fmt.Fprintf(w, "<!-- jitctx plan --diff | feature: %s -->\n\n", out.Feature); err != nil {
			return err
		}
	}

	// 3. Top-level heading.
	if _, err := fmt.Fprintln(w, "## Diff Plan"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	// 4. Layered section (CREATE / MODIFY, Layer >= 0).
	// The slice is already sorted by (Layer, ContractName, ActionType).
	// Emit a "### Layer N" sub-heading whenever the Layer value changes.
	lastLayer := -2 // sentinel: no heading emitted yet
	for _, a := range out.Actions {
		if a.Layer < 0 {
			// EXTRA actions are rendered in the Extras section below.
			continue
		}
		if a.Layer != lastLayer {
			if lastLayer != -2 {
				// Blank line between layer blocks.
				if _, err := fmt.Fprintln(w); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(w, "### Layer %d\n", a.Layer); err != nil {
				return err
			}
			lastLayer = a.Layer
		}
		if err := writeDiffActionLine(w, a); err != nil {
			return err
		}
	}

	// 5. Extras section (Layer == -1).
	hasExtras := false
	for _, a := range out.Actions {
		if a.Layer == -1 {
			hasExtras = true
			break
		}
	}
	if hasExtras {
		if lastLayer != -2 {
			// Blank line separating the layered section from Extras.
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w, "### Extras (manifest contracts not in spec)"); err != nil {
			return err
		}
		for _, a := range out.Actions {
			if a.Layer == -1 {
				if err := writeDiffActionLine(w, a); err != nil {
					return err
				}
			}
		}
	}

	// 6. Trailing newline (mirrors audit.go).
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	return nil
}

// writeDiffActionLine emits one bullet for a single DiffAction.
// For MODIFY actions it appends indented sub-bullets for added/removed methods.
func writeDiffActionLine(w io.Writer, a diffvo.DiffAction) error {
	badge := diffSeverityBadge(a.Severity)
	switch a.Type {
	case diffvo.DiffActionCreate:
		if _, err := fmt.Fprintf(w, "- %s %s: %s (%s)\n",
			badge, string(a.Type), a.ContractName, a.ContractType); err != nil {
			return err
		}
	case diffvo.DiffActionModify:
		if _, err := fmt.Fprintf(w, "- %s %s: %s\n",
			badge, string(a.Type), a.ContractName); err != nil {
			return err
		}
		if len(a.AddedMethods) > 0 {
			if _, err := fmt.Fprintf(w, "  - added: %s\n", strings.Join(a.AddedMethods, ", ")); err != nil {
				return err
			}
		}
		if len(a.RemovedMethods) > 0 {
			if _, err := fmt.Fprintf(w, "  - removed: %s\n", strings.Join(a.RemovedMethods, ", ")); err != nil {
				return err
			}
		}
	case diffvo.DiffActionExtra:
		if _, err := fmt.Fprintf(w, "- %s %s: %s (%s)\n",
			badge, string(a.Type), a.ContractName, a.ContractType); err != nil {
			return err
		}
	}
	return nil
}

// diffSeverityBadge returns the emoji+label string for a DiffSeverity value.
// Kept package-local so this file does not import vo/audit (layer independence).
func diffSeverityBadge(s diffvo.DiffSeverity) string {
	switch s {
	case diffvo.DiffSeverityError:
		return "🔴 ERROR"
	case diffvo.DiffSeverityWarning:
		return "🟡 WARNING"
	case diffvo.DiffSeverityInfo:
		return "🔵 INFO"
	default:
		return string(s)
	}
}
