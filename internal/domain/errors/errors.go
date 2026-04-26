package errors

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrManifestNotFound = errors.New("manifest not found")
	ErrModuleNotFound   = errors.New("module not found")
	ErrProfileInvalid   = errors.New("profile invalid")
	ErrNotImplemented   = errors.New("not implemented") // used only by stubs

	// US-001 sentinels
	ErrNoProfileMatch = errors.New("no matching framework profile")
	ErrParseFailure   = errors.New("java parse failure") // fatal parse (not partial)
	ErrPartialParse   = errors.New("java partial parse") // recoverable, file skipped
	ErrManifestWrite  = errors.New("manifest write failed")

	// US-001 (spec parser) sentinels
	ErrSpecParse = errors.New("spec parse error") // sentinel for all fatal spec parse errors

	// EP02US-002 sentinels
	ErrSpecFileExists  = errors.New("spec file already exists")
	ErrSpecWriteFailed = errors.New("spec template write failed")

	// EP02US-003 sentinels
	ErrDependencyCycle         = errors.New("dependency cycle detected")
	ErrUnsupportedContractType = errors.New("unsupported contract type")
	ErrSpecFileNotFound        = errors.New("spec file not found")
)

type ProfileConflictError struct {
	Name string
}

func (e *ProfileConflictError) Error() string {
	return fmt.Sprintf("framework profile conflict: %s", e.Name)
}

func (e *ProfileConflictError) Is(target error) bool {
	return errors.Is(target, ErrProfileInvalid)
}

// ModuleNotFoundError carries the queried module id and the sorted list of
// available module ids so the presentation layer can produce the
// user-friendly stderr required by EP01RNF-003.
type ModuleNotFoundError struct {
	Queried         string   // the id the user asked for
	AvailableSorted []string // module ids, alphabetically sorted
}

func (e *ModuleNotFoundError) Error() string {
	return fmt.Sprintf("module %q not found", e.Queried)
}

// Is makes errors.Is(err, ErrModuleNotFound) return true so existing
// call-sites keep working after we introduce the typed variant.
func (e *ModuleNotFoundError) Is(target error) bool {
	return errors.Is(target, ErrModuleNotFound)
}

// SpecParseError carries structured information about a fatal parse failure.
type SpecParseError struct {
	Line    int    // 1-based line number where the error was detected
	Message string // human-readable description
}

func (e *SpecParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}

func (e *SpecParseError) Is(target error) bool {
	return target == ErrSpecParse
}

// SpecParseWarning represents a non-fatal parse issue (unknown field, etc.).
// Warnings are collected during parsing and logged to stderr; they do not
// abort the parse.
type SpecParseWarning struct {
	Line    int
	Message string
}

func (w *SpecParseWarning) Error() string {
	return fmt.Sprintf("line %d: %s", w.Line, w.Message)
}

// DuplicateContractError is a specific fatal error for duplicate contract names.
// It records both occurrence line numbers for the user-facing message.
type DuplicateContractError struct {
	Name      string
	FirstLine int
	DupeLine  int
}

func (e *DuplicateContractError) Error() string {
	return fmt.Sprintf("duplicate contract name '%s' (first at line %d, again at line %d)",
		e.Name, e.FirstLine, e.DupeLine)
}

func (e *DuplicateContractError) Is(target error) bool {
	return target == ErrSpecParse
}

// SpecFileExistsError carries the conflicting path so the presentation
// layer can echo it to stderr verbatim (matches the EP02US-002 acceptance
// criterion: stderr contains "spec file already exists").
type SpecFileExistsError struct {
	Path string
}

func (e *SpecFileExistsError) Error() string {
	return fmt.Sprintf("spec file already exists: %s", e.Path)
}

func (e *SpecFileExistsError) Is(target error) bool {
	return target == ErrSpecFileExists
}

// SpecFileNotFoundError carries the list of paths the resolver searched
// so the presentation layer can echo them verbatim per the .feature
// acceptance (stderr lists every path that was searched).
type SpecFileNotFoundError struct {
	Feature  string
	Searched []string
}

func (e *SpecFileNotFoundError) Error() string {
	return fmt.Sprintf("spec file not found for feature %q; searched %d location(s)", e.Feature, len(e.Searched))
}

func (e *SpecFileNotFoundError) Is(target error) bool {
	return target == ErrSpecFileNotFound
}

// UnsupportedContractTypeError carries the offending type and the sorted
// list of supported types for the user-facing message.
type UnsupportedContractTypeError struct {
	Type            string
	SupportedSorted []string
}

func (e *UnsupportedContractTypeError) Error() string {
	return fmt.Sprintf("unsupported contract type %q", e.Type)
}

func (e *UnsupportedContractTypeError) Is(target error) bool {
	return target == ErrUnsupportedContractType
}

// EP02US-004 sentinels
var ErrContractTargetNotFound = errors.New("contract target not found in spec or manifest")

// EP02US-005 sentinels
var (
	ErrScaffoldConflict   = errors.New("scaffold conflict: target file already exists")
	ErrScaffoldRender     = errors.New("scaffold render failed")
	ErrSpecMissingPackage = errors.New("spec is missing required Package field")
)

// EP03US-002 sentinels
var (
	ErrAuditProfileMissing = errors.New("active profile has no audit rules")
)

// EP03US-006 sentinels
var ErrGitUnavailable = errors.New("git not available")

// AuditUnknownRuleKindError is raised when a profile YAML declares a rule
// kind the running jitctx binary does not know about. Surfaced as a
// warning by the use case (logged to stderr, rule skipped); never fails
// the audit. Wraps ErrProfileInvalid for errors.Is matching.
type AuditUnknownRuleKindError struct {
	RuleID string
	Kind   string
}

func (e *AuditUnknownRuleKindError) Error() string {
	return fmt.Sprintf("audit rule %q: unknown kind %q", e.RuleID, e.Kind)
}

func (e *AuditUnknownRuleKindError) Is(target error) bool {
	return errors.Is(target, ErrProfileInvalid)
}

// ContractTargetNotFoundError carries the resolution context so the
// presentation layer can produce the user-friendly stderr required by the
// .feature scenario "Contracts slice fails for unknown file".
type ContractTargetNotFoundError struct {
	TargetFile       string // verbatim --for value
	ContractName     string // basename-derived name from ContractTargetResolver
	SearchedSpec     bool   // true when a spec lookup was attempted
	SearchedManifest bool   // true when a manifest fallback was attempted
}

func (e *ContractTargetNotFoundError) Error() string {
	return fmt.Sprintf("contract %q (from %s) not found in spec or manifest",
		e.ContractName, e.TargetFile)
}

func (e *ContractTargetNotFoundError) Is(target error) bool {
	return target == ErrContractTargetNotFound
}

// ScaffoldConflictError carries the alphabetically sorted list of conflicting
// paths so format/errors.go can render them as bullets and so tests can
// assert them deterministically.
type ScaffoldConflictError struct {
	Conflicts []string
}

func (e *ScaffoldConflictError) Error() string {
	return "scaffold conflict: target files already exist: " + strings.Join(e.Conflicts, ", ")
}

func (e *ScaffoldConflictError) Is(target error) bool { return target == ErrScaffoldConflict }

// ScaffoldRenderError carries the offending contract name and the underlying
// cause (typically a text/template execution error or an
// UnsupportedContractTypeError surfaced from the renderer).
type ScaffoldRenderError struct {
	Contract string
	Cause    error
}

func (e *ScaffoldRenderError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("scaffold render contract %q failed", e.Contract)
	}
	return fmt.Sprintf("scaffold render contract %q failed: %s", e.Contract, e.Cause.Error())
}

func (e *ScaffoldRenderError) Unwrap() error { return e.Cause }

func (e *ScaffoldRenderError) Is(target error) bool { return target == ErrScaffoldRender }

// EP04US-002 sentinels — appended at the bottom of the existing var block.
var (
	// ErrClassificationInvalid is returned when a classification rule entry
	// is structurally malformed at YAML decode time (e.g., the `kind` value
	// is a sequence instead of a string). Wraps ErrProfileInvalid for
	// errors.Is matching.
	ErrClassificationInvalid = errors.New("classification rule invalid")
)

// EP04US-001 sentinels — appended at the bottom of the existing var blocks.
var (
	// ErrProfileYamlMissing is returned when a profile bundle directory
	// does not contain a profile.yaml file. Its Error() string is the
	// literal "profile.yaml not found" required by the .feature file.
	ErrProfileYamlMissing = errors.New("profile.yaml not found")

	// ErrTemplateMissing is returned when a profile.yaml declares a type
	// whose template file is not present under templates/. Wraps
	// ErrProfileInvalid for errors.Is matching.
	ErrTemplateMissing = errors.New("profile template missing")

	// ErrBundledProfileNotFound is returned when LoadBundledProfilePort
	// is asked for a profile name that is not embedded in the binary.
	ErrBundledProfileNotFound = errors.New("bundled profile not found")
)

// TemplateMissingError carries the profile name, the type ID, and the
// missing template filename so the canonical user-facing message can be
// reproduced by any consumer. The Error() string is the EP04US-001
// pinned literal:
//
//	profile "spring-boot-hexagonal": type "service" references missing template "service.java.tmpl"
//
// errors.Is(err, ErrTemplateMissing) and errors.Is(err, ErrProfileInvalid)
// both return true.
type TemplateMissingError struct {
	ProfileName string
	TypeID      string
	Template    string
}

func (e *TemplateMissingError) Error() string {
	return fmt.Sprintf("profile %q: type %q references missing template %q",
		e.ProfileName, e.TypeID, e.Template)
}

func (e *TemplateMissingError) Is(target error) bool {
	return target == ErrTemplateMissing || errors.Is(target, ErrProfileInvalid)
}
