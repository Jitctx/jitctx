package audituc

import (
	"context"

	auditvo "github.com/jitctx/jitctx/internal/domain/vo/audit"
)

type UseCase interface {
	Execute(ctx context.Context, input auditvo.AuditProjectInput) (auditvo.AuditProjectOutput, error)
}
