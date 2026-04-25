package plan

// LayersOutput is the output VO for planuc.UseCase. Layers preserves the
// topological ordering; within each layer, targets are alphabetically
// sorted by Name (deterministic per EP02RNF-002). Externals carries the
// distinct, alphabetically-sorted list of names referenced by any
// contract's Uses/Implements/DependsOn that did NOT resolve to an
// in-spec contract — the presentation layer renders one warning per
// entry.
type LayersOutput struct {
	Feature   string
	Module    string
	Layers    []ExecutionLayer
	Externals []string
}
