package audituc_test

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	appaudituc "github.com/jitctx/jitctx/internal/application/usecase/audituc"
	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
	auditvo "github.com/jitctx/jitctx/internal/domain/vo/audit"
)

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

type fakeLoadManifestPort struct {
	load func(ctx context.Context) (*model.ProjectState, error)
}

func (f *fakeLoadManifestPort) Load(ctx context.Context) (*model.ProjectState, error) {
	return f.load(ctx)
}

type fakeDetectProfilePort struct {
	detect func(ctx context.Context, fsys fs.FS) (*model.FrameworkProfile, error)
}

func (f *fakeDetectProfilePort) Detect(ctx context.Context, fsys fs.FS) (*model.FrameworkProfile, error) {
	return f.detect(ctx, fsys)
}

type fakeLoadAuditRulesPort struct {
	load func(ctx context.Context, profileName string) ([]model.AuditRule, error)
}

func (f *fakeLoadAuditRulesPort) LoadAuditRules(ctx context.Context, profileName string) ([]model.AuditRule, error) {
	return f.load(ctx, profileName)
}

type fakeWalkJavaFilesPort struct {
	walk func(ctx context.Context, fsys fs.FS) ([]string, error)
}

func (f *fakeWalkJavaFilesPort) WalkJavaFiles(ctx context.Context, fsys fs.FS) ([]string, error) {
	return f.walk(ctx, fsys)
}

type fakeParseJavaFilePort struct {
	parse func(ctx context.Context, fsys fs.FS, path string) (model.JavaFileSummary, error)
}

func (f *fakeParseJavaFilePort) ParseJavaFile(ctx context.Context, fsys fs.FS, path string) (model.JavaFileSummary, error) {
	return f.parse(ctx, fsys, path)
}

type fakeListJavaFieldsPort struct {
	list func(ctx context.Context, fsys fs.FS, path string) (model.JavaFileSummary, error)
}

func (f *fakeListJavaFieldsPort) ListJavaFields(ctx context.Context, fsys fs.FS, path string) (model.JavaFileSummary, error) {
	return f.list(ctx, fsys, path)
}

// stubConfigLoader is an in-memory fake for config.LoadJitctxConfigPort.
// The default zero value returns (model.JitctxConfig{}, nil) — "no overrides".
type stubConfigLoader struct {
	cfg model.JitctxConfig
	err error
}

func (s *stubConfigLoader) LoadJitctxConfig(_ context.Context, _ string) (model.JitctxConfig, error) {
	return s.cfg, s.err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, &slog.HandlerOptions{Level: slog.LevelError}))
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

// buildUseCase wires an Impl with all fakes provided and a default
// stubConfigLoader (empty config, nil error) plus a real AuditRuleFilter.
// listFields is optional; when nil a no-op fake is used.
func buildUseCase(
	loadFn func(context.Context) (*model.ProjectState, error),
	detectFn func(context.Context, fs.FS) (*model.FrameworkProfile, error),
	auditRulesFn func(context.Context, string) ([]model.AuditRule, error),
	walkFn func(context.Context, fs.FS) ([]string, error),
	parseFn func(context.Context, fs.FS, string) (model.JavaFileSummary, error),
) *appaudituc.Impl {
	return buildUseCaseWithConfig(
		loadFn, detectFn, auditRulesFn, walkFn, parseFn,
		&stubConfigLoader{}, // empty config — no disabled rules
	)
}

// buildUseCaseWithConfig is like buildUseCase but accepts an explicit config
// loader stub so tests can inject custom disabled-rules or errors.
func buildUseCaseWithConfig(
	loadFn func(context.Context) (*model.ProjectState, error),
	detectFn func(context.Context, fs.FS) (*model.FrameworkProfile, error),
	auditRulesFn func(context.Context, string) ([]model.AuditRule, error),
	walkFn func(context.Context, fs.FS) ([]string, error),
	parseFn func(context.Context, fs.FS, string) (model.JavaFileSummary, error),
	cfgLoader *stubConfigLoader,
) *appaudituc.Impl {
	listFieldsFake := &fakeListJavaFieldsPort{
		list: func(_ context.Context, _ fs.FS, _ string) (model.JavaFileSummary, error) {
			return model.JavaFileSummary{}, nil
		},
	}
	return appaudituc.New(
		&fakeLoadManifestPort{load: loadFn},
		&fakeDetectProfilePort{detect: detectFn},
		&fakeLoadAuditRulesPort{load: auditRulesFn},
		&fakeWalkJavaFilesPort{walk: walkFn},
		&fakeParseJavaFilePort{parse: parseFn},
		listFieldsFake,
		cfgLoader,
		service.NewAuditRuleFilter(),
		service.NewAuditEvaluator(),
		noopLogger(),
	)
}

// minimalState returns a ProjectState with one module rooted at "src/".
func minimalState(moduleID, modulePath string) *model.ProjectState {
	return &model.ProjectState{
		Modules: []model.Module{
			{ID: moduleID, Path: modulePath},
		},
	}
}

// minimalProfile returns a FrameworkProfile with a single interface_naming rule
// so the evaluator can produce a real violation.
func minimalProfileWithRule() *model.FrameworkProfile {
	return &model.FrameworkProfile{
		Name:      "spring-boot-hexagonal",
		Languages: []string{"java"},
	}
}

// interfaceNamingRule returns a rule that flags interfaces in /port/in/ whose
// name does not end with "UseCase".
func interfaceNamingRule() model.AuditRule {
	return model.AuditRule{
		ID:          "rule-001",
		Kind:        model.AuditKindInterfaceNaming,
		Severity:    model.AuditSeverityError,
		Description: "interface in port/in/ must end with UseCase",
		Suggestion:  "rename {name}",
		Params: map[string]string{
			"path_required": "/port/in/",
			"name_suffix":   "UseCase",
		},
	}
}

// ruleA returns a distinct audit rule with ID "rule-A" for filter-related tests.
func ruleA() model.AuditRule {
	return model.AuditRule{
		ID:          "rule-A",
		Kind:        model.AuditKindInterfaceNaming,
		Severity:    model.AuditSeverityError,
		Description: "rule A",
		Suggestion:  "fix A",
		Params: map[string]string{
			"path_required": "/port/in/",
			"name_suffix":   "UseCase",
		},
	}
}

// ruleB returns a distinct audit rule with ID "rule-B" for filter-related tests.
func ruleB() model.AuditRule {
	return model.AuditRule{
		ID:          "rule-B",
		Kind:        model.AuditKindInterfaceNaming,
		Severity:    model.AuditSeverityError,
		Description: "rule B",
		Suggestion:  "fix B",
		Params: map[string]string{
			"path_required": "/port/in/",
			"name_suffix":   "UseCase",
		},
	}
}

// ---------------------------------------------------------------------------
// Existing tests — behaviour unchanged
// ---------------------------------------------------------------------------

// TestAuditUC_HappyPath: 1 module, 1 file, 1 rule → produces 1 violation in
// the right module bucket. Asserts on Modules list, Sintatic list, and
// SemanticPlaceholder string verbatim.
func TestAuditUC_HappyPath(t *testing.T) {
	t.Parallel()

	const moduleID = "mod-user"
	const modulePath = "src/main/java/com/app/user"
	const filePath = "src/main/java/com/app/user/port/in/Broken.java"

	uc := buildUseCase(
		func(_ context.Context) (*model.ProjectState, error) {
			return minimalState(moduleID, modulePath), nil
		},
		func(_ context.Context, _ fs.FS) (*model.FrameworkProfile, error) {
			return minimalProfileWithRule(), nil
		},
		func(_ context.Context, _ string) ([]model.AuditRule, error) {
			return []model.AuditRule{interfaceNamingRule()}, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{filePath}, nil
		},
		func(_ context.Context, _ fs.FS, path string) (model.JavaFileSummary, error) {
			return model.JavaFileSummary{
				Path:    path,
				Package: "com.app.user.port.in",
				Declarations: []model.JavaDeclaration{
					{
						NodeType: "interface_declaration",
						Name:     "BrokenPort", // does NOT end with "UseCase" → violation
					},
				},
			}, nil
		},
	)

	out, err := uc.Execute(context.Background(), auditvo.AuditProjectInput{
		WorkDir:      t.TempDir(),
		ManifestPath: "project-state.yaml",
	})
	require.NoError(t, err)

	// Modules list must include the one manifest module.
	require.Len(t, out.Modules, 1)
	require.Equal(t, moduleID, out.Modules[0].ModuleID)

	// Sintatic list must carry exactly 1 violation belonging to the right module.
	require.Len(t, out.Sintatic, 1)
	require.Equal(t, "rule-001", out.Sintatic[0].RuleID)
	require.Equal(t, moduleID, out.Sintatic[0].ModuleID)
	require.Equal(t, filePath, out.Sintatic[0].FilePath)

	// SemanticPlaceholder must match the verbatim constant (RNF-005).
	require.Equal(t, auditvo.SemanticPlaceholder, out.SemanticPlaceholder)

	// ManifestPath must be echoed from the input.
	require.Equal(t, "project-state.yaml", out.ManifestPath)
}

// TestAuditUC_ManifestNotFound: LoadManifestPort returns ErrManifestNotFound →
// use case propagates it UNWRAPPED so errors.Is still matches.
func TestAuditUC_ManifestNotFound(t *testing.T) {
	t.Parallel()

	uc := buildUseCase(
		func(_ context.Context) (*model.ProjectState, error) {
			return nil, domerr.ErrManifestNotFound
		},
		func(_ context.Context, _ fs.FS) (*model.FrameworkProfile, error) {
			return minimalProfileWithRule(), nil
		},
		func(_ context.Context, _ string) ([]model.AuditRule, error) {
			return nil, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return nil, nil
		},
		func(_ context.Context, _ fs.FS, _ string) (model.JavaFileSummary, error) {
			return model.JavaFileSummary{}, nil
		},
	)

	_, err := uc.Execute(context.Background(), auditvo.AuditProjectInput{
		WorkDir: t.TempDir(),
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, domerr.ErrManifestNotFound))
}

// TestAuditUC_EmptyRules: profile has no audit_rules → use case still succeeds,
// returns clean output (zero violations, but Sintatic and SemanticPlaceholder
// both populated as non-nil / non-empty).
func TestAuditUC_EmptyRules(t *testing.T) {
	t.Parallel()

	const moduleID = "mod-payments"
	const modulePath = "src/main/java/com/app/payments"

	uc := buildUseCase(
		func(_ context.Context) (*model.ProjectState, error) {
			return minimalState(moduleID, modulePath), nil
		},
		func(_ context.Context, _ fs.FS) (*model.FrameworkProfile, error) {
			return minimalProfileWithRule(), nil
		},
		func(_ context.Context, _ string) ([]model.AuditRule, error) {
			// No rules at all.
			return []model.AuditRule{}, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{"src/main/java/com/app/payments/SomeService.java"}, nil
		},
		func(_ context.Context, _ fs.FS, path string) (model.JavaFileSummary, error) {
			return model.JavaFileSummary{
				Path:    path,
				Package: "com.app.payments",
				Declarations: []model.JavaDeclaration{
					{NodeType: "class_declaration", Name: "SomeService"},
				},
			}, nil
		},
	)

	out, err := uc.Execute(context.Background(), auditvo.AuditProjectInput{
		WorkDir: t.TempDir(),
	})
	require.NoError(t, err)

	// Zero violations → Sintatic has no entries (nil or empty slice — both are valid).
	require.Empty(t, out.Sintatic)

	// SemanticPlaceholder must still be set verbatim.
	require.Equal(t, auditvo.SemanticPlaceholder, out.SemanticPlaceholder)

	// The module still appears in the Modules list.
	require.Len(t, out.Modules, 1)
	require.Equal(t, moduleID, out.Modules[0].ModuleID)
}

// TestAuditUC_PartialParseSkip: parser returns ErrPartialParse for one file →
// use case skips silently (not propagated as error), violations from other
// files still come through.
func TestAuditUC_PartialParseSkip(t *testing.T) {
	t.Parallel()

	const moduleID = "mod-orders"
	const modulePath = "src/main/java/com/app/orders"
	brokenFile := "src/main/java/com/app/orders/BrokenSyntax.java"
	goodFile := "src/main/java/com/app/orders/port/in/GoodPort.java"

	uc := buildUseCase(
		func(_ context.Context) (*model.ProjectState, error) {
			return minimalState(moduleID, modulePath), nil
		},
		func(_ context.Context, _ fs.FS) (*model.FrameworkProfile, error) {
			return minimalProfileWithRule(), nil
		},
		func(_ context.Context, _ string) ([]model.AuditRule, error) {
			return []model.AuditRule{interfaceNamingRule()}, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{brokenFile, goodFile}, nil
		},
		func(_ context.Context, _ fs.FS, path string) (model.JavaFileSummary, error) {
			if path == brokenFile {
				return model.JavaFileSummary{HasErrors: true}, domerr.ErrPartialParse
			}
			// goodFile has an interface that violates the naming rule.
			return model.JavaFileSummary{
				Path:    path,
				Package: "com.app.orders.port.in",
				Declarations: []model.JavaDeclaration{
					{NodeType: "interface_declaration", Name: "BadName"},
				},
			}, nil
		},
	)

	out, err := uc.Execute(context.Background(), auditvo.AuditProjectInput{
		WorkDir: t.TempDir(),
	})
	require.NoError(t, err)

	// The violation from the good file must still appear.
	require.Len(t, out.Sintatic, 1)
	require.Equal(t, goodFile, out.Sintatic[0].FilePath)

	// The broken file must NOT have introduced a violation or an error.
	for _, v := range out.Sintatic {
		require.NotEqual(t, brokenFile, v.FilePath, "violation from broken file must not appear")
	}
}

// TestAuditUC_Determinism: two consecutive runs with the same fakes return
// outputs that compare deeply equal (Sintatic in identical order).
func TestAuditUC_Determinism(t *testing.T) {
	t.Parallel()

	const moduleID = "mod-catalog"
	const modulePath = "src/main/java/com/app/catalog"

	files := []string{
		"src/main/java/com/app/catalog/port/in/AlphaPort.java",
		"src/main/java/com/app/catalog/port/in/BetaPort.java",
	}

	makeParseFn := func() func(context.Context, fs.FS, string) (model.JavaFileSummary, error) {
		return func(_ context.Context, _ fs.FS, path string) (model.JavaFileSummary, error) {
			var name string
			if path == files[0] {
				name = "AlphaPort"
			} else {
				name = "BetaPort"
			}
			return model.JavaFileSummary{
				Path:    path,
				Package: "com.app.catalog.port.in",
				Declarations: []model.JavaDeclaration{
					{NodeType: "interface_declaration", Name: name},
				},
			}, nil
		}
	}

	makeUC := func() *appaudituc.Impl {
		return buildUseCase(
			func(_ context.Context) (*model.ProjectState, error) {
				return minimalState(moduleID, modulePath), nil
			},
			func(_ context.Context, _ fs.FS) (*model.FrameworkProfile, error) {
				return minimalProfileWithRule(), nil
			},
			func(_ context.Context, _ string) ([]model.AuditRule, error) {
				return []model.AuditRule{interfaceNamingRule()}, nil
			},
			func(_ context.Context, _ fs.FS) ([]string, error) {
				return []string{files[0], files[1]}, nil
			},
			makeParseFn(),
		)
	}

	in := auditvo.AuditProjectInput{WorkDir: t.TempDir()}

	out1, err1 := makeUC().Execute(context.Background(), in)
	require.NoError(t, err1)

	out2, err2 := makeUC().Execute(context.Background(), in)
	require.NoError(t, err2)

	require.Equal(t, out1.Sintatic, out2.Sintatic,
		"Sintatic violations must be in identical order across consecutive runs")
	require.Equal(t, out1.Modules, out2.Modules,
		"Modules list must be identical across consecutive runs")
}

// ---------------------------------------------------------------------------
// New behavioural tests — EP03US-005 config + filter integration
// ---------------------------------------------------------------------------

// TestAuditUC_DisabledRuleDropped verifies that a rule whose ID appears in
// audit.disabled_rules is removed before evaluation so no violations from
// that rule appear in the output. Violations from other (non-disabled) rules
// still appear. UnknownDisabledRules is empty when all disabled IDs matched.
func TestAuditUC_DisabledRuleDropped(t *testing.T) {
	t.Parallel()

	const moduleID = "mod-user"
	const modulePath = "src/main/java/com/app/user"
	// Both files are in /port/in/ so both rules can produce violations.
	fileA := "src/main/java/com/app/user/port/in/BrokenA.java"
	fileB := "src/main/java/com/app/user/port/in/BrokenB.java"

	cfgLoader := &stubConfigLoader{
		cfg: model.JitctxConfig{
			Audit: model.JitctxAuditConfig{
				DisabledRules: []string{"rule-A"},
			},
		},
	}

	uc := buildUseCaseWithConfig(
		func(_ context.Context) (*model.ProjectState, error) {
			return minimalState(moduleID, modulePath), nil
		},
		func(_ context.Context, _ fs.FS) (*model.FrameworkProfile, error) {
			return minimalProfileWithRule(), nil
		},
		func(_ context.Context, _ string) ([]model.AuditRule, error) {
			return []model.AuditRule{ruleA(), ruleB()}, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{fileA, fileB}, nil
		},
		func(_ context.Context, _ fs.FS, path string) (model.JavaFileSummary, error) {
			// Every file exposes an interface that triggers every naming rule.
			return model.JavaFileSummary{
				Path:    path,
				Package: "com.app.user.port.in",
				Declarations: []model.JavaDeclaration{
					{NodeType: "interface_declaration", Name: "BrokenPort"},
				},
			}, nil
		},
		cfgLoader,
	)

	out, err := uc.Execute(context.Background(), auditvo.AuditProjectInput{
		WorkDir: t.TempDir(),
	})
	require.NoError(t, err)

	// rule-A was disabled — none of its violations must appear.
	for _, v := range out.Sintatic {
		require.NotEqual(t, "rule-A", v.RuleID, "disabled rule-A must not produce violations")
	}

	// rule-B was NOT disabled — at least one of its violations must appear.
	var ruleB bool
	for _, v := range out.Sintatic {
		if v.RuleID == "rule-B" {
			ruleB = true
		}
	}
	require.True(t, ruleB, "non-disabled rule-B must produce at least one violation")

	// No unknown disabled rules — "rule-A" is present in the profile.
	require.Empty(t, out.UnknownDisabledRules)
}

// TestAuditUC_UnknownDisabledRuleSurfaces verifies scenario 2: when the
// disabled list contains an ID that does not match any rule in the profile,
// the audit still completes successfully and
// output.UnknownDisabledRules contains the unmatched ID.
func TestAuditUC_UnknownDisabledRuleSurfaces(t *testing.T) {
	t.Parallel()

	const moduleID = "mod-user"
	const modulePath = "src/main/java/com/app/user"

	cfgLoader := &stubConfigLoader{
		cfg: model.JitctxConfig{
			Audit: model.JitctxAuditConfig{
				DisabledRules: []string{"nonexistent"},
			},
		},
	}

	uc := buildUseCaseWithConfig(
		func(_ context.Context) (*model.ProjectState, error) {
			return minimalState(moduleID, modulePath), nil
		},
		func(_ context.Context, _ fs.FS) (*model.FrameworkProfile, error) {
			return minimalProfileWithRule(), nil
		},
		func(_ context.Context, _ string) ([]model.AuditRule, error) {
			// Profile does NOT contain "nonexistent".
			return []model.AuditRule{interfaceNamingRule()}, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{}, nil
		},
		func(_ context.Context, _ fs.FS, _ string) (model.JavaFileSummary, error) {
			return model.JavaFileSummary{}, nil
		},
		cfgLoader,
	)

	out, err := uc.Execute(context.Background(), auditvo.AuditProjectInput{
		WorkDir: t.TempDir(),
	})
	require.NoError(t, err)
	require.Equal(t, []string{"nonexistent"}, out.UnknownDisabledRules)
}

// TestAuditUC_MultipleDisabledRules verifies scenario 4: when multiple rule
// IDs are disabled, all of them are removed from evaluation and the output
// contains only violations from rules that were NOT disabled.
func TestAuditUC_MultipleDisabledRules(t *testing.T) {
	t.Parallel()

	const moduleID = "mod-user"
	const modulePath = "src/main/java/com/app/user"
	fileC := "src/main/java/com/app/user/port/in/BrokenC.java"

	// Third rule that will NOT be disabled.
	ruleC := model.AuditRule{
		ID:          "rule-C",
		Kind:        model.AuditKindInterfaceNaming,
		Severity:    model.AuditSeverityError,
		Description: "rule C",
		Suggestion:  "fix C",
		Params: map[string]string{
			"path_required": "/port/in/",
			"name_suffix":   "UseCase",
		},
	}

	cfgLoader := &stubConfigLoader{
		cfg: model.JitctxConfig{
			Audit: model.JitctxAuditConfig{
				DisabledRules: []string{"rule-A", "rule-B"},
			},
		},
	}

	uc := buildUseCaseWithConfig(
		func(_ context.Context) (*model.ProjectState, error) {
			return minimalState(moduleID, modulePath), nil
		},
		func(_ context.Context, _ fs.FS) (*model.FrameworkProfile, error) {
			return minimalProfileWithRule(), nil
		},
		func(_ context.Context, _ string) ([]model.AuditRule, error) {
			return []model.AuditRule{ruleA(), ruleB(), ruleC}, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{fileC}, nil
		},
		func(_ context.Context, _ fs.FS, path string) (model.JavaFileSummary, error) {
			return model.JavaFileSummary{
				Path:    path,
				Package: "com.app.user.port.in",
				Declarations: []model.JavaDeclaration{
					{NodeType: "interface_declaration", Name: "BrokenPort"},
				},
			}, nil
		},
		cfgLoader,
	)

	out, err := uc.Execute(context.Background(), auditvo.AuditProjectInput{
		WorkDir: t.TempDir(),
	})
	require.NoError(t, err)

	// rule-A and rule-B were disabled — no violations from either.
	for _, v := range out.Sintatic {
		require.NotEqual(t, "rule-A", v.RuleID, "disabled rule-A must not produce violations")
		require.NotEqual(t, "rule-B", v.RuleID, "disabled rule-B must not produce violations")
	}

	// rule-C was NOT disabled — its violations must appear.
	var hasRuleC bool
	for _, v := range out.Sintatic {
		if v.RuleID == "rule-C" {
			hasRuleC = true
		}
	}
	require.True(t, hasRuleC, "non-disabled rule-C must produce at least one violation")

	// All disabled IDs matched real rules → UnknownDisabledRules must be empty.
	require.Empty(t, out.UnknownDisabledRules)
}

// TestAuditUC_ConfigLoadErrorPropagated verifies that when the config loader
// returns an error the use case wraps it with the "audit: load config:" prefix
// and returns it to the caller.
func TestAuditUC_ConfigLoadErrorPropagated(t *testing.T) {
	t.Parallel()

	const moduleID = "mod-user"
	const modulePath = "src/main/java/com/app/user"

	sentinel := errors.New("disk on fire")
	cfgLoader := &stubConfigLoader{err: sentinel}

	uc := buildUseCaseWithConfig(
		func(_ context.Context) (*model.ProjectState, error) {
			return minimalState(moduleID, modulePath), nil
		},
		func(_ context.Context, _ fs.FS) (*model.FrameworkProfile, error) {
			return minimalProfileWithRule(), nil
		},
		func(_ context.Context, _ string) ([]model.AuditRule, error) {
			return []model.AuditRule{interfaceNamingRule()}, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return nil, nil
		},
		func(_ context.Context, _ fs.FS, _ string) (model.JavaFileSummary, error) {
			return model.JavaFileSummary{}, nil
		},
		cfgLoader,
	)

	_, err := uc.Execute(context.Background(), auditvo.AuditProjectInput{
		WorkDir: t.TempDir(),
	})
	require.Error(t, err)
	// The error must be wrapped with the "audit: load config:" prefix.
	require.ErrorIs(t, err, sentinel, "original error must be reachable via errors.Is")
	require.Contains(t, err.Error(), "audit: load config:")
}

// TestAuditUC_FilterAlwaysCalled verifies that the filter step is always
// executed — even when DisabledRules is nil/empty — and that it returns all
// rules unchanged with an empty UnknownDisabledRules. This guards against a
// future regression where someone short-circuits past the filter on empty
// input.
func TestAuditUC_FilterAlwaysCalled(t *testing.T) {
	t.Parallel()

	const moduleID = "mod-user"
	const modulePath = "src/main/java/com/app/user"

	// Config with nil DisabledRules — the filter must still run and return
	// the full rule set with empty unknown.
	cfgLoader := &stubConfigLoader{
		cfg: model.JitctxConfig{
			Audit: model.JitctxAuditConfig{DisabledRules: nil},
		},
	}

	uc := buildUseCaseWithConfig(
		func(_ context.Context) (*model.ProjectState, error) {
			return minimalState(moduleID, modulePath), nil
		},
		func(_ context.Context, _ fs.FS) (*model.FrameworkProfile, error) {
			return minimalProfileWithRule(), nil
		},
		func(_ context.Context, _ string) ([]model.AuditRule, error) {
			return []model.AuditRule{interfaceNamingRule()}, nil
		},
		func(_ context.Context, _ fs.FS) ([]string, error) {
			return []string{
				"src/main/java/com/app/user/port/in/BadPort.java",
			}, nil
		},
		func(_ context.Context, _ fs.FS, path string) (model.JavaFileSummary, error) {
			return model.JavaFileSummary{
				Path:    path,
				Package: "com.app.user.port.in",
				Declarations: []model.JavaDeclaration{
					{NodeType: "interface_declaration", Name: "BrokenPort"},
				},
			}, nil
		},
		cfgLoader,
	)

	out, err := uc.Execute(context.Background(), auditvo.AuditProjectInput{
		WorkDir: t.TempDir(),
	})
	require.NoError(t, err)

	// All rules passed through the filter unchanged — violations must appear.
	require.NotEmpty(t, out.Sintatic, "all rules must still be evaluated when DisabledRules is nil")

	// UnknownDisabledRules must be empty (nil disabled list → no unknowns).
	require.Empty(t, out.UnknownDisabledRules)
}
