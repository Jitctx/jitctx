// Package profilevalidateuc implements the "jitctx profile validate <path>"
// use case. EP04US-007.
//
// yaml.v3 import exception: this use case walks raw yaml.Node trees to detect
// unknown classification field keys in types[].classification[] entries. The
// canonical key set is part of the EP-04 declarative-types schema and lives
// naturally here rather than in an infrastructure adapter. Discovery accepted
// the yaml.v3 import in this single file as the simpler design (see plan
// Section 8 Q2 resolution).
package profilevalidateuc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	profileport "github.com/jitctx/jitctx/internal/domain/port/profile"
	profilevalidateucport "github.com/jitctx/jitctx/internal/domain/usecase/profilevalidateuc"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// knownClassificationKeys is the canonical set of field keys allowed inside a
// types[].classification[] mapping entry. Sourced from bundleDto.go:89-95.
var knownClassificationKeys = map[string]struct{}{
	"kind":            {},
	"implements_all":  {},
	"implements_none": {},
	"has_annotation":  {},
	"path_contains":   {},
}

// bundleDTO is a minimal unexported DTO used only to run the name-empty and
// duplicate-type-id checks that LoadBundle is too lenient about today.
type bundleDTO struct {
	Name  string `yaml:"name"`
	Types []struct {
		ID string `yaml:"id"`
	} `yaml:"types"`
}

// Impl satisfies profilevalidateucport.UseCase. EP04US-007.
type Impl struct {
	loader profileport.LoadProfileBundlePort
	logger *slog.Logger
}

// New constructs an Impl. When logger is nil, slog.Default() is used.
func New(loader profileport.LoadProfileBundlePort, logger *slog.Logger) *Impl {
	if logger == nil {
		logger = slog.Default()
	}
	return &Impl{loader: loader, logger: logger}
}

// Execute validates the profile directory at in.Path.
func (u *Impl) Execute(
	ctx context.Context,
	in profilevo.ValidateProfileInput,
) (profilevo.ValidateProfileOutput, error) {
	if err := ctx.Err(); err != nil {
		return profilevo.ValidateProfileOutput{}, err
	}

	out := profilevo.ValidateProfileOutput{Path: in.Path}

	// Step 1 — path-exists guard (EP04RF-013 exception: immediate exit 1).
	abs, err := filepath.Abs(in.Path)
	if err != nil {
		return out, fmt.Errorf("validate profile: resolve path %q: %w", in.Path, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		msg := fmt.Sprintf("profile path %q does not exist", in.Path)
		out.Errors = append(out.Errors, profilevo.ValidationIssue{
			Code:    "path_not_found",
			Message: msg,
		})
		return out, &domerr.ProfileValidationError{
			Path:   abs,
			Errors: []string{msg},
		}
	}
	if !info.IsDir() {
		msg := fmt.Sprintf("profile path %q is not a directory", in.Path)
		out.Errors = append(out.Errors, profilevo.ValidationIssue{Code: "not_a_directory", Message: msg})
		return out, &domerr.ProfileValidationError{
			Path:   abs,
			Errors: []string{msg},
		}
	}

	// Step 2 — yaml.Node walk for unknown-classification-field warnings.
	// Done before LoadBundle so warnings are surfaced even when LoadBundle
	// fails for a separate reason (e.g., missing template).
	for _, w := range scanClassificationKeyTypos(abs) {
		out.Warnings = append(out.Warnings, profilevo.ValidationIssue{
			Code:    "unknown_classification_field",
			Message: w,
		})
	}

	// Step 3 — delegate to LoadBundle for structural fatals.
	bundle, loadErr := u.loader.LoadBundle(ctx, profilevo.LoadProfileBundleInput{Dir: abs})
	if loadErr != nil {
		out.Errors = append(out.Errors, profilevo.ValidationIssue{
			Code:    classifyLoadErr(loadErr),
			Message: humanizeLoadErr(loadErr),
		})
	}

	// Step 4a — explicit "missing name" check via a lightweight YAML re-decode.
	// Only attempted when profile.yaml was readable (i.e., LoadBundle did not
	// fail because the file was missing or undecodable).
	if loadErr == nil || isAfterNameDecode(loadErr) {
		if rawName, _ := readRawNameField(abs); rawName == "" {
			out.Errors = append(out.Errors, profilevo.ValidationIssue{
				Code:    "missing_name",
				Message: "missing required field: name",
			})
		}
	}

	// Step 4b — duplicate type-id detection (independent of LoadBundle).
	for _, dup := range scanDuplicateTypeIDs(abs) {
		out.Errors = append(out.Errors, profilevo.ValidationIssue{
			Code:    "duplicate_type_id",
			Message: fmt.Sprintf("duplicate type id: %s", dup),
		})
	}

	// Step 5 — aggregate.
	_ = bundle
	if len(out.Errors) == 0 {
		return out, nil
	}
	errMsgs := make([]string, 0, len(out.Errors))
	for _, e := range out.Errors {
		errMsgs = append(errMsgs, e.Message)
	}
	sort.Strings(errMsgs) // deterministic order for stderr/test asserts
	warnMsgs := make([]string, 0, len(out.Warnings))
	for _, w := range out.Warnings {
		warnMsgs = append(warnMsgs, w.Message)
	}
	return out, &domerr.ProfileValidationError{
		Path:     abs,
		Errors:   errMsgs,
		Warnings: warnMsgs,
	}
}

// scanClassificationKeyTypos opens <dir>/profile.yaml, decodes it into a
// yaml.Node tree, and walks types[].classification[] mapping nodes comparing
// each key against knownClassificationKeys. Returns formatted warning strings
// of the form "unknown classification field 'KEY'" (single-quoted key).
func scanClassificationKeyTypos(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, "profile.yaml"))
	if err != nil {
		return nil
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil
	}
	// root is a Document node whose first child is the top-level mapping.
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil
	}
	topMap := root.Content[0]
	if topMap.Kind != yaml.MappingNode {
		return nil
	}

	typesSeq := mappingValue(topMap, "types")
	if typesSeq == nil || typesSeq.Kind != yaml.SequenceNode {
		return nil
	}

	var warnings []string
	seen := map[string]bool{}
	for _, typeEntry := range typesSeq.Content {
		if typeEntry.Kind != yaml.MappingNode {
			continue
		}
		classSeq := mappingValue(typeEntry, "classification")
		if classSeq == nil || classSeq.Kind != yaml.SequenceNode {
			continue
		}
		for _, classEntry := range classSeq.Content {
			if classEntry.Kind != yaml.MappingNode {
				continue
			}
			// Walk key-value pairs in the mapping.
			for i := 0; i+1 < len(classEntry.Content); i += 2 {
				key := classEntry.Content[i].Value
				if _, ok := knownClassificationKeys[key]; !ok {
					warnKey := fmt.Sprintf("unknown classification field '%s'", key)
					if !seen[warnKey] {
						seen[warnKey] = true
						warnings = append(warnings, warnKey)
					}
				}
			}
		}
	}
	return warnings
}

// scanDuplicateTypeIDs walks types[].id fields in document order and returns
// the IDs that appear more than once (one entry per duplicate occurrence).
func scanDuplicateTypeIDs(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, "profile.yaml"))
	if err != nil {
		return nil
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil
	}
	topMap := root.Content[0]
	if topMap.Kind != yaml.MappingNode {
		return nil
	}
	typesSeq := mappingValue(topMap, "types")
	if typesSeq == nil || typesSeq.Kind != yaml.SequenceNode {
		return nil
	}

	seen := map[string]bool{}
	var duplicates []string
	for _, typeEntry := range typesSeq.Content {
		if typeEntry.Kind != yaml.MappingNode {
			continue
		}
		idNode := mappingValue(typeEntry, "id")
		if idNode == nil {
			continue
		}
		id := idNode.Value
		if seen[id] {
			duplicates = append(duplicates, id)
		} else {
			seen[id] = true
		}
	}
	return duplicates
}

// readRawNameField decodes only the top-level "name:" scalar from profile.yaml.
func readRawNameField(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "profile.yaml"))
	if err != nil {
		return "", err
	}
	var dto bundleDTO
	if err := yaml.Unmarshal(data, &dto); err != nil {
		return "", err
	}
	return dto.Name, nil
}

// classifyLoadErr returns a stable machine-friendly code for the given
// LoadBundle error.
func classifyLoadErr(err error) string {
	var tmpl *domerr.TemplateMissingError
	switch {
	case errors.Is(err, domerr.ErrProfileYamlMissing):
		return "yaml_missing"
	case errors.As(err, &tmpl):
		return "template_missing"
	case errors.Is(err, domerr.ErrLanguageUnsupported):
		return "language_unsupported"
	default:
		return "profile_invalid"
	}
}

// humanizeLoadErr produces a user-friendly message for a LoadBundle error.
// For *TemplateMissingError the verbatim Error() already names the template
// file (satisfying the .feature scenario 3 criterion).
func humanizeLoadErr(err error) string {
	if errors.Is(err, domerr.ErrProfileYamlMissing) {
		return "profile.yaml not found"
	}
	return err.Error()
}

// isAfterNameDecode returns true when the LoadBundle failure happened after
// profile.yaml was decoded (e.g., missing template, language unsupported) so
// the name-empty check can still be attempted on the raw YAML.
func isAfterNameDecode(err error) bool {
	if errors.Is(err, domerr.ErrProfileYamlMissing) {
		return false
	}
	return true
}

// mappingValue returns the value node for the given key in a MappingNode, or
// nil when the key is not present.
func mappingValue(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// compile-time assertion that *Impl satisfies the domain UseCase interface.
var _ profilevalidateucport.UseCase = (*Impl)(nil)
