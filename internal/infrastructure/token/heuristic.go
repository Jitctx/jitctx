package token

import (
	"context"
	"math"
	"strings"
)

// HeuristicEstimator estimates token count using words * 1.3 heuristic.
type HeuristicEstimator struct{}

// NewHeuristicEstimator creates a new HeuristicEstimator.
func NewHeuristicEstimator() *HeuristicEstimator {
	return &HeuristicEstimator{}
}

// Estimate returns an approximate token count using words * 1.3 (per contract §8).
func (e *HeuristicEstimator) Estimate(_ context.Context, text string) (int, error) {
	words := len(strings.Fields(text))
	return int(math.Round(float64(words) * 1.3)), nil
}
