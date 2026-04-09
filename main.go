package main

import (
	"fmt"
	"os"

	"monzo-scraper/internal/cli"
	"monzo-scraper/internal/crawler"
)

func main() {
	cfg, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fmt.Fprintln(os.Stderr, "usage: monzo-scraper <seed-url> [max-depth] [--debug]")
		os.Exit(1)
	}

	c, err := crawler.NewCrawler(crawler.NewURLRules(), crawler.NewHTTPChildSource(nil))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: initialize crawler: %v\n", err)
		os.Exit(1)
	}
	c.SetDebug(cfg.Debug)

	runResult, err := c.Run(cfg.SeedURL, cfg.MaxDepth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	for _, visitedURL := range runResult.VisitOrder {
		fmt.Println(visitedURL)
		page := runResult.Results[visitedURL]
		for _, link := range page.Links {
			fmt.Printf("  - %s\n", link)
		}
	}
}