package audit

// AuditProjectInput is the use case input for `jitctx audit`.
// All paths are absolute or relative to WorkDir; ManifestPath is resolved
// in PreRunE and passed in verbatim.
type AuditProjectInput struct {
	WorkDir      string // project root, absolute preferred
	ManifestPath string // absolute or relative to WorkDir
	ProfileName  string // "" means auto-detect via DetectProfilePort
}
