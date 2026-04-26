package format_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/cli/format"
	diffvo "github.com/jitctx/jitctx/internal/domain/vo/diff"
)

// TestWriteDiffPlanReport_CleanDiff asserts that an empty Actions slice
// emits the verbatim "No diff detected" line and nothing else.
func TestWriteDiffPlanReport_CleanDiff(t *testing.T) {
	t.Parallel()

	out := diffvo.DiffPlanOutput{
		Feature:    "some-feature",
		Actions:    nil,
		HasChanges: false,
	}

	var buf bytes.Buffer
	err := format.WriteDiffPlanReport(&buf, out)
	require.NoError(t, err)

	got := buf.String()

	// Verbatim clean line assertion (Gherkin scenario 4 / §7 note 7).
	require.Contains(t, got, "No diff detected. Current state matches spec.")

	// Must be exactly that line with a trailing newline and nothing more.
	require.Equal(t, "No diff detected. Current state matches spec.\n", got)
}

// TestWriteDiffPlanReport_SingleCreate asserts that a single CREATE action
// produces the HTML comment header, ## heading, ### Layer 0 heading, and the
// correct action bullet.
func TestWriteDiffPlanReport_SingleCreate(t *testing.T) {
	t.Parallel()

	out := diffvo.DiffPlanOutput{
		Feature: "update-user-flow",
		Actions: []diffvo.DiffAction{
			{
				Type:         diffvo.DiffActionCreate,
				ContractName: "ChangeUserStatusUseCase",
				ContractType: "input-port",
				Severity:     diffvo.DiffSeverityError,
				Layer:        0,
			},
		},
		HasChanges: true,
	}

	var buf bytes.Buffer
	err := format.WriteDiffPlanReport(&buf, out)
	require.NoError(t, err)

	got := buf.String()

	// HTML comment with feature name.
	require.Contains(t, got, "<!-- jitctx plan --diff | feature: update-user-flow -->")

	// Top-level heading.
	require.Contains(t, got, "## Diff Plan")

	// Layer 0 sub-heading.
	require.Contains(t, got, "### Layer 0")

	// Action bullet with correct badge, type, name, and contract type.
	require.Contains(t, got, "- 🔴 ERROR CREATE: ChangeUserStatusUseCase (input-port)")

	// No Extras section.
	require.NotContains(t, got, "Extras")

	// No "No diff detected" line.
	require.NotContains(t, got, "No diff detected")
}

// TestWriteDiffPlanReport_SingleModifyWithAddedAndRemoved asserts that a
// MODIFY action renders the parent bullet plus indented added/removed sub-bullets.
func TestWriteDiffPlanReport_SingleModifyWithAddedAndRemoved(t *testing.T) {
	t.Parallel()

	out := diffvo.DiffPlanOutput{
		Feature: "user-repo-update",
		Actions: []diffvo.DiffAction{
			{
				Type:           diffvo.DiffActionModify,
				ContractName:   "UserRepository",
				ContractType:   "output-port",
				Severity:       diffvo.DiffSeverityWarning,
				Layer:          0,
				AddedMethods:   []string{"save"},
				RemovedMethods: []string{"persist"},
			},
		},
		HasChanges: true,
	}

	var buf bytes.Buffer
	err := format.WriteDiffPlanReport(&buf, out)
	require.NoError(t, err)

	got := buf.String()

	// Parent bullet — MODIFY does NOT include contract type.
	require.Contains(t, got, "- 🟡 WARNING MODIFY: UserRepository")

	// Indented sub-bullets.
	require.Contains(t, got, "  - added: save")
	require.Contains(t, got, "  - removed: persist")

	// No Extras section.
	require.NotContains(t, got, "Extras")
}

// TestWriteDiffPlanReport_ModifyOmitsEmptySubBullets verifies that when
// AddedMethods is empty its sub-bullet is omitted, and vice-versa.
func TestWriteDiffPlanReport_ModifyOmitsEmptySubBullets(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		addedMethods   []string
		removedMethods []string
		wantAdded      bool
		wantRemoved    bool
	}{
		{
			name:           "added-only",
			addedMethods:   []string{"save"},
			removedMethods: nil,
			wantAdded:      true,
			wantRemoved:    false,
		},
		{
			name:           "removed-only",
			addedMethods:   nil,
			removedMethods: []string{"persist"},
			wantAdded:      false,
			wantRemoved:    true,
		},
		{
			name:           "both-present",
			addedMethods:   []string{"save"},
			removedMethods: []string{"persist"},
			wantAdded:      true,
			wantRemoved:    true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out := diffvo.DiffPlanOutput{
				Actions: []diffvo.DiffAction{
					{
						Type:           diffvo.DiffActionModify,
						ContractName:   "Repo",
						ContractType:   "output-port",
						Severity:       diffvo.DiffSeverityWarning,
						Layer:          0,
						AddedMethods:   tc.addedMethods,
						RemovedMethods: tc.removedMethods,
					},
				},
				HasChanges: true,
			}

			var buf bytes.Buffer
			err := format.WriteDiffPlanReport(&buf, out)
			require.NoError(t, err)

			got := buf.String()

			if tc.wantAdded {
				require.Contains(t, got, "  - added:")
			} else {
				require.NotContains(t, got, "  - added:")
			}

			if tc.wantRemoved {
				require.Contains(t, got, "  - removed:")
			} else {
				require.NotContains(t, got, "  - removed:")
			}
		})
	}
}

// TestWriteDiffPlanReport_ExtraOnly asserts that EXTRA-only output emits
// only the Extras section — no layer headings.
func TestWriteDiffPlanReport_ExtraOnly(t *testing.T) {
	t.Parallel()

	out := diffvo.DiffPlanOutput{
		Feature: "legacy-clean",
		Actions: []diffvo.DiffAction{
			{
				Type:         diffvo.DiffActionExtra,
				ContractName: "DeprecatedHelper",
				ContractType: "service",
				Severity:     diffvo.DiffSeverityInfo,
				Layer:        -1,
			},
		},
		HasChanges: true,
	}

	var buf bytes.Buffer
	err := format.WriteDiffPlanReport(&buf, out)
	require.NoError(t, err)

	got := buf.String()

	// Extras section heading.
	require.Contains(t, got, "### Extras (manifest contracts not in spec)")

	// Correct EXTRA bullet.
	require.Contains(t, got, "- 🔵 INFO EXTRA: DeprecatedHelper (service)")

	// No layer headings at all.
	require.NotContains(t, got, "### Layer")

	// No clean-diff line.
	require.NotContains(t, got, "No diff detected")
}

// TestWriteDiffPlanReport_MixedAcrossLayers asserts that Layer 0, Layer 1,
// and the Extras section all appear in the correct order.
func TestWriteDiffPlanReport_MixedAcrossLayers(t *testing.T) {
	t.Parallel()

	out := diffvo.DiffPlanOutput{
		Feature: "multi-layer",
		Actions: []diffvo.DiffAction{
			// Layer 0 — CREATE
			{
				Type:         diffvo.DiffActionCreate,
				ContractName: "ContractB",
				ContractType: "service",
				Severity:     diffvo.DiffSeverityError,
				Layer:        0,
			},
			// Layer 1 — MODIFY
			{
				Type:           diffvo.DiffActionModify,
				ContractName:   "ContractA",
				ContractType:   "input-port",
				Severity:       diffvo.DiffSeverityWarning,
				Layer:          1,
				AddedMethods:   []string{"execute"},
				RemovedMethods: nil,
			},
			// EXTRA
			{
				Type:         diffvo.DiffActionExtra,
				ContractName: "OldHelper",
				ContractType: "service",
				Severity:     diffvo.DiffSeverityInfo,
				Layer:        -1,
			},
		},
		HasChanges: true,
	}

	var buf bytes.Buffer
	err := format.WriteDiffPlanReport(&buf, out)
	require.NoError(t, err)

	got := buf.String()

	// All three section headings present.
	require.Contains(t, got, "### Layer 0")
	require.Contains(t, got, "### Layer 1")
	require.Contains(t, got, "### Extras (manifest contracts not in spec)")

	// Correct ordering: Layer 0 before Layer 1 before Extras.
	layer0Pos := strings.Index(got, "### Layer 0")
	layer1Pos := strings.Index(got, "### Layer 1")
	extrasPos := strings.Index(got, "### Extras")
	require.Less(t, layer0Pos, layer1Pos, "Layer 0 should appear before Layer 1")
	require.Less(t, layer1Pos, extrasPos, "Layer 1 should appear before Extras")

	// Each action is in its expected section.
	require.Contains(t, got, "- 🔴 ERROR CREATE: ContractB (service)")
	require.Contains(t, got, "- 🟡 WARNING MODIFY: ContractA")
	require.Contains(t, got, "- 🔵 INFO EXTRA: OldHelper (service)")

	// ContractB (Layer 0) should appear before ContractA (Layer 1) in output.
	contractBPos := strings.Index(got, "ContractB")
	contractAPos := strings.Index(got, "ContractA")
	require.Less(t, contractBPos, contractAPos, "ContractB (Layer 0) should appear before ContractA (Layer 1)")
}

// TestWriteDiffPlanReport_Determinism asserts that two consecutive calls on
// the same DiffPlanOutput produce byte-identical output (RNF-003).
func TestWriteDiffPlanReport_Determinism(t *testing.T) {
	t.Parallel()

	out := diffvo.DiffPlanOutput{
		Feature: "det-feature",
		Actions: []diffvo.DiffAction{
			{
				Type:         diffvo.DiffActionCreate,
				ContractName: "Alpha",
				ContractType: "service",
				Severity:     diffvo.DiffSeverityError,
				Layer:        0,
			},
			{
				Type:           diffvo.DiffActionModify,
				ContractName:   "Beta",
				ContractType:   "input-port",
				Severity:       diffvo.DiffSeverityWarning,
				Layer:          1,
				AddedMethods:   []string{"run"},
				RemovedMethods: []string{"exec"},
			},
			{
				Type:         diffvo.DiffActionExtra,
				ContractName: "Gamma",
				ContractType: "output-port",
				Severity:     diffvo.DiffSeverityInfo,
				Layer:        -1,
			},
		},
		HasChanges: true,
	}

	var buf1, buf2 bytes.Buffer

	err := format.WriteDiffPlanReport(&buf1, out)
	require.NoError(t, err)

	err = format.WriteDiffPlanReport(&buf2, out)
	require.NoError(t, err)

	require.Equal(t, buf1.String(), buf2.String(), "WriteDiffPlanReport must be deterministic")
}

// TestWriteDiffPlanReport_VerbatimCleanLine asserts the exact literal string
// from Gherkin scenario 4 appears in the clean-diff output.
func TestWriteDiffPlanReport_VerbatimCleanLine(t *testing.T) {
	t.Parallel()

	out := diffvo.DiffPlanOutput{}

	var buf bytes.Buffer
	err := format.WriteDiffPlanReport(&buf, out)
	require.NoError(t, err)

	require.Contains(t, buf.String(), "No diff detected. Current state matches spec.")
}

// TestWriteDiffPlanReport_FeatureHeaderOmittedWhenEmpty asserts that the HTML
// comment header is suppressed when Feature is an empty string.
func TestWriteDiffPlanReport_FeatureHeaderOmittedWhenEmpty(t *testing.T) {
	t.Parallel()

	out := diffvo.DiffPlanOutput{
		Feature: "", // empty → no header comment
		Actions: []diffvo.DiffAction{
			{
				Type:         diffvo.DiffActionCreate,
				ContractName: "SomeContract",
				ContractType: "service",
				Severity:     diffvo.DiffSeverityError,
				Layer:        0,
			},
		},
		HasChanges: true,
	}

	var buf bytes.Buffer
	err := format.WriteDiffPlanReport(&buf, out)
	require.NoError(t, err)

	got := buf.String()
	require.NotContains(t, got, "<!--")
	require.Contains(t, got, "## Diff Plan")
}
