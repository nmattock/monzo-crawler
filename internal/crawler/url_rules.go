package crawler

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// URLRules is the default single-domain URL rules implementation.
type URLRules struct{}

// NewURLRules builds the default URL-based crawler rules.
func NewURLRules() Rules {
	return URLRules{}
}

// Normalize standardizes a URL for policy checks:
// - lowercases host
// - removes default ports (80 for http, 443 for https)
// - strips fragments
func (URLRules) Normalize(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("invalid URL %q", rawURL)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid URL %q", rawURL)
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = normalizeHost(parsed.Scheme, parsed.Host)
	parsed.Fragment = ""

	return parsed.String(), nil
}

// IsDescendant returns true when candidate is in the same normalized origin
// (scheme + host) as the already-normalized seed URL.
func (r URLRules) IsDescendant(normalizedSeedURL *url.URL, candidateURL string) (bool, error) {
	if normalizedSeedURL == nil {
		return false, fmt.Errorf("%w: normalized seed URL cannot be nil", ErrInvalidSeedURL)
	}
	if normalizedSeedURL.Scheme == "" || normalizedSeedURL.Host == "" {
		return false, fmt.Errorf("%w: %q", ErrInvalidSeedURL, normalizedSeedURL.String())
	}

	normalizedCandidate, err := r.Normalize(candidateURL)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrInvalidCandidateURL, err)
	}
	candidateParsed, err := url.Parse(normalizedCandidate)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrInvalidCandidateURL, err)
	}

	return normalizedSeedURL.Scheme == candidateParsed.Scheme &&
		normalizedSeedURL.Host == candidateParsed.Host, nil
}

func normalizeHost(scheme, host string) string {
	hostname := strings.ToLower(host)
	port := ""

	if parsedHost, parsedPort, err := net.SplitHostPort(host); err == nil {
		hostname = strings.ToLower(parsedHost)
		port = parsedPort
	}

	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		return hostname
	}
	if port != "" {
		return net.JoinHostPort(hostname, port)
	}
	return hostname
}
