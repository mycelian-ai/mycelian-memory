package http

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/mycelian/mycelian-memory/server/internal/model"
	platformHttp "github.com/mycelian/mycelian-memory/server/internal/platform/http"
	"github.com/mycelian/mycelian-memory/server/internal/services"
)

// VaultV2Handler is a thin HTTP transport using the new VaultService.
type VaultV2Handler struct {
	svc *services.VaultService
}

func NewVaultV2Handler(svc *services.VaultService) *VaultV2Handler { return &VaultV2Handler{svc: svc} }

// CreateVault POST /api/users/{userId}/vaults
func (h *VaultV2Handler) CreateVault(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platformHttp.WriteBadRequest(w, "Invalid JSON")
		return
	}
	v := &model.Vault{UserID: userID, Title: req.Title}
	out, err := h.svc.CreateVault(r.Context(), v)
	if err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusCreated, out)
}

// ListVaults GET /api/users/{userId}/vaults
func (h *VaultV2Handler) ListVaults(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	vts, err := h.svc.ListVaults(r.Context(), userID)
	if err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusOK, map[string]interface{}{"vaults": vts, "count": len(vts)})
}

// GetVault GET /api/users/{userId}/vaults/{vaultId}
func (h *VaultV2Handler) GetVault(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	v, err := h.svc.GetVault(r.Context(), vars["userId"], vars["vaultId"])
	if err != nil {
		platformHttp.WriteNotFound(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusOK, v)
}

// DeleteVault DELETE /api/users/{userId}/vaults/{vaultId}
func (h *VaultV2Handler) DeleteVault(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := h.svc.DeleteVault(r.Context(), vars["userId"], vars["vaultId"]); err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AttachMemoryToVault POST /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/attach
func (h *VaultV2Handler) AttachMemoryToVault(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := h.svc.AddMemoryToVault(r.Context(), vars["userId"], vars["vaultId"], vars["memoryId"]); err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
