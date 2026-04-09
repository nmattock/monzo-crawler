package cli

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

var (
	ErrMissingSeedURL   = errors.New("seed URL is required")
	ErrTooManyArguments = errors.New("too many arguments: expected <seed-url> [max-depth]")
)

// ParseArgs parses and validates command line arguments.
// Expected args: <seed-url> [max-depth]
func ParseArgs(args []string) (Config, error) {
	if len(args) == 0 {
		return Config{}, ErrMissingSeedURL
	}
	if len(args) > 2 {
		return Config{}, ErrTooManyArguments
	}

	seedURL := strings.TrimSpace(args[0])
	if err := validateSeedURL(seedURL); err != nil {
		return Config{}, err
	}

	cfg := Config{
		SeedURL: seedURL,
	}

	if len(args) == 2 {
		depth, err := parseDepth(args[1])
		if err != nil {
			return Config{}, err
		}
		cfg.MaxDepth = &depth
	}

	return cfg, nil
}

func parseDepth(raw string) (int, error) {
	depth, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid max depth %q: must be a positive integer", raw)
	}
	if depth <= 0 {
		return 0, fmt.Errorf("invalid max depth %q: must be greater than zero", raw)
	}
	return depth, nil
}

func validateSeedURL(raw string) error {
	if raw == "" {
		return ErrMissingSeedURL
	}

	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return fmt.Errorf("invalid seed URL %q", raw)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("invalid seed URL %q", raw)
	}
	return nil
}
