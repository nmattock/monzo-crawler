package crawler

import (
	"errors"
	"net/url"
	"testing"
)

func TestNormalize_LowercasesHost(t *testing.T) {
	rules := URLRules{}

	got, err := rules.Normalize("https://CRAWLME.MONZO.COM/Docs")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := "https://crawlme.monzo.com/Docs"
	if got != want {
		t.Fatalf("unexpected normalized URL: got %q, want %q", got, want)
	}
}

func TestNormalize_RemovesDefaultPortAndFragment(t *testing.T) {
	rules := URLRules{}

	got, err := rules.Normalize("https://crawlme.monzo.com:443/path#section")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := "https://crawlme.monzo.com/path"
	if got != want {
		t.Fatalf("unexpected normalized URL: got %q, want %q", got, want)
	}
}

func TestNormalize_KeepsNonDefaultPort(t *testing.T) {
	rules := URLRules{}

	got, err := rules.Normalize("https://crawlme.monzo.com:8443/path")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := "https://crawlme.monzo.com:8443/path"
	if got != want {
		t.Fatalf("unexpected normalized URL: got %q, want %q", got, want)
	}
}

func TestIsDescendant_SameNormalizedOrigin(t *testing.T) {
	rules := URLRules{}
	seed, err := rules.Normalize("https://crawlme.monzo.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	seedURL, err := url.Parse(seed)
	if err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}

	ok, err := rules.IsDescendant(
		seedURL,
		"https://CRAWLME.MONZO.COM:443/pricing#top",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !ok {
		t.Fatalf("expected candidate to be descendant")
	}
}

func TestIsDescendant_DifferentHost(t *testing.T) {
	rules := URLRules{}
	seed, err := rules.Normalize("https://crawlme.monzo.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	seedURL, err := url.Parse(seed)
	if err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}

	ok, err := rules.IsDescendant(
		seedURL,
		"https://api.crawlme.monzo.com/docs",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ok {
		t.Fatalf("expected candidate to be non-descendant")
	}
}

func TestIsDescendant_DifferentScheme(t *testing.T) {
	rules := URLRules{}
	seed, err := rules.Normalize("https://crawlme.monzo.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	seedURL, err := url.Parse(seed)
	if err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}

	ok, err := rules.IsDescendant(
		seedURL,
		"http://crawlme.monzo.com",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ok {
		t.Fatalf("expected candidate to be non-descendant")
	}
}

func TestIsDescendant_InvalidNormalizedSeed(t *testing.T) {
	rules := URLRules{}

	_, err := rules.IsDescendant(&url.URL{}, "https://crawlme.monzo.com/docs")
	if err == nil {
		t.Fatalf("expected error for invalid normalized seed URL")
	}
	if !errors.Is(err, ErrInvalidSeedURL) {
		t.Fatalf("expected ErrInvalidSeedURL, got %v", err)
	}
}

func TestIsDescendant_NilNormalizedSeed(t *testing.T) {
	rules := URLRules{}

	_, err := rules.IsDescendant(nil, "https://crawlme.monzo.com/docs")
	if err == nil {
		t.Fatalf("expected error for nil normalized seed URL")
	}
	if !errors.Is(err, ErrInvalidSeedURL) {
		t.Fatalf("expected ErrInvalidSeedURL, got %v", err)
	}
}

func TestIsDescendant_InvalidCandidate(t *testing.T) {
	rules := URLRules{}
	seed, err := rules.Normalize("https://crawlme.monzo.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	seedURL, err := url.Parse(seed)
	if err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}

	_, err = rules.IsDescendant(seedURL, "not-a-valid-url")
	if err == nil {
		t.Fatalf("expected error for invalid candidate URL")
	}
	if !errors.Is(err, ErrInvalidCandidateURL) {
		t.Fatalf("expected ErrInvalidCandidateURL, got %v", err)
	}
}
