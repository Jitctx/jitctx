package profile

// ResolveProfileInput is the input VO for ResolveProfilePort.Resolve.
// It encodes the two axes the resolver discriminates on:
//   - Name: explicit selector (the --profile flag); empty means
//     auto-detect.
//   - WorkDir: the project root used to resolve the user-profile
//     directory (<WorkDir>/<ProfilesDir>/<Name>/).
//   - ProfilesDir: the profiles directory relative to WorkDir
//     (typically ".jitctx/profiles" from cfg.ProfilesDir).
//
// The resolver does NOT widen its surface to discover Tree-sitter
// queries or other side effects — Resolve returns a *ProfileBundle
// and nothing else.
type ResolveProfileInput struct {
	Name        string
	WorkDir     string
	ProfilesDir string
}
