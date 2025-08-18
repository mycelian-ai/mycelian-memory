package api

import (
	"fmt"
	"net/http"
	"time"

	respond "github.com/mycelian/mycelian-memory/server/internal/api/respond"
	"github.com/mycelian/mycelian-memory/server/internal/auth"
	emb "github.com/mycelian/mycelian-memory/server/internal/embeddings"
	"github.com/mycelian/mycelian-memory/server/internal/searchindex"
)

// SearchHandler handles POST /api/search using native searchindex and embeddings.
type SearchHandler struct {
	emb        emb.EmbeddingProvider
	idx        searchindex.Index
	alpha      float32
	authorizer auth.Authorizer
}

func NewSearchHandler(emb emb.EmbeddingProvider, idx searchindex.Index, alpha float32, authorizer auth.Authorizer) (*SearchHandler, error) {
	if alpha < 0.0 || alpha > 1.0 {
		return nil, fmt.Errorf("alpha parameter must be in the range [0.0, 1.0], got %f", alpha)
	}
	return &SearchHandler{emb: emb, idx: idx, alpha: alpha, authorizer: authorizer}, nil
}

func (h *SearchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	// Extract API key from Authorization header
	apiKey, err := auth.ExtractAPIKey(r)
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// Authorize the request
	actorInfo, err := h.authorizer.Authorize(r.Context(), apiKey, "memory.search", "default")
	if err != nil {
		respond.WriteError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	req, err := decodeSearchRequest(w, r)
	if err != nil {
		respond.WriteBadRequest(w, err.Error())
		return
	}
	if h.emb == nil || h.idx == nil {
		respond.WriteError(w, http.StatusServiceUnavailable, "search not configured")
		return
	}

	vec, err := h.emb.Embed(r.Context(), req.Query)
	if err != nil {
		respond.WriteError(w, http.StatusInternalServerError, "embedding service unavailable")
		return
	}

	hits, err := h.idx.Search(r.Context(), actorInfo.ActorID, req.MemoryID, req.Query, vec, req.TopK, h.alpha)
	if err != nil {
		respond.WriteError(w, http.StatusInternalServerError, "search service unavailable")
		return
	}

	// Build response consistent with previous keys
	resp := map[string]interface{}{
		"entries": hits,
		"count":   len(hits),
	}

	// Latest context
	ctxStr, ts, err := h.idx.LatestContext(r.Context(), actorInfo.ActorID, req.MemoryID)
	if err != nil {
		respond.WriteError(w, http.StatusInternalServerError, "latest context unavailable")
		return
	}
	resp["latestContext"] = ctxStr
	resp["contextTimestamp"] = ts.Format(time.RFC3339)

	// Best-matching context
	best, bts, score, err := h.idx.BestContext(r.Context(), actorInfo.ActorID, req.MemoryID, req.Query, vec, h.alpha)
	if err != nil {
		respond.WriteError(w, http.StatusInternalServerError, "best context unavailable")
		return
	}
	resp["bestContext"] = best
	resp["bestContextTimestamp"] = bts.Format(time.RFC3339)
	resp["bestContextScore"] = score

	respond.WriteJSON(w, http.StatusOK, resp)
}
