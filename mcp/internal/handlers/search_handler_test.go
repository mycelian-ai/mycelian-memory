package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mycelian/mycelian-memory/client"
)

func TestSearchMemoriesTool(t *testing.T) {
	// stub backend search endpoint
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0/search" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "entries": [],
            "count": 0,
            "latestContext": "{}",
            "contextTimestamp": "2025-07-27T00:00:00Z",
            "bestContext": "{\"summary\": \"test context\"}",
            "bestContextTimestamp": "2025-07-27T01:00:00Z",
            "bestContextScore": 0.85
        }`))
	}))
	defer ts.Close()

	sdk, err := client.NewWithDevMode(ts.URL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}
	sh := NewSearchHandler(sdk)
	// Build request
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
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

	// Verify the response contains best context fields
	if len(res.Content) == 0 {
		t.Fatalf("no content in response")
	}

	textContent, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}

	content := textContent.Text

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	// Check that best context fields are present
	expectedFields := []string{"bestContext", "bestContextTimestamp", "bestContextScore"}
	for _, field := range expectedFields {
		if _, exists := payload[field]; !exists {
			t.Errorf("missing field %s in response", field)
		}
	}

	// Verify best context score is correct
	if score, ok := payload["bestContextScore"].(float64); !ok || score != 0.85 {
		t.Errorf("expected bestContextScore=0.85, got %v", payload["bestContextScore"])
	}
}
