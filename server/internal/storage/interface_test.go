package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageInterface_UserOperations(t *testing.T) {
	ctx := context.Background()
	storage := NewSpannerStorage(spannerClient)

	t.Run("CreateUser", func(t *testing.T) {
		email := "test@example.com"
		displayName := "Test User"

		req := CreateUserRequest{
			UserID:      "user_create_1",
			Email:       email,
			DisplayName: &displayName,
			TimeZone:    "UTC",
		}

		user, err := storage.CreateUser(ctx, req)
		require.NoError(t, err)

		assert.NotEmpty(t, user.UserID) // Client-generated UUID
		assert.Equal(t, email, user.Email)
		assert.Equal(t, &displayName, user.DisplayName)
		assert.Equal(t, "UTC", user.TimeZone)
		assert.Equal(t, "ACTIVE", user.Status)
		assert.WithinDuration(t, time.Now(), user.CreationTime, time.Second)
		assert.Nil(t, user.LastActiveTime)
	})

	t.Run("GetUser", func(t *testing.T) {
		// Create a user first
		email := "get@example.com"

		req := CreateUserRequest{
			UserID:   "user_get_1",
			Email:    email,
			TimeZone: "UTC",
		}

		createdUser, err := storage.CreateUser(ctx, req)
		require.NoError(t, err)

		// Get the user using the generated ID
		user, err := storage.GetUser(ctx, createdUser.UserID)
		require.NoError(t, err)

		assert.Equal(t, createdUser.UserID, user.UserID)
		assert.Equal(t, createdUser.Email, user.Email)
		assert.Equal(t, createdUser.TimeZone, user.TimeZone)
	})

	t.Run("GetUserByEmail", func(t *testing.T) {
		// Create a user first
		email := "email@example.com"

		req := CreateUserRequest{
			UserID:   "user_email_1",
			Email:    email,
			TimeZone: "UTC",
		}

		createdUser, err := storage.CreateUser(ctx, req)
		require.NoError(t, err)

		// Get the user by email
		user, err := storage.GetUserByEmail(ctx, email)
		require.NoError(t, err)

		assert.Equal(t, createdUser.UserID, user.UserID)
		assert.Equal(t, createdUser.Email, user.Email)
	})

	t.Run("UpdateUserLastActive", func(t *testing.T) {
		// Create a user first
		email := "active@example.com"

		req := CreateUserRequest{
			UserID:   "user_active_1",
			Email:    email,
			TimeZone: "UTC",
		}

		createdUser, err := storage.CreateUser(ctx, req)
		require.NoError(t, err)

		// Update last active using generated ID
		err = storage.UpdateUserLastActive(ctx, createdUser.UserID)
		require.NoError(t, err)

		// Verify the update
		user, err := storage.GetUser(ctx, createdUser.UserID)
		require.NoError(t, err)
		assert.NotNil(t, user.LastActiveTime)
		assert.WithinDuration(t, time.Now(), *user.LastActiveTime, time.Second)
	})
}

func TestStorageInterface_MemoryOperations(t *testing.T) {
	ctx := context.Background()
	storage := NewSpannerStorage(spannerClient)

	// Create a user for testing
	userReq := CreateUserRequest{
		UserID:   "user_memory_ops",
		Email:    "memory@example.com",
		TimeZone: "UTC",
	}
	createdUser, err := storage.CreateUser(ctx, userReq)
	require.NoError(t, err)
	userID := createdUser.UserID

	t.Run("CreateMemory", func(t *testing.T) {
		vaultID := uuid.New()

		// Create vault first
		_, err := storage.CreateVault(ctx, CreateVaultRequest{
			UserID:  userID,
			VaultID: vaultID,
			Title:   "Test Vault",
		})
		require.NoError(t, err)

		title := "Test Memory"
		description := "Test memory description"

		req := CreateMemoryRequest{
			UserID:      userID,
			VaultID:     vaultID,
			MemoryType:  "PROJECT",
			Title:       title,
			Description: &description,
		}

		memory, err := storage.CreateMemory(ctx, req)
		require.NoError(t, err)

		assert.Equal(t, userID, memory.UserID)
		assert.NotEmpty(t, memory.MemoryID) // Client-generated UUID
		assert.Equal(t, "PROJECT", memory.MemoryType)
		assert.Equal(t, title, memory.Title)
		assert.Equal(t, &description, memory.Description)
		assert.WithinDuration(t, time.Now(), memory.CreationTime, time.Second)
	})

	t.Run("GetMemory", func(t *testing.T) {
		// Create a vault and memory first
		vaultID := uuid.New()

		_, err := storage.CreateVault(ctx, CreateVaultRequest{
			UserID:  userID,
			VaultID: vaultID,
			Title:   "Get Test Vault",
		})
		require.NoError(t, err)

		title := "Get Test Memory"

		req := CreateMemoryRequest{
			UserID:     userID,
			VaultID:    vaultID,
			MemoryType: "CONVERSATION",
			Title:      title,
		}

		createdMemory, err := storage.CreateMemory(ctx, req)
		require.NoError(t, err)

		// Get the memory using the generated ID
		memory, err := storage.GetMemory(ctx, userID, vaultID, createdMemory.MemoryID)
		require.NoError(t, err)

		assert.Equal(t, createdMemory.UserID, memory.UserID)
		assert.Equal(t, createdMemory.MemoryID, memory.MemoryID)
		assert.Equal(t, createdMemory.Title, memory.Title)
	})

	t.Run("ListMemories", func(t *testing.T) {
		// Create a vault for listing test
		vaultID := uuid.New()

		_, err := storage.CreateVault(ctx, CreateVaultRequest{
			UserID:  userID,
			VaultID: vaultID,
			Title:   "List Test Vault",
		})
		require.NoError(t, err)

		// Create multiple memories
		for i := 0; i < 2; i++ {
			req := CreateMemoryRequest{
				UserID:     userID,
				VaultID:    vaultID,
				MemoryType: "CONTEXT",
				Title:      fmt.Sprintf("List Test Memory %d", i+1),
			}
			_, err := storage.CreateMemory(ctx, req)
			require.NoError(t, err)
		}

		// List memories
		memories, err := storage.ListMemories(ctx, userID, vaultID)
		require.NoError(t, err)

		// Should have at least the ones we created
		assert.GreaterOrEqual(t, len(memories), 2)

		// Verify all returned memories belong to the user
		for _, memory := range memories {
			assert.Equal(t, userID, memory.UserID)
		}
	})

	t.Run("DeleteMemory", func(t *testing.T) {
		// Create a vault and memory first
		vaultID := uuid.New()

		_, err := storage.CreateVault(ctx, CreateVaultRequest{
			UserID:  userID,
			VaultID: vaultID,
			Title:   "Delete Test Vault",
		})
		require.NoError(t, err)

		req := CreateMemoryRequest{
			UserID:     userID,
			VaultID:    vaultID,
			MemoryType: "TEMPORARY",
			Title:      "Delete Test Memory",
		}

		createdMemory, err := storage.CreateMemory(ctx, req)
		require.NoError(t, err)

		// Delete the memory using the generated ID
		err = storage.DeleteMemory(ctx, userID, vaultID, createdMemory.MemoryID)
		require.NoError(t, err)

		// Verify it's deleted
		_, err = storage.GetMemory(ctx, userID, vaultID, createdMemory.MemoryID)
		assert.Error(t, err) // Should not find the memory
	})
}

func TestStorageInterface_MemoryEntryOperations(t *testing.T) {
	ctx := context.Background()
	storage := NewSpannerStorage(spannerClient)

	// Create a user and memory for testing
	var memoryID string

	userReq := CreateUserRequest{
		UserID:   "user_entry_ops",
		Email:    "entry@example.com",
		TimeZone: "UTC",
	}
	createdUser, err := storage.CreateUser(ctx, userReq)
	require.NoError(t, err)
	userID := createdUser.UserID

	vaultID := uuid.New()

	// create vault
	_, err = storage.CreateVault(ctx, CreateVaultRequest{
		UserID:  userID,
		VaultID: vaultID,
		Title:   "Entry Vault",
	})
	require.NoError(t, err)

	memoryReq := CreateMemoryRequest{
		UserID:     userID,
		VaultID:    vaultID,
		MemoryType: "PROJECT",
		Title:      "Entry Test Memory",
	}
	createdMemory, err := storage.CreateMemory(ctx, memoryReq)
	require.NoError(t, err)
	memoryID = createdMemory.MemoryID

	t.Run("CreateMemoryEntry", func(t *testing.T) {
		rawEntry := "This is a test memory entry"
		summary := "Test entry summary"
		metadata := map[string]interface{}{"key": "value"}
		tags := map[string]interface{}{"status": "draft", "priority": "high"}

		req := CreateMemoryEntryRequest{
			UserID:   userID,
			VaultID:  vaultID,
			MemoryID: memoryID,
			RawEntry: rawEntry,
			Summary:  &summary,
			Metadata: metadata,
			Tags:     tags,
		}

		entry, err := storage.CreateMemoryEntry(ctx, req)
		require.NoError(t, err)

		assert.Equal(t, userID, entry.UserID)
		assert.Equal(t, memoryID, entry.MemoryID)
		assert.NotEmpty(t, entry.EntryID) // Client-generated UUID
		assert.Equal(t, rawEntry, entry.RawEntry)
		assert.Equal(t, &summary, entry.Summary)
		assert.Equal(t, metadata, entry.Metadata)
		assert.Equal(t, tags, entry.Tags)
		assert.WithinDuration(t, time.Now(), entry.CreationTime, time.Second)
	})

	t.Run("ListMemoryEntries", func(t *testing.T) {
		// Create multiple entries
		for i := 0; i < 2; i++ {
			req := CreateMemoryEntryRequest{
				UserID:   userID,
				VaultID:  vaultID,
				MemoryID: memoryID,
				RawEntry: fmt.Sprintf("List test entry %d", i+1),
			}
			_, err := storage.CreateMemoryEntry(ctx, req)
			require.NoError(t, err)
		}

		// List entries
		listReq := ListMemoryEntriesRequest{
			UserID:   userID,
			VaultID:  vaultID,
			MemoryID: memoryID,
			Limit:    10,
		}

		entries, err := storage.ListMemoryEntries(ctx, listReq)
		require.NoError(t, err)

		// Should have at least the ones we created
		assert.GreaterOrEqual(t, len(entries), 2)

		// Verify all returned entries belong to the memory
		for _, entry := range entries {
			assert.Equal(t, userID, entry.UserID)
			assert.Equal(t, memoryID, entry.MemoryID)
		}

		// Verify chronological ordering (newest first)
		for i := 1; i < len(entries); i++ {
			assert.True(t, entries[i-1].CreationTime.After(entries[i].CreationTime) ||
				entries[i-1].CreationTime.Equal(entries[i].CreationTime))
		}
	})

	t.Run("UpdateMemoryEntryTags", func(t *testing.T) {
		// Create an entry first
		rawEntry := "Entry for tags testing"
		initialTags := map[string]interface{}{"status": "draft"}

		createReq := CreateMemoryEntryRequest{
			UserID:   userID,
			VaultID:  vaultID,
			MemoryID: memoryID,
			RawEntry: rawEntry,
			Tags:     initialTags,
		}

		createdEntry, err := storage.CreateMemoryEntry(ctx, createReq)
		require.NoError(t, err)
		assert.Equal(t, initialTags, createdEntry.Tags)

		// Update tags
		updatedTags := map[string]interface{}{
			"status":   "in_progress",
			"priority": "high",
			"assignee": "user123",
		}

		updateReq := UpdateMemoryEntryTagsRequest{
			UserID:   userID,
			VaultID:  vaultID,
			MemoryID: memoryID,
			EntryID:  createdEntry.EntryID,
			Tags:     updatedTags,
		}

		updatedEntry, err := storage.UpdateMemoryEntryTags(ctx, updateReq)
		require.NoError(t, err)

		// Verify tags were updated but other fields remain unchanged
		assert.Equal(t, updatedTags, updatedEntry.Tags)
		assert.Equal(t, rawEntry, updatedEntry.RawEntry)
		assert.NotEmpty(t, updatedEntry.EntryID) // Client-generated UUID
		assert.NotNil(t, updatedEntry.LastUpdateTime)

		// Verify the updated entry can be retrieved
		retrievedEntry, err := storage.GetMemoryEntry(ctx, userID, vaultID, memoryID, createdEntry.CreationTime)
		require.NoError(t, err)
		assert.Equal(t, updatedTags, retrievedEntry.Tags)
	})

	t.Run("UpdateMemoryEntryTags_NonexistentEntry", func(t *testing.T) {
		updateReq := UpdateMemoryEntryTagsRequest{
			UserID:   userID,
			VaultID:  vaultID,
			MemoryID: memoryID,
			EntryID:  "00000000-0000-0000-0000-000000000000",
			Tags:     map[string]interface{}{"status": "test"},
		}

		_, err := storage.UpdateMemoryEntryTags(ctx, updateReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ENTRY_NOT_FOUND")
	})
}

func TestStorageInterface_HealthCheck(t *testing.T) {
	ctx := context.Background()
	storage := NewSpannerStorage(spannerClient)

	err := storage.HealthCheck(ctx)
	assert.NoError(t, err)
}
