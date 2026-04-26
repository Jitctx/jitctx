package fsconfig_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/infrastructure/fsconfig"
)

// writeConfig writes content to <dir>/.jitctx/config.yaml, creating the
// intermediate directory as needed.
func writeConfig(t *testing.T, dir string, content []byte) {
	t.Helper()
	jitctxDir := filepath.Join(dir, ".jitctx")
	require.NoError(t, os.MkdirAll(jitctxDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(jitctxDir, "config.yaml"), content, 0o644))
}

// newLoader builds a Loader with a nil logger (falls back to slog.Default()).
func newLoader() *fsconfig.Loader {
	return fsconfig.New(nil)
}

// ----------------------------------------------------------------------------
// Case 1 — Missing file
// ----------------------------------------------------------------------------

func TestLoader_MissingFile_ReturnsZeroConfigNoError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// No .jitctx/config.yaml written.

	cfg, err := newLoader().LoadJitctxConfig(context.Background(), dir)
	require.NoError(t, err)
	require.Nil(t, cfg.Audit.DisabledRules)
}

// ----------------------------------------------------------------------------
// Case 2 — Empty file (zero bytes)
// ----------------------------------------------------------------------------

func TestLoader_EmptyFile_ReturnsZeroConfigNoError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfig(t, dir, []byte{})

	cfg, err := newLoader().LoadJitctxConfig(context.Background(), dir)
	require.NoError(t, err)
	require.Nil(t, cfg.Audit.DisabledRules)
}

// ----------------------------------------------------------------------------
// Case 3 — Comment-only file
// ----------------------------------------------------------------------------

func TestLoader_CommentOnlyFile_ReturnsZeroConfigNoError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfig(t, dir, []byte("# nothing here\n"))

	cfg, err := newLoader().LoadJitctxConfig(context.Background(), dir)
	require.NoError(t, err)
	require.Nil(t, cfg.Audit.DisabledRules)
}

// ----------------------------------------------------------------------------
// Case 4 — audit.disabled_rules: null
// ----------------------------------------------------------------------------

func TestLoader_DisabledRulesNull_ReturnsNilSliceNoError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfig(t, dir, []byte("audit:\n  disabled_rules: null\n"))

	cfg, err := newLoader().LoadJitctxConfig(context.Background(), dir)
	require.NoError(t, err)
	require.Nil(t, cfg.Audit.DisabledRules)
}

// ----------------------------------------------------------------------------
// Case 5 — audit.disabled_rules: []
// ----------------------------------------------------------------------------

func TestLoader_DisabledRulesEmptyList_ReturnsNilSliceNoError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfig(t, dir, []byte("audit:\n  disabled_rules: []\n"))

	cfg, err := newLoader().LoadJitctxConfig(context.Background(), dir)
	require.NoError(t, err)
	// mapper collapses empty list to nil
	require.Empty(t, cfg.Audit.DisabledRules)
}

// ----------------------------------------------------------------------------
// Case 6 — Valid list with two rule IDs
// ----------------------------------------------------------------------------

func TestLoader_ValidDisabledRulesList_ReturnsTwoIDs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfig(t, dir, []byte("audit:\n  disabled_rules: [domain-leak, port-naming]\n"))

	cfg, err := newLoader().LoadJitctxConfig(context.Background(), dir)
	require.NoError(t, err)
	require.Equal(t, []string{"domain-leak", "port-naming"}, cfg.Audit.DisabledRules)
}

// ----------------------------------------------------------------------------
// Case 7 — Unknown top-level key → error containing "read <path>:"
// ----------------------------------------------------------------------------

func TestLoader_UnknownTopLevelKey_ReturnsWrappedError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfig(t, dir, []byte("audit_typo:\n  disabled_rules: [r1]\n"))

	_, err := newLoader().LoadJitctxConfig(context.Background(), dir)
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), "read "), "error should start with 'read ': %v", err)
}

// ----------------------------------------------------------------------------
// Case 8 — Malformed YAML → error containing "read <path>:"
// ----------------------------------------------------------------------------

func TestLoader_MalformedYAML_ReturnsWrappedError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfig(t, dir, []byte("audit: [\n  - unclosed\n"))

	_, err := newLoader().LoadJitctxConfig(context.Background(), dir)
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), "read "), "error should start with 'read ': %v", err)
}

// ----------------------------------------------------------------------------
// Case 9 — Cancelled context on entry → ctx.Err() returned immediately
// ----------------------------------------------------------------------------

func TestLoader_CancelledContext_ReturnsCtxErrNoIO(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Write a valid file to make sure any I/O would have succeeded; the
	// cancelled context must short-circuit before reaching os.ReadFile.
	writeConfig(t, dir, []byte("audit:\n  disabled_rules: [r1]\n"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := newLoader().LoadJitctxConfig(ctx, dir)
	require.ErrorIs(t, err, context.Canceled)
}

// ----------------------------------------------------------------------------
// Case 10 — plans_dir present alongside audit.disabled_rules (EP02 compat)
// ----------------------------------------------------------------------------

func TestLoader_PlansDir_LoadsCleanlyAlongsideAudit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := []byte("plans_dir: .claude/plans\naudit:\n  disabled_rules: [domain-leak]\n")
	writeConfig(t, dir, content)

	cfg, err := newLoader().LoadJitctxConfig(context.Background(), dir)
	require.NoError(t, err)
	require.Equal(t, []string{"domain-leak"}, cfg.Audit.DisabledRules)
}
