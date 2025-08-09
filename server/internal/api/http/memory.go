package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/api/validate"
	"github.com/mycelian/mycelian-memory/server/internal/core/memory"
	"github.com/mycelian/mycelian-memory/server/internal/core/vault"
	platformHttp "github.com/mycelian/mycelian-memory/server/internal/platform/http"
	"github.com/mycelian/mycelian-memory/server/internal/search"
	"github.com/mycelian/mycelian-memory/server/internal/storage"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// MemoryHandler handles memory-related HTTP requests (thin transport layer)
type MemoryHandler struct {
	memoryService *memory.Service
	vaultService  *vault.Service

	// Optional search index connector for best-effort delete propagation
	search search.Searcher
}

// NewMemoryHandler creates a new memory handler
func NewMemoryHandler(memoryService *memory.Service, vaultService *vault.Service) *MemoryHandler {
	return &MemoryHandler{
		memoryService: memoryService,
		vaultService:  vaultService,
	}
}

// WithSearcher wires a searcher into the handler (optional).
func (h *MemoryHandler) WithSearcher(s search.Searcher) *MemoryHandler {
	h.search = s
	return h
}

// CreateUser handles POST /api/users
func (h *MemoryHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID      string  `json:"userId"`
		Email       string  `json:"email"`
		DisplayName *string `json:"displayName,omitempty"`
		TimeZone    string  `json:"timeZone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platformHttp.WriteBadRequest(w, "Invalid JSON")
		return
	}

	// Allow email-only user creation in dev flows by deriving a valid userId when omitted
	if req.UserID == "" && req.Email != "" {
		req.UserID = deriveUserIDFromEmail(req.Email)
	}

	// Validation
	if err := validate.CreateUser(req.UserID, req.Email, req.DisplayName); err != nil {
		platformHttp.WriteBadRequest(w, err.Error())
		return
	}

	// Convert to domain request
	createReq := memory.CreateUserRequest{
		UserID:      req.UserID,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		TimeZone:    req.TimeZone,
	}

	user, err := h.memoryService.CreateUser(r.Context(), createReq)
	if err != nil {
		if memory.IsValidationError(err) {
			platformHttp.WriteBadRequest(w, err.Error())
		} else if memory.IsConflictError(err) {
			platformHttp.WriteError(w, http.StatusConflict, err.Error())
		} else {
			platformHttp.WriteInternalError(w, err.Error())
		}
		return
	}

	platformHttp.WriteJSON(w, http.StatusCreated, user)
}

// deriveUserIDFromEmail creates a valid userId from an email address using the
// allowed character set [a-z0-9_] and max length 20. If derivation fails, it
// falls back to a short UUID-based id.
func deriveUserIDFromEmail(email string) string {
	local := email
	if i := strings.IndexByte(email, '@'); i >= 0 {
		local = email[:i]
	}
	local = strings.ToLower(local)
	// map invalid characters to underscore
	b := make([]byte, 0, len(local))
	for i := 0; i < len(local) && len(b) < 20; i++ {
		c := local[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			b = append(b, c)
		} else if c == '-' || c == '.' {
			b = append(b, '_')
		} else {
			b = append(b, '_')
		}
	}
	// collapse multiple underscores and trim leading/trailing underscores
	// without importing regexp for lightweight processing
	// First collapse
	collapsed := make([]byte, 0, len(b))
	var prevUnderscore bool
	for _, c := range b {
		if c == '_' {
			if prevUnderscore {
				continue
			}
			prevUnderscore = true
			collapsed = append(collapsed, c)
			continue
		}
		prevUnderscore = false
		collapsed = append(collapsed, c)
	}
	// Trim leading/trailing underscores
	start, end := 0, len(collapsed)
	for start < end && collapsed[start] == '_' {
		start++
	}
	for end > start && collapsed[end-1] == '_' {
		end--
	}
	if end > start {
		id := string(collapsed[start:end])
		if len(id) > 20 {
			id = id[:20]
		}
		return id
	}
	// fallback
	uid := strings.ReplaceAll(uuid.New().String(), "-", "")
	if len(uid) > 12 {
		uid = uid[:12]
	}
	return "user_" + uid
}

// GetUser handles GET /api/users/{userId}
func (h *MemoryHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	user, err := h.memoryService.GetUser(r.Context(), userID)
	if err != nil {
		platformHttp.WriteNotFound(w, err.Error())
		return
	}

	platformHttp.WriteJSON(w, http.StatusOK, user)
}

// CreateMemory handles POST /api/users/{userId}/memories
func (h *MemoryHandler) CreateMemory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	var req struct {
		MemoryType  string  `json:"memoryType"`
		Title       string  `json:"title"`
		Description *string `json:"description,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platformHttp.WriteBadRequest(w, "Invalid JSON")
		return
	}

	// Validate input
	if err := validate.CreateMemory(req.MemoryType, req.Title, req.Description); err != nil {
		platformHttp.WriteBadRequest(w, err.Error())
		return
	}

	vaultIDStr := vars["vaultId"]
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		platformHttp.WriteBadRequest(w, "invalid vaultId")
		return
	}

	// Convert to domain request
	createReq := memory.CreateMemoryRequest{
		UserID:      userID,
		VaultID:     vaultID,
		MemoryType:  req.MemoryType,
		Title:       req.Title,
		Description: req.Description,
	}

	mem, err := h.memoryService.CreateMemory(r.Context(), createReq)
	if err != nil {
		if memory.IsValidationError(err) {
			platformHttp.WriteBadRequest(w, err.Error())
		} else {
			platformHttp.WriteInternalError(w, err.Error())
		}
		return
	}

	platformHttp.WriteJSON(w, http.StatusCreated, mem)
}

// GetMemory handles GET /api/users/{userId}/memories/{memoryId}
func (h *MemoryHandler) GetMemory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultIDStr := vars["vaultId"]
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		platformHttp.WriteBadRequest(w, "invalid vaultId")
		return
	}
	memoryID := vars["memoryId"]

	mem, err := h.memoryService.GetMemory(r.Context(), userID, vaultID, memoryID)
	if err != nil {
		platformHttp.WriteNotFound(w, err.Error())
		return
	}

	platformHttp.WriteJSON(w, http.StatusOK, mem)
}

// ListMemories handles GET /api/users/{userId}/memories
func (h *MemoryHandler) ListMemories(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultIDStr := vars["vaultId"]
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		platformHttp.WriteBadRequest(w, "invalid vaultId")
		return
	}

	memories, err := h.memoryService.ListMemories(r.Context(), userID, vaultID)
	if err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}

	// Ensure memories is never nil - return empty slice instead
	if memories == nil {
		memories = []*storage.Memory{}
	}

	response := map[string]interface{}{
		"memories": memories,
		"count":    len(memories),
	}

	platformHttp.WriteJSON(w, http.StatusOK, response)
}

// UpdateMemoryEntryTags handles PATCH /api/users/{userId}/memories/{memoryId}/entries/{entryId}/tags
func (h *MemoryHandler) UpdateMemoryEntryTags(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultIDStr := vars["vaultId"]
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		platformHttp.WriteBadRequest(w, "invalid vaultId")
		return
	}
	memoryID := vars["memoryId"]
	entryID := vars["entryId"]
	if entryID == "" {
		platformHttp.WriteBadRequest(w, "entryId is required")
		return
	}

	var req struct {
		Tags map[string]interface{} `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platformHttp.WriteBadRequest(w, "Invalid JSON")
		return
	}

	// Basic validation on tags object
	if req.Tags == nil {
		platformHttp.WriteBadRequest(w, "tags field is required")
		return
	}
	if err := validate.IsJSONObject(req.Tags); err != nil {
		platformHttp.WriteBadRequest(w, err.Error())
		return
	}

	// Convert to domain request
	updateReq := memory.UpdateMemoryEntryTagsRequest{
		UserID:   userID,
		VaultID:  vaultID,
		MemoryID: memoryID,
		EntryID:  entryID,
		Tags:     req.Tags,
	}

	entry, err := h.memoryService.UpdateMemoryEntryTags(r.Context(), updateReq)
	if err != nil {
		if memory.IsValidationError(err) {
			platformHttp.WriteBadRequest(w, err.Error())
		} else {
			platformHttp.WriteInternalError(w, err.Error())
		}
		return
	}

	platformHttp.WriteJSON(w, http.StatusOK, entry)
}

// DeleteMemory handles DELETE /api/users/{userId}/memories/{memoryId}
func (h *MemoryHandler) DeleteMemory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultIDStr := vars["vaultId"]
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		platformHttp.WriteBadRequest(w, "invalid vaultId")
		return
	}
	memoryID := vars["memoryId"]

	err = h.memoryService.DeleteMemory(r.Context(), userID, vaultID, memoryID)
	if err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateMemoryEntry handles POST /api/users/{userId}/memories/{memoryId}/entries
func (h *MemoryHandler) CreateMemoryEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultIDStr := vars["vaultId"]
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		platformHttp.WriteBadRequest(w, "invalid vaultId")
		return
	}
	memoryID := vars["memoryId"]

	var req struct {
		RawEntry       string                 `json:"rawEntry"`
		Summary        *string                `json:"summary,omitempty"`
		Metadata       map[string]interface{} `json:"metadata,omitempty"`
		Tags           map[string]interface{} `json:"tags,omitempty"`
		ExpirationTime *time.Time             `json:"expirationTime,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platformHttp.WriteBadRequest(w, "Invalid JSON")
		return
	}

	// Validate
	if err := validate.CreateMemoryEntry(req.RawEntry, req.Summary, req.Metadata, req.Tags); err != nil {
		platformHttp.WriteBadRequest(w, err.Error())
		return
	}

	// Convert to domain request
	createReq := memory.CreateMemoryEntryRequest{
		UserID:         userID,
		VaultID:        vaultID,
		MemoryID:       memoryID,
		RawEntry:       req.RawEntry,
		Summary:        req.Summary,
		Metadata:       req.Metadata,
		Tags:           req.Tags,
		ExpirationTime: req.ExpirationTime,
	}

	entry, err := h.memoryService.CreateMemoryEntry(r.Context(), createReq)
	if err != nil {
		if memory.IsValidationError(err) {
			platformHttp.WriteBadRequest(w, err.Error())
		} else {
			platformHttp.WriteInternalError(w, err.Error())
		}
		return
	}

	platformHttp.WriteJSON(w, http.StatusCreated, entry)
}

// ListMemoryEntries handles GET /api/users/{userId}/memories/{memoryId}/entries
func (h *MemoryHandler) ListMemoryEntries(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	memoryID := vars["memoryId"]

	// Parse query parameters
	query := r.URL.Query()

	req := memory.ListMemoryEntriesRequest{
		UserID:   userID,
		MemoryID: memoryID,
	}

	// Parse limit
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			req.Limit = limit
		}
	}

	// Parse before timestamp
	if beforeStr := query.Get("before"); beforeStr != "" {
		if before, err := time.Parse(time.RFC3339, beforeStr); err == nil {
			req.Before = &before
		}
	}

	// Parse after timestamp
	if afterStr := query.Get("after"); afterStr != "" {
		if after, err := time.Parse(time.RFC3339, afterStr); err == nil {
			req.After = &after
		}
	}

	// add vaultId parse
	vaultIDStr := vars["vaultId"]
	vaultID, err2 := uuid.Parse(vaultIDStr)
	if err2 != nil {
		platformHttp.WriteBadRequest(w, "invalid vaultId")
		return
	}

	req.VaultID = vaultID

	entries, err := h.memoryService.ListMemoryEntries(r.Context(), req)
	if err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}

	// Ensure entries is never nil - return empty slice instead
	if entries == nil {
		entries = []*storage.MemoryEntry{}
	}

	response := map[string]interface{}{
		"entries": entries,
		"count":   len(entries),
	}

	platformHttp.WriteJSON(w, http.StatusOK, response)
}

// PutMemoryContext handles PUT /api/users/{userId}/memories/{memoryId}/contexts
func (h *MemoryHandler) PutMemoryContext(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultIDStr := vars["vaultId"]
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		platformHttp.WriteBadRequest(w, "invalid vaultId")
		return
	}
	memoryID := vars["memoryId"]

	// Parse request body
	var body struct {
		Context map[string]interface{} `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platformHttp.WriteBadRequest(w, "Invalid JSON")
		return
	}
	if err := validate.ContextFragments(body.Context); err != nil {
		platformHttp.WriteBadRequest(w, err.Error())
		return
	}

	// Marshal the context object back to raw JSON for pass-through
	raw, err := json.Marshal(body.Context)
	if err != nil {
		platformHttp.WriteBadRequest(w, "context must be a valid JSON object")
		return
	}

	// Build domain request
	createReq := memory.CreateMemoryContextRequest{
		UserID:   userID,
		VaultID:  vaultID,
		MemoryID: memoryID,
		Context:  raw,
	}

	ctxObj, err := h.memoryService.CreateMemoryContext(r.Context(), createReq)
	if err != nil {
		if memory.IsValidationError(err) {
			platformHttp.WriteBadRequest(w, err.Error())
		} else {
			platformHttp.WriteInternalError(w, err.Error())
		}
		return
	}

	platformHttp.WriteJSON(w, http.StatusCreated, ctxObj)
}

// GetLatestMemoryContext handles GET /api/users/{userId}/memories/{memoryId}/contexts
func (h *MemoryHandler) GetLatestMemoryContext(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultIDStr := vars["vaultId"]
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		platformHttp.WriteBadRequest(w, "invalid vaultId")
		return
	}
	memoryID := vars["memoryId"]

	ctxObj, err := h.memoryService.GetLatestMemoryContext(r.Context(), userID, vaultID, memoryID)
	if err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}

	platformHttp.WriteJSON(w, http.StatusOK, ctxObj)
}

// DeleteMemoryEntryByID handles DELETE /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}
func (h *MemoryHandler) DeleteMemoryEntryByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultIDStr := vars["vaultId"]
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		platformHttp.WriteBadRequest(w, "invalid vaultId")
		return
	}
	memoryID := vars["memoryId"]
	entryID := vars["entryId"]
	if entryID == "" {
		platformHttp.WriteBadRequest(w, "entryId is required")
		return
	}
	if err := h.memoryService.DeleteMemoryEntryByID(r.Context(), userID, vaultID, memoryID, entryID); err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	// Best-effort index cleanup (fire-and-forget)
	if h.search != nil {
		go func(uid, eid string) { _ = h.search.DeleteEntry(context.Background(), uid, eid) }(userID, entryID)
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteMemoryContextByID handles DELETE /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts/{contextId}
func (h *MemoryHandler) DeleteMemoryContextByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultIDStr := vars["vaultId"]
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		platformHttp.WriteBadRequest(w, "invalid vaultId")
		return
	}
	memoryID := vars["memoryId"]
	contextID := vars["contextId"]
	if contextID == "" {
		platformHttp.WriteBadRequest(w, "contextId is required")
		return
	}
	if err := h.memoryService.DeleteMemoryContextByID(r.Context(), userID, vaultID, memoryID, contextID); err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	if h.search != nil {
		go func(uid, cid string) { _ = h.search.DeleteContext(context.Background(), uid, cid) }(userID, contextID)
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Title-based endpoints ---

// ListMemoriesByVaultTitle GET /api/users/{userId}/vaults/{vaultTitle}/memories
func (h *MemoryHandler) ListMemoriesByVaultTitle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultTitle := vars["vaultTitle"]

	v, err := h.vaultService.GetVaultByTitle(r.Context(), userID, vaultTitle)
	if err != nil {
		platformHttp.WriteNotFound(w, err.Error())
		return
	}

	memories, err := h.memoryService.ListMemories(r.Context(), userID, v.VaultID)
	if err != nil {
		platformHttp.WriteInternalError(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusOK, map[string]interface{}{"memories": memories, "count": len(memories)})
}

// GetMemoryByTitle GET /api/users/{userId}/vaults/{vaultTitle}/memories/{memoryTitle}
func (h *MemoryHandler) GetMemoryByTitle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultTitle := vars["vaultTitle"]
	memTitle := vars["memoryTitle"]

	v, err := h.vaultService.GetVaultByTitle(r.Context(), userID, vaultTitle)
	if err != nil {
		platformHttp.WriteNotFound(w, err.Error())
		return
	}

	m, err := h.memoryService.GetMemoryByTitle(r.Context(), userID, v.VaultID, memTitle)
	if err != nil {
		platformHttp.WriteNotFound(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusOK, m)
}

// GetMemoryEntryByID handles GET /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}
func (h *MemoryHandler) GetMemoryEntryByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultIDStr := vars["vaultId"]
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		platformHttp.WriteBadRequest(w, "invalid vaultId")
		return
	}
	memoryID := vars["memoryId"]
	entryID := vars["entryId"]

	if entryID == "" {
		platformHttp.WriteBadRequest(w, "entryId is required")
		return
	}

	entry, err := h.memoryService.GetMemoryEntryByID(r.Context(), userID, vaultID, memoryID, entryID)
	if err != nil {
		platformHttp.WriteNotFound(w, err.Error())
		return
	}
	platformHttp.WriteJSON(w, http.StatusOK, entry)
}

// AttachMemoryToVault handles POST /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/attach
// This operation moves the memory (and its entries/contexts) to the target vault.
// No request body is expected; parameters are taken from the path.
func (h *MemoryHandler) AttachMemoryToVault(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	vaultIDStr := vars["vaultId"]
	memoryID := vars["memoryId"]

	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		platformHttp.WriteBadRequest(w, "invalid vaultId")
		return
	}
	if userID == "" || memoryID == "" {
		platformHttp.WriteBadRequest(w, "userId and memoryId are required")
		return
	}

	req := vault.AddMemoryToVaultRequest{
		UserID:   userID,
		VaultID:  vaultID,
		MemoryID: memoryID,
	}
	if err := h.vaultService.AddMemoryToVault(r.Context(), req); err != nil {
		// Map expected domain/storage errors to HTTP
		switch {
		case memory.IsValidationError(err):
			platformHttp.WriteBadRequest(w, err.Error())
		default:
			// Treat conflict and not found explicitly by substring match (thin mapping)
			if strings.Contains(err.Error(), "VAULT_NOT_FOUND") || strings.Contains(err.Error(), "MEMORY_NOT_FOUND") {
				platformHttp.WriteNotFound(w, err.Error())
			} else if strings.Contains(err.Error(), "MEMORY_TITLE_CONFLICT") {
				platformHttp.WriteError(w, http.StatusConflict, err.Error())
			} else {
				platformHttp.WriteInternalError(w, err.Error())
			}
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
