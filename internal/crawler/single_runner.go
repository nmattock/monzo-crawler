package crawler

import "time"

// SingleRunner performs a sequential breadth-first crawl.
type SingleRunner struct {
	rules  Rules
	source ChildSource
	debugLogger
	progressReporter
}

type queueItem struct {
	URL   string
	Depth int
}

// NewSingleRunner creates a single-threaded runner.
func NewSingleRunner(rules Rules, source ChildSource) (*SingleRunner, error) {
	if rules == nil {
		return nil, errNilRules
	}
	if source == nil {
		return nil, errNilSource
	}
	return &SingleRunner{
		rules:       rules,
		source:      source,
		debugLogger: newDebugLogger(),
	}, nil
}

// Run performs a breadth-first crawl from seedURL.
// maxDepth is optional: nil means unlimited depth.
func (r *SingleRunner) Run(seedURL string, maxDepth *int) (RunResult, error) {
	seedParsed, normalizedSeed, err := parseSeed(r.rules, seedURL)
	if err != nil {
		return RunResult{}, err
	}
	r.debugf("starting crawl seed=%s maxDepth=%s", normalizedSeed, depthForDebug(maxDepth))

	visited := map[string]bool{}
	results := map[string]PageResult{}
	visitOrder := make([]string, 0)
	queue := []queueItem{{URL: normalizedSeed, Depth: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		r.debugf("dequeue url=%s depth=%d queueRemaining=%d", current.URL, current.Depth, len(queue))

		if visited[current.URL] {
			r.debugf("skip visited url=%s", current.URL)
			continue
		}
		visited[current.URL] = true
		visitOrder = append(visitOrder, current.URL)
		r.reportProgress(len(visitOrder))

		startedAt := time.Now()
		candidates, childrenErr := r.source.Children(current.URL)
		page := PageResult{
			Depth:          current.Depth,
			ScrapeDuration: time.Since(startedAt),
		}
		if childrenErr != nil {
			r.debugf("children fetch error url=%s err=%v", current.URL, childrenErr)
			page.Err = childrenErr
			results[current.URL] = page
			continue
		}
		r.debugf("found candidates url=%s count=%d", current.URL, len(candidates))

		links, toEnqueue := filterCandidates(
			r.rules,
			seedParsed,
			current.URL,
			current.Depth,
			candidates,
			maxDepth,
			func(child string) bool { return visited[child] },
			r.debugf,
		)
		page.Links = links
		for _, child := range toEnqueue {
			queue = append(queue, queueItem{URL: child, Depth: current.Depth + 1})
		}

		results[current.URL] = page
	}

	return RunResult{
		Visited:    visited,
		Results:    results,
		VisitOrder: visitOrder,
	}, nil
}
