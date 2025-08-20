package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	client "github.com/mycelian/mycelian-memory/client"
)

func TestClient_Search_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost || r.URL.Path != "/v0/search" {
			t.Fatalf("expected POST /v0/search")
		}
		resp := client.SearchResponse{Entries: []client.SearchEntry{{Entry: client.Entry{ID: "e1"}}}, Count: 1}
		_ = json.NewEncoder(w).Encode(&resp)
	}))
	defer srv.Close()

	c, err := client.New(srv.URL, "test-api-key")
	if err != nil {
		t.Fatalf("client.New error: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	res, err := c.Search(context.Background(), client.SearchRequest{UserID: "user1", MemoryID: "m1", Query: "x"})
	if err != nil || len(res.Entries) != 1 {
		t.Fatalf("Search error: %v", err)
	}
}
