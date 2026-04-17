package cli

import (
	"log/slog"

	appcontractsuc "github.com/jitctx/jitctx/internal/application/usecase/contractsuc"
	appplanuc "github.com/jitctx/jitctx/internal/application/usecase/planuc"
	appqueryuc "github.com/jitctx/jitctx/internal/application/usecase/queryuc"
	appscanuc "github.com/jitctx/jitctx/internal/application/usecase/scanuc"
	"github.com/jitctx/jitctx/internal/cli/command"
	"github.com/jitctx/jitctx/internal/config"
	"github.com/jitctx/jitctx/internal/domain/usecase/contractsuc"
	"github.com/jitctx/jitctx/internal/domain/usecase/planuc"
	"github.com/jitctx/jitctx/internal/domain/usecase/queryuc"
	"github.com/jitctx/jitctx/internal/domain/usecase/scanuc"
	"github.com/jitctx/jitctx/internal/infrastructure/fscontext"
	"github.com/jitctx/jitctx/internal/infrastructure/fsmanifest"
	"github.com/jitctx/jitctx/internal/infrastructure/fsprofile"
	"github.com/jitctx/jitctx/internal/infrastructure/token"
	"github.com/jitctx/jitctx/internal/infrastructure/treesitter"
)

type Deps struct {
	ScanFactory command.ScanUseCaseFactory
	Query       queryuc.UseCase
	Plan        planuc.UseCase
	Contracts   contractsuc.UseCase
	Logger      *slog.Logger
}

func Wire(cfg config.Config, logger *slog.Logger) Deps {
	manifestStore := fsmanifest.New(cfg.ManifestPath)
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

	return Deps{
		ScanFactory: scanFactory,
		Query:       appqueryuc.New(manifestStore, ctxDiscoverer, estimator, logger),
		Plan:        appplanuc.New(manifestStore, logger),
		Contracts:   appcontractsuc.New(manifestStore, tsParser, logger),
		Logger:      logger,
	}
}
