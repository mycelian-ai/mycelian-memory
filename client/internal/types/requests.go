package types

import "time"

// ------------------------------
// Request Types
// ------------------------------

// CreateUserRequest holds parameters for new user
type CreateUserRequest struct {
	UserID      string `json:"userId"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName,omitempty"`
	TimeZone    string `json:"timeZone,omitempty"`
}

// CreateVaultRequest holds parameters for new vault
type CreateVaultRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// CreateMemoryRequest holds parameters for new memory
type CreateMemoryRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	MemoryType  string `json:"memoryType"`
}

// AddEntryRequest holds parameters for new entry
type AddEntryRequest struct {
	RawEntry       string                 `json:"rawEntry"`
	Summary        string                 `json:"summary,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Tags           map[string]string      `json:"tags,omitempty"`
	ExpirationTime *time.Time             `json:"expirationTime,omitempty"`
}

// SearchRequest holds search parameters
type SearchRequest struct {
	UserID            string `json:"actorId"`
	VaultID           string `json:"vaultId,omitempty"`
	MemoryID          string `json:"memoryId"`
	Query             string `json:"query"`
	TopKE             *int   `json:"top_ke,omitempty"`
	TopKC             *int   `json:"top_kc,omitempty"`
	IncludeRawEntries bool   `json:"include_raw_entries,omitempty"`
}
