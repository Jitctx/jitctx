package diff

// DiffPlanInput is the input VO for diffuc.UseCase. Either Feature or
// FilePath is set (mutually exclusive — same rule as planuc.LayersInput).
// BaseDir is the workdir root used both for spec resolution AND for
// passing to the manifest loader's filesystem context. PlansDir mirrors
// Config.PlansDir verbatim ("" when unset).
type DiffPlanInput struct {
	Feature  string // exclusive with FilePath
	FilePath string // exclusive with Feature; absolute or BaseDir-relative
	BaseDir  string // workdir root for resolution
	PlansDir string // configured override; "" when unset
}
