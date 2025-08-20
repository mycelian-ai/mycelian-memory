package client

import (
	"github.com/mycelian/mycelian-memory/client/internal/types"
	prompts "github.com/mycelian/mycelian-memory/client/prompts"
)

// Public re-exports so SDK users only import package client.
type DefaultPromptResponse = prompts.DefaultPromptResponse

// Public type aliases so SDK consumers can import only the client package.
// Requests
// Note: User-related requests are intentionally omitted (external user management).
type (
	CreateVaultRequest  = types.CreateVaultRequest
	CreateMemoryRequest = types.CreateMemoryRequest
	AddEntryRequest     = types.AddEntryRequest
	PutContextRequest   = types.PutContextRequest
	SearchRequest       = types.SearchRequest

	// Domain entities
	Vault  = types.Vault
	Memory = types.Memory
	Entry  = types.Entry

	// Responses
	EnqueueAck          = types.EnqueueAck
	ListEntriesResponse = types.ListEntriesResponse
	GetContextResponse  = types.GetContextResponse
	SearchEntry         = types.SearchEntry
	SearchResponse      = types.SearchResponse
)

// Errors re-exported in errors.go
