package command

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/vo"
)

// bufferedLogger returns a *slog.Logger that writes text records into buf.
// The caller can inspect buf.String() to assert on Warn lines.
func bufferedLogger(buf *bytes.Buffer) *slog.Logger {
	h := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(h)
}

// countWarnLines counts lines in s that contain level=WARN.
func countWarnLines(s string) int {
	n := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "level=WARN") {
			n++
		}
	}
	return n
}

func TestParseArtifactTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		raw       []string
		want      []vo.ArtifactType
		wantWarns int
	}{
		{
			name:      "empty-input-returns-empty-slice",
			raw:       []string{},
			want:      []vo.ArtifactType{},
			wantWarns: 0,
		},
		{
			name:      "nil-input-returns-empty-slice",
			raw:       nil,
			want:      []vo.ArtifactType{},
			wantWarns: 0,
		},
		{
			name: "all-valid-types-preserved-in-order",
			raw:  []string{"guidelines", "scenarios", "requirements", "contracts"},
			want: []vo.ArtifactType{
				vo.ArtifactGuidelines,
				vo.ArtifactScenarios,
				vo.ArtifactRequirements,
				vo.ArtifactContracts,
			},
			wantWarns: 0,
		},
		{
			name:      "whitespace-around-entries-is-trimmed",
			raw:       []string{" guidelines ", " scenarios"},
			want:      []vo.ArtifactType{vo.ArtifactGuidelines, vo.ArtifactScenarios},
			wantWarns: 0,
		},
		{
			name:      "unknown-value-is-dropped-and-warn-emitted",
			raw:       []string{"guidelines", "junk"},
			want:      []vo.ArtifactType{vo.ArtifactGuidelines},
			wantWarns: 1,
		},
		{
			name:      "all-unknown-returns-empty-slice-with-warn-per-entry",
			raw:       []string{"junk", "foo"},
			want:      []vo.ArtifactType{},
			wantWarns: 2,
		},
		{
			name:      "case-variation-is-rejected-validate-is-case-sensitive",
			raw:       []string{"Guidelines"},
			want:      []vo.ArtifactType{},
			wantWarns: 1,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			logger := bufferedLogger(&buf)

			got := parseArtifactTypes(tc.raw, logger)

			require.Equal(t, tc.want, got)
			require.Equal(t, tc.wantWarns, countWarnLines(buf.String()),
				"unexpected number of WARN log lines; log output:\n%s", buf.String())

			// For warn cases, assert the log mentions the "ignoring" message
			// and the offending value attribute, to lock EP01RF-008 contract.
			if tc.wantWarns > 0 {
				logOut := buf.String()
				require.Contains(t, logOut, "ignoring unknown --type value")
			}
		})
	}
}

// TestParseArtifactTypes_WarnContainsValue asserts that the Warn record
// carries the "value" attribute so callers can diagnose the bad input.
func TestParseArtifactTypes_WarnContainsValue(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := bufferedLogger(&buf)

	_ = parseArtifactTypes([]string{"junk"}, logger)

	logOut := buf.String()
	require.Contains(t, logOut, "value=junk",
		"WARN record must carry value= attribute; log output:\n%s", logOut)
	require.Contains(t, logOut, "accepted=",
		"WARN record must carry accepted= attribute; log output:\n%s", logOut)
}
