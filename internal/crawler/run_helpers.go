package crawler

import "net/url"

// filterCandidates validates and normalizes raw candidate URLs against domain rules,
// returning two slices:
//   - links: every valid in-domain normalized URL found on the page (stored in PageResult.Links)
//   - toEnqueue: the subset of links not yet seen and within the depth limit
//
// alreadySeen lets each runner supply its own "is this URL already queued or visited?"
// check, since single-threaded and concurrent runners track that state differently.
func filterCandidates(
	rules Rules,
	seedParsed *url.URL,
	pageURL string,
	depth int,
	candidates []string,
	maxDepth *int,
	alreadySeen func(string) bool,
	dbg func(string, ...any),
) (links []string, toEnqueue []string) {
	for _, candidate := range candidates {
		isDescendant, err := rules.IsDescendant(seedParsed, candidate)
		if err != nil {
			dbg("skip invalid candidate parent=%s candidate=%s err=%v", pageURL, candidate, err)
			continue
		}
		if !isDescendant {
			dbg("skip external candidate parent=%s candidate=%s", pageURL, candidate)
			continue
		}

		normalizedChild, err := rules.Normalize(candidate)
		if err != nil {
			dbg("skip candidate normalize failure parent=%s candidate=%s err=%v", pageURL, candidate, err)
			continue
		}

		links = append(links, normalizedChild)

		if alreadySeen(normalizedChild) {
			dbg("skip enqueue already-seen child=%s", normalizedChild)
			continue
		}
		if maxDepth != nil && depth >= *maxDepth {
			dbg("skip enqueue due to depth-limit parentDepth=%d maxDepth=%d child=%s", depth, *maxDepth, normalizedChild)
			continue
		}

		toEnqueue = append(toEnqueue, normalizedChild)
		dbg("enqueue child=%s depth=%d", normalizedChild, depth+1)
	}
	return links, toEnqueue
}
