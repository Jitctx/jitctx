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
