package main

import (
	"fmt"
	"os"

	"monzo-scraper/internal/cli"
)

func main() {
	cfg, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fmt.Fprintln(os.Stderr, "usage: monzo-scraper <seed-url> [max-depth]")
		os.Exit(1)
	}

	if cfg.CrawlFully() {
		fmt.Printf("Starting crawl from %s with unlimited depth\n", cfg.SeedURL)
		return
	}

	fmt.Printf("Starting crawl from %s up to depth %d\n", cfg.SeedURL, *cfg.MaxDepth)
}