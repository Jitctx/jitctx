package fsmanifest_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
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

// TestStore_Load_RejectsV1Manifest verifies that loading a v1-shaped manifest
// (with singular "type:" field on contracts and no schema_version) returns
// domerr.ErrManifestSchemaOutdated and a helpful error message.
func TestStore_Load_RejectsV1Manifest(t *testing.T) {
	t.Parallel()

	// The fixture is committed under testdata/ep04us003/v1Manifest.
	manifestPath := filepath.Join("..", "..", "..", "testdata", "ep04us003", "v1Manifest", "project-state.yaml")

	s := fsmanifest.New(manifestPath)
	_, err := s.Load(t.Context())

	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrManifestSchemaOutdated),
		"expected ErrManifestSchemaOutdated, got: %v", err)
	require.Contains(t, err.Error(), "run jitctx scan to upgrade the manifest")
}

// TestStore_Load_AcceptsV2Manifest verifies that a v2-shaped manifest
// (with schema_version: 2 and "types:" sequence on contracts) loads without
// error and returns the expected contract data.
func TestStore_Load_AcceptsV2Manifest(t *testing.T) {
	t.Parallel()

	manifestPath := filepath.Join("..", "..", "..", "testdata", "ep04us003", "v2Manifest", "project-state.yaml")

	s := fsmanifest.New(manifestPath)
	state, err := s.Load(t.Context())

	require.NoError(t, err)
	require.NotNil(t, state)
	require.Len(t, state.Modules, 1)
	require.Len(t, state.Modules[0].Contracts, 1)
	require.Equal(t, []string{"service"}, state.Modules[0].Contracts[0].Types)
}

// TestStore_RoundTripPreservesEmptyTypes saves a manifest whose contract has
// Types: nil, reloads it, and asserts the reloaded contract has a non-nil
// empty Types slice. It also inspects the raw YAML to confirm "types: []"
// is present and the singular "type:" form is absent.
func TestStore_RoundTripPreservesEmptyTypes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "project-state.yaml")
	store := fsmanifest.New(path)

	original := &model.ProjectState{
		Stack: model.Stack{
			Languages:  []string{},
			Frameworks: []string{},
		},
		Modules: []model.Module{
			{
				ID:   "alpha",
				Path: "internal/alpha",
				Tags: []string{},
				Contracts: []model.Contract{
					{
						Name:    "AlphaPort",
						Types:   nil, // intentionally nil
						Path:    "internal/alpha/port.go",
						Methods: []model.Method{},
					},
				},
				Dependencies: []string{},
			},
		},
	}

	require.NoError(t, store.Save(t.Context(), original))

	// Inspect raw YAML bytes.
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	rawStr := string(raw)

	require.Contains(t, rawStr, "types: []",
		"on-disk YAML must contain 'types: []' for nil-typed contracts")
	require.NotContains(t, rawStr, "\n      type:",
		"on-disk YAML must not contain singular 'type:' field")

	// Reload and verify non-nil empty slice.
	loaded, err := store.Load(t.Context())
	require.NoError(t, err)
	require.Len(t, loaded.Modules, 1)
	require.Len(t, loaded.Modules[0].Contracts, 1)

	got := loaded.Modules[0].Contracts[0]
	require.NotNil(t, got.Types, "reloaded contract Types must not be nil")
	require.Empty(t, got.Types, "reloaded contract Types must be empty slice")
}
