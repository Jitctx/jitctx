package fsmanifest_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
)

func TestStore_RejectsUnknownFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "project-state.yaml")

	// Write a manifest YAML that contains an unknown top-level field.
	yaml := `generated_at: 2026-01-01T00:00:00Z
rogue_field: boom
stack:
  languages: [java]
  frameworks: [spring-boot]
modules: []
contexts: []
`
	require.NoError(t, os.WriteFile(manifestPath, []byte(yaml), 0o644))

	s := fsmanifest.New(manifestPath)
	_, err := s.Load(context.Background())
	require.Error(t, err, "Load should reject manifests with unknown fields")
}
