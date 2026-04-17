package fsprofile_test

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
)

func TestDetector_PomXML(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"pom.xml": &fstest.MapFile{Data: []byte(`<project>
			<parent>
				<groupId>org.springframework.boot</groupId>
			</parent>
		</project>`)},
	}

	d := fsprofile.NewDetector(t.TempDir())
	prof, err := d.Detect(context.Background(), fsys)
	require.NoError(t, err)
	require.Equal(t, "spring-boot-hexagonal", prof.Name)
}

func TestDetector_BuildGradle(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"build.gradle": &fstest.MapFile{Data: []byte(`plugins {
			id 'org.springframework.boot' version '3.2.0'
		}`)},
	}

	d := fsprofile.NewDetector(t.TempDir())
	prof, err := d.Detect(context.Background(), fsys)
	require.NoError(t, err)
	require.Equal(t, "spring-boot-hexagonal", prof.Name)
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
