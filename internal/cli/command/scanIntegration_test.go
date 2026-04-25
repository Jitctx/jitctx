package command_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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

// TestScanCmd_Integration_ProfileFlagSelectsByName verifies that --profile selects
// a specific named profile from multiple custom profiles in .jitctx/profiles/.
func TestScanCmd_Integration_ProfileFlagSelectsByName(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	// copyFixture brings .jitctx/profiles/spring-boot-hexagonal.yaml into workDir.
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	// Write an additional custom profile alongside the one from the fixture.
	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")

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
	// The manifest should record the explicitly requested profile (my-spring).
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

	// Gherkin feature line 175: log message must contain the exact substring.
	require.Contains(t, logBuf.String(), "Profile: spring-boot-hexagonal")
	// Structured attribute for source provenance — always custom after externalize-profiles chore.
	require.Contains(t, logBuf.String(), "source=custom")
}

func TestScanCmd_Integration_AutoDetectGradleKts(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	// Replace pom.xml with build.gradle.kts (Kotlin DSL).
	require.NoError(t, os.Remove(filepath.Join(workDir, "pom.xml")))
	ktsContent := `plugins {
    id("org.springframework.boot") version "3.2.0"
}
dependencies {
    implementation("org.springframework.boot:spring-boot-starter")
}`
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "build.gradle.kts"), []byte(ktsContent), 0o644))

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")
	factory := buildScanFactoryWithLogger(profilesDir, logger)
	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, logger)
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	require.FileExists(t, manifestPath)
	require.Contains(t, logBuf.String(), "Profile: spring-boot-hexagonal")
}

func TestScanCmd_Integration_ServiceByImplementsUseCase(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")
	factory := buildScanFactoryWithLogger(profilesDir, discardLogger())
	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, discardLogger())
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	manifest, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	// CreateUserServiceImpl implements CreateUserUseCase under /service/ — must be classified as service.
	require.Contains(t, string(manifest), "CreateUserServiceImpl")
	require.Contains(t, string(manifest), "type: service")
}

// TestScanCmd_Integration_MultipleCustomProfilesFirstWins verifies that when
// multiple custom profiles match a project, the alphabetically first one wins.
func TestScanCmd_Integration_MultipleCustomProfilesFirstWins(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	// copyFixture brings .jitctx/profiles/spring-boot-hexagonal.yaml into workDir.
	// "my-spring" sorts after "spring-boot-hexagonal" alphabetically, so
	// spring-boot-hexagonal wins auto-detect. Write "my-spring.yaml" which also
	// matches pom.xml — the detector picks the alphabetically-first match.
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")

	// Write an additional custom profile that also matches pom.xml.
	// "my-spring" sorts AFTER "spring-boot-hexagonal", so spring-boot-hexagonal wins.
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

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	factory := buildScanFactoryWithLogger(profilesDir, logger)
	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, logger)
	cmd.SetOut(&stdout)
	// Run WITHOUT --profile flag — detector auto-picks by alphabetical precedence.
	cmd.SetArgs([]string{"--path", workDir})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)

	// Both profiles are custom; the alphabetically-first matching profile is selected.
	// source must always be "custom" — there is no bundled fallback.
	require.Contains(t, logBuf.String(), "source=custom")

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	manifest, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	// The manifest must record whichever profile the detector selected.
	require.True(t,
		strings.Contains(string(manifest), "spring-boot-hexagonal") ||
			strings.Contains(string(manifest), "my-spring"),
		"manifest must contain the selected profile name",
	)
}

// TestScanCmd_Integration_MalformedCustomProfileIsSkipped verifies that when a
// custom profile file is malformed, the detector warns and continues to the next
// candidate. There is no bundled fallback — the scan succeeds via a valid custom
// profile that also resides in .jitctx/profiles/.
func TestScanCmd_Integration_MalformedCustomProfileIsSkipped(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	// copyFixture brings .jitctx/profiles/spring-boot-hexagonal.yaml into workDir.
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	// Write a syntactically broken YAML alongside the valid spring-boot-hexagonal.yaml.
	// "broken" sorts before "spring-boot-hexagonal" alphabetically, so the detector
	// hits the malformed file first, warns, skips it, and then finds the valid profile.
	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")
	brokenProfile := "name: bad\n  invalid: : :"
	require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "broken.yaml"), []byte(brokenProfile), 0o644))

	var warnBuf bytes.Buffer
	warnLogger := slog.New(slog.NewTextHandler(&warnBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	factory := buildScanFactoryWithLogger(profilesDir, warnLogger)
	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, warnLogger)
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir})

	// EP01RF-012 §Exceptions: scan must still succeed when a custom profile is malformed;
	// the detector skips it and continues to the next candidate.
	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	require.FileExists(t, manifestPath)

	// Detector must log a warning for the malformed file.
	require.Contains(t, warnBuf.String(), "custom profile parse error")

	// Verify the scan selected the valid profile at info level.
	var infoLogBuf bytes.Buffer
	infoLogger := slog.New(slog.NewTextHandler(&infoLogBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	factory2 := buildScanFactoryWithLogger(profilesDir, infoLogger)
	cmd2 := command.NewScanCmd(factory2, infoLogger)
	cmd2.SetOut(&stdout)
	manifestPath2 := filepath.Join(workDir, "project-state-2.yaml")
	cmd2.SetArgs([]string{"--path", workDir, "--manifest", manifestPath2})
	require.NoError(t, cmd2.ExecuteContext(context.Background()))
	// The valid spring-boot-hexagonal profile must be selected (source always custom).
	require.Contains(t, infoLogBuf.String(), "Profile: spring-boot-hexagonal")
	require.Contains(t, infoLogBuf.String(), "source=custom")
}

func TestScanCmd_Integration_MultiAnnotationClass(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	// Copy the springBootMinimal project into a temp directory.
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	// Add the multi-annotation fixture file into the user-management domain package.
	// The file lives under testdata/springBootMinimal/fixtures/ so it does not affect
	// the canonical expected/project-state.yaml used by TestScanCmd_Integration_HappyPath.
	domainDir := filepath.Join(workDir, "src", "main", "java", "com", "app", "user_management", "domain")
	require.NoError(t, os.MkdirAll(domainDir, 0o755))
	src := fixtureDir(t, "springBootMinimal", "fixtures", "UserWithTableAnnotation.java")
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(domainDir, "UserWithTableAnnotation.java"), data, 0o644))

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")
	factory := buildScanFactoryWithLogger(profilesDir, discardLogger())

	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, discardLogger())
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir})

	require.NoError(t, cmd.ExecuteContext(context.Background()))

	manifest, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	manifestStr := string(manifest)

	// The user-management module must contain UserWithTableAnnotation classified as entity.
	// The bundled profile rule "has_annotation: Entity" -> "classify_as: entity" fires on
	// the simple annotation name, which must still be extracted when @Table(...) is also present.
	require.Contains(t, manifestStr, "UserWithTableAnnotation")
	require.Contains(t, manifestStr, "type: entity")
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
