// Package scaffolduc constants for framework annotations used when emitting
// scaffold output for the bundled persistence-hexagonal profile. They live
// in their own file so the scaffolding logic stays separated from the
// orchestration code in usecase.go.
//
// PC01US-014 trimmed the framework-name comments and identifiers in this
// file while preserving the @Service, @Entity, @Repository, and
// @RestController string literals — those remain as the values templates
// emit into the generated Java sources. Because those literals match
// entries in qualitygate.ForbiddenTokens, the qualitygate exemption for
// this file is retained (see internal/qualitygate/exemptions.go). The
// engine-neutrality enforcement test (PC01RNF-001) is case-sensitive on
// a different token list and finds no hits in this file.
package scaffolduc

const (
	// annotationService is the stereotype annotation applied to
	// ContractService contracts (usecase.go switch).
	annotationService = "@Service"

	// annotationRestController is the inbound-HTTP annotation applied to
	// ContractRestAdapter contracts (usecase.go switch).
	annotationRestController = "@RestController"

	// annotationEntity is the persistence-marker annotation applied to
	// ContractEntity and ContractAggregate contracts (usecase.go switch).
	annotationEntity = "@Entity"

	// annotationRepository is the persistence-stereotype annotation applied
	// to ContractPersistenceAdapter contracts (usecase.go switch).
	annotationRepository = "@Repository"
)
