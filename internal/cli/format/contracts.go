package format

import (
	"encoding/json"
	"fmt"
	"io"

	contractsvo "github.com/jitctx/jitctx/internal/domain/vo/contracts"
)

type contractsSliceJSON struct {
	Source  string                 `json:"source"`
	Target  contractFragmentJSON   `json:"target"`
	Related []contractFragmentJSON `json:"related"`
}

type contractFragmentJSON struct {
	Name        string   `json:"name"`
	Type        string   `json:"type,omitempty"`
	Types       []string `json:"types,omitempty"` // EP04US-003: manifest-sourced multi-types
	Path        string   `json:"path"`
	Role        string   `json:"role"`
	Methods     []string `json:"methods,omitempty"`
	Fields      []string `json:"fields,omitempty"`
	Uses        []string `json:"uses,omitempty"`
	Implements  string   `json:"implements,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
	Endpoints   []string `json:"endpoints,omitempty"`
	Annotations []string `json:"annotations,omitempty"`
}

// WriteContractsText renders a slice as markdown sections to w.
// Layout (see §8 Q5 Option A — markdown is the user-facing default):
//
//	# Target: <Name> (<Type>)
//	Source: <spec|manifest>
//	Path: <path>
//	Role: <role>
//
//	Methods:
//	- <sig>
//
//	Fields:
//	- <field>
//
//	Endpoints:
//	- <endpoint>
//
//	## Dependencies (<N>)
//
//	### <DepName> (<Type>)
//	Path: <path>
//	Role: <role>
//	Methods:
//	- <sig>
//
// Empty sections (e.g. no Methods) are omitted entirely. When Related is
// empty, the "## Dependencies (0)" header is still emitted with the count
// so JSON-style tooling can grep deterministically.
func WriteContractsText(w io.Writer, out contractsvo.ExtractContractsOutput) error {
	t := out.Target
	if _, err := fmt.Fprintf(w, "# Target: %s (%s)\n", t.Name, fragmentTypeLabel(t)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Source: %s\n", out.Source); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Path: %s\n", t.Path); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Role: %s\n", t.Role); err != nil {
		return err
	}
	if err := writeFragmentSections(w, t); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "\n## Dependencies (%d)\n", len(out.Related)); err != nil {
		return err
	}
	for _, dep := range out.Related {
		if _, err := fmt.Fprintf(w, "\n### %s (%s)\n", dep.Name, fragmentTypeLabel(dep)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Path: %s\n", dep.Path); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Role: %s\n", dep.Role); err != nil {
			return err
		}
		if err := writeFragmentSections(w, dep); err != nil {
			return err
		}
	}
	return nil
}

// writeFragmentSections writes the Methods, Fields, and Endpoints sections for
// a fragment. Empty sections are omitted entirely.
func writeFragmentSections(w io.Writer, f contractsvo.ContractFragment) error {
	if len(f.Methods) > 0 {
		if _, err := fmt.Fprintf(w, "\nMethods:\n"); err != nil {
			return err
		}
		for _, m := range f.Methods {
			if _, err := fmt.Fprintf(w, "- %s\n", m); err != nil {
				return err
			}
		}
	}
	if len(f.Fields) > 0 {
		if _, err := fmt.Fprintf(w, "\nFields:\n"); err != nil {
			return err
		}
		for _, field := range f.Fields {
			if _, err := fmt.Fprintf(w, "- %s\n", field); err != nil {
				return err
			}
		}
	}
	if len(f.Endpoints) > 0 {
		if _, err := fmt.Fprintf(w, "\nEndpoints:\n"); err != nil {
			return err
		}
		for _, ep := range f.Endpoints {
			if _, err := fmt.Fprintf(w, "- %s\n", ep); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteContractsJSON renders the slice as indented JSON to w.
// Matches internal/cli/format/planLayers.go's WriteLayersJSON style.
//
// JSON shape (DTO defined in this file, NOT in the domain VO):
//
//	{
//	  "source": "spec",
//	  "target": { "name": "...", "type": "...", "path": "...", "role": "...",
//	              "methods": [...], "fields": [...], "uses": [...],
//	              "implements": "...", "depends_on": [...], "endpoints": [...],
//	              "annotations": [...] },
//	  "related": [ { same shape as target } ]
//	}
func WriteContractsJSON(w io.Writer, out contractsvo.ExtractContractsOutput) error {
	// Always emit `related` as a JSON array, never null — explicit empty
	// initialisation handles both the nil and empty cases.
	related := []contractFragmentJSON{}
	for _, r := range out.Related {
		related = append(related, toFragmentJSON(r))
	}
	dto := contractsSliceJSON{
		Source:  out.Source,
		Target:  toFragmentJSON(out.Target),
		Related: related,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(dto)
}

func toFragmentJSON(f contractsvo.ContractFragment) contractFragmentJSON {
	return contractFragmentJSON{
		Name:        f.Name,
		Type:        f.Type,
		Types:       f.Types,
		Path:        f.Path,
		Role:        f.Role,
		Methods:     f.Methods,
		Fields:      f.Fields,
		Uses:        f.Uses,
		Implements:  f.Implements,
		DependsOn:   f.DependsOn,
		Endpoints:   f.Endpoints,
		Annotations: f.Annotations,
	}
}

// fragmentTypeLabel returns the human-readable type label for a ContractFragment.
// It returns f.Type directly when non-empty (spec-sourced projection, singular
// per RF-015), and falls back to formatTypeLabel(f.Types) for manifest-sourced
// projections (EP04US-003 dual-shape).
func fragmentTypeLabel(f contractsvo.ContractFragment) string {
	if f.Type != "" {
		return f.Type
	}
	return formatTypeLabel(f.Types)
}
