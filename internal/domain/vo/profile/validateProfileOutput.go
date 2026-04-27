package profile

// ValidationIssue describes a single problem found during validation.
// Code is a short, stable, machine-friendly tag (e.g.,
// "missing_name", "duplicate_type_id", "unknown_classification_field",
// "missing_template", "yaml_missing", "language_unsupported"). Message
// is the human-readable string written to stderr — its format matches
// the literals pinned by the .feature scenarios.
type ValidationIssue struct {
	Code    string
	Message string
}

// ValidateProfileOutput is the success-path result of
// profilevalidateuc.UseCase.Execute. Fatals are present only when
// Execute also returns *domerr.ProfileValidationError; on the success
// path Errors is always empty and Warnings may be non-empty (the
// .feature explicitly allows exit 0 with warnings on stderr).
type ValidateProfileOutput struct {
	Path     string
	Errors   []ValidationIssue
	Warnings []ValidationIssue
}
