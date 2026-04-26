package queryuc

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"sort"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	ctxport "github.com/jitctx/jitctx/internal/domain/port/contexts"
	"github.com/jitctx/jitctx/internal/domain/port/manifest"
	"github.com/jitctx/jitctx/internal/domain/port/token"
	"github.com/jitctx/jitctx/internal/domain/service"
	queryvo "github.com/jitctx/jitctx/internal/domain/vo/query"
)

type Impl struct {
	manifest  manifest.LoadManifestPort
	reader    ctxport.ReadContextBodyPort // reads context file bodies
	estimator token.EstimateTokensPort    // kept for signature compat (unused here)
	logger    *slog.Logger
}

func New(
	m manifest.LoadManifestPort,
	r ctxport.ReadContextBodyPort,
	e token.EstimateTokensPort,
	l *slog.Logger,
) *Impl {
	return &Impl{manifest: m, reader: r, estimator: e, logger: l}
}

func (u *Impl) Execute(ctx context.Context, input queryvo.QueryContextInput) (queryvo.QueryContextOutput, error) {
	// Step 1: ctx guard on entry.
	if err := ctx.Err(); err != nil {
		return queryvo.QueryContextOutput{}, err
	}

	// Step 2: load manifest; propagate ErrManifestNotFound unwrapped.
	state, err := u.manifest.Load(ctx)
	if err != nil {
		return queryvo.QueryContextOutput{}, err
	}

	// Step 3: find module; return typed error when absent.
	module, ok := state.FindModule(input.Module)
	if !ok {
		return queryvo.QueryContextOutput{}, &domerr.ModuleNotFoundError{
			Queried:         input.Module,
			AvailableSorted: service.ModuleIDsSorted(state),
		}
	}

	// Step 4: project effective tag set = module.Tags ∪ state.Stack.Languages.
	effectiveTags := append([]string{}, module.Tags...)
	effectiveTags = append(effectiveTags, state.Stack.Languages...)

	// Step 5: filter contexts using the effective tag set.
	filtered := service.FilterContexts(
		state.Contexts,
		&model.Module{ID: module.ID, Tags: effectiveTags},
		input.Tags,
		input.Types,
		input.FilePath,
	)

	// Step 6: stable sort by Context.ID ascending (defensive determinism).
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ID < filtered[j].ID
	})

	// Step 7: open the filesystem rooted at WorkDir.
	workDir := input.WorkDir
	if workDir == "" {
		workDir = "."
	}
	fsys := os.DirFS(workDir)

	// Step 8: read each context body; log warn and skip on error.
	loaded := make([]queryvo.LoadedContext, 0, len(filtered))
	for _, c := range filtered {
		// Step 8 / Step 5.3: check cancellation inside the loop.
		if err := ctx.Err(); err != nil {
			return queryvo.QueryContextOutput{}, err
		}

		body, readErr := u.reader.ReadContextBody(ctx, fsys.(fs.FS), c.Path)
		if readErr != nil {
			u.logger.Warn("skipping context: body read failed",
				"context_id", c.ID,
				"path", c.Path,
				"error", readErr,
			)
			continue
		}

		loaded = append(loaded, queryvo.LoadedContext{
			ID:            c.ID,
			Type:          c.Type,
			Path:          c.Path,
			Tags:          append([]string(nil), c.Tags...),
			Body:          body,
			TokenEstimate: c.TokenEstimate,
		})
	}

	// Step 9: Budget=0 path — nothing trimmed.
	var trimmed []queryvo.LoadedContext
	total := 0
	for _, lc := range loaded {
		total += lc.TokenEstimate
	}

	// Step 10: build ModuleSummary from module.Contracts.
	summary := queryvo.ModuleSummary{ID: module.ID}
	for _, c := range module.Contracts {
		cs := queryvo.ContractSummary{
			Name:  c.Name,
			Types: append([]string(nil), c.Types...),
		}
		for _, m := range c.Methods {
			cs.Methods = append(cs.Methods, m.Signature)
		}
		summary.Contracts = append(summary.Contracts, cs)
	}

	// Step 11: return the assembled output.
	return queryvo.QueryContextOutput{
		Module:      summary,
		Loaded:      loaded,
		Trimmed:     trimmed,
		TotalTokens: total,
	}, nil
}
