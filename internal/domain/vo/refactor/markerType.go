package refactor

import "fmt"

// MarkerType is the typed category of a refactor marker. Recognised values
// come from EP03RF-006. "other" buckets unknown user-defined types
// (per .feature scenario "Scanner handles unknown marker type").
// "unparseable" is reserved for comments that match the marker prefix
// but do not contain the required " - " separator.
type MarkerType string

const (
	MarkerTypeExtractMethod MarkerType = "extract-method"
	MarkerTypeRename        MarkerType = "rename"
	MarkerTypeMove          MarkerType = "move"
	MarkerTypeInline        MarkerType = "inline"
	MarkerTypeSimplify      MarkerType = "simplify"
	MarkerTypeOther         MarkerType = "other"
	MarkerTypeUnparseable   MarkerType = "unparseable"
)

// Validate reports whether t is one of the known constants.
func (t MarkerType) Validate() error {
	switch t {
	case MarkerTypeExtractMethod, MarkerTypeRename, MarkerTypeMove,
		MarkerTypeInline, MarkerTypeSimplify, MarkerTypeOther,
		MarkerTypeUnparseable:
		return nil
	}
	return fmt.Errorf("unknown marker type: %q", t)
}

// IsKnownUserType returns true for the explicit RF-006 user-facing types.
// Used by the parser to decide whether to bucket as "other" + warn.
func (t MarkerType) IsKnownUserType() bool {
	switch t {
	case MarkerTypeExtractMethod, MarkerTypeRename, MarkerTypeMove,
		MarkerTypeInline, MarkerTypeSimplify:
		return true
	}
	return false
}
