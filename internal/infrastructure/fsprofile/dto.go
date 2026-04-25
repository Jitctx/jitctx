package fsprofile

// profileDTO is the YAML deserialization shape for a framework profile.
type profileDTO struct {
	Name            string             `yaml:"name"`
	Languages       []string           `yaml:"languages"`
	QueryLang       string             `yaml:"query_lang"`
	Detect          detectDTO          `yaml:"detect"`
	ModuleDetection moduleDetectionDTO `yaml:"module_detection"`
	Rules           []ruleDTO          `yaml:"rules"`
	AuditRules      []auditRuleDTO     `yaml:"audit_rules"`
}

type detectDTO struct {
	Files []fileMatcherDTO `yaml:"files"`
}

type fileMatcherDTO struct {
	Name     string `yaml:"name"`
	Contains string `yaml:"contains"`
}

type moduleDetectionDTO struct {
	Strategy string      `yaml:"strategy"`
	Roots    []string    `yaml:"roots"`
	Markers  []markerDTO `yaml:"markers"`
}

type markerDTO struct {
	Kind  string `yaml:"kind"`
	Value string `yaml:"value"`
}

type ruleDTO struct {
	Match      matchDTO `yaml:"match"`
	ClassifyAs string   `yaml:"classify_as"`
}

type auditRuleDTO struct {
	ID          string            `yaml:"id"`
	Kind        string            `yaml:"kind"`
	Severity    string            `yaml:"severity"`
	Description string            `yaml:"description"`
	Suggestion  string            `yaml:"suggestion"`
	Params      map[string]string `yaml:"params"`
}

type matchDTO struct {
	NodeType      string `yaml:"node_type"`
	PathContains  string `yaml:"path_contains"`
	HasAnnotation string `yaml:"has_annotation"`
	Implements    string `yaml:"implements"`
}
