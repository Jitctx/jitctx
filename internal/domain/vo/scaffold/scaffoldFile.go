package scaffold

// ScaffoldFile is one rendered artifact pending atomic batch write.
// It generalises ProductionFile so the writer can take a single flat
// slice that mixes production AND test files in one rename pass per
// EP02RF-009 (§8 Q4 freeze).
//
//	Path:    BaseDir-anchored absolute path (filepath.Clean'd).
//	Content: fully-rendered Java source.
//	Kind:    classification used by callers/observers; the writer is
//	         agnostic and treats every file the same way.
type ScaffoldFile struct {
	Path    string
	Content []byte
	Kind    ScaffoldFileKind
}

// ScaffoldFileKind is the provenance tag attached to each ScaffoldFile.
type ScaffoldFileKind string

const (
	KindProduction ScaffoldFileKind = "production"
	KindTest       ScaffoldFileKind = "test"
)
