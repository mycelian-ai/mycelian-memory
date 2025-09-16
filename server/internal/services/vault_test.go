package services

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/model"
	"github.com/mycelian/mycelian-memory/server/internal/store"
)

// --- Fakes ---

type fakeIndex struct {
	deletedEntries  []string
	deletedContexts []string
	deleteVaultArgs []struct{ userID, vaultID string }
}

func (f *fakeIndex) Search(ctx context.Context, userID, memoryID, query string, vec []float32, topKE int, alpha float32, includeRawEntries bool) ([]model.SearchHit, error) {
	return nil, nil
}
func (f *fakeIndex) LatestContext(ctx context.Context, userID, memoryID string) (string, time.Time, error) {
	return "", time.Time{}, nil
}

func (f *fakeIndex) SearchContexts(ctx context.Context, userID, memoryID, query string, vec []float32, topKC int, alpha float32) ([]model.ContextHit, error) {
	return []model.ContextHit{}, nil
}
func (f *fakeIndex) DeleteEntry(ctx context.Context, userID, entryID string) error {
	f.deletedEntries = append(f.deletedEntries, entryID)
	return nil
}
func (f *fakeIndex) DeleteContext(ctx context.Context, userID, contextID string) error {
	f.deletedContexts = append(f.deletedContexts, contextID)
	return nil
}
func (f *fakeIndex) DeleteMemory(ctx context.Context, userID, memoryID string) error { return nil }
func (f *fakeIndex) DeleteVault(ctx context.Context, userID, vaultID string) error {
	f.deleteVaultArgs = append(f.deleteVaultArgs, struct{ userID, vaultID string }{userID, vaultID})
	return nil
}
func (f *fakeIndex) UpsertEntry(ctx context.Context, entryID string, vec []float32, payload map[string]interface{}) error {
	return nil
}
func (f *fakeIndex) UpsertContext(ctx context.Context, contextID string, vec []float32, payload map[string]interface{}) error {
	return nil
}

type fakeStore struct {
	mems         []*model.Memory
	entriesByMem map[string][]*model.MemoryEntry
	ctxByMem     map[string]*model.MemoryContext
	vaultDeleted struct {
		userID, vaultID string
		called          bool
	}
}

func (f *fakeStore) Users() store.Users       { return fakeUsers{} }
func (f *fakeStore) Vaults() store.Vaults     { return &fakeVaults{f} }
func (f *fakeStore) Memories() store.Memories { return &fakeMemories{f} }
func (f *fakeStore) Entries() store.Entries   { return &fakeEntries{f} }
func (f *fakeStore) Contexts() store.Contexts { return &fakeContexts{f} }

type fakeUsers struct{}

func (fakeUsers) Create(context.Context, *model.User) (*model.User, error) { panic("unused") }
func (fakeUsers) Get(context.Context, string) (*model.User, error)         { panic("unused") }
func (fakeUsers) Delete(context.Context, string) error                     { panic("unused") }

type fakeVaults struct{ p *fakeStore }

func (v *fakeVaults) Create(context.Context, *model.Vault) (*model.Vault, error)    { panic("unused") }
func (v *fakeVaults) GetByID(context.Context, string, string) (*model.Vault, error) { panic("unused") }
func (v *fakeVaults) GetByTitle(context.Context, string, string) (*model.Vault, error) {
	panic("unused")
}
func (v *fakeVaults) List(context.Context, string) ([]*model.Vault, error) { panic("unused") }
func (v *fakeVaults) Delete(_ context.Context, userID, vaultID string) error {
	v.p.vaultDeleted.userID = userID
	v.p.vaultDeleted.vaultID = vaultID
	v.p.vaultDeleted.called = true
	return nil
}
func (v *fakeVaults) AddMemory(context.Context, string, string, string) error { panic("unused") }

type fakeMemories struct{ p *fakeStore }

func (m *fakeMemories) Create(context.Context, *model.Memory) (*model.Memory, error) { panic("unused") }
func (m *fakeMemories) GetByID(context.Context, string, string, string) (*model.Memory, error) {
	panic("unused")
}
func (m *fakeMemories) GetByTitle(context.Context, string, string, string) (*model.Memory, error) {
	panic("unused")
}
func (m *fakeMemories) List(context.Context, string, string) ([]*model.Memory, error) {
	return m.p.mems, nil
}
func (m *fakeMemories) Delete(context.Context, string, string, string) error { panic("unused") }

type fakeEntries struct{ p *fakeStore }

func (e *fakeEntries) Create(context.Context, *model.MemoryEntry) (*model.MemoryEntry, error) {
	panic("unused")
}
func (e *fakeEntries) List(_ context.Context, req model.ListEntriesRequest) ([]*model.MemoryEntry, error) {
	return e.p.entriesByMem[req.MemoryID], nil
}
func (e *fakeEntries) GetByID(context.Context, string, string, string, string) (*model.MemoryEntry, error) {
	panic("unused")
}
func (e *fakeEntries) UpdateTags(context.Context, string, string, string, string, map[string]interface{}) (*model.MemoryEntry, error) {
	panic("unused")
}
func (e *fakeEntries) DeleteByID(context.Context, string, string, string, string) error {
	panic("unused")
}

type fakeContexts struct{ p *fakeStore }

func (c *fakeContexts) Put(context.Context, *model.MemoryContext) (*model.MemoryContext, error) {
	panic("unused")
}
func (c *fakeContexts) Latest(_ context.Context, _ string, _ string, memoryID string) (*model.MemoryContext, error) {
	if mc, ok := c.p.ctxByMem[memoryID]; ok {
		return mc, nil
	}
	return nil, model.ErrNotFound
}
func (c *fakeContexts) DeleteByID(context.Context, string, string, string, string) error {
	panic("unused")
}

// --- Test ---

func TestVaultDeletePropagatesToIndex(t *testing.T) {
	idx := &fakeIndex{}
	fs := &fakeStore{
		mems: []*model.Memory{
			{ActorID: "u1", VaultID: "v1", MemoryID: "m1"},
			{ActorID: "u1", VaultID: "v1", MemoryID: "m2"},
		},
		entriesByMem: map[string][]*model.MemoryEntry{
			"m1": {&model.MemoryEntry{ActorID: "u1", VaultID: "v1", MemoryID: "m1", EntryID: "e1"}, &model.MemoryEntry{ActorID: "u1", VaultID: "v1", MemoryID: "m1", EntryID: "e2"}},
			"m2": {&model.MemoryEntry{ActorID: "u1", VaultID: "v1", MemoryID: "m2", EntryID: "e3"}},
		},
		ctxByMem: map[string]*model.MemoryContext{
			"m1": {ActorID: "u1", VaultID: "v1", MemoryID: "m1", ContextID: "c1"},
			// m2 has no context
		},
	}

	svc := NewVaultService(fs, idx)
	if err := svc.DeleteVault(context.Background(), "u1", "v1"); err != nil {
		t.Fatalf("DeleteVault error: %v", err)
	}

	// Verify index deletions occurred for all entries
	wantEntries := []string{"e1", "e2", "e3"}
	sort.Strings(wantEntries)
	gotEntries := append([]string(nil), idx.deletedEntries...)
	sort.Strings(gotEntries)
	if !reflect.DeepEqual(wantEntries, gotEntries) {
		t.Fatalf("deleted entries mismatch: want %v, got %v", wantEntries, gotEntries)
	}

	// Verify context deletion for m1
	if len(idx.deletedContexts) != 1 || idx.deletedContexts[0] != "c1" {
		t.Fatalf("deleted contexts mismatch: want [c1], got %v", idx.deletedContexts)
	}

	// Verify coarse vault delete called once
	if len(idx.deleteVaultArgs) != 1 || idx.deleteVaultArgs[0].userID != "u1" || idx.deleteVaultArgs[0].vaultID != "v1" {
		t.Fatalf("delete vault not called correctly: got %+v", idx.deleteVaultArgs)
	}

	// Verify storage delete invoked
	if !fs.vaultDeleted.called || fs.vaultDeleted.userID != "u1" || fs.vaultDeleted.vaultID != "v1" {
		t.Fatalf("storage vault delete not invoked correctly: %+v", fs.vaultDeleted)
	}
}
