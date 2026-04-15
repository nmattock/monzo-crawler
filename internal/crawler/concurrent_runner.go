package crawler

import (
	"context"
	"sync"
	"time"
)

// DefaultConcurrentWorkers is the default max simultaneous page scrapes for the multi runner.
const DefaultConcurrentWorkers = 1000

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

// newDispatcher returns a channel that the result loop can send jobs to without
// ever blocking. An internal goroutine buffers jobs in a plain slice and forwards
// them one at a time to the unbuffered jobs channel consumed by workers.
//
// This separates result-loop concerns from worker-dispatch concerns, preventing
// the deadlock that arises when a single goroutine both reads results and sends
// new jobs: if workers fill the result channel the single goroutine blocks on
// jobs<-, and workers block on out<-, and nothing moves.
//
// Closing the returned channel signals that no more jobs will be sent; the
// forwarder drains its internal buffer, closes jobs, then exits.
func newDispatcher(jobs chan<- scrapeJob) chan<- scrapeJob {
	in := make(chan scrapeJob)

	go func() {
		defer close(jobs)
		var buf []scrapeJob

		for {
			if len(buf) == 0 {
				// Nothing buffered: block until a job arrives or in is closed.
				job, ok := <-in
				if !ok {
					return
				}
				buf = append(buf, job)
			}

			// Either send the head job to a worker, or accept the next incoming
			// job — whichever is ready first.
			select {
			case jobs <- buf[0]:
				buf[0] = scrapeJob{} // zero for GC
				buf = buf[1:]
			case job, ok := <-in:
				if !ok {
					// in closed: drain remaining buffer then exit.
					for _, j := range buf {
						jobs <- j
					}
					return
				}
				buf = append(buf, job)
			}
		}
	}()

	return in
}

// ConcurrentRunner executes page scrapes in parallel with bounded concurrency.
type ConcurrentRunner struct {
	rules       Rules
	source      ChildSource
	concurrency int
	debugLogger
	progressReporter
}

// NewConcurrentRunner creates a concurrent runner.
func NewConcurrentRunner(rules Rules, source ChildSource, concurrency int) (*ConcurrentRunner, error) {
	if rules == nil {
		return nil, errNilRules
	}
	if source == nil {
		return nil, errNilSource
	}
	if concurrency <= 0 {
		return nil, errBadConcurrency
	}
	return &ConcurrentRunner{
		rules:       rules,
		source:      source,
		concurrency: concurrency,
		debugLogger: newDebugLogger(),
	}, nil
}

// Run crawls from seed URL using a worker pool. Workers are the only goroutines
// that send on out; out is closed only after workerWG confirms all workers have
// exited, so no send races with close. Cancelling ctx stops the crawl: in-flight
// HTTP requests are cancelled and no new jobs are dispatched.
func (r *ConcurrentRunner) Run(ctx context.Context, seedURL string, maxDepth *int) (RunResult, error) {
	seedParsed, normalizedSeed, err := parseSeed(r.rules, seedURL)
	if err != nil {
		return RunResult{}, err
	}
	r.debugf("starting concurrent crawl seed=%s maxDepth=%s workers=%d", normalizedSeed, depthForDebug(maxDepth), r.concurrency)

	visited := map[string]bool{}
	enqueued := map[string]bool{normalizedSeed: true}
	results := map[string]PageResult{}
	visitOrder := make([]string, 0)
	var mu sync.Mutex

	// filterConfig is built once; alreadySeen closes over visited and enqueued,
	// which are always accessed under mu so reads here are safe.
	fcfg := filterConfig{
		rules:      r.rules,
		seedParsed: seedParsed,
		maxDepth:   maxDepth,
		alreadySeen: func(child string) bool {
			return visited[child] || enqueued[child]
		},
		dbg: r.debugf,
	}

	jobs := make(chan scrapeJob) // unbuffered: synchronous handoff dispatcher → worker
	in := newDispatcher(jobs)
	out := make(chan scrapeResult, r.concurrency) // one slot per worker keeps workers flowing

	var workerWG sync.WaitGroup
	for range r.concurrency {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			for job := range jobs {
				startedAt := time.Now()
				candidates, childrenErr := r.source.Children(ctx, job.URL)
				out <- scrapeResult{
					URL:            job.URL,
					Depth:          job.Depth,
					Candidates:     candidates,
					Err:            childrenErr,
					ScrapeDuration: time.Since(startedAt),
				}
			}
		}()
	}

	// Close out only after every worker has exited so no worker sends on a closed channel.
	go func() { workerWG.Wait(); close(out) }()

	pending := 1
	in <- scrapeJob{URL: normalizedSeed, Depth: 0}

	// closeIn ensures in is closed exactly once whether we finish normally
	// (pending==0) or early due to context cancellation.
	inOpen := true
	closeIn := func() {
		if inOpen {
			inOpen = false
			close(in)
		}
	}

	for res := range out {
		// On cancellation, stop dispatching new jobs; workers will fail their
		// in-flight requests and drain out naturally.
		if ctx.Err() != nil {
			closeIn()
			continue
		}

		r.debugf("result url=%s depth=%d", res.URL, res.Depth)

		mu.Lock()

		if visited[res.URL] {
			r.debugf("skip duplicate result url=%s", res.URL)
			pending--
			done := pending == 0
			mu.Unlock()
			if done {
				closeIn()
			}
			continue
		}
		visited[res.URL] = true
		visitOrder = append(visitOrder, res.URL)
		r.reportProgress(len(visitOrder))

		page := PageResult{Depth: res.Depth, ScrapeDuration: res.ScrapeDuration}

		if res.Err != nil {
			r.debugf("children fetch error url=%s err=%v", res.URL, res.Err)
			page.Err = res.Err
			results[res.URL] = page
			pending--
			done := pending == 0
			mu.Unlock()
			if done {
				closeIn()
			}
			continue
		}
		r.debugf("found candidates url=%s count=%d", res.URL, len(res.Candidates))

		// filterCandidates is called inside the lock so that visited/enqueued reads
		// are consistent; it does no I/O so lock contention is negligible.
		links, toEnqueue := filterCandidates(fcfg, res.URL, res.Depth, res.Candidates)
		page.Links = links
		for _, child := range toEnqueue {
			enqueued[child] = true
		}

		results[res.URL] = page
		pending += len(toEnqueue) - 1
		done := pending == 0
		mu.Unlock()

		for _, child := range toEnqueue {
			in <- scrapeJob{URL: child, Depth: res.Depth + 1}
		}
		if done {
			closeIn()
		}
	}

	if err := ctx.Err(); err != nil {
		return RunResult{}, err
	}
	return RunResult{
		Results:    results,
		VisitOrder: visitOrder,
	}, nil
}
