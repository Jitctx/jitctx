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
