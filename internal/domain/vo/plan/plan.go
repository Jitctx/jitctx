package plan

// ExecutionLayer is one stratum of the topological sort. Index is 0-based
// and monotonically increasing; targets within a single layer are
// independent of one another and may be implemented in parallel.
type ExecutionLayer struct {
	Index   int
	Targets []PlanTarget
}

// PlanTarget describes a single contract scheduled within a layer.
// All fields are populated by planuc; the formatter is responsible for
// presentation. Uses/Implements/DependsOn are echoed verbatim from the
// SpecContract so JSON consumers can recompute the graph if they wish.
type PlanTarget struct {
	Name       string   // PascalCase contract identifier (e.g. "UserServiceImpl")
	Type       string   // model.ContractType as a string ("input-port", …)
	TargetPath string   // workdir-relative path produced by ContractPathMapper
	Uses       []string // copy of SpecContract.Uses
	Implements string   // copy of SpecContract.Implements (may be empty)
	DependsOn  []string // copy of SpecContract.DependsOn
}
