package model

import "github.com/jitctx/jitctx/internal/domain/vo"

// ProfileSource identifies where a profile was loaded from. Populated
// by the infrastructure adapter at load/detect time; the domain never
// constructs ProfileSource values itself.
//
// After the externalize-profiles chore there is exactly one valid source:
// "custom" — a YAML file on the filesystem under the user's profiles
// directory (default ".jitctx/profiles/"). The zero value is only valid
// in tests that construct FrameworkProfile literals directly.
type ProfileSource string

const (
	// ProfileSourceCustom marks a profile loaded from the user's profiles
	// directory on disk. (Existing — not modified.)
	ProfileSourceCustom ProfileSource = "custom"

	// ProfileSourceBundled marks a profile loaded from the binary embed
	// (//go:embed in fsprofile/bundled.go). Introduced by EP04US-001.
	ProfileSourceBundled ProfileSource = "bundled"
)

type FrameworkProfile struct {
	Name            string
	Source          ProfileSource // populated by fsprofile; zero value means "unknown" and is only valid in tests
	Detect          ProfileDetect
	ModuleDetection ModuleDetection
	Rules           []ProfileRule
	QueryLang       string
	Languages       []string

	// Language is the singular, canonical EP-04 language id derived from
	// the profile.yaml `language:` scalar. Empty when the profile pre-dates
	// EP-04 (legacy `query_lang` / `languages:[…]` only). When non-empty,
	// the bundled query registry has resolved it successfully — the
	// loader fails with ErrLanguageUnsupported before reaching this
	// assignment otherwise.
	Language vo.Language
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
