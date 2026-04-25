package format

import (
	"encoding/json"
	"fmt"
	"io"

	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
)

type planLayersJSON struct {
	Feature string                `json:"feature"`
	Module  string                `json:"module"`
	Layers  [][]planLayerJSONItem `json:"layers"`
}

type planLayerJSONItem struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	TargetPath string `json:"target_path"`
}

// WriteLayersText renders a LayersOutput as human-readable text to w.
// Output format:
//
//	plan: <feature> (module: <module>)
//	Layer 0
//	  - <Name> (<type>) → <target_path>
func WriteLayersText(w io.Writer, out planvo.LayersOutput) error {
	if _, err := fmt.Fprintf(w, "plan: %s (module: %s)\n", out.Feature, out.Module); err != nil {
		return err
	}
	for _, layer := range out.Layers {
		if _, err := fmt.Fprintf(w, "Layer %d\n", layer.Index); err != nil {
			return err
		}
		for _, t := range layer.Targets {
			if _, err := fmt.Fprintf(w, "  - %s (%s) → %s\n", t.Name, t.Type, t.TargetPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteLayersJSON renders a LayersOutput as indented JSON to w.
// Externals are not serialised — they surface as stderr warnings from the use case.
func WriteLayersJSON(w io.Writer, out planvo.LayersOutput) error {
	dto := planLayersJSON{
		Feature: out.Feature,
		Module:  out.Module,
		Layers:  make([][]planLayerJSONItem, len(out.Layers)),
	}
	for i, layer := range out.Layers {
		items := make([]planLayerJSONItem, len(layer.Targets))
		for j, t := range layer.Targets {
			items[j] = planLayerJSONItem{
				Name:       t.Name,
				Type:       t.Type,
				TargetPath: t.TargetPath,
			}
		}
		dto.Layers[i] = items
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(dto)
}
