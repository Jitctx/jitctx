package fsprofile

import "gopkg.in/yaml.v3"

// bundleDTO is a SUPERSET of profileDTO covering the EP-04 directory shape.
// We do NOT inherit profileDTO; the directory shape will diverge in
// US-002+. For US-001 we only decode the fields needed to satisfy the
// .feature scenarios.
//
// Legacy EP-03 schema fields (detect, module_detection, rules, query_lang) are
// preserved here so that user-dir profiles initialised from the old flat-YAML
// shape remain usable when loaded via BundleLoader. These fields are mapped into
// FrameworkProfile in toBundleDomain so the scanner's module-detection path
// continues to work after the Detector → Resolver migration (EP04US-006).
type bundleDTO struct {
	Name      string   `yaml:"name"`
	Language  string   `yaml:"language"`  // singular — EP-04 schema
	Languages []string `yaml:"languages"` // legacy plural — kept so the
	//  same decoder works on EP-03 sample files
	QueryLang       string             `yaml:"query_lang"`       // legacy EP-03
	Detect          bundleDetectDTO    `yaml:"detect"`           // legacy EP-03
	ModuleDetection bundleModuleDetDTO `yaml:"module_detection"` // legacy EP-03
	Rules           []bundleRuleDTO    `yaml:"rules"`            // legacy EP-03
	Types           []bundleTypeDTO    `yaml:"types"`
	Packaging       *yaml.Node         `yaml:"packaging"`
}

// bundleDetectDTO mirrors detectDTO for the legacy flat-YAML schema.
type bundleDetectDTO struct {
	Files []bundleFileMatcherDTO `yaml:"files"`
}

// bundleFileMatcherDTO mirrors fileMatcherDTO for the legacy flat-YAML schema.
type bundleFileMatcherDTO struct {
	Name     string `yaml:"name"`
	Contains string `yaml:"contains"`
}

// bundleModuleDetDTO mirrors moduleDetectionDTO for the legacy flat-YAML schema.
type bundleModuleDetDTO struct {
	Strategy string            `yaml:"strategy"`
	Roots    []string          `yaml:"roots"`
	Markers  []bundleMarkerDTO `yaml:"markers"`
}

// bundleMarkerDTO mirrors markerDTO for the legacy flat-YAML schema.
type bundleMarkerDTO struct {
	Kind  string `yaml:"kind"`
	Value string `yaml:"value"`
}

// bundleRuleDTO mirrors ruleDTO for the legacy flat-YAML schema.
type bundleRuleDTO struct {
	Match      bundleMatchDTO `yaml:"match"`
	ClassifyAs string         `yaml:"classify_as"`
}

// bundleMatchDTO mirrors matchDTO for the legacy flat-YAML schema.
type bundleMatchDTO struct {
	NodeType      string `yaml:"node_type"`
	PathContains  string `yaml:"path_contains"`
	HasAnnotation string `yaml:"has_annotation"`
	Implements    string `yaml:"implements"`
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
