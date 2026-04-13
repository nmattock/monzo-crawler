package crawler

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type trackingSource struct {
	children map[string][]string
	sleep    time.Duration

	mu       sync.Mutex
	inFlight int
	maxSeen  int
}

func (s *trackingSource) Children(pageURL string) ([]string, error) {
	s.mu.Lock()
	s.inFlight++
	if s.inFlight > s.maxSeen {
		s.maxSeen = s.inFlight
	}
	s.mu.Unlock()

	time.Sleep(s.sleep)

	s.mu.Lock()
	s.inFlight--
	s.mu.Unlock()

	if links, ok := s.children[pageURL]; ok {
		return links, nil
	}
	return []string{}, nil
}

func (s *trackingSource) MaxSeen() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.maxSeen
}

func TestConcurrentRunner_Run_PreventsCycles(t *testing.T) {
	source := fakeSource{
		children: map[string][]string{
			"https://crawlme.monzo.com": {
				"https://crawlme.monzo.com/a",
			},
			"https://crawlme.monzo.com/a": {
				"https://crawlme.monzo.com",
			},
		},
	}
	r, err := NewConcurrentRunner(URLRules{}, source, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	out, err := r.Run("https://crawlme.monzo.com", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(out.Visited) != 2 {
		t.Fatalf("expected 2 visited pages, got %d", len(out.Visited))
	}
}

func TestConcurrentRunner_Run_RespectsConcurrencyLimit(t *testing.T) {
	children := map[string][]string{
		"https://crawlme.monzo.com": {},
	}
	for i := 0; i < 25; i++ {
		child := fmt.Sprintf("https://crawlme.monzo.com/p%d", i)
		children["https://crawlme.monzo.com"] = append(children["https://crawlme.monzo.com"], child)
		children[child] = []string{}
	}

	source := &trackingSource{
		children: children,
		sleep:    20 * time.Millisecond,
	}
	const maxConcurrent = 3
	r, err := NewConcurrentRunner(URLRules{}, source, maxConcurrent)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = r.Run("https://crawlme.monzo.com", intPtr(1))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	maxSeen := source.MaxSeen()
	if maxSeen > maxConcurrent {
		t.Fatalf("expected max in-flight <= %d, got %d", maxConcurrent, maxSeen)
	}
	if maxSeen < 2 {
		t.Fatalf("expected at least two concurrent processes, got %d", maxSeen)
	}
}
