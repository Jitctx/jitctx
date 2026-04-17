package fsprofile_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
)

func TestLoader_BundledLoad(t *testing.T) {
	t.Parallel()

	l := fsprofile.New(t.TempDir())
	prof, err := l.Load(context.Background(), "spring-boot-hexagonal")
	require.NoError(t, err)
	require.Equal(t, "spring-boot-hexagonal", prof.Name)
	require.Contains(t, prof.Languages, "java")
	require.NotEmpty(t, prof.Rules)
}

func TestLoader_CustomOverride(t *testing.T) {
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

func TestLoader_MalformedCustomFallsBackToBundled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Write a malformed YAML file named spring-boot-hexagonal.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "spring-boot-hexagonal.yaml"),
		[]byte("invalid: yaml: {{{"), 0o644))

	l := fsprofile.New(dir)
	// Should fall back to bundled.
	prof, err := l.Load(context.Background(), "spring-boot-hexagonal")
	require.NoError(t, err)
	require.Equal(t, "spring-boot-hexagonal", prof.Name)
}
