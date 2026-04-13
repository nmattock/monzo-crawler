package crawler

import (
	"sync"
	"time"
)

// DefaultConcurrentWorkers is the default max simultaneous page scrapes for the multi runner.
const DefaultConcurrentWorkers = 100

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

// jobQueue holds URLs waiting to be handed to workers. The result-processing loop
// only calls push — it never blocks on the worker jobs channel. A dedicated dispatcher
// goroutine pulls from this queue and sends on jobs, so backpressure from workers cannot
// stall result handling and deadlock the pipeline (unlike a single goroutine that both
// reads out and writes jobs).
type jobQueue struct {
	mu         sync.Mutex
	cond       *sync.Cond
	q          []scrapeJob
	noMoreJobs bool
}

func newJobQueue() *jobQueue {
	jq := &jobQueue{}
	jq.cond = sync.NewCond(&jq.mu)
	return jq
}

func (jq *jobQueue) push(job scrapeJob) {
	jq.mu.Lock()
	jq.q = append(jq.q, job)
	jq.cond.Signal()
	jq.mu.Unlock()
}

// markNoMoreJobs signals the dispatcher that no further push() calls will happen after
// pending work reaches zero; the dispatcher drains the queue then closes jobs.
func (jq *jobQueue) markNoMoreJobs() {
	jq.mu.Lock()
	jq.noMoreJobs = true
	jq.cond.Broadcast()
	jq.mu.Unlock()
}

// runDispatcher sends jobs to workers until the queue is drained and crawling is finished.
// It is the only goroutine that sends on jobs and the only place that closes jobs.
func (jq *jobQueue) runDispatcher(jobs chan<- scrapeJob) {
	defer close(jobs)

	for {
		jq.mu.Lock()
		for len(jq.q) == 0 && !jq.noMoreJobs {
			jq.cond.Wait()
		}
		if len(jq.q) == 0 {
			jq.mu.Unlock()
			return
		}
		job := jq.q[0]
		jq.q = jq.q[1:]
		jq.mu.Unlock()

		// Blocking here only stalls the dispatcher, not the result loop.
		jobs <- job
	}
}

// ConcurrentRunner executes page scrapes in parallel with bounded concurrency.
type ConcurrentRunner struct {
	rules       Rules
	source      ChildSource
	concurrency int
	debugLogger
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

// Run crawls from seed URL using a worker pool. Workers are the only goroutines that
// send on out; we close out only after workerWG confirms all workers exited, so no
// send races with close. A dispatcher goroutine feeds an unbuffered jobs channel so the
// result loop never blocks on job dispatch — eliminating the coordinator deadlock.
func (r *ConcurrentRunner) Run(seedURL string, maxDepth *int) (RunResult, error) {
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

	jq := newJobQueue()
	jobs := make(chan scrapeJob)                  // unbuffered: synchronous handoff dispatcher → worker
	out := make(chan scrapeResult, r.concurrency) // one slot per worker keeps workers flowing

	var workerWG sync.WaitGroup
	for range r.concurrency {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			for job := range jobs {
				startedAt := time.Now()
				candidates, childrenErr := r.source.Children(job.URL)
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

	go jq.runDispatcher(jobs)

	pending := 1
	jq.push(scrapeJob{URL: normalizedSeed, Depth: 0})

	for res := range out {
		r.debugf("result url=%s depth=%d", res.URL, res.Depth)

		mu.Lock()

		if visited[res.URL] {
			r.debugf("skip duplicate result url=%s", res.URL)
			pending--
			done := pending == 0
			mu.Unlock()
			if done {
				jq.markNoMoreJobs()
			}
			continue
		}
		visited[res.URL] = true
		visitOrder = append(visitOrder, res.URL)

		page := PageResult{Depth: res.Depth, ScrapeDuration: res.ScrapeDuration}

		if res.Err != nil {
			r.debugf("children fetch error url=%s err=%v", res.URL, res.Err)
			page.Err = res.Err
			results[res.URL] = page
			pending--
			done := pending == 0
			mu.Unlock()
			if done {
				jq.markNoMoreJobs()
			}
			continue
		}
		r.debugf("found candidates url=%s count=%d", res.URL, len(res.Candidates))

		// filterCandidates is called inside the lock so that visited/enqueued reads
		// are consistent; it does no I/O so lock contention is negligible.
		links, toEnqueue := filterCandidates(
			r.rules,
			seedParsed,
			res.URL,
			res.Depth,
			res.Candidates,
			maxDepth,
			func(child string) bool { return visited[child] || enqueued[child] },
			r.debugf,
		)
		page.Links = links
		for _, child := range toEnqueue {
			enqueued[child] = true
		}

		results[res.URL] = page
		pending += len(toEnqueue) - 1
		done := pending == 0
		mu.Unlock()

		for _, child := range toEnqueue {
			jq.push(scrapeJob{URL: child, Depth: res.Depth + 1})
		}
		if done {
			jq.markNoMoreJobs()
		}
	}

	return RunResult{
		Visited:    visited,
		Results:    results,
		VisitOrder: visitOrder,
	}, nil
}
