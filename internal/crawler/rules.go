package crawler

import "net/url"

// Rules defines URL business rules used by the crawler.
// Different implementations can support different crawl policies.
type Rules interface {
	Normalize(rawURL string) (string, error)
	// IsDescendant reports whether candidateURL belongs to the same origin as
	// normalizedSeedURL. It also returns the normalized form of the candidate so
	// callers avoid a second Normalize call on the same URL.
	IsDescendant(normalizedSeedURL *url.URL, candidateURL string) (bool, string, error)
}
