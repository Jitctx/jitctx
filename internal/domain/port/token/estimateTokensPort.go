package token

import "context"

type EstimateTokensPort interface {
	Estimate(ctx context.Context, text string) (int, error)
}
