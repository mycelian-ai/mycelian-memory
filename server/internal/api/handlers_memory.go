package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	respond "github.com/mycelian/mycelian-memory/server/internal/api/respond"
	"github.com/mycelian/mycelian-memory/server/internal/auth"
	"github.com/mycelian/mycelian-memory/server/internal/model"
	"github.com/mycelian/mycelian-memory/server/internal/services"
)

type MemoryHandler struct {
	svc        *services.MemoryService
	vaultSv    *services.VaultService
	authorizer auth.Authorizer
}

func NewMemoryHandler(svc *services.MemoryService, vaultSvc *services.VaultService, authorizer auth.Authorizer) *MemoryHandler {
	return &MemoryHandler{svc: svc, vaultSv: vaultSvc, authorizer: authorizer}
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
	q := r.URL.Query()
	req := model.ListEntriesRequest{ActorID: actorInfo.ActorID, VaultID: v["vaultId"], MemoryID: v["memoryId"]}
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
		ActorID: actorInfo.ActorID, VaultID: v["vaultId"], MemoryID: v["memoryId"],
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
	var in struct {
		Tags map[string]interface{} `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		respond.WriteBadRequest(w, "Invalid JSON")
		return
	}
	out, err := h.svc.UpdateEntryTags(r.Context(), actorInfo.ActorID, v["vaultId"], v["memoryId"], v["entryId"], in.Tags)
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
	var body struct {
		Context map[string]interface{} `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.WriteBadRequest(w, "Invalid JSON")
		return
	}
	raw, err := json.Marshal(body.Context)
	if err != nil {
		respond.WriteBadRequest(w, "context must be a valid JSON object")
		return
	}
	mc := &model.MemoryContext{ActorID: actorInfo.ActorID, VaultID: v["vaultId"], MemoryID: v["memoryId"], ContextJSON: raw}
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
	out, err := h.svc.GetLatestContext(r.Context(), actorInfo.ActorID, v["vaultId"], v["memoryId"])
	if err != nil {
		respond.WriteInternalError(w, err.Error())
		return
	}
	// Decode stored JSON context into a native JSON value so clients receive an object rather than base64.
	var ctxVal interface{}
	if err := json.Unmarshal(out.ContextJSON, &ctxVal); err != nil {
		// Fallback to raw string if not valid JSON
		ctxVal = string(out.ContextJSON)
	}
	resp := map[string]interface{}{
		"contextId":    out.ContextID,
		"actorId":      out.ActorID,
		"vaultId":      out.VaultID,
		"memoryId":     out.MemoryID,
		"creationTime": out.CreationTime,
		"context":      ctxVal,
	}
	respond.WriteJSON(w, http.StatusOK, resp)
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
