package scaffold

// ScaffoldOutput is the output VO for scaffolduc.UseCase.
//
// WrittenPaths is the FLAT, alphabetically sorted list of every file
// committed to disk (production + test) — chosen over a partitioned
// {ProductionPaths, TestPaths} shape because:
//
//	(a) the existing JSON DTO and stderr summary already sort flat lists,
//	(b) determinism (RNF-002) only requires sorted ordering, and
//	(c) downstream consumers (tools / CI parsers) can re-classify by
//	    ".../src/test/java/" substring without an enum lookup.
//
// ProductionCount and TestCount are convenience counters so the
// presentation layer can render "wrote N files (P production, T test):"
// without re-scanning the path list.
type ScaffoldOutput struct {
	Feature         string
	Module          string
	Package         string
	WrittenPaths    []string
	ProductionCount int
	TestCount       int
}
