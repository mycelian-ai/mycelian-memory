package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mycelian/mycelian-memory/client"
)

func TestAddEntry_WithConversationTime(t *testing.T) {
	// Channel to capture the request body from the async job
	capturedChan := make(chan []byte, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v0/vaults/v1/memories/m1/entries" {
			body, _ := io.ReadAll(r.Body)
			// Send captured body through channel
			select {
			case capturedChan <- body:
			default:
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"entryId":"e1","status":"enqueued"}`))
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer ts.Close()

	sdk, err := client.NewWithDevMode(ts.URL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}

	eh := NewEntryHandler(sdk)

	// Test with past conversation time
	pastTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	conversationTimeStr := pastTime.Format(time.RFC3339)

	result, err := eh.handleAddEntry(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"vault_id":          "v1",
				"memory_id":         "m1",
				"raw_entry":         "Meeting about Q3 planning",
				"summary":           "Q3 planning discussion",
				"conversation_time": conversationTimeStr,
			},
		},
	})

	if err != nil {
		t.Fatalf("handleAddEntry failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Wait for the job to complete and capture the request
	_ = sdk.AwaitConsistency(context.Background(), "m1")

	// Get the captured body from the async job
	select {
	case capturedBody := <-capturedChan:
		// Verify the conversation_time was sent in the request
		var decoded map[string]interface{}
		if err := json.Unmarshal(capturedBody, &decoded); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if convTime, ok := decoded["conversationTime"].(string); ok {
			parsedTime, err := time.Parse(time.RFC3339, convTime)
			if err != nil {
				t.Errorf("Failed to parse conversationTime from request: %v", err)
			}
			if !parsedTime.Equal(pastTime) {
				t.Errorf("ConversationTime mismatch: got %v, want %v", parsedTime, pastTime)
			}
		} else {
			t.Error("Expected conversationTime in request body")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for request capture")
	}
}

func TestAddEntry_WithoutConversationTime(t *testing.T) {
	// Channel to capture the request body from the async job
	capturedChan := make(chan []byte, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v0/vaults/v1/memories/m1/entries" {
			body, _ := io.ReadAll(r.Body)
			// Send captured body through channel
			select {
			case capturedChan <- body:
			default:
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"entryId":"e2","status":"enqueued"}`))
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer ts.Close()

	sdk, err := client.NewWithDevMode(ts.URL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}

	eh := NewEntryHandler(sdk)

	result, err := eh.handleAddEntry(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"vault_id":  "v1",
				"memory_id": "m1",
				"raw_entry": "Current conversation",
				"summary":   "Real-time entry",
				// conversation_time not provided
			},
		},
	})

	if err != nil {
		t.Fatalf("handleAddEntry failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Wait for the job to complete and capture the request
	_ = sdk.AwaitConsistency(context.Background(), "m1")

	// Get the captured body from the async job
	select {
	case capturedBody := <-capturedChan:
		// Verify conversation_time was not sent when not provided
		var decoded map[string]interface{}
		if err := json.Unmarshal(capturedBody, &decoded); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// conversationTime should be omitted when not provided
		if _, ok := decoded["conversationTime"]; ok {
			t.Error("Expected conversationTime to be omitted when not provided")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for request capture")
	}
}

func TestAddEntry_InvalidConversationTimeFormat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server shouldn't be called for invalid time format
		if r.Method == http.MethodPost && r.URL.Path == "/v0/vaults/v1/memories/m1/entries" {
			// Check if the request has conversationTime
			body, _ := io.ReadAll(r.Body)
			var decoded map[string]interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				// If we cannot decode, fail the request to surface issues in test
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"bad json"}`))
				return
			}

			// If invalid format, conversationTime should be nil/omitted
			if _, ok := decoded["conversationTime"]; ok {
				t.Error("conversationTime should not be sent with invalid format")
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"entryId":"e3","status":"enqueued"}`))
		}
	}))
	defer ts.Close()

	sdk, err := client.NewWithDevMode(ts.URL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}

	eh := NewEntryHandler(sdk)

	// Test with invalid conversation_time format
	result, err := eh.handleAddEntry(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"vault_id":          "v1",
				"memory_id":         "m1",
				"raw_entry":         "Test entry",
				"summary":           "Test",
				"conversation_time": "invalid-date-format",
			},
		},
	})

	// Should still succeed but ignore the invalid conversation_time
	if err != nil {
		t.Fatalf("handleAddEntry failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestListEntries_WithTemporalFilters(t *testing.T) {
	// Mock server that returns entries with conversation_time
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v0/vaults/v1/memories/m1/entries" {
			// Check query parameters
			query := r.URL.Query()
			before := query.Get("before")
			after := query.Get("after")

			// Return different responses based on filters
			var response string
			if before != "" && after != "" {
				t.Error("Should not have both before and after")
			} else if before != "" {
				// Return entries before the specified time
				response = `{
					"entries": [{
						"entryId": "e1",
						"rawEntry": "Old entry",
						"summary": "Past",
						"creationTime": "2024-01-10T10:00:00Z",
						"conversationTime": "2024-01-10T10:00:00Z"
					}],
					"count": 1
				}`
			} else if after != "" {
				// Return entries after the specified time
				response = `{
					"entries": [{
						"entryId": "e2",
						"rawEntry": "Recent entry",
						"summary": "Recent",
						"creationTime": "2024-01-20T10:00:00Z",
						"conversationTime": "2024-01-20T10:00:00Z"
					}],
					"count": 1
				}`
			} else {
				// Return all entries
				response = `{
					"entries": [
						{
							"entryId": "e1",
							"rawEntry": "Entry 1",
							"summary": "First",
							"creationTime": "2024-01-10T10:00:00Z",
							"conversationTime": "2024-01-09T10:00:00Z"
						},
						{
							"entryId": "e2",
							"rawEntry": "Entry 2",
							"summary": "Second",
							"creationTime": "2024-01-20T10:00:00Z",
							"conversationTime": "2024-01-20T10:00:00Z"
						}
					],
					"count": 2
				}`
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer ts.Close()

	sdk, err := client.NewWithDevMode(ts.URL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}

	eh := NewEntryHandler(sdk)

	// Test with before filter
	result, err := eh.handleListEntries(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"vault_id":  "v1",
				"memory_id": "m1",
				"before":    "2024-01-15T00:00:00Z",
				"limit":     10,
			},
		},
	})

	if err != nil {
		t.Fatalf("handleListEntries with before filter failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Test with after filter
	result, err = eh.handleListEntries(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"vault_id":  "v1",
				"memory_id": "m1",
				"after":     "2024-01-15T00:00:00Z",
				"limit":     10,
			},
		},
	})

	if err != nil {
		t.Fatalf("handleListEntries with after filter failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Test without filters (should return all)
	result, err = eh.handleListEntries(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"vault_id":  "v1",
				"memory_id": "m1",
				"limit":     10,
			},
		},
	})

	if err != nil {
		t.Fatalf("handleListEntries without filters failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestListEntries_ReturnsConversationTime(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v0/vaults/v1/memories/m1/entries" {
			// Return entries with different conversation_time values
			response := `{
				"entries": [
					{
						"entryId": "e1",
						"rawEntry": "Past conversation",
						"summary": "Meeting notes",
						"creationTime": "2024-01-20T10:00:00Z",
						"conversationTime": "2024-01-15T14:30:00Z"
					},
					{
						"entryId": "e2",
						"rawEntry": "Current entry",
						"summary": "Real-time",
						"creationTime": "2024-01-20T11:00:00Z",
						"conversationTime": "2024-01-20T11:00:00Z"
					}
				],
				"count": 2
			}`
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer ts.Close()

	sdk, err := client.NewWithDevMode(ts.URL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}

	eh := NewEntryHandler(sdk)

	result, err := eh.handleListEntries(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"vault_id":  "v1",
				"memory_id": "m1",
			},
		},
	})

	if err != nil {
		t.Fatalf("handleListEntries failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check that the result contains the conversation_time information
	// The handler returns JSON text content
	if len(result.Content) == 0 {
		t.Fatal("Expected content in result")
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	// Parse the JSON to verify conversation_time is included
	var entries map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &entries); err != nil {
		t.Fatalf("Failed to parse result JSON: %v", err)
	}

	entriesList, ok := entries["entries"].([]interface{})
	if !ok || len(entriesList) != 2 {
		t.Fatal("Expected 2 entries in result")
	}

	// Check first entry has different conversation and creation times
	firstEntry := entriesList[0].(map[string]interface{})
	if firstEntry["conversationTime"] != "2024-01-15T14:30:00Z" {
		t.Errorf("First entry conversationTime mismatch: got %v", firstEntry["conversationTime"])
	}
	if firstEntry["creationTime"] != "2024-01-20T10:00:00Z" {
		t.Errorf("First entry creationTime mismatch: got %v", firstEntry["creationTime"])
	}

	// Check second entry has same conversation and creation times
	secondEntry := entriesList[1].(map[string]interface{})
	if secondEntry["conversationTime"] != secondEntry["creationTime"] {
		t.Error("Second entry should have same conversation and creation times")
	}
}
