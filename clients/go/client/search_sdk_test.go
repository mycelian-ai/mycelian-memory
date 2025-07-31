package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSearch_SDK(t *testing.T) {
	t.Parallel()

	// Start a stub HTTP server that mimics the /api/search endpoint.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/search" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "entries": [{
                "entryId": "entry-1",
                "userId": "u1",
                "memoryId": "m1",
                "summary": "demo summary",
                "rawEntry": "demo raw",
                "score": 0.82
            }],
            "count": 1,
            "latestContext": "ctx-text",
            "contextTimestamp": "2025-07-27T00:00:00Z"
        }`))
	}))
	defer ts.Close()

	c := MustNew(ts.URL)

	ctx := context.Background()
	resp, err := c.Search(ctx, SearchRequest{UserID: "u1", MemoryID: "m1", Query: "demo", TopK: 5})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if resp.Count != 1 || len(resp.Entries) != 1 {
		t.Fatalf("expected 1 result, got %+v", resp)
	}
	if string(resp.LatestContext) != "\"ctx-text\"" {
		t.Fatalf("latestContext mismatch: %s", string(resp.LatestContext))
	}
	if resp.ContextTimestamp == nil {
		t.Fatal("contextTimestamp should be non-nil")
	}
	expectedTs := time.Date(2025, 7, 27, 0, 0, 0, 0, time.UTC)
	if !resp.ContextTimestamp.Equal(expectedTs) {
		t.Fatalf("timestamp mismatch: got %v want %v", resp.ContextTimestamp, expectedTs)
	}
	if resp.Entries[0].Score <= 0 {
		t.Fatalf("expected positive score, got %f", resp.Entries[0].Score)
	}
}
