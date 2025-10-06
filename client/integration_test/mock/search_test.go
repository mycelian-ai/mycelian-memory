package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	res, err := c.Search(context.Background(), client.SearchRequest{MemoryID: "m1", Query: "x"})
	if err != nil || len(res.Entries) != 1 {
		t.Fatalf("Search error: %v", err)
	}
}

func TestClient_Search_WithConversationTime(t *testing.T) {
	t.Parallel()
	now := time.Now().Round(time.Second)
	pastTime := now.Add(-24 * time.Hour)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost || r.URL.Path != "/v0/search" {
			t.Fatalf("expected POST /v0/search")
		}

		// Mock response with ConversationTime
		creationTime1 := now
		creationTime2 := now.Add(-time.Hour)
		resp := client.SearchResponse{
			Entries: []client.SearchEntry{
				{
					Entry: client.Entry{
						ID:               "e1",
						MemoryID:         "m1",
						Summary:          "test entry 1",
						ConversationTime: pastTime,
					},
					Score:        0.95,
					CreationTime: &creationTime1,
				},
				{
					Entry: client.Entry{
						ID:               "e2",
						MemoryID:         "m1",
						Summary:          "test entry 2",
						ConversationTime: now.Add(-2 * time.Hour),
					},
					Score:        0.85,
					CreationTime: &creationTime2,
				},
			},
			Count: 2,
		}
		_ = json.NewEncoder(w).Encode(&resp)
	}))
	defer srv.Close()

	c, err := client.New(srv.URL, "test-api-key")
	if err != nil {
		t.Fatalf("client.New error: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	topKE := 5
	res, err := c.Search(context.Background(), client.SearchRequest{
		MemoryID: "m1",
		Query:    "test",
		TopKE:    &topKE,
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	if res.Count != 2 {
		t.Fatalf("expected 2 entries, got %d", res.Count)
	}

	// Verify ConversationTime is properly deserialized
	for i, entry := range res.Entries {
		if entry.ConversationTime.IsZero() {
			t.Errorf("entry %d has zero ConversationTime", i)
		}

		// Verify ConversationTime is before or equal to CreationTime
		if entry.CreationTime != nil && entry.ConversationTime.After(*entry.CreationTime) {
			t.Errorf("entry %d: ConversationTime (%v) is after CreationTime (%v)",
				i, entry.ConversationTime, *entry.CreationTime)
		}
	}

	// Verify specific times match what we sent
	if !res.Entries[0].ConversationTime.Equal(pastTime) {
		t.Errorf("entry 0 ConversationTime mismatch: got %v, want %v",
			res.Entries[0].ConversationTime, pastTime)
	}
}
