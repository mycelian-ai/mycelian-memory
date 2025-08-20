package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mycelian/mycelian-memory/client"
)

func TestHandlersEndToEnd(t *testing.T) {
	// stub backend responding to various endpoints
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v0/vaults/v1/memories/m1/entries":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"entryId":"e1"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v0/vaults/v1/memories/m1/entries":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"entries":[{"entryId":"e1","context":"{\"activeContext\":\"hello\"}"}],"count":1}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v0/search":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"entries":[],"count":0,"latestContext":"{}"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v0/vaults/v1/memories/m1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"memoryId":"m1","title":"demo"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v0/users/u1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"userId":"u1","email":"x@example.com"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v0/vaults/v1/memories/m1/contexts":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"contextId":"ctx1","memoryId":"m1","vaultId":"v1","actorId":"u1","context":{"activeContext":"hello"},"creationTime":"2025-07-02T00:00:00Z"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v0/vaults/v1/memories/m1/contexts":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"contextId":"ctx1","memoryId":"m1","vaultId":"v1","actorId":"u1","creationTime":"2025-07-02T00:00:00Z"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer ts.Close()

	sdk, err := client.NewWithDevMode(ts.URL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}

	// ----- ContextHandler -----
	ch := NewContextHandler(sdk)
	// put_context
	putRes, err := ch.handlePutContext(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"user_id":   "u1",
		"vault_id":  "v1",
		"memory_id": "m1",
		"content":   "hello",
	}}})
	if err != nil || putRes == nil {
		t.Fatalf("put_context failed: %v", err)
	}
	// wait for local executor flush
	_ = sdk.AwaitConsistency(context.Background(), "m1")
	// get_context
	getRes, err := ch.handleGetContext(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"user_id":   "u1",
		"vault_id":  "v1",
		"memory_id": "m1",
	}}})
	if err != nil || getRes == nil {
		t.Fatalf("get_context failed: %v", err)
	}
	var obj map[string]string
	if err := json.Unmarshal([]byte(getRes.Content[0].(mcp.TextContent).Text), &obj); err != nil || obj["activeContext"] != "hello" {
		t.Fatalf("get_context mismatch: %+v", getRes.Content)
	}

	// ----- EntryHandler -----
	eh := NewEntryHandler(sdk)
	if _, err := eh.handleAddEntry(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"user_id":   "u1",
		"vault_id":  "v1",
		"memory_id": "m1",
		"raw_entry": "test",
		"summary":   "s",
	}}}); err != nil {
		t.Fatalf("add_entry error: %v", err)
	}
	if _, err := eh.handleListEntries(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"user_id":   "u1",
		"vault_id":  "v1",
		"memory_id": "m1",
		"limit":     1,
	}}}); err != nil {
		t.Fatalf("list_entries error: %v", err)
	}

	// ----- ConsistencyHandler -----
	cons := NewConsistencyHandler(sdk)
	if _, err := cons.handleAwait(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"memory_id": "m1",
	}}}); err != nil {
		t.Fatalf("await_consistency error: %v", err)
	}

	// ----- MemoryHandler -----
	mh := NewMemoryHandler(sdk)
	if _, err := mh.handleGetMemory(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"user_id":   "u1",
		"vault_id":  "v1",
		"memory_id": "m1",
	}}}); err != nil {
		t.Fatalf("get_memory error: %v", err)
	}

	// ----- SearchHandler -----
	sh := NewSearchHandler(sdk)
	if _, err := sh.handleSearch(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"user_id":   "u1",
		"vault_id":  "v1",
		"memory_id": "m1",
		"query":     "foo",
	}}}); err != nil {
		t.Fatalf("search_memories error: %v", err)
	}

	// verify backend context reflects the write
	resCtx, err := sdk.GetLatestContext(context.Background(), "v1", "m1")
	if err != nil || resCtx == nil {
		t.Fatalf("get latest context failed: %v", err)
	}
	if m, ok := resCtx.Context.(map[string]interface{}); !ok || m["activeContext"] != "hello" {
		t.Fatalf("unexpected backend context: %#v", resCtx.Context)
	}

	// ensure tests run quickly
	t.Logf("handlers all OK at %v", time.Now())
}
