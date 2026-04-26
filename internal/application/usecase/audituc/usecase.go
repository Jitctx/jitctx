package audituc

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"sort"
	"strings"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/port/config"
	"github.com/jitctx/jitctx/internal/domain/port/manifest"
	"github.com/jitctx/jitctx/internal/domain/port/parser"
	"github.com/jitctx/jitctx/internal/domain/port/profile"
	"github.com/jitctx/jitctx/internal/domain/service"
	auditvo "github.com/jitctx/jitctx/internal/domain/vo/audit"
)

// Impl satisfies audituc.UseCase.
type Impl struct {
	manifests  manifest.LoadManifestPort
	profiles   profile.DetectProfilePort
	auditRules profile.LoadAuditRulesPort
	walker     parser.WalkJavaFilesPort
	parseFile  parser.ParseJavaFilePort
	listFields parser.ListJavaFieldsPort
	config     config.LoadJitctxConfigPort // EP03US-005
	filter     *service.AuditRuleFilter    // EP03US-005
	evaluator  *service.AuditEvaluator
	logger     *slog.Logger
}

// New creates a new audituc.Impl with all required ports injected.
// The constructor accepts only read-shaped ports (RNF-002 read-only enforcement).
func New(
	manifests manifest.LoadManifestPort,
	profiles profile.DetectProfilePort,
	auditRules profile.LoadAuditRulesPort,
	walker parser.WalkJavaFilesPort,
	parseFile parser.ParseJavaFilePort,
	listFields parser.ListJavaFieldsPort,
	cfg config.LoadJitctxConfigPort, // EP03US-005
	filter *service.AuditRuleFilter, // EP03US-005
	evaluator *service.AuditEvaluator,
	logger *slog.Logger,
) *Impl {
	return &Impl{
		manifests:  manifests,
		profiles:   profiles,
		auditRules: auditRules,
		walker:     walker,
		parseFile:  parseFile,
		listFields: listFields,
		config:     cfg,
		filter:     filter,
		evaluator:  evaluator,
		logger:     logger,
	}
}

// Execute runs the full audit workflow.
//
// Orchestration:
//  1. ctx.Err() guard on entry.
//  2. Load manifest; if missing return ErrManifestNotFound unwrapped so
//     the existing CLI translator in internal/cli/format/errors.go handles it.
//  3. Build fs.FS for source-file reads (read-only).
//  4. Detect the active profile (mirrors scanuc pattern).
//  5. Load audit rules from the active profile; empty rules → clean report.
//     5a. Load .jitctx/config.yaml (EP03US-005); missing file → empty config.
//     5b. Filter disabled rules; stash unknown IDs for the output (EP03US-005).
//  6. Walk Java files via WalkJavaFilesPort.
//  7. For each file: parse via ParseJavaFilePort; on ErrPartialParse skip
//     silently (mirrors scanuc tolerance).
//  8. Resolve module ID per file via longest-prefix match on manifest modules.
//  9. Call AuditEvaluator.EvaluateFile and collect violations.
//  10. Sort violations deterministically by (ModuleID, FilePath, Line, RuleID).
//  11. Build module section list and return AuditProjectOutput.
func (u *Impl) Execute(ctx context.Context, in auditvo.AuditProjectInput) (auditvo.AuditProjectOutput, error) {
	// Step 1: honour context cancellation on entry.
	if err := ctx.Err(); err != nil {
		return auditvo.AuditProjectOutput{}, err
	}

	// Step 2: load manifest. ErrManifestNotFound is returned UNWRAPPED so
	// the existing translator branch in cli/format/errors.go handles it
	// (prints "run jitctx scan first" hint and exits with code 1).
	state, err := u.manifests.Load(ctx)
	if err != nil {
		if errors.Is(err, domerr.ErrManifestNotFound) {
			return auditvo.AuditProjectOutput{}, err
		}
		return auditvo.AuditProjectOutput{}, fmt.Errorf("audit: load manifest: %w", err)
	}

	// Step 3: build read-only fs.FS rooted at WorkDir.
	var fsys fs.FS
	if in.WorkDir == "" || in.WorkDir == "." {
		fsys = os.DirFS(".")
	} else {
		fsys = os.DirFS(in.WorkDir)
	}

	// Step 4: detect active profile (mirrors scanuc pattern).
	prof, err := u.profiles.Detect(ctx, fsys)
	if err != nil {
		return auditvo.AuditProjectOutput{}, err
	}
	if in.ProfileName != "" && in.ProfileName != prof.Name {
		return auditvo.AuditProjectOutput{}, fmt.Errorf(
			"requested profile %q not matched: %w", in.ProfileName, domerr.ErrNoProfileMatch)
	}

	// Step 5: load audit rules. An empty slice is valid (clean-state profile).
	rules, err := u.auditRules.LoadAuditRules(ctx, prof.Name)
	if err != nil {
		return auditvo.AuditProjectOutput{}, fmt.Errorf("audit: load rules: %w", err)
	}

	// Step 5a (EP03US-005): load .jitctx/config.yaml. Missing file → empty config.
	cfg, err := u.config.LoadJitctxConfig(ctx, in.WorkDir)
	if err != nil {
		return auditvo.AuditProjectOutput{}, fmt.Errorf("audit: load config: %w", err)
	}

	// Step 5b (EP03US-005): filter disabled rules. unknown is forwarded to the
	// presentation layer via AuditProjectOutput.UnknownDisabledRules so the
	// CLI can emit one stderr line per entry. Disabling by ID is silent in
	// the report (scenario 1) — the warning targets the developer's config
	// file, not the audit report itself.
	filteredRules, unknownDisabled := u.filter.Filter(rules, cfg.Audit.DisabledRules)

	// Step 6: walk Java files.
	paths, err := u.walker.WalkJavaFiles(ctx, fsys)
	if err != nil {
		return auditvo.AuditProjectOutput{}, fmt.Errorf("audit: walk: %w", err)
	}

	// Step 7–9: parse each file, resolve module ID, evaluate rules.
	moduleByPath := indexModulesByPath(state.Modules)
	var violations []auditvo.AuditViolation
	for _, p := range paths {
		if err := ctx.Err(); err != nil {
			return auditvo.AuditProjectOutput{}, err
		}
		summary, parseErr := u.parseFile.ParseJavaFile(ctx, fsys, p)
		if parseErr != nil {
			if errors.Is(parseErr, domerr.ErrPartialParse) {
				u.logger.Warn("audit: partial parse skipped", "path", p)
				continue
			}
			return auditvo.AuditProjectOutput{}, fmt.Errorf("audit: parse %s: %w", p, parseErr)
		}
		moduleID := moduleByPath(p)
		violations = append(violations, u.evaluator.EvaluateFile(moduleID, summary, filteredRules)...)
	}

	// Step 10: sort violations deterministically (RNF-003).
	sort.SliceStable(violations, func(i, j int) bool {
		return lessViolation(violations[i], violations[j])
	})

	// Step 11: build module section list and return.
	mods := buildModuleReports(state.Modules, violations)

	return auditvo.AuditProjectOutput{
		ProfileName:          prof.Name,
		ManifestPath:         in.ManifestPath,
		Modules:              mods,
		Sintatic:             violations,
		SemanticPlaceholder:  auditvo.SemanticPlaceholder,
		UnknownDisabledRules: unknownDisabled, // EP03US-005
	}, nil
}

// indexModulesByPath returns a closure that resolves a file path to a module
// ID using longest-prefix match against the manifest modules. Files that match
// no module use the synthetic ID "<unmoduled>".
func indexModulesByPath(modules []model.Module) func(path string) string {
	return func(path string) string {
		best := ""
		bestID := "<unmoduled>"
		for _, m := range modules {
			mp := m.Path
			if mp == "" {
				continue
			}
			// Normalise separator to forward-slash for comparison.
			mp = strings.ReplaceAll(mp, "\\", "/")
			p := strings.ReplaceAll(path, "\\", "/")
			if strings.HasPrefix(p, mp) && len(mp) > len(best) {
				best = mp
				bestID = m.ID
			}
		}
		return bestID
	}
}

// lessViolation defines the deterministic sort order (RNF-003):
// (ModuleID, FilePath, Line, RuleID).
func lessViolation(a, b auditvo.AuditViolation) bool {
	if a.ModuleID != b.ModuleID {
		// "<unmoduled>" must sort last.
		if a.ModuleID == "<unmoduled>" {
			return false
		}
		if b.ModuleID == "<unmoduled>" {
			return true
		}
		return a.ModuleID < b.ModuleID
	}
	if a.FilePath != b.FilePath {
		return a.FilePath < b.FilePath
	}
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	return a.RuleID < b.RuleID
}

// buildModuleReports constructs the sorted AuditModuleReport list.
// It includes every module from the manifest that has at least one file
// scanned, plus the synthetic "<unmoduled>" entry when there are violations
// with that module ID. The list is sorted by ModuleID with "<unmoduled>"
// appended last.
func buildModuleReports(modules []model.Module, violations []auditvo.AuditViolation) []auditvo.AuditModuleReport {
	// Collect module IDs that appear in violations.
	seen := make(map[string]struct{})
	for _, v := range violations {
		seen[v.ModuleID] = struct{}{}
	}

	var reports []auditvo.AuditModuleReport
	var hasUnmoduled bool

	// Add manifest modules (sorted by ID in the manifest or re-sorted here).
	// We include all manifest modules regardless of violations — per the plan
	// spec: "modules that have at least one source file scanned".
	// Since we have no per-module file-scan counter, we include all manifest
	// modules plus the synthetic "<unmoduled>" entry if it has violations.
	for _, m := range modules {
		reports = append(reports, auditvo.AuditModuleReport{
			ModuleID: m.ID,
			Path:     m.Path,
		})
		delete(seen, m.ID)
	}

	// Check if "<unmoduled>" appears in violations.
	if _, ok := seen["<unmoduled>"]; ok {
		hasUnmoduled = true
	}

	// Sort manifest-sourced reports by ModuleID.
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].ModuleID < reports[j].ModuleID
	})

	// Append "<unmoduled>" last if needed.
	if hasUnmoduled {
		reports = append(reports, auditvo.AuditModuleReport{
			ModuleID: "<unmoduled>",
			Path:     "",
		})
	}

	return reports
}
