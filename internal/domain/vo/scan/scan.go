package scan

type ScanProjectInput struct {
	WorkDir      string // project root, absolute preferred
	ProfileName  string // "" means auto-detect
	ManifestPath string // absolute or relative to WorkDir
}

type ScanProjectOutput struct {
	ManifestPath string
	ModuleCount  int
	ContextCount int      // number of context files discovered
	SkippedFiles []string // unparseable Java files that were tolerated
}
