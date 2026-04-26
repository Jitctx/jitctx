package diff

// DiffActionType is the small enum of action kinds emitted by the
// contractDiffer. Stringly-typed for renderer convenience.
type DiffActionType string

const (
	DiffActionCreate DiffActionType = "CREATE"
	DiffActionModify DiffActionType = "MODIFY"
	DiffActionExtra  DiffActionType = "EXTRA"
)

// DiffSeverity is the diff renderer's severity enum. Independent of
// model.AuditSeverity to avoid a vo/diff → vo/audit cross-import; the
// emoji+label strings are intentionally identical to keep the audit and
// diff renderers visually consistent (RF-007).
type DiffSeverity string

const (
	DiffSeverityError   DiffSeverity = "ERROR"
	DiffSeverityWarning DiffSeverity = "WARNING"
	DiffSeverityInfo    DiffSeverity = "INFO"
)

// DiffAction is the per-contract diff output. EP04US-003 splits
// ContractType into ContractType (singular, sourced from SpecContract
// for CREATE/MODIFY actions) and ContractTypes ([]string, sourced
// from manifest Contract for EXTRA actions). The differ writes one or
// the other; presentation layers render whichever is non-zero.
//
//   - Type: CREATE | MODIFY | EXTRA.
//   - ContractName: PascalCase identity of the contract.
//   - ContractType: SpecContract.Type for CREATE/MODIFY (singular per RF-015).
//   - ContractTypes: manifest Contract.Types for EXTRA (plural, EP04US-003 NEW).
//   - Severity: ERROR for CREATE, WARNING for MODIFY, INFO for EXTRA.
//   - Layer: 0-based execution layer for CREATE/MODIFY; -1 for EXTRA
//     (EXTRA is layer-less and rendered in its own section).
//   - AddedMethods / RemovedMethods: populated for MODIFY only;
//     alphabetically sorted; both empty for CREATE / EXTRA.
type DiffAction struct {
	Type           DiffActionType
	ContractName   string
	ContractType   string   // SpecContract.Type for CREATE/MODIFY
	ContractTypes  []string // manifest Contract.Types for EXTRA  (EP04US-003 NEW)
	Severity       DiffSeverity
	Layer          int
	AddedMethods   []string
	RemovedMethods []string
}
