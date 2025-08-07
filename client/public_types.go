package client

import "github.com/mycelian/mycelian-memory/client/internal/types"

// Public type aliases so SDK consumers can import only the client package.
// Requests
type (
	CreateUserRequest   = types.CreateUserRequest
	CreateVaultRequest  = types.CreateVaultRequest
	CreateMemoryRequest = types.CreateMemoryRequest
	AddEntryRequest     = types.AddEntryRequest
	PutContextRequest   = types.PutContextRequest
	SearchRequest       = types.SearchRequest

	// Domain entities
	User   = types.User
	Vault  = types.Vault
	Memory = types.Memory
	Entry  = types.Entry

	// Responses
	EnqueueAck           = types.EnqueueAck
	ListEntriesResponse  = types.ListEntriesResponse
	GetContextResponse   = types.GetContextResponse
	SearchEntry          = types.SearchEntry
	SearchResponse       = types.SearchResponse
	ListMemoriesResponse = types.ListMemoriesResponse
	ListVaultsResponse   = types.ListVaultsResponse
)

// Errors re-exported in errors.go
