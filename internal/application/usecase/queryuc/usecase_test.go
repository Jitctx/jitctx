package queryuc_test

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	appqueryuc "github.com/jitctx/jitctx/internal/application/usecase/queryuc"
	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/vo"
	queryvo "github.com/jitctx/jitctx/internal/domain/vo/query"
)

// --- Fakes ---

type fakeLoadManifestPort struct {
	load      func(ctx context.Context) (*model.ProjectState, error)
	loadCount int
}

func (f *fakeLoadManifestPort) Load(ctx context.Context) (*model.ProjectState, error) {
	f.loadCount++
	return f.load(ctx)
}

type fakeReadContextBodyPort struct {
	read func(ctx context.Context, fsys fs.FS, path string) (string, error)
}

func (f *fakeReadContextBodyPort) ReadContextBody(ctx context.Context, fsys fs.FS, path string) (string, error) {
	return f.read(ctx, fsys, path)
}

type fakeEstimateTokensPort struct{}

func (fakeEstimateTokensPort) Estimate(_ context.Context, text string) (int, error) {
	return len(text) / 4, nil
}

// --- Test helpers ---

func nopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, &slog.HandlerOptions{Level: slog.LevelError}))
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func zeroBudget(t *testing.T) vo.TokenBudget {
	t.Helper()
	b, err := vo.NewTokenBudget(0)
	require.NoError(t, err)
	return b
}

// stateWithTwoContextsAndContracts returns a ProjectState with one module
// ("user-management") that has two associated contexts and one contract.
func stateWithTwoContextsAndContracts() *model.ProjectState {
	return &model.ProjectState{
		Stack: model.Stack{Languages: []string{"java"}},
		Modules: []model.Module{
			{
				ID:   "user-management",
				Path: "src/user",
				Tags: []string{"user"},
				Contracts: []model.Contract{
					{
						Name: "CreateUserUseCase",
						Type: model.ContractInputPort,
						Methods: []model.Method{
							{Signature: "UserResponse execute(CreateUserCommand cmd)"},
						},
					},
				},
			},
		},
		Contexts: []model.Context{
			{
				ID:     "user-guidelines",
				Module: "user-management",
				Path:   ".jitctx/guidelines/user-guidelines.md",
				Tags:   []string{"user"},
			},
			{
				ID:     "user-scenarios",
				Module: "user-management",
				Path:   ".jitctx/scenarios/user-scenarios.md",
				Tags:   []string{"user"},
			},
		},
	}
}

// --- Tests ---

func TestQueryUseCase_HappyPath(t *testing.T) {
	t.Parallel()

	state := stateWithTwoContextsAndContracts()

	loader := &fakeLoadManifestPort{
		load: func(_ context.Context) (*model.ProjectState, error) { return state, nil },
	}
	reader := &fakeReadContextBodyPort{
		read: func(_ context.Context, _ fs.FS, path string) (string, error) {
			return "body of " + path, nil
		},
	}
	uc := appqueryuc.New(loader, reader, fakeEstimateTokensPort{}, nopLogger())

	out, err := uc.Execute(context.Background(), queryvo.QueryContextInput{
		Module: "user-management",
		Budget: zeroBudget(t),
	})

	require.NoError(t, err)
	require.Len(t, out.Loaded, 2, "expected 2 loaded contexts")

	// All bodies must be populated.
	for _, lc := range out.Loaded {
		require.NotEmpty(t, lc.Body, "loaded context body must not be empty")
	}

	// Module summary must carry contract details.
	require.Equal(t, "user-management", out.Module.ID)
	require.Len(t, out.Module.Contracts, 1)
	require.Equal(t, "CreateUserUseCase", out.Module.Contracts[0].Name)
	require.Equal(t, "input-port", out.Module.Contracts[0].Type)
	require.Equal(t, []string{"UserResponse execute(CreateUserCommand cmd)"}, out.Module.Contracts[0].Methods)
}

func TestQueryUseCase_ModuleNotFound(t *testing.T) {
	t.Parallel()

	state := &model.ProjectState{
		Modules: []model.Module{
			{ID: "billing", Path: "src/billing"},
			{ID: "user-management", Path: "src/user"},
		},
	}

	loader := &fakeLoadManifestPort{
		load: func(_ context.Context) (*model.ProjectState, error) { return state, nil },
	}
	reader := &fakeReadContextBodyPort{
		read: func(_ context.Context, _ fs.FS, _ string) (string, error) {
			return "body", nil
		},
	}
	uc := appqueryuc.New(loader, reader, fakeEstimateTokensPort{}, nopLogger())

	_, err := uc.Execute(context.Background(), queryvo.QueryContextInput{
		Module: "payments",
		Budget: zeroBudget(t),
	})

	require.Error(t, err)

	var mnf *domerr.ModuleNotFoundError
	require.True(t, errors.As(err, &mnf), "expected *domerr.ModuleNotFoundError")
	require.Equal(t, []string{"billing", "user-management"}, mnf.AvailableSorted)
}

func TestQueryUseCase_ManifestNotFound(t *testing.T) {
	t.Parallel()

	loader := &fakeLoadManifestPort{
		load: func(_ context.Context) (*model.ProjectState, error) {
			return nil, domerr.ErrManifestNotFound
		},
	}
	reader := &fakeReadContextBodyPort{
		read: func(_ context.Context, _ fs.FS, _ string) (string, error) {
			return "body", nil
		},
	}
	uc := appqueryuc.New(loader, reader, fakeEstimateTokensPort{}, nopLogger())

	_, err := uc.Execute(context.Background(), queryvo.QueryContextInput{
		Module: "user-management",
		Budget: zeroBudget(t),
	})

	require.True(t, errors.Is(err, domerr.ErrManifestNotFound))
}

func TestQueryUseCase_BodyReadFailure(t *testing.T) {
	t.Parallel()

	state := &model.ProjectState{
		Modules: []model.Module{
			{ID: "user-management", Path: "src/user"},
		},
		Contexts: []model.Context{
			{
				ID:     "ctx-ok",
				Module: "user-management",
				Path:   ".jitctx/guidelines/ok.md",
			},
			{
				ID:     "ctx-fail",
				Module: "user-management",
				Path:   ".jitctx/guidelines/fail.md",
			},
		},
	}

	loader := &fakeLoadManifestPort{
		load: func(_ context.Context) (*model.ProjectState, error) { return state, nil },
	}
	reader := &fakeReadContextBodyPort{
		read: func(_ context.Context, _ fs.FS, path string) (string, error) {
			if path == ".jitctx/guidelines/fail.md" {
				return "", errors.New("read error")
			}
			return "ok body", nil
		},
	}
	uc := appqueryuc.New(loader, reader, fakeEstimateTokensPort{}, nopLogger())

	out, err := uc.Execute(context.Background(), queryvo.QueryContextInput{
		Module: "user-management",
		Budget: zeroBudget(t),
	})

	require.NoError(t, err)
	require.Len(t, out.Loaded, 1, "only the successfully-read context should appear")
	require.Equal(t, "ctx-ok", out.Loaded[0].ID)
}

func TestQueryUseCase_Cancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before invocation

	loader := &fakeLoadManifestPort{
		load: func(_ context.Context) (*model.ProjectState, error) {
			return &model.ProjectState{}, nil
		},
	}
	reader := &fakeReadContextBodyPort{
		read: func(_ context.Context, _ fs.FS, _ string) (string, error) {
			return "body", nil
		},
	}
	uc := appqueryuc.New(loader, reader, fakeEstimateTokensPort{}, nopLogger())

	_, err := uc.Execute(ctx, queryvo.QueryContextInput{
		Module: "user-management",
		Budget: zeroBudget(t),
	})

	require.True(t, errors.Is(err, context.Canceled))
	require.Equal(t, 0, loader.loadCount, "Load must NOT be called when context is already cancelled")
}

func TestQueryUseCase_AppliesToViaStackLanguages(t *testing.T) {
	t.Parallel()

	// Module has empty tags; the context applies_to ["java"];
	// Stack.Languages = ["java"] — so the context should be loaded.
	state := &model.ProjectState{
		Stack: model.Stack{Languages: []string{"java"}},
		Modules: []model.Module{
			{ID: "user-management", Path: "src/user", Tags: []string{}},
		},
		Contexts: []model.Context{
			{
				ID:        "java-conventions",
				AppliesTo: []string{"java"},
				Path:      ".jitctx/guidelines/java-conventions.md",
			},
		},
	}

	loader := &fakeLoadManifestPort{
		load: func(_ context.Context) (*model.ProjectState, error) { return state, nil },
	}
	reader := &fakeReadContextBodyPort{
		read: func(_ context.Context, _ fs.FS, _ string) (string, error) {
			return "java conventions body", nil
		},
	}
	uc := appqueryuc.New(loader, reader, fakeEstimateTokensPort{}, nopLogger())

	out, err := uc.Execute(context.Background(), queryvo.QueryContextInput{
		Module: "user-management",
		Budget: zeroBudget(t),
	})

	require.NoError(t, err)
	require.Len(t, out.Loaded, 1, "java-conventions context must be loaded via stack language match")
	require.Equal(t, "java-conventions", out.Loaded[0].ID)
}
