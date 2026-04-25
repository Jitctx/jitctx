package format

import (
	"errors"
	"fmt"
	"strings"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
)

// TranslateError maps a domain error to a CLI-friendly error.
// cobra (with SilenceUsage=true) prints the returned error to stderr and exits 1.
func TranslateError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domerr.ErrNoProfileMatch):
		return fmt.Errorf("no matching framework profile found; create a custom profile in .jitctx/profiles/ or add a pom.xml/build.gradle with a Spring Boot dependency")
	case errors.Is(err, domerr.ErrManifestWrite):
		return fmt.Errorf("failed to write project-state.yaml: %w; check filesystem permissions on the target directory", err)
	case errors.Is(err, domerr.ErrParseFailure):
		return fmt.Errorf("fatal java parse failure: %w", err)
	case errors.Is(err, domerr.ErrManifestNotFound):
		return fmt.Errorf("project-state.yaml not found; run 'jitctx scan' first")
	}
	var existsErr *domerr.SpecFileExistsError
	if errors.As(err, &existsErr) {
		return fmt.Errorf("spec file already exists: %s", existsErr.Path)
	}
	var mnf *domerr.ModuleNotFoundError
	if errors.As(err, &mnf) {
		return fmt.Errorf("module %q not found; available modules: %s",
			mnf.Queried, strings.Join(mnf.AvailableSorted, ", "))
	}
	switch {
	case errors.Is(err, domerr.ErrModuleNotFound):
		return fmt.Errorf("module not found in manifest")
	case errors.Is(err, domerr.ErrProfileInvalid):
		return fmt.Errorf("framework profile is invalid: %w", err)
	case errors.Is(err, domerr.ErrNotImplemented):
		return fmt.Errorf("command not yet implemented")
	}
	return fmt.Errorf("error: %w", err)
}
