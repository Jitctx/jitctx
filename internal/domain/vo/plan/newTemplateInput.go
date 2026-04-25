package plan

// NewTemplateInput is the input VO for plannewuc.UseCase. It carries the
// already-resolved feature/module identifiers and the resolved base
// directory under which the template file must be written. Resolution of
// PlansDir (config vs. convention) happens in the presentation layer
// before Execute is invoked, so the use case stays pure.
type NewTemplateInput struct {
	Feature string // kebab-case feature name (used as <feature>.md)
	Module  string // kebab-case module identifier
	BaseDir string // absolute or working-dir-relative directory where <feature>.md is written
}
