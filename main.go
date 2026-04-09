package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"monzo-scraper/internal/cli"
	"monzo-scraper/internal/crawler"
)

func main() {
	cfg, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fmt.Fprintln(os.Stderr, "usage: monzo-scraper <seed-url> [max-depth] [--debug] [--summary]")
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

	if cfg.Summary {
		printSummary(runResult)
		return
	}

	for _, visitedURL := range runResult.VisitOrder {
		fmt.Println(visitedURL)
		page := runResult.Results[visitedURL]
		for _, link := range page.Links {
			fmt.Printf("  - %s\n", link)
		}
	}
}

type depthSummary struct {
	found        int
	scraped      int
	totalScrape  time.Duration
}

func printSummary(runResult crawler.RunResult) {
	byDepth := make(map[int]*depthSummary)
	overall := depthSummary{}
	for _, page := range runResult.Results {
		stats, ok := byDepth[page.Depth]
		if !ok {
			stats = &depthSummary{}
			byDepth[page.Depth] = stats
		}
		stats.found++
		if page.Err == nil {
			stats.scraped++
		}
		stats.totalScrape += page.ScrapeDuration

		overall.found++
		if page.Err == nil {
			overall.scraped++
		}
		overall.totalScrape += page.ScrapeDuration
	}

	depths := make([]int, 0, len(byDepth))
	for depth := range byDepth {
		depths = append(depths, depth)
	}
	sort.Ints(depths)

	fmt.Printf("Total pages found: %d\n", len(runResult.Results))
	for _, depth := range depths {
		stats := byDepth[depth]
		avg := time.Duration(0)
		if stats.found > 0 {
			avg = stats.totalScrape / time.Duration(stats.found)
		}
		fmt.Printf(
			"Depth %d: found=%d scraped=%d avg_scrape_time=%s\n",
			depth,
			stats.found,
			stats.scraped,
			avg.Truncate(time.Microsecond),
		)
	}

	overallAvg := time.Duration(0)
	if overall.found > 0 {
		overallAvg = overall.totalScrape / time.Duration(overall.found)
	}
	fmt.Println("Overall totals:")
	fmt.Printf(
		"  found=%d scraped=%d total_scrape_time=%s avg_scrape_time=%s\n",
		overall.found,
		overall.scraped,
		overall.totalScrape.Truncate(time.Microsecond),
		overallAvg.Truncate(time.Microsecond),
	)
}