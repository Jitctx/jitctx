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
	domspecsvc "github.com/jitctx/jitctx/internal/domain/service"
	"github.com/jitctx/jitctx/internal/domain/usecase/scanuc"
	"github.com/jitctx/jitctx/internal/infrastructure/fscontext"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
	"github.com/jitctx/jitctx/internal/infrastructure/token"
	"github.com/jitctx/jitctx/internal/infrastructure/treesitter"
)

// buildScanFactoryWithLogger creates a ScanUseCaseFactory backed by fsprofile.NewResolver.
// The profilesDir argument is the relative profiles directory (e.g. ".jitctx/profiles");
// it is forwarded to appscanuc.New so the resolver can locate user-dir profiles at
// <workDir>/<profilesDir>/<name>/. An empty string is allowed; it causes the resolver
// to consult only the bundled embed for auto-detect paths.
func buildScanFactoryWithLogger(profilesDir string, logger *slog.Logger) command.ScanUseCaseFactory {
	resolver := fsprofile.NewResolver(
		fsprofile.NewBundleLoader(logger),
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

func TestScanCmd_Integration_HappyPath(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	factory := buildScanFactoryWithLogger(".jitctx/profiles", discardLogger())

	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, nil, discardLogger())
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
	// Only a README - no pom.xml or build.gradle, no user-dir profiles.
	// The Resolver falls back to the bundled spring-boot-hexagonal profile
	// (EP04RF-012 — bundled embed is always consulted last). The scan
	// succeeds but produces no modules (no Java files to parse).
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# My Project"), 0o644))

	factory := buildScanFactoryWithLogger(".jitctx/profiles", discardLogger())

	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, nil, discardLogger())
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir})

	// With the Resolver the bundled profile is always available; an empty project
	// results in a valid (but empty) manifest rather than an error.
	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "scanned:")
}

func TestScanCmd_Integration_PartialParse(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	// Capture stderr to check for Broken.java warning.
	var stderrBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&stderrBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	factory := buildScanFactoryWithLogger(".jitctx/profiles", logger)

	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, nil, logger)
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

	runScan := func() string {
		manifestPath := filepath.Join(workDir, "project-state-det.yaml")
		factory := buildScanFactoryWithLogger(".jitctx/profiles", discardLogger())
		cmd := command.NewScanCmd(factory, nil, discardLogger())
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

	factory := buildScanFactoryWithLogger(".jitctx/profiles", discardLogger())
	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, nil, discardLogger())
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "scanned:")
}

// TestScanCmd_Integration_ProfileFlagSelectsByName verifies that --profile selects
// a specific named user-dir profile when multiple profiles exist in .jitctx/profiles/.
// The Resolver (EP04RF-012) tries the user-dir first for the explicit --profile name.
func TestScanCmd_Integration_ProfileFlagSelectsByName(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	// copyFixture brings the full spring-boot-hexagonal directory profile into workDir.
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	// Write an additional custom profile as a DIRECTORY alongside the one from the fixture.
	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")
	mySpringDir := filepath.Join(profilesDir, "my-spring")
	require.NoError(t, os.MkdirAll(mySpringDir, 0o755))

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
	require.NoError(t, os.WriteFile(filepath.Join(mySpringDir, "profile.yaml"), []byte(customProfile), 0o644))

	factory := buildScanFactoryWithLogger(".jitctx/profiles", discardLogger())
	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, nil, discardLogger())
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
	// copyFixture brings the directory-based spring-boot-hexagonal profile into workDir.
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	// Use a real logger writing to a buffer.
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	factory := buildScanFactoryWithLogger(".jitctx/profiles", logger)
	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, nil, logger)
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)

	// Gherkin feature line 175: log message must contain the exact substring.
	require.Contains(t, logBuf.String(), "Profile: spring-boot-hexagonal")
	// Source is "custom" because the user-dir profile is found at
	// <workdir>/.jitctx/profiles/spring-boot-hexagonal/ (EP04RF-012).
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

	factory := buildScanFactoryWithLogger(".jitctx/profiles", logger)
	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, nil, logger)
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

	factory := buildScanFactoryWithLogger(".jitctx/profiles", discardLogger())
	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, nil, discardLogger())
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	manifest, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	// CreateUserServiceImpl implements CreateUserUseCase under /service/ — must be classified as service.
	require.Contains(t, string(manifest), "CreateUserServiceImpl")
	require.Contains(t, string(manifest), "schema_version: 2")
	require.Contains(t, string(manifest), "types:")
	require.Contains(t, string(manifest), "- service")
}

// TestScanCmd_Integration_MultipleCustomProfilesFirstWins verifies that when
// multiple user-dir profiles exist, the alphabetically first one wins (EP04RF-012).
func TestScanCmd_Integration_MultipleCustomProfilesFirstWins(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	// copyFixture brings the spring-boot-hexagonal directory profile into workDir.
	// "my-spring" sorts AFTER "spring-boot-hexagonal" alphabetically, so
	// spring-boot-hexagonal wins the Resolver's auto-detect loop.
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")

	// Write an additional directory-based profile that also contains a valid
	// profile.yaml. "my-spring" > "spring-boot-hexagonal" alphabetically — the
	// Resolver returns the first alphabetical match (spring-boot-hexagonal).
	mySpringDir := filepath.Join(profilesDir, "my-spring")
	require.NoError(t, os.MkdirAll(mySpringDir, 0o755))
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
	require.NoError(t, os.WriteFile(filepath.Join(mySpringDir, "profile.yaml"), []byte(customProfile), 0o644))

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	factory := buildScanFactoryWithLogger(".jitctx/profiles", logger)
	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, nil, logger)
	cmd.SetOut(&stdout)
	// Run WITHOUT --profile flag — Resolver picks alphabetically first user-dir.
	cmd.SetArgs([]string{"--path", workDir})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)

	// User-dir profile wins — source is always "custom".
	require.Contains(t, logBuf.String(), "source=custom")

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	manifest, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	// The manifest must record whichever profile the resolver selected.
	require.True(t,
		strings.Contains(string(manifest), "spring-boot-hexagonal") ||
			strings.Contains(string(manifest), "my-spring"),
		"manifest must contain the selected profile name",
	)
}

// TestScanCmd_Integration_MalformedCustomProfileIsSkipped verifies that when a
// user-dir profile cannot be loaded (missing or malformed profile.yaml), the
// Resolver warns and continues to the next candidate (EP04RF-012).
func TestScanCmd_Integration_MalformedCustomProfileIsSkipped(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	// copyFixture brings the spring-boot-hexagonal directory profile into workDir.
	copyFixture(t, fixtureDir(t, "springBootMinimal", "project"), workDir)

	// Create a "broken" directory (no profile.yaml inside) — "broken" sorts before
	// "spring-boot-hexagonal" alphabetically, so the Resolver hits it first, logs a WARN,
	// skips it, then successfully loads spring-boot-hexagonal.
	profilesDir := filepath.Join(workDir, ".jitctx", "profiles")
	require.NoError(t, os.MkdirAll(filepath.Join(profilesDir, "broken"), 0o755))
	// Deliberately leave the directory empty (no profile.yaml) to trigger the skip.

	var warnBuf bytes.Buffer
	warnLogger := slog.New(slog.NewTextHandler(&warnBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	factory := buildScanFactoryWithLogger(".jitctx/profiles", warnLogger)
	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, nil, warnLogger)
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--path", workDir})

	// EP04RF-012: scan must still succeed when a user-dir profile is unloadable;
	// the Resolver skips it and continues to the next candidate.
	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err)

	manifestPath := filepath.Join(workDir, "project-state.yaml")
	require.FileExists(t, manifestPath)

	// Resolver logs a warning for the unloadable directory.
	require.Contains(t, warnBuf.String(), "resolver: skipping malformed user-dir profile")

	// Verify the scan selected the valid spring-boot-hexagonal profile at info level.
	var infoLogBuf bytes.Buffer
	infoLogger := slog.New(slog.NewTextHandler(&infoLogBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	factory2 := buildScanFactoryWithLogger(".jitctx/profiles", infoLogger)
	cmd2 := command.NewScanCmd(factory2, nil, infoLogger)
	cmd2.SetOut(&stdout)
	manifestPath2 := filepath.Join(workDir, "project-state-2.yaml")
	cmd2.SetArgs([]string{"--path", workDir, "--manifest", manifestPath2})
	require.NoError(t, cmd2.ExecuteContext(context.Background()))
	// The valid spring-boot-hexagonal profile must be selected (source=custom — user-dir).
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
	factory := buildScanFactoryWithLogger(".jitctx/profiles", discardLogger())

	var stdout bytes.Buffer
	cmd := command.NewScanCmd(factory, nil, discardLogger())
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
	require.Contains(t, manifestStr, "schema_version: 2")
	require.Contains(t, manifestStr, "types:")
	require.Contains(t, manifestStr, "- entity")
}

func TestScanCmd_Integration_ManifestTraversalRejected(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()

	factory := buildScanFactoryWithLogger(".jitctx/profiles", discardLogger())

	cmd := command.NewScanCmd(factory, nil, discardLogger())
	cmd.SetOut(os.Stdout)
	cmd.SetArgs([]string{"--path", workDir, "--manifest", "../escape.yaml"})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err, "scan should reject --manifest that escapes --dir")
	require.Contains(t, err.Error(), "escapes project directory")
}
