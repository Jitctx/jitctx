package model

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

type Contract struct {
	Name    string
	Type    ContractType
	Path    string
	Methods []Method
}

type Method struct {
	Signature string
}
