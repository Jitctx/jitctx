package errors

import (
	"errors"
	"fmt"
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
