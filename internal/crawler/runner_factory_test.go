package crawler

import "testing"

func TestNewRunner_DefaultsToMulti(t *testing.T) {
	r, err := NewRunner("", URLRules{}, fakeSource{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	cr, ok := r.(*ConcurrentRunner)
	if !ok {
		t.Fatalf("expected default runner to be *ConcurrentRunner")
	}
	if cr.concurrency != DefaultConcurrentWorkers {
		t.Fatalf("expected default concurrency %d, got %d", DefaultConcurrentWorkers, cr.concurrency)
	}
}

func TestNewRunner_Multi(t *testing.T) {
	r, err := NewRunner("multi", URLRules{}, fakeSource{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	cr, ok := r.(*ConcurrentRunner)
	if !ok {
		t.Fatalf("expected multi runner to be *ConcurrentRunner")
	}
	if cr.concurrency != DefaultConcurrentWorkers {
		t.Fatalf("expected default concurrency %d, got %d", DefaultConcurrentWorkers, cr.concurrency)
	}
}

func TestNewRunner_Single(t *testing.T) {
	r, err := NewRunner("single", URLRules{}, fakeSource{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := r.(*SingleRunner); !ok {
		t.Fatalf("expected single runner to be *SingleRunner")
	}
}

func TestNewRunnerWithConcurrency_Multi(t *testing.T) {
	const want = 4
	r, err := NewRunnerWithConcurrency("multi", URLRules{}, fakeSource{}, want)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	cr, ok := r.(*ConcurrentRunner)
	if !ok {
		t.Fatalf("expected multi runner to be *ConcurrentRunner")
	}
	if cr.concurrency != want {
		t.Fatalf("expected concurrency %d, got %d", want, cr.concurrency)
	}
}
