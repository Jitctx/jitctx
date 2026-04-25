package fsprofile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
)

// Loader implements LoadProfilePort and ListProfilesPort.
type Loader struct {
	userDir string
	logger  *slog.Logger
}

// New creates a new Loader. userDir is the directory for custom profile YAML files.
func New(userDir string) *Loader {
	return &Loader{userDir: userDir, logger: slog.Default()}
}

// NewWithLogger creates a new Loader with a custom logger.
func NewWithLogger(userDir string, logger *slog.Logger) *Loader {
	return &Loader{userDir: userDir, logger: logger}
}

// Load loads a profile by name from the user profiles directory.
// Returns ErrProfileInvalid if the profile is not found or fails to parse.
func (l *Loader) Load(ctx context.Context, name string) (*model.FrameworkProfile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// SEC-001: reject names that attempt path traversal.
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return nil, fmt.Errorf("profile %q: invalid name: %w", name, domerr.ErrProfileInvalid)
	}
	rootAbs, err := filepath.Abs(l.userDir)
	if err != nil {
		return nil, fmt.Errorf("profile %q: resolve profiles dir: %w", name, err)
	}
	for _, ext := range []string{".yaml", ".yml"} {
		candidate := filepath.Clean(filepath.Join(rootAbs, name+ext))
		if !strings.HasPrefix(candidate, rootAbs+string(filepath.Separator)) {
			return nil, fmt.Errorf("profile %q: escapes profiles dir: %w", name, domerr.ErrProfileInvalid)
		}
	}
	// Try user directory.
	for _, ext := range []string{".yaml", ".yml"} {
		path := filepath.Join(l.userDir, name+ext)
		data, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read profile %q: %w", path, err)
		}
		prof, parseErr := decodeProfile(data, true)
		if parseErr != nil {
			return nil, fmt.Errorf("profile %q: %w: %w", path, parseErr, domerr.ErrProfileInvalid)
		}
		prof.Source = model.ProfileSourceCustom
		return prof, nil
	}
	return nil, fmt.Errorf("profile %q not found: %w", name, domerr.ErrProfileInvalid)
}

// List returns the names of all available profiles from the user directory.
// Returns an empty slice (not an error) when the directory is absent.
func (l *Loader) List(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(l.userDir)
	if errors.Is(err, fs.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read profiles dir %q: %w", l.userDir, err)
	}

	names := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasSuffix(n, ".yaml") && !strings.HasSuffix(n, ".yml") {
			continue
		}
		base := strings.TrimSuffix(strings.TrimSuffix(n, ".yml"), ".yaml")
		names[base] = true
	}

	result := make([]string, 0, len(names))
	for n := range names {
		result = append(result, n)
	}
	sort.Strings(result)
	return result, nil
}

// decodeProfile decodes a YAML profile. If strict=true uses KnownFields(true).
func decodeProfile(data []byte, strict bool) (*model.FrameworkProfile, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(strict)
	var dto profileDTO
	if err := dec.Decode(&dto); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
	}
	return toDomain(dto)
}
