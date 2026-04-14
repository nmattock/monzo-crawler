package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const defaultUserAgent = "monzo-scraper/0.1"

// HTTPChildSource discovers links by downloading and parsing HTML with net/http and goquery.
//
// Limitations:
// - It only sees links present in the fetched HTML response.
// - It does not execute JavaScript, so dynamically injected links are not discovered.
// - It does not interact with user flows (clicks/forms/infinite scroll).
type HTTPChildSource struct {
	client    *http.Client
	userAgent string
}

// NewHTTPChildSource creates an HTTP-backed ChildSource.
func NewHTTPChildSource(client *http.Client) *HTTPChildSource {
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}
	return &HTTPChildSource{
		client:    client,
		userAgent: defaultUserAgent,
	}
}

// Children fetches pageURL and returns absolute candidate links found in anchor href attributes.
func (s *HTTPChildSource) Children(ctx context.Context, pageURL string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch page: unexpected status %d", resp.StatusCode)
	}

	baseURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	seen := make(map[string]struct{})
	links := make([]string, 0)
	doc.Find("a[href]").Each(func(_ int, sel *goquery.Selection) {
		href, ok := sel.Attr("href")
		if !ok {
			return
		}
		href = strings.TrimSpace(href)
		if href == "" {
			return
		}
		if strings.HasPrefix(href, "#") {
			return
		}
		if strings.HasPrefix(strings.ToLower(href), "mailto:") {
			return
		}
		if strings.HasPrefix(strings.ToLower(href), "tel:") {
			return
		}
		if strings.HasPrefix(strings.ToLower(href), "javascript:") {
			return
		}

		u, parseErr := url.Parse(href)
		if parseErr != nil {
			return
		}
		resolved := baseURL.ResolveReference(u)
		link := resolved.String()
		if _, ok := seen[link]; ok {
			return
		}
		seen[link] = struct{}{}
		links = append(links, link)
	})
	return links, nil
}
