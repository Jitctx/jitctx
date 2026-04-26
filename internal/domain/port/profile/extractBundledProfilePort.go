package profile

import "context"

// ExtractBundledProfilePort writes a bundled profile's complete
// directory tree (profile.yaml, templates/<all files>, README.md when
// present) to a target filesystem path. Intended for "jitctx profile
// init" — gives the user an editable copy of an embedded profile.
//
// Implementations MUST:
//   - return errors wrapping domerr.ErrBundledProfileNotFound (or
//     a *domerr.UnknownBundledProfileError carrying the same Is
//     semantics) when the bundled profile name is unknown;
//   - return *domerr.ProfileTargetExistsError when the target
//     directory already exists;
//   - perform an atomic-ish write — assemble into a sibling tmp dir
//     and rename on success; cleanup on any error;
//   - copy bytes verbatim — no YAML re-marshalling, no template
//     parsing — so the user gets exactly what is in the binary;
//   - preserve relative directory structure (templates/ subdir).
type ExtractBundledProfilePort interface {
	Extract(ctx context.Context, name string, targetDir string) error
}
