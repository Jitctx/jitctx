package scanuc

import (
	"context"

	scanvo "github.com/jitctx/jitctx/internal/domain/vo/scan"
)

type UseCase interface {
	Execute(ctx context.Context, input scanvo.ScanProjectInput) (scanvo.ScanProjectOutput, error)
}
