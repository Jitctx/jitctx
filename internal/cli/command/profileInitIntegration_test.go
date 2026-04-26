package command_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/application/usecase/profileinituc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
)

// buildProfileInitCmd constructs a "profile init" cobra command wired
// with real infrastructure adapters (fsprofile.NewBundled and
// fsprofile.NewExtractor). profilesDir is the profiles directory
// relative to the workDir passed via --dir.
func buildProfileInitCmd(t *testing.T, profilesDir string) *cobra.Command {
	t.Helper()
	uc := profileinituc.New(fsprofile.NewBundled(), fsprofile.NewExtractor(), discardLogger())
	return command.NewProfileInitCmd(uc, profilesDir, discardLogger())
}

// TestProfileInitCmd_CopiesBundledProfile verifies the happy path:
// "jitctx profile init spring-boot-hexagonal" extracts the bundled
// profile into <workDir>/.jitctx/profiles/spring-boot-hexagonal/.
// EP04US-006 / EP04RF-011 — Scenario 1.
func TestProfileInitCmd_CopiesBundledProfile(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	profilesDir := ".jitctx/profiles"

	cmd := buildProfileInitCmd(t, profilesDir)

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", workDir, "spring-boot-hexagonal"})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Initialised profile")

	// Filesystem assertions.
	extractedDir := filepath.Join(workDir, profilesDir, "spring-boot-hexagonal")
	require.DirExists(t, extractedDir, "extracted profile directory must exist")
	require.FileExists(t, filepath.Join(extractedDir, "profile.yaml"), "profile.yaml must be copied")
	require.DirExists(t, filepath.Join(extractedDir, "templates"), "templates/ subdirectory must be copied")
	require.FileExists(t, filepath.Join(extractedDir, "README.md"), "README.md must be copied")
}

// TestProfileInitCmd_FailsWhenTargetExists verifies that running
// "profile init" a second time (when the target directory is already
// present) returns a non-nil error with a diagnostic pointing to the
// existing directory.
// EP04US-006 / EP04RF-011 — Scenario 3.
func TestProfileInitCmd_FailsWhenTargetExists(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	profilesDir := ".jitctx/profiles"

	// Pre-create the target directory to trigger the failure branch.
	targetDir := filepath.Join(workDir, profilesDir, "spring-boot-hexagonal")
	require.NoError(t, os.MkdirAll(targetDir, 0o755))

	cmd := buildProfileInitCmd(t, profilesDir)

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", workDir, "spring-boot-hexagonal"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
	require.Contains(t, err.Error(), "remove it first or choose a different name")
}

// TestProfileInitCmd_FailsForUnknownName verifies that requesting a
// profile name that is not bundled returns a diagnostic listing the
// available bundled profiles.
// EP04US-006 / EP04RF-011 — Scenario 4.
func TestProfileInitCmd_FailsForUnknownName(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	profilesDir := ".jitctx/profiles"

	cmd := buildProfileInitCmd(t, profilesDir)

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", workDir, "fake-profile"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown bundled profile "fake-profile"`)
	require.Contains(t, err.Error(), "available: spring-boot-hexagonal")
}

// TestProfileInitCmd_RejectsTraversalName verifies that a positional
// argument containing a path separator is rejected before any filesystem
// access, with a stderr message matching the existing ErrProfileInvalid
// translation literal.
func TestProfileInitCmd_RejectsTraversalName(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	profilesDir := ".jitctx/profiles"

	cmd := buildProfileInitCmd(t, profilesDir)

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", workDir, "../bad"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "framework profile is invalid")
}

// TestProfileInitCmd_NoDebrisOnFailure verifies that when init fails
// because the target directory already exists, no sibling tmp
// directories are left under the parent profiles directory.
// EP04US-006 — Bonus no-debris assertion.
func TestProfileInitCmd_NoDebrisOnFailure(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	profilesDir := ".jitctx/profiles"

	// Pre-create the target to induce a failure.
	targetDir := filepath.Join(workDir, profilesDir, "spring-boot-hexagonal")
	require.NoError(t, os.MkdirAll(targetDir, 0o755))

	cmd := buildProfileInitCmd(t, profilesDir)

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", workDir, "spring-boot-hexagonal"})

	// The command must fail.
	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)

	// The parent profiles directory must contain exactly one entry (the
	// pre-created target). No *.tmp-* siblings should exist.
	parentDir := filepath.Join(workDir, profilesDir)
	entries, readErr := os.ReadDir(parentDir)
	require.NoError(t, readErr)
	require.Len(t, entries, 1, "parent profiles dir must contain exactly one entry after a failed init")
	require.Equal(t, "spring-boot-hexagonal", entries[0].Name())
}
