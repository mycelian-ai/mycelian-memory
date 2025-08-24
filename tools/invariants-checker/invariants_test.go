//go:build invariants
// +build invariants

//
// 🔒 CRITICAL INVARIANT TESTS - Never Mutate to Get Features Working
// ⚠️  These tests must ALWAYS pass - they define system integrity
// 🛡️  Uses blackbox API testing - treats service as external system
// 📋  Rule: NEVER modify invariants to make incremental changes work
//

package invariants

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAllSystemInvariants runs all critical system invariant tests
// This is the master test that ensures system integrity
func TestAllSystemInvariants(t *testing.T) {
	// Start test server (this would be configured per test environment)
	baseURL := "http://localhost:11545" // TODO: Get from test config

	// Create invariant checker
	checker := NewInvariantChecker(baseURL)

	// Verify service is running
	resp, err := http.Get(baseURL + "/v0/health")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Service must be running for invariant tests")
	_ = resp.Body.Close()

	// Create test users
	userID1 := createTestUser(t, checker, "invariant-test-1@example.com")
	userID2 := createTestUser(t, checker, "invariant-test-2@example.com")

	t.Run("🔒 CRITICAL: MemoryEntryImmutabilityInvariant", func(t *testing.T) {
		checker.TestMemoryEntryImmutabilityInvariant(t, userID1)
	})

	t.Run("🔒 CRITICAL: SummaryOnlyUpdateInvariant", func(t *testing.T) {
		checker.TestSummaryOnlyUpdateInvariant(t, userID1)
	})

	t.Run("🔒 CRITICAL: UserDataIsolationInvariant", func(t *testing.T) {
		checker.TestUserDataIsolationInvariant(t, userID1, userID2)
	})

	t.Run("🔒 CRITICAL: SoftDeleteInvariant", func(t *testing.T) {
		checker.TestSoftDeleteInvariant(t, userID1)
	})

	// Cleanup test users
	cleanupTestUser(t, checker, userID1)
	cleanupTestUser(t, checker, userID2)
}

// TestInvariantViolationPrevention ensures our protection mechanisms work
func TestInvariantViolationPrevention(t *testing.T) {
	baseURL := "http://localhost:11545"
	checker := NewInvariantChecker(baseURL)

	userID := createTestUser(t, checker, "violation-test@example.com")
	defer cleanupTestUser(t, checker, userID)

	t.Run("🔒 CRITICAL: CannotBypassImmutabilityValidation", func(t *testing.T) {
		// Create memory and entry
		memoryID := checker.createTestMemory(t, userID, "Bypass Test", "CONVERSATION")
		entry := checker.createTestEntry(t, userID, memoryID, "Original content")

		// Correct the entry
		_ = checker.correctTestEntry(t, userID, memoryID, entry.CreationTime, "Corrected content", "Test correction")

		// Now try various bypass attempts - ALL MUST FAIL

		// 1. Direct update attempts (these endpoints should not exist)
		updateReq := map[string]interface{}{"rawEntry": "Bypass attempt"}
		checker.makeRequest(t, "PUT",
			fmt.Sprintf("/v0/users/%s/memories/%s/entries/%s", userID, memoryID, entry.EntryID),
			updateReq, http.StatusNotFound) // Should be 404 - no direct update endpoint

		// 2. Metadata bypass attempts
		metadataReq := map[string]interface{}{"correctionTime": nil}
		checker.makeRequest(t, "PUT",
			fmt.Sprintf("/v0/users/%s/memories/%s/entries/%s/reset", userID, memoryID, entry.EntryID),
			metadataReq, http.StatusNotFound) // Should be 404 - no reset endpoint

		// 3. Attempting to create entry with past timestamp (should use server timestamp)
		pastTime := time.Now().Add(-1 * time.Hour)
		createReq := map[string]interface{}{
			"rawEntry":     "Backdated entry",
			"creationTime": pastTime, // This should be ignored
		}
		resp := checker.makeRequest(t, "POST",
			fmt.Sprintf("/v0/users/%s/memories/%s/entries", userID, memoryID),
			createReq, http.StatusCreated)

		// Verify server used its own timestamp, not client's
		var newEntry map[string]interface{}
		err := json.Unmarshal(resp, &newEntry)
		require.NoError(t, err)

		entryTime, err := time.Parse(time.RFC3339, newEntry["creationTime"].(string))
		require.NoError(t, err)

		// Should be recent, not the past time we sent
		assert.True(t, entryTime.After(pastTime.Add(30*time.Minute)),
			"Server must use its own timestamp, not client's")
	})
}

// TestConcurrentInvariantViolationAttempts tests invariants under concurrency
func TestConcurrentInvariantViolationAttempts(t *testing.T) {
	baseURL := "http://localhost:11545"
	checker := NewInvariantChecker(baseURL)

	userID := createTestUser(t, checker, "concurrent-test@example.com")
	defer cleanupTestUser(t, checker, userID)

	t.Run("🔒 CRITICAL: ConcurrentCorrectionAttempts", func(t *testing.T) {
		// Create memory and entry
		memoryID := checker.createTestMemory(t, userID, "Concurrent Test", "CONVERSATION")
		entry := checker.createTestEntry(t, userID, memoryID, "Entry to correct concurrently")

		// Attempt multiple concurrent corrections - only one should succeed
		results := make(chan error, 3)

		for i := 0; i < 3; i++ {
			go func(attempt int) {
				defer func() {
					if r := recover(); r != nil {
						results <- fmt.Errorf("panic in attempt %d: %v", attempt, r)
					}
				}()

				correctionReq := CorrectionRequest{
					OriginalCreationTime: entry.CreationTime,
					CorrectedContent:     fmt.Sprintf("Concurrent correction %d", attempt),
					CorrectionReason:     fmt.Sprintf("Concurrent attempt %d", attempt),
				}

				resp := checker.makeRequestNoAssert(
					"POST",
					fmt.Sprintf("/v0/users/%s/memories/%s/entries/correct", userID, memoryID),
					correctionReq)

				if resp.StatusCode == http.StatusCreated {
					results <- nil // Success
				} else if resp.StatusCode == http.StatusBadRequest {
					results <- fmt.Errorf("immutability_violation") // Expected failure
				} else {
					results <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
				}
			}(i)
		}

		// Collect results
		var successCount, violationCount int
		for i := 0; i < 3; i++ {
			err := <-results
			if err == nil {
				successCount++
			} else if err.Error() == "immutability_violation" {
				violationCount++
			} else {
				t.Errorf("Unexpected error: %v", err)
			}
		}

		// Exactly one should succeed, others should fail with immutability violation
		assert.Equal(t, 1, successCount, "Exactly one concurrent correction should succeed")
		assert.Equal(t, 2, violationCount, "Other attempts should fail with immutability violation")
	})
}

// Helper functions

func createTestUser(t *testing.T, checker *InvariantChecker, email string) string {
	createReq := map[string]interface{}{
		"email":    email,
		"timeZone": "UTC",
	}

	resp := checker.makeRequest(t, "POST", "/v0/users", createReq, http.StatusCreated)

	var user map[string]interface{}
	err := json.Unmarshal(resp, &user)
	require.NoError(t, err)

	return user["userId"].(string)
}

func cleanupTestUser(t *testing.T, checker *InvariantChecker, userID string) {
	// Delete user - this should cascade to all memories and entries
	checker.makeRequest(t, "DELETE", fmt.Sprintf("/v0/users/%s", userID), nil, http.StatusNoContent)
}

// makeRequestNoAssert is like makeRequest but doesn't assert on status (for concurrent tests)
func (ic *InvariantChecker) makeRequestNoAssert(method, path string, body interface{}) *http.Response {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil
		}
	}

	req, err := http.NewRequest(method, ic.baseURL+path, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := ic.client.Do(req)
	if err != nil {
		return nil
	}

	return resp
}
