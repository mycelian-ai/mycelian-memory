package http

import (
	"net/http"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/embeddings"
	platformHttp "github.com/mycelian/mycelian-memory/server/internal/platform/http"
	"github.com/mycelian/mycelian-memory/server/internal/searchindex"
)

// SearchV2Handler handles POST /api/search using native searchindex and embeddings.
type SearchV2Handler struct {
	emb   embeddings.Provider
	idx   searchindex.Index
	alpha float32
}

func NewSearchV2Handler(emb embeddings.Provider, idx searchindex.Index, alpha float32) *SearchV2Handler {
	return &SearchV2Handler{emb: emb, idx: idx, alpha: alpha}
}

func (h *SearchV2Handler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	req, err := decodeSearchRequest(w, r)
	if err != nil {
		platformHttp.WriteBadRequest(w, err.Error())
		return
	}
	if h.emb == nil || h.idx == nil {
		platformHttp.WriteError(w, http.StatusServiceUnavailable, "search not configured")
		return
	}

	vec, err := h.emb.Embed(r.Context(), req.Query)
	if err != nil {
		platformHttp.WriteError(w, http.StatusInternalServerError, "embedding service unavailable")
		return
	}

	hits, err := h.idx.Search(r.Context(), req.UserID, req.MemoryID, req.Query, vec, req.TopK, h.alpha)
	if err != nil {
		platformHttp.WriteError(w, http.StatusInternalServerError, "search service unavailable")
		return
	}

	// Build response consistent with v1 keys
	resp := map[string]interface{}{
		"entries": hits,
		"count":   len(hits),
	}

	// Latest context
	ctxStr, ts, err := h.idx.LatestContext(r.Context(), req.UserID, req.MemoryID)
	if err != nil {
		platformHttp.WriteError(w, http.StatusInternalServerError, "latest context unavailable")
		return
	}
	resp["latestContext"] = ctxStr
	resp["contextTimestamp"] = ts.Format(time.RFC3339)

	// Best-matching context
	best, bts, score, err := h.idx.BestContext(r.Context(), req.UserID, req.MemoryID, req.Query, vec, h.alpha)
	if err != nil {
		platformHttp.WriteError(w, http.StatusInternalServerError, "best context unavailable")
		return
	}
	resp["bestContext"] = best
	resp["bestContextTimestamp"] = bts.Format(time.RFC3339)
	resp["bestContextScore"] = score

	platformHttp.WriteJSON(w, http.StatusOK, resp)
}
