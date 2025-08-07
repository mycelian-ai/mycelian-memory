package client

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestPutAndGetContext(t *testing.T) {
    userID, vaultID, memID := "user1", "v1", "m1"
    var putCalled bool
    ctxResp := GetContextResponse{Context: map[string]interface{}{"activeContext": "foo"}}
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

    c := New(srv.URL)
    ctx := context.Background()
    if _, err := c.PutContext(ctx, userID, vaultID, memID, PutContextRequest{Context: map[string]interface{}{"activeContext": "foo"}}); err != nil {
        t.Fatalf("PutContext: %v", err)
    }
    if err := c.AwaitConsistency(ctx, memID); err != nil {
        t.Fatalf("AwaitConsistency: %v", err)
    }
    if !putCalled {
        t.Fatalf("PUT not called")
    }
    got, err := c.GetContext(ctx, userID, vaultID, memID)
    if err != nil {
        t.Fatalf("GetContext: %v", err)
    }
    ctxMap, ok := got.Context.(map[string]interface{})
    if !ok || ctxMap["activeContext"].(string) != "foo" {
        t.Fatalf("unexpected context")
    }
}

func TestGetContextNotFound(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusNotFound)
    }))
    defer srv.Close()
    c := New(srv.URL)
    _, err := c.GetContext(context.Background(), "user1", "v1", "m1")
    if err != ErrNotFound {
        t.Fatalf("expected ErrNotFound")
    }
}
