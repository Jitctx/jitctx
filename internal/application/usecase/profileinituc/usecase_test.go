package profileinituc_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/application/usecase/profileinituc"
	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// fakeListBundledProfilesPort is a hand-rolled fake for
// profileport.ListBundledProfilesPort.
type fakeListBundledProfilesPort struct {
	list func(ctx context.Context) ([]string, error)
}

func (f *fakeListBundledProfilesPort) ListBundled(ctx context.Context) ([]string, error) {
	return f.list(ctx)
}

// fakeExtractBundledProfilePort is a hand-rolled fake for
// profileport.ExtractBundledProfilePort. It records the last (name, target)
// pair passed to Extract and returns the configured error.
type fakeExtractBundledProfilePort struct {
	extract      func(ctx context.Context, name string, targetDir string) error
	calledName   string
	calledTarget string
	callCount    int
}

func (f *fakeExtractBundledProfilePort) Extract(ctx context.Context, name string, targetDir string) error {
	f.callCount++
	f.calledName = name
	f.calledTarget = targetDir
	if f.extract != nil {
		return f.extract(ctx, name, targetDir)
	}
	return nil
}

// bundledList returns a fakeListBundledProfilesPort that always returns the
// given names.
func bundledList(names ...string) *fakeListBundledProfilesPort {
	return &fakeListBundledProfilesPort{
		list: func(_ context.Context) ([]string, error) { return names, nil },
	}
}

// noopExtractor returns a fakeExtractBundledProfilePort that always succeeds.
func noopExtractor() *fakeExtractBundledProfilePort {
	return &fakeExtractBundledProfilePort{}
}

func TestProfileInitUC_HappyPath(t *testing.T) {
	t.Parallel()

	extractor := noopExtractor()
	uc := profileinituc.New(bundledList("spring-boot-hexagonal"), extractor, nil)

	out, err := uc.Execute(context.Background(), profilevo.ProfileInitInput{
		Name:        "spring-boot-hexagonal",
		WorkDir:     t.TempDir(),
		ProfilesDir: ".jitctx/profiles",
	})

	require.NoError(t, err)
	require.Equal(t, "spring-boot-hexagonal", out.Name)
	require.True(t, strings.HasSuffix(out.TargetDir, "/.jitctx/profiles/spring-boot-hexagonal"),
		"TargetDir %q should end in /.jitctx/profiles/spring-boot-hexagonal", out.TargetDir)
	require.GreaterOrEqual(t, out.FilesWritten, 0)
	require.Equal(t, 1, extractor.callCount)
	require.Equal(t, "spring-boot-hexagonal", extractor.calledName)
}

func TestProfileInitUC_RejectsEmptyName(t *testing.T) {
	t.Parallel()

	extractor := noopExtractor()
	uc := profileinituc.New(bundledList("spring-boot-hexagonal"), extractor, nil)

	_, err := uc.Execute(context.Background(), profilevo.ProfileInitInput{
		Name:        "",
		WorkDir:     ".",
		ProfilesDir: ".jitctx/profiles",
	})

	require.Error(t, err)
	require.ErrorIs(t, err, domerr.ErrProfileInvalid)
	require.Equal(t, 0, extractor.callCount, "extractor must NOT be called on validation failure")
}

func TestProfileInitUC_RejectsTraversalName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		profileName string
	}{
		{"double-dot-prefix", "../foo"},
		{"slash-in-name", "a/b"},
		{"backslash-in-name", `a\b`},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			extractor := noopExtractor()
			uc := profileinituc.New(bundledList("spring-boot-hexagonal"), extractor, nil)

			_, err := uc.Execute(context.Background(), profilevo.ProfileInitInput{
				Name:        tc.profileName,
				WorkDir:     ".",
				ProfilesDir: ".jitctx/profiles",
			})

			require.Error(t, err)
			require.ErrorIs(t, err, domerr.ErrProfileInvalid)
			require.Equal(t, 0, extractor.callCount, "extractor must NOT be called for invalid name %q", tc.profileName)
		})
	}
}

func TestProfileInitUC_UnknownName_ReturnsTypedError(t *testing.T) {
	t.Parallel()

	extractor := noopExtractor()
	uc := profileinituc.New(bundledList("spring-boot-hexagonal"), extractor, nil)

	_, err := uc.Execute(context.Background(), profilevo.ProfileInitInput{
		Name:        "fake",
		WorkDir:     ".",
		ProfilesDir: ".jitctx/profiles",
	})

	require.Error(t, err)

	var ubp *domerr.UnknownBundledProfileError
	require.ErrorAs(t, err, &ubp)
	require.Equal(t, "fake", ubp.Name)
	require.Equal(t, []string{"spring-boot-hexagonal"}, ubp.Available)
	require.Equal(t, 0, extractor.callCount, "extractor must NOT be called when name is unknown")
}

func TestProfileInitUC_TargetExists_PropagatesError(t *testing.T) {
	t.Parallel()

	targetErr := &domerr.ProfileTargetExistsError{Target: "/x"}
	extractor := &fakeExtractBundledProfilePort{
		extract: func(_ context.Context, _ string, _ string) error {
			return targetErr
		},
	}
	uc := profileinituc.New(bundledList("spring-boot-hexagonal"), extractor, nil)

	out, err := uc.Execute(context.Background(), profilevo.ProfileInitInput{
		Name:        "spring-boot-hexagonal",
		WorkDir:     ".",
		ProfilesDir: ".jitctx/profiles",
	})

	require.Error(t, err)
	require.ErrorIs(t, err, domerr.ErrProfileTargetExists)

	var pte *domerr.ProfileTargetExistsError
	require.ErrorAs(t, err, &pte)
	require.Equal(t, "/x", pte.Target)

	require.Equal(t, profilevo.ProfileInitOutput{}, out, "output must be zero on error")
}

func TestProfileInitUC_DefaultProfilesDir(t *testing.T) {
	t.Parallel()

	extractor := noopExtractor()
	uc := profileinituc.New(bundledList("spring-boot-hexagonal"), extractor, nil)

	_, err := uc.Execute(context.Background(), profilevo.ProfileInitInput{
		Name:        "spring-boot-hexagonal",
		WorkDir:     t.TempDir(),
		ProfilesDir: "", // intentionally empty — use case must default to .jitctx/profiles
	})

	require.NoError(t, err)
	require.True(t, strings.HasSuffix(extractor.calledTarget, ".jitctx/profiles/spring-boot-hexagonal"),
		"extractor target %q should end in .jitctx/profiles/spring-boot-hexagonal", extractor.calledTarget)
}

func TestProfileInitUC_ContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	extractor := noopExtractor()
	// Use a bundled port that would panic if called — the ctx check should
	// fire first.
	listPort := &fakeListBundledProfilesPort{
		list: func(_ context.Context) ([]string, error) {
			t.Fatal("ListBundled must not be called after ctx is cancelled")
			return nil, nil
		},
	}
	uc := profileinituc.New(listPort, extractor, nil)

	_, err := uc.Execute(ctx, profilevo.ProfileInitInput{
		Name:        "spring-boot-hexagonal",
		WorkDir:     ".",
		ProfilesDir: ".jitctx/profiles",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled), "expected context.Canceled, got %v", err)
	require.Equal(t, 0, extractor.callCount, "extractor must NOT be called when ctx is already cancelled")
}
