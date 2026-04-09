package crawler

import (
	"fmt"
	"strings"
)

// NewRunner builds a Runner implementation by name using default multi-runner concurrency.
func NewRunner(name string, rules Rules, source ChildSource) (Runner, error) {
	return NewRunnerWithConcurrency(name, rules, source, 0)
}

// NewRunnerWithConcurrency builds a Runner by name. For the multi runner, concurrency caps
// simultaneous scrapes; if concurrency is <= 0, DefaultConcurrentWorkers is used.
// Other runners ignore concurrency. Intended for tests and programmatic composition, not CLI.
func NewRunnerWithConcurrency(name string, rules Rules, source ChildSource, concurrency int) (Runner, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "multi":
		if concurrency <= 0 {
			concurrency = DefaultConcurrentWorkers
		}
		return NewConcurrentRunner(rules, source, concurrency)
	case "single", "bfs":
		return NewCrawler(rules, source)
	default:
		return nil, fmt.Errorf("unknown runner %q", name)
	}
}
