package scaffold

// ScaffoldOutput is the output VO for scaffolduc.UseCase. WrittenPaths is
// the list of FILES actually committed to disk (post atomic rename),
// alphabetically sorted for deterministic stdout (RNF-002).
type ScaffoldOutput struct {
	Feature      string
	Module       string
	Package      string
	WrittenPaths []string
}
