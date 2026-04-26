// Package fsgit provides a read-only adapter that satisfies
// domain/port/git ports via subprocess invocations of the `git` binary.
// RNF-002: no mutating subcommand is ever invoked.
package fsgit

import (
	"bytes"
	"context"
	"os/exec"
)

// IsGitAvailable reports whether `git` is present on PATH and the given
// repoRoot directory is inside a git work tree.
//
// The check performs two probes:
//  1. exec.LookPath("git") — returns false if the binary is not found.
//  2. git -C <repoRoot> rev-parse --is-inside-work-tree — returns false
//     when the exit code is non-zero or stdout is not "true".
//
// Results are not cached; repoRoot is request-scoped and the per-call cost
// (single-digit ms) is acceptable for a CLI tool.
func IsGitAvailable(ctx context.Context, repoRoot string) bool {
	if _, err := exec.LookPath("git"); err != nil {
		return false
	}
	cmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return bytes.TrimSpace(out) != nil && string(bytes.TrimSpace(out)) == "true"
}
