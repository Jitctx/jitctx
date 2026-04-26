package service

import (
	"sort"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// AuditRuleFilter removes rules whose ID appears in the disabled list and
// reports any disabled IDs that did NOT match a real rule (used by the
// presentation layer to emit stderr warnings per EP03US-005 scenario 2).
//
// Pure: no I/O, no goroutines, no context. Deterministic per RNF-003 —
// the unknown list is sorted ascending and deduped.
type AuditRuleFilter struct{}

// NewAuditRuleFilter returns the singleton-like filter.
func NewAuditRuleFilter() *AuditRuleFilter { return &AuditRuleFilter{} }

// Filter returns:
//   - kept: the input rules with every rule whose ID is in disabled removed.
//     Order is preserved (the use case re-sorts violations downstream, so
//     rule order does not affect determinism of the report).
//   - unknown: the sorted, deduped subset of disabled IDs that did NOT
//     match any rule in rules. Empty disabled list yields empty unknown.
//
// Both return values are non-nil; empty results are []model.AuditRule{}
// and []string{} respectively to keep downstream nil-checks unnecessary.
func (f *AuditRuleFilter) Filter(
	rules []model.AuditRule, disabled []string,
) (kept []model.AuditRule, unknown []string) {
	// Build a set of disabled IDs (dedupes input).
	disabledSet := make(map[string]struct{}, len(disabled))
	for _, id := range disabled {
		if id == "" {
			continue
		}
		disabledSet[id] = struct{}{}
	}

	kept = make([]model.AuditRule, 0, len(rules))
	matched := make(map[string]struct{}, len(disabledSet))
	for _, r := range rules {
		if _, drop := disabledSet[r.ID]; drop {
			matched[r.ID] = struct{}{}
			continue
		}
		kept = append(kept, r)
	}

	unknown = make([]string, 0, len(disabledSet))
	for id := range disabledSet {
		if _, ok := matched[id]; !ok {
			unknown = append(unknown, id)
		}
	}
	sort.Strings(unknown)
	return kept, unknown
}
