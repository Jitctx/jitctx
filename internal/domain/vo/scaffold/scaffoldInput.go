package scaffold

// ScaffoldInput is the input VO for scaffolduc.UseCase. Exactly one of
// Feature OR FilePath MUST be non-empty (validated by the use case).
// BaseDir is the workdir root used to (a) resolve --feature via FindSpecFilePort
// and (b) ANCHOR every generated path under <BaseDir>/src/main/java/<...>.
type ScaffoldInput struct {
	Feature  string // exclusive with FilePath
	FilePath string // exclusive with Feature; absolute or BaseDir-relative
	BaseDir  string // workdir root for resolution + write anchoring
	PlansDir string // configured override; empty when not set
}
