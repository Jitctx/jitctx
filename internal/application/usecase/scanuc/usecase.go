package scanuc

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"time"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	ctxport "github.com/jitctx/jitctx/internal/domain/port/contexts"
	"github.com/jitctx/jitctx/internal/domain/port/manifest"
	"github.com/jitctx/jitctx/internal/domain/port/parser"
	"github.com/jitctx/jitctx/internal/domain/port/profile"
	"github.com/jitctx/jitctx/internal/domain/port/token"
	"github.com/jitctx/jitctx/internal/domain/service"
	"github.com/jitctx/jitctx/internal/domain/vo"
	scanvo "github.com/jitctx/jitctx/internal/domain/vo/scan"
)

// Impl implements the scanuc.UseCase interface.
type Impl struct {
	profiles              profile.DetectProfilePort
	declarativeClassifier profile.ClassifyDeclarativePort
	walker                parser.WalkJavaFilesPort
	javaParse             parser.ParseJavaFilePort
	ctxDisc               ctxport.DiscoverContextsPort
	ctxRead               ctxport.ReadContextBodyPort
	estimator             token.EstimateTokensPort
	manifest              manifest.SaveManifestPort
	logger                *slog.Logger
}

// New creates a new scanuc.Impl with all required ports.
func New(
	profiles profile.DetectProfilePort,
	declarativeClassifier profile.ClassifyDeclarativePort,
	walker parser.WalkJavaFilesPort,
	javaParse parser.ParseJavaFilePort,
	ctxDisc ctxport.DiscoverContextsPort,
	ctxRead ctxport.ReadContextBodyPort,
	estimator token.EstimateTokensPort,
	mani manifest.SaveManifestPort,
	logger *slog.Logger,
) *Impl {
	return &Impl{
		profiles:              profiles,
		declarativeClassifier: declarativeClassifier,
		walker:                walker,
		javaParse:             javaParse,
		ctxDisc:               ctxDisc,
		ctxRead:               ctxRead,
		estimator:             estimator,
		manifest:              mani,
		logger:                logger,
	}
}

// Execute runs the full scan workflow per contract §5.4.
func (u *Impl) Execute(ctx context.Context, input scanvo.ScanProjectInput) (scanvo.ScanProjectOutput, error) {
	// Step 1: Honor ctx.Err() on entry.
	if err := ctx.Err(); err != nil {
		return scanvo.ScanProjectOutput{}, err
	}

	// Step 2: Construct an fs.FS rooted at input.WorkDir.
	var fsys fs.FS
	if input.WorkDir == "" || input.WorkDir == "." {
		fsys = os.DirFS(".")
	} else {
		fsys = os.DirFS(input.WorkDir)
	}

	// Step 3: Detect profile.
	prof, err := u.profiles.Detect(ctx, fsys)
	if err != nil {
		return scanvo.ScanProjectOutput{}, err
	}

	// Step 3 (continued): Validate requested profile name.
	if input.ProfileName != "" && input.ProfileName != prof.Name {
		return scanvo.ScanProjectOutput{}, fmt.Errorf("requested profile %q not matched: %w",
			input.ProfileName, domerr.ErrNoProfileMatch)
	}

	// Step 4: Log profile selected.
	u.logger.Info(
		fmt.Sprintf("Profile: %s", prof.Name),
		"source", string(prof.Source),
	)

	// Step 5: Walk Java files.
	paths, err := u.walker.WalkJavaFiles(ctx, fsys)
	if err != nil {
		return scanvo.ScanProjectOutput{}, fmt.Errorf("walk java: %w", err)
	}

	// Step 6: Parse each file.
	var summaries []model.JavaFileSummary
	var skipped []string
	for _, p := range paths {
		if err := ctx.Err(); err != nil {
			return scanvo.ScanProjectOutput{}, err
		}
		u.logger.Debug("parsing java file", "path", p)
		s, parseErr := u.javaParse.ParseJavaFile(ctx, fsys, p)
		if parseErr != nil {
			if errors.Is(parseErr, domerr.ErrPartialParse) {
				u.logger.Warn("skipped unparseable file", "path", p, "reason", parseErr)
				skipped = append(skipped, p)
				// Still use the partial summary if it has content.
				if len(s.Declarations) > 0 {
					summaries = append(summaries, s)
				}
				continue
			}
			return scanvo.ScanProjectOutput{}, fmt.Errorf("parse %s: %w", p, parseErr)
		}
		summaries = append(summaries, s)
	}

	// Step 7 & 8: Classify declarations and group into modules.
	if err := ctx.Err(); err != nil {
		return scanvo.ScanProjectOutput{}, err
	}
	modules, err := service.BuildModules(ctx, u.declarativeClassifier, summaries, prof, nil)
	if err != nil {
		return scanvo.ScanProjectOutput{}, fmt.Errorf("build modules: %w", err)
	}

	// Step 9: Detect inter-module dependencies.
	for i := range modules {
		modules[i].Dependencies = service.ResolveDependencies(summaries, modules[i], modules)
	}
	for i := range modules {
		u.logger.Debug("module detected", "module", modules[i].ID, "path", modules[i].Path)
	}

	// Step 10: Discover contexts and compute token estimates.
	ctxs, err := u.ctxDisc.DiscoverContexts(ctx, fsys)
	if err != nil {
		return scanvo.ScanProjectOutput{}, fmt.Errorf("discover contexts: %w", err)
	}
	for i := range ctxs {
		u.logger.Debug("context discovered", "id", ctxs[i].ID, "type", ctxs[i].Type)
		body, readErr := u.ctxRead.ReadContextBody(ctx, fsys, ctxs[i].Path)
		if readErr != nil {
			u.logger.Warn("context read failed", "path", ctxs[i].Path, "reason", readErr)
			ctxs[i].TokenEstimate = 0
			continue
		}
		n, estErr := u.estimator.Estimate(ctx, body)
		if estErr != nil {
			u.logger.Warn("token estimate failed", "path", ctxs[i].Path, "reason", estErr)
			ctxs[i].TokenEstimate = 0
			continue
		}
		ctxs[i].TokenEstimate = n
	}

	// Step 11: Sort and assemble ProjectState.
	// Derive framework name: strip "-hexagonal" suffix for clarity? No — use prof.Name as-is per plan.
	state := &model.ProjectState{
		GeneratedAt: time.Now().UTC().Truncate(time.Second),
		Stack: model.Stack{
			Languages:  []string{string(vo.LanguageJava)},
			Frameworks: []string{prof.Name},
		},
		Modules:  modules,
		Contexts: ctxs,
	}
	service.SortProjectState(state)

	// Step 12: Save manifest.
	if err := u.manifest.Save(ctx, state); err != nil {
		return scanvo.ScanProjectOutput{}, fmt.Errorf("save manifest: %w", domerr.ErrManifestWrite)
	}

	u.logger.Info("scan complete",
		"modules", len(state.Modules),
		"contexts", len(state.Contexts),
		"skipped", len(skipped),
		"manifest", input.ManifestPath,
	)

	// Step 13: Return output.
	return scanvo.ScanProjectOutput{
		ManifestPath: input.ManifestPath,
		ModuleCount:  len(state.Modules),
		ContextCount: len(state.Contexts),
		SkippedFiles: skipped,
	}, nil
}
