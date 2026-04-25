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
//	wrote N files (P production, T test):
//	  - <abs path 1>
//	  - <abs path 2>
//
// The parenthetical "(P production, T test)" is omitted when both
// ProductionCount and TestCount are zero.
func WriteScaffoldText(w io.Writer, out scaffoldvo.ScaffoldOutput) error {
	if _, err := fmt.Fprintf(w, "scaffolded: %s (module: %s, package: %s)\n",
		out.Feature, out.Module, out.Package); err != nil {
		return err
	}
	var header string
	if out.ProductionCount == 0 && out.TestCount == 0 {
		header = fmt.Sprintf("wrote %d files:\n", len(out.WrittenPaths))
	} else {
		header = fmt.Sprintf("wrote %d files (%d production, %d test):\n",
			len(out.WrittenPaths), out.ProductionCount, out.TestCount)
	}
	if _, err := fmt.Fprint(w, header); err != nil {
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
	Feature         string   `json:"feature"`
	Module          string   `json:"module"`
	Package         string   `json:"package"`
	Written         []string `json:"written"`
	ProductionCount int      `json:"production_count"`
	TestCount       int      `json:"test_count"`
}

// WriteScaffoldJSON renders a ScaffoldOutput as indented JSON to w.
// The "written" field is always an array (never null).
func WriteScaffoldJSON(w io.Writer, out scaffoldvo.ScaffoldOutput) error {
	written := out.WrittenPaths
	if written == nil {
		written = []string{}
	}
	dto := scaffoldOutputJSON{
		Feature:         out.Feature,
		Module:          out.Module,
		Package:         out.Package,
		Written:         written,
		ProductionCount: out.ProductionCount,
		TestCount:       out.TestCount,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(dto)
}
