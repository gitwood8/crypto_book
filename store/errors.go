package store

import "errors"

var (
	ErrPortfolioLimitReached = errors.New("portfolio limit reached")
	ErrPortfolioNameExists   = errors.New("portfolio with this name already exists")
)
