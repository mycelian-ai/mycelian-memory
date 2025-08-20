package client

import (
	"github.com/mycelian/mycelian-memory/client/internal/types"
	prompts "github.com/mycelian/mycelian-memory/client/prompts"
)

// Public type surface
//
// Re-export a small set of types so callers only import
// "github.com/mycelian/mycelian-memory/client". These are zero-cost aliases,
// identical to the original definitions in their source packages.

// DefaultPromptResponse is returned by LoadDefaultPrompts.
// Re-exported to avoid importing the prompts subpackage in user code.
type DefaultPromptResponse = prompts.DefaultPromptResponse

// Request, entity, and response types
// Note: user-related types are intentionally omitted.
type (
	// Requests
	CreateVaultRequest  = types.CreateVaultRequest
	CreateMemoryRequest = types.CreateMemoryRequest
	AddEntryRequest     = types.AddEntryRequest
	PutContextRequest   = types.PutContextRequest
	SearchRequest       = types.SearchRequest

	// Entities
	Vault   = types.Vault
	Memory  = types.Memory
	Entry   = types.Entry
	Context = types.Context

	// Responses
	EnqueueAck          = types.EnqueueAck
	ListEntriesResponse = types.ListEntriesResponse
	SearchEntry         = types.SearchEntry
	SearchResponse      = types.SearchResponse
)

// See errors.go for exported error variables (e.g., ErrNotFound).
