package cli

import (
	"log/slog"

	appaudituc "github.com/jitctx/jitctx/internal/application/usecase/audituc"
	appcontractsuc "github.com/jitctx/jitctx/internal/application/usecase/contractsuc"
	appdiffuc "github.com/jitctx/jitctx/internal/application/usecase/diffuc"
	appplannewuc "github.com/jitctx/jitctx/internal/application/usecase/plannewuc"
	appplanuc "github.com/jitctx/jitctx/internal/application/usecase/planuc"
	appprofileinituc "github.com/jitctx/jitctx/internal/application/usecase/profileinituc"
	appqueryuc "github.com/jitctx/jitctx/internal/application/usecase/queryuc"
	apprefactoruc "github.com/jitctx/jitctx/internal/application/usecase/refactoruc"
	appscaffolduc "github.com/jitctx/jitctx/internal/application/usecase/scaffolduc"
	appscanuc "github.com/jitctx/jitctx/internal/application/usecase/scanuc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/config"
	"github.com/jitctx/jitctx/internal/domain/usecase/audituc"
	"github.com/jitctx/jitctx/internal/domain/usecase/contractsuc"
	"github.com/jitctx/jitctx/internal/domain/usecase/diffuc"
	"github.com/jitctx/jitctx/internal/domain/usecase/plannewuc"
	"github.com/jitctx/jitctx/internal/domain/usecase/planuc"
	"github.com/jitctx/jitctx/internal/domain/usecase/profileinituc"
	"github.com/jitctx/jitctx/internal/domain/usecase/queryuc"
	"github.com/jitctx/jitctx/internal/domain/usecase/refactoruc"
	"github.com/jitctx/jitctx/internal/domain/usecase/scaffolduc"
	"github.com/jitctx/jitctx/internal/domain/usecase/scanuc"
	"github.com/jitctx/jitctx/internal/infrastructure/fsconfig"
	"github.com/jitctx/jitctx/internal/infrastructure/fscontext"
	"github.com/jitctx/jitctx/internal/infrastructure/fsgit"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
	"github.com/jitctx/jitctx/internal/infrastructure/fsscaffold"
	"github.com/jitctx/jitctx/internal/infrastructure/fsspec"
	"github.com/jitctx/jitctx/internal/infrastructure/mdspec"
	"github.com/jitctx/jitctx/internal/infrastructure/token"
	"github.com/jitctx/jitctx/internal/infrastructure/treesitter"

	domspecsvc "github.com/jitctx/jitctx/internal/domain/service"

	profileport "github.com/jitctx/jitctx/internal/domain/port/profile"
	specport "github.com/jitctx/jitctx/internal/domain/port/spec"
)

// bundledProfiles composes the two ISP ports backed by *fsprofile.Bundled so
// wiring code can hand the same struct to consumers expecting either port.
type bundledProfiles interface {
	profileport.LoadBundledProfilePort
	profileport.ListBundledProfilesPort
}

type Deps struct {
	ScanFactory command.ScanUseCaseFactory
	Query       queryuc.UseCase
	Plan        planuc.UseCase
	PlanNew     plannewuc.UseCase
	Diff        diffuc.UseCase
	Contracts   contractsuc.UseCase
	Scaffold    scaffolduc.UseCase
	Audit       audituc.UseCase
	Refactor    refactoruc.UseCase
	WorkDir     string
	PlansDir    string
	Logger      *slog.Logger

	// ProfileBundleLoader satisfies profile.LoadProfileBundlePort.
	// Backed by *fsprofile.BundleLoader.
	ProfileBundleLoader profileport.LoadProfileBundlePort

	// BundledProfiles satisfies both profile.LoadBundledProfilePort and
	// profile.ListBundledProfilesPort. Backed by *fsprofile.Bundled.
	BundledProfiles bundledProfiles

	// DeclarativeClassifier satisfies profile.ClassifyDeclarativePort.
	// Backed by *service.DeclarativeClassifier. Exposed so US-003 can
	// inject it into scanuc.Impl without a second wire.go edit.
	DeclarativeClassifier profileport.ClassifyDeclarativePort

	// ProfilesDir is plumbed through so the profile init command can
	// resolve <WorkDir>/<ProfilesDir>/<name>/ without re-reading
	// config. Mirrors PlansDir/WorkDir convention.
	ProfilesDir string

	// ProfileResolver satisfies profile.ResolveProfilePort. Backed by
	// *fsprofile.Resolver. Consumed by scanuc and by the profile init
	// use case (for the existence check via list).
	ProfileResolver profileport.ResolveProfilePort

	// ProfileExtractor satisfies profile.ExtractBundledProfilePort.
	// Backed by *fsprofile.Extractor. Consumed by the profile init use case.
	ProfileExtractor profileport.ExtractBundledProfilePort

	// InitProfile is the profile init use case. Consumed by the new
	// "profile init" cobra subcommand.
	InitProfile profileinituc.UseCase

	// BundleAuditRulesLoader satisfies profile.LoadBundleAuditRulesPort.
	// Backed by *fsprofile.BundleAuditRulesAdapter. EP04US-004.
	BundleAuditRulesLoader profileport.LoadBundleAuditRulesPort

	// BundleScaffoldRenderer satisfies spec.RenderBundleProductionTemplatePort.
	// Backed by *fsscaffold.TemplateRegistry's RenderWithBundle method.
	// EP04US-004.
	BundleScaffoldRenderer specport.RenderBundleProductionTemplatePort

	// BundleScaffoldTestRenderer satisfies spec.RenderBundleTestTemplatePort.
	// Backed by *fsscaffold.TestTemplateRegistry's RenderWithBundleTest method.
	// EP04US-004.
	BundleScaffoldTestRenderer specport.RenderBundleTestTemplatePort
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
	declarativeClassifier := domspecsvc.NewDeclarativeClassifier()

	profileBundleLoader := fsprofile.NewBundleLoader(logger)
	bundled := fsprofile.NewBundled()
	profileResolver := fsprofile.NewResolver(profileBundleLoader, bundled, logger)
	profileExtractor := fsprofile.NewExtractor()

	initProfileUC := appprofileinituc.New(
		bundled,          // ListBundledProfilesPort (also satisfies LoadBundledProfilePort)
		profileExtractor, // ExtractBundledProfilePort
		logger,
	)

	scanFactory := func(manifestPath string) scanuc.UseCase {
		store := fsmanifest.New(manifestPath)
		return appscanuc.New(
			profileResolver, // CHANGED — was profileDetector
			declarativeClassifier,
			tsWalker,
			tsParser,
			ctxDiscoverer,
			ctxDiscoverer,
			estimator,
			store,
			cfg.ProfilesDir, // NEW — the resolver needs to know the profiles dir
			logger,
		)
	}

	specRenderer := fsspec.New()
	specWriter := fsspec.NewWriter()
	pathResolver := domspecsvc.NewSpecPathResolver()

	specFinder := fsspec.NewFinder()
	mdParser := mdspec.New()

	scaffoldRegistry := fsscaffold.NewRegistry()
	scaffoldTestRegistry := fsscaffold.NewTestRegistry()
	scaffoldWriter := fsscaffold.NewMultiFileWriter()
	importResolver := domspecsvc.NewJavaImportResolver(domspecsvc.NewContractPathMapper())
	endpointSynth := domspecsvc.NewEndpointSynthesizer()
	idUtils := domspecsvc.NewJavaIdentifierUtils()
	testMapper := domspecsvc.NewTestPathMapper()
	methodParser := domspecsvc.NewMethodSignatureParser()
	jpaAnnotator := domspecsvc.NewJPAFieldAnnotator()

	auditRulesLoader := fsprofile.NewAuditRulesLoader(cfg.ProfilesDir, logger)
	bundleAuditRulesLoader := fsprofile.NewBundleAuditRulesAdapter() // EP04US-004
	auditEvaluator := domspecsvc.NewAuditEvaluator()

	jitctxConfigLoader := fsconfig.New(logger)
	auditFilter := domspecsvc.NewAuditRuleFilter()

	layerer := domspecsvc.NewDependencyLayerer()
	differ := domspecsvc.NewContractDiffer(domspecsvc.NewSignatureNormalizer())

	markerParser := domspecsvc.NewMarkerParser()

	return Deps{
		ScanFactory: scanFactory,
		Query:       appqueryuc.New(manifestStore, ctxDiscoverer, estimator, logger),
		Plan: func() planuc.UseCase {
			mapper := domspecsvc.NewContractPathMapper()
			return appplanuc.New(specFinder, mdParser, layerer, mapper, logger)
		}(),
		PlanNew: appplannewuc.New(specRenderer, specWriter, pathResolver, logger),
		Diff:    appdiffuc.New(specFinder, mdParser, manifestStore, differ, layerer, logger),
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
			specFinder,
			mdParser,
			domspecsvc.NewContractPathMapper(),
			testMapper,
			importResolver,
			endpointSynth,
			idUtils,
			methodParser,
			jpaAnnotator,
			scaffoldRegistry,
			scaffoldTestRegistry,
			scaffoldWriter,
			logger,
		),
		Audit: appaudituc.New(
			manifestStore,      // manifest.LoadManifestPort
			profileDetector,    // profile.DetectProfilePort
			auditRulesLoader,   // profile.LoadAuditRulesPort
			tsWalker,           // parser.WalkJavaFilesPort
			tsParser,           // parser.ParseJavaFilePort
			tsParser,           // parser.ListJavaFieldsPort (same *Parser satisfies both)
			jitctxConfigLoader, // config.LoadJitctxConfigPort (EP03US-005)
			auditFilter,        // *service.AuditRuleFilter (EP03US-005)
			auditEvaluator,
			logger,
			bundleAuditRulesLoader, // profile.LoadBundleAuditRulesPort (EP04US-004)
			profileResolver,        // profile.ResolveProfilePort (EP04US-004)
			cfg.ProfilesDir,        // profilesDir (EP04US-004)
		),
		Refactor: func() refactoruc.UseCase {
			gitReader := fsgit.New(logger)
			return apprefactoruc.New(
				manifestStore,          // manifest.LoadManifestPort — same instance as audit/diff
				tsWalker,               // parser.WalkJavaFilesPort — same instance as audit
				tsParser,               // parser.ListJavaCommentsPort — same *Parser satisfies multiple ports
				gitReader,              // git.FileLastModifiedTimePort
				gitReader.LineReader(), // git.LineIntroducedTimePort
				markerParser,
				logger,
			)
		}(),
		WorkDir:                    cfg.WorkDir,
		PlansDir:                   cfg.PlansDir,
		Logger:                     logger,
		ProfileBundleLoader:        profileBundleLoader,
		BundledProfiles:            bundled,
		DeclarativeClassifier:      declarativeClassifier,
		ProfilesDir:                cfg.ProfilesDir,
		ProfileResolver:            profileResolver,
		ProfileExtractor:           profileExtractor,
		InitProfile:                initProfileUC,
		BundleAuditRulesLoader:     bundleAuditRulesLoader, // EP04US-004
		BundleScaffoldRenderer:     scaffoldRegistry,       // EP04US-004
		BundleScaffoldTestRenderer: scaffoldTestRegistry,   // EP04US-004
	}
}
