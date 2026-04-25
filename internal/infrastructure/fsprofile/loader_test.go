package fsprofile_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
)

func TestLoader_LoadsFromUserDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSampleProfile(t, dir)

	l := fsprofile.New(dir)
	prof, err := l.Load(context.Background(), "spring-boot-hexagonal")
	require.NoError(t, err)
	require.Equal(t, "spring-boot-hexagonal", prof.Name)
	require.Contains(t, prof.Languages, "java")
	require.NotEmpty(t, prof.Rules)
	require.Equal(t, model.ProfileSourceCustom, prof.Source)
}

func TestLoader_LoadsCustomByName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	customYAML := []byte(`name: my-spring
languages: [java]
query_lang: java
detect:
  files:
    - name: pom.xml
      contains: "my-company"
module_detection:
  strategy: hexagonal
  roots: []
  markers: []
rules:
  - match:
      node_type: class_declaration
      has_annotation: Service
    classify_as: service
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "my-spring.yaml"), customYAML, 0o644))

	l := fsprofile.New(dir)
	prof, err := l.Load(context.Background(), "my-spring")
	require.NoError(t, err)
	require.Equal(t, "my-spring", prof.Name)
}

func TestLoader_RejectsTraversal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		profileName string
	}{
		{name: "dotdot_slash", profileName: "../../etc/passwd"},
		{name: "dotdot_backslash", profileName: `..\..\etc\passwd`},
		{name: "slash_in_name", profileName: "subdir/profile"},
		{name: "backslash_in_name", profileName: `subdir\profile`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			l := fsprofile.New(t.TempDir())
			_, err := l.Load(context.Background(), tc.profileName)
			require.Error(t, err)
			require.ErrorIs(t, err, domerr.ErrProfileInvalid)
		})
	}
}

// TestLoader_MalformedUserProfileReturnsError asserts that a malformed YAML
// file in userDir produces a hard error wrapped with ErrProfileInvalid rather
// than any silent fallback (there is no bundled fallback after the
// externalize-profiles chore).
func TestLoader_MalformedUserProfileReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "spring-boot-hexagonal.yaml"),
		[]byte("invalid: yaml: {{{"), 0o644))

	l := fsprofile.New(dir)
	_, err := l.Load(context.Background(), "spring-boot-hexagonal")
	require.Error(t, err)
	require.ErrorIs(t, err, domerr.ErrProfileInvalid)
}

// TestLoader_NotFound asserts that loading a non-existent profile from an
// empty directory returns an error matching ErrProfileInvalid with a message
// containing the profile name.
func TestLoader_NotFound(t *testing.T) {
	t.Parallel()

	l := fsprofile.New(t.TempDir())
	_, err := l.Load(context.Background(), "nonexistent")
	require.Error(t, err)
	require.ErrorIs(t, err, domerr.ErrProfileInvalid)
	require.Contains(t, err.Error(), "nonexistent")
}

// TestLoader_ListEmptyDir asserts that List on an existing but empty directory
// returns an empty slice with no error.
func TestLoader_ListEmptyDir(t *testing.T) {
	t.Parallel()

	l := fsprofile.New(t.TempDir())
	names, err := l.List(context.Background())
	require.NoError(t, err)
	require.Empty(t, names)
}

// TestLoader_ListMissingDir asserts that List on a non-existent directory
// returns an empty slice with no error (natural state of a fresh project).
func TestLoader_ListMissingDir(t *testing.T) {
	t.Parallel()

	nonExistent := filepath.Join(t.TempDir(), "does-not-exist")
	l := fsprofile.New(nonExistent)
	names, err := l.List(context.Background())
	require.NoError(t, err)
	require.Empty(t, names)
}
