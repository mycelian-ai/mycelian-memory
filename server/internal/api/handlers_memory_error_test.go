package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"github.com/mycelian/mycelian-memory/server/internal/auth"
	"github.com/mycelian/mycelian-memory/server/internal/model"
	"github.com/mycelian/mycelian-memory/server/internal/services"
	"github.com/mycelian/mycelian-memory/server/internal/storage"
)

// --- fakes for MemoryHandler error mapping tests ---

type allowAllAuthorizer struct{}

func (allowAllAuthorizer) Authorize(ctx context.Context, apiKey, operation, resource string) (*auth.ActorInfo, error) {
	return &auth.ActorInfo{ActorID: "u1"}, nil
}

type nfMemories struct{}

func (nfMemories) Create(context.Context, *model.Memory) (*model.Memory, error) { return nil, nil }
func (nfMemories) GetByID(context.Context, string, string, string) (*model.Memory, error) {
	return nil, storage.ErrNotFound
}
func (nfMemories) GetByTitle(context.Context, string, string, string) (*model.Memory, error) {
	return nil, storage.ErrNotFound
}
func (nfMemories) List(context.Context, string, string) ([]*model.Memory, error) { return nil, nil }
func (nfMemories) Delete(context.Context, string, string, string) error          { return nil }

type nfEntries struct{}

func (nfEntries) Create(context.Context, *model.MemoryEntry) (*model.MemoryEntry, error) {
	return nil, nil
}
func (nfEntries) List(context.Context, model.ListEntriesRequest) ([]*model.MemoryEntry, error) {
	return nil, nil
}
func (nfEntries) GetByID(context.Context, string, string, string, string) (*model.MemoryEntry, error) {
	return nil, storage.ErrNotFound
}
func (nfEntries) UpdateTags(context.Context, string, string, string, string, map[string]interface{}) (*model.MemoryEntry, error) {
	return nil, nil
}
func (nfEntries) DeleteByID(context.Context, string, string, string, string) error { return nil }

type fakeStoreMem struct {
	mem nfMemories
	ent nfEntries
}

func (f *fakeStoreMem) Vaults() storage.Vaults     { return nil }
func (f *fakeStoreMem) Memories() storage.Memories { return f.mem }
func (f *fakeStoreMem) Entries() storage.Entries   { return f.ent }
func (f *fakeStoreMem) Contexts() storage.Contexts { return nil }

type errResp struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func TestGetMemory_NotFoundMapsTo404(t *testing.T) {
	svc := services.NewMemoryService(&fakeStoreMem{}, nil, nil)
	h := NewMemoryHandler(svc, nil, allowAllAuthorizer{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/v0/vaults/v1/memories/m-1", nil)
	req.Header.Set("Authorization", "Bearer dev-key")
	req = mux.SetURLVars(req, map[string]string{"vaultId": "v1", "memoryId": "m-1"})
	rr := httptest.NewRecorder()

	h.GetMemory(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d; body=%s", rr.Code, http.StatusNotFound, rr.Body.String())
	}
	var er errResp
	_ = json.Unmarshal(rr.Body.Bytes(), &er)
	if er.Message != "memory not found" {
		t.Fatalf("message: got %q, want %q", er.Message, "memory not found")
	}
}

func TestGetMemoryEntryByID_NotFoundMapsTo404(t *testing.T) {
	svc := services.NewMemoryService(&fakeStoreMem{}, nil, nil)
	h := NewMemoryHandler(svc, nil, allowAllAuthorizer{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/v0/vaults/v1/memories/m-1/entries/e-9", nil)
	req.Header.Set("Authorization", "Bearer dev-key")
	req = mux.SetURLVars(req, map[string]string{"vaultId": "v1", "memoryId": "m-1", "entryId": "e-9"})
	rr := httptest.NewRecorder()

	h.GetMemoryEntryByID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d; body=%s", rr.Code, http.StatusNotFound, rr.Body.String())
	}
	var er errResp
	_ = json.Unmarshal(rr.Body.Bytes(), &er)
	if er.Message != "entry not found" {
		t.Fatalf("message: got %q, want %q", er.Message, "entry not found")
	}
}
