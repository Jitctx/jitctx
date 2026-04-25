package model

// SpecContract represents one "## Contract:" block in a feature spec.
// Named SpecContract (not Contract) to avoid collision with the existing
// model.Contract used by EP-01's scan/query flow.
type SpecContract struct {
	Name        string       // PascalCase class/interface name
	Type        ContractType // reuses existing ContractType enum
	Methods     []string     // raw method signatures as declared
	Fields      []string     // field declarations (e.g., "UUID id")
	Uses        []string     // contracts this one calls
	Implements  string       // the input-port this service implements (single value)
	DependsOn   []string     // output-ports or use cases injected
	Endpoints   []string     // HTTP method + path (e.g., "POST /users")
	Annotations []string     // additional annotations beyond profile defaults
}
