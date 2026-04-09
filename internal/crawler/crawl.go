package crawler

import (
	"fmt"
	"net/url"
)

// ChildSource provides candidate child links for a URL.
// A concrete scraper implementation can be added later.
type ChildSource interface {
	Children(pageURL string) ([]string, error)
}

// Crawler traverses links under the configured rules.
type Crawler struct {
	rules  Rules
	source ChildSource
}

// PageResult stores crawl output for one visited page.
type PageResult struct {
	Links []string
	Err   error
	Depth int
}

// RunResult stores the aggregated crawl state.
type RunResult struct {
	Visited map[string]bool
	Results map[string]PageResult
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
		rules:  rules,
		source: source,
	}, nil
}

// Run performs a breadth-first crawl from seedURL.
// maxDepth is optional: nil means unlimited depth.
func (c *Crawler) Run(seedURL string, maxDepth *int) (RunResult, error) {
	normalizedSeed, err := c.rules.Normalize(seedURL)
	if err != nil {
		return RunResult{}, fmt.Errorf("%w: %v", ErrInvalidSeedURL, err)
	}

	seedParsed, err := url.Parse(normalizedSeed)
	if err != nil || seedParsed.Scheme == "" || seedParsed.Host == "" {
		return RunResult{}, fmt.Errorf("%w: %q", ErrInvalidSeedURL, normalizedSeed)
	}

	visited := map[string]bool{}
	results := map[string]PageResult{}
	queue := []queueItem{{URL: normalizedSeed, Depth: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.URL] {
			continue
		}
		visited[current.URL] = true

		page := PageResult{
			Depth: current.Depth,
		}

		candidates, childrenErr := c.source.Children(current.URL)
		if childrenErr != nil {
			page.Err = childrenErr
			results[current.URL] = page
			continue
		}

		for _, candidate := range candidates {
			isDescendant, descendantErr := c.rules.IsDescendant(seedParsed, candidate)
			if descendantErr != nil {
				// Invalid candidates are skipped so crawling can continue.
				continue
			}
			if !isDescendant {
				continue
			}

			normalizedChild, normalizeErr := c.rules.Normalize(candidate)
			if normalizeErr != nil {
				continue
			}

			page.Links = append(page.Links, normalizedChild)

			if visited[normalizedChild] {
				continue
			}
			if maxDepth != nil && current.Depth >= *maxDepth {
				continue
			}

			queue = append(queue, queueItem{
				URL:   normalizedChild,
				Depth: current.Depth + 1,
			})
		}

		results[current.URL] = page
	}

	return RunResult{
		Visited: visited,
		Results: results,
	}, nil
}
