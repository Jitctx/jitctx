package profile

// ProfileInitInput parameters for the profile init use case.
//
// WorkDir defaults to "." at the cobra layer; ProfilesDir defaults to
// the value from cfg.ProfilesDir (typically ".jitctx/profiles") at the
// composition root.
type ProfileInitInput struct {
	// Name of the bundled profile to extract; required. Must match a
	// directory name under fsprofile/bundled/.
	Name string

	// WorkDir is the project root. The target directory becomes
	// <WorkDir>/<ProfilesDir>/<Name>/.
	WorkDir string

	// ProfilesDir is the profiles directory relative to WorkDir.
	// Typically ".jitctx/profiles".
	ProfilesDir string
}
