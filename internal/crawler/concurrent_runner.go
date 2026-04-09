package crawler

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"sync"
	"time"
)

// DefaultConcurrentWorkers is the default max simultaneous page scrapes for the multi runner.
const DefaultConcurrentWorkers = 10

type scrapeJob struct {
	URL   string
	Depth int
}

type scrapeResult struct {
	URL            string
	Depth          int
	Candidates     []string
	Err            error
	ScrapeDuration time.Duration
}

// ConcurrentRunner executes page scrapes in parallel with bounded concurrency.
type ConcurrentRunner struct {
	rules       Rules
	source      ChildSource
	concurrency int
	debug       bool
	debugTo     io.Writer
}

// NewConcurrentRunner creates a concurrent runner.
func NewConcurrentRunner(rules Rules, source ChildSource, concurrency int) (*ConcurrentRunner, error) {
	if rules == nil {
		return nil, fmt.Errorf("rules cannot be nil")
	}
	if source == nil {
		return nil, fmt.Errorf("child source cannot be nil")
	}
	if concurrency <= 0 {
		return nil, fmt.Errorf("concurrency must be greater than zero")
	}
	return &ConcurrentRunner{
		rules:       rules,
		source:      source,
		concurrency: concurrency,
		debugTo:     os.Stderr,
	}, nil
}

// SetDebug toggles verbose runner diagnostics.
func (r *ConcurrentRunner) SetDebug(enabled bool) {
	r.debug = enabled
}

// SetDebugOutput sets where debug logs are written.
func (r *ConcurrentRunner) SetDebugOutput(w io.Writer) {
	if w == nil {
		r.debugTo = os.Stderr
		return
	}
	r.debugTo = w
}

// Run crawls from seed URL using concurrent child scraping workers.
func (r *ConcurrentRunner) Run(seedURL string, maxDepth *int) (RunResult, error) {
	normalizedSeed, err := r.rules.Normalize(seedURL)
	if err != nil {
		return RunResult{}, fmt.Errorf("%w: %v", ErrInvalidSeedURL, err)
	}
	seedParsed, err := url.Parse(normalizedSeed)
	if err != nil || seedParsed.Scheme == "" || seedParsed.Host == "" {
		return RunResult{}, fmt.Errorf("%w: %q", ErrInvalidSeedURL, normalizedSeed)
	}

	visited := map[string]bool{}
	enqueued := map[string]bool{normalizedSeed: true}
	results := map[string]PageResult{}
	visitOrder := make([]string, 0)
	var mu sync.Mutex

	out := make(chan scrapeResult, r.concurrency)
	semaphore := make(chan struct{}, r.concurrency)

	launch := func(job scrapeJob) {
		go func() {
			semaphore <- struct{}{}
			startedAt := time.Now()
			candidates, childrenErr := r.source.Children(job.URL)
			<-semaphore
			out <- scrapeResult{
				URL:            job.URL,
				Depth:          job.Depth,
				Candidates:     candidates,
				Err:            childrenErr,
				ScrapeDuration: time.Since(startedAt),
			}
		}()
	}

	active := 1
	launch(scrapeJob{URL: normalizedSeed, Depth: 0})
	r.debugf("starting concurrent crawl seed=%s maxDepth=%v workers=%d", normalizedSeed, depthForDebug(maxDepth), r.concurrency)

	for res := range out {
		r.debugf("result url=%s depth=%d", res.URL, res.Depth)

		childJobs := make([]scrapeJob, 0)
		shouldClose := false

		mu.Lock()
		if visited[res.URL] {
			r.debugf("skip duplicate result url=%s", res.URL)
			active--
			shouldClose = active == 0
			mu.Unlock()
			if shouldClose {
				close(out)
			}
			continue
		}
		visited[res.URL] = true
		visitOrder = append(visitOrder, res.URL)

		page := PageResult{
			Depth:          res.Depth,
			ScrapeDuration: res.ScrapeDuration,
		}
		if res.Err != nil {
			r.debugf("children fetch error url=%s err=%v", res.URL, res.Err)
			page.Err = res.Err
			results[res.URL] = page
			active--
			shouldClose = active == 0
			mu.Unlock()
			if shouldClose {
				close(out)
			}
			continue
		}
		r.debugf("found candidates url=%s count=%d", res.URL, len(res.Candidates))

		for _, candidate := range res.Candidates {
			isDescendant, descendantErr := r.rules.IsDescendant(seedParsed, candidate)
			if descendantErr != nil {
				r.debugf("skip invalid candidate parent=%s candidate=%s err=%v", res.URL, candidate, descendantErr)
				continue
			}
			if !isDescendant {
				r.debugf("skip external candidate parent=%s candidate=%s", res.URL, candidate)
				continue
			}

			normalizedChild, normalizeErr := r.rules.Normalize(candidate)
			if normalizeErr != nil {
				r.debugf("skip candidate normalize failure parent=%s candidate=%s err=%v", res.URL, candidate, normalizeErr)
				continue
			}
			page.Links = append(page.Links, normalizedChild)

			if visited[normalizedChild] || enqueued[normalizedChild] {
				r.debugf("skip enqueue already-seen child=%s", normalizedChild)
				continue
			}
			if maxDepth != nil && res.Depth >= *maxDepth {
				r.debugf("skip enqueue due to depth-limit parentDepth=%d maxDepth=%d child=%s", res.Depth, *maxDepth, normalizedChild)
				continue
			}

			enqueued[normalizedChild] = true
			childJobs = append(childJobs, scrapeJob{
				URL:   normalizedChild,
				Depth: res.Depth + 1,
			})
			r.debugf("enqueue child=%s depth=%d", normalizedChild, res.Depth+1)
		}

		results[res.URL] = page
		active += len(childJobs) - 1
		shouldClose = active == 0
		mu.Unlock()

		for _, childJob := range childJobs {
			launch(childJob)
		}
		if shouldClose {
			close(out)
		}
	}

	return RunResult{
		Visited:    visited,
		Results:    results,
		VisitOrder: visitOrder,
	}, nil
}

func (r *ConcurrentRunner) debugf(format string, args ...any) {
	if !r.debug {
		return
	}
	_, _ = fmt.Fprintf(r.debugTo, "[debug] "+format+"\n", args...)
}
