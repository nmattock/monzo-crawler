package crawler

import (
	"context"
	"errors"
	"testing"
)

type fakeSource struct {
	children map[string][]string
	errs     map[string]error
}

func (f fakeSource) Children(_ context.Context, pageURL string) ([]string, error) {
	if err, ok := f.errs[pageURL]; ok {
		return nil, err
	}
	if children, ok := f.children[pageURL]; ok {
		return children, nil
	}
	return []string{}, nil
}

func intPtr(v int) *int {
	return &v
}

func TestSingleRunner_StopsOnInvalidSeed(t *testing.T) {
	r, err := NewSingleRunner(URLRules{}, fakeSource{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = r.Run(context.Background(), "not-a-url", nil)
	if err == nil {
		t.Fatalf("expected error for invalid seed URL")
	}
	if !errors.Is(err, ErrInvalidSeedURL) {
		t.Fatalf("expected ErrInvalidSeedURL, got %v", err)
	}
}

func TestSingleRunner_SkipsInvalidCandidateAndExternalLinks(t *testing.T) {
	r, err := NewSingleRunner(URLRules{}, fakeSource{
		children: map[string][]string{
			"https://crawlme.monzo.com": {
				"https://crawlme.monzo.com/about",
				"::invalid",
				"https://other.monzo.com/home",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	out, err := r.Run(context.Background(), "https://crawlme.monzo.com", intPtr(1))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	root := out.Results["https://crawlme.monzo.com"]
	if len(root.Links) != 1 {
		t.Fatalf("expected 1 valid descendant link, got %d", len(root.Links))
	}
	if root.Links[0] != "https://crawlme.monzo.com/about" {
		t.Fatalf("unexpected child link: %q", root.Links[0])
	}
}

func TestSingleRunner_UsesVisitedToPreventCycles(t *testing.T) {
	r, err := NewSingleRunner(URLRules{}, fakeSource{
		children: map[string][]string{
			"https://crawlme.monzo.com": {
				"https://crawlme.monzo.com/a",
			},
			"https://crawlme.monzo.com/a": {
				"https://crawlme.monzo.com",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	out, err := r.Run(context.Background(), "https://crawlme.monzo.com", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(out.Visited) != 2 {
		t.Fatalf("expected 2 visited pages, got %d", len(out.Visited))
	}
}

func TestSingleRunner_RespectsMaxDepth(t *testing.T) {
	r, err := NewSingleRunner(URLRules{}, fakeSource{
		children: map[string][]string{
			"https://crawlme.monzo.com": {
				"https://crawlme.monzo.com/a",
			},
			"https://crawlme.monzo.com/a": {
				"https://crawlme.monzo.com/a/deeper",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	out, err := r.Run(context.Background(), "https://crawlme.monzo.com", intPtr(1))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if out.Visited["https://crawlme.monzo.com/a/deeper"] {
		t.Fatalf("expected depth-2 page not to be visited with maxDepth=1")
	}
	if !out.Visited["https://crawlme.monzo.com"] || !out.Visited["https://crawlme.monzo.com/a"] {
		t.Fatalf("expected root and depth-1 page to be visited")
	}
}

func TestSingleRunner_StoresPageErrorWhenChildFetchFails(t *testing.T) {
	fetchErr := errors.New("source failed")
	r, err := NewSingleRunner(URLRules{}, fakeSource{
		errs: map[string]error{
			"https://crawlme.monzo.com": fetchErr,
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	out, err := r.Run(context.Background(), "https://crawlme.monzo.com", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	root := out.Results["https://crawlme.monzo.com"]
	if !errors.Is(root.Err, fetchErr) {
		t.Fatalf("expected page error to be stored in results")
	}
}

func TestSingleRunner_SeedWithNoChildren(t *testing.T) {
	r, err := NewSingleRunner(URLRules{}, fakeSource{
		children: map[string][]string{
			"https://crawlme.monzo.com": {},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	out, err := r.Run(context.Background(), "https://crawlme.monzo.com", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(out.Visited) != 1 {
		t.Fatalf("expected only seed to be visited, got %d", len(out.Visited))
	}
	if !out.Visited["https://crawlme.monzo.com"] {
		t.Fatalf("expected seed to be marked visited")
	}

	root, ok := out.Results["https://crawlme.monzo.com"]
	if !ok {
		t.Fatalf("expected seed page to be present in results")
	}
	if len(root.Links) != 0 {
		t.Fatalf("expected seed to have no links, got %d", len(root.Links))
	}
	if root.Err != nil {
		t.Fatalf("expected no page error, got %v", root.Err)
	}
	if root.Depth != 0 {
		t.Fatalf("expected seed depth to be 0, got %d", root.Depth)
	}
}
