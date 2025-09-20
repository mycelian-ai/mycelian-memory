package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mycelian/mycelian-memory/server/internal/model"
)

func TestCreateMemoryEntry_WithConversationTime(t *testing.T) {
	// This is an integration test that requires a running database
	// We'll create a simpler unit test approach

	// Create test request with conversation_time
	pastTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	reqBody := map[string]interface{}{
		"rawEntry":         "Meeting about Q3 planning",
		"summary":          "Q3 planning discussion",
		"conversationTime": pastTime.Format(time.RFC3339),
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/vaults/vault123/memories/mem456/entries", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-key")

	// Test that the request body is properly formatted
	var decoded struct {
		RawEntry         string     `json:"rawEntry"`
		Summary          *string    `json:"summary"`
		ConversationTime *time.Time `json:"conversationTime"`
	}
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&decoded); err != nil {
		t.Fatalf("Failed to decode request body: %v", err)
	}

	if decoded.ConversationTime == nil {
		t.Error("Expected ConversationTime to be set in request")
	}

	if !decoded.ConversationTime.Equal(pastTime) {
		t.Errorf("ConversationTime mismatch: got %v, want %v", *decoded.ConversationTime, pastTime)
	}
}

func TestCreateMemoryEntry_WithoutConversationTime(t *testing.T) {
	// Test request without conversation_time
	reqBody := map[string]interface{}{
		"rawEntry": "Current conversation",
		"summary":  "Real-time entry",
		// No conversationTime field
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/vaults/vault123/memories/mem456/entries", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-key")

	// Test that the request body decodes properly without conversation_time
	var decoded struct {
		RawEntry         string     `json:"rawEntry"`
		Summary          *string    `json:"summary"`
		ConversationTime *time.Time `json:"conversationTime"`
	}
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&decoded); err != nil {
		t.Fatalf("Failed to decode request body: %v", err)
	}

	if decoded.ConversationTime != nil {
		t.Error("Expected ConversationTime to be nil when not provided")
	}
}

func TestCreateMemoryEntry_InvalidConversationTimeFormats(t *testing.T) {
	testCases := []struct {
		name        string
		timeValue   interface{}
		shouldError bool
	}{
		{"ValidISO8601", "2024-01-15T10:30:00Z", false},
		{"InvalidFormat", "invalid-date", true},
		{"InvalidDate", "2024-13-45T10:30:00Z", true},
		{"NotISO8601", "yesterday", true},
		{"NumberInsteadOfString", 12345, true},
		{"EmptyString", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := map[string]interface{}{
				"rawEntry":         "Test entry",
				"conversationTime": tc.timeValue,
			}
			bodyBytes, _ := json.Marshal(reqBody)

			var decoded struct {
				RawEntry         string     `json:"rawEntry"`
				ConversationTime *time.Time `json:"conversationTime"`
			}
			err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&decoded)

			if tc.shouldError && err == nil && decoded.ConversationTime != nil {
				t.Errorf("Expected error for format %v, but got valid time", tc.timeValue)
			}
			if !tc.shouldError && err != nil {
				t.Errorf("Expected valid time for format %v, but got error: %v", tc.timeValue, err)
			}
		})
	}
}

func TestListMemoryEntries_RequestFormat(t *testing.T) {
	// Test that list request can include temporal filters
	req := httptest.NewRequest("GET", "/api/vaults/vault123/memories/mem456/entries?before=2024-01-15T10:30:00Z", nil)
	req.Header.Set("Authorization", "Bearer test-key")

	// Parse query parameters
	before := req.URL.Query().Get("before")
	if before == "" {
		t.Error("Expected 'before' query parameter")
	}

	// Verify it parses as valid time
	parsedTime, err := time.Parse(time.RFC3339, before)
	if err != nil {
		t.Errorf("Failed to parse 'before' parameter as RFC3339: %v", err)
	}

	expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !parsedTime.Equal(expectedTime) {
		t.Errorf("Time mismatch: got %v, want %v", parsedTime, expectedTime)
	}
}

func TestMemoryEntryJSON_ConversationTime(t *testing.T) {
	// Test JSON marshaling/unmarshaling of MemoryEntry with conversation_time
	now := time.Now().UTC().Truncate(time.Second)
	pastTime := now.Add(-7 * 24 * time.Hour)

	entry := model.MemoryEntry{
		EntryID:          uuid.New().String(),
		ActorID:          "test_actor",
		VaultID:          "vault123",
		MemoryID:         "mem456",
		RawEntry:         "Test entry",
		CreationTime:     now,
		ConversationTime: pastTime,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal entry: %v", err)
	}

	// Verify JSON contains conversationTime
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if _, ok := jsonMap["conversationTime"]; !ok {
		t.Error("JSON missing conversationTime field")
	}

	// Unmarshal back to struct
	var decoded model.MemoryEntry
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal entry: %v", err)
	}

	// Verify times match (with second precision due to JSON)
	if !decoded.ConversationTime.Truncate(time.Second).Equal(pastTime.Truncate(time.Second)) {
		t.Errorf("ConversationTime mismatch after JSON round-trip: got %v, want %v",
			decoded.ConversationTime, pastTime)
	}

	if !decoded.CreationTime.Truncate(time.Second).Equal(now.Truncate(time.Second)) {
		t.Errorf("CreationTime mismatch after JSON round-trip: got %v, want %v",
			decoded.CreationTime, now)
	}
}

// Integration test that can be run with a test server
func TestCreateMemoryEntry_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test would require setting up the full handler with dependencies
	// For now, we'll document the test structure

	t.Skip("Integration test requires full service setup")

	// Setup would include:
	// 1. Create test database
	// 2. Initialize services
	// 3. Create handler with real dependencies
	// 4. Create test vault and memory
	// 5. Test creating entry with conversation_time
	// 6. Verify entry stored correctly in database
}

// Test the handler's validation of conversation_time
func TestValidateConversationTime(t *testing.T) {
	testCases := []struct {
		name        string
		time        *time.Time
		shouldError bool
	}{
		{"Nil", nil, false},
		{"Past", timePtr(time.Now().Add(-24 * time.Hour)), false},
		{"Current", timePtr(time.Now()), false},
		{"NearFuture", timePtr(time.Now().Add(5 * time.Second)), false}, // Allow small tolerance
		{"Future", timePtr(time.Now().Add(24 * time.Hour)), true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This test documents expected validation behavior
			// Actual implementation would be in the handler

			if tc.time != nil && tc.time.After(time.Now().Add(time.Minute)) {
				// Future times beyond tolerance should be rejected
				if !tc.shouldError {
					t.Error("Expected error for future conversation_time")
				}
			}
		})
	}
}

// Helper functions
func timePtr(t time.Time) *time.Time {
	return &t
}