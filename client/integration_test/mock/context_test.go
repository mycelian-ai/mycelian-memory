package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	client "github.com/mycelian/mycelian-memory/client"
)

func TestClient_PutAndGetContext(t *testing.T) {
	t.Parallel()
	vaultID, memID := "v1", "m1"
	var putCalled bool
	ctxResp := client.GetContextResponse{Context: map[string]interface{}{"activeContext": "foo"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPut:
			putCalled = true
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(&ctxResp)
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
	if _, err := c.PutContext(ctx, vaultID, memID, client.PutContextRequest{Context: map[string]interface{}{"activeContext": "foo"}}); err != nil {
		t.Fatalf("PutContext: %v", err)
	}
	if err := c.AwaitConsistency(ctx, memID); err != nil {
		t.Fatalf("AwaitConsistency: %v", err)
	}
	if !putCalled {
		t.Fatalf("PUT not called")
	}
	got, err := c.GetContext(ctx, vaultID, memID)
	if err != nil {
		t.Fatalf("GetContext: %v", err)
	}
	ctxMap, ok := got.Context.(map[string]interface{})
	if !ok || ctxMap["activeContext"].(string) != "foo" {
		t.Fatalf("unexpected context")
	}
}
