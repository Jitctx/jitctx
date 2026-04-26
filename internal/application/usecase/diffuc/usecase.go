package diffuc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/port/manifest"
	"github.com/jitctx/jitctx/internal/domain/port/spec"
	"github.com/jitctx/jitctx/internal/domain/service"
	domdiffuc "github.com/jitctx/jitctx/internal/domain/usecase/diffuc"
	diffvo "github.com/jitctx/jitctx/internal/domain/vo/diff"
)

// Impl satisfies diffuc.UseCase.
type Impl struct {
	finder    spec.FindSpecFilePort
	parser    spec.ParseSpecPort
	manifests manifest.LoadManifestPort
	differ    service.ContractDiffer
	layerer   service.DependencyLayerer
	logger    *slog.Logger
}

// New wires the diff use case with READ-ONLY ports only (RNF-002).
func New(
	finder spec.FindSpecFilePort,
	parser spec.ParseSpecPort,
	manifests manifest.LoadManifestPort,
	differ service.ContractDiffer,
	layerer service.DependencyLayerer,
	logger *slog.Logger,
) *Impl {
	return &Impl{
		finder:    finder,
		parser:    parser,
		manifests: manifests,
		differ:    differ,
		layerer:   layerer,
		logger:    logger,
	}
}

var _ domdiffuc.UseCase = (*Impl)(nil)

// Execute runs the full diff pipeline; see port.go for the step list.
func (u *Impl) Execute(ctx context.Context, in diffvo.DiffPlanInput) (diffvo.DiffPlanOutput, error) {
	// Step 1: ctx.Err() guard on entry.
	if err := ctx.Err(); err != nil {
		return diffvo.DiffPlanOutput{}, err
	}

	// Step 2: Validate exclusivity between Feature and FilePath.
	if in.Feature == "" && in.FilePath == "" {
		return diffvo.DiffPlanOutput{}, errors.New("either --feature or --file is required")
	}
	if in.Feature != "" && in.FilePath != "" {
		return diffvo.DiffPlanOutput{}, errors.New("--feature and --file are mutually exclusive")
	}

	// Step 3: Resolve & read spec.
	var (
		path    string
		content []byte
		alts    []string
	)
	if in.FilePath != "" {
		var err error
		content, err = os.ReadFile(in.FilePath)
		if err != nil {
			return diffvo.DiffPlanOutput{}, fmt.Errorf("read spec file %s: %w", in.FilePath, err)
		}
		path = in.FilePath
	} else {
		var err error
		path, content, alts, err = u.finder.Find(ctx, in.Feature, in.BaseDir, in.PlansDir)
		if err != nil {
			var nf *domerr.SpecFileNotFoundError
			if errors.As(err, &nf) {
				return diffvo.DiffPlanOutput{}, err
			}
			return diffvo.DiffPlanOutput{}, fmt.Errorf("find spec file: %w", err)
		}
	}

	// Step 4: Parse spec via u.parser.ParseSpec.
	parsed, warns, err := u.parser.ParseSpec(ctx, string(content))
	if err != nil {
		return diffvo.DiffPlanOutput{}, fmt.Errorf("parse spec %s: %w", path, err)
	}
	for _, w := range warns {
		u.logger.Warn("spec warning", slog.Int("line", w.Line), slog.String("msg", w.Message))
	}
	for _, alt := range alts {
		u.logger.Warn("spec found in additional location", slog.String("primary", path), slog.String("additional", alt))
	}

	// Step 5: Load manifest. Propagate ErrManifestNotFound unwrapped.
	state, err := u.manifests.Load(ctx)
	if err != nil {
		if errors.Is(err, domerr.ErrManifestNotFound) {
			return diffvo.DiffPlanOutput{}, err
		}
		return diffvo.DiffPlanOutput{}, fmt.Errorf("diff: load manifest: %w", err)
	}

	// Step 6: Flatten manifest contracts into a single name-keyed slice.
	flatManifest := flattenContracts(state, u.logger)

	// Step 7: Compute diff via u.differ.Diff.
	actions := u.differ.Diff(parsed.Contracts, flatManifest)

	// Step 8: Layer the CREATE/MODIFY subset.
	actions, err = u.assignLayers(ctx, parsed.Contracts, actions)
	if err != nil {
		return diffvo.DiffPlanOutput{}, err
	}

	// Step 9: Sort all actions deterministically per §2.3 ordering policy.
	sortActions(actions)

	// Step 10: Set HasChanges = len(actions) > 0.
	hasChanges := len(actions) > 0

	// Step 11: Return DiffPlanOutput.
	return diffvo.DiffPlanOutput{
		Feature:    parsed.Feature,
		Module:     parsed.Module,
		Actions:    actions,
		HasChanges: hasChanges,
	}, nil
}

// flattenContracts concatenates all module contracts into a single slice.
// On duplicate name (EP-01 invariant should prevent this), logs a warning
// and keeps the first occurrence.
func flattenContracts(state *model.ProjectState, logger *slog.Logger) []model.Contract {
	seen := make(map[string]bool)
	var result []model.Contract
	for _, mod := range state.Modules {
		for _, c := range mod.Contracts {
			if seen[c.Name] {
				logger.Warn("duplicate contract name in manifest; keeping first occurrence",
					slog.String("name", c.Name))
				continue
			}
			seen[c.Name] = true
			result = append(result, c)
		}
	}
	return result
}

// assignLayers calls DependencyLayerer on the CREATE/MODIFY subset and
// reassigns Layer values back into the actions slice. EXTRA actions remain
// at Layer=-1. Cycle errors propagate unwrapped.
func (u *Impl) assignLayers(
	_ context.Context,
	specContracts []model.SpecContract,
	actions []diffvo.DiffAction,
) ([]diffvo.DiffAction, error) {
	// Build a name→SpecContract index for quick lookup.
	specIndex := make(map[string]model.SpecContract, len(specContracts))
	for _, sc := range specContracts {
		specIndex[sc.Name] = sc
	}

	// Collect the subset of spec contracts that produced CREATE or MODIFY.
	var subset []model.SpecContract
	for _, a := range actions {
		if a.Type == diffvo.DiffActionCreate || a.Type == diffvo.DiffActionModify {
			if sc, ok := specIndex[a.ContractName]; ok {
				subset = append(subset, sc)
			}
		}
	}

	if len(subset) == 0 {
		return actions, nil
	}

	layers, externals, err := u.layerer.Layer(subset)
	if err != nil {
		// CycleError propagates unwrapped (translator handles it).
		return nil, err
	}

	// Log externals with the same shape as planuc.
	for _, ex := range externals {
		u.logger.Warn("external reference", slog.String("name", "'"+ex+"'"))
	}

	// Build a name→layerIndex map from the layerer output.
	nameToLayer := make(map[string]int)
	for _, layer := range layers {
		for _, target := range layer.Targets {
			nameToLayer[target.Name] = layer.Index
		}
	}

	// Reassign Layer values in the actions slice.
	for i := range actions {
		if actions[i].Type == diffvo.DiffActionCreate || actions[i].Type == diffvo.DiffActionModify {
			if idx, ok := nameToLayer[actions[i].ContractName]; ok {
				actions[i].Layer = idx
			}
		}
	}

	return actions, nil
}

// sortActions sorts actions in-place per the DiffPlanOutput ordering policy:
//  1. CREATE/MODIFY first, by (Layer ASC, ContractName ASC, ActionType ASC
//     where CREATE < MODIFY).
//  2. EXTRA last, by ContractName ASC.
func sortActions(actions []diffvo.DiffAction) {
	sort.SliceStable(actions, func(i, j int) bool {
		ai, aj := actions[i], actions[j]

		iIsExtra := ai.Type == diffvo.DiffActionExtra
		jIsExtra := aj.Type == diffvo.DiffActionExtra

		// EXTRA actions come last.
		if iIsExtra != jIsExtra {
			return !iIsExtra
		}

		// Both EXTRA: sort by ContractName.
		if iIsExtra {
			return ai.ContractName < aj.ContractName
		}

		// Both CREATE/MODIFY: sort by (Layer, ContractName, ActionType).
		if ai.Layer != aj.Layer {
			return ai.Layer < aj.Layer
		}
		if ai.ContractName != aj.ContractName {
			return ai.ContractName < aj.ContractName
		}
		// CREATE < MODIFY lexicographically.
		return string(ai.Type) < string(aj.Type)
	})
}
