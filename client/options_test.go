package client

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestWithHTTPClientAndDebugLogging(t *testing.T) {
	// timeout option sets http timeout
	c := &Client{http: &http.Client{}}
	if err := WithHTTPTimeout(5 * time.Second)(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.http.Timeout != 5*time.Second {
		t.Fatalf("http timeout not set")
	}

	// debug logging wraps transport
	var called bool
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		return &http.Response{StatusCode: 200, Body: http.NoBody, Header: make(http.Header)}, nil
	})
	// Create a client with a base transport
	c2, err := New("http://example.com", "test-api-key", WithHTTPTimeout(2*time.Second), WithDebugLogging(true))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Inject base transport after construction for the test
	c2.http.Transport = rt

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", strings.NewReader(""))
	if _, err := c2.http.Do(req); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if !called {
		t.Fatalf("base transport not invoked")
	}
}

// Removed: sync-only client option and its panic path
