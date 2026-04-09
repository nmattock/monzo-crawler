# monzo-scraper

A Go-based web crawler project with a clean, extendable structure.

## Prerequisites

- Go 1.25+ installed

## Setup

1. Clone the repository.
2. Open the project root.
3. Verify Go is available:

```bash
go version
```

## Run tests

```bash
go test ./...
```

## Run the CLI

```bash
go run . <seed-url> [max-depth] [--debug]
```

- `seed-url` (required): The initial URL to start crawling from.
- `max-depth` (optional): Positive integer crawl depth.
  - If omitted, crawling is treated as unlimited depth.
- `--debug` (optional): Enables verbose crawl diagnostics to help investigate slow runs.

### Examples

Run with unlimited depth:

```bash
go run . https://example.com
```

Run with a depth limit of 3:

```bash
go run . https://example.com 3
```

Run with debug output enabled:

```bash
go run . https://example.com --debug
```

Output format:

- each visited page URL is printed on its own line
- discovered in-domain links for that page are printed underneath as indented bullet lines

## Link Extraction Strategy (Initial)

Current child-link extraction uses `net/http` + `goquery` via an HTTP-backed `ChildSource`.

Limitations of this approach:

- It only sees links present in the raw HTML response.
- It does not execute JavaScript, so client-rendered links are not discovered.
- It does not perform browser interactions (clicks, form submission, infinite scroll, auth flows).
