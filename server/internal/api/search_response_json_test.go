package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/model"
)

func TestSearchHit_JSONMarshaling(t *testing.T) {
	now := time.Now().Round(time.Second)
	past := now.Add(-24 * time.Hour)

	hit := model.SearchHit{
		EntryID:          "test-entry-1",
		ActorID:          "test-actor",
		MemoryID:         "test-memory",
		Summary:          "test summary",
		RawEntry:         "test raw entry content",
		Score:            0.95,
		CreationTime:     now,
		ConversationTime: past,
	}

	// Marshal to JSON
	data, err := json.Marshal(hit)
	if err != nil {
		t.Fatalf("failed to marshal SearchHit: %v", err)
	}

	// Unmarshal back
	var decoded model.SearchHit
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal SearchHit: %v", err)
	}

	// Verify all fields are preserved
	if decoded.EntryID != hit.EntryID {
		t.Errorf("EntryID mismatch: got %v, want %v", decoded.EntryID, hit.EntryID)
	}
	if decoded.ActorID != hit.ActorID {
		t.Errorf("ActorID mismatch: got %v, want %v", decoded.ActorID, hit.ActorID)
	}
	if decoded.MemoryID != hit.MemoryID {
		t.Errorf("MemoryID mismatch: got %v, want %v", decoded.MemoryID, hit.MemoryID)
	}
	if decoded.Summary != hit.Summary {
		t.Errorf("Summary mismatch: got %v, want %v", decoded.Summary, hit.Summary)
	}
	if decoded.RawEntry != hit.RawEntry {
		t.Errorf("RawEntry mismatch: got %v, want %v", decoded.RawEntry, hit.RawEntry)
	}
	if decoded.Score != hit.Score {
		t.Errorf("Score mismatch: got %v, want %v", decoded.Score, hit.Score)
	}
	if !decoded.CreationTime.Equal(hit.CreationTime) {
		t.Errorf("CreationTime mismatch: got %v, want %v", decoded.CreationTime, hit.CreationTime)
	}
	if !decoded.ConversationTime.Equal(hit.ConversationTime) {
		t.Errorf("ConversationTime mismatch: got %v, want %v", decoded.ConversationTime, hit.ConversationTime)
	}
}

func TestSearchHit_JSONMarshaling_ZeroConversationTime(t *testing.T) {
	// Test that zero ConversationTime is still included in JSON
	hit := model.SearchHit{
		EntryID:          "test-entry",
		ActorID:          "test-actor",
		MemoryID:         "test-memory",
		Summary:          "test",
		Score:            0.5,
		CreationTime:     time.Now(),
		ConversationTime: time.Time{}, // Zero time
	}

	data, err := json.Marshal(hit)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Check that conversationTime field is present in JSON
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, exists := m["conversationTime"]; !exists {
		t.Error("conversationTime field missing from JSON when value is zero")
	}

	// Verify it unmarshals correctly
	var decoded model.SearchHit
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !decoded.ConversationTime.IsZero() {
		t.Errorf("expected zero ConversationTime, got %v", decoded.ConversationTime)
	}
}

func TestSearchResponse_WithMultipleEntries(t *testing.T) {
	now := time.Now().Round(time.Second)

	// Create response with multiple entries having different conversation times
	entries := []model.SearchHit{
		{
			EntryID:          "e1",
			MemoryID:         "m1",
			Summary:          "first",
			Score:            0.9,
			CreationTime:     now,
			ConversationTime: now.Add(-time.Hour),
		},
		{
			EntryID:          "e2",
			MemoryID:         "m1",
			Summary:          "second",
			Score:            0.8,
			CreationTime:     now.Add(-time.Hour),
			ConversationTime: now.Add(-2 * time.Hour),
		},
	}

	responseData := map[string]interface{}{
		"entries":                entries,
		"count":                  len(entries),
		"latestContext":          "test context",
		"latestContextTimestamp": now.Format(time.RFC3339),
	}

	// Marshal the entire response
	data, err := json.Marshal(responseData)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	// Unmarshal and verify
	var decoded struct {
		Entries []model.SearchHit `json:"entries"`
		Count   int               `json:"count"`
	}

	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(decoded.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(decoded.Entries))
	}

	// Verify conversation times are preserved
	for i, entry := range decoded.Entries {
		if entry.ConversationTime.IsZero() {
			t.Errorf("entry %d has zero ConversationTime", i)
		}
		if !entry.ConversationTime.Equal(entries[i].ConversationTime) {
			t.Errorf("entry %d ConversationTime mismatch: got %v, want %v",
				i, entry.ConversationTime, entries[i].ConversationTime)
		}
	}
}
