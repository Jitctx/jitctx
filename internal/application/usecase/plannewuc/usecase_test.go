package plannewuc_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/application/usecase/plannewuc"
	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/service"
	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
)

// ── fakes ──────────────────────────────────────────────────────────────────

type fakeRenderer struct {
	render func(context.Context, string, string) ([]byte, error)
}

func (f fakeRenderer) Render(ctx context.Context, feat, mod string) ([]byte, error) {
	return f.render(ctx, feat, mod)
}

type fakeWriter struct {
	write func(context.Context, string, []byte) (string, error)
}

func (f fakeWriter) Write(ctx context.Context, p string, b []byte) (string, error) {
	return f.write(ctx, p, b)
}

// ── helpers ────────────────────────────────────────────────────────────────

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func neverCalledRenderer(t *testing.T) fakeRenderer {
	t.Helper()
	return fakeRenderer{render: func(_ context.Context, _, _ string) ([]byte, error) {
		t.Error("renderer should not have been called")
		return nil, nil
	}}
}

func neverCalledWriter(t *testing.T) fakeWriter {
	t.Helper()
	return fakeWriter{write: func(_ context.Context, _ string, _ []byte) (string, error) {
		t.Error("writer should not have been called")
		return "", nil
	}}
}

// ── tests ──────────────────────────────────────────────────────────────────

func TestPlanNewUseCase_HappyPath(t *testing.T) {
	t.Parallel()

	const (
		feature = "create-user"
		module  = "user-management"
		baseDir = "base"
	)

	expectedPath := filepath.Join(baseDir, feature+".md")
	renderedBytes := []byte("ok")

	var writerReceivedPath string
	renderer := fakeRenderer{render: func(_ context.Context, _, _ string) ([]byte, error) {
		return renderedBytes, nil
	}}
	writer := fakeWriter{write: func(_ context.Context, p string, _ []byte) (string, error) {
		writerReceivedPath = p
		return p, nil
	}}

	uc := plannewuc.New(renderer, writer, service.NewSpecPathResolver(), discardLogger())
	out, err := uc.Execute(context.Background(), planvo.NewTemplateInput{
		Feature: feature,
		Module:  module,
		BaseDir: baseDir,
	})

	require.NoError(t, err)
	require.Equal(t, expectedPath, out.Path)
	require.Equal(t, expectedPath, writerReceivedPath)
}

func TestPlanNewUseCase_ResolverRejectsEmptyFeature(t *testing.T) {
	t.Parallel()

	renderer := neverCalledRenderer(t)
	writer := neverCalledWriter(t)

	uc := plannewuc.New(renderer, writer, service.NewSpecPathResolver(), discardLogger())
	_, err := uc.Execute(context.Background(), planvo.NewTemplateInput{
		Feature: "",
		BaseDir: "base",
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "must not be empty")
}

func TestPlanNewUseCase_RendererFails(t *testing.T) {
	t.Parallel()

	renderer := fakeRenderer{render: func(_ context.Context, _, _ string) ([]byte, error) {
		return nil, errors.New("boom")
	}}
	writer := neverCalledWriter(t)

	uc := plannewuc.New(renderer, writer, service.NewSpecPathResolver(), discardLogger())
	_, err := uc.Execute(context.Background(), planvo.NewTemplateInput{
		Feature: "my-feature",
		BaseDir: "base",
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "render template")
}

func TestPlanNewUseCase_WriterReturnsExistsError(t *testing.T) {
	t.Parallel()

	renderer := fakeRenderer{render: func(_ context.Context, _, _ string) ([]byte, error) {
		return []byte("ok"), nil
	}}
	writer := fakeWriter{write: func(_ context.Context, _ string, _ []byte) (string, error) {
		return "", &domerr.SpecFileExistsError{Path: "x.md"}
	}}

	uc := plannewuc.New(renderer, writer, service.NewSpecPathResolver(), discardLogger())
	_, err := uc.Execute(context.Background(), planvo.NewTemplateInput{
		Feature: "my-feature",
		BaseDir: "base",
	})

	require.Error(t, err)

	var spefe *domerr.SpecFileExistsError
	require.True(t, errors.As(err, &spefe))
	require.Equal(t, "x.md", spefe.Path)
}

func TestPlanNewUseCase_WriterReturnsOtherError(t *testing.T) {
	t.Parallel()

	renderer := fakeRenderer{render: func(_ context.Context, _, _ string) ([]byte, error) {
		return []byte("ok"), nil
	}}
	writer := fakeWriter{write: func(_ context.Context, _ string, _ []byte) (string, error) {
		return "", errors.New("disk full")
	}}

	uc := plannewuc.New(renderer, writer, service.NewSpecPathResolver(), discardLogger())
	_, err := uc.Execute(context.Background(), planvo.NewTemplateInput{
		Feature: "my-feature",
		BaseDir: "base",
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "write template")
}

func TestPlanNewUseCase_CtxCancelledBeforeResolve(t *testing.T) {
	t.Parallel()

	renderer := neverCalledRenderer(t)
	writer := neverCalledWriter(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	uc := plannewuc.New(renderer, writer, service.NewSpecPathResolver(), discardLogger())
	_, err := uc.Execute(ctx, planvo.NewTemplateInput{
		Feature: "my-feature",
		BaseDir: "base",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled))
}
