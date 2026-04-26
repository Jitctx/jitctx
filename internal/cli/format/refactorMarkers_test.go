package format_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/cli/format"
	refactorvo "github.com/jitctx/jitctx/internal/domain/vo/refactor"
)

// TestWriteRefactorMarkersReport_EmptyMarkers verifies that an empty Markers
// slice emits exactly "No refactor markers found.\n" and nothing else —
// no HTML header, no headings.
func TestWriteRefactorMarkersReport_EmptyMarkers(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	out := refactorvo.ScanRefactorsOutput{Markers: nil}

	err := format.WriteRefactorMarkersReport(&buf, out)
	require.NoError(t, err)
	require.Equal(t, "No refactor markers found.\n", buf.String())
}

// TestWriteRefactorMarkersReport_EmptyMarkersSlice verifies the same behaviour
// when Markers is an explicitly empty (non-nil) slice.
func TestWriteRefactorMarkersReport_EmptyMarkersSlice(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	out := refactorvo.ScanRefactorsOutput{Markers: []refactorvo.RefactorMarker{}}

	err := format.WriteRefactorMarkersReport(&buf, out)
	require.NoError(t, err)
	require.Equal(t, "No refactor markers found.\n", buf.String())
}

// TestWriteRefactorMarkersReport_SingleMarkerPerType table-drives all known
// MarkerType constants (except Unparseable, which is handled separately) and
// verifies the type-string label is correct in the rendered bullet.
func TestWriteRefactorMarkersReport_SingleMarkerPerType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		markerType  refactorvo.MarkerType
		description string
		wantType    string
	}{
		{
			name:        "rename",
			markerType:  refactorvo.MarkerTypeRename,
			description: "rename foo to handleFoo",
			wantType:    "rename",
		},
		{
			name:        "extract-method",
			markerType:  refactorvo.MarkerTypeExtractMethod,
			description: "extract validateEmail into its own method",
			wantType:    "extract-method",
		},
		{
			name:        "move",
			markerType:  refactorvo.MarkerTypeMove,
			description: "move to infrastructure layer",
			wantType:    "move",
		},
		{
			name:        "inline",
			markerType:  refactorvo.MarkerTypeInline,
			description: "inline trivial helper",
			wantType:    "inline",
		},
		{
			name:        "simplify",
			markerType:  refactorvo.MarkerTypeSimplify,
			description: "simplify nested conditionals",
			wantType:    "simplify",
		},
		{
			name:        "other",
			markerType:  refactorvo.MarkerTypeOther,
			description: "some unknown action",
			wantType:    "other",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			marker := refactorvo.RefactorMarker{
				ModuleID:    "billing",
				FilePath:    "src/main/java/com/app/billing/BillingService.java",
				Line:        42,
				Type:        tc.markerType,
				Description: tc.description,
			}
			out := refactorvo.ScanRefactorsOutput{
				Markers: []refactorvo.RefactorMarker{marker},
			}

			var buf bytes.Buffer
			err := format.WriteRefactorMarkersReport(&buf, out)
			require.NoError(t, err)

			got := buf.String()

			// Header and top-level heading must be present.
			require.Contains(t, got, "<!-- jitctx scan --refactors -->")
			require.Contains(t, got, "## Refactor Markers")

			// Module heading.
			require.Contains(t, got, "## Module: billing")

			// Bullet uses the correct type label, em-dash (U+2014), backtick-wrapped
			// filepath:line, and the description.
			wantBullet := "- " + tc.wantType + " — `src/main/java/com/app/billing/BillingService.java:42` — " + tc.description
			require.Contains(t, got, wantBullet)
		})
	}
}

// TestWriteRefactorMarkersReport_UnparseableRendersOriginalText verifies that
// a marker with Type=Unparseable renders marker.OriginalText (not Description)
// as the trailing field in the bullet.
func TestWriteRefactorMarkersReport_UnparseableRendersOriginalText(t *testing.T) {
	t.Parallel()

	marker := refactorvo.RefactorMarker{
		ModuleID:     "user-management",
		FilePath:     "src/main/java/com/app/UserService.java",
		Line:         7,
		Type:         refactorvo.MarkerTypeUnparseable,
		Description:  "", // must NOT appear in output
		OriginalText: "// TODO(jitctx): missing separator here",
	}
	out := refactorvo.ScanRefactorsOutput{
		Markers: []refactorvo.RefactorMarker{marker},
	}

	var buf bytes.Buffer
	err := format.WriteRefactorMarkersReport(&buf, out)
	require.NoError(t, err)

	got := buf.String()

	// The bullet must carry OriginalText as trailing field.
	wantBullet := "- unparseable — `src/main/java/com/app/UserService.java:7` — // TODO(jitctx): missing separator here"
	require.Contains(t, got, wantBullet)

	// Description (empty here) should not appear as a double em-dash gap.
	require.NotContains(t, got, "— —")
}

// TestWriteRefactorMarkersReport_MultiModuleGroupingOrder verifies that
// markers in two different modules produce two ## Module: headings in the
// exact order the use case provides them, and that each marker lands under
// the correct heading.
func TestWriteRefactorMarkersReport_MultiModuleGroupingOrder(t *testing.T) {
	t.Parallel()

	markers := []refactorvo.RefactorMarker{
		{
			ModuleID:    "billing",
			FilePath:    "src/main/java/com/app/billing/BillingService.java",
			Line:        10,
			Type:        refactorvo.MarkerTypeRename,
			Description: "rename chargeUser to processPayment",
		},
		{
			ModuleID:    "billing",
			FilePath:    "src/main/java/com/app/billing/InvoiceRepository.java",
			Line:        55,
			Type:        refactorvo.MarkerTypeExtractMethod,
			Description: "extract toPdf",
		},
		{
			ModuleID:    "user-management",
			FilePath:    "src/main/java/com/app/user/UserService.java",
			Line:        22,
			Type:        refactorvo.MarkerTypeSimplify,
			Description: "simplify role check",
		},
	}
	out := refactorvo.ScanRefactorsOutput{Markers: markers}

	var buf bytes.Buffer
	err := format.WriteRefactorMarkersReport(&buf, out)
	require.NoError(t, err)

	got := buf.String()

	// Both module headings must appear.
	billingIdx := bytes.Index([]byte(got), []byte("## Module: billing"))
	userIdx := bytes.Index([]byte(got), []byte("## Module: user-management"))
	require.True(t, billingIdx >= 0, "billing module heading not found")
	require.True(t, userIdx >= 0, "user-management module heading not found")

	// billing heading appears before user-management (as provided by use case).
	require.Less(t, billingIdx, userIdx, "billing should appear before user-management")

	// Billing markers appear between the billing heading and the user-management heading.
	billingBullet1 := "- rename — `src/main/java/com/app/billing/BillingService.java:10` — rename chargeUser to processPayment"
	billingBullet2 := "- extract-method — `src/main/java/com/app/billing/InvoiceRepository.java:55` — extract toPdf"
	userBullet := "- simplify — `src/main/java/com/app/user/UserService.java:22` — simplify role check"

	billing1Idx := bytes.Index([]byte(got), []byte(billingBullet1))
	billing2Idx := bytes.Index([]byte(got), []byte(billingBullet2))
	userBulletIdx := bytes.Index([]byte(got), []byte(userBullet))

	require.True(t, billing1Idx >= 0, "first billing bullet not found")
	require.True(t, billing2Idx >= 0, "second billing bullet not found")
	require.True(t, userBulletIdx >= 0, "user-management bullet not found")

	// Billing bullets appear before user-management bullet.
	require.Less(t, billing1Idx, userBulletIdx)
	require.Less(t, billing2Idx, userBulletIdx)

	// User bullet appears after billing section heading.
	require.Greater(t, userBulletIdx, userIdx)
}

// TestWriteRefactorMarkersReport_Determinism verifies that two consecutive
// calls to WriteRefactorMarkersReport with identical input produce byte-identical
// output (RNF-003).
func TestWriteRefactorMarkersReport_Determinism(t *testing.T) {
	t.Parallel()

	markers := []refactorvo.RefactorMarker{
		{
			ModuleID:    "billing",
			FilePath:    "src/main/java/com/app/billing/BillingService.java",
			Line:        10,
			Type:        refactorvo.MarkerTypeRename,
			Description: "rename chargeUser to processPayment",
		},
		{
			ModuleID:    "user-management",
			FilePath:    "src/main/java/com/app/user/UserService.java",
			Line:        22,
			Type:        refactorvo.MarkerTypeSimplify,
			Description: "simplify role check",
		},
	}
	out := refactorvo.ScanRefactorsOutput{Markers: markers}

	var buf1, buf2 bytes.Buffer

	err := format.WriteRefactorMarkersReport(&buf1, out)
	require.NoError(t, err)

	err = format.WriteRefactorMarkersReport(&buf2, out)
	require.NoError(t, err)

	require.Equal(t, buf1.Bytes(), buf2.Bytes(),
		"two WriteRefactorMarkersReport calls on identical input must produce byte-identical output")
}

// TestWriteRefactorMarkersReport_StaleTypedMarker verifies that a typed marker
// with Stale=true appends " (stale candidate)" after the description in the bullet.
func TestWriteRefactorMarkersReport_StaleTypedMarker(t *testing.T) {
	t.Parallel()

	marker := refactorvo.RefactorMarker{
		ModuleID:    "billing",
		FilePath:    "src/main/java/com/app/billing/BillingService.java",
		Line:        42,
		Type:        refactorvo.MarkerTypeRename,
		Description: "rename foo to handleFoo",
		Stale:       true,
	}
	out := refactorvo.ScanRefactorsOutput{
		Markers: []refactorvo.RefactorMarker{marker},
	}

	var buf bytes.Buffer
	err := format.WriteRefactorMarkersReport(&buf, out)
	require.NoError(t, err)

	got := buf.String()

	wantBullet := "- rename — `src/main/java/com/app/billing/BillingService.java:42` — rename foo to handleFoo (stale candidate)"
	require.Contains(t, got, wantBullet)
}

// TestWriteRefactorMarkersReport_NoStaleWhenFalse verifies that a typed marker
// with Stale=false does NOT append " (stale candidate)" to the bullet.
func TestWriteRefactorMarkersReport_NoStaleWhenFalse(t *testing.T) {
	t.Parallel()

	marker := refactorvo.RefactorMarker{
		ModuleID:    "billing",
		FilePath:    "src/main/java/com/app/billing/BillingService.java",
		Line:        42,
		Type:        refactorvo.MarkerTypeRename,
		Description: "rename foo to handleFoo",
		Stale:       false,
	}
	out := refactorvo.ScanRefactorsOutput{
		Markers: []refactorvo.RefactorMarker{marker},
	}

	var buf bytes.Buffer
	err := format.WriteRefactorMarkersReport(&buf, out)
	require.NoError(t, err)

	got := buf.String()

	require.NotContains(t, got, "(stale candidate)")
	wantBullet := "- rename — `src/main/java/com/app/billing/BillingService.java:42` — rename foo to handleFoo\n"
	require.Contains(t, got, wantBullet)
}

// TestWriteRefactorMarkersReport_StaleUnparseableMarker verifies that an
// unparseable marker with Stale=true renders OriginalText in the bullet AND
// appends " (stale candidate)" as the suffix.
func TestWriteRefactorMarkersReport_StaleUnparseableMarker(t *testing.T) {
	t.Parallel()

	marker := refactorvo.RefactorMarker{
		ModuleID:     "user-management",
		FilePath:     "src/main/java/com/app/UserService.java",
		Line:         7,
		Type:         refactorvo.MarkerTypeUnparseable,
		Description:  "",
		OriginalText: "// TODO(jitctx): missing separator here",
		Stale:        true,
	}
	out := refactorvo.ScanRefactorsOutput{
		Markers: []refactorvo.RefactorMarker{marker},
	}

	var buf bytes.Buffer
	err := format.WriteRefactorMarkersReport(&buf, out)
	require.NoError(t, err)

	got := buf.String()

	// Bullet must use OriginalText (not Description) AND end with the stale suffix.
	wantBullet := "- unparseable — `src/main/java/com/app/UserService.java:7` — // TODO(jitctx): missing separator here (stale candidate)"
	require.Contains(t, got, wantBullet)
}

// TestWriteRefactorMarkersReport_FullOutputStructure verifies the complete
// output structure for a non-empty report: HTML comment header present,
// top-level ## Refactor Markers heading present, and the order of sections.
func TestWriteRefactorMarkersReport_FullOutputStructure(t *testing.T) {
	t.Parallel()

	out := refactorvo.ScanRefactorsOutput{
		Markers: []refactorvo.RefactorMarker{
			{
				ModuleID:    "billing",
				FilePath:    "src/main/java/com/app/billing/BillingService.java",
				Line:        1,
				Type:        refactorvo.MarkerTypeMove,
				Description: "move to domain layer",
			},
		},
	}

	var buf bytes.Buffer
	err := format.WriteRefactorMarkersReport(&buf, out)
	require.NoError(t, err)

	got := buf.String()

	htmlHeaderIdx := bytes.Index([]byte(got), []byte("<!-- jitctx scan --refactors -->"))
	refactorHeadingIdx := bytes.Index([]byte(got), []byte("## Refactor Markers"))
	moduleHeadingIdx := bytes.Index([]byte(got), []byte("## Module: billing"))
	bulletIdx := bytes.Index([]byte(got), []byte("- move"))

	require.True(t, htmlHeaderIdx >= 0, "HTML comment header missing")
	require.True(t, refactorHeadingIdx >= 0, "## Refactor Markers heading missing")
	require.True(t, moduleHeadingIdx >= 0, "## Module: billing heading missing")
	require.True(t, bulletIdx >= 0, "bullet line missing")

	// Structural order: HTML comment → ## Refactor Markers → ## Module: → bullet.
	require.Less(t, htmlHeaderIdx, refactorHeadingIdx)
	require.Less(t, refactorHeadingIdx, moduleHeadingIdx)
	require.Less(t, moduleHeadingIdx, bulletIdx)
}
