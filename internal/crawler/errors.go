package crawler

import "errors"

var (
	// ErrInvalidSeedURL indicates the seed URL is invalid and crawling should stop.
	ErrInvalidSeedURL = errors.New("invalid seed URL")
	// ErrInvalidCandidateURL indicates a child URL is invalid and can be skipped.
	ErrInvalidCandidateURL = errors.New("invalid candidate URL")
)
