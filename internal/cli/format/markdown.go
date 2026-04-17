package format

import (
	"encoding/json"
	"fmt"
	"io"

	contractsvo "github.com/jitctx/jitctx/internal/domain/vo/contracts"
	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
	queryvo "github.com/jitctx/jitctx/internal/domain/vo/query"
	scanvo "github.com/jitctx/jitctx/internal/domain/vo/scan"
)

// scanReportJSON is the JSON representation for the scan report.
type scanReportJSON struct {
	ManifestPath string   `json:"manifest_path"`
	ModuleCount  int      `json:"module_count"`
	ContextCount int      `json:"context_count"`
	SkippedFiles []string `json:"skipped_files"`
}

func WriteScanReport(w io.Writer, outputFmt string, out scanvo.ScanProjectOutput) error {
	if outputFmt == "json" {
		skipped := out.SkippedFiles
		if skipped == nil {
			skipped = []string{}
		}
		return writeJSON(w, scanReportJSON{
			ManifestPath: out.ManifestPath,
			ModuleCount:  out.ModuleCount,
			ContextCount: out.ContextCount,
			SkippedFiles: skipped,
		})
	}
	_, err := fmt.Fprintf(w, "scanned: %d modules, %d contexts → %s\n",
		out.ModuleCount, out.ContextCount, out.ManifestPath)
	return err
}

func WriteQueryResult(w io.Writer, format string, out queryvo.QueryContextOutput) error {
	switch format {
	case "json":
		return writeJSON(w, out)
	case "raw":
		for _, c := range out.Loaded {
			if _, err := fmt.Fprintln(w, c.Body); err != nil {
				return err
			}
		}
		return nil
	}
	return writeQueryMarkdown(w, out)
}

func writeQueryMarkdown(w io.Writer, out queryvo.QueryContextOutput) error {
	if _, err := fmt.Fprintf(w,
		"<!-- jitctx: %d contexts loaded | ~%d tokens | trimmed: %d -->\n\n",
		len(out.Loaded), out.TotalTokens, len(out.Trimmed),
	); err != nil {
		return err
	}
	if out.Module.ID != "" && len(out.Module.Contracts) > 0 {
		if _, err := fmt.Fprintf(w, "## Contracts — %s\n\n", out.Module.ID); err != nil {
			return err
		}
		for _, c := range out.Module.Contracts {
			if _, err := fmt.Fprintf(w, "- **%s** (%s)\n", c.Name, c.Type); err != nil {
				return err
			}
			for _, sig := range c.Methods {
				if _, err := fmt.Fprintf(w, "    - %s\n", sig); err != nil {
					return err
				}
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	for _, c := range out.Loaded {
		if _, err := fmt.Fprintf(w, "---\n<!-- source: %s | tags: %v -->\n\n%s\n\n", c.Path, c.Tags, c.Body); err != nil {
			return err
		}
	}
	return nil
}

func WritePlan(w io.Writer, format string, out planvo.PlanModuleOutput) error {
	if format == "json" {
		return writeJSON(w, out)
	}
	if _, err := fmt.Fprintf(w, "plan for %s:\n", out.Module); err != nil {
		return err
	}
	for _, layer := range out.Layers {
		kind := "sequential"
		if layer.Parallel {
			kind = "parallel"
		}
		if _, err := fmt.Fprintf(w, "  layer %d [%s]:\n", layer.Index, kind); err != nil {
			return err
		}
		for _, t := range layer.Targets {
			if _, err := fmt.Fprintf(w, "    - %s (%s)\n", t.Path, t.Kind); err != nil {
				return err
			}
		}
	}
	return nil
}

func WriteContracts(w io.Writer, format string, out contractsvo.ExtractContractsOutput) error {
	if format == "json" {
		return writeJSON(w, out)
	}
	for _, c := range out.Contracts {
		if _, err := fmt.Fprintf(w, "## %s (%s)\nPath: %s\n\n%s\n\n", c.Name, c.Type, c.Path, c.Body); err != nil {
			return err
		}
	}
	return nil
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
