package scaffold

// ProductionFile is one rendered artifact pending atomic batch write.
// Path is BaseDir-anchored absolute (filepath.Clean'd) so the writer never
// has to recompute roots. Content is the fully-rendered Java source.
type ProductionFile struct {
	Path    string
	Content []byte
}
