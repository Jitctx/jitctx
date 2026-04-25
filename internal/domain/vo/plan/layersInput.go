package plan

// LayersInput is the input VO for planuc.UseCase. Exactly one of Feature
// or FilePath MUST be non-empty (validated by the use case before any
// I/O). BaseDir is the workdir root that the read-side resolver
// considers "current" — for production runs it is Config.WorkDir; for
// tests, t.TempDir(). PlansDir mirrors Config.PlansDir verbatim ("" when
// unset).
type LayersInput struct {
	Feature  string // exclusive with FilePath
	FilePath string // exclusive with Feature; absolute or BaseDir-relative
	BaseDir  string // workdir root for resolution + target_path computation
	PlansDir string // configured override; empty when not set
}
