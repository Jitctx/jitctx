package model

// FeatureSpec is the parsed representation of a markdown feature specification.
// It contains the feature identity and an ordered list of contracts declared
// in the spec file. No YAML/JSON tags — serialization is an infra concern.
//
// Package is OPTIONAL at parse time so existing US-001 / US-002 / US-003 /
// US-004 specs continue to parse unchanged. Consumers that REQUIRE a Package
// (currently only the scaffold use case) raise ErrSpecMissingPackage when it
// is empty.
type FeatureSpec struct {
	Feature   string         // kebab-case feature name from "# Feature:"
	Module    string         // module identifier from "Module:"
	Package   string         // OPTIONAL Java package from "Package:"; "" when absent
	Contracts []SpecContract // ordered as declared in the spec
}
