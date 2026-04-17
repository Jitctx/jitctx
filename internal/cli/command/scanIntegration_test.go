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
	"github.com/jitctx/jitctx/internal/domain/usecase/scanuc"
	"github.com/jitctx/jitctx/internal/infrastructure/fscontext"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
	"github.com/jitctx/jitctx/internal/infrastructure/token"
	"github.com/jitctx/jitctx/internal/infrastructure/treesitter"
)

// buildScanFactoryWithLogger creates a factory that uses real adapters with a given logger.
func buildScanFactoryWithLogger(profilesDir string, logger *slog.Logger) command.ScanUseCaseFactory {
	return func(manifestPath string) scanuc.UseCase {
		return appscanuc.New(
			fsprofile.NewDetectorWithLogger(profilesDir, logger),
			treesitter.NewWalker(),
			treesitter.New(),
			fscontext.New(),
			fscontext.New(),
			token.NewHeuristicEstimator(),
			fsmanifest.New(manifestPath),
			logger,
		)
	}
}

func TestScanCmd_Integration_HappyPath(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")
	factory := buildScanFactoryWithLogger(profilesDir, discardLogger())

	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, discardLogger())
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)

	actual, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	expected, err := os.ReadFile(fixtureDir(t, "springBootMinimal", "expected", "project-state.yaml"))
	require.NoError(t, err)

	actualStripped := normalizeYAML(stripGeneratedAt(string(actual)))
	expectedStripped := normalizeYAML(stripGeneratedAt(string(expected)))
	require.Equal(t, expectedStripped, actualStripped)

	t.Run("stdout_format", func(t *testing.T) {
		require.Contains(t, stdout.String(), "scanned:")
		require.Contains(t, stdout.String(), "modules")
		require.Contains(t, stdout.String(), "contexts")
	})

	t.Run("inter_module_dependency", func(t *testing.T) {
		require.Contains(t, string(actual), "notification")
	})

	t.Run("context_discovery", func(t *testing.T) {
		require.Contains(t, string(actual), "java-conventions")
		require.Contains(t, string(actual), "token_estimate:")
	})
}

func TestScanCmd_Integration_NoProfile(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	// Only a README - no pom.xml or build.gradle.
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# My Project"), 0o644))

	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")
	factory := buildScanFactoryWithLogger(profilesDir, discardLogger())

	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, discardLogger())
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "no matching framework profile found")
}

func TestScanCmd_Integration_PartialParse(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	// Capture stderr to check for Broken.java warning.
	var stderrBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&stderrBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")
	factory := buildScanFactoryWithLogger(profilesDir, logger)

	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, logger)
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir})

	// Command should succeed (exit 0) despite Broken.java.
	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)

	// stderr should contain a warning about Broken.java.
	require.Contains(t, stderrBuf.String(), "Broken.java")
}

func TestScanCmd_Integration_Deterministic(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")

	runScan := func() string {
		manifestPath := filepath.Join(workDir, "project-state-det.yaml")
		factory := buildScanFactoryWithLogger(profilesDir, discardLogger())
		cmd := command.NewScanCmd(factory, discardLogger())
		cmd.SetOut(os.Stdout)
		cmd.SetArgs([]string{"--path", workDir, "--manifest", manifestPath})
		require.NoError(t, cmd.ExecuteContext(context.Background()))
		data, err := os.ReadFile(manifestPath)
		require.NoError(t, err)
		return string(data)
	}

	run1 := stripGeneratedAt(runScan())
	run2 := stripGeneratedAt(runScan())
	require.Equal(t, run1, run2)
}

func TestScanCmd_Integration_AutoDetectGradle(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	// Replace pom.xml with build.gradle.
	require.NoError(t, os.Remove(filepath.Join(workDir, "pom.xml")))
	gradleContent := `plugins {
    id 'org.springframework.boot' version '3.2.0'
}
dependencies {
    implementation 'org.springframework.boot:spring-boot-starter'
}`
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "build.gradle"), []byte(gradleContent), 0o644))

	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")
	factory := buildScanFactoryWithLogger(profilesDir, discardLogger())
	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, discardLogger())
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "scanned:")
}

func TestScanCmd_Integration_CustomProfileOverride(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	// Add custom profile.
	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")
	require.NoError(t, os.MkdirAll(profilesDir, 0o755))

	customProfile := `name: my-spring
languages: [java]
query_lang: java
detect:
  files:
    - name: pom.xml
      contains: "org.springframework.boot"
module_detection:
  strategy: hexagonal
  roots: []
  markers: []
rules:
  - match:
      node_type: interface_declaration
      path_contains: /port/in/
    classify_as: input-port
  - match:
      node_type: class_declaration
      has_annotation: Entity
    classify_as: entity
`
	require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "my-spring.yaml"), []byte(customProfile), 0o644))

	factory := buildScanFactoryWithLogger(profilesDir, discardLogger())
	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, discardLogger())
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir, "--profile", "my-spring"})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)

	manifest, err := os.ReadFile(filepath.Join(workDir, "project-state.yaml"))
	require.NoError(t, err)
	// The manifest should use the custom profile (my-spring).
	require.Contains(t, string(manifest), "my-spring")
}

func TestScanCmd_Integration_ProfileLog(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")

	// Use a real logger writing to a buffer.
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	factory := buildScanFactoryWithLogger(profilesDir, logger)
	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, logger)
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)

	// slog TextHandler writes "profile=spring-boot-hexagonal".
	require.Contains(t, logBuf.String(), "profile=spring-boot-hexagonal")
}

func TestScanCmd_Integration_ManifestTraversalRejected(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()

	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")
	factory := buildScanFactoryWithLogger(profilesDir, discardLogger())

	cmd := command.NewScanCmd(factory, discardLogger())
	cmd.SetOut(os.Stdout)
	cmd.SetArgs([]string{"--path", workDir, "--manifest", "../escape.yaml"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err, "scan should reject --manifest that escapes --dir")
	require.Contains(t, err.Error(), "escapes project directory")
}
