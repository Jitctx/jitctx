package command_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	appscanuc "github.com/jitctx/jitctx/internal/application/usecase/scanuc"
	"github.com/jitctx/jitctx/internal/cli/command"
	domspecsvc "github.com/jitctx/jitctx/internal/domain/service"
	"github.com/jitctx/jitctx/internal/domain/usecase/scanuc"
	"github.com/jitctx/jitctx/internal/infrastructure/fscontext"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
	"github.com/jitctx/jitctx/internal/infrastructure/token"
	"github.com/jitctx/jitctx/internal/infrastructure/treesitter"
)

// captureHandler is a slog.Handler that records every log record in a slice.
// Used to assert structured attributes (e.g. source=custom) in integration tests.
type captureHandler struct {
	records []slog.Record
	level   slog.Level
}

func newCaptureHandler(level slog.Level) *captureHandler {
	return &captureHandler{level: level}
}

func (h *captureHandler) Enabled(_ context.Context, l slog.Level) bool { return l >= h.level }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nh := &captureHandler{level: h.level, records: h.records}
	return nh
}
func (h *captureHandler) WithGroup(name string) slog.Handler {
	return h
}

// hasAttr returns true when any captured record contains an attribute with the
// given key and value.
func (h *captureHandler) hasAttr(key, value string) bool {
	for _, r := range h.records {
		var found bool
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == key && a.Value.String() == value {
				found = true
				return false
			}
			return true
		})
		if found {
			return true
		}
	}
	return false
}

// hasMsg returns true when any captured record has the given message as a prefix.
func (h *captureHandler) hasMsg(msg string) bool {
	for _, r := range h.records {
		if r.Message == msg {
			return true
		}
	}
	return false
}

// buildResolvingScanFactory wires a ScanUseCaseFactory that uses fsprofile.NewResolver
// so that EP04RF-012 precedence (user-dir wins over bundled) is exercised. The
// profilesDir is a relative path (e.g. ".jitctx/profiles") used by the resolver to
// locate user-dir profiles inside the project workDir.
func buildResolvingScanFactory(profilesDir string, logger *slog.Logger) command.ScanUseCaseFactory {
	resolver := fsprofile.NewResolver(
		fsprofile.NewBundleLoader(logger, nil),
		fsprofile.NewBundled(),
		logger,
	)
	return func(manifestPath string) scanuc.UseCase {
		return appscanuc.New(
			resolver,
			domspecsvc.NewDeclarativeClassifier(),
			treesitter.NewWalker(),
			treesitter.New(),
			fscontext.New(),
			fscontext.New(),
			token.NewHeuristicEstimator(),
			fsmanifest.New(manifestPath),
			profilesDir,
			logger,
		)
	}
}

// TestScanCmd_UserProfileTakesPrecedenceOverBundled verifies EP04US-006 Scenario 2:
// when the user has extracted a bundled profile to disk, jitctx scan loads the
// user-dir version (source=custom) rather than the bundled embed (EP04RF-012).
func TestScanCmd_UserProfileTakesPrecedenceOverBundled(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	// The fixture already placed a spring-boot-hexagonal/ user-dir profile in workDir
	// via copyFixture. Add a distinguishing comment to prove the scanner reads the
	// user-dir copy, not the bundled embed.
	profilesDir := ".jitctx/profiles"
	userProfilePath := filepath.Join(workDir, profilesDir, "spring-boot-hexagonal", "profile.yaml")
	require.FileExists(t, userProfilePath, "fixture must have placed the user-dir profile")

	profileBytes, err := os.ReadFile(userProfilePath)
	require.NoError(t, err)
	editedBytes := append([]byte("# user-edit-marker\n"), profileBytes...)
	require.NoError(t, os.WriteFile(userProfilePath, editedBytes, 0o644))

	// Build the scan factory and command with a capturing logger.
	capture := newCaptureHandler(slog.LevelInfo)
	logger := slog.New(capture)
	factory := buildResolvingScanFactory(profilesDir, logger)

	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, nil, logger)
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	// Assert the resolver selected the user-dir profile (source=custom).
	require.True(t, capture.hasAttr("source", "custom"),
		"expected slog record with source=custom; records: %v", capture.records)

	// Assert the "Profile: spring-boot-hexagonal" log line was emitted.
	require.True(t, capture.hasMsg("Profile: spring-boot-hexagonal"),
		"expected Profile: spring-boot-hexagonal log message")

	// Confirm the user-dir file contains the edit marker (belt-and-suspenders
	// check that the file we edited is indeed the one the resolver picked).
	updatedBytes, err := os.ReadFile(userProfilePath)
	require.NoError(t, err)
	require.Contains(t, string(updatedBytes), "# user-edit-marker")
}

// TestScanCmd_ExplicitProfileFlagOverridesAuto verifies EP04US-006 Bonus Scenario 5:
// when the user passes --profile <name> with no matching user-dir profile, the
// resolver falls back to the bundled embed (source=bundled).
func TestScanCmd_ExplicitProfileFlagOverridesAuto(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	// Empty project dir — no user-dir profiles, only the bundled embed.
	require.NoError(t, os.MkdirAll(filepath.Join(workDir, ".jitctx", "profiles"), 0o755))

	// Build the scan factory and command with a capturing logger.
	capture := newCaptureHandler(slog.LevelInfo)
	logger := slog.New(capture)
	factory := buildResolvingScanFactory(".jitctx/profiles", logger)

	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, nil, logger)
	cmd.SetOut(&stdout)
	// Explicit --profile flag: resolver's Name != "" branch fires.
	// No user-dir profile named spring-boot-hexagonal → falls back to bundled embed.
	cmd.SetArgs([]string{"--path", workDir, "--profile", "spring-boot-hexagonal"})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	// Assert the resolver selected the bundled embed (source=bundled).
	require.True(t, capture.hasAttr("source", "bundled"),
		"expected slog record with source=bundled; records: %v", capture.records)

	// Assert the log line records the profile name.
	require.True(t, capture.hasMsg("Profile: spring-boot-hexagonal"),
		"expected Profile: spring-boot-hexagonal log message")
}
