package cli

import (
	"log/slog"

	appcontractsuc "github.com/jitctx/jitctx/internal/application/usecase/contractsuc"
	appplannewuc "github.com/jitctx/jitctx/internal/application/usecase/plannewuc"
	appplanuc "github.com/jitctx/jitctx/internal/application/usecase/planuc"
	appqueryuc "github.com/jitctx/jitctx/internal/application/usecase/queryuc"
	appscaffolduc "github.com/jitctx/jitctx/internal/application/usecase/scaffolduc"
	appscanuc "github.com/jitctx/jitctx/internal/application/usecase/scanuc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/config"
	"github.com/jitctx/jitctx/internal/domain/usecase/contractsuc"
	"github.com/jitctx/jitctx/internal/domain/usecase/plannewuc"
	"github.com/jitctx/jitctx/internal/domain/usecase/planuc"
	"github.com/jitctx/jitctx/internal/domain/usecase/queryuc"
	"github.com/jitctx/jitctx/internal/domain/usecase/scaffolduc"
	"github.com/jitctx/jitctx/internal/domain/usecase/scanuc"
	"github.com/jitctx/jitctx/internal/infrastructure/fscontext"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
	"github.com/jitctx/jitctx/internal/infrastructure/fsscaffold"
	"github.com/jitctx/jitctx/internal/infrastructure/fsspec"
	"github.com/jitctx/jitctx/internal/infrastructure/mdspec"
	"github.com/jitctx/jitctx/internal/infrastructure/token"
	"github.com/jitctx/jitctx/internal/infrastructure/treesitter"

	domspecsvc "github.com/jitctx/jitctx/internal/domain/service"
)

type Deps struct {
	ScanFactory command.ScanUseCaseFactory
	Query       queryuc.UseCase
	Plan        planuc.UseCase
	PlanNew     plannewuc.UseCase
	Contracts   contractsuc.UseCase
	Scaffold    scaffolduc.UseCase
	WorkDir     string
	PlansDir    string
	Logger      *slog.Logger
}

func Wire(cfg config.Config, logger *slog.Logger) Deps {
	manifestStore := fsmanifest.New(cfg.ManifestPath)
	// cfg.ProfilesDir is the sole source of framework profiles — there is no
	// bundled fallback. Users copy sample YAMLs from profiles/ in the source
	// repo into their project's .jitctx/profiles/ directory.
	profileDetector := fsprofile.NewDetectorWithLogger(cfg.ProfilesDir, logger)
	tsParser := treesitter.New()
	tsWalker := treesitter.NewWalker()
	ctxDiscoverer := fscontext.New()
	estimator := token.NewHeuristicEstimator()

	scanFactory := func(manifestPath string) scanuc.UseCase {
		store := fsmanifest.New(manifestPath)
		return appscanuc.New(
			profileDetector,
			tsWalker,
			tsParser,
			ctxDiscoverer,
			ctxDiscoverer,
			estimator,
			store,
			logger,
		)
	}

	specRenderer := fsspec.New()
	specWriter := fsspec.NewWriter()
	pathResolver := domspecsvc.NewSpecPathResolver()

	specFinder := fsspec.NewFinder()
	mdParser := mdspec.New()

	scaffoldRegistry := fsscaffold.NewRegistry()
	scaffoldWriter := fsscaffold.NewMultiFileWriter()
	importResolver := domspecsvc.NewJavaImportResolver(domspecsvc.NewContractPathMapper())
	endpointSynth := domspecsvc.NewEndpointSynthesizer()
	idUtils := domspecsvc.NewJavaIdentifierUtils()

	return Deps{
		ScanFactory: scanFactory,
		Query:       appqueryuc.New(manifestStore, ctxDiscoverer, estimator, logger),
		Plan: func() planuc.UseCase {
			layerer := domspecsvc.NewDependencyLayerer()
			mapper := domspecsvc.NewContractPathMapper()
			return appplanuc.New(specFinder, mdParser, layerer, mapper, logger)
		}(),
		PlanNew: appplannewuc.New(specRenderer, specWriter, pathResolver, logger),
		Contracts: appcontractsuc.New(
			specFinder,
			mdParser,
			domspecsvc.NewContractPathMapper(),
			domspecsvc.NewContractRoleDescriber(),
			domspecsvc.NewContractTargetResolver(),
			manifestStore,
			logger,
		),
		Scaffold: appscaffolduc.New(
			specFinder, mdParser,
			domspecsvc.NewContractPathMapper(),
			importResolver, endpointSynth, idUtils,
			scaffoldRegistry, scaffoldWriter,
			logger,
		),
		WorkDir:  cfg.WorkDir,
		PlansDir: cfg.PlansDir,
		Logger:   logger,
	}
}
