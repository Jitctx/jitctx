package command_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// differentSpec is a spec with a module and contract names that are
// recognisably distinct from canonicalSpec, so assertions can prove which
// spec was actually parsed.
const differentSpec = `# Feature: other-feature
Module: different-module

## Contract: OtherUseCase
Type: input-port
Methods:
- void run()

## Contract: OtherServiceImpl
Type: service
Implements: OtherUseCase
`

func TestSpecPathResolution_Integration_ExplicitFileWinsOverConventions(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	otherTmp := t.TempDir()

	// Write canonicalSpec (user-management / CreateUserUseCase) to the
	// conventional location so the finder would pick it up if --file were
	// not honoured.
	writeSpec(t, filepath.Join(tmp, "jitctx-plans"), "create-user", canonicalSpec)

	// Write a recognisably different spec that the --file flag should load.
	differentPath := filepath.Join(otherTmp, "different.md")
	require.NoError(t, os.MkdirAll(otherTmp, 0o755))
	require.NoError(t, os.WriteFile(differentPath, []byte(differentSpec), 0o644))

	cmd, stdout, _ := newPlanLayersCmdFor(t, tmp, "")
	cmd.SetArgs([]string{"--file", differentPath})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	out := stdout.String()
	require.Contains(t, out, "different-module")
	require.Contains(t, out, "OtherUseCase")
	require.NotContains(t, out, "user-management")
	require.NotContains(t, out, "CreateUserUseCase")
}

func TestSpecPathResolution_Integration_FeatureChecksJitctxPlansFirst(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	primaryPath := writeSpec(t, filepath.Join(tmp, "jitctx-plans"), "create-user", canonicalSpec)
	additionalPath := writeSpec(t, filepath.Join(tmp, ".jitctx", "plans"), "create-user", canonicalSpec)

	cmd, stdout, stderr := newPlanLayersCmdFor(t, tmp, "")
	cmd.SetArgs([]string{"--feature", "create-user"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	out := stdout.String()
	require.Contains(t, out, "CreateUserUseCase")

	stderrStr := stderr.String()
	require.Contains(t, stderrStr, "spec found in additional location")
	require.Contains(t, stderrStr, "additional="+additionalPath)
	require.Contains(t, stderrStr, "primary="+primaryPath)
}

func TestSpecPathResolution_Integration_FeatureFallsBackToJitctxPlansHidden(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// Write spec only to the hidden-dir location; do NOT create jitctx-plans/.
	writeSpec(t, filepath.Join(tmp, ".jitctx", "plans"), "create-user", canonicalSpec)

	cmd, stdout, stderr := newPlanLayersCmdFor(t, tmp, "")
	cmd.SetArgs([]string{"--feature", "create-user"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	require.Contains(t, stdout.String(), "CreateUserUseCase")
	require.NotContains(t, stderr.String(), "spec found in additional location")
}

func TestSpecPathResolution_Integration_FeatureRespectsConfigPlansDir(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// Write spec only under the custom plans dir; do NOT create jitctx-plans/
	// or .jitctx/plans/.
	writeSpec(t, filepath.Join(tmp, "specs", "features"), "create-user", canonicalSpec)

	cmd, stdout, stderr := newPlanLayersCmdFor(t, tmp, "specs/features")
	cmd.SetArgs([]string{"--feature", "create-user"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	require.Contains(t, stdout.String(), "CreateUserUseCase")
	require.NotContains(t, stderr.String(), "spec found in additional location")
}

func TestSpecPathResolution_Integration_FeatureNoSpecFoundListsAllSearched(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir() // empty — no spec files

	cmd, _, stderr := newPlanLayersCmdFor(t, tmp, "specs/features")
	cmd.SetArgs([]string{"--feature", "create-user"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)

	combined := err.Error() + stderr.String()
	require.Contains(t, combined, `spec file not found for feature "create-user"`)
	require.Contains(t, combined, filepath.Join(tmp, "jitctx-plans", "create-user.md"))
	require.Contains(t, combined, filepath.Join(tmp, ".jitctx", "plans", "create-user.md"))
	require.Contains(t, combined, filepath.Join(tmp, "specs", "features", "create-user.md"))
}
