package cli

// Config stores validated CLI input for the app.
type Config struct {
	SeedURL  string
	MaxDepth *int
}

// CrawlFully returns true when no max depth is set.
func (c Config) CrawlFully() bool {
	return c.MaxDepth == nil
}
