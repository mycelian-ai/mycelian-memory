package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	respond "github.com/mycelian/mycelian-memory/server/internal/api/respond"
	"github.com/mycelian/mycelian-memory/server/internal/model"
	"github.com/mycelian/mycelian-memory/server/internal/services"
)

// VaultHandler is a thin HTTP transport using the new VaultService.
type VaultHandler struct {
	svc *services.VaultService
}

func NewVaultHandler(svc *services.VaultService) *VaultHandler { return &VaultHandler{svc: svc} }

// CreateVault POST /api/users/{userId}/vaults
func (h *VaultHandler) CreateVault(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.WriteBadRequest(w, "Invalid JSON")
		return
	}
	v := &model.Vault{UserID: userID, Title: req.Title}
	out, err := h.svc.CreateVault(r.Context(), v)
	if err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	respond.WriteJSON(w, http.StatusCreated, out)
}

// ListVaults GET /api/users/{userId}/vaults
func (h *VaultHandler) ListVaults(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	vts, err := h.svc.ListVaults(r.Context(), userID)
	if err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	respond.WriteJSON(w, http.StatusOK, map[string]interface{}{"vaults": vts, "count": len(vts)})
}

// GetVault GET /api/users/{userId}/vaults/{vaultId}
func (h *VaultHandler) GetVault(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	v, err := h.svc.GetVault(r.Context(), vars["userId"], vars["vaultId"])
	if err != nil {
		respond.WriteNotFound(w, err.Error())
		return
	}
	respond.WriteJSON(w, http.StatusOK, v)
}

// DeleteVault DELETE /api/users/{userId}/vaults/{vaultId}
func (h *VaultHandler) DeleteVault(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := h.svc.DeleteVault(r.Context(), vars["userId"], vars["vaultId"]); err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AttachMemoryToVault POST /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/attach
func (h *VaultHandler) AttachMemoryToVault(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := h.svc.AddMemoryToVault(r.Context(), vars["userId"], vars["vaultId"], vars["memoryId"]); err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
