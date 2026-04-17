package fscontext

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFrontMatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantHasFM   bool
		wantID      string
		wantTags    []string
		wantBodyHas string
		wantErr     bool
	}{
		{
			name:        "no_opening_delimiter",
			input:       "# Just a body\nno front matter here\n",
			wantHasFM:   false,
			wantBodyHas: "Just a body",
		},
		{
			name:        "missing_closing_delimiter",
			input:       "---\nid: foo\ntags: [bar]\n# body starts here\n",
			wantHasFM:   false,
			wantBodyHas: "body starts here",
		},
		{
			name:  "empty_front_matter",
			input: "---\n---\nbody content here\n",
			// yaml.Decode of empty string returns an error, so parseFrontMatter
			// treats it as no front matter and returns the full content as body.
			wantHasFM:   false,
			wantBodyHas: "body content here",
		},
		{
			name:        "valid_front_matter",
			input:       "---\nid: my-doc\ntags: [java, spring]\n---\n# Body\ncontent\n",
			wantHasFM:   true,
			wantID:      "my-doc",
			wantTags:    []string{"java", "spring"},
			wantBodyHas: "content",
		},
		{
			name:        "crlf_line_endings",
			input:       "---\r\nid: crlf-doc\r\ntags: [crlf]\r\n---\r\nbody text\r\n",
			wantHasFM:   true,
			wantID:      "crlf-doc",
			wantTags:    []string{"crlf"},
			wantBodyHas: "body text",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fm, body, hasFM, err := parseFrontMatter([]byte(tc.input))

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantHasFM, hasFM, "hasFM mismatch")

			if tc.wantBodyHas != "" {
				require.True(t, strings.Contains(body, tc.wantBodyHas),
					"body %q does not contain %q", body, tc.wantBodyHas)
			}

			if tc.wantHasFM {
				require.Equal(t, tc.wantID, fm.ID, "id mismatch")
				if tc.wantTags != nil {
					require.Equal(t, tc.wantTags, fm.Tags, "tags mismatch")
				}
			}
		})
	}
}
