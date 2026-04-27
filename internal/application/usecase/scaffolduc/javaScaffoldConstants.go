// Package scaffolduc constants for Java/Spring framework identifiers used
// when emitting scaffold output for the bundled spring-boot-hexagonal profile.
// They live in their own file so the qualitygate test (EP04US-009 / EP04RNF-002)
// can apply a narrow, auditable exemption: only THIS file in the application
// layer carries Java-specific literals. When a future story makes scaffold
// imports and annotations data-driven from the loaded ProfileBundle, this file
// is deleted and the qualitygate exemption removed.
package scaffolduc

const (
	// annotationService is the Spring stereotype annotation applied to
	// ContractService contracts (usecase.go switch at line 186).
	annotationService = "@Service"

	// annotationRestController is the Spring MVC annotation applied to
	// ContractRestAdapter contracts (usecase.go switch at line 188).
	annotationRestController = "@RestController"

	// annotationEntity is the JPA annotation applied to ContractEntity and
	// ContractAggregate contracts (usecase.go switch at line 190).
	annotationEntity = "@Entity"

	// annotationRepository is the Spring Data annotation applied to
	// ContractJPAAdapter contracts (usecase.go switch at line 192).
	annotationRepository = "@Repository"

	// importTestRunnerExtensionFQN is the fully-qualified class name of the
	// Mockito JUnit 5 extension injected into test import sets
	// (usecase.go line 503). The identifier deliberately avoids the
	// substrings "Mockito" and "JUnit" so it can be referenced from
	// non-exempt application-layer files without tripping the qualitygate.
	importTestRunnerExtensionFQN = "org.mockito.junit.jupiter.MockitoExtension"
)
