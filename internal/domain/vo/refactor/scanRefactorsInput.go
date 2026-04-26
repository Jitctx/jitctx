package refactor

// ScanRefactorsInput is the use case input for `jitctx scan --refactors`.
type ScanRefactorsInput struct {
	// WorkDir is the project root, absolute preferred. "" or "." means cwd.
	WorkDir string

	// ManifestPath is the resolved path to project-state.yaml. May be a
	// path that does not exist on disk — the use case treats absence as
	// "no module index" and groups every marker under "<unmoduled>".
	ManifestPath string
}
