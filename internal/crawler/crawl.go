package crawler

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"time"
)

// ChildSource provides candidate child links for a URL.
// A concrete scraper implementation can be added later.
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

// Crawler traverses links under the configured rules.
type Crawler struct {
	rules   Rules
	source  ChildSource
	debug   bool
	debugTo io.Writer
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
	Visited   map[string]bool
	Results   map[string]PageResult
	VisitOrder []string
}

type queueItem struct {
	URL   string
	Depth int
}

// NewCrawler creates a crawler with rule and child-source dependencies.
func NewCrawler(rules Rules, source ChildSource) (*Crawler, error) {
	if rules == nil {
		return nil, fmt.Errorf("rules cannot be nil")
	}
	if source == nil {
		return nil, fmt.Errorf("child source cannot be nil")
	}
	return &Crawler{
		rules:   rules,
		source:  source,
		debugTo: os.Stderr,
	}, nil
}

// SetDebug toggles verbose crawler diagnostics.
func (c *Crawler) SetDebug(enabled bool) {
	c.debug = enabled
}

// SetDebugOutput sets where debug logs are written.
// By default logs go to stderr.
func (c *Crawler) SetDebugOutput(w io.Writer) {
	if w == nil {
		c.debugTo = os.Stderr
		return
	}
	c.debugTo = w
}

// Run performs a breadth-first crawl from seedURL.
// maxDepth is optional: nil means unlimited depth.
func (c *Crawler) Run(seedURL string, maxDepth *int) (RunResult, error) {
	normalizedSeed, err := c.rules.Normalize(seedURL)
	if err != nil {
		return RunResult{}, fmt.Errorf("%w: %v", ErrInvalidSeedURL, err)
	}
	c.debugf("starting crawl seed=%s maxDepth=%v", normalizedSeed, depthForDebug(maxDepth))

	seedParsed, err := url.Parse(normalizedSeed)
	if err != nil || seedParsed.Scheme == "" || seedParsed.Host == "" {
		return RunResult{}, fmt.Errorf("%w: %q", ErrInvalidSeedURL, normalizedSeed)
	}

	visited := map[string]bool{}
	results := map[string]PageResult{}
	visitOrder := make([]string, 0)
	queue := []queueItem{{URL: normalizedSeed, Depth: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		c.debugf("dequeue url=%s depth=%d queueRemaining=%d", current.URL, current.Depth, len(queue))

		if visited[current.URL] {
			c.debugf("skip visited url=%s", current.URL)
			continue
		}
		visited[current.URL] = true
		visitOrder = append(visitOrder, current.URL)

		page := PageResult{
			Depth: current.Depth,
		}

		startedAt := time.Now()
		candidates, childrenErr := c.source.Children(current.URL)
		page.ScrapeDuration = time.Since(startedAt)
		if childrenErr != nil {
			c.debugf("children fetch error url=%s err=%v", current.URL, childrenErr)
			page.Err = childrenErr
			results[current.URL] = page
			continue
		}
		c.debugf("found candidates url=%s count=%d", current.URL, len(candidates))

		for _, candidate := range candidates {
			isDescendant, descendantErr := c.rules.IsDescendant(seedParsed, candidate)
			if descendantErr != nil {
				// Invalid candidates are skipped so crawling can continue.
				c.debugf("skip invalid candidate parent=%s candidate=%s err=%v", current.URL, candidate, descendantErr)
				continue
			}
			if !isDescendant {
				c.debugf("skip external candidate parent=%s candidate=%s", current.URL, candidate)
				continue
			}

			normalizedChild, normalizeErr := c.rules.Normalize(candidate)
			if normalizeErr != nil {
				c.debugf("skip candidate normalize failure parent=%s candidate=%s err=%v", current.URL, candidate, normalizeErr)
				continue
			}

			page.Links = append(page.Links, normalizedChild)

			if visited[normalizedChild] {
				c.debugf("skip enqueue already-visited child=%s", normalizedChild)
				continue
			}
			if maxDepth != nil && current.Depth >= *maxDepth {
				c.debugf("skip enqueue due to depth-limit parentDepth=%d maxDepth=%d child=%s", current.Depth, *maxDepth, normalizedChild)
				continue
			}

			queue = append(queue, queueItem{
				URL:   normalizedChild,
				Depth: current.Depth + 1,
			})
			c.debugf("enqueue child=%s depth=%d queueSize=%d", normalizedChild, current.Depth+1, len(queue))
		}

		results[current.URL] = page
	}

	return RunResult{
		Visited:    visited,
		Results:    results,
		VisitOrder: visitOrder,
	}, nil
}

func (c *Crawler) debugf(format string, args ...any) {
	if !c.debug {
		return
	}
	_, _ = fmt.Fprintf(c.debugTo, "[debug] "+format+"\n", args...)
}

func depthForDebug(maxDepth *int) string {
	if maxDepth == nil {
		return "unlimited"
	}
	return fmt.Sprintf("%d", *maxDepth)
}
