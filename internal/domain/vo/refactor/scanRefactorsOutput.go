package refactor

// ScanRefactorsOutput is the use case output. The presentation layer
// renders this into markdown; the use case never produces strings itself.
type ScanRefactorsOutput struct {
	// Markers is the flat, sorted list of every detected marker.
	// Sort key: (ModuleID, FilePath, Line, Type, Description) with
	// "<unmoduled>" sorting last. The renderer groups by ModuleID at
	// render time.
	Markers []RefactorMarker

	// UnknownTypes is the deduped, sorted list of marker types the
	// parser bucketed into "other" because they were not in the
	// recognised set. Consumed by the CLI to emit one stderr warning
	// per entry: "unknown marker type 'X'".
	UnknownTypes []string

	// ManifestPresent is true when the manifest was loaded successfully.
	// When false, the CLI may emit an info-level note that module
	// resolution is unavailable. Markers still flow under "<unmoduled>".
	ManifestPresent bool

	// StaleSkipped is true when stale-detection was skipped because
	// `.git` was absent or the `git` binary was not on PATH (EP03RF-009).
	// The CLI emits the verbatim stderr line
	//     git not detected, stale flag skipped
	// exactly once when this is true. When true, every marker in
	// Markers has Stale=false.
	StaleSkipped bool
}
