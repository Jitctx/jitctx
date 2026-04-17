package format_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/cli/format"
	scanvo "github.com/jitctx/jitctx/internal/domain/vo/scan"
)

func TestWriteScanReport_Markdown(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	out := scanvo.ScanProjectOutput{
		ManifestPath: "/project/project-state.yaml",
		ModuleCount:  3,
		ContextCount: 2,
	}
	err := format.WriteScanReport(&buf, "markdown", out)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "scanned: 3 modules, 2 contexts")
	require.Contains(t, buf.String(), "/project/project-state.yaml")
}

func TestWriteScanReport_JSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	out := scanvo.ScanProjectOutput{
		ManifestPath: "/project/project-state.yaml",
		ModuleCount:  2,
		ContextCount: 1,
		SkippedFiles: []string{"Broken.java"},
	}
	err := format.WriteScanReport(&buf, "json", out)
	require.NoError(t, err)
	body := buf.String()
	require.True(t, strings.Contains(body, `"manifest_path"`))
	require.True(t, strings.Contains(body, `"module_count": 2`))
	require.True(t, strings.Contains(body, `"context_count": 1`))
	require.True(t, strings.Contains(body, `"Broken.java"`))
}
