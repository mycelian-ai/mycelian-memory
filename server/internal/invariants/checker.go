//
// 🔒 CRITICAL SYSTEM FILE - Invariant Contract Testing
// ⚠️  These tests ensure system invariants are never violated
// 🛡️  Uses customer-facing APIs only (blackbox testing)
// 📋  Never mutate invariants to get incremental changes working
//

package invariants

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// InvariantChecker tests system invariants using customer-facing APIs
// This is a blackbox test that treats the service as an external system
type InvariantChecker struct {
	baseURL string
	client  *http.Client
}

// NewInvariantChecker creates a new invariant checker
func NewInvariantChecker(baseURL string) *InvariantChecker {
	return &InvariantChecker{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// 🔒 INVARIANT: Memory entries are immutable once corrected
func (ic *InvariantChecker) TestMemoryEntryImmutabilityInvariant(t *testing.T, userID string) {
	// Step 1: Create a memory
	memoryID := ic.createTestMemory(t, userID, "Test Memory", "CONVERSATION")

	// Step 2: Create an entry
	entryResp := ic.createTestEntry(t, userID, memoryID, "Original content that needs correction")

	// Step 3: Correct the entry
	_ = ic.correctTestEntry(t, userID, memoryID, entryResp.CreationTime, "Corrected content", "Fixed error")

	// 🔒 INVARIANT: Cannot correct already corrected entries
	t.Run("CorrectedEntriesCannotBeCorrectedAgain", func(t *testing.T) {
		// Attempt to correct the original entry again
		correctionReq := CorrectionRequest{
			OriginalCreationTime: entryResp.CreationTime,
			CorrectedContent:     "Second correction attempt",
			CorrectionReason:     "Trying to correct again",
		}

		// This MUST fail with immutability violation
		resp := ic.makeRequest(t, "POST",
			fmt.Sprintf("/api/users/%s/memories/%s/entries/correct", userID, memoryID),
			correctionReq, http.StatusBadRequest)

		// Verify error message indicates immutability violation
		var errorResp map[string]interface{}
		err := json.Unmarshal(resp, &errorResp)
		require.NoError(t, err)

		errorMessage := errorResp["error"].(string)
		assert.Contains(t, errorMessage, "IMMUTABILITY_VIOLATION", "Must return immutability violation error")
		assert.Contains(t, errorMessage, "already corrected", "Must indicate entry was already corrected")
	})

	// 🔒 INVARIANT: Correction entry itself can be corrected (if not yet corrected)
	t.Run("CorrectionEntryCanBeCorrected", func(t *testing.T) {
		// Create another entry to correct
		entry2 := ic.createTestEntry(t, userID, memoryID, "Another entry to correct")
		correction2 := ic.correctTestEntry(t, userID, memoryID, entry2.CreationTime, "First correction", "Initial fix")

		// Now correct the correction entry
		ic.correctTestEntry(t, userID, memoryID, correction2.CreationTime, "Second correction", "Further fix")

		// But cannot correct it again
		correctionReq := CorrectionRequest{
			OriginalCreationTime: correction2.CreationTime,
			CorrectedContent:     "Third correction attempt",
			CorrectionReason:     "Should fail",
		}

		ic.makeRequest(t, "POST",
			fmt.Sprintf("/api/users/%s/memories/%s/entries/correct", userID, memoryID),
			correctionReq, http.StatusBadRequest)
	})
}

// 🔒 INVARIANT: Summary is the only updatable field
func (ic *InvariantChecker) TestSummaryOnlyUpdateInvariant(t *testing.T, userID string) {
	// Step 1: Create memory and entry
	memoryID := ic.createTestMemory(t, userID, "Update Test Memory", "CONVERSATION")
	entryResp := ic.createTestEntry(t, userID, memoryID, "Content for update testing")

	// 🔒 INVARIANT: Summary can be updated on fresh entries
	t.Run("SummaryCanBeUpdated", func(t *testing.T) {
		updateReq := map[string]interface{}{
			"summary": "Updated summary for search optimization",
		}

		resp := ic.makeRequest(t, "PUT",
			fmt.Sprintf("/api/users/%s/memories/%s/entries/%s/summary", userID, memoryID, entryResp.EntryID),
			updateReq, http.StatusOK)

		var updatedEntry map[string]interface{}
		err := json.Unmarshal(resp, &updatedEntry)
		require.NoError(t, err)

		assert.Equal(t, "Updated summary for search optimization", updatedEntry["summary"])
		assert.NotNil(t, updatedEntry["lastUpdateTime"], "LastUpdateTime must be set")
	})

	// 🔒 INVARIANT: Content cannot be updated (no endpoint should exist)
	t.Run("ContentCannotBeUpdated", func(t *testing.T) {
		updateReq := map[string]interface{}{
			"rawEntry": "Attempted content update",
		}

		// This endpoint should not exist or should fail
		ic.makeRequest(t, "PUT",
			fmt.Sprintf("/api/users/%s/memories/%s/entries/%s/content", userID, memoryID, entryResp.EntryID),
			updateReq, http.StatusNotFound) // Should be 404 - endpoint doesn't exist
	})

	// 🔒 INVARIANT: Metadata cannot be updated
	t.Run("MetadataCannotBeUpdated", func(t *testing.T) {
		updateReq := map[string]interface{}{
			"metadata": map[string]interface{}{"attempted": "update"},
		}

		// This endpoint should not exist or should fail
		ic.makeRequest(t, "PUT",
			fmt.Sprintf("/api/users/%s/memories/%s/entries/%s/metadata", userID, memoryID, entryResp.EntryID),
			updateReq, http.StatusNotFound) // Should be 404 - endpoint doesn't exist
	})
}

// 🔒 INVARIANT: User data isolation
func (ic *InvariantChecker) TestUserDataIsolationInvariant(t *testing.T, userID1, userID2 string) {
	// Step 1: Create memories for both users
	memoryID1 := ic.createTestMemory(t, userID1, "User1 Memory", "CONVERSATION")
	memoryID2 := ic.createTestMemory(t, userID2, "User2 Memory", "CONVERSATION")

	// Step 2: Create entries for both users
	ic.createTestEntry(t, userID1, memoryID1, "User1 private content")
	ic.createTestEntry(t, userID2, memoryID2, "User2 private content")

	// 🔒 INVARIANT: Users cannot access each other's memories
	t.Run("CrossUserMemoryAccessForbidden", func(t *testing.T) {
		// User1 trying to access User2's memory
		ic.makeRequest(t, "GET",
			fmt.Sprintf("/api/users/%s/memories/%s", userID1, memoryID2),
			nil, http.StatusNotFound) // Should not find other user's memory

		// User2 trying to access User1's memory
		ic.makeRequest(t, "GET",
			fmt.Sprintf("/api/users/%s/memories/%s", userID2, memoryID1),
			nil, http.StatusNotFound) // Should not find other user's memory
	})

	// 🔒 INVARIANT: User lists only show own data
	t.Run("UserListsShowOnlyOwnData", func(t *testing.T) {
		// Get User1's memories
		resp1 := ic.makeRequest(t, "GET",
			fmt.Sprintf("/api/users/%s/memories", userID1),
			nil, http.StatusOK)

		var user1Memories map[string]interface{}
		err := json.Unmarshal(resp1, &user1Memories)
		require.NoError(t, err)

		memories1 := user1Memories["memories"].([]interface{})
		assert.Len(t, memories1, 1, "User1 should see only their own memory")

		// Verify it's the correct memory
		memory1 := memories1[0].(map[string]interface{})
		assert.Equal(t, memoryID1, memory1["memoryId"])
		assert.Equal(t, userID1, memory1["userId"])
	})
}

// 🔒 INVARIANT: Soft delete behavior
func (ic *InvariantChecker) TestSoftDeleteInvariant(t *testing.T, userID string) {
	// Step 1: Create memory and entry
	memoryID := ic.createTestMemory(t, userID, "Delete Test Memory", "CONVERSATION")
	entryResp := ic.createTestEntry(t, userID, memoryID, "Content to be deleted")

	// 🔒 INVARIANT: Deleted items disappear from lists immediately
	t.Run("DeletedItemsNotInLists", func(t *testing.T) {
		// Verify entry exists initially
		resp := ic.makeRequest(t, "GET",
			fmt.Sprintf("/api/users/%s/memories/%s/entries", userID, memoryID),
			nil, http.StatusOK)

		var entriesList map[string]interface{}
		err := json.Unmarshal(resp, &entriesList)
		require.NoError(t, err)

		entries := entriesList["entries"].([]interface{})
		assert.Len(t, entries, 1, "Entry should exist before deletion")

		// Delete the entry
		ic.makeRequest(t, "DELETE",
			fmt.Sprintf("/api/users/%s/memories/%s/entries/%s", userID, memoryID, entryResp.EntryID),
			nil, http.StatusNoContent)

		// Verify entry no longer appears in lists
		resp = ic.makeRequest(t, "GET",
			fmt.Sprintf("/api/users/%s/memories/%s/entries", userID, memoryID),
			nil, http.StatusOK)

		err = json.Unmarshal(resp, &entriesList)
		require.NoError(t, err)

		entries = entriesList["entries"].([]interface{})
		assert.Len(t, entries, 0, "Entry should not appear in lists after deletion")
	})

	// 🔒 INVARIANT: Deleting already deleted is idempotent
	t.Run("DeletionIsIdempotent", func(t *testing.T) {
		// Delete again - should be noop, no error
		ic.makeRequest(t, "DELETE",
			fmt.Sprintf("/api/users/%s/memories/%s/entries/%s", userID, memoryID, entryResp.EntryID),
			nil, http.StatusNoContent)
	})
}

// Helper methods for API interactions

func (ic *InvariantChecker) createTestMemory(t *testing.T, userID, title, memoryType string) string {
	createReq := map[string]interface{}{
		"title":      title,
		"memoryType": memoryType,
	}

	resp := ic.makeRequest(t, "POST",
		fmt.Sprintf("/api/users/%s/memories", userID),
		createReq, http.StatusCreated)

	var memory map[string]interface{}
	err := json.Unmarshal(resp, &memory)
	require.NoError(t, err)

	return memory["memoryId"].(string)
}

func (ic *InvariantChecker) createTestEntry(t *testing.T, userID, memoryID, content string) *EntryResponse {
	createReq := map[string]interface{}{
		"rawEntry": content,
		"summary":  "Test summary",
	}

	resp := ic.makeRequest(t, "POST",
		fmt.Sprintf("/api/users/%s/memories/%s/entries", userID, memoryID),
		createReq, http.StatusCreated)

	var entry EntryResponse
	err := json.Unmarshal(resp, &entry)
	require.NoError(t, err)

	return &entry
}

func (ic *InvariantChecker) correctTestEntry(t *testing.T, userID, memoryID string, originalCreationTime time.Time, correctedContent, reason string) *EntryResponse {
	correctionReq := CorrectionRequest{
		OriginalCreationTime: originalCreationTime,
		CorrectedContent:     correctedContent,
		CorrectionReason:     reason,
	}

	resp := ic.makeRequest(t, "POST",
		fmt.Sprintf("/api/users/%s/memories/%s/entries/correct", userID, memoryID),
		correctionReq, http.StatusCreated)

	var correctionEntry EntryResponse
	err := json.Unmarshal(resp, &correctionEntry)
	require.NoError(t, err)

	return &correctionEntry
}

func (ic *InvariantChecker) makeRequest(t *testing.T, method, path string, body interface{}, expectedStatus int) []byte {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		require.NoError(t, err)
	}

	req, err := http.NewRequest(method, ic.baseURL+path, bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := ic.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify expected status
	assert.Equal(t, expectedStatus, resp.StatusCode,
		"Expected status %d but got %d for %s %s", expectedStatus, resp.StatusCode, method, path)

	// Read the full response body
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return respBody
}

// Request/Response models for API interactions

type CorrectionRequest struct {
	OriginalCreationTime time.Time `json:"originalCreationTime"`
	CorrectedContent     string    `json:"correctedContent"`
	CorrectionReason     string    `json:"correctionReason"`
}

type EntryResponse struct {
	EntryID      string    `json:"entryId"`
	CreationTime time.Time `json:"creationTime"`
	RawEntry     string    `json:"rawEntry"`
	Summary      *string   `json:"summary,omitempty"`
}
