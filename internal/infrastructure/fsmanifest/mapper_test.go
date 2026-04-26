package fsmanifest_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
)

// buildStateWithContract constructs a minimal *model.ProjectState containing a
// single contract with the provided Types slice.
func buildStateWithContract(types []string) *model.ProjectState {
	return &model.ProjectState{
		GeneratedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Stack: model.Stack{
			Languages:  []string{"go"},
			Frameworks: []string{"cobra"},
		},
		Modules: []model.Module{
			{
				ID:   "core",
				Path: "internal/core",
				Tags: []string{},
				Contracts: []model.Contract{
					{
						Name:  "CoreService",
						Types: types,
						Path:  "internal/core/service.go",
						Methods: []model.Method{
							{Signature: "func Execute() error"},
						},
					},
				},
				Dependencies: []string{},
			},
		},
	}
}

// TestMapper_RoundTripMultipleTypes verifies that toDomain(toDTO(state)) == state
// for a contract with multiple types. The round-trip is exercised via Store.Save
// + Store.Load (the only public surface that exposes the mapper pair).
func TestMapper_RoundTripMultipleTypes(t *testing.T) {
	t.Parallel()

	original := buildStateWithContract([]string{"service", "input-port", "entity"})

	dir := t.TempDir()
	path := dir + "/project-state.yaml"
	store := fsmanifest.New(path)

	require.NoError(t, store.Save(t.Context(), original))

	loaded, err := store.Load(t.Context())
	require.NoError(t, err)

	require.Len(t, loaded.Modules, 1)
	require.Len(t, loaded.Modules[0].Contracts, 1)

	got := loaded.Modules[0].Contracts[0]
	require.Equal(t, "CoreService", got.Name)
	require.Equal(t, []string{"service", "input-port", "entity"}, got.Types)
	require.Equal(t, "internal/core/service.go", got.Path)
	require.Len(t, got.Methods, 1)
	require.Equal(t, "func Execute() error", got.Methods[0].Signature)
}

// TestMapper_NilTypesNormalisedToEmpty verifies that toDTO of a contract with
// Types: nil produces a DTO with Types: []string{} (NOT nil), so the on-disk
// YAML carries "types: []" and after a reload the contract's Types is non-nil
// and empty.
func TestMapper_NilTypesNormalisedToEmpty(t *testing.T) {
	t.Parallel()

	original := buildStateWithContract(nil) // nil Types

	dir := t.TempDir()
	path := dir + "/project-state.yaml"
	store := fsmanifest.New(path)

	require.NoError(t, store.Save(t.Context(), original))

	// Assert the on-disk YAML uses "types: []", not "types: null" or singular "type:".
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(raw), "types: []",
		"YAML should contain 'types: []' for nil-typed contracts")
	require.NotContains(t, string(raw), "\n    type:",
		"YAML must not contain singular 'type:' field")

	loaded, err := store.Load(t.Context())
	require.NoError(t, err)

	require.Len(t, loaded.Modules, 1)
	require.Len(t, loaded.Modules[0].Contracts, 1)

	got := loaded.Modules[0].Contracts[0]
	require.NotNil(t, got.Types, "reloaded contract Types must not be nil")
	require.Empty(t, got.Types, "reloaded contract Types must be empty slice")
}

// TestMapper_SchemaVersionStamped verifies that every toDTO output carries
// SchemaVersion == fsmanifest.CurrentManifestSchemaVersion.
func TestMapper_SchemaVersionStamped(t *testing.T) {
	t.Parallel()

	original := buildStateWithContract([]string{"service"})

	dir := t.TempDir()
	path := dir + "/project-state.yaml"
	store := fsmanifest.New(path)

	require.NoError(t, store.Save(t.Context(), original))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var probe struct {
		SchemaVersion int `yaml:"schema_version"`
	}
	require.NoError(t, yaml.NewDecoder(bytes.NewReader(raw)).Decode(&probe))
	require.Equal(t, fsmanifest.CurrentManifestSchemaVersion, probe.SchemaVersion,
		"saved manifest must carry the current schema version")
}
