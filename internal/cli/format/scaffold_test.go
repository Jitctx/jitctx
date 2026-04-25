package format_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/cli/format"
	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

func fixtureScaffoldOutput() scaffoldvo.ScaffoldOutput {
	return scaffoldvo.ScaffoldOutput{
		Feature:      "create-user",
		Module:       "user-management",
		Package:      "com.app.user",
		WrittenPaths: []string{"/abs/A.java", "/abs/B.java"},
	}
}

func TestWriteScaffoldText_HappyPath(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := format.WriteScaffoldText(&buf, fixtureScaffoldOutput())
	require.NoError(t, err)

	got := buf.String()
	require.Contains(t, got, "scaffolded: create-user (module: user-management, package: com.app.user)")
	require.Contains(t, got, "wrote 2 files:")
	require.Contains(t, got, "  - /abs/A.java")
	require.Contains(t, got, "  - /abs/B.java")
}

func TestWriteScaffoldText_ZeroFiles(t *testing.T) {
	t.Parallel()

	out := scaffoldvo.ScaffoldOutput{
		Feature:      "create-user",
		Module:       "user-management",
		Package:      "com.app.user",
		WrittenPaths: []string{},
	}

	var buf bytes.Buffer
	err := format.WriteScaffoldText(&buf, out)
	require.NoError(t, err)

	got := buf.String()
	require.Contains(t, got, "wrote 0 files:")
	require.NotContains(t, got, "  - ")
}

func TestWriteScaffoldJSON_Shape(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := format.WriteScaffoldJSON(&buf, fixtureScaffoldOutput())
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &m))

	require.Equal(t, "create-user", m["feature"])
	require.Equal(t, "user-management", m["module"])
	require.Equal(t, "com.app.user", m["package"])

	written, ok := m["written"].([]any)
	require.True(t, ok, "expected 'written' to be an array")
	require.Len(t, written, 2)
}

func TestWriteScaffoldJSON_NeverNullArray(t *testing.T) {
	t.Parallel()

	out := scaffoldvo.ScaffoldOutput{
		Feature:      "create-user",
		Module:       "user-management",
		Package:      "com.app.user",
		WrittenPaths: nil,
	}

	var buf bytes.Buffer
	err := format.WriteScaffoldJSON(&buf, out)
	require.NoError(t, err)

	require.Contains(t, buf.String(), `"written": []`)
}

func TestWriteScaffoldJSON_Deterministic(t *testing.T) {
	t.Parallel()

	fixture := fixtureScaffoldOutput()

	var buf1 bytes.Buffer
	require.NoError(t, format.WriteScaffoldJSON(&buf1, fixture))

	var buf2 bytes.Buffer
	require.NoError(t, format.WriteScaffoldJSON(&buf2, fixture))

	require.Equal(t, buf1.Bytes(), buf2.Bytes())
}
