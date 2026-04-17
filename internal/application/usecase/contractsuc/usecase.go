package contractsuc

import (
	"context"
	"log/slog"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/port/manifest"
	"github.com/jitctx/jitctx/internal/domain/port/parser"
	contractsvo "github.com/jitctx/jitctx/internal/domain/vo/contracts"
)

type Impl struct {
	manifest  manifest.LoadManifestPort
	javaParse parser.ParseJavaFilePort
	logger    *slog.Logger
}

func New(m manifest.LoadManifestPort, p parser.ParseJavaFilePort, l *slog.Logger) *Impl {
	return &Impl{manifest: m, javaParse: p, logger: l}
}

func (u *Impl) Execute(ctx context.Context, _ contractsvo.ExtractContractsInput) (contractsvo.ExtractContractsOutput, error) {
	if err := ctx.Err(); err != nil {
		return contractsvo.ExtractContractsOutput{}, err
	}
	return contractsvo.ExtractContractsOutput{}, domerr.ErrNotImplemented
}
