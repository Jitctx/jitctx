package queryuc

import (
	"context"

	queryvo "github.com/jitctx/jitctx/internal/domain/vo/query"
)

type UseCase interface {
	Execute(ctx context.Context, input queryvo.QueryContextInput) (queryvo.QueryContextOutput, error)
}
