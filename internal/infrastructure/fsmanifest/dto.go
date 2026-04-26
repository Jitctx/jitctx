package fsmanifest

import "time"

// CurrentManifestSchemaVersion is the schema version written by this binary.
// Manifests with a schema_version < 2 (including v1 manifests without the
// field, which decode as 0) are rejected by Store.Load with
// domerr.ErrManifestSchemaOutdated.
const CurrentManifestSchemaVersion = 2

type projectStateDTO struct {
	SchemaVersion int          `yaml:"schema_version"`
	GeneratedAt   time.Time    `yaml:"generated_at"`
	Stack         stackDTO     `yaml:"stack"`
	Modules       []moduleDTO  `yaml:"modules"`
	Contexts      []contextDTO `yaml:"contexts"`
}

type stackDTO struct {
	Languages  []string `yaml:"languages"`
	Frameworks []string `yaml:"frameworks"`
}

type moduleDTO struct {
	ID           string        `yaml:"id"`
	Path         string        `yaml:"path"`
	Tags         []string      `yaml:"tags"` // always emitted, no omitempty
	Contracts    []contractDTO `yaml:"contracts"`
	Dependencies []string      `yaml:"dependencies"` // always emitted, no omitempty
}

type contractDTO struct {
	Name    string      `yaml:"name"`
	Types   []string    `yaml:"types"`
	Path    string      `yaml:"path"`
	Methods []methodDTO `yaml:"methods"`
}

type methodDTO struct {
	Signature string `yaml:"signature"`
}

type contextDTO struct {
	ID            string   `yaml:"id"`
	Type          string   `yaml:"type"`
	AppliesTo     []string `yaml:"applies_to,omitempty"`
	Module        string   `yaml:"module,omitempty"`
	Tags          []string `yaml:"tags"`
	Path          string   `yaml:"path"`
	TokenEstimate int      `yaml:"token_estimate"`
}
