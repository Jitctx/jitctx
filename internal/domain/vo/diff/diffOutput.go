package diff

// DiffPlanOutput is the output VO for diffuc.UseCase.
//
//   - Feature / Module: echoed from the parsed spec.
//   - Actions: every CREATE/MODIFY/EXTRA, sorted deterministically.
//     Ordering policy:
//     1. CREATE/MODIFY first, by (Layer ASC, ContractName ASC,
//     ActionType ASC where CREATE < MODIFY).
//     2. EXTRA last, by ContractName ASC.
//   - HasChanges: true iff Actions contains at least one CREATE or
//     MODIFY entry. Renderer uses this to decide between the verbatim
//     "No diff detected" line and the full report.
//     EXTRA-only diffs are NOT clean — Gherkin scenario 3 expects the
//     EXTRA entry to appear, so HasChanges considers EXTRA as changes
//     for rendering purposes; the verbatim clean line fires ONLY when
//     Actions is empty.
type DiffPlanOutput struct {
	Feature    string
	Module     string
	Actions    []DiffAction
	HasChanges bool
}
