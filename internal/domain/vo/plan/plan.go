package plan

type PlanModuleInput struct {
	Module string
}

type PlanModuleOutput struct {
	Module string
	Layers []ExecutionLayer
}

type ExecutionLayer struct {
	Index    int
	Parallel bool
	Targets  []PlanTarget
}

type PlanTarget struct {
	Path string
	Kind string
}
