package service

import (
	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
)

// TestPathMapper maps a (ContractType, name) pair to a workdir-relative
// JUnit 5 test source path under the Spring Boot Hexagonal layout. It is
// the test-side mirror of ContractPathMapper: it returns the same package
// suffix as the production mapper so that the test class lives in the same
// Java package as its subject, only rooted at src/test/java instead of
// src/main/java.
//
// Mapping (Spring Boot Hexagonal):
//
//	service         → application/<Name>Test.java
//	rest-adapter    → adapter/in/web/<Name>Test.java
//	entity          → domain/<Name>Test.java
//	aggregate-root  → domain/<Name>Test.java
//
// Interface contracts (input-port, output-port) and jpa-adapter
// (deferred — see §8 Q5) are NOT testable by this mapper. Calling Map
// with one of those types returns ("", nil) so the use case can SKIP the
// contract WITHOUT aborting the scaffold.
type TestPathMapper struct{}

// NewTestPathMapper returns a stateless mapper.
func NewTestPathMapper() TestPathMapper { return TestPathMapper{} }

// testSupportedTypes is the alphabetically sorted list of testable contract
// type string values used in UnsupportedContractTypeError.
var testSupportedTypes = []string{
	"aggregate-root",
	"entity",
	"input-port",
	"jpa-adapter",
	"output-port",
	"rest-adapter",
	"service",
}

// Map returns the relative test path for the contract or
// *UnsupportedContractTypeError when the contract type is unknown to the
// mapper. Returns ("", nil) for intentionally non-testable types
// (input-port, output-port, jpa-adapter). The returned path uses '/'
// separators.
func (TestPathMapper) Map(t model.ContractType, name string) (string, error) {
	switch t {
	case model.ContractService:
		return "application/" + name + "Test.java", nil
	case model.ContractRestAdapter:
		return "adapter/in/web/" + name + "Test.java", nil
	case model.ContractEntity:
		return "domain/" + name + "Test.java", nil
	case model.ContractAggregate:
		return "domain/" + name + "Test.java", nil
	case model.ContractInputPort, model.ContractOutputPort, model.ContractJPAAdapter:
		// Intentionally non-testable; caller should skip silently.
		return "", nil
	default:
		return "", &domerr.UnsupportedContractTypeError{
			Type:            string(t),
			SupportedSorted: testSupportedTypes,
		}
	}
}
