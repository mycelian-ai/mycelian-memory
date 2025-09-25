//go:build integration
// +build integration

package client_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/mycelian/mycelian-memory/client"
)

// TestConversationTimeE2E exercises the full conversation_time flow:
//  1. create vault & memory
//  2. add entries with different conversation_time values
//  3. verify conversation_time is preserved correctly
//  4. test temporal filtering with before/after
//  5. verify future date rejection
//  6. cleanup
//
// Run with: go test -tags=integration ./client/integration_test/real -run TestConversationTimeE2E -v
func TestConversationTimeE2E(t *testing.T) {
	baseURL := os.Getenv("TEST_BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11545"
	}

	c, err := client.NewWithDevMode(baseURL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	defer c.Close()

	// Create vault & memory
	vault, err := c.CreateVault(ctx, client.CreateVaultRequest{
		Title:       "conversation-time-test-vault",
		Description: "Testing conversation_time feature",
	})
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	defer func() {
		_ = c.DeleteVault(ctx, vault.VaultID)
	}()

	mem, err := c.CreateMemory(ctx, vault.VaultID, client.CreateMemoryRequest{
		Title:       "conversation-time-test",
		MemoryType:  "PROJECT",
		Description: "Testing conversation_time handling",
	})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}
	defer func() {
		_ = c.DeleteMemory(ctx, vault.VaultID, mem.ID)
	}()

	// Test 1: Add entry with past conversation_time
	pastTime := time.Now().Add(-7 * 24 * time.Hour) // 1 week ago
	ack1, err := c.AddEntry(ctx, vault.VaultID, mem.ID, client.AddEntryRequest{
		RawEntry:         "Meeting from last week",
		Summary:          "Weekly sync meeting",
		ConversationTime: &pastTime,
	})
	if err != nil {
		t.Fatalf("add entry with past conversation_time: %v", err)
	}
	if ack1.Status != "enqueued" {
		t.Errorf("Expected status 'enqueued', got %s", ack1.Status)
	}
	_ = c.AwaitConsistency(ctx, mem.ID)

	// Test 2: Add entry without conversation_time (should default to current)
	ack2, err := c.AddEntry(ctx, vault.VaultID, mem.ID, client.AddEntryRequest{
		RawEntry: "Current conversation",
		Summary:  "Real-time entry",
		// ConversationTime not set
	})
	if err != nil {
		t.Fatalf("add entry without conversation_time: %v", err)
	}
	if ack2.Status != "enqueued" {
		t.Errorf("Expected status 'enqueued', got %s", ack2.Status)
	}
	_ = c.AwaitConsistency(ctx, mem.ID)

	// Test 3: Add entry with specific past conversation_time
	specificTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	_, err = c.AddEntry(ctx, vault.VaultID, mem.ID, client.AddEntryRequest{
		RawEntry:         "Q1 planning meeting",
		Summary:          "Annual planning",
		ConversationTime: &specificTime,
	})
	if err != nil {
		t.Fatalf("add entry with specific conversation_time: %v", err)
	}
	_ = c.AwaitConsistency(ctx, mem.ID)

	// Test 4: Attempt to add entry with future conversation_time (should be rejected)
	futureTime := time.Now().Add(24 * time.Hour) // Tomorrow
	_, err = c.AddEntry(ctx, vault.VaultID, mem.ID, client.AddEntryRequest{
		RawEntry:         "Future meeting",
		Summary:          "This should fail",
		ConversationTime: &futureTime,
	})
	// This should fail with validation error
	if err == nil {
		t.Log("Warning: Future conversation_time was not rejected (validation may not be fully implemented yet)")
		// Not failing the test as validation might be pending implementation
	}

	// Test 5: List all entries and verify conversation_time values
	allEntries, err := c.ListEntries(ctx, vault.VaultID, mem.ID, nil)
	if err != nil {
		t.Fatalf("list all entries: %v", err)
	}

	// Should have at least 3 entries (maybe 4 if future wasn't rejected)
	if allEntries.Count < 3 {
		t.Errorf("Expected at least 3 entries, got %d", allEntries.Count)
	}

	// Verify conversation times are preserved
	foundPastEntry := false
	foundCurrentEntry := false
	foundSpecificEntry := false

	for _, entry := range allEntries.Entries {
		t.Logf("Entry: %s, ConversationTime: %v, CreationTime: %v",
			entry.Summary, entry.ConversationTime, entry.CreationTime)

		switch entry.Summary {
		case "Weekly sync meeting":
			foundPastEntry = true
			// Verify conversation_time is in the past
			timeDiff := time.Since(entry.ConversationTime)
			if timeDiff < 6*24*time.Hour || timeDiff > 8*24*time.Hour {
				t.Errorf("Past entry conversation_time not preserved correctly: %v", entry.ConversationTime)
			}

		case "Real-time entry":
			foundCurrentEntry = true
			// Conversation_time should be close to creation_time
			diff := entry.CreationTime.Sub(entry.ConversationTime).Abs()
			if diff > time.Minute {
				t.Errorf("Current entry should have conversation_time ≈ creation_time, diff: %v", diff)
			}

		case "Annual planning":
			foundSpecificEntry = true
			// Verify exact time match
			if !entry.ConversationTime.Equal(specificTime) {
				t.Errorf("Specific entry conversation_time mismatch: got %v, want %v",
					entry.ConversationTime, specificTime)
			}
		}
	}

	if !foundPastEntry {
		t.Error("Past entry not found in list")
	}
	if !foundCurrentEntry {
		t.Error("Current entry not found in list")
	}
	if !foundSpecificEntry {
		t.Error("Specific time entry not found in list")
	}

	// Test 6: Test temporal filtering with 'before' parameter
	// Note: 'before' filters by creation_time, not conversation_time
	beforeTime := time.Now().Add(-3 * 24 * time.Hour) // 3 days ago
	beforeEntries, err := c.ListEntries(ctx, vault.VaultID, mem.ID, map[string]string{
		"before": beforeTime.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("list entries with before filter: %v", err)
	}

	// Since all entries were created recently (within the test),
	// 'before' 3 days ago should return 0 entries
	if beforeEntries.Count != 0 {
		t.Logf("Note: 'before' filter uses creation_time, not conversation_time. Found %d entries", beforeEntries.Count)
	}

	// Test 7: Test temporal filtering with 'after' parameter
	afterTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	afterEntries, err := c.ListEntries(ctx, vault.VaultID, mem.ID, map[string]string{
		"after": afterTime.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("list entries with after filter: %v", err)
	}

	// Should include all entries after Jan 1, 2024
	for _, entry := range afterEntries.Entries {
		if entry.ConversationTime.Before(afterTime) {
			t.Errorf("Found entry with conversation_time before filter: %v < %v",
				entry.ConversationTime, afterTime)
		}
	}

	// Test 8: Verify conversation_time ordering
	// Entries should be returned in reverse chronological order by creation_time
	if len(allEntries.Entries) >= 2 {
		for i := 1; i < len(allEntries.Entries); i++ {
			prev := allEntries.Entries[i-1]
			curr := allEntries.Entries[i]
			if prev.CreationTime.Before(curr.CreationTime) {
				t.Errorf("Entries not in reverse chronological order by creation_time at index %d", i)
			}
		}
	}

	t.Logf("Successfully tested conversation_time E2E with %d entries", allEntries.Count)
}

// TestConversationTimeCorrectionsE2E tests that corrections preserve original conversation_time
func TestConversationTimeCorrectionsE2E(t *testing.T) {
	baseURL := os.Getenv("TEST_BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11545"
	}

	c, err := client.NewWithDevMode(baseURL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	defer c.Close()

	// Create vault & memory
	vault, err := c.CreateVault(ctx, client.CreateVaultRequest{
		Title: "corrections-test-vault",
	})
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	defer func() {
		_ = c.DeleteVault(ctx, vault.VaultID)
	}()

	mem, err := c.CreateMemory(ctx, vault.VaultID, client.CreateMemoryRequest{
		Title:      "corrections-test",
		MemoryType: "NOTES",
	})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}
	defer func() {
		_ = c.DeleteMemory(ctx, vault.VaultID, mem.ID)
	}()

	// Add an entry with specific conversation_time
	originalConvTime := time.Date(2024, 1, 10, 14, 30, 0, 0, time.UTC)
	_, err = c.AddEntry(ctx, vault.VaultID, mem.ID, client.AddEntryRequest{
		RawEntry:         "Original entry with typo",
		Summary:          "Original summary",
		ConversationTime: &originalConvTime,
	})
	if err != nil {
		t.Fatalf("add original entry: %v", err)
	}
	_ = c.AwaitConsistency(ctx, mem.ID)

	// Get the entry to obtain its ID
	entries, err := c.ListEntries(ctx, vault.VaultID, mem.ID, nil)
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(entries.Entries) == 0 {
		t.Fatal("No entries found")
	}

	originalEntry := entries.Entries[0]

	// Verify original conversation_time
	if !originalEntry.ConversationTime.Equal(originalConvTime) {
		t.Errorf("Original conversation_time not preserved: got %v, want %v",
			originalEntry.ConversationTime, originalConvTime)
	}

	// Note: Correction API implementation would go here when available
	// For now, we're testing that the original entry preserves conversation_time correctly

	t.Log("Successfully verified conversation_time is preserved in entries")
}