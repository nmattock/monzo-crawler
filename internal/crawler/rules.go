package crawler

import "net/url"

// Rules defines URL business rules used by the crawler.
// Different implementations can support different crawl policies.
type Rules interface {
	Normalize(rawURL string) (string, error)
	IsDescendant(normalizedSeedURL *url.URL, candidateURL string) (bool, error)
}
