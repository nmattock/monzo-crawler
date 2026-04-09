package cli

import (
	"errors"
	"testing"
)

func TestParseArgs_OnlySeedURL_CrawlsFully(t *testing.T) {
	cfg, err := ParseArgs([]string{"https://example.com"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.SeedURL != "https://example.com" {
		t.Fatalf("unexpected seed URL: %s", cfg.SeedURL)
	}
	if !cfg.CrawlFully() {
		t.Fatalf("expected CrawlFully=true when max depth omitted")
	}
	if cfg.Debug {
		t.Fatalf("expected debug to default to false")
	}
}

func TestParseArgs_WithDepth(t *testing.T) {
	cfg, err := ParseArgs([]string{"https://example.com", "3"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.MaxDepth == nil {
		t.Fatalf("expected max depth to be set")
	}
	if *cfg.MaxDepth != 3 {
		t.Fatalf("unexpected max depth: %d", *cfg.MaxDepth)
	}
}

func TestParseArgs_MissingSeedURL(t *testing.T) {
	_, err := ParseArgs([]string{})
	if !errors.Is(err, ErrMissingSeedURL) {
		t.Fatalf("expected ErrMissingSeedURL, got %v", err)
	}
}

func TestParseArgs_InvalidSeedURL(t *testing.T) {
	_, err := ParseArgs([]string{"example.com"})
	if err == nil {
		t.Fatalf("expected error for invalid URL")
	}
}

func TestParseArgs_InvalidDepth(t *testing.T) {
	_, err := ParseArgs([]string{"https://example.com", "0"})
	if err == nil {
		t.Fatalf("expected error for invalid depth")
	}
}

func TestParseArgs_WithDebugFlag(t *testing.T) {
	cfg, err := ParseArgs([]string{"https://example.com", "--debug"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.Debug {
		t.Fatalf("expected debug to be true")
	}
	if !cfg.CrawlFully() {
		t.Fatalf("expected unlimited crawl when depth omitted")
	}
}

func TestParseArgs_WithDepthAndDebugFlag(t *testing.T) {
	cfg, err := ParseArgs([]string{"https://example.com", "2", "--debug"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.Debug {
		t.Fatalf("expected debug to be true")
	}
	if cfg.MaxDepth == nil || *cfg.MaxDepth != 2 {
		t.Fatalf("expected max depth of 2")
	}
}
