package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gorilla/mux"

	respond "github.com/mycelian/mycelian-memory/server/internal/api/respond"
	"github.com/mycelian/mycelian-memory/server/internal/auth"
	"github.com/mycelian/mycelian-memory/server/internal/config"
	"github.com/mycelian/mycelian-memory/server/internal/model"
	"github.com/mycelian/mycelian-memory/server/internal/services"
)

type MemoryHandler struct {
	svc        *services.MemoryService
	vaultSv    *services.VaultService
	authorizer auth.Authorizer
	cfg        *config.Config
}

func NewMemoryHandler(svc *services.MemoryService, vaultSvc *services.VaultService, authorizer auth.Authorizer, cfg *config.Config) *MemoryHandler {
	return &MemoryHandler{svc: svc, vaultSv: vaultSvc, authorizer: authorizer, cfg: cfg}
}

// CreateMemory POST /api/vaults/{vaultId}/memories
func (h *MemoryHandler) CreateMemory(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "memory.create", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	vars := mux.Vars(r)
	vaultID := vars["vaultId"]
	var req struct {
		MemoryType  string  `json:"memoryType"`
		Title       string  `json:"title"`
		Description *string `json:"description,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.WriteBadRequest(w, "Invalid JSON")
		return
	}
	m := &model.Memory{ActorID: actorInfo.ActorID, VaultID: vaultID, MemoryType: req.MemoryType, Title: req.Title, Description: req.Description}
	out, err := h.svc.CreateMemory(r.Context(), m)
	if err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	respond.WriteJSON(w, http.StatusCreated, out)
}

// ListMemories GET /api/vaults/{vaultId}/memories
func (h *MemoryHandler) ListMemories(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "memory.read", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	v := mux.Vars(r)
	out, err := h.svc.ListMemories(r.Context(), actorInfo.ActorID, v["vaultId"])
	if err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	if out == nil {
		out = []*model.Memory{}
	}
	respond.WriteJSON(w, http.StatusOK, map[string]interface{}{"memories": out, "count": len(out)})
}

// GetMemory GET /api/vaults/{vaultId}/memories/{memoryId}
func (h *MemoryHandler) GetMemory(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "memory.read", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	v := mux.Vars(r)
	out, err := h.svc.GetMemory(r.Context(), actorInfo.ActorID, v["vaultId"], v["memoryId"])
	if err != nil {
		respond.WriteNotFound(w, err.Error())
		return
	}
	respond.WriteJSON(w, http.StatusOK, out)
}

// ListMemoryEntries GET /api/vaults/{vaultId}/memories/{memoryId}/entries
func (h *MemoryHandler) ListMemoryEntries(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "memory.read", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	v := mux.Vars(r)
	vaultID := v["vaultId"]
	memoryID := v["memoryId"]

	// SECURITY: Validate vault exists and actor owns it
	if h.vaultSv != nil {
		_, err := h.vaultSv.GetVault(r.Context(), actorInfo.ActorID, vaultID)
		if err != nil {
			respond.WriteNotFound(w, "vault not found")
			return
		}
	}

	// SECURITY: Validate memory exists in the vault and actor owns it
	_, err = h.svc.GetMemory(r.Context(), actorInfo.ActorID, vaultID, memoryID)
	if err != nil {
		respond.WriteNotFound(w, "memory not found")
		return
	}

	q := r.URL.Query()
	req := model.ListEntriesRequest{ActorID: actorInfo.ActorID, VaultID: vaultID, MemoryID: memoryID}
	if s := q.Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			req.Limit = n
		}
	}
	if s := q.Get("before"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			req.Before = &t
		}
	}
	if s := q.Get("after"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			req.After = &t
		}
	}
	outs, err := h.svc.ListEntries(r.Context(), req)
	if err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	if outs == nil {
		outs = []*model.MemoryEntry{}
	}
	respond.WriteJSON(w, http.StatusOK, map[string]interface{}{"entries": outs, "count": len(outs)})
}

// CreateMemoryEntry POST /api/vaults/{vaultId}/memories/{memoryId}/entries
func (h *MemoryHandler) CreateMemoryEntry(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "memory.create", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	v := mux.Vars(r)
	vaultID := v["vaultId"]
	memoryID := v["memoryId"]

	// SECURITY: Validate vault exists and actor owns it
	if h.vaultSv != nil {
		_, err := h.vaultSv.GetVault(r.Context(), actorInfo.ActorID, vaultID)
		if err != nil {
			respond.WriteNotFound(w, "vault not found")
			return
		}
	}

	// SECURITY: Validate memory exists in the vault and actor owns it
	_, err = h.svc.GetMemory(r.Context(), actorInfo.ActorID, vaultID, memoryID)
	if err != nil {
		respond.WriteNotFound(w, "memory not found")
		return
	}

	var in struct {
		RawEntry       string                 `json:"rawEntry"`
		Summary        *string                `json:"summary,omitempty"`
		Metadata       map[string]interface{} `json:"metadata,omitempty"`
		Tags           map[string]interface{} `json:"tags,omitempty"`
		ExpirationTime *time.Time             `json:"expirationTime,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		respond.WriteBadRequest(w, "Invalid JSON")
		return
	}
	e := &model.MemoryEntry{
		ActorID: actorInfo.ActorID, VaultID: vaultID, MemoryID: memoryID,
		RawEntry: in.RawEntry, Summary: in.Summary, Metadata: in.Metadata, Tags: in.Tags, ExpirationTime: in.ExpirationTime,
	}
	out, err := h.svc.CreateEntry(r.Context(), e)
	if err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	respond.WriteJSON(w, http.StatusCreated, out)
}

// GetMemoryEntryByID GET /api/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}
func (h *MemoryHandler) GetMemoryEntryByID(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "memory.read", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	v := mux.Vars(r)
	out, err := h.svc.GetEntryByID(r.Context(), actorInfo.ActorID, v["vaultId"], v["memoryId"], v["entryId"])
	if err != nil {
		respond.WriteNotFound(w, err.Error())
		return
	}
	respond.WriteJSON(w, http.StatusOK, out)
}

// UpdateMemoryEntryTags PATCH /api/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}/tags
func (h *MemoryHandler) UpdateMemoryEntryTags(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "memory.write", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	v := mux.Vars(r)
	vaultID := v["vaultId"]
	memoryID := v["memoryId"]
	entryID := v["entryId"]

	// SECURITY: Validate vault exists and actor owns it
	if h.vaultSv != nil {
		_, err := h.vaultSv.GetVault(r.Context(), actorInfo.ActorID, vaultID)
		if err != nil {
			respond.WriteNotFound(w, "vault not found")
			return
		}
	}

	// SECURITY: Validate memory exists in the vault and actor owns it
	_, err = h.svc.GetMemory(r.Context(), actorInfo.ActorID, vaultID, memoryID)
	if err != nil {
		respond.WriteNotFound(w, "memory not found")
		return
	}

	var in struct {
		Tags map[string]interface{} `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		respond.WriteBadRequest(w, "Invalid JSON")
		return
	}
	out, err := h.svc.UpdateEntryTags(r.Context(), actorInfo.ActorID, vaultID, memoryID, entryID, in.Tags)
	if err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	respond.WriteJSON(w, http.StatusOK, out)
}

// PutMemoryContext PUT /api/vaults/{vaultId}/memories/{memoryId}/contexts
func (h *MemoryHandler) PutMemoryContext(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "memory.write", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	v := mux.Vars(r)
	vaultID := v["vaultId"]
	memoryID := v["memoryId"]

	// SECURITY: Validate vault exists and actor owns it
	if h.vaultSv != nil {
		_, err := h.vaultSv.GetVault(r.Context(), actorInfo.ActorID, vaultID)
		if err != nil {
			respond.WriteNotFound(w, "vault not found")
			return
		}
	}

	// SECURITY: Validate memory exists in the vault and actor owns it
	_, err = h.svc.GetMemory(r.Context(), actorInfo.ActorID, vaultID, memoryID)
	if err != nil {
		respond.WriteNotFound(w, "memory not found")
		return
	}

	if ct := r.Header.Get("Content-Type"); ct != "" && ct != "text/plain" && ct != "text/plain; charset=utf-8" {
		respond.WriteError(w, http.StatusUnsupportedMediaType, "Content-Type must be text/plain")
		return
	}
	doc, err := io.ReadAll(r.Body)
	if err != nil {
		respond.WriteBadRequest(w, "unable to read body")
		return
	}
	if len(doc) == 0 {
		respond.WriteBadRequest(w, "context document must not be empty")
		return
	}
	// UTF-8 and character-set validation
	if !utf8.Valid(doc) {
		respond.WriteBadRequest(w, "context must be valid UTF-8")
		return
	}
	s := string(doc)
	for _, r := range s {
		// Allow common whitespace
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		// Disallow other control characters
		if unicode.IsControl(r) {
			respond.WriteBadRequest(w, fmt.Sprintf("invalid control character: U+%04X", r))
			return
		}
		// Disallow Unicode noncharacters (U+FDD0..U+FDEF and U+..FFFE/FFFF in every plane)
		if (r&0xFFFE == 0xFFFE) || (r >= 0xFDD0 && r <= 0xFDEF) {
			respond.WriteBadRequest(w, fmt.Sprintf("invalid noncharacter: U+%04X", r))
			return
		}
	}
	if h.cfg != nil && h.cfg.MaxContextChars > 0 {
		if len(doc) > h.cfg.MaxContextChars {
			if utf8.RuneCount(doc) > h.cfg.MaxContextChars {
				respond.WriteError(w, http.StatusRequestEntityTooLarge, "context exceeds maximum size")
				return
			}
		}
	}

	mc := &model.MemoryContext{ActorID: actorInfo.ActorID, VaultID: vaultID, MemoryID: memoryID, Context: s}
	out, err := h.svc.PutContext(r.Context(), mc)
	if err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	respond.WriteJSON(w, http.StatusCreated, out)
}

// GetLatestMemoryContext GET /api/vaults/{vaultId}/memories/{memoryId}/contexts
func (h *MemoryHandler) GetLatestMemoryContext(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "memory.read", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	v := mux.Vars(r)
	vaultID := v["vaultId"]
	memoryID := v["memoryId"]

	// SECURITY: Validate vault exists and actor owns it
	if h.vaultSv != nil {
		_, err := h.vaultSv.GetVault(r.Context(), actorInfo.ActorID, vaultID)
		if err != nil {
			respond.WriteNotFound(w, "vault not found")
			return
		}
	}

	// SECURITY: Validate memory exists in the vault and actor owns it
	_, err = h.svc.GetMemory(r.Context(), actorInfo.ActorID, vaultID, memoryID)
	if err != nil {
		respond.WriteNotFound(w, "memory not found")
		return
	}

	out, err := h.svc.GetLatestContext(r.Context(), actorInfo.ActorID, vaultID, memoryID)
	if err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(out.Context))
}

// GetMemoryByTitle GET /api/vaults/{vaultTitle}/memories/{memoryTitle}
func (h *MemoryHandler) GetMemoryByTitle(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "memory.read", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	v := mux.Vars(r)
	if h.vaultSv == nil {
		respond.WriteInternalError(w, "vault service unavailable")
		return
	}
	vaultObj, err := h.vaultSv.GetVaultByTitle(r.Context(), actorInfo.ActorID, v["vaultTitle"])
	if err != nil {
		respond.WriteNotFound(w, err.Error())
		return
	}
	mem, err := h.svc.GetMemoryByTitle(r.Context(), actorInfo.ActorID, vaultObj.VaultID, v["memoryTitle"])
	if err != nil {
		respond.WriteNotFound(w, err.Error())
		return
	}
	respond.WriteJSON(w, http.StatusOK, mem)
}

// ListMemoriesByVaultTitle GET /api/vaults/{vaultTitle}/memories
func (h *MemoryHandler) ListMemoriesByVaultTitle(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "memory.read", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	v := mux.Vars(r)
	if h.vaultSv == nil {
		respond.WriteInternalError(w, "vault service unavailable")
		return
	}
	vaultObj, err := h.vaultSv.GetVaultByTitle(r.Context(), actorInfo.ActorID, v["vaultTitle"])
	if err != nil {
		respond.WriteNotFound(w, err.Error())
		return
	}
	mems, err := h.svc.ListMemories(r.Context(), actorInfo.ActorID, vaultObj.VaultID)
	if err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	if mems == nil {
		mems = []*model.Memory{}
	}
	respond.WriteJSON(w, http.StatusOK, map[string]interface{}{"memories": mems, "count": len(mems)})
}

// DeleteMemory DELETE /api/vaults/{vaultId}/memories/{memoryId}
func (h *MemoryHandler) DeleteMemory(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "memory.delete", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	v := mux.Vars(r)
	if err := h.svc.DeleteMemory(r.Context(), actorInfo.ActorID, v["vaultId"], v["memoryId"]); err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteMemoryEntryByID DELETE /api/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}
func (h *MemoryHandler) DeleteMemoryEntryByID(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "memory.delete", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	v := mux.Vars(r)
	if err := h.svc.DeleteEntry(r.Context(), actorInfo.ActorID, v["vaultId"], v["memoryId"], v["entryId"]); err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteMemoryContextByID DELETE /api/vaults/{vaultId}/memories/{memoryId}/contexts/{contextId}
func (h *MemoryHandler) DeleteMemoryContextByID(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "memory.delete", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	v := mux.Vars(r)
	if err := h.svc.DeleteContext(r.Context(), actorInfo.ActorID, v["vaultId"], v["memoryId"], v["contextId"]); err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
