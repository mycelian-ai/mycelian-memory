package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mycelian/mycelian-memory/clients/go/client"
)

func TestSearchMemoriesTool(t *testing.T) {
	// stub backend search endpoint
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/search" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "entries": [],
            "count": 0,
            "latestContext": "{}",
            "contextTimestamp": "2025-07-27T00:00:00Z"
        }`))
	}))
	defer ts.Close()

	sdk := client.New(ts.URL)
	sh := NewSearchHandler(sdk)
	// Build request
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"user_id":   "u1",
				"memory_id": "m1",
				"query":     "hello",
				"top_k":     5,
			},
		},
	}

	res, err := sh.handleSearch(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res == nil {
		t.Fatalf("nil result")
	}
}
