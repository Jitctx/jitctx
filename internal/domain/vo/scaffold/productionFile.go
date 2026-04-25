package scaffold

// ProductionFile is preserved as a thin alias of the legacy 1-kind shape
// so callers that explicitly model "production-only" remain expressive.
// The atomic writer no longer consumes ProductionFile directly — the use
// case wraps each ProductionFile in a ScaffoldFile{Kind: KindProduction}
// before invoking WriteAll.
//
// The struct shape is UNCHANGED to minimise churn in tests not touched
// by this story.
//
// Path is BaseDir-anchored absolute (filepath.Clean'd) so the writer never
// has to recompute roots. Content is the fully-rendered Java source.
type ProductionFile struct {
	Path    string
	Content []byte
}
