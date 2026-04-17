package fsprofile

import (
	"bytes"
	"context"
	"embed"
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

//go:embed bundled/*.yaml
var embeddedProfiles embed.FS

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

// Load loads a profile by name. Looks in userDir first, then bundled profiles.
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
		if err == nil {
			prof, parseErr := decodeProfile(data, true)
			if parseErr != nil {
				l.logger.Warn("custom profile parse error", "file", path, "reason", parseErr)
				// Fall through to bundled.
				break
			}
			return prof, nil
		}
	}
	// Try bundled.
	data, err := embeddedProfiles.ReadFile("bundled/" + name + ".yaml")
	if err != nil {
		return nil, fmt.Errorf("profile %q not found: %w", name, domerr.ErrProfileInvalid)
	}
	return decodeProfile(data, true)
}

// List returns the names of all available profiles (custom + bundled, deduplicated).
func (l *Loader) List(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	names := make(map[string]bool)

	// Custom profiles.
	if entries, err := os.ReadDir(l.userDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && (strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml")) {
				name := strings.TrimSuffix(strings.TrimSuffix(e.Name(), ".yml"), ".yaml")
				names[name] = true
			}
		}
	}

	// Bundled profiles.
	entries, err := fs.ReadDir(embeddedProfiles, "bundled")
	if err != nil {
		return nil, fmt.Errorf("list bundled profiles: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			name := strings.TrimSuffix(e.Name(), ".yaml")
			names[name] = true
		}
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
