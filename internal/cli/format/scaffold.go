package format

import (
	"encoding/json"
	"fmt"
	"io"

	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

// WriteScaffoldText renders a ScaffoldOutput as human-readable text to w.
//
// Format:
//
//	scaffolded: <feature> (module: <module>, package: <pkg>)
//	wrote N files:
//	  - <abs path 1>
//	  - <abs path 2>
func WriteScaffoldText(w io.Writer, out scaffoldvo.ScaffoldOutput) error {
	if _, err := fmt.Fprintf(w, "scaffolded: %s (module: %s, package: %s)\n",
		out.Feature, out.Module, out.Package); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "wrote %d files:\n", len(out.WrittenPaths)); err != nil {
		return err
	}
	for _, p := range out.WrittenPaths {
		if _, err := fmt.Fprintf(w, "  - %s\n", p); err != nil {
			return err
		}
	}
	return nil
}

// scaffoldOutputJSON is the private JSON DTO for WriteScaffoldJSON.
type scaffoldOutputJSON struct {
	Feature string   `json:"feature"`
	Module  string   `json:"module"`
	Package string   `json:"package"`
	Written []string `json:"written"`
}

// WriteScaffoldJSON renders a ScaffoldOutput as indented JSON to w.
// The "written" field is always an array (never null).
func WriteScaffoldJSON(w io.Writer, out scaffoldvo.ScaffoldOutput) error {
	written := out.WrittenPaths
	if written == nil {
		written = []string{}
	}
	dto := scaffoldOutputJSON{
		Feature: out.Feature,
		Module:  out.Module,
		Package: out.Package,
		Written: written,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(dto)
}
