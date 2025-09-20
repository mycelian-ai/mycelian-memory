package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mycelian/mycelian-memory/client/internal/types"
)

func TestAddEntry_WithConversationTime(t *testing.T) {
	t.Parallel()

	// Capture the request to verify conversation_time is sent
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		capturedBody = body
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status": "enqueued", "memoryId": "m1"}`))
	}))
	defer srv.Close()

	exec := &mockExec{}
	pastTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	req := types.AddEntryRequest{
		RawEntry:         "Meeting about Q3 planning",
		Summary:          "Q3 planning discussion",
		ConversationTime: &pastTime,
	}

	ack, err := AddEntry(context.Background(), exec, srv.Client(), srv.URL, "v1", "m1", req)
	if err != nil {
		t.Fatalf("AddEntry error: %v", err)
	}
	if ack == nil || ack.MemoryID != "m1" || ack.Status != "enqueued" {
		t.Fatalf("unexpected ack: %+v", ack)
	}

	// Verify the conversation_time was sent in the request
	var decoded map[string]interface{}
	if err := json.Unmarshal(capturedBody, &decoded); err != nil {
		t.Fatalf("Failed to decode request body: %v", err)
	}

	if _, ok := decoded["conversationTime"]; !ok {
		t.Error("Expected conversationTime in request body")
	}

	// Verify the time format
	if convTime, ok := decoded["conversationTime"].(string); ok {
		parsedTime, err := time.Parse(time.RFC3339, convTime)
		if err != nil {
			t.Errorf("Failed to parse conversationTime: %v", err)
		}
		if !parsedTime.Equal(pastTime) {
			t.Errorf("ConversationTime mismatch: got %v, want %v", parsedTime, pastTime)
		}
	} else {
		t.Error("conversationTime not a string in request")
	}
}

func TestAddEntry_WithoutConversationTime(t *testing.T) {
	t.Parallel()

	// Capture the request to verify conversation_time is NOT sent
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = body
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status": "enqueued", "memoryId": "m1"}`))
	}))
	defer srv.Close()

	exec := &mockExec{}
	req := types.AddEntryRequest{
		RawEntry: "Current conversation",
		Summary:  "Real-time entry",
		// ConversationTime not set
	}

	ack, err := AddEntry(context.Background(), exec, srv.Client(), srv.URL, "v1", "m1", req)
	if err != nil {
		t.Fatalf("AddEntry error: %v", err)
	}
	if ack == nil {
		t.Fatal("Expected non-nil acknowledgment")
	}

	// Verify conversation_time was not sent when nil
	var decoded map[string]interface{}
	if err := json.Unmarshal(capturedBody, &decoded); err != nil {
		t.Fatalf("Failed to decode request body: %v", err)
	}

	// conversationTime should be omitted when nil
	if _, ok := decoded["conversationTime"]; ok {
		t.Error("Expected conversationTime to be omitted when nil")
	}
}

func TestAddEntry_ValidatesConversationTimeFormat(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		time          *time.Time
		expectSuccess bool
	}{
		{
			name:          "ValidPastTime",
			time:          timePtr(time.Now().Add(-24 * time.Hour)),
			expectSuccess: true,
		},
		{
			name:          "ValidCurrentTime",
			time:          timePtr(time.Now()),
			expectSuccess: true,
		},
		{
			name:          "ValidNearFuture",
			time:          timePtr(time.Now().Add(30 * time.Second)),
			expectSuccess: true, // Allow small tolerance
		},
		{
			name:          "NilTime",
			time:          nil,
			expectSuccess: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the request is well-formed
				body, _ := io.ReadAll(r.Body)
				var decoded map[string]interface{}
				if err := json.Unmarshal(body, &decoded); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// Check conversation_time format if present
				if convTimeStr, ok := decoded["conversationTime"].(string); ok {
					if _, err := time.Parse(time.RFC3339, convTimeStr); err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
				}

				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"status": "enqueued", "memoryId": "m1"}`))
			}))
			defer srv.Close()

			exec := &mockExec{}
			req := types.AddEntryRequest{
				RawEntry:         "Test entry",
				ConversationTime: tc.time,
			}

			_, err := AddEntry(context.Background(), exec, srv.Client(), srv.URL, "v1", "m1", req)
			if tc.expectSuccess && err != nil {
				t.Errorf("Expected success, got error: %v", err)
			}
			if !tc.expectSuccess && err == nil {
				t.Error("Expected error, got success")
			}
		})
	}
}

func TestListEntries_ParsesConversationTime(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		// Return entries with conversation_time
		response := map[string]interface{}{
			"entries": []map[string]interface{}{
				{
					"entryId":          "e1",
					"rawEntry":         "Entry 1",
					"summary":          "Summary 1",
					"creationTime":     time.Now().Format(time.RFC3339),
					"conversationTime": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				},
				{
					"entryId":          "e2",
					"rawEntry":         "Entry 2",
					"creationTime":     time.Now().Format(time.RFC3339),
					"conversationTime": time.Now().Format(time.RFC3339), // Same as creation
				},
			},
			"count": 2,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer srv.Close()

	resp, err := ListEntries(context.Background(), srv.Client(), srv.URL, "v1", "m1", nil)
	if err != nil {
		t.Fatalf("ListEntries error: %v", err)
	}

	if resp.Count != 2 {
		t.Errorf("Expected count=2, got %d", resp.Count)
	}
	if len(resp.Entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(resp.Entries))
	}

	// Verify conversation_time is parsed correctly
	for i, entry := range resp.Entries {
		if entry.ConversationTime.IsZero() {
			t.Errorf("Entry %d: ConversationTime should not be zero", i)
		}
		if entry.CreationTime.IsZero() {
			t.Errorf("Entry %d: CreationTime should not be zero", i)
		}

		// First entry should have different times
		if i == 0 && entry.ConversationTime.Equal(entry.CreationTime) {
			t.Errorf("Entry 0: Expected different conversation and creation times")
		}
	}
}

func TestListEntries_WithTemporalFilters(t *testing.T) {
	t.Parallel()

	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"entries": [], "count": 0}`))
	}))
	defer srv.Close()

	beforeTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	afterTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// ListEntries takes map[string]string params
	params := map[string]string{
		"before": beforeTime.Format(time.RFC3339),
		"after":  afterTime.Format(time.RFC3339),
		"limit":  "10",
	}

	_, err := ListEntries(context.Background(), srv.Client(), srv.URL, "v1", "m1", params)
	if err != nil {
		t.Fatalf("ListEntries error: %v", err)
	}

	// Verify temporal filters were sent
	if capturedQuery == "" {
		t.Error("Expected query parameters")
	}

	// Check for before and after parameters
	if !bytes.Contains([]byte(capturedQuery), []byte("before=")) {
		t.Error("Expected 'before' parameter in query")
	}
	if !bytes.Contains([]byte(capturedQuery), []byte("after=")) {
		t.Error("Expected 'after' parameter in query")
	}
	if !bytes.Contains([]byte(capturedQuery), []byte("limit=10")) {
		t.Error("Expected 'limit' parameter in query")
	}
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}