package http

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
//	userId – required, non-empty string
//	memoryId – required, non-empty string
//	query – required, non-empty string
//	topK  – optional, 1-100 (defaults to 10)
//
// Validation is done via the Validate method.
//
// This DTO is intentionally small; future versions may add filters.
type SearchRequest struct {
	UserID   string `json:"userId"`
	MemoryID string `json:"memoryId"`
	Query    string `json:"query"`
	TopK     int    `json:"topK,omitempty"`
}

// Validate sanitises the struct and applies defaults.
func (r *SearchRequest) Validate() error {
	r.Query = strings.TrimSpace(r.Query)
	if r.UserID == "" {
		return errors.New("userId is required")
	}
	if r.MemoryID == "" {
		return errors.New("memoryId is required")
	}
	if r.Query == "" {
		return errors.New("query cannot be empty")
	}
	if r.TopK <= 0 {
		r.TopK = 10
	}
	if r.TopK > 100 {
		r.TopK = 100
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
