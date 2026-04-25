package format

import (
	"errors"
	"fmt"
	"strings"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/service"
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
	var sfnf *domerr.SpecFileNotFoundError
	if errors.As(err, &sfnf) {
		var b strings.Builder
		fmt.Fprintf(&b, "spec file not found for feature %q; searched:", sfnf.Feature)
		for _, p := range sfnf.Searched {
			fmt.Fprintf(&b, "\n  - %s", p)
		}
		return errors.New(b.String())
	}
	var uct *domerr.UnsupportedContractTypeError
	if errors.As(err, &uct) {
		return fmt.Errorf("unsupported contract type %q; supported: %s",
			uct.Type, strings.Join(uct.SupportedSorted, ", "))
	}
	var ce *service.CycleError
	if errors.As(err, &ce) {
		return fmt.Errorf("dependency cycle detected: %s", strings.Join(ce.Path, " -> "))
	}
	var ctnf *domerr.ContractTargetNotFoundError
	if errors.As(err, &ctnf) {
		var hint string
		switch {
		case ctnf.SearchedSpec && ctnf.SearchedManifest:
			hint = "run `jitctx scan` or `jitctx plan` first"
		case ctnf.SearchedManifest:
			hint = "run `jitctx scan` first to populate project-state.yaml, " +
				"or pass --feature/--file to consult a spec"
		case ctnf.SearchedSpec:
			hint = "run `jitctx plan --new` to create a spec, " +
				"or `jitctx scan` to populate project-state.yaml"
		default:
			hint = "run `jitctx scan` or `jitctx plan` first"
		}
		return fmt.Errorf("could not find contract %q (from %s) in spec or project-state.yaml; %s",
			ctnf.ContractName, ctnf.TargetFile, hint)
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
