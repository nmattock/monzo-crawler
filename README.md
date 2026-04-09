# monzo-scraper

A Go-based web crawler project with a clean, extendable structure.

## Prerequisites

- Go 1.23+ installed

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
go run . <seed-url> [max-depth]
```

- `seed-url` (required): The initial URL to start crawling from.
- `max-depth` (optional): Positive integer crawl depth.
  - If omitted, crawling is treated as unlimited depth.

### Examples

Run with unlimited depth:

```bash
go run . https://example.com
```

Run with a depth limit of 3:

```bash
go run . https://example.com 3
```
