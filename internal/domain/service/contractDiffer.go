package service

import (
	"sort"
	"strings"

	"github.com/jitctx/jitctx/internal/domain/model"
	diffvo "github.com/jitctx/jitctx/internal/domain/vo/diff"
)

// ContractDiffer computes the set of DiffActions between the desired
// state (a parsed FeatureSpec) and the current state (a flat list of
// model.Contract from the manifest). Pure, stateless, deterministic.
//
// Matching key is Contract.Name (PascalCase). Per discovery finding
// #11 the matcher is name-keyed across modules; the manifest does not
// allow name collisions across modules in practice (EP-01 invariant).
type ContractDiffer struct {
	normalizer SignatureNormalizer
}

// NewContractDiffer wires the differ with a SignatureNormalizer.
func NewContractDiffer(n SignatureNormalizer) ContractDiffer {
	return ContractDiffer{normalizer: n}
}

// Diff returns the unsorted slice of DiffActions:
//
//   - For each spec contract not in the manifest index → CREATE.
//   - For each spec contract present in the manifest index, compute
//     the symmetric difference of the normalised method-signature sets:
//   - empty diff → no action emitted.
//   - non-empty diff → MODIFY with AddedMethods (in spec, not in
//     manifest) and RemovedMethods (in manifest, not in spec); each
//     sorted alphabetically. Per Gherkin scenario 2 the lists carry
//     method NAMES (extracted from the signature), not full
//     signatures. Method-name extraction reuses the same "last
//     whitespace before '('" rule as MethodSignatureParser.
//   - For each manifest contract whose Name is NOT in the spec index
//     → EXTRA (severity INFO).
//
// CREATE / MODIFY layers are filled by the use case after a separate
// DependencyLayerer call; the differ leaves Layer = 0 as a default
// placeholder for now, and EXTRA actions are emitted with Layer = -1.
func (d ContractDiffer) Diff(
	spec []model.SpecContract,
	manifest []model.Contract,
) []diffvo.DiffAction {
	// Build a name-keyed index of manifest contracts.
	manifestIndex := make(map[string]model.Contract, len(manifest))
	for _, c := range manifest {
		// Keep first occurrence on duplicate name (defensive per §5.1 finding).
		if _, exists := manifestIndex[c.Name]; !exists {
			manifestIndex[c.Name] = c
		}
	}

	// Build a name-keyed index of spec contracts for EXTRA detection.
	specIndex := make(map[string]bool, len(spec))
	for _, c := range spec {
		specIndex[c.Name] = true
	}

	var actions []diffvo.DiffAction

	// Iterate spec contracts: CREATE or MODIFY.
	for _, sc := range spec {
		mc, found := manifestIndex[sc.Name]
		if !found {
			// Contract is in spec but not in manifest → CREATE.
			actions = append(actions, diffvo.DiffAction{
				Type:         diffvo.DiffActionCreate,
				ContractName: sc.Name,
				ContractType: string(sc.Type),
				Severity:     diffvo.DiffSeverityError,
				Layer:        0, // placeholder; use case will fill via layerer
			})
			continue
		}

		// Contract exists in both spec and manifest — compute method delta.
		added, removed := d.methodDelta(sc.Methods, mc.Methods)
		if len(added) == 0 && len(removed) == 0 {
			// No difference — emit nothing.
			continue
		}

		actions = append(actions, diffvo.DiffAction{
			Type:           diffvo.DiffActionModify,
			ContractName:   sc.Name,
			ContractType:   string(sc.Type),
			Severity:       diffvo.DiffSeverityWarning,
			Layer:          0, // placeholder; use case will fill via layerer
			AddedMethods:   added,
			RemovedMethods: removed,
		})
	}

	// Iterate manifest contracts not present in spec → EXTRA.
	for _, mc := range manifest {
		if !specIndex[mc.Name] {
			actions = append(actions, diffvo.DiffAction{
				Type:          diffvo.DiffActionExtra,
				ContractName:  mc.Name,
				ContractTypes: append([]string(nil), mc.Types...),
				Severity:      diffvo.DiffSeverityInfo,
				Layer:         -1,
			})
		}
	}

	return actions
}

// methodDelta computes the symmetric method difference between spec methods
// and manifest methods using normalised signatures as the comparison key.
// Returns (added, removed) where:
//   - added: method NAMES in spec but not in manifest.
//   - removed: method NAMES in manifest but not in spec.
//
// Both slices are sorted alphabetically.
func (d ContractDiffer) methodDelta(specMethods []string, manifestMethods []model.Method) (added, removed []string) {
	// Build normalised-signature → method-name map for spec.
	specNorm := make(map[string]string, len(specMethods))
	for _, sig := range specMethods {
		norm := d.normalizer.Normalize(sig)
		name := extractMethodName(sig)
		specNorm[norm] = name
	}

	// Build normalised-signature → method-name map for manifest.
	manifestNorm := make(map[string]string, len(manifestMethods))
	for _, m := range manifestMethods {
		norm := d.normalizer.Normalize(m.Signature)
		name := extractMethodName(m.Signature)
		manifestNorm[norm] = name
	}

	// Added: in spec but not in manifest (keyed by normalised signature).
	for norm, name := range specNorm {
		if _, found := manifestNorm[norm]; !found {
			added = append(added, name)
		}
	}

	// Removed: in manifest but not in spec (keyed by normalised signature).
	for norm, name := range manifestNorm {
		if _, found := specNorm[norm]; !found {
			removed = append(removed, name)
		}
	}

	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

// extractMethodName extracts the method name from a Java signature using
// the "last whitespace before '('" rule (same algorithm as MethodSignatureParser).
// Returns the raw trimmed signature as a fallback when the rule cannot apply.
func extractMethodName(sig string) string {
	s := strings.TrimSpace(sig)
	s = strings.TrimRight(s, ";")
	s = strings.TrimSpace(s)

	parenIdx := strings.IndexByte(s, '(')
	if parenIdx < 0 {
		return s
	}

	left := strings.TrimSpace(s[:parenIdx])
	lastSpace := strings.LastIndexAny(left, " \t")
	if lastSpace < 0 {
		// No space found — the entire left side is the name (unlikely for Java but safe).
		return left
	}
	return strings.TrimSpace(left[lastSpace+1:])
}
