package fsmanifest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
)

type Store struct {
	path string
}

func New(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load(ctx context.Context) (*model.ProjectState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, domerr.ErrManifestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	// Phase 1 (strict): decode with KnownFields(true).
	var dto projectStateDTO
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	strictErr := dec.Decode(&dto)

	if strictErr == nil && dto.SchemaVersion == CurrentManifestSchemaVersion {
		// Happy path: v2 manifest decoded cleanly.
		return toDomain(dto), nil
	}

	// Phase 2 (lenient probe): extract the schema_version from the raw YAML
	// without unknown-field rejection, so we can distinguish v1 from a
	// genuinely forward-incompatible version.
	var probe struct {
		SchemaVersion int `yaml:"schema_version"`
	}
	lenDec := yaml.NewDecoder(bytes.NewReader(b))
	// KnownFields(false) is the default; lenient decode ignores unknown fields.
	_ = lenDec.Decode(&probe)

	// schema_version absent (== 0), explicitly 1, or strict decode failed with
	// a v1-shaped document → v1 manifest: ask the user to re-scan.
	if probe.SchemaVersion < CurrentManifestSchemaVersion {
		return nil, fmt.Errorf("load manifest %q: %w", s.path, domerr.ErrManifestSchemaOutdated)
	}

	// schema_version > CurrentManifestSchemaVersion → forward incompatibility.
	if strictErr != nil {
		return nil, fmt.Errorf("load manifest %q: schema version %d is not supported by this binary (max %d): %w",
			s.path, probe.SchemaVersion, CurrentManifestSchemaVersion, strictErr)
	}

	// schema_version == CurrentManifestSchemaVersion but strict decode failed
	// (unknown field from a patch-level extension). Surface the decode error.
	return nil, fmt.Errorf("load manifest %q: %w", s.path, strictErr)
}

func (s *Store) Save(ctx context.Context, state *model.ProjectState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir manifest dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "project-state-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("tempfile: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	enc := yaml.NewEncoder(tmp)
	enc.SetIndent(2)
	if err := enc.Encode(toDTO(state)); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("encode manifest: %w", err)
	}
	if err := enc.Close(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("close encoder: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close tempfile: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("rename manifest: %w", err)
	}
	return nil
}

func (s *Store) Exists(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	_, err := os.Stat(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat manifest: %w", err)
	}
	return true, nil
}
