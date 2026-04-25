package model

// FeatureSpec is the parsed representation of a markdown feature specification.
// It contains the feature identity and an ordered list of contracts declared
// in the spec file. No YAML/JSON tags — serialization is an infra concern.
type FeatureSpec struct {
	Feature   string         // kebab-case feature name from "# Feature:"
	Module    string         // module identifier from "Module:"
	Contracts []SpecContract // ordered as declared in the spec
}
