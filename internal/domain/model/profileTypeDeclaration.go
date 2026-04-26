package model

// ProfileTypeDeclaration is the minimal shape this US requires from each
// `types:` entry: enough to enforce template-file existence at load time
// (EP04RF-001 third scenario), no more. EP04US-002 expands the struct
// with classification fields, target_path, has_test, etc.
type ProfileTypeDeclaration struct {
	// ID is the type identifier (e.g., "service"). Required;
	// a missing id is treated as ErrProfileInvalid by the loader.
	ID string

	// Template is the filename inside templates/ this type references.
	// Empty string means the type does not require a template (e.g.
	// future "abstract" type declarations); non-empty values are
	// verified against the loaded Templates map at load time.
	Template string

	// Raw holds the full YAML node bytes for this type entry, preserved
	// for downstream USes that have not yet landed their schemas.
	Raw []byte
}
