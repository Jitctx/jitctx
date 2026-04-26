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

// DiffAction is one reportable item produced by the diff use case.
//
//   - Type: CREATE | MODIFY | EXTRA.
//   - ContractName: PascalCase identity of the contract.
//   - ContractType: spec/manifest type as a string (e.g. "input-port",
//     "service"). For EXTRA, sourced from the manifest contract; for
//     CREATE/MODIFY, sourced from the spec.
//   - Severity: ERROR for CREATE, WARNING for MODIFY, INFO for EXTRA.
//   - Layer: 0-based execution layer for CREATE/MODIFY; -1 for EXTRA
//     (EXTRA is layer-less and rendered in its own section).
//   - AddedMethods / RemovedMethods: populated for MODIFY only;
//     alphabetically sorted; both empty for CREATE / EXTRA.
type DiffAction struct {
	Type           DiffActionType
	ContractName   string
	ContractType   string
	Severity       DiffSeverity
	Layer          int
	AddedMethods   []string
	RemovedMethods []string
}
