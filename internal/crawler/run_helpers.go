package crawler

import "net/url"

// filterConfig holds the configuration that is fixed for every filterCandidates
// call within a single crawl run. Bundling these avoids an 8-parameter signature.
type filterConfig struct {
	rules       Rules
	seedParsed  *url.URL
	maxDepth    *int
	alreadySeen func(string) bool
	dbg         func(string, ...any)
}

// filterCandidates validates and normalizes raw candidate URLs against domain rules,
// returning two slices:
//   - links: every valid in-domain normalized URL found on the page (stored in PageResult.Links)
//   - toEnqueue: the subset of links not yet seen and within the depth limit
func filterCandidates(cfg filterConfig, pageURL string, depth int, candidates []string) (links []string, toEnqueue []string) {
	for _, candidate := range candidates {
		isDescendant, err := cfg.rules.IsDescendant(cfg.seedParsed, candidate)
		if err != nil {
			cfg.dbg("skip invalid candidate parent=%s candidate=%s err=%v", pageURL, candidate, err)
			continue
		}
		if !isDescendant {
			cfg.dbg("skip external candidate parent=%s candidate=%s", pageURL, candidate)
			continue
		}

		normalizedChild, err := cfg.rules.Normalize(candidate)
		if err != nil {
			cfg.dbg("skip candidate normalize failure parent=%s candidate=%s err=%v", pageURL, candidate, err)
			continue
		}

		links = append(links, normalizedChild)

		if cfg.alreadySeen(normalizedChild) {
			cfg.dbg("skip enqueue already-seen child=%s", normalizedChild)
			continue
		}
		if cfg.maxDepth != nil && depth >= *cfg.maxDepth {
			cfg.dbg("skip enqueue due to depth-limit parentDepth=%d maxDepth=%d child=%s", depth, *cfg.maxDepth, normalizedChild)
			continue
		}

		toEnqueue = append(toEnqueue, normalizedChild)
		cfg.dbg("enqueue child=%s depth=%d", normalizedChild, depth+1)
	}
	return links, toEnqueue
}
