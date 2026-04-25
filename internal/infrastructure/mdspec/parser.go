package mdspec

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/port/spec"
)

// Compile-time check: Parser must implement ParseSpecPort.
var _ spec.ParseSpecPort = (*Parser)(nil)

// Parser is a stateless markdown spec parser. It implements spec.ParseSpecPort.
type Parser struct{}

// New returns a new Parser.
func New() *Parser {
	return &Parser{}
}

// parse states
type parseState int

const (
	stateExpectingFeature parseState = iota
	stateInFeature
	stateInContract
)

// recognizedFields is the set of field names that are valid inside a contract block.
var recognizedFields = map[string]bool{
	"Type":        true,
	"Methods":     true,
	"Fields":      true,
	"Uses":        true,
	"Implements":  true,
	"DependsOn":   true,
	"Endpoints":   true,
	"Annotations": true,
}

// knownContractTypes maps spec type strings to model.ContractType constants.
var knownContractTypes = map[string]model.ContractType{
	"input-port":     model.ContractInputPort,
	"output-port":    model.ContractOutputPort,
	"entity":         model.ContractEntity,
	"aggregate-root": model.ContractAggregate,
	"service":        model.ContractService,
	"rest-adapter":   model.ContractRestAdapter,
	"jpa-adapter":    model.ContractJPAAdapter,
}

// contractEntry tracks a contract being built and the line it was opened on.
type contractEntry struct {
	contract model.SpecContract
	openedAt int
}

// ParseSpec parses markdown content into a FeatureSpec using a state machine.
// It returns the parsed spec, any non-fatal warnings, and a fatal error if parsing fails.
func (p *Parser) ParseSpec(ctx context.Context, content string) (model.FeatureSpec, []domerr.SpecParseWarning, error) {
	if err := ctx.Err(); err != nil {
		return model.FeatureSpec{}, nil, err
	}

	var (
		spec     model.FeatureSpec
		warnings []domerr.SpecParseWarning
		state    = stateExpectingFeature

		// contract tracking
		current   *contractEntry
		seen      = make(map[string]int) // contract name → line number of first occurrence
		contracts []model.SpecContract

		// multiline list mode
		activeListField string

		moduleFound bool
	)

	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		switch state {
		case stateExpectingFeature:
			if trimmed == "" {
				continue
			}
			name, ok := parseFeatureHeader(trimmed)
			if !ok {
				return model.FeatureSpec{}, nil, &domerr.SpecParseError{
					Line:    lineNum,
					Message: fmt.Sprintf("expected '# Feature: <name>', got: %q", trimmed),
				}
			}
			spec.Feature = name
			state = stateInFeature

		case stateInFeature:
			if trimmed == "" {
				continue
			}
			if mod, ok := parseFieldValue("Module", trimmed); ok {
				spec.Module = mod
				moduleFound = true
				continue
			}
			if pkg, ok := parseFieldValue("Package", trimmed); ok {
				if !moduleFound {
					// Package before Module: emit warning but accept.
					warnings = append(warnings, domerr.SpecParseWarning{
						Line:    lineNum,
						Message: "found 'Package:' before 'Module:'",
					})
				}
				spec.Package = pkg
				continue
			}
			if name, ok := parseContractHeader(trimmed); ok {
				if !moduleFound {
					return model.FeatureSpec{}, nil, &domerr.SpecParseError{
						Line:    lineNum,
						Message: "missing 'Module:' before first contract",
					}
				}
				current = &contractEntry{
					contract: model.SpecContract{Name: name},
					openedAt: lineNum,
				}
				seen[name] = lineNum
				state = stateInContract
				activeListField = ""
				continue
			}
			// Any other non-blank line before the first contract → warning
			warnings = append(warnings, domerr.SpecParseWarning{
				Line:    lineNum,
				Message: fmt.Sprintf("unexpected line in feature header: %q", trimmed),
			})

		case stateInContract:
			// Blank line: close any open multiline list
			if trimmed == "" {
				activeListField = ""
				continue
			}

			// New contract header
			if name, ok := parseContractHeader(trimmed); ok {
				// Close any open multiline list
				activeListField = ""
				// Finalize current contract into the list
				if current != nil {
					contracts = append(contracts, current.contract)
				}
				// Check for duplicate
				if firstLine, dup := seen[name]; dup {
					return model.FeatureSpec{}, nil, &domerr.DuplicateContractError{
						Name:      name,
						FirstLine: firstLine,
						DupeLine:  lineNum,
					}
				}
				current = &contractEntry{
					contract: model.SpecContract{Name: name},
					openedAt: lineNum,
				}
				seen[name] = lineNum
				continue
			}

			// List item (only valid when a multiline list field is active)
			if item, ok := parseListItem(trimmed); ok {
				if activeListField != "" && current != nil {
					appendFieldValue(&current.contract, activeListField, item)
				} else {
					warnings = append(warnings, domerr.SpecParseWarning{
						Line:    lineNum,
						Message: fmt.Sprintf("list item outside of a multiline field: %q", trimmed),
					})
				}
				continue
			}

			// Field line: FieldName: value  OR  FieldName:  (empty → multiline)
			if fieldName, value, ok := parseFieldLine(trimmed); ok {
				if !recognizedFields[fieldName] {
					warnings = append(warnings, domerr.SpecParseWarning{
						Line:    lineNum,
						Message: fmt.Sprintf("unknown field %q", fieldName),
					})
					activeListField = ""
					continue
				}
				// Close any previous multiline field
				activeListField = ""
				if value == "" {
					// Opens multiline list mode
					activeListField = fieldName
				} else {
					// Inline value or inline list
					if current != nil {
						setFieldValue(&current.contract, fieldName, value)
					}
				}
				continue
			}

			// Anything else → warning
			warnings = append(warnings, domerr.SpecParseWarning{
				Line:    lineNum,
				Message: fmt.Sprintf("unrecognized line: %q", trimmed),
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return model.FeatureSpec{}, nil, fmt.Errorf("read spec content: %w", err)
	}

	// EOF finalization: close any open multiline list (nothing to do — already tracked)
	_ = activeListField

	// Finalize the last contract
	if current != nil {
		contracts = append(contracts, current.contract)
	}

	// Validate: module must be present
	if state == stateExpectingFeature {
		return model.FeatureSpec{}, nil, &domerr.SpecParseError{
			Line:    lineNum,
			Message: "missing '# Feature:' header",
		}
	}
	if !moduleFound {
		return model.FeatureSpec{}, nil, &domerr.SpecParseError{
			Line:    lineNum,
			Message: "missing 'Module:' field",
		}
	}

	// Validate: every contract must have a Type
	for _, c := range contracts {
		if c.Type == "" {
			firstLine := seen[c.Name]
			return model.FeatureSpec{}, nil, &domerr.SpecParseError{
				Line:    firstLine,
				Message: fmt.Sprintf("contract %q is missing required 'Type:' field", c.Name),
			}
		}
	}

	spec.Contracts = contracts
	return spec, warnings, nil
}

// parseFeatureHeader parses "# Feature: <name>" and returns (name, true) on success.
func parseFeatureHeader(line string) (string, bool) {
	const prefix = "# Feature:"
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	name := strings.TrimSpace(line[len(prefix):])
	if name == "" {
		return "", false
	}
	return name, true
}

// parseContractHeader parses "## Contract: <name>" and returns (name, true) on success.
func parseContractHeader(line string) (string, bool) {
	const prefix = "## Contract:"
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	name := strings.TrimSpace(line[len(prefix):])
	if name == "" {
		return "", false
	}
	return name, true
}

// parseFieldValue parses "<fieldName>: <value>" and returns (value, true) on success.
func parseFieldValue(fieldName, line string) (string, bool) {
	prefix := fieldName + ":"
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	value := strings.TrimSpace(line[len(prefix):])
	return value, true
}

// parseFieldLine parses any "<FieldName>: <value>" line (where value may be empty).
// Returns (fieldName, value, true) on success. The colon must be present.
func parseFieldLine(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return "", "", false
	}
	fieldName := strings.TrimSpace(line[:idx])
	// Field names should not contain spaces
	if strings.ContainsAny(fieldName, " \t") {
		return "", "", false
	}
	value := strings.TrimSpace(line[idx+1:])
	return fieldName, value, true
}

// parseListItem parses "- <item>" and returns (item, true) on success.
func parseListItem(line string) (string, bool) {
	if !strings.HasPrefix(line, "- ") {
		return "", false
	}
	item := strings.TrimSpace(line[2:])
	return item, true
}

// setFieldValue assigns an inline value to the appropriate field of a SpecContract.
// For list fields with a comma-separated value, it splits on comma.
// For single-value fields (Implements, Type), it assigns directly.
func setFieldValue(c *model.SpecContract, fieldName, value string) {
	switch fieldName {
	case "Type":
		c.Type = resolveContractType(value)
	case "Implements":
		c.Implements = value
	case "Methods":
		c.Methods = splitList(value)
	case "Fields":
		c.Fields = splitList(value)
	case "Uses":
		c.Uses = splitList(value)
	case "DependsOn":
		c.DependsOn = splitList(value)
	case "Endpoints":
		c.Endpoints = splitList(value)
	case "Annotations":
		c.Annotations = splitList(value)
	}
}

// appendFieldValue appends a single item to the appropriate list field of a SpecContract.
func appendFieldValue(c *model.SpecContract, fieldName, item string) {
	switch fieldName {
	case "Methods":
		c.Methods = append(c.Methods, item)
	case "Fields":
		c.Fields = append(c.Fields, item)
	case "Uses":
		c.Uses = append(c.Uses, item)
	case "DependsOn":
		c.DependsOn = append(c.DependsOn, item)
	case "Endpoints":
		c.Endpoints = append(c.Endpoints, item)
	case "Annotations":
		c.Annotations = append(c.Annotations, item)
	case "Type":
		// Type is a single-value field; multiline doesn't apply, ignore
	case "Implements":
		// Implements is a single-value field; multiline doesn't apply, ignore
	}
}

// resolveContractType maps a spec type string to a model.ContractType.
// If the type is unknown, it is stored as-is (caller may add a warning separately).
func resolveContractType(typeStr string) model.ContractType {
	if ct, ok := knownContractTypes[typeStr]; ok {
		return ct
	}
	// Unknown type — stored as-is per Section 8 Q1 resolution.
	return model.ContractType(typeStr)
}

// splitList splits a comma-separated string into trimmed, non-empty elements.
func splitList(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			result = append(result, t)
		}
	}
	return result
}
