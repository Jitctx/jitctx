package contracts

// ExtractContractsInput is the input VO for contractsuc.UseCase.
//
// Validation rules enforced by the use case (NOT by this struct):
//   - TargetFile MUST be non-empty.
//   - Exactly one of Feature OR FilePath MAY be set; both empty is allowed
//     (then ONLY the manifest fallback is attempted). Both set is rejected.
//   - BaseDir is the workdir root (Config.WorkDir at call sites; t.TempDir()
//     in tests). It is forwarded to the FindSpecFilePort and to the
//     manifest store paths through the wired Deps.
//   - PlansDir mirrors Config.PlansDir verbatim ("" when unset).
type ExtractContractsInput struct {
	TargetFile string // java file path; basename (minus extension) becomes the contract name
	Feature    string // mutually exclusive with FilePath; spec source for slice
	FilePath   string // mutually exclusive with Feature; explicit spec path
	BaseDir    string // workdir root for spec resolution
	PlansDir   string // configured override; empty when not set
}

// ExtractContractsOutput is the output VO for contractsuc.UseCase.
//
// Target is the contract that owns the file at TargetFile.
// Related is the deduplicated, alphabetically-sorted set of contracts that
// Target's Uses, Implements, and DependsOn fields reference AND that exist
// in the consulted source (spec or manifest). Unknown references do NOT
// appear in Related — they are logged to stderr as a warning by the use
// case (mirrors planuc's external-reference behaviour from US-003).
type ExtractContractsOutput struct {
	Source  string             // "spec" or "manifest"; informational, used by formatter header
	Target  ContractFragment   // the contract named like the file
	Related []ContractFragment // dependencies, alphabetical by Name, distinct
}

// ContractFragment is the projection of one contract for slice
// rendering. EP04US-003 keeps Type (string) for the spec-sourced
// projection (which stays singular per RF-015) and adds Types
// ([]string) for the manifest-sourced projection. Exactly one of
// {Type, Types} is non-zero per fragment depending on Source.
type ContractFragment struct {
	Name        string
	Type        string   // populated when Source == "spec"; empty when Source == "manifest"
	Types       []string // populated when Source == "manifest"; nil when Source == "spec"  (EP04US-003 NEW)
	Path        string   // relative target path computed via ContractPathMapper
	Methods     []string // raw signatures as declared
	Fields      []string
	Uses        []string
	Implements  string
	DependsOn   []string
	Endpoints   []string
	Annotations []string
	Role        string // human-readable role string from ContractRoleDescriber
}
