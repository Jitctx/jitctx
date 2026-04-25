package format_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/cli/format"
	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
)

func TestWriteLayersText_emptyLayers(t *testing.T) {
	t.Parallel()

	out := planvo.LayersOutput{
		Feature:   "x",
		Module:    "m",
		Layers:    nil,
		Externals: nil,
	}

	var buf bytes.Buffer
	err := format.WriteLayersText(&buf, out)
	require.NoError(t, err)

	got := buf.String()
	require.Contains(t, got, "plan: x (module: m)")

	// No "Layer N" lines should appear.
	require.False(t, strings.Contains(got, "Layer 0"), "expected no Layer 0 line for empty layers")
	require.False(t, strings.Contains(got, "Layer 1"), "expected no Layer 1 line for empty layers")
}

func TestWriteLayersText_twoLayers(t *testing.T) {
	t.Parallel()

	out := planvo.LayersOutput{
		Feature: "my-feature",
		Module:  "my-module",
		Layers: []planvo.ExecutionLayer{
			{
				Index: 0,
				Targets: []planvo.PlanTarget{
					{Name: "A", Type: "input-port", TargetPath: "port/in/A.java"},
					{Name: "B", Type: "output-port", TargetPath: "port/out/B.java"},
				},
			},
			{
				Index: 1,
				Targets: []planvo.PlanTarget{
					{Name: "C", Type: "service", TargetPath: "application/C.java"},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := format.WriteLayersText(&buf, out)
	require.NoError(t, err)

	got := buf.String()

	require.Contains(t, got, "Layer 0")
	require.Contains(t, got, "Layer 1")

	// Verify entry format with unicode arrow for layer 0.
	require.Contains(t, got, "  - A (input-port) → port/in/A.java")
	require.Contains(t, got, "  - B (output-port) → port/out/B.java")

	// Verify entry format with unicode arrow for layer 1.
	require.Contains(t, got, "  - C (service) → application/C.java")

	// Verify order: A appears before B in the output.
	idxA := strings.Index(got, "  - A (input-port)")
	idxB := strings.Index(got, "  - B (output-port)")
	require.Less(t, idxA, idxB, "A should appear before B within Layer 0")

	// Verify layer ordering: Layer 0 before Layer 1.
	idxLayer0 := strings.Index(got, "Layer 0")
	idxLayer1 := strings.Index(got, "Layer 1")
	require.Less(t, idxLayer0, idxLayer1, "Layer 0 should appear before Layer 1")
}

func TestWriteLayersJSON_shape(t *testing.T) {
	t.Parallel()

	out := planvo.LayersOutput{
		Feature: "my-feature",
		Module:  "my-module",
		Layers: []planvo.ExecutionLayer{
			{
				Index: 0,
				Targets: []planvo.PlanTarget{
					{Name: "A", Type: "input-port", TargetPath: "port/in/A.java"},
					{Name: "B", Type: "output-port", TargetPath: "port/out/B.java"},
				},
			},
			{
				Index: 1,
				Targets: []planvo.PlanTarget{
					{Name: "C", Type: "service", TargetPath: "application/C.java"},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := format.WriteLayersJSON(&buf, out)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))

	// Assert top-level keys.
	require.Contains(t, decoded, "feature")
	require.Contains(t, decoded, "module")
	require.Contains(t, decoded, "layers")

	// Assert layers is a []any of length 2.
	layers, ok := decoded["layers"].([]any)
	require.True(t, ok, "layers should be a JSON array")
	require.Len(t, layers, 2)

	// Pick the first contract object inside layers[0].
	layer0, ok := layers[0].([]any)
	require.True(t, ok, "layers[0] should be a JSON array of contract objects")
	require.NotEmpty(t, layer0)

	contract0, ok := layer0[0].(map[string]any)
	require.True(t, ok, "first contract in layers[0] should be a JSON object")

	// Assert exact snake_case keys.
	require.Contains(t, contract0, "name")
	require.Contains(t, contract0, "type")
	require.Contains(t, contract0, "target_path")
}

func TestWriteLayersJSON_deterministic(t *testing.T) {
	t.Parallel()

	out := planvo.LayersOutput{
		Feature: "my-feature",
		Module:  "my-module",
		Layers: []planvo.ExecutionLayer{
			{
				Index: 0,
				Targets: []planvo.PlanTarget{
					{Name: "A", Type: "input-port", TargetPath: "port/in/A.java"},
					{Name: "B", Type: "output-port", TargetPath: "port/out/B.java"},
				},
			},
			{
				Index: 1,
				Targets: []planvo.PlanTarget{
					{Name: "C", Type: "service", TargetPath: "application/C.java"},
				},
			},
		},
	}

	var buf1, buf2 bytes.Buffer

	require.NoError(t, format.WriteLayersJSON(&buf1, out))
	require.NoError(t, format.WriteLayersJSON(&buf2, out))

	require.Equal(t, buf1.Bytes(), buf2.Bytes(), "WriteLayersJSON must be byte-identical across calls (RNF-002)")
}
