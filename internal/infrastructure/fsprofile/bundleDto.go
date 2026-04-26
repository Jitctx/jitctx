package fsprofile

import "gopkg.in/yaml.v3"

// bundleDTO is a SUPERSET of profileDTO covering the EP-04 directory shape.
// We do NOT inherit profileDTO; the directory shape will diverge in
// US-002+. For US-001 we only decode the fields needed to satisfy the
// .feature scenarios.
type bundleDTO struct {
	Name      string   `yaml:"name"`
	Language  string   `yaml:"language"`  // singular — EP-04 schema
	Languages []string `yaml:"languages"` // legacy plural — kept so the
	//  same decoder works on EP-03 sample files
	Types     []bundleTypeDTO `yaml:"types"`
	Packaging *yaml.Node      `yaml:"packaging"`
}

type bundleTypeDTO struct {
	ID             string                    `yaml:"id"`
	Template       string                    `yaml:"template"`
	Description    string                    `yaml:"description"`
	Classification []bundleClassificationDTO `yaml:"classification"`
}

// bundleClassificationDTO mirrors model.ClassificationRule one-to-one.
// The list semantics for ImplementsAll / ImplementsNone are encoded as
// YAML sequences. HasAnnotation accepts a single string (US-002 pin —
// list form deferred to US-004).
type bundleClassificationDTO struct {
	Kind           string   `yaml:"kind"`
	ImplementsAll  []string `yaml:"implements_all"`
	ImplementsNone []string `yaml:"implements_none"`
	HasAnnotation  string   `yaml:"has_annotation"`
	PathContains   string   `yaml:"path_contains"`
}
