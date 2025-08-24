package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	respond "github.com/mycelian/mycelian-memory/server/internal/api/respond"
	"github.com/mycelian/mycelian-memory/server/internal/auth"
	"github.com/mycelian/mycelian-memory/server/internal/model"
	"github.com/mycelian/mycelian-memory/server/internal/services"
)

// VaultHandler is a thin HTTP transport using the new VaultService.
type VaultHandler struct {
	svc        *services.VaultService
	authorizer auth.Authorizer
}

func NewVaultHandler(svc *services.VaultService, authorizer auth.Authorizer) *VaultHandler {
	return &VaultHandler{svc: svc, authorizer: authorizer}
}

// CreateVault POST /api/vaults
func (h *VaultHandler) CreateVault(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "vault.create", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.WriteBadRequest(w, "Invalid JSON")
		return
	}
	v := &model.Vault{ActorID: actorInfo.ActorID, Title: req.Title}
	out, err := h.svc.CreateVault(r.Context(), v)
	if err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	respond.WriteJSON(w, http.StatusCreated, out)
}

// ListVaults GET /api/vaults
func (h *VaultHandler) ListVaults(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "vault.read", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	vts, err := h.svc.ListVaults(r.Context(), actorInfo.ActorID)
	if err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	respond.WriteJSON(w, http.StatusOK, map[string]interface{}{"vaults": vts, "count": len(vts)})
}

// GetVault GET /api/vaults/{vaultId}
func (h *VaultHandler) GetVault(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "vault.read", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	vars := mux.Vars(r)
	v, err := h.svc.GetVault(r.Context(), actorInfo.ActorID, vars["vaultId"])
	if err != nil {
		respond.WriteNotFound(w, err.Error())
		return
	}
	respond.WriteJSON(w, http.StatusOK, v)
}

// DeleteVault DELETE /api/vaults/{vaultId}
func (h *VaultHandler) DeleteVault(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "vault.delete", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	vars := mux.Vars(r)
	if err := h.svc.DeleteVault(r.Context(), actorInfo.ActorID, vars["vaultId"]); err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AttachMemoryToVault POST /api/vaults/{vaultId}/memories/{memoryId}/attach
func (h *VaultHandler) AttachMemoryToVault(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "vault.write", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	vars := mux.Vars(r)
	if err := h.svc.AddMemoryToVault(r.Context(), actorInfo.ActorID, vars["vaultId"], vars["memoryId"]); err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
