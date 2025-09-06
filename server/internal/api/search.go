package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"strings"
)

// SearchRequest represents the payload for POST /api/search
//
// Fields:
//
//	memoryId – required, non-empty string
//	query – required, non-empty string
//	top_ke – optional, entries top-k (default: 5, range: 0-25)
//	top_kc – optional, context shards top-k (default: 2, range: 1-10)
//	include_raw_entries – optional, whether to include raw entries in response (default: false)
//
// Validation is done via the Validate method.
// User identification comes from API key authorization.
//
// This DTO is intentionally small; future versions may add filters.
type SearchRequest struct {
	MemoryID          string `json:"memoryId"`
	Query             string `json:"query"`
	TopKE             *int   `json:"top_ke,omitempty"`
	TopKC             *int   `json:"top_kc,omitempty"`
	IncludeRawEntries bool   `json:"include_raw_entries,omitempty"`
}

// Validate sanitises the struct and applies defaults.
func (r *SearchRequest) Validate() error {
	r.Query = strings.TrimSpace(r.Query)

	if r.MemoryID == "" {
		return errors.New("memoryId is required")
	}
	if r.Query == "" {
		return errors.New("query cannot be empty")
	}

	// Apply defaults if not set (nil pointer)
	if r.TopKE == nil {
		defaultKE := 5
		r.TopKE = &defaultKE
	}
	if r.TopKC == nil {
		defaultKC := 2
		r.TopKC = &defaultKC
	}

	// Validate ranges
	if *r.TopKE < 0 || *r.TopKE > 25 {
		return errors.New("top_ke must be between 0 and 25")
	}
	if *r.TopKC < 1 || *r.TopKC > 10 {
		return errors.New("top_kc must be between 1 and 10")
	}

	return nil
}

// decodeSearchRequest helper parses JSON into SearchRequest and validates it.
func decodeSearchRequest(w http.ResponseWriter, r *http.Request) (*SearchRequest, error) {
	// w is currently unused but kept for compatibility; mark to avoid linters
	_ = w
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return &req, nil
}
