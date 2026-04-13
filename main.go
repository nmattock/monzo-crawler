package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"monzo-scraper/internal/cli"
	"monzo-scraper/internal/crawler"
)

func main() {
	cfg, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fmt.Fprintln(os.Stderr, "usage: monzo-scraper <seed-url> [max-depth] [--debug] [--summary] [--write-to-file] [--runner=<name>]")
		os.Exit(1)
	}

	runner, err := crawler.NewRunner(
		cfg.Runner,
		crawler.NewURLRules(),
		crawler.NewHTTPChildSource(nil),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: initialize crawler: %v\n", err)
		os.Exit(1)
	}
	if debugRunner, ok := runner.(crawler.DebuggableRunner); ok {
		debugRunner.SetDebug(cfg.Debug)
	}
	if pr, ok := runner.(crawler.ProgressableRunner); ok {
		pr.SetProgress(func(visited int) {
			if visited%500 == 0 {
				fmt.Fprintf(os.Stderr, "progress: %d pages crawled at %s\n", visited, time.Now().Format("15:04:05"))
			}
		})
	}

	crawlStart := time.Now()
	runResult, err := runner.Run(cfg.SeedURL, cfg.MaxDepth)
	totalRunTime := time.Since(crawlStart)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// stdout: whichever mode the user chose
	var stdout strings.Builder
	if cfg.Summary {
		writeSummary(&stdout, runResult, totalRunTime)
	} else {
		writeResults(&stdout, runResult)
	}
	fmt.Print(stdout.String())

	if cfg.WriteToFile {
		// File always contains summary first, then the full per-page listing.
		var file strings.Builder
		writeSummary(&file, runResult, totalRunTime)
		fmt.Fprintln(&file)
		writeResults(&file, runResult)

		filename := safeFilename(cfg.SeedURL, crawlStart)
		if writeErr := os.WriteFile(filename, []byte(file.String()), 0644); writeErr != nil {
			fmt.Fprintf(os.Stderr, "error: write output file: %v\n", writeErr)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "output written to %s\n", filename)
	}
}

func writeResults(w io.Writer, runResult crawler.RunResult) {
	for _, visitedURL := range runResult.VisitOrder {
		fmt.Fprintln(w, visitedURL)
		page := runResult.Results[visitedURL]
		for _, link := range page.Links {
			fmt.Fprintf(w, "  - %s\n", link)
		}
	}
}

type depthSummary struct {
	found       int
	scraped     int
	totalScrape time.Duration
}

func writeSummary(w io.Writer, runResult crawler.RunResult, totalRunTime time.Duration) {
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

	fmt.Fprintf(w, "Total pages found: %d\n", len(runResult.Results))
	for _, depth := range depths {
		stats := byDepth[depth]
		avg := time.Duration(0)
		if stats.found > 0 {
			avg = stats.totalScrape / time.Duration(stats.found)
		}
		fmt.Fprintf(w,
			"Depth %d: found=%d scraped=%d avg_scrape_time=%s\n",
			depth, stats.found, stats.scraped,
			avg.Truncate(time.Microsecond),
		)
	}

	overallAvg := time.Duration(0)
	if overall.found > 0 {
		overallAvg = overall.totalScrape / time.Duration(overall.found)
	}
	fmt.Fprintln(w, "Overall totals:")
	fmt.Fprintf(w,
		"  found=%d scraped=%d total_run_time=%s avg_scrape_time=%s\n",
		overall.found, overall.scraped,
		totalRunTime.Truncate(time.Microsecond),
		overallAvg.Truncate(time.Microsecond),
	)
}

// safeFilename builds a filesystem-safe filename from the seed URL and crawl
// start time: e.g. "crawlme.monzo.com--2026-04-13--15-04-05.txt".
func safeFilename(seedURL string, t time.Time) string {
	base := seedURL
	if u, err := url.Parse(seedURL); err == nil && u.Host != "" {
		base = u.Host + u.Path
	}
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-':
			return r
		default:
			return '-'
		}
	}, base)
	safe = strings.Trim(safe, "-.")
	return safe + "--" + t.Format("2006-01-02--15-04-05") + ".txt"
}
