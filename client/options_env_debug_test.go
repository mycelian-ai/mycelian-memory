package client

import (
	"context"
	"net/http"
	"testing"
)

func TestNew_AutoEnableDebugViaEnv(t *testing.T) {
	t.Setenv("MYCELIAN_DEBUG", "true")
	c := New("http://example.com")
	if _, ok := c.http.Transport.(*debugTransport); !ok {
		t.Fatalf("expected debugTransport to be installed when MYCELIAN_DEBUG=true")
	}
}

func TestDebugTransport_ErrorPath(t *testing.T) {
	// base transport returns error
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})
	c := New("http://example.com", WithHTTPClient(&http.Client{Transport: rt}), WithDebugLogging(true))
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", http.NoBody)
	if _, err := c.http.Do(req); err == nil {
		t.Fatalf("expected error from underlying transport")
	}
}
