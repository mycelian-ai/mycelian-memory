package services

import (
	"context"
	"testing"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/model"
	"github.com/mycelian/mycelian-memory/server/internal/storage"
)

// mockStore implements store.Store for testing
type mockStore struct {
	entries *mockEntryStore
}

func (m *mockStore) Users() storage.Users       { return nil }
func (m *mockStore) Vaults() storage.Vaults     { return nil }
func (m *mockStore) Memories() storage.Memories { return nil }
func (m *mockStore) Entries() storage.Entries   { return m.entries }
func (m *mockStore) Contexts() storage.Contexts { return nil }

// mockEntryStore implements storage.Entries
type mockEntryStore struct {
	createFunc func(ctx context.Context, e *model.MemoryEntry) (*model.MemoryEntry, error)
}

func (m *mockEntryStore) Create(ctx context.Context, e *model.MemoryEntry) (*model.MemoryEntry, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, e)
	}
	// Default implementation: return the entry with timestamps set
	result := *e
	result.CreationTime = time.Now()
	if result.ConversationTime.IsZero() {
		result.ConversationTime = result.CreationTime
	}
	return &result, nil
}

func (m *mockEntryStore) List(ctx context.Context, req model.ListEntriesRequest) ([]*model.MemoryEntry, error) {
	return nil, nil
}

func (m *mockEntryStore) GetByID(ctx context.Context, userID, vaultID, memoryID, entryID string) (*model.MemoryEntry, error) {
	return nil, nil
}

func (m *mockEntryStore) UpdateTags(ctx context.Context, userID, vaultID, memoryID, entryID string, tags map[string]interface{}) (*model.MemoryEntry, error) {
	return nil, nil
}

func (m *mockEntryStore) DeleteByID(ctx context.Context, userID, vaultID, memoryID, entryID string) error {
	return nil
}

func TestMemoryService_CreateEntry_RejectsFutureConversationTime(t *testing.T) {
	store := &mockStore{
		entries: &mockEntryStore{},
	}
	svc := &MemoryService{store: store}
	ctx := context.Background()

	// Test with future conversation time (should be rejected)
	futureTime := time.Now().Add(2 * time.Hour)
	entry := &model.MemoryEntry{
		ActorID:          "test-actor",
		VaultID:          "test-vault",
		MemoryID:         "test-memory",
		RawEntry:         "Future meeting",
		ConversationTime: futureTime,
	}

	_, err := svc.CreateEntry(ctx, entry)
	if err == nil {
		t.Error("Expected error for future conversation_time, got nil")
	}
	if err != nil && err.Error() != "conversation_time cannot be in the future" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestMemoryService_CreateEntry_AllowsPastConversationTime(t *testing.T) {
	store := &mockStore{
		entries: &mockEntryStore{},
	}
	svc := &MemoryService{store: store}
	ctx := context.Background()

	// Test with past conversation time (should be allowed)
	pastTime := time.Now().Add(-24 * time.Hour)
	entry := &model.MemoryEntry{
		ActorID:          "test-actor",
		VaultID:          "test-vault",
		MemoryID:         "test-memory",
		RawEntry:         "Yesterday's meeting",
		ConversationTime: pastTime,
	}

	result, err := svc.CreateEntry(ctx, entry)
	if err != nil {
		t.Fatalf("Unexpected error for past conversation_time: %v", err)
	}
	if !result.ConversationTime.Equal(pastTime) {
		t.Errorf("ConversationTime not preserved: got %v, want %v", result.ConversationTime, pastTime)
	}
}

func TestMemoryService_CreateEntry_AllowsNearFutureConversationTime(t *testing.T) {
	store := &mockStore{
		entries: &mockEntryStore{},
	}
	svc := &MemoryService{store: store}
	ctx := context.Background()

	// Test with near-future conversation time (within tolerance)
	nearFutureTime := time.Now().Add(30 * time.Second)
	entry := &model.MemoryEntry{
		ActorID:          "test-actor",
		VaultID:          "test-vault",
		MemoryID:         "test-memory",
		RawEntry:         "Current conversation",
		ConversationTime: nearFutureTime,
	}

	result, err := svc.CreateEntry(ctx, entry)
	if err != nil {
		t.Fatalf("Unexpected error for near-future conversation_time: %v", err)
	}
	if !result.ConversationTime.Equal(nearFutureTime) {
		t.Errorf("ConversationTime not preserved: got %v, want %v", result.ConversationTime, nearFutureTime)
	}
}

func TestMemoryService_CreateEntry_DefaultsToCurrentTime(t *testing.T) {
	store := &mockStore{
		entries: &mockEntryStore{},
	}
	svc := &MemoryService{store: store}
	ctx := context.Background()

	// Test with zero conversation time (should default to current)
	entry := &model.MemoryEntry{
		ActorID:  "test-actor",
		VaultID:  "test-vault",
		MemoryID: "test-memory",
		RawEntry: "Current conversation",
		// ConversationTime not set (zero value)
	}

	result, err := svc.CreateEntry(ctx, entry)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Should be set to current time (within last minute)
	if time.Since(result.ConversationTime) > time.Minute {
		t.Errorf("ConversationTime should be recent when not specified, got %v", result.ConversationTime)
	}
}
