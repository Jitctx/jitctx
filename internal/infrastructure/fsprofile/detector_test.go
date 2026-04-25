package fsprofile_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
)

// writeSampleProfile reads the shared spring-boot-hexagonal sample YAML from
// testdata/fsprofile/ and writes it into dir so the detector can find it.
func writeSampleProfile(t *testing.T, dir string) {
	t.Helper()
	yamlPath := filepath.Join("..", "..", "..", "testdata", "fsprofile", "spring-boot-hexagonal.yaml")
	data, err := os.ReadFile(yamlPath)
	require.NoError(t, err, "read sample profile fixture")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "spring-boot-hexagonal.yaml"), data, 0o644))
}

func TestDetector_PomXML(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"pom.xml": &fstest.MapFile{Data: []byte(`<project>
			<parent>
				<groupId>org.springframework.boot</groupId>
			</parent>
		</project>`)},
	}

	dir := t.TempDir()
	writeSampleProfile(t, dir)

	d := fsprofile.NewDetector(dir)
	prof, err := d.Detect(context.Background(), fsys)
	require.NoError(t, err)
	require.Equal(t, "spring-boot-hexagonal", prof.Name)
	require.Equal(t, model.ProfileSourceCustom, prof.Source)
}

func TestDetector_BuildGradle(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"build.gradle": &fstest.MapFile{Data: []byte(`plugins {
			id 'org.springframework.boot' version '3.2.0'
		}`)},
	}

	dir := t.TempDir()
	writeSampleProfile(t, dir)

	d := fsprofile.NewDetector(dir)
	prof, err := d.Detect(context.Background(), fsys)
	require.NoError(t, err)
	require.Equal(t, "spring-boot-hexagonal", prof.Name)
	require.Equal(t, model.ProfileSourceCustom, prof.Source)
}

func TestDetector_NoMatch(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"README.md": &fstest.MapFile{Data: []byte("# My Project")},
	}

	d := fsprofile.NewDetector(t.TempDir())
	_, err := d.Detect(context.Background(), fsys)
	require.True(t, errors.Is(err, domerr.ErrNoProfileMatch))
}

func TestDetector_BuildGradleKts(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"build.gradle.kts": &fstest.MapFile{Data: []byte(`plugins {
			id("org.springframework.boot") version "3.2.0"
		}`)},
	}

	dir := t.TempDir()
	writeSampleProfile(t, dir)

	d := fsprofile.NewDetector(dir)
	prof, err := d.Detect(context.Background(), fsys)
	require.NoError(t, err)
	require.Equal(t, "spring-boot-hexagonal", prof.Name)
	require.Equal(t, model.ProfileSourceCustom, prof.Source)
}

// TestDetector_MultipleCustomProfiles_FirstAlphabeticalWins verifies that when
// multiple custom profiles both match a project, the alphabetically-first one
// is selected (EP01RNF-002 deterministic precedence).
func TestDetector_MultipleCustomProfiles_FirstAlphabeticalWins(t *testing.T) {
	t.Parallel()

	// Both profiles match pom.xml containing "org.springframework.boot".
	sharedYAML := func(name string) []byte {
		return []byte(`name: ` + name + `
languages: [java]
query_lang: java
detect:
  files:
    - name: pom.xml
      contains: "org.springframework.boot"
module_detection:
  strategy: hexagonal
  roots:
    - src/main/java/**
  markers:
    - kind: path_contains
      value: /port/in/
rules: []
`)
	}

	customDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(customDir, "aaa-spring.yaml"), sharedYAML("aaa-spring"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(customDir, "my-spring.yaml"), sharedYAML("my-spring"), 0o644))

	fsys := fstest.MapFS{
		"pom.xml": &fstest.MapFile{Data: []byte(`<project>
			<parent>
				<groupId>org.springframework.boot</groupId>
			</parent>
		</project>`)},
	}

	d := fsprofile.NewDetector(customDir)
	prof, err := d.Detect(context.Background(), fsys)
	require.NoError(t, err)
	require.Equal(t, "aaa-spring", prof.Name)
	require.Equal(t, model.ProfileSourceCustom, prof.Source)
}

func TestDetector_CustomSourceStamp(t *testing.T) {
	t.Parallel()

	// Write a minimal custom profile that matches a specific marker file.
	customDir := t.TempDir()
	customYAML := []byte(`name: custom-profile
languages: [java]
query_lang: java
detect:
  files:
    - name: custom-marker.txt
      contains: "custom"
module_detection:
  strategy: hexagonal
  roots:
    - src/main/java/**
  markers:
    - kind: path_contains
      value: /port/in/
rules: []
`)
	err := os.WriteFile(filepath.Join(customDir, "custom-profile.yaml"), customYAML, 0o644)
	require.NoError(t, err)

	fsys := fstest.MapFS{
		"custom-marker.txt": &fstest.MapFile{Data: []byte("custom project marker")},
	}

	d := fsprofile.NewDetector(customDir)
	prof, err := d.Detect(context.Background(), fsys)
	require.NoError(t, err)
	require.Equal(t, model.ProfileSourceCustom, prof.Source)
}
