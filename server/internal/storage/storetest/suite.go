// Package storetest provides a compliance test suite for storage.Store implementations.
//
// Usage:
//
//	func TestMyStore(t *testing.T) {
//	    storetest.Run(t, func(t *testing.T) storage.Store {
//	        return setupTestStore(t)
//	    })
//	}
//
// The suite exercises all storage operations including CRUD, pagination,
// filtering, and cascading deletes.
package storetest

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/mycelian/mycelian-memory/server/internal/model"
	"github.com/mycelian/mycelian-memory/server/internal/storage"
)

// Run exercises a storage.Store implementation with a compliance test suite.
// It creates a test store using makeStore and exercises common behaviors: vault and memory CRUD, entry creation/listing/updating/deletion,
// memory context put/latest/delete, pagination (Limit), temporal filtering (Before/After), tag updates, and cascading deletes.
// The function fails the provided *testing.T via t.Fatalf on any unexpected result or error.
func Run(t *testing.T, makeStore func(t *testing.T) storage.Store) {
	t.Helper()

	s := makeStore(t)
	ctx := context.Background()

	// Unique test identifiers (actor_id is an opaque string from API key)
	actorID := "actor-" + uuid.New().String()

	// Vaults
	v, err := s.Vaults().Create(ctx, &model.Vault{ActorID: actorID, Title: "test-vault"})
	if err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if v.VaultID == "" {
		t.Fatalf("CreateVault: empty vault id")
	}
	if got, err := s.Vaults().GetByID(ctx, actorID, v.VaultID); err != nil || got == nil || got.Title != "test-vault" {
		t.Fatalf("GetVault: got=%v err=%v", got, err)
	}
	if lst, err := s.Vaults().List(ctx, actorID); err != nil || len(lst) == 0 {
		t.Fatalf("ListVaults: n=%d err=%v", len(lst), err)
	}

	// Memories
	m, err := s.Memories().Create(ctx, &model.Memory{ActorID: actorID, VaultID: v.VaultID, MemoryType: "text", Title: "m1"})
	if err != nil {
		t.Fatalf("CreateMemory: %v", err)
	}
	if got, err := s.Memories().GetByID(ctx, actorID, v.VaultID, m.MemoryID); err != nil || got == nil || got.Title != "m1" {
		t.Fatalf("GetMemory: got=%v err=%v", got, err)
	}
	if got, err := s.Memories().GetByTitle(ctx, actorID, v.VaultID, "m1"); err != nil || got == nil || got.MemoryID != m.MemoryID {
		t.Fatalf("GetMemoryByTitle: got=%v err=%v", got, err)
	}

	// Entries
	e1, err := s.Entries().Create(ctx, &model.MemoryEntry{ActorID: actorID, VaultID: v.VaultID, MemoryID: m.MemoryID, RawEntry: "hello"})
	if err != nil {
		t.Fatalf("CreateEntry e1: %v", err)
	}
	e2, err := s.Entries().Create(ctx, &model.MemoryEntry{ActorID: actorID, VaultID: v.VaultID, MemoryID: m.MemoryID, RawEntry: "world"})
	if err != nil {
		t.Fatalf("CreateEntry e2: %v", err)
	}

	// ListEntries
	lst, err := s.Entries().List(ctx, model.ListEntriesRequest{ActorID: actorID, VaultID: v.VaultID, MemoryID: m.MemoryID})
	if err != nil || len(lst) < 2 {
		t.Fatalf("ListEntries: n=%d err=%v", len(lst), err)
	}

	// UpdateTags
	tags := map[string]interface{}{"k": "v", "num": 42}
	if _, err := s.Entries().UpdateTags(ctx, actorID, v.VaultID, m.MemoryID, e1.EntryID, tags); err != nil {
		t.Fatalf("UpdateTags: %v", err)
	}
	if got, err := s.Entries().GetByID(ctx, actorID, v.VaultID, m.MemoryID, e1.EntryID); err != nil || got == nil || len(got.Tags) == 0 {
		b, _ := json.Marshal(got)
		t.Fatalf("GetByID after UpdateTags: got=%s err=%v", string(b), err)
	}

	// Contexts
	ctxBody := `{"foo":"bar"}`
	c, err := s.Contexts().Put(ctx, &model.MemoryContext{ActorID: actorID, VaultID: v.VaultID, MemoryID: m.MemoryID, Context: ctxBody})
	if err != nil {
		t.Fatalf("PutContext: %v", err)
	}
	// Verify Latest() returns the most recent context
	if latest, err := s.Contexts().Latest(ctx, actorID, v.VaultID, m.MemoryID); err != nil || latest == nil || latest.ContextID != c.ContextID {
		t.Fatalf("LatestContext: expected ContextID=%s, got=%v err=%v", c.ContextID, latest, err)
	}
	// Put a second context and verify Latest() returns it (not the first one)
	ctxBody2 := `{"foo":"baz","updated":true}`
	c2, err := s.Contexts().Put(ctx, &model.MemoryContext{ActorID: actorID, VaultID: v.VaultID, MemoryID: m.MemoryID, Context: ctxBody2})
	if err != nil {
		t.Fatalf("PutContext c2: %v", err)
	}
	if latest2, err := s.Contexts().Latest(ctx, actorID, v.VaultID, m.MemoryID); err != nil || latest2 == nil || latest2.ContextID != c2.ContextID {
		t.Fatalf("LatestContext after second put: expected ContextID=%s, got=%v err=%v", c2.ContextID, latest2, err)
	}
	// Verify timestamp ordering: c2 should be created after c
	if !c2.CreationTime.After(c.CreationTime) {
		t.Fatalf("Context timestamp ordering: c2.CreationTime=%v should be after c.CreationTime=%v", c2.CreationTime, c.CreationTime)
	}
	if err := s.Contexts().DeleteByID(ctx, actorID, v.VaultID, m.MemoryID, c.ContextID); err != nil {
		t.Fatalf("DeleteContextByID: %v", err)
	}

	// Delete entry
	if err := s.Entries().DeleteByID(ctx, actorID, v.VaultID, m.MemoryID, e2.EntryID); err != nil {
		t.Fatalf("DeleteEntryByID: %v", err)
	}

	// Paging and time filters
	// Create additional entries
	e3, err := s.Entries().Create(ctx, &model.MemoryEntry{ActorID: actorID, VaultID: v.VaultID, MemoryID: m.MemoryID, RawEntry: "three"})
	if err != nil {
		t.Fatalf("CreateEntry e3: %v", err)
	}
	e4, err := s.Entries().Create(ctx, &model.MemoryEntry{ActorID: actorID, VaultID: v.VaultID, MemoryID: m.MemoryID, RawEntry: "four"})
	if err != nil {
		t.Fatalf("CreateEntry e4: %v", err)
	}
	// Verify timestamp ordering
	if !e4.CreationTime.After(e3.CreationTime) && !e4.CreationTime.Equal(e3.CreationTime) {
		t.Fatalf("Entry timestamp ordering: e4.CreationTime=%v should be >= e3.CreationTime=%v", e4.CreationTime, e3.CreationTime)
	}

	// Limit should cap results
	if lst2, err := s.Entries().List(ctx, model.ListEntriesRequest{ActorID: actorID, VaultID: v.VaultID, MemoryID: m.MemoryID, Limit: 2}); err != nil || len(lst2) != 2 {
		t.Fatalf("ListEntries limit: n=%d err=%v", len(lst2), err)
	}

	// Before filter should exclude the newest item
	if all, err := s.Entries().List(ctx, model.ListEntriesRequest{ActorID: actorID, VaultID: v.VaultID, MemoryID: m.MemoryID}); err == nil && len(all) >= 2 {
		bf := all[0].CreationTime
		if bef, err := s.Entries().List(ctx, model.ListEntriesRequest{ActorID: actorID, VaultID: v.VaultID, MemoryID: m.MemoryID, Before: &bf}); err != nil || len(bef) >= len(all) {
			t.Fatalf("before should reduce results: before=%d all=%d err=%v", len(bef), len(all), err)
		}
	}

	// After filter should include at least one when using older timestamp
	if all, err := s.Entries().List(ctx, model.ListEntriesRequest{ActorID: actorID, VaultID: v.VaultID, MemoryID: m.MemoryID}); err == nil && len(all) >= 2 {
		oldest := all[len(all)-1].CreationTime
		if aft, err := s.Entries().List(ctx, model.ListEntriesRequest{ActorID: actorID, VaultID: v.VaultID, MemoryID: m.MemoryID, After: &oldest}); err != nil || len(aft) == 0 {
			t.Fatalf("after should return at least one entry: n=%d err=%v", len(aft), err)
		}
	}

	// Delete memory and vault
	if err := s.Memories().Delete(ctx, actorID, v.VaultID, m.MemoryID); err != nil {
		t.Fatalf("DeleteMemory: %v", err)
	}
	if err := s.Vaults().Delete(ctx, actorID, v.VaultID); err != nil {
		t.Fatalf("DeleteVault: %v", err)
	}
}
