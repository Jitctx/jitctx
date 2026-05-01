package command_test

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// forbiddenEngineTokens — see §2.6. Frozen, alphabetical, verbatim
// from PC01RNF-001's metric regex. This slice is the ONE location in
// the codebase where the literals are permitted; the file that declares
// them is in the self-exempt list so the test does not flag itself.
var forbiddenEngineTokens = []string{
	"Autowired",
	"JPA",
	"Lombok",
	"Mockito",
	"Spring",
}

// engineRoots — see §2.6. The three directories scoped by
// PC01RNF-001's metric grep argument list.
var engineRoots = []string{
	"internal/domain",
	"internal/application",
	"internal/cli",
}

// TestEngineLanguageNeutrality_NoFrameworkIdentifiers_PC01US014 walks
// every regular .go file under engineRoots (recursive) and asserts that
// none of forbiddenEngineTokens appears as a substring. Runs the
// equivalent of:
//
//	grep -rE "(Lombok|Spring|Mockito|Autowired|JPA)" \
//	     internal/domain internal/application internal/cli
//
// PC01RNF-001 (engine language-neutrality), PC01RF-010
// (language-adapter abstraction), R-005 (mitigation).
//
// Exemptions:
//   - This test file itself (engineLanguageNeutralityIntegration_test.go)
//     is skipped — it is the only legitimate home for the frozen token list.
//   - Any file residing under a testdata/ directory is skipped.
func TestEngineLanguageNeutrality_NoFrameworkIdentifiers_PC01US014(t *testing.T) {
	t.Parallel()

	// repoRoot: go test runs from the package directory, which is
	// internal/cli/command — walk up three levels to reach the module root.
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err, "must resolve repo root from package dir")

	type hit struct {
		path  string
		line  int
		token string
		text  string
	}
	var hits []hit

	const selfBasename = "engineLanguageNeutralityIntegration_test.go"

	for _, root := range engineRoots {
		absRoot := filepath.Join(repoRoot, root)
		walkErr := filepath.WalkDir(absRoot, func(p string, d os.DirEntry, entryErr error) error {
			if entryErr != nil {
				return entryErr
			}

			// Skip testdata/ directories entirely (do not recurse).
			if d.IsDir() && d.Name() == "testdata" {
				return filepath.SkipDir
			}
			if d.IsDir() {
				return nil
			}

			// Only scan Go source files.
			if !strings.HasSuffix(p, ".go") {
				return nil
			}

			// Self-exemption: skip this test file so the frozen token
			// slice does not flag itself.
			if filepath.Base(p) == selfBasename {
				return nil
			}

			content, readErr := os.ReadFile(p)
			require.NoError(t, readErr, "read %s", p)

			relPath := strings.TrimPrefix(p, repoRoot+string(filepath.Separator))
			for lineIdx, line := range strings.Split(string(content), "\n") {
				for _, tok := range forbiddenEngineTokens {
					if strings.Contains(line, tok) {
						hits = append(hits, hit{
							path:  relPath,
							line:  lineIdx + 1,
							token: tok,
							text:  strings.TrimSpace(line),
						})
					}
				}
			}
			return nil
		})
		require.NoError(t, walkErr, "walk %s", absRoot)
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].path != hits[j].path {
			return hits[i].path < hits[j].path
		}
		if hits[i].line != hits[j].line {
			return hits[i].line < hits[j].line
		}
		return hits[i].token < hits[j].token
	})

	if len(hits) > 0 {
		var b strings.Builder
		b.WriteString("PC01RNF-001 violation: forbidden framework " +
			"identifiers in the engine layers. Move per-language " +
			"behaviour into internal/infrastructure/treesitter/<lang>/.\n")
		b.WriteString("Scoped roots: " + strings.Join(engineRoots, ", ") + "\n")
		b.WriteString("Forbidden tokens: " + strings.Join(forbiddenEngineTokens, ", ") + "\n\n")
		for _, h := range hits {
			b.WriteString(h.path)
			b.WriteString(":")
			b.WriteString(strconv.Itoa(h.line))
			b.WriteString(": ")
			b.WriteString(h.text)
			b.WriteString("\n")
		}
		t.Fatal(b.String())
	}
}
