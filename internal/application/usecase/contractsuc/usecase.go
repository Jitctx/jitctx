package contractsuc

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
	domcontractsuc "github.com/jitctx/jitctx/internal/domain/usecase/contractsuc"
	contractsvo "github.com/jitctx/jitctx/internal/domain/vo/contracts"
)

// Compile-time interface conformance check.
var _ domcontractsuc.UseCase = (*Impl)(nil)

// Impl orchestrates the contracts slice use case for EP02US-004.
type Impl struct {
	finder    spec.FindSpecFilePort
	parser    spec.ParseSpecPort
	mapper    service.ContractPathMapper
	describer service.ContractRoleDescriber
	resolver  service.ContractTargetResolver
	manifest  manifest.LoadManifestPort
	logger    *slog.Logger
}

// New constructs a contracts slice use case Impl with all required dependencies.
func New(
	finder spec.FindSpecFilePort,
	parser spec.ParseSpecPort,
	mapper service.ContractPathMapper,
	describer service.ContractRoleDescriber,
	resolver service.ContractTargetResolver,
	manifest manifest.LoadManifestPort,
	logger *slog.Logger,
) *Impl {
	return &Impl{
		finder:    finder,
		parser:    parser,
		mapper:    mapper,
		describer: describer,
		resolver:  resolver,
		manifest:  manifest,
		logger:    logger,
	}
}

// Execute resolves the contract slice for the target file in in.TargetFile.
//
// Flow:
//  1. Validate ctx, input, and resolve the target contract name.
//  2. If --feature or --file was supplied, attempt to find the contract in
//     the spec; on hit, project and return.
//  3. Otherwise (or on spec miss), attempt the manifest fallback.
//  4. On total miss, return a typed ContractTargetNotFoundError.
func (u *Impl) Execute(ctx context.Context, in contractsvo.ExtractContractsInput) (contractsvo.ExtractContractsOutput, error) {
	if err := ctx.Err(); err != nil {
		return contractsvo.ExtractContractsOutput{}, err
	}
	if err := validateInput(in); err != nil {
		return contractsvo.ExtractContractsOutput{}, err
	}
	name, err := u.resolver.Resolve(in.TargetFile)
	if err != nil {
		return contractsvo.ExtractContractsOutput{}, err
	}

	specAttempted := in.Feature != "" || in.FilePath != ""
	if specAttempted {
		out, found, err := u.tryFindInSpec(ctx, in, name)
		if err != nil {
			return contractsvo.ExtractContractsOutput{}, err
		}
		if found {
			return out, nil
		}
		// Fall through to manifest fallback.
	}

	out, found, manifestAttempted, err := u.tryFindInManifest(ctx, name)
	if err != nil {
		return contractsvo.ExtractContractsOutput{}, err
	}
	if found {
		return out, nil
	}

	return contractsvo.ExtractContractsOutput{}, &domerr.ContractTargetNotFoundError{
		TargetFile:       in.TargetFile,
		ContractName:     name,
		SearchedSpec:     specAttempted,
		SearchedManifest: manifestAttempted,
	}
}

// validateInput enforces the use case input invariants.
func validateInput(in contractsvo.ExtractContractsInput) error {
	if in.TargetFile == "" {
		return errors.New("target file path must not be empty")
	}
	if in.Feature != "" && in.FilePath != "" {
		return errors.New("--feature and --file are mutually exclusive")
	}
	return nil
}

// loadSpecContent returns the spec path and content using either the explicit
// --file path or the FindSpecFilePort. alts contains additional matching
// locations (only populated by the finder branch).
func (u *Impl) loadSpecContent(ctx context.Context, in contractsvo.ExtractContractsInput) (path string, content []byte, alts []string, err error) {
	if in.FilePath != "" {
		content, err = os.ReadFile(in.FilePath)
		if err != nil {
			return "", nil, nil, fmt.Errorf("read spec file %s: %w", in.FilePath, err)
		}
		return in.FilePath, content, nil, nil
	}
	path, content, alts, err = u.finder.Find(ctx, in.Feature, in.BaseDir, in.PlansDir)
	if err != nil {
		var snf *domerr.SpecFileNotFoundError
		if errors.As(err, &snf) {
			return "", nil, nil, err
		}
		return "", nil, nil, fmt.Errorf("find spec file: %w", err)
	}
	return path, content, alts, nil
}

// tryFindInSpec loads the spec, parses it, logs warnings/alts, and looks up
// the named contract. found=true means a hit (out is populated); found=false
// with err=nil means the spec was loaded but did not contain the contract.
func (u *Impl) tryFindInSpec(ctx context.Context, in contractsvo.ExtractContractsInput, name string) (contractsvo.ExtractContractsOutput, bool, error) {
	path, content, alts, err := u.loadSpecContent(ctx, in)
	if err != nil {
		return contractsvo.ExtractContractsOutput{}, false, err
	}

	parsed, warns, parseErr := u.parser.ParseSpec(ctx, string(content))
	if parseErr != nil {
		return contractsvo.ExtractContractsOutput{}, false, fmt.Errorf("parse spec %s: %w", path, parseErr)
	}
	for _, w := range warns {
		u.logger.Warn("spec warning",
			slog.Int("line", w.Line),
			slog.String("msg", w.Message))
	}
	for _, alt := range alts {
		u.logger.Warn("spec found in additional location",
			slog.String("primary", path),
			slog.String("additional", alt))
	}

	for _, c := range parsed.Contracts {
		if c.Name != name {
			continue
		}
		target, err := u.projectSpecContract(c)
		if err != nil {
			return contractsvo.ExtractContractsOutput{}, false, err
		}
		related, err := u.collectRelated(c, parsed)
		if err != nil {
			return contractsvo.ExtractContractsOutput{}, false, err
		}
		return contractsvo.ExtractContractsOutput{
			Source:  "spec",
			Target:  target,
			Related: related,
		}, true, nil
	}
	return contractsvo.ExtractContractsOutput{}, false, nil
}

// tryFindInManifest loads project-state.yaml and walks modules looking for a
// contract with the given name. The manifestAttempted return tracks whether
// the load succeeded (false when ErrManifestNotFound), enabling the caller to
// build an accurate ContractTargetNotFoundError.
func (u *Impl) tryFindInManifest(ctx context.Context, name string) (out contractsvo.ExtractContractsOutput, found bool, manifestAttempted bool, err error) {
	state, err := u.manifest.Load(ctx)
	if err != nil {
		if errors.Is(err, domerr.ErrManifestNotFound) {
			return contractsvo.ExtractContractsOutput{}, false, false, nil
		}
		return contractsvo.ExtractContractsOutput{}, false, false, err
	}
	for _, mod := range state.Modules {
		for _, c := range mod.Contracts {
			if c.Name == name {
				return contractsvo.ExtractContractsOutput{
					Source: "manifest",
					Target: u.projectManifestContract(c),
				}, true, true, nil
			}
		}
	}
	return contractsvo.ExtractContractsOutput{}, false, true, nil
}

// projectSpecContract projects a model.SpecContract into a ContractFragment.
func (u *Impl) projectSpecContract(c model.SpecContract) (contractsvo.ContractFragment, error) {
	p, err := u.mapper.Map(c.Type, c.Name)
	if err != nil {
		return contractsvo.ContractFragment{}, err
	}
	return contractsvo.ContractFragment{
		Name:        c.Name,
		Type:        string(c.Type),
		Path:        p,
		Methods:     append([]string(nil), c.Methods...),
		Fields:      append([]string(nil), c.Fields...),
		Uses:        append([]string(nil), c.Uses...),
		Implements:  c.Implements,
		DependsOn:   append([]string(nil), c.DependsOn...),
		Endpoints:   append([]string(nil), c.Endpoints...),
		Annotations: append([]string(nil), c.Annotations...),
		Role:        u.describer.Describe(c),
	}, nil
}

// collectRelated gathers the directly referenced contracts (Uses ∪ {Implements} ∪ DependsOn)
// that exist in the spec, deduplicates and sorts them alphabetically, then projects each.
func (u *Impl) collectRelated(target model.SpecContract, parsed model.FeatureSpec) ([]contractsvo.ContractFragment, error) {
	// Build name → SpecContract index.
	idx := make(map[string]model.SpecContract, len(parsed.Contracts))
	for _, c := range parsed.Contracts {
		idx[c.Name] = c
	}

	// Collect deduplicated references.
	refs := make(map[string]struct{}, 8)
	add := func(n string) {
		if n == "" || n == target.Name {
			return
		}
		refs[n] = struct{}{}
	}
	for _, n := range target.Uses {
		add(n)
	}
	add(target.Implements)
	for _, n := range target.DependsOn {
		add(n)
	}

	// Separate known (in spec) from external (log warning, exclude).
	names := make([]string, 0, len(refs))
	for n := range refs {
		if _, ok := idx[n]; !ok {
			u.logger.Warn("external reference",
				slog.String("from", target.Name),
				slog.String("name", "'"+n+"'"))
			continue
		}
		names = append(names, n)
	}
	sort.Strings(names)

	// Project each into ContractFragment.
	out := make([]contractsvo.ContractFragment, 0, len(names))
	for _, n := range names {
		f, err := u.projectSpecContract(idx[n])
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

// projectManifestContract projects a model.Contract (EP-01 manifest) into a ContractFragment.
// Only Name, Types, Path, and Methods are populated — the manifest does not track
// relational fields (Uses, DependsOn, etc.).
func (u *Impl) projectManifestContract(c model.Contract) contractsvo.ContractFragment {
	methods := make([]string, len(c.Methods))
	for i, m := range c.Methods {
		methods[i] = m.Signature
	}
	return contractsvo.ContractFragment{
		Name:    c.Name,
		Types:   append([]string(nil), c.Types...),
		Path:    c.Path,
		Methods: methods,
		Role:    "Manifest-tracked contract",
	}
}
