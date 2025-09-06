package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	respond "github.com/mycelian/mycelian-memory/server/internal/api/respond"
	"github.com/mycelian/mycelian-memory/server/internal/auth"
	emb "github.com/mycelian/mycelian-memory/server/internal/embeddings"
	"github.com/mycelian/mycelian-memory/server/internal/model"
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

	log.Info().Str("memoryId", req.MemoryID).Str("query", req.Query).Int("top_ke", *req.TopKE).Int("top_kc", *req.TopKC).Str("actorId", actorInfo.ActorID).Msg("search request received")

	vec, err := h.emb.Embed(r.Context(), req.Query)
	if err != nil {
		log.Error().Err(err).Str("query", req.Query).Msg("embedding failed")
		respond.WriteError(w, http.StatusInternalServerError, "embedding service unavailable")
		return
	}
	log.Debug().Int("vectorLength", len(vec)).Msg("embedding generated")

	// Search for entries if top_ke > 0
	var hits []model.SearchHit
	if *req.TopKE > 0 {
		hits, err = h.idx.Search(r.Context(), actorInfo.ActorID, req.MemoryID, req.Query, vec, *req.TopKE, h.alpha, req.IncludeRawEntries)
		if err != nil {
			log.Error().Err(err).Str("memoryId", req.MemoryID).Str("query", req.Query).Msg("search failed")
			respond.WriteError(w, http.StatusInternalServerError, "search service unavailable")
			return
		}
	}
	log.Info().Int("hitCount", len(hits)).Str("memoryId", req.MemoryID).Msg("search completed")

	// Always fetch latest context
	latestCtx, latestTs, err := h.idx.LatestContext(r.Context(), actorInfo.ActorID, req.MemoryID)
	if err != nil {
		log.Error().Err(err).Str("memoryId", req.MemoryID).Msg("latest context fetch failed")
		respond.WriteError(w, http.StatusInternalServerError, "latest context unavailable")
		return
	}

	// Search for context shards (always, since minimum is 1)
	ctxHits, err := h.idx.SearchContexts(r.Context(), actorInfo.ActorID, req.MemoryID, req.Query, vec, *req.TopKC, h.alpha)
	if err != nil {
		log.Error().Err(err).Str("memoryId", req.MemoryID).Msg("context search failed")
		respond.WriteError(w, http.StatusInternalServerError, "context search unavailable")
		return
	}

	// Build contexts array
	contexts := make([]map[string]any, 0, len(ctxHits))
	for _, ch := range ctxHits {
		contexts = append(contexts, map[string]any{
			"context":   ch.Context,
			"timestamp": ch.Timestamp.Format(time.RFC3339),
			"score":     ch.Score,
		})
	}

	// Build response with consistent structure
	resp := map[string]interface{}{
		"entries":                hits,
		"count":                  len(hits),
		"latestContext":          latestCtx,
		"latestContextTimestamp": latestTs.Format(time.RFC3339),
		"contexts":               contexts,
	}

	respond.WriteJSON(w, http.StatusOK, resp)
}
