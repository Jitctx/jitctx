package model

// ProfileSource identifies where a profile was loaded from. Populated
// by the infrastructure adapter at load/detect time; the domain never
// constructs ProfileSource values itself.
type ProfileSource string

const (
	ProfileSourceBundled ProfileSource = "bundled"
	ProfileSourceCustom  ProfileSource = "custom"
)

type FrameworkProfile struct {
	Name            string
	Source          ProfileSource // populated by fsprofile; zero value means "unknown" and is only valid in tests
	Detect          ProfileDetect
	ModuleDetection ModuleDetection
	Rules           []ProfileRule
	QueryLang       string
	Languages       []string
}

type ProfileDetect struct {
	Files []ProfileFileMatcher
}

type ProfileFileMatcher struct {
	Name     string // filename relative to project root (e.g., "pom.xml")
	Contains string // substring required in the file content (case-insensitive)
}

type ModuleDetection struct {
	Strategy string   // "hexagonal" is the only supported value for US-001
	Roots    []string // glob-style path prefixes
	Markers  []ModuleMarker
}

type ModuleMarker struct {
	Kind  string // "path_contains" | "annotation"
	Value string
}

type ProfileRule struct {
	Match      ProfileMatch
	ClassifyAs ContractType
}

type ProfileMatch struct {
	NodeType      string
	PathContains  string
	HasAnnotation string
	Implements    string
}
