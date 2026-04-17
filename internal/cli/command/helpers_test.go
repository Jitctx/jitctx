package command_test

import (
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// copyFixture copies all files from src directory into dst directory.
func copyFixture(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copyFixture: %v", err)
	}
}

var generatedAtRe = regexp.MustCompile(`(?m)^generated_at:.*\n`)

// stripGeneratedAt removes the generated_at line so manifests can be compared.
func stripGeneratedAt(y string) string {
	return generatedAtRe.ReplaceAllString(y, "")
}

// fixtureDir returns the absolute path to a testdata directory.
func fixtureDir(t *testing.T, parts ...string) string {
	t.Helper()
	// The test runs from internal/cli/command/, so testdata is at ../../testdata.
	// Use absolute path by walking up from the source file.
	base := findProjectRoot(t)
	parts = append([]string{base, "testdata"}, parts...)
	return filepath.Join(parts...)
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	// Walk up until we find go.mod.
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// normalizeYAML normalizes YAML for comparison (trims whitespace).
func normalizeYAML(y string) string {
	lines := strings.Split(y, "\n")
	var out []string
	for _, l := range lines {
		trimmed := strings.TrimRight(l, " \t")
		out = append(out, trimmed)
	}
	return strings.Join(out, "\n")
}
