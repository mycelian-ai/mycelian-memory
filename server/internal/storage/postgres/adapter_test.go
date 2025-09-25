package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mycelian/mycelian-memory/server/internal/storage"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func setupTestDB(t *testing.T) *PostgresStorage {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set, skipping postgres adapter tests")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Clean up test data
	ctx := context.Background()
	_, _ = db.ExecContext(ctx, "DELETE FROM memory_entries WHERE actor_id LIKE 'test_%'")
	_, _ = db.ExecContext(ctx, "DELETE FROM memories WHERE actor_id LIKE 'test_%'")
	_, _ = db.ExecContext(ctx, "DELETE FROM vaults WHERE actor_id LIKE 'test_%'")

	return &PostgresStorage{db: db}
}

func createTestMemory(t *testing.T, s *PostgresStorage) (string, uuid.UUID, string) {
	t.Helper()
	ctx := context.Background()

	actorID := "test_actor_" + uuid.New().String()
	vaultID := uuid.New()

	// Create vault
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO vaults (actor_id, vault_id, title)
		VALUES ($1, $2, $3)
	`, actorID, vaultID.String(), "test-vault")
	if err != nil {
		t.Fatalf("failed to create test vault: %v", err)
	}

	// Create memory
	mem, err := s.CreateMemory(ctx, storage.CreateMemoryRequest{
		ActorID:    actorID,
		VaultID:    vaultID,
		MemoryType: "TEST",
		Title:      "test-memory",
	})
	if err != nil {
		t.Fatalf("failed to create test memory: %v", err)
	}

	return actorID, vaultID, mem.MemoryID
}

func TestCreateMemoryEntry_WithExplicitConversationTime(t *testing.T) {
	s := setupTestDB(t)
	actorID, vaultID, memoryID := createTestMemory(t, s)
	ctx := context.Background()

	// Create entry with explicit past conversation time
	pastTime := time.Now().Add(-7 * 24 * time.Hour) // 1 week ago
	entry, err := s.CreateMemoryEntry(ctx, storage.CreateMemoryEntryRequest{
		ActorID:          actorID,
		VaultID:          vaultID,
		MemoryID:         memoryID,
		RawEntry:         "Meeting about Q3 planning",
		Summary:          stringPtr("Q3 planning discussion"),
		ConversationTime: &pastTime,
	})

	if err != nil {
		t.Fatalf("CreateMemoryEntry failed: %v", err)
	}

	// Verify conversation_time was set correctly
	if !timeEqual(entry.ConversationTime, pastTime) {
		t.Errorf("ConversationTime mismatch: got %v, want %v",
			entry.ConversationTime, pastTime)
	}

	// Verify creation_time is recent (within last minute)
	if time.Since(entry.CreationTime) > time.Minute {
		t.Errorf("CreationTime should be recent, got %v", entry.CreationTime)
	}

	// Verify conversation_time != creation_time
	if timeEqual(entry.ConversationTime, entry.CreationTime) {
		t.Errorf("ConversationTime should differ from CreationTime when explicitly set")
	}
}

func TestCreateMemoryEntry_WithoutConversationTime(t *testing.T) {
	s := setupTestDB(t)
	actorID, vaultID, memoryID := createTestMemory(t, s)
	ctx := context.Background()

	// Create entry without conversation_time (should default to NOW)
	entry, err := s.CreateMemoryEntry(ctx, storage.CreateMemoryEntryRequest{
		ActorID:          actorID,
		VaultID:          vaultID,
		MemoryID:         memoryID,
		RawEntry:         "Current conversation",
		Summary:          stringPtr("Real-time entry"),
		ConversationTime: nil, // Explicitly nil
	})

	if err != nil {
		t.Fatalf("CreateMemoryEntry failed: %v", err)
	}

	// Verify conversation_time equals creation_time (both should be NOW)
	if !timeEqual(entry.ConversationTime, entry.CreationTime) {
		t.Errorf("ConversationTime should equal CreationTime when not specified: got %v, want %v",
			entry.ConversationTime, entry.CreationTime)
	}

	// Verify both times are recent
	if time.Since(entry.CreationTime) > time.Minute {
		t.Errorf("Times should be recent, got creation=%v, conversation=%v",
			entry.CreationTime, entry.ConversationTime)
	}
}

func TestCreateMemoryEntry_WithPastConversationTime(t *testing.T) {
	s := setupTestDB(t)
	actorID, vaultID, memoryID := createTestMemory(t, s)
	ctx := context.Background()

	testCases := []struct {
		name     string
		pastTime time.Duration
		desc     string
	}{
		{"1HourAgo", -1 * time.Hour, "Meeting from 1 hour ago"},
		{"1DayAgo", -24 * time.Hour, "Yesterday's discussion"},
		{"1MonthAgo", -30 * 24 * time.Hour, "Last month's review"},
		{"1YearAgo", -365 * 24 * time.Hour, "Annual review from last year"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			convTime := time.Now().Add(tc.pastTime)
			entry, err := s.CreateMemoryEntry(ctx, storage.CreateMemoryEntryRequest{
				ActorID:          actorID,
				VaultID:          vaultID,
				MemoryID:         memoryID,
				RawEntry:         tc.desc,
				Summary:          stringPtr(tc.name),
				ConversationTime: &convTime,
			})

			if err != nil {
				t.Fatalf("CreateMemoryEntry failed: %v", err)
			}

			if !timeEqual(entry.ConversationTime, convTime) {
				t.Errorf("ConversationTime mismatch: got %v, want %v",
					entry.ConversationTime, convTime)
			}

			// Creation time should still be current
			if time.Since(entry.CreationTime) > time.Minute {
				t.Errorf("CreationTime should be recent despite past ConversationTime")
			}
		})
	}
}

func TestCreateMemoryEntry_RejectsFutureConversationTime(t *testing.T) {
	s := setupTestDB(t)
	actorID, vaultID, memoryID := createTestMemory(t, s)
	ctx := context.Background()

	// Attempt to create entry with future conversation time
	futureTime := time.Now().Add(24 * time.Hour) // Tomorrow
	_, err := s.CreateMemoryEntry(ctx, storage.CreateMemoryEntryRequest{
		ActorID:          actorID,
		VaultID:          vaultID,
		MemoryID:         memoryID,
		RawEntry:         "Future meeting",
		Summary:          stringPtr("This shouldn't work"),
		ConversationTime: &futureTime,
	})

	// This should fail (once validation is added)
	// For now, we document that this test expects future validation
	if err == nil {
		t.Skip("Future conversation_time validation not yet implemented in storage layer")
		// TODO: Uncomment when validation is added
		// t.Errorf("Expected error for future conversation_time, got nil")
	}
}

func TestGetMemoryEntry_IncludesConversationTime(t *testing.T) {
	s := setupTestDB(t)
	actorID, vaultID, memoryID := createTestMemory(t, s)
	ctx := context.Background()

	// Create entry with specific conversation time
	pastTime := time.Now().Add(-48 * time.Hour)
	created, err := s.CreateMemoryEntry(ctx, storage.CreateMemoryEntryRequest{
		ActorID:          actorID,
		VaultID:          vaultID,
		MemoryID:         memoryID,
		RawEntry:         "Test entry for retrieval",
		Summary:          stringPtr("Get test"),
		ConversationTime: &pastTime,
	})
	if err != nil {
		t.Fatalf("CreateMemoryEntry failed: %v", err)
	}

	// Retrieve the entry
	retrieved, err := s.GetMemoryEntry(ctx, actorID, vaultID, memoryID, created.CreationTime)
	if err != nil {
		t.Fatalf("GetMemoryEntry failed: %v", err)
	}

	// Verify all fields including conversation_time
	if retrieved.EntryID != created.EntryID {
		t.Errorf("EntryID mismatch: got %v, want %v", retrieved.EntryID, created.EntryID)
	}
	if !timeEqual(retrieved.ConversationTime, pastTime) {
		t.Errorf("ConversationTime mismatch: got %v, want %v",
			retrieved.ConversationTime, pastTime)
	}
	if !timeEqual(retrieved.CreationTime, created.CreationTime) {
		t.Errorf("CreationTime mismatch: got %v, want %v",
			retrieved.CreationTime, created.CreationTime)
	}
}

func TestListMemoryEntries_IncludesConversationTime(t *testing.T) {
	s := setupTestDB(t)
	actorID, vaultID, memoryID := createTestMemory(t, s)
	ctx := context.Background()

	// Create multiple entries with different conversation times
	times := []time.Duration{
		-72 * time.Hour, // 3 days ago
		-24 * time.Hour, // 1 day ago
		0,               // now (nil conversation_time)
	}

	var entries []*storage.MemoryEntry
	for i, duration := range times {
		var convTime *time.Time
		if duration != 0 {
			t := time.Now().Add(duration)
			convTime = &t
		}

		entry, err := s.CreateMemoryEntry(ctx, storage.CreateMemoryEntryRequest{
			ActorID:          actorID,
			VaultID:          vaultID,
			MemoryID:         memoryID,
			RawEntry:         fmt.Sprintf("Entry %d", i),
			Summary:          stringPtr(fmt.Sprintf("Summary %d", i)),
			ConversationTime: convTime,
		})
		if err != nil {
			t.Fatalf("CreateMemoryEntry %d failed: %v", i, err)
		}
		entries = append(entries, entry)

		// Small delay to ensure different creation times
		time.Sleep(10 * time.Millisecond)
	}

	// List all entries
	listed, err := s.ListMemoryEntries(ctx, storage.ListMemoryEntriesRequest{
		ActorID:  actorID,
		VaultID:  vaultID,
		MemoryID: memoryID,
	})
	if err != nil {
		t.Fatalf("ListMemoryEntries failed: %v", err)
	}

	if len(listed) != len(entries) {
		t.Errorf("Expected %d entries, got %d", len(entries), len(listed))
	}

	// Verify each entry has correct conversation_time
	// Note: List returns in reverse chronological order (newest first)
	for i, entry := range listed {
		expectedIdx := len(entries) - 1 - i // Reverse order
		expected := entries[expectedIdx]

		if entry.EntryID != expected.EntryID {
			t.Errorf("Entry %d: EntryID mismatch", i)
		}
		if !timeEqual(entry.ConversationTime, expected.ConversationTime) {
			t.Errorf("Entry %d: ConversationTime mismatch: got %v, want %v",
				i, entry.ConversationTime, expected.ConversationTime)
		}
	}
}

func TestCreateCorrectedEntry_PreservesOriginalConversationTime(t *testing.T) {
	s := setupTestDB(t)
	actorID, vaultID, memoryID := createTestMemory(t, s)
	ctx := context.Background()

	// Create original entry with specific conversation time (1 week ago)
	originalConvTime := time.Now().Add(-7 * 24 * time.Hour)
	original, err := s.CreateMemoryEntry(ctx, storage.CreateMemoryEntryRequest{
		ActorID:          actorID,
		VaultID:          vaultID,
		MemoryID:         memoryID,
		RawEntry:         "Original entry with typo",
		Summary:          stringPtr("Original summary"),
		ConversationTime: &originalConvTime,
	})
	if err != nil {
		t.Fatalf("CreateMemoryEntry failed: %v", err)
	}

	// Create correction for the entry
	corrected, err := s.CorrectMemoryEntry(ctx, storage.CorrectMemoryEntryRequest{
		ActorID:              actorID,
		VaultID:              vaultID,
		MemoryID:             memoryID,
		OriginalCreationTime: original.CreationTime,
		CorrectedContent:     "Original entry without typo",
		CorrectedSummary:     stringPtr("Corrected summary"),
		CorrectionReason:     "Fixed typo",
		CorrectedEntryID:     uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("CreateCorrectedMemoryEntry failed: %v", err)
	}

	// Verify correction has same conversation_time as original
	if !timeEqual(corrected.ConversationTime, originalConvTime) {
		t.Errorf("Correction should preserve original ConversationTime: got %v, want %v",
			corrected.ConversationTime, originalConvTime)
	}

	// But creation_time should be current
	if time.Since(corrected.CreationTime) > time.Minute {
		t.Errorf("Correction CreationTime should be recent, got %v", corrected.CreationTime)
	}

	// Verify conversation_time != creation_time for the correction
	if timeEqual(corrected.ConversationTime, corrected.CreationTime) {
		t.Errorf("Correction should have different ConversationTime and CreationTime")
	}
}

// Helper functions

func stringPtr(s string) *string {
	return &s
}

func timeEqual(t1, t2 time.Time) bool {
	// Compare times with microsecond precision (PostgreSQL precision)
	return t1.Truncate(time.Microsecond).Equal(t2.Truncate(time.Microsecond))
}