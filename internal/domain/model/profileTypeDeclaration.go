package model

// ProfileTypeDeclaration is one `types:` entry in profile.yaml. EP04US-001
// shipped the minimal shape (ID, Template, Raw); EP04US-002 extends it with
// the Description and Classification fields. The Raw byte blob is preserved
// verbatim so downstream USes (US-004 target_path, US-007 validate, etc.)
// still see the original YAML node bytes for fields not yet promoted to
// typed members.
//
// Field ordering rationale: ID and Template stay first to keep US-001's
// struct-literal tests valid (they construct the type by name not position,
// but readers expect the ID-first contract). Description is third because
// it is the next-most-prominent "metadata" field. Classification carries
// the load-bearing logic and lands fourth. Raw is last (it was last in
// US-001 too).
type ProfileTypeDeclaration struct {
	// ID is the type identifier (e.g., "service"). Required — a missing id
	// is treated as ErrProfileInvalid by the loader.
	ID string

	// Template is the filename inside templates/ this type references.
	// Empty string means the type does not require a template (e.g.
	// "abstract" or classification-only types). Non-empty values are
	// verified against the loaded Templates map at load time.
	Template string

	// Description is the human-readable purpose of the type. Used by
	// US-007 (profile validate) and US-008 (authoring docs) but not by
	// the classifier engine. Optional — empty string is acceptable.
	Description string

	// Classification is the OR-list of AND-internal rules. A type matches
	// a code element when ANY rule in this slice matches the element.
	// nil/empty slice means the type matches nothing.
	Classification []ClassificationRule

	// Raw holds the full YAML node bytes for this type entry, preserved
	// for downstream USes that have not yet landed their schemas.
	// (Untouched by EP04US-002 — re-marshal of the parsed node bytes.)
	Raw []byte
}
