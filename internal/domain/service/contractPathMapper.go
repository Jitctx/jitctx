package service

import (
	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
)

// ContractPathMapper maps a (ContractType, name) pair to a workdir-
// relative target file path under the Spring Boot Hexagonal layout.
// This is a thin pure function; when EP02US-008 lands the framework-
// driven templates, the mapping moves into the profile and this service
// either becomes a thin wrapper or is deleted. Documented decision: see
// §8 Q6.
//
// Mapping (Spring Boot Hexagonal, hardcoded for EP02US-003):
//
//	input-port      → port/in/<Name>.java
//	output-port     → port/out/<Name>.java
//	service         → application/<Name>.java
//	rest-adapter    → adapter/in/web/<Name>.java
//	entity          → domain/<Name>.java
//	aggregate-root  → domain/<Name>.java
//	jpa-adapter     → adapter/out/persistence/<Name>.java
type ContractPathMapper struct{}

// NewContractPathMapper returns a stateless mapper.
func NewContractPathMapper() ContractPathMapper { return ContractPathMapper{} }

// supportedTypes is the alphabetically sorted list of supported contract type
// string values used in UnsupportedContractTypeError.
var supportedTypes = []string{
	"aggregate-root",
	"entity",
	"input-port",
	"jpa-adapter",
	"output-port",
	"rest-adapter",
	"service",
}

// Map returns the relative path for the contract or wraps
// ErrUnsupportedContractType when the type is unknown to the mapper.
// The returned path uses '/' separators (joined by filepath later by
// the caller if needed).
func (ContractPathMapper) Map(t model.ContractType, name string) (string, error) {
	switch t {
	case model.ContractInputPort:
		return "port/in/" + name + ".java", nil
	case model.ContractOutputPort:
		return "port/out/" + name + ".java", nil
	case model.ContractService:
		return "application/" + name + ".java", nil
	case model.ContractRestAdapter:
		return "adapter/in/web/" + name + ".java", nil
	case model.ContractEntity:
		return "domain/" + name + ".java", nil
	case model.ContractAggregate:
		return "domain/" + name + ".java", nil
	case model.ContractJPAAdapter:
		return "adapter/out/persistence/" + name + ".java", nil
	default:
		return "", &domerr.UnsupportedContractTypeError{
			Type:            string(t),
			SupportedSorted: supportedTypes,
		}
	}
}
