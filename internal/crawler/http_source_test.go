package crawler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
)

func TestHTTPChildSource_Children(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`
<!doctype html>
<html>
  <body>
    <a href="/about">About</a>
    <a href="https://example.com/abs">Absolute</a>
    <a href="#intro">FragmentOnly</a>
    <a href="mailto:test@example.com">Mail</a>
    <a href="tel:+44123">Phone</a>
    <a href="javascript:void(0)">JS</a>
    <a href="/about">Duplicate</a>
  </body>
</html>
`))
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	source := NewHTTPChildSource(server.Client())
	got, err := source.Children(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	sort.Strings(got)
	want := []string{
		server.URL + "/about",
		"https://example.com/abs",
	}
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("unexpected number of links: got %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected link at %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestHTTPChildSource_Non2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	source := NewHTTPChildSource(server.Client())
	_, err := source.Children(context.Background(), server.URL)
	if err == nil {
		t.Fatalf("expected error for non-2xx response")
	}
}
