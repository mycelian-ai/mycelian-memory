package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/mycelian/mycelian-memory/server/internal/model"
	platformHttp "github.com/mycelian/mycelian-memory/server/internal/platform/http"
	"github.com/mycelian/mycelian-memory/server/internal/services"
)

type MemoryV2Handler struct {
	svc     *services.MemoryService
	vaultSv *services.VaultService
}

func NewMemoryV2Handler(svc *services.MemoryService, vaultSvc *services.VaultService) *MemoryV2Handler {
	return &MemoryV2Handler{svc: svc, vaultSv: vaultSvc}
}

// CreateMemory POST /api/users/{userId}/vaults/{vaultId}/memories
func (h *MemoryV2Handler) CreateMemory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultID := vars["vaultId"]
	var req struct {
		MemoryType  string  `json:"memoryType"`
		Title       string  `json:"title"`
		Description *string `json:"description,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platformHttp.WriteBadRequest(w, "Invalid JSON")
		return
	}
	m := &model.Memory{UserID: userID, VaultID: vaultID, MemoryType: req.MemoryType, Title: req.Title, Description: req.Description}
	out, err := h.svc.CreateMemory(r.Context(), m)
	if err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusCreated, out)
}

// ListMemories GET /api/users/{userId}/vaults/{vaultId}/memories
func (h *MemoryV2Handler) ListMemories(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	out, err := h.svc.ListMemories(r.Context(), v["userId"], v["vaultId"])
	if err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	if out == nil {
		out = []*model.Memory{}
	}
	platformHttp.WriteJSON(w, http.StatusOK, map[string]interface{}{"memories": out, "count": len(out)})
}

// GetMemory GET /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}
func (h *MemoryV2Handler) GetMemory(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	out, err := h.svc.GetMemory(r.Context(), v["userId"], v["vaultId"], v["memoryId"])
	if err != nil {
		platformHttp.WriteNotFound(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusOK, out)
}

// ListMemoryEntries GET /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries
func (h *MemoryV2Handler) ListMemoryEntries(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	q := r.URL.Query()
	req := model.ListEntriesRequest{UserID: v["userId"], VaultID: v["vaultId"], MemoryID: v["memoryId"]}
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
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	if outs == nil {
		outs = []*model.MemoryEntry{}
	}
	platformHttp.WriteJSON(w, http.StatusOK, map[string]interface{}{"entries": outs, "count": len(outs)})
}

// CreateMemoryEntry POST /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries
func (h *MemoryV2Handler) CreateMemoryEntry(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	var in struct {
		RawEntry       string                 `json:"rawEntry"`
		Summary        *string                `json:"summary,omitempty"`
		Metadata       map[string]interface{} `json:"metadata,omitempty"`
		Tags           map[string]interface{} `json:"tags,omitempty"`
		ExpirationTime *time.Time             `json:"expirationTime,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		platformHttp.WriteBadRequest(w, "Invalid JSON")
		return
	}
	e := &model.MemoryEntry{
		UserID: v["userId"], VaultID: v["vaultId"], MemoryID: v["memoryId"],
		RawEntry: in.RawEntry, Summary: in.Summary, Metadata: in.Metadata, Tags: in.Tags, ExpirationTime: in.ExpirationTime,
	}
	out, err := h.svc.CreateEntry(r.Context(), e)
	if err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusCreated, out)
}

// GetMemoryEntryByID GET /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}
func (h *MemoryV2Handler) GetMemoryEntryByID(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	out, err := h.svc.GetEntryByID(r.Context(), v["userId"], v["vaultId"], v["memoryId"], v["entryId"])
	if err != nil {
		platformHttp.WriteNotFound(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusOK, out)
}

// UpdateMemoryEntryTags PATCH /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}/tags
func (h *MemoryV2Handler) UpdateMemoryEntryTags(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	var in struct {
		Tags map[string]interface{} `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		platformHttp.WriteBadRequest(w, "Invalid JSON")
		return
	}
	out, err := h.svc.UpdateEntryTags(r.Context(), v["userId"], v["vaultId"], v["memoryId"], v["entryId"], in.Tags)
	if err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusOK, out)
}

// PutMemoryContext PUT /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts
func (h *MemoryV2Handler) PutMemoryContext(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	var body struct {
		Context map[string]interface{} `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platformHttp.WriteBadRequest(w, "Invalid JSON")
		return
	}
	raw, err := json.Marshal(body.Context)
	if err != nil {
		platformHttp.WriteBadRequest(w, "context must be a valid JSON object")
		return
	}
	mc := &model.MemoryContext{UserID: v["userId"], VaultID: v["vaultId"], MemoryID: v["memoryId"], ContextJSON: raw}
	out, err := h.svc.PutContext(r.Context(), mc)
	if err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusCreated, out)
}

// GetLatestMemoryContext GET /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts
func (h *MemoryV2Handler) GetLatestMemoryContext(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	out, err := h.svc.GetLatestContext(r.Context(), v["userId"], v["vaultId"], v["memoryId"])
	if err != nil {
		platformHttp.WriteInternalError(w, err.Error())
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
		"userId":       out.UserID,
		"vaultId":      out.VaultID,
		"memoryId":     out.MemoryID,
		"creationTime": out.CreationTime,
		"context":      ctxVal,
	}
	platformHttp.WriteJSON(w, http.StatusOK, resp)
}

// GetMemoryByTitle GET /api/users/{userId}/vaults/{vaultTitle}/memories/{memoryTitle}
func (h *MemoryV2Handler) GetMemoryByTitle(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	if h.vaultSv == nil {
		platformHttp.WriteInternalError(w, "vault service unavailable")
		return
	}
	vaultObj, err := h.vaultSv.GetVaultByTitle(r.Context(), v["userId"], v["vaultTitle"])
	if err != nil {
		platformHttp.WriteNotFound(w, err.Error())
		return
	}
	mem, err := h.svc.GetMemoryByTitle(r.Context(), v["userId"], vaultObj.VaultID, v["memoryTitle"])
	if err != nil {
		platformHttp.WriteNotFound(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusOK, mem)
}

// ListMemoriesByVaultTitle GET /api/users/{userId}/vaults/{vaultTitle}/memories
func (h *MemoryV2Handler) ListMemoriesByVaultTitle(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	if h.vaultSv == nil {
		platformHttp.WriteInternalError(w, "vault service unavailable")
		return
	}
	vaultObj, err := h.vaultSv.GetVaultByTitle(r.Context(), v["userId"], v["vaultTitle"])
	if err != nil {
		platformHttp.WriteNotFound(w, err.Error())
		return
	}
	mems, err := h.svc.ListMemories(r.Context(), v["userId"], vaultObj.VaultID)
	if err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	if mems == nil {
		mems = []*model.Memory{}
	}
	platformHttp.WriteJSON(w, http.StatusOK, map[string]interface{}{"memories": mems, "count": len(mems)})
}

// DeleteMemory DELETE /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}
func (h *MemoryV2Handler) DeleteMemory(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	if err := h.svc.DeleteMemory(r.Context(), v["userId"], v["vaultId"], v["memoryId"]); err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteMemoryEntryByID DELETE /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}
func (h *MemoryV2Handler) DeleteMemoryEntryByID(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	if err := h.svc.DeleteEntry(r.Context(), v["userId"], v["vaultId"], v["memoryId"], v["entryId"]); err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteMemoryContextByID DELETE /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts/{contextId}
func (h *MemoryV2Handler) DeleteMemoryContextByID(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	if err := h.svc.DeleteContext(r.Context(), v["userId"], v["vaultId"], v["memoryId"], v["contextId"]); err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
