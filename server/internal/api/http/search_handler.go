package http

import (
	"errors"
	"net/http"
	"time"

	platformHttp "memory-backend/internal/platform/http"
	"memory-backend/internal/search"

	"github.com/rs/zerolog/log"

	gql "github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
)

// SearchHandler handles POST /api/search
type SearchHandler struct {
	embedder search.Embedder
	searcher search.Searcher
	alpha    float32
}

// NewSearchHandler instantiates the handler with dependencies.
func NewSearchHandler(embedder search.Embedder, searcher search.Searcher, alpha float32) *SearchHandler {
	return &SearchHandler{embedder: embedder, searcher: searcher, alpha: alpha}
}

// HandleSearch processes incoming search requests.
func (h *SearchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	req, err := decodeSearchRequest(w, r)
	if err != nil {
		platformHttp.WriteBadRequest(w, err.Error())
		return
	}

	// Generate embedding
	vec, err := h.embedder.Embed(r.Context(), req.Query)
	if err != nil {
		log.Warn().Err(err).Str("query", req.Query).Msg("embedding failed")
		platformHttp.WriteError(w, http.StatusInternalServerError, "embedding service unavailable")
		return
	}

	// Execute hybrid search (tenant scoped)
	results, err := h.searcher.Search(r.Context(), req.UserID, req.MemoryID, req.Query, vec, req.TopK, h.alpha)
	if err != nil {
		if errors.Is(err, search.ErrTenantNotFound) {
			platformHttp.WriteBadRequest(w, "tenant not found")
			return
		}
		log.Warn().Err(err).Msg("vector search failed")
		platformHttp.WriteError(w, http.StatusInternalServerError, "search service unavailable")
		return
	}

	response := map[string]interface{}{
		"entries": results,
		"count":   len(results),
	}

	// Always include latestContext and bestContext keys
	response["latestContext"] = ""
	response["bestContext"] = ""

	// Fetch latest context snapshot independent of entries
	ctxStr, ts, err := h.searcher.LatestContext(r.Context(), req.UserID, req.MemoryID)
	if err != nil || ctxStr == "" {
		log.Error().Err(err).Msg("missing latest context for memory; invariant violated")
		platformHttp.WriteError(w, http.StatusInternalServerError, "latest context unavailable")
		return
	}
	response["latestContext"] = ctxStr
	response["contextTimestamp"] = ts.Format(time.RFC3339)

	// Fetch best-matching context using hybrid search
	bestCtx, bts, score, err := h.searcher.BestContext(r.Context(), req.UserID, req.MemoryID, req.Query, vec, h.alpha)
	if err != nil || bestCtx == "" {
		log.Error().Err(err).Msg("missing best context for memory; invariant violated")
		platformHttp.WriteError(w, http.StatusInternalServerError, "best context unavailable")
		return
	}
	response["bestContext"] = bestCtx
	response["bestContextTimestamp"] = bts.Format(time.RFC3339)
	response["bestContextScore"] = score

	platformHttp.WriteJSON(w, http.StatusOK, response)
}

// buildHybrid constructs a hybrid argument with summary/rawEntry properties.
func buildHybrid(query string, vec []float32, alpha float32) *gql.HybridArgumentBuilder {
	hy := (&gql.HybridArgumentBuilder{}).
		WithQuery(query).
		WithVector(vec).
		WithAlpha(alpha).
		WithProperties([]string{"summary", "rawEntry"})
	return hy
}
