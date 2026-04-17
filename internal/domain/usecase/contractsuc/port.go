package contractsuc

import (
	"context"

	contractsvo "github.com/jitctx/jitctx/internal/domain/vo/contracts"
)

type UseCase interface {
	Execute(ctx context.Context, input contractsvo.ExtractContractsInput) (contractsvo.ExtractContractsOutput, error)
}
