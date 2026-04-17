package fscontext

import (
	"bufio"
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"
)

// frontMatterResult holds parsed front matter and the remaining body.
type frontMatterResult struct {
	ID        string   `yaml:"id"`
	Tags      []string `yaml:"tags"`
	AppliesTo []string `yaml:"applies_to"`
	Module    string   `yaml:"module"`
}

// parseFrontMatter splits a markdown file into front matter and body.
// Front matter must start with "---\n" on the very first line.
// Returns (metadata, body, hasFrontMatter, error).
func parseFrontMatter(content []byte) (frontMatterResult, string, bool, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))

	// Check for opening "---".
	if !scanner.Scan() {
		return frontMatterResult{}, string(content), false, nil
	}
	firstLine := strings.TrimRight(scanner.Text(), "\r")
	if firstLine != "---" {
		return frontMatterResult{}, string(content), false, nil
	}

	// Collect front matter lines until the closing "---".
	var fmLines []string
	foundClose := false
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "---" {
			foundClose = true
			break
		}
		fmLines = append(fmLines, line)
	}

	if !foundClose {
		// No closing ---; treat entire file as body.
		return frontMatterResult{}, string(content), false, nil
	}

	// Collect body.
	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}

	fmYAML := strings.Join(fmLines, "\n")
	var fm frontMatterResult
	dec := yaml.NewDecoder(strings.NewReader(fmYAML))
	dec.KnownFields(false) // permissive — unknown keys tolerated
	if err := dec.Decode(&fm); err != nil {
		// Treat as no front matter on parse error.
		return frontMatterResult{}, string(content), false, nil
	}

	body := strings.Join(bodyLines, "\n")
	return fm, body, true, nil
}
