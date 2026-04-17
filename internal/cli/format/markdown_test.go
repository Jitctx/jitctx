package format_test

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/cli/format"
	queryvo "github.com/jitctx/jitctx/internal/domain/vo/query"
	scanvo "github.com/jitctx/jitctx/internal/domain/vo/scan"
)

func TestWriteQueryResult_MarkdownIncludesContracts(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	out := queryvo.QueryContextOutput{
		Module: queryvo.ModuleSummary{
			ID: "user-management",
			Contracts: []queryvo.ContractSummary{
				{
					Name:    "CreateUserUseCase",
					Type:    "input-port",
					Methods: []string{"UserResponse execute(CreateUserCommand cmd)"},
				},
			},
		},
	}
	err := format.WriteQueryResult(&buf, "markdown", out)
	require.NoError(t, err)
	body := buf.String()
	require.Contains(t, body, "## Contracts — user-management")
	require.Contains(t, body, "CreateUserUseCase")
	require.Contains(t, body, "input-port")
	require.Contains(t, body, "UserResponse execute(CreateUserCommand cmd)")
}

func TestWriteQueryResult_MarkdownHeaderFormat(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	out := queryvo.QueryContextOutput{
		Loaded: []queryvo.LoadedContext{
			{ID: "ctx-1", Body: "body one"},
			{ID: "ctx-2", Body: "body two"},
		},
	}
	err := format.WriteQueryResult(&buf, "markdown", out)
	require.NoError(t, err)
	firstLine := strings.SplitN(buf.String(), "\n", 2)[0]
	re := regexp.MustCompile(`^<!-- jitctx: \d+ contexts loaded`)
	require.Regexp(t, re, firstLine)
}

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
