package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	memcore "memory-backend/internal/core/memory"
	"memory-backend/internal/core/vault"
	platformHttp "memory-backend/internal/platform/http"
)

// VaultHandler provides HTTP transport for Vault operations.
type VaultHandler struct {
	vaultService *vault.Service
}

func NewVaultHandler(svc *vault.Service) *VaultHandler {
	return &VaultHandler{vaultService: svc}
}

// CreateVault POST /api/users/{userId}/vaults
func (h *VaultHandler) CreateVault(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	var req struct {
		Title       string  `json:"title"`
		Description *string `json:"description,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platformHttp.WriteBadRequest(w, "Invalid JSON")
		return
	}
	vreq := vault.CreateVaultRequest{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
	}
	v, err := h.vaultService.CreateVault(r.Context(), vreq)
	if err != nil {
		// Map domain errors to HTTP status codes for better client feedback
		if memcore.IsValidationError(err) {
			platformHttp.WriteBadRequest(w, err.Error())
			return
		}
		if memcore.IsConflictError(err) {
			platformHttp.WriteError(w, http.StatusConflict, err.Error())
			return
		}
		if memcore.IsNotFoundError(err) {
			platformHttp.WriteNotFound(w, err.Error())
			return
		}
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusCreated, v)
}

// ListVaults GET /api/users/{userId}/vaults
func (h *VaultHandler) ListVaults(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	vts, err := h.vaultService.ListVaults(r.Context(), userID)
	if err != nil {
		if memcore.IsValidationError(err) {
			platformHttp.WriteBadRequest(w, err.Error())
			return
		}
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusOK, map[string]interface{}{"vaults": vts, "count": len(vts)})
}

// GetVault GET /api/users/{userId}/vaults/{vaultId}
func (h *VaultHandler) GetVault(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultIDStr := vars["vaultId"]
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		platformHttp.WriteBadRequest(w, "invalid vaultId")
		return
	}
	v, err := h.vaultService.GetVault(r.Context(), userID, vaultID)
	if err != nil {
		platformHttp.WriteNotFound(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusOK, v)
}

// DeleteVault DELETE /api/users/{userId}/vaults/{vaultId}
func (h *VaultHandler) DeleteVault(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultIDStr := vars["vaultId"]
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		platformHttp.WriteBadRequest(w, "invalid vaultId")
		return
	}
	if err := h.vaultService.DeleteVault(r.Context(), userID, vaultID); err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
