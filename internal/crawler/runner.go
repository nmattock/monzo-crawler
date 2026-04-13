package crawler

import (
	"fmt"
	"net/url"
	"time"
)

// ChildSource provides candidate child links for a URL.
type ChildSource interface {
	Children(pageURL string) ([]string, error)
}

// Runner defines the crawl execution contract.
// Different implementations can provide different traversal strategies.
type Runner interface {
	Run(seedURL string, maxDepth *int) (RunResult, error)
}

// DebuggableRunner exposes optional debug controls for runner implementations.
type DebuggableRunner interface {
	Runner
	SetDebug(enabled bool)
}

// ProgressableRunner exposes an optional progress hook for runner implementations.
// fn is called with the running count of pages visited each time a new page is recorded.
type ProgressableRunner interface {
	Runner
	SetProgress(fn func(visited int))
}

// PageResult stores crawl output for one visited page.
type PageResult struct {
	Links          []string
	Err            error
	Depth          int
	ScrapeDuration time.Duration
}

// RunResult stores the aggregated crawl state.
type RunResult struct {
	Visited    map[string]bool
	Results    map[string]PageResult
	VisitOrder []string
}

// parseSeed normalizes and validates the seed URL, returning the parsed form
// ready for use in IsDescendant checks.
func parseSeed(rules Rules, rawURL string) (*url.URL, string, error) {
	normalized, err := rules.Normalize(rawURL)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrInvalidSeedURL, err)
	}
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, "", fmt.Errorf("%w: %q", ErrInvalidSeedURL, normalized)
	}
	return parsed, normalized, nil
}

// depthForDebug formats maxDepth for debug log lines.
func depthForDebug(maxDepth *int) string {
	if maxDepth == nil {
		return "unlimited"
	}
	return fmt.Sprintf("%d", *maxDepth)
}
