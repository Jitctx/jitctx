package vo

import "errors"

type TokenBudget struct {
	max int
}

func NewTokenBudget(max int) (TokenBudget, error) {
	if max < 0 {
		return TokenBudget{}, errors.New("token budget must not be negative")
	}
	return TokenBudget{max: max}, nil
}

func (b TokenBudget) Max() int        { return b.max }
func (b TokenBudget) Unlimited() bool { return b.max == 0 }
