package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	client "github.com/mycelian/mycelian-memory/client"
)

func TestClient_PutAndGetContext(t *testing.T) {
	t.Parallel()
	vaultID, memID := "v1", "m1"
	var putCalled bool
	ctxText := "foo"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			putCalled = true
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte(ctxText))
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()

	c, err := client.NewWithDevMode(srv.URL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	ctx := context.Background()
	if _, err := c.PutContext(ctx, vaultID, memID, ctxText); err != nil {
		t.Fatalf("PutContext: %v", err)
	}
	if err := c.AwaitConsistency(ctx, memID); err != nil {
		t.Fatalf("AwaitConsistency: %v", err)
	}
	if !putCalled {
		t.Fatalf("PUT not called")
	}
	got, err := c.GetLatestContext(ctx, vaultID, memID)
	if err != nil {
		t.Fatalf("GetLatestContext: %v", err)
	}
	if got != ctxText {
		t.Fatalf("unexpected context: %q", got)
	}
}
