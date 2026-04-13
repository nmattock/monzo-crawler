package cli

// Config stores validated CLI input for the app.
type Config struct {
	SeedURL     string
	MaxDepth    *int
	Debug       bool
	Summary     bool
	Runner      string
	WriteToFile bool
}

// CrawlFully returns true when no max depth is set.
func (c Config) CrawlFully() bool {
	return c.MaxDepth == nil
}
