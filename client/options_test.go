package client

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestWithHTTPClientAndDebugLogging(t *testing.T) {
	// custom client
	hc := &http.Client{}
	c := &Client{}
	if err := WithHTTPClient(hc)(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.http != hc {
		t.Fatalf("http client not set")
	}
	if err := WithHTTPClient(nil)(c); err == nil {
		t.Fatalf("expected error for nil http client")
	}

	// debug logging wraps transport
	var called bool
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		return &http.Response{StatusCode: 200, Body: http.NoBody, Header: make(http.Header)}, nil
	})
	base := &http.Client{Transport: rt}
	c2 := New("http://example.com", "test-api-key", WithHTTPClient(base), WithDebugLogging(true))
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", strings.NewReader(""))
	if _, err := c2.http.Do(req); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if !called {
		t.Fatalf("base transport not invoked")
	}
}

// Removed: sync-only client option and its panic path
