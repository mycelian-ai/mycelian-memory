package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	testing "testing"

	"github.com/gorilla/mux"

	"github.com/mycelian/mycelian-memory/server/internal/auth"
	"github.com/mycelian/mycelian-memory/server/internal/model"
	"github.com/mycelian/mycelian-memory/server/internal/services"
	"github.com/mycelian/mycelian-memory/server/internal/storage"
)

type allowAllVaultAuthorizer struct{}

func (allowAllVaultAuthorizer) Authorize(ctx context.Context, apiKey, operation, resource string) (*auth.ActorInfo, error) {
	return &auth.ActorInfo{ActorID: "u1"}, nil
}

type nfVaults struct{}

func (nfVaults) Create(context.Context, *model.Vault) (*model.Vault, error)       { return nil, nil }
func (nfVaults) GetByID(context.Context, string, string) (*model.Vault, error)   { return nil, storage.ErrNotFound }
func (nfVaults) GetByTitle(context.Context, string, string) (*model.Vault, error) { return nil, storage.ErrNotFound }
func (nfVaults) List(context.Context, string) ([]*model.Vault, error)            { return nil, nil }
func (nfVaults) Delete(context.Context, string, string) error                     { return nil }
func (nfVaults) AddMemory(context.Context, string, string, string) error          { return nil }

type fakeStoreVault struct{ v nfVaults }

func (f *fakeStoreVault) Vaults() storage.Vaults     { return f.v }
func (f *fakeStoreVault) Memories() storage.Memories { return nil }
func (f *fakeStoreVault) Entries() storage.Entries   { return nil }
func (f *fakeStoreVault) Contexts() storage.Contexts { return nil }

type errResp struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func TestGetVault_NotFoundMapsTo404(t *testing.T) {
	svc := services.NewVaultService(&fakeStoreVault{}, nil)
	h := NewVaultHandler(svc, allowAllVaultAuthorizer{})

	req := httptest.NewRequest(http.MethodGet, "/v0/vaults/v-1", nil)
	req.Header.Set("Authorization", "Bearer dev-key")
	req = mux.SetURLVars(req, map[string]string{"vaultId": "v-1"})
	rr := httptest.NewRecorder()

	h.GetVault(rr, req)
	if rr.Code \!= http.StatusNotFound {
		t.Fatalf("status: got %d, want %d; body=%s", rr.Code, http.StatusNotFound, rr.Body.String())
	}
	var er errResp
	_ = json.Unmarshal(rr.Body.Bytes(), &er)
	if er.Message \!= "vault not found" {
		t.Fatalf("message: got %q, want %q", er.Message, "vault not found")
	}
}