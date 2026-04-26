package model

// ContractType is the canonical identifier enum used by SpecContract
// (the spec-side authoring path). RF-015 explicitly preserves singular
// Type semantics for specs. The seven existing constants stay as the
// reference vocabulary for spec authoring.
type ContractType string

const (
	ContractInputPort   ContractType = "input-port"
	ContractOutputPort  ContractType = "output-port"
	ContractEntity      ContractType = "entity"
	ContractAggregate   ContractType = "aggregate-root"
	ContractService     ContractType = "service"
	ContractRestAdapter ContractType = "rest-adapter"
	ContractJPAAdapter  ContractType = "jpa-adapter"
)

// Contract is the manifest-side projection of a code element. EP04US-003
// migrates the singular Type field to Types []string to support
// multi-classification (RF-005). Types carries the IDs returned by
// profile.ClassifyDeclarativePort in declared order; a contract that
// matched zero rules has an empty (non-nil) slice. The slice element
// type is plain string — NOT model.ContractType — because profile
// authors may declare custom IDs (e.g. "domain-event") that are not
// in the constant block above.
type Contract struct {
	Name    string
	Types   []string // EP04US-003 (was: Type ContractType)
	Path    string
	Methods []Method
}

type Method struct {
	Signature string
}
