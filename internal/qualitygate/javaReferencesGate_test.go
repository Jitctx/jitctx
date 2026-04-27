package qualitygate_test

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/qualitygate"
)

// repoRoot returns the absolute path of the repository root. Go tests run with
// CWD = the package directory, so two levels up from internal/qualitygate/ is
// the repo root.
func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../..")
	require.NoError(t, err, "resolving repo root")
	return abs
}

// isExempt reports whether the repo-relative path is listed in ExemptFiles.
func isExempt(rel string) bool {
	for _, e := range qualitygate.ExemptFiles {
		if e == rel {
			return true
		}
	}
	return false
}

// isSkippedDir reports whether the slash-separated path contains a segment
// named "testdata" or "bundled", which are exempt by design.
func isSkippedDir(slashPath string) bool {
	for _, seg := range strings.Split(slashPath, "/") {
		if seg == "testdata" || seg == "bundled" {
			return true
		}
	}
	return false
}

// strippedBytes returns a copy of src where:
//  1. Every byte inside a Go comment is replaced by a space (newlines kept).
//  2. Every Go import-path value starting with the jitctx application/ prefix
//     is replaced by spaces of the same width.
//
// Byte offsets are preserved so that line information remains accurate.
func strippedBytes(t *testing.T, fset *token.FileSet, f *ast.File, src []byte) []byte {
	t.Helper()
	out := make([]byte, len(src))
	copy(out, src)

	file := fset.File(f.Pos())
	if file == nil {
		return out
	}

	// Strip comment ranges.
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			start := file.Offset(c.Slash)
			end := file.Offset(c.End())
			for i := start; i < end && i < len(out); i++ {
				if out[i] != '\n' {
					out[i] = ' '
				}
			}
		}
	}

	// Strip import-path strings that resolve to jitctx's application/ layer.
	const appPrefix = `"github.com/jitctx/jitctx/internal/application/`
	for _, imp := range f.Imports {
		if imp.Path == nil {
			continue
		}
		if !strings.HasPrefix(imp.Path.Value, appPrefix) {
			continue
		}
		start := file.Offset(imp.Path.Pos())
		end := file.Offset(imp.Path.End())
		for i := start; i < end && i < len(out); i++ {
			if out[i] != '\n' {
				out[i] = ' '
			}
		}
	}

	return out
}

// finding captures a single forbidden-token match.
type finding struct {
	rel  string // repo-relative path (slash-separated)
	line int
	tok  string
	text string // trimmed source line containing the match
}

// lineOf returns the 1-based line number for byte offset off inside src.
func lineOf(src []byte, off int) int {
	return bytes.Count(src[:off], []byte{'\n'}) + 1
}

// lineText returns the trimmed content of the line in src that contains off.
func lineText(src []byte, off int) string {
	start := off
	for start > 0 && src[start-1] != '\n' {
		start--
	}
	end := off
	for end < len(src) && src[end] != '\n' {
		end++
	}
	return strings.TrimSpace(string(src[start:end]))
}

// TestJavaReferencesGate walks every *.go production file under internal/ and
// cmd/ in the repository root, strips comments and jitctx application/ import
// paths, and asserts that none of qualitygate.ForbiddenTokens appear in the
// remaining bytes.
//
// Files in qualitygate.ExemptFiles are skipped. Files whose names end in
// _test.go are fully exempt (plan §8 Q1). Directories named testdata or
// bundled are skipped entirely.
func TestJavaReferencesGate(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)

	var (
		findings      []finding
		filesVisited  int
		exemptionsHit int
	)

	scanDirs := []string{
		filepath.Join(root, "internal"),
		filepath.Join(root, "cmd"),
	}

	for _, dir := range scanDirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			relSlash := filepath.ToSlash(rel)

			if d.IsDir() {
				if isSkippedDir(relSlash) {
					return filepath.SkipDir
				}
				return nil
			}

			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil // _test.go files are fully exempt per plan §8 Q1
			}
			if isSkippedDir(relSlash) {
				return nil
			}
			if isExempt(relSlash) {
				exemptionsHit++
				return nil
			}

			filesVisited++

			src, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Errorf("cannot read %s: %v", relSlash, readErr)
				return nil
			}

			fset := token.NewFileSet()
			f, parseErr := parser.ParseFile(fset, path, src, parser.ParseComments)
			if parseErr != nil && f == nil {
				t.Errorf("cannot parse %s: %v", relSlash, parseErr)
				return nil
			}

			stripped := strippedBytes(t, fset, f, src)

			for _, tok := range qualitygate.ForbiddenTokens {
				needle := []byte(tok)
				off := 0
				for {
					idx := bytes.Index(stripped[off:], needle)
					if idx < 0 {
						break
					}
					absOff := off + idx
					findings = append(findings, finding{
						rel:  relSlash,
						line: lineOf(stripped, absOff),
						tok:  tok,
						text: lineText(stripped, absOff),
					})
					off = absOff + len(needle)
				}
			}

			return nil
		})
		require.NoError(t, err, "walking %s", dir)
	}

	t.Logf("gate: files_visited=%d exemptions_hit=%d violations=%d",
		filesVisited, exemptionsHit, len(findings))

	if len(findings) == 0 {
		return
	}

	// Sort deterministically by (file, line, token).
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].rel != findings[j].rel {
			return findings[i].rel < findings[j].rel
		}
		if findings[i].line != findings[j].line {
			return findings[i].line < findings[j].line
		}
		return findings[i].tok < findings[j].tok
	})

	// Group by file and emit one t.Errorf per file.
	type fileGroup struct {
		rel      string
		findings []finding
	}
	var groups []fileGroup
	for _, ff := range findings {
		if len(groups) == 0 || groups[len(groups)-1].rel != ff.rel {
			groups = append(groups, fileGroup{rel: ff.rel})
		}
		groups[len(groups)-1].findings = append(groups[len(groups)-1].findings, ff)
	}

	for _, g := range groups {
		var sb strings.Builder
		for _, ff := range g.findings {
			fmt.Fprintf(&sb, "  L%d  token %q  ↦  %s\n", ff.line, ff.tok, ff.text)
		}
		t.Errorf("forbidden Java/Spring identifiers in %s:\n%s"+
			"  => if intentional, add the file to qualitygate.ExemptFiles with a justification "+
			"(internal/qualitygate/exemptions.go)",
			g.rel, sb.String())
	}
}

// TestJavaReferencesGate_HonoursExemptions is a self-test that ensures every
// entry in qualitygate.ExemptFiles still contains at least one ForbiddenTokens
// entry in its raw bytes (no comment stripping). When a file is cleaned up so
// that it no longer contains any forbidden token, this test fails with a
// message asking the author to remove the stale exemption entry from
// qualitygate.ExemptFiles. This forces hygiene: a clean-up PR and its
// exemption removal must land together.
func TestJavaReferencesGate_HonoursExemptions(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)

	for _, rel := range qualitygate.ExemptFiles {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			t.Parallel()

			abs := filepath.Join(root, filepath.FromSlash(rel))
			src, err := os.ReadFile(abs)
			require.NoError(t, err, "exempt file must exist: %s", rel)

			found := false
			for _, tok := range qualitygate.ForbiddenTokens {
				if bytes.Contains(src, []byte(tok)) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("exempt file %s no longer contains any ForbiddenTokens entry; "+
					"the exemption is stale — remove it from qualitygate.ExemptFiles in "+
					"internal/qualitygate/exemptions.go", rel)
			}
		})
	}
}

// TestJavaReferencesGate_NegativeControl validates the comment-stripping
// pipeline itself: a forbidden token in a string literal must be detected,
// while the same token inside a line comment must not survive stripping.
func TestJavaReferencesGate_NegativeControl(t *testing.T) {
	t.Parallel()

	// Snippet: "@Entity" as a string literal (forbidden) and "@RestController"
	// inside a comment (should be stripped and therefore not detected).
	src := []byte(`package foo

// @RestController lives in a comment — must be stripped.
var annotation = "@Entity"
`)

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "fixture.go", src, parser.ParseComments)
	require.NoError(t, err)

	stripped := strippedBytes(t, fset, f, src)

	// Token in a string literal must survive stripping.
	require.True(t, bytes.Contains(stripped, []byte("@Entity")),
		"@Entity in a string literal must be detected after stripping")

	// Token in a comment must be gone.
	require.False(t, bytes.Contains(stripped, []byte("@RestController")),
		"@RestController inside a comment must not be detected after stripping")
}
