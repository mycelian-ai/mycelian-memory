package services

import (
	"context"

	"github.com/mycelian/mycelian-memory/server/internal/model"
	"github.com/mycelian/mycelian-memory/server/internal/searchindex"
	"github.com/mycelian/mycelian-memory/server/internal/store"
)

type VaultService struct {
	store store.Store
	idx   searchindex.Index
}

func NewVaultService(s store.Store, idx searchindex.Index) *VaultService {
	return &VaultService{store: s, idx: idx}
}

func (s *VaultService) CreateVault(ctx context.Context, v *model.Vault) (*model.Vault, error) {
	return s.store.Vaults().Create(ctx, v)
}
func (s *VaultService) GetVault(ctx context.Context, userID, vaultID string) (*model.Vault, error) {
	return s.store.Vaults().GetByID(ctx, userID, vaultID)
}
func (s *VaultService) GetVaultByTitle(ctx context.Context, userID, title string) (*model.Vault, error) {
	return s.store.Vaults().GetByTitle(ctx, userID, title)
}
func (s *VaultService) ListVaults(ctx context.Context, userID string) ([]*model.Vault, error) {
	return s.store.Vaults().List(ctx, userID)
}
func (s *VaultService) DeleteVault(ctx context.Context, userID, vaultID string) error {
	// Enumerate affected objects first so we can update the index even if
	// storage delete succeeds and data becomes unavailable for listing.
	memories, err := s.store.Memories().List(ctx, userID, vaultID)
	if err != nil {
		return err
	}

	type deletions struct {
		entryIDs   []string
		contextIDs []string
	}
	dels := deletions{}

	for _, m := range memories {
		// List all entries under this memory
		entries, err := s.store.Entries().List(ctx, model.ListEntriesRequest{UserID: userID, VaultID: vaultID, MemoryID: m.MemoryID, Limit: 0})
		if err != nil {
			return err
		}
		for _, e := range entries {
			dels.entryIDs = append(dels.entryIDs, e.EntryID)
		}

		// Get latest context (if any) for this memory
		if ctxRec, err := s.store.Contexts().Latest(ctx, userID, vaultID, m.MemoryID); err == nil && ctxRec != nil {
			if ctxRec.ContextID != "" {
				dels.contextIDs = append(dels.contextIDs, ctxRec.ContextID)
			}
		}
	}

	// First update the index (best-effort synchronous propagation). If any
	// index call fails, abort before mutating storage to avoid stale index.
	if s.idx != nil {
		for _, id := range dels.entryIDs {
			if err := s.idx.DeleteEntry(ctx, userID, id); err != nil {
				return err
			}
		}
		for _, id := range dels.contextIDs {
			if err := s.idx.DeleteContext(ctx, userID, id); err != nil {
				return err
			}
		}
		// Try a coarse-grained vault delete if supported by the adapter.
		if err := s.idx.DeleteVault(ctx, userID, vaultID); err != nil {
			return err
		}
	}

	// Finally, delete from storage (cascades to memories/entries/contexts).
	return s.store.Vaults().Delete(ctx, userID, vaultID)
}
func (s *VaultService) AddMemoryToVault(ctx context.Context, userID, vaultID, memoryID string) error {
	return s.store.Vaults().AddMemory(ctx, userID, vaultID, memoryID)
}
