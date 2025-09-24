//go:build integration
// +build integration

package client_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mycelian/mycelian-memory/client"
)

// TestSearchE2E performs comprehensive end-to-end search testing with step verification.
// Combines search flow testing with individual backend component validation.
func TestSearchE2E(t *testing.T) {
	baseURL := os.Getenv("TEST_BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11545"
	}

	c, err := client.NewWithDevMode(baseURL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer c.Close()

	// 1. User management is now external - use MockAuthorizer's actor ID

	// 2. create vault & memory and verify
	vaultTitle := fmt.Sprintf("search-vault-%s", uuid.NewString()[:8])
	vault, err := c.CreateVault(ctx, client.CreateVaultRequest{Title: vaultTitle})
	if err != nil || vault.VaultID == "" {
		t.Fatalf("create vault failed: %v", err)
	}
	mem, err := c.CreateMemory(ctx, vault.VaultID, client.CreateMemoryRequest{Title: "search-mem", MemoryType: "NOTES"})
	if err != nil || mem.ID == "" {
		t.Fatalf("create memory failed: %v", err)
	}

	// 3. write context and wait for consistency
	if _, err := c.PutContext(ctx, vault.VaultID, mem.ID, "integration context"); err != nil {
		t.Fatalf("put context: %v", err)
	}
	if err := c.AwaitConsistency(ctx, mem.ID); err != nil {
		t.Fatalf("await consistency after context: %v", err)
	}

	// 4. add keyword entries and verify via ListEntries
	for i := 0; i < 3; i++ {
		raw := fmt.Sprintf("the quick brown fox %d", i)
		if _, err := c.AddEntry(ctx, vault.VaultID, mem.ID, client.AddEntryRequest{RawEntry: raw, Summary: "story"}); err != nil {
			t.Fatalf("add entry %d: %v", i, err)
		}
	}
	if err := c.AwaitConsistency(ctx, mem.ID); err != nil {
		t.Fatalf("await consistency after entries: %v", err)
	}

	// verify entries were added
	entries, err := c.ListEntries(ctx, vault.VaultID, mem.ID, nil)
	if err != nil || entries.Count != 3 {
		t.Fatalf("list entries unexpected: err=%v count=%d (expected 3)", err, entries.Count)
	}

	// small delay to ensure indexer processed entries
	time.Sleep(2 * time.Second)

	// 5. perform search with retry mechanism (handles indexer lag)
	var sr *client.SearchResponse
	deadline := time.Now().Add(20 * time.Second)
	for {
		topKE := 3
		topKC := 2
		sr, err = c.Search(ctx, client.SearchRequest{MemoryID: mem.ID, Query: "fox", TopKE: &topKE, TopKC: &topKC})
		if err == nil && sr.Count > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("search retries exhausted: last err %v, sr %#v", err, sr)
		}
		time.Sleep(1 * time.Second)
	}

	// 6. validate search results and context deeply
	if sr.Count == 0 {
		t.Fatalf("search returned zero results")
	}
	if string(sr.LatestContext) != "\"integration context\"" && string(sr.LatestContext) != "integration context" {
		t.Fatalf("latestContext mismatch: %s", string(sr.LatestContext))
	}
	if sr.LatestContextTimestamp == nil {
		t.Fatalf("latestContextTimestamp nil")
	}

	// Verify ConversationTime is included in search results
	if len(sr.Entries) > 0 {
		for i, entry := range sr.Entries {
			if entry.ConversationTime.IsZero() {
				t.Errorf("entry %d has zero ConversationTime", i)
			}
			// ConversationTime should be set (defaults to CreationTime if not specified)
			if !entry.ConversationTime.Equal(entry.CreationTime) && entry.ConversationTime.After(time.Now()) {
				t.Errorf("entry %d has invalid ConversationTime: %v", i, entry.ConversationTime)
			}
		}
	}

	t.Logf("search completed successfully: found %d results with valid context and conversation times", sr.Count)

	// cleanup
	_ = c.DeleteMemory(ctx, vault.VaultID, mem.ID)
	_ = c.DeleteVault(ctx, vault.VaultID)
	// User deletion is now external - no user cleanup needed
}

// TestSearchWithConversationTimeE2E tests that conversation_time is properly indexed and returned in search results
func TestSearchWithConversationTimeE2E(t *testing.T) {
	baseURL := os.Getenv("TEST_BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11545"
	}

	c, err := client.NewWithDevMode(baseURL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer c.Close()

	// Create vault and memory
	vaultTitle := fmt.Sprintf("conv-time-vault-%s", uuid.NewString()[:8])
	vault, err := c.CreateVault(ctx, client.CreateVaultRequest{Title: vaultTitle})
	if err != nil {
		t.Fatalf("create vault failed: %v", err)
	}
	defer c.DeleteVault(ctx, vault.VaultID)

	mem, err := c.CreateMemory(ctx, vault.VaultID, client.CreateMemoryRequest{Title: "conv-time-mem", MemoryType: "NOTES"})
	if err != nil {
		t.Fatalf("create memory failed: %v", err)
	}
	defer c.DeleteMemory(ctx, vault.VaultID, mem.ID)

	// Add entries with specific conversation times
	pastTime := time.Now().Add(-24 * time.Hour)
	recentTime := time.Now().Add(-1 * time.Hour)

	// Add entry with past conversation time
	req1 := client.AddEntryRequest{
		RawEntry:         "Historical conversation about weather",
		Summary:          "weather discussion",
		ConversationTime: &pastTime,
	}
	entry1, err := c.AddEntry(ctx, vault.VaultID, mem.ID, req1)
	if err != nil {
		t.Fatalf("add entry 1: %v", err)
	}

	// Add entry with recent conversation time
	req2 := client.AddEntryRequest{
		RawEntry:         "Recent conversation about weather patterns",
		Summary:          "weather patterns",
		ConversationTime: &recentTime,
	}
	entry2, err := c.AddEntry(ctx, vault.VaultID, mem.ID, req2)
	if err != nil {
		t.Fatalf("add entry 2: %v", err)
	}

	// Add entry without conversation time (should default to creation time)
	req3 := client.AddEntryRequest{
		RawEntry: "Current conversation about weather forecast",
		Summary:  "weather forecast",
	}
	entry3, err := c.AddEntry(ctx, vault.VaultID, mem.ID, req3)
	if err != nil {
		t.Fatalf("add entry 3: %v", err)
	}

	// Wait for consistency
	if err := c.AwaitConsistency(ctx, mem.ID); err != nil {
		t.Fatalf("await consistency: %v", err)
	}

	// Small delay for indexer
	time.Sleep(3 * time.Second)

	// Search for entries
	topKE := 10
	topKC := 1
	sr, err := c.Search(ctx, client.SearchRequest{
		MemoryID: mem.ID,
		Query:    "weather",
		TopKE:    &topKE,
		TopKC:    &topKC,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// Verify we got results
	if sr.Count < 3 {
		t.Fatalf("expected at least 3 results, got %d", sr.Count)
	}

	// Check each entry has correct conversation time
	foundEntry1, foundEntry2, foundEntry3 := false, false, false
	for _, searchEntry := range sr.Entries {
		if searchEntry.ID == entry1.ID {
			foundEntry1 = true
			// Should have the past conversation time
			if !searchEntry.ConversationTime.Round(time.Second).Equal(pastTime.Round(time.Second)) {
				t.Errorf("entry1 conversation time mismatch: got %v, want %v",
					searchEntry.ConversationTime, pastTime)
			}
		} else if searchEntry.ID == entry2.ID {
			foundEntry2 = true
			// Should have the recent conversation time
			if !searchEntry.ConversationTime.Round(time.Second).Equal(recentTime.Round(time.Second)) {
				t.Errorf("entry2 conversation time mismatch: got %v, want %v",
					searchEntry.ConversationTime, recentTime)
			}
		} else if searchEntry.ID == entry3.ID {
			foundEntry3 = true
			// Should have conversation time equal to creation time
			if !searchEntry.ConversationTime.Equal(searchEntry.CreationTime) {
				t.Errorf("entry3 conversation time should equal creation time: conv=%v, creation=%v",
					searchEntry.ConversationTime, searchEntry.CreationTime)
			}
		}
	}

	if !foundEntry1 || !foundEntry2 || !foundEntry3 {
		t.Errorf("not all entries found in search results: entry1=%v, entry2=%v, entry3=%v",
			foundEntry1, foundEntry2, foundEntry3)
	}

	t.Logf("search with conversation_time completed successfully")
}

// TestSearchWithConversationTimeEdgeCasesE2E tests edge cases for conversation_time handling
func TestSearchWithConversationTimeEdgeCasesE2E(t *testing.T) {
	baseURL := os.Getenv("TEST_BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11545"
	}

	c, err := client.NewWithDevMode(baseURL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer c.Close()

	// Create vault and memory
	vaultTitle := fmt.Sprintf("edge-case-vault-%s", uuid.NewString()[:8])
	vault, err := c.CreateVault(ctx, client.CreateVaultRequest{Title: vaultTitle})
	if err != nil {
		t.Fatalf("create vault failed: %v", err)
	}
	defer c.DeleteVault(ctx, vault.VaultID)

	mem, err := c.CreateMemory(ctx, vault.VaultID, client.CreateMemoryRequest{Title: "edge-mem", MemoryType: "NOTES"})
	if err != nil {
		t.Fatalf("create memory failed: %v", err)
	}
	defer c.DeleteMemory(ctx, vault.VaultID, mem.ID)

	// Test 1: Very old conversation time (> 1 year ago)
	veryOldTime := time.Now().Add(-400 * 24 * time.Hour) // ~13 months ago
	req1 := client.AddEntryRequest{
		RawEntry:         "Ancient conversation from over a year ago",
		Summary:          "ancient history",
		ConversationTime: &veryOldTime,
	}
	entry1, err := c.AddEntry(ctx, vault.VaultID, mem.ID, req1)
	if err != nil {
		t.Fatalf("add very old entry: %v", err)
	}

	// Test 2: Conversation time in different timezone (UTC)
	utcTime := time.Now().UTC().Add(-6 * time.Hour)
	req2 := client.AddEntryRequest{
		RawEntry:         "UTC timezone conversation",
		Summary:          "utc time",
		ConversationTime: &utcTime,
	}
	entry2, err := c.AddEntry(ctx, vault.VaultID, mem.ID, req2)
	if err != nil {
		t.Fatalf("add UTC entry: %v", err)
	}

	// Test 3: Conversation time exactly equal to creation time
	// We'll add without conversation_time and verify it defaults correctly
	req3 := client.AddEntryRequest{
		RawEntry: "Entry without explicit conversation time",
		Summary:  "default time",
		// No ConversationTime - should default to creation time
	}
	entry3, err := c.AddEntry(ctx, vault.VaultID, mem.ID, req3)
	if err != nil {
		t.Fatalf("add default entry: %v", err)
	}

	// Test 4: Future conversation time (should be rejected by API)
	futureTime := time.Now().Add(24 * time.Hour)
	req4 := client.AddEntryRequest{
		RawEntry:         "Future conversation (should fail)",
		Summary:          "future",
		ConversationTime: &futureTime,
	}
	_, futureErr := c.AddEntry(ctx, vault.VaultID, mem.ID, req4)
	if futureErr == nil {
		t.Error("expected error for future conversation_time, but got none")
	} else {
		t.Logf("correctly rejected future conversation_time: %v", futureErr)
	}

	// Wait for consistency
	if err := c.AwaitConsistency(ctx, mem.ID); err != nil {
		t.Fatalf("await consistency: %v", err)
	}

	// Small delay for indexer
	time.Sleep(3 * time.Second)

	// Search for all entries
	topKE := 10
	topKC := 1
	sr, err := c.Search(ctx, client.SearchRequest{
		MemoryID: mem.ID,
		Query:    "conversation",
		TopKE:    &topKE,
		TopKC:    &topKC,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// Should have 3 entries (future one was rejected)
	if sr.Count < 3 {
		t.Fatalf("expected at least 3 results, got %d", sr.Count)
	}

	// Verify edge cases handled correctly
	for _, searchEntry := range sr.Entries {
		if searchEntry.ConversationTime.IsZero() {
			t.Errorf("entry %s has zero ConversationTime", searchEntry.ID)
		}

		// Verify no future times
		if searchEntry.ConversationTime.After(time.Now().Add(time.Minute)) {
			t.Errorf("entry %s has future ConversationTime: %v", searchEntry.ID, searchEntry.ConversationTime)
		}

		// Check specific entries
		switch searchEntry.ID {
		case entry1.ID:
			// Very old time should be preserved
			timeDiff := time.Since(searchEntry.ConversationTime)
			if timeDiff < 365*24*time.Hour {
				t.Errorf("entry1: expected very old ConversationTime, got %v (age: %v)",
					searchEntry.ConversationTime, timeDiff)
			}

		case entry2.ID:
			// UTC time should be preserved correctly
			if searchEntry.ConversationTime.Location() == nil {
				t.Logf("Note: timezone information may be normalized in storage")
			}

		case entry3.ID:
			// Should have defaulted to creation time
			if !searchEntry.ConversationTime.Equal(searchEntry.CreationTime) {
				t.Errorf("entry3: ConversationTime should equal CreationTime when not specified")
			}
		}
	}

	t.Logf("edge case tests completed successfully")
}
