package services

import (
	"context"

	"github.com/mycelian/mycelian-memory/server/internal/model"
	"github.com/mycelian/mycelian-memory/server/internal/searchindex"
	"github.com/mycelian/mycelian-memory/server/internal/store"
)

// MemoryService orchestrates memory-related use cases.
type MemoryService struct {
	store store.Store
	idx   searchindex.Index
	emb   searchindex.Embeddings
}

func NewMemoryService(s store.Store, idx searchindex.Index, emb searchindex.Embeddings) *MemoryService {
	return &MemoryService{store: s, idx: idx, emb: emb}
}

func (s *MemoryService) DeleteMemory(ctx context.Context, userID, vaultID, memoryID string) error {
	if err := s.store.Memories().Delete(ctx, userID, vaultID, memoryID); err != nil {
		return err
	}
	// Synchronous propagation to the index (hard-delete policy).
	if s.idx != nil {
		return s.idx.DeleteMemory(ctx, userID, memoryID)
	}
	return nil
}

func (s *MemoryService) DeleteEntry(ctx context.Context, userID, vaultID, memoryID, entryID string) error {
	if err := s.store.Entries().DeleteByID(ctx, userID, vaultID, memoryID, entryID); err != nil {
		return err
	}
	if s.idx != nil {
		return s.idx.DeleteEntry(ctx, userID, entryID)
	}
	return nil
}

func (s *MemoryService) DeleteContext(ctx context.Context, userID, vaultID, memoryID, contextID string) error {
	if err := s.store.Contexts().DeleteByID(ctx, userID, vaultID, memoryID, contextID); err != nil {
		return err
	}
	if s.idx != nil {
		return s.idx.DeleteContext(ctx, userID, contextID)
	}
	return nil
}

func (s *MemoryService) CreateEntry(ctx context.Context, e *model.MemoryEntry) (*model.MemoryEntry, error) {
	// For now, delegate to store; indexing is handled out of band for create.
	return s.store.Entries().Create(ctx, e)
}

func (s *MemoryService) ListEntries(ctx context.Context, req model.ListEntriesRequest) ([]*model.MemoryEntry, error) {
	return s.store.Entries().List(ctx, req)
}

func (s *MemoryService) GetEntryByID(ctx context.Context, userID, vaultID, memoryID, entryID string) (*model.MemoryEntry, error) {
	return s.store.Entries().GetByID(ctx, userID, vaultID, memoryID, entryID)
}

func (s *MemoryService) UpdateEntryTags(ctx context.Context, userID, vaultID, memoryID, entryID string, tags map[string]interface{}) (*model.MemoryEntry, error) {
	return s.store.Entries().UpdateTags(ctx, userID, vaultID, memoryID, entryID, tags)
}

func (s *MemoryService) PutContext(ctx context.Context, c *model.MemoryContext) (*model.MemoryContext, error) {
	return s.store.Contexts().Put(ctx, c)
}

func (s *MemoryService) GetLatestContext(ctx context.Context, userID, vaultID, memoryID string) (*model.MemoryContext, error) {
	return s.store.Contexts().Latest(ctx, userID, vaultID, memoryID)
}

// Memory CRUD (container)
func (s *MemoryService) CreateMemory(ctx context.Context, m *model.Memory) (*model.Memory, error) {
	return s.store.Memories().Create(ctx, m)
}

func (s *MemoryService) GetMemory(ctx context.Context, userID, vaultID, memoryID string) (*model.Memory, error) {
	return s.store.Memories().GetByID(ctx, userID, vaultID, memoryID)
}

func (s *MemoryService) ListMemories(ctx context.Context, userID, vaultID string) ([]*model.Memory, error) {
	return s.store.Memories().List(ctx, userID, vaultID)
}

func (s *MemoryService) GetMemoryByTitle(ctx context.Context, userID, vaultID, title string) (*model.Memory, error) {
	return s.store.Memories().GetByTitle(ctx, userID, vaultID, title)
}
