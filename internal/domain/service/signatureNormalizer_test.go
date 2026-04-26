package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/service"
)

func TestSignatureNormalizer_Normalize(t *testing.T) {
	t.Parallel()

	n := service.NewSignatureNormalizer()

	cases := []struct {
		name string
		sig  string
		want string
	}{
		// Gherkin scenario 6 binding: whitespace inside parens must collapse.
		{
			name: "whitespace-inside-parens-gherkin-scenario-6",
			sig:  "User save( User user )",
			want: "User save(User user)",
		},
		// Idempotent: already normalised input must be unchanged.
		{
			name: "already-normalised-idempotent",
			sig:  "User save(User user)",
			want: "User save(User user)",
		},
		// Multiple internal spaces collapsed to a single space.
		{
			name: "multiple-internal-spaces-collapsed",
			sig:  "User  save(User  user)",
			want: "User save(User user)",
		},
		// Leading and trailing whitespace trimmed.
		{
			name: "leading-and-trailing-whitespace-trimmed",
			sig:  "  User save(User user)  ",
			want: "User save(User user)",
		},
		// Trailing semicolon trimmed.
		{
			name: "trailing-semicolon-trimmed",
			sig:  "User save(User user) ;",
			want: "User save(User user)",
		},
		// Trailing semicolon with no space trimmed.
		{
			name: "trailing-semicolon-no-space-trimmed",
			sig:  "User save(User user);",
			want: "User save(User user)",
		},
		// Trailing newline trimmed (outer whitespace rule).
		{
			name: "trailing-newline-trimmed",
			sig:  "User save(User user)\n",
			want: "User save(User user)",
		},
		// Generic types preserved inside and outside parens.
		{
			name: "generic-type-preserved",
			sig:  "Optional<User> findByEmail( String email )",
			want: "Optional<User> findByEmail(String email)",
		},
		// Generic type in params preserved.
		{
			name: "generic-type-in-params-preserved",
			sig:  "List<String> findAll()",
			want: "List<String> findAll()",
		},
		// Multi-param signature: whitespace around commas normalised.
		{
			name: "multi-param-whitespace-normalised",
			sig:  "User save( User user , Locale locale )",
			want: "User save(User user , Locale locale)",
		},
		// Multi-param clean input idempotent.
		{
			name: "multi-param-already-normalised-idempotent",
			sig:  "User save(User user, Locale locale)",
			want: "User save(User user, Locale locale)",
		},
		// Signature with leading/trailing spaces and generic return type.
		{
			name: "leading-trailing-spaces-with-generic-return",
			sig:  "  Optional<User> findByEmail( String email )  ",
			want: "Optional<User> findByEmail(String email)",
		},
		// No parens: method name only (edge case — collapses whitespace only).
		{
			name: "method-name-only-no-parens",
			sig:  "save",
			want: "save",
		},
		// No parens with extra spaces.
		{
			name: "method-name-with-spaces-no-parens",
			sig:  "  find  ",
			want: "find",
		},
		// Unmatched paren: falls back to trimmed original.
		{
			name: "unmatched-open-paren-falls-back-to-trimmed-original",
			sig:  "garbage(",
			want: "garbage(",
		},
		// Unmatched paren with extra spaces: trimmed before fallback.
		{
			name: "unmatched-paren-with-spaces-trimmed-original",
			sig:  "  broken(sig  ",
			want: "broken(sig",
		},
		// Void return, empty params.
		{
			name: "void-return-empty-params",
			sig:  "void execute()",
			want: "void execute()",
		},
		// Space between method name and parens collapsed.
		{
			name: "space-between-name-and-parens-collapsed",
			sig:  "void execute ()",
			want: "void execute()",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := n.Normalize(tc.sig)
			require.Equal(t, tc.want, got)
		})
	}
}
