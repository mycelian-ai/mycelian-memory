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
            "latestContextTimestamp": "2025-07-27T00:00:00Z",
            "contexts": [
              {"context": "{\"summary\": \"test context\"}", "timestamp": "2025-07-27T01:00:00Z", "score": 0.85}
            ]
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
				"top_ke":    5,
				"top_kc":    1,
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

	// Verify the response contains contexts array and score
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

	// Check that contexts array exists with one item and correct score
	ctxs, exists := payload["contexts"].([]interface{})
	if !exists || len(ctxs) != 1 {
		t.Fatalf("expected one context, got %v", payload["contexts"])
	}
	first, _ := ctxs[0].(map[string]interface{})
	if score, ok := first["score"].(float64); !ok || score != 0.85 {
		t.Errorf("expected contexts[0].score=0.85, got %v", first["score"])
	}
}
