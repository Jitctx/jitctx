package refactor

// RefactorMarker is one detected refactor marker in source code.
// Sorted by the use case before handoff to the renderer; sort key is
// (ModuleID, FilePath, Line, Type, Description). "<unmoduled>" sorts last.
type RefactorMarker struct {
	// ModuleID is the resolved module owner of FilePath, or "<unmoduled>"
	// when no manifest module covers the file (or no manifest exists).
	ModuleID string

	// FilePath is forward-slash, relative to the scan root.
	FilePath string

	// Line is 1-based; for block comments, the line of the opening "/*".
	Line int

	// Type is one of the MarkerType constants.
	Type MarkerType

	// Description is the parsed text after " - " for well-formed markers.
	// Empty for MarkerTypeUnparseable (see OriginalText instead).
	Description string

	// OriginalText is the full comment text (including delimiters) for
	// MarkerTypeUnparseable markers. Empty for all other types.
	OriginalText string

	// Stale is true when the marker's containing file has been modified
	// in git history more recently than the marker's own line was last
	// touched (EP03RF-009). Always false when git is unavailable
	// (see ScanRefactorsOutput.StaleSkipped) or when per-marker git
	// queries fail (e.g., file not tracked, line uncommitted).
	Stale bool
}
