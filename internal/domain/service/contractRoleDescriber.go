package service

import (
	"strings"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// ContractRoleDescriber is a stateless pure function that produces a
// human-readable one-line role description for a contract. Used by the
// contracts slice renderer to populate ContractFragment.Role.
//
// The mapping is deterministic and matches the templates documented in §8 Q6:
//
//	input-port      → "Input port (use case interface)"
//	output-port     → "Output port (driven port)"
//	entity          → "Domain entity"
//	aggregate-root  → "Aggregate root"
//	service         → "Service implementing <Implements>; depends on <DependsOn[*]>"
//	                  (degenerates gracefully when fields are empty)
//	rest-adapter    → "REST adapter; calls <Uses[*]>"
//	jpa-adapter     → "JPA adapter implementing <Implements>"
//	<unknown>       → "Contract of type <type>"  (no error — this is a
//	                  presentational hint, not a domain invariant check)
type ContractRoleDescriber struct{}

// NewContractRoleDescriber returns a stateless describer.
func NewContractRoleDescriber() ContractRoleDescriber { return ContractRoleDescriber{} }

// Describe returns the role string for the given spec contract. Pure.
func (ContractRoleDescriber) Describe(c model.SpecContract) string {
	switch c.Type {
	case model.ContractInputPort:
		return "Input port (use case interface)"
	case model.ContractOutputPort:
		return "Output port (driven port)"
	case model.ContractEntity:
		return "Domain entity"
	case model.ContractAggregate:
		return "Aggregate root"
	case model.ContractService:
		var b strings.Builder
		b.WriteString("Service")
		if c.Implements != "" {
			b.WriteString(" implementing ")
			b.WriteString(c.Implements)
		}
		if len(c.DependsOn) > 0 {
			if c.Implements != "" {
				b.WriteString(";")
			}
			b.WriteString(" depends on ")
			b.WriteString(strings.Join(c.DependsOn, ", "))
		}
		return b.String()
	case model.ContractRestAdapter:
		if len(c.Uses) == 0 {
			return "REST adapter"
		}
		return "REST adapter; calls " + strings.Join(c.Uses, ", ")
	case model.ContractJPAAdapter:
		if c.Implements == "" {
			return "JPA adapter"
		}
		return "JPA adapter implementing " + c.Implements
	default:
		return "Contract of type " + string(c.Type)
	}
}
