package audit

// AuditProjectOutput is the use case output. The presentation layer
// renders this into markdown; the use case never produces strings itself.
type AuditProjectOutput struct {
	// ProfileName is the name of the profile the rules came from. Echoed
	// in the report header; empty when no profile matched (which is an
	// error path — see ErrNoProfileMatch).
	ProfileName string

	// ManifestPath is the path that was loaded; surfaced for debugging.
	ManifestPath string

	// Modules is the sorted (by ID) list of modules that produced violations
	// OR have at least one source file scanned. Modules with zero violations
	// still appear so the report is grouped consistently — but each module
	// renders only when it has at least one violation under it.
	Modules []AuditModuleReport

	// Sintatic is the flat, sorted list of every violation found across all
	// modules. The renderer groups by ModuleID at render time. Sorting key:
	// (ModuleID, FilePath, Line, RuleID).
	Sintatic []AuditViolation

	// SemanticPlaceholder is always the literal string
	// "Semantic analysis not enabled. Future versions of jitctx may support deeper checks via analyzer plugins."
	// The renderer emits it under "## Semantic Analysis". Centralised here
	// so unit tests can assert the string verbatim (RNF-005).
	SemanticPlaceholder string
}

// AuditModuleReport is the per-module section header data. Renderer emits
// a "## Module: <ID>" heading per non-empty entry.
type AuditModuleReport struct {
	ModuleID string
	Path     string
}
