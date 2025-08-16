# Client SDK

**Type**: Component Documentation  
**Status**: Active  

## Overview

The Mycelian Go Client SDK provides a type-safe, idiomatic Go interface for interacting with the Memory Service backend. It serves as the primary library for applications, tools, and the MCP server to perform memory operations.

The SDK replaced the legacy `pkg/memoryclient` to eliminate code duplication, provide proper retry/sharding support, and deliver a cleaner developer experience.

## Architecture

```
Application Code
    ↓ (Go API calls)
Go Client SDK
    ├── Direct Methods (client.CreateMemory, client.AddEntry, etc.)
    ├── Async Executor (ShardQueue for ordering)
    └── HTTP Layer (REST calls)
        ↓ (HTTP/REST)
Memory Service Backend
```

The SDK provides a unified interface that handles:
- **HTTP communication** with the Memory Service backend
- **Async operation ordering** via ShardQueue executor
- **Type safety** through comprehensive Go types
- **Error handling** with proper Go error patterns
- **Configuration** via functional options

## Core Components

### Client Structure

```go
type Client struct {
    baseURL string        // Memory Service endpoint
    http    *http.Client  // HTTP transport
    exec    executor      // Async operation executor
}
```

### Direct Method Pattern

The SDK uses a direct method pattern instead of resource namespaces:

```go
// Clean, idiomatic Go API
client.CreateMemory(ctx, userID, vaultID, req)
client.AddEntry(ctx, userID, vaultID, memID, entry)
client.Search(ctx, searchReq)

// Instead of namespaced resources (old pattern)
// client.Memories.Create(...)
// client.Entries.Add(...)
```

### Async Executor

Critical operations that require ordering use an async executor:

```go
type executor interface {
    Submit(context.Context, string, shardqueue.Job) error
    Stop()
}
```

**Async Operations** (return `*EnqueueAck`):
- `AddEntry` - Add content to memory  
- `PutContext` - Upload context snapshot
- `DeleteEntry` - Remove entry (with consistency guarantee)
- `DeleteContext` - Remove context (with consistency guarantee)

**Sync Operations** (return results immediately):
- All read operations (`Get*`, `List*`, `Search`)
- All admin operations (`Create*`, `Delete*` for users/vaults/memories)

## API Surface

### User Management
```go
CreateUser(ctx, req) (*User, error)
GetUser(ctx, userID) (*User, error)  
DeleteUser(ctx, userID) error
```

### Vault Operations
```go
CreateVault(ctx, userID, req) (*Vault, error)
ListVaults(ctx, userID) ([]Vault, error)
GetVault(ctx, userID, vaultID) (*Vault, error)
GetVaultByTitle(ctx, userID, title) (*Vault, error)
DeleteVault(ctx, userID, vaultID) error
```

### Memory Operations
```go
CreateMemory(ctx, userID, vaultID, req) (*Memory, error)
ListMemories(ctx, userID, vaultID) ([]Memory, error)
GetMemory(ctx, userID, vaultID, memoryID) (*Memory, error)
DeleteMemory(ctx, userID, vaultID, memoryID) error
```

### Entry Operations
```go
AddEntry(ctx, userID, vaultID, memID, req) (*EnqueueAck, error)     // Async
ListEntries(ctx, userID, vaultID, memID, params) (*ListEntriesResponse, error)
GetEntry(ctx, userID, vaultID, memID, entryID) (*Entry, error)
DeleteEntry(ctx, userID, vaultID, memID, entryID) error             // Async
```

### Context Management  
```go
PutContext(ctx, userID, vaultID, memID, req) (*EnqueueAck, error)   // Async
GetContext(ctx, userID, vaultID, memID) (*GetContextResponse, error)
DeleteContext(ctx, userID, vaultID, memID, contextID) error         // Async
```

### Search & Consistency
```go
Search(ctx, req) (*SearchResponse, error)
AwaitConsistency(ctx, memoryID) error                               // Wait for async ops
```

### Prompt Management
```go
LoadDefaultPrompts(ctx, memoryType) (*DefaultPromptResponse, error)
```

## Concurrency Model

The SDK implements a **three-class concurrency model** for optimal performance and consistency:

| Class | Behavior | Example Operations |
|-------|----------|-------------------|
| **Async Ordered** | FIFO per memory, immediate acknowledgment | `AddEntry`, `PutContext`, `DeleteEntry` |
| **Direct Reads** | Immediate response, eventual consistency | `Get*`, `List*`, `Search` |
| **Admin Strong** | Backend-enforced consistency | User/vault/memory CRUD |

**Key Pattern**: Async operations return `*EnqueueAck` immediately, use `AwaitConsistency(memoryID)` when read-after-write guarantees are needed.

**For detailed concurrency design**, see `docs/designs/client-api-concurrency.md`.

## Type System

### Public Type Aliases

The SDK re-exports all types from `client/internal/types` so consumers only import the `client` package:

```go
// Users only need: import "github.com/mycelian/mycelian-memory/client"

type CreateMemoryRequest = types.CreateMemoryRequest
type Memory = types.Memory
type Entry = types.Entry
type EnqueueAck = types.EnqueueAck
// ... etc
```

### Domain Entities

**Core Types**:
- `User` - User account with ID, email, display name
- `Vault` - Memory container with title, description  
- `Memory` - Individual memory with metadata and type
- `Entry` - Content entries with text, metadata, timestamps

**Request/Response Types**:
- `CreateMemoryRequest`, `AddEntryRequest`, `SearchRequest`
- `EnqueueAck`, `ListEntriesResponse`, `SearchResponse`

## Configuration

### Client Creation

```go
// Basic client
client := client.New("http://localhost:11545")

// With options
client := client.New("http://localhost:11545",
    client.WithHTTPClient(customHTTP),
    client.WithDebugLogging(true),
    client.WithShardQueueConfig(cfg),
)
```

### Environment Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `MYCELIAN_DEBUG` | `false` | Enable debug HTTP logging |
| `SQ_SHARDS` | `4` | Number of shard workers |
| `SQ_QUEUE_SIZE` | `1000` | Queue buffer per shard |
| `SQ_ENQUEUE_TIMEOUT` | `100ms` | Queue full timeout |

### Functional Options

```go
WithHTTPClient(*http.Client)              // Custom HTTP client
WithDebugLogging(bool)                    // Enable request/response logging  
WithShardQueueConfig(shardqueue.Config)   // Custom async execution config
```

## Error Handling

### Error Types

```go
var ErrNotFound = errors.New("resource not found")
var ErrValidation = errors.New("validation failed")

// Async operation errors
var ErrExecutorClosed = errors.New("shard executor closed")
var ErrQueueFull = errors.New("shard queue full")
```

### Error Patterns

**Sync Operations**:
- Return errors immediately
- HTTP status codes mapped to Go errors
- Validation errors include field-specific messages

**Async Operations**:
- Enqueue errors (queue full, validation) returned immediately
- Execution errors handled by retry mechanism or callbacks
- Use `AwaitConsistency()` to surface async execution errors

## Usage Examples

### Basic CRUD Operations

```go
package main

import (
    "context"
    "github.com/mycelian/mycelian-memory/client"
)

func main() {
    ctx := context.Background()
    client := client.New("http://localhost:11545")
    defer client.Close()

    // Create user and vault
    user, _ := client.CreateUser(ctx, client.CreateUserRequest{
        Email: "user@example.com",
    })
    
    vault, _ := client.CreateVault(ctx, user.ID, client.CreateVaultRequest{
        Title: "My Vault",
    })
    
    // Create memory and add entries
    memory, _ := client.CreateMemory(ctx, user.ID, vault.VaultID, client.CreateMemoryRequest{
        Title: "Project Notes",
        Type:  "code",
    })
    
    // Add entries asynchronously
    ack1, _ := client.AddEntry(ctx, user.ID, vault.VaultID, memory.ID, client.AddEntryRequest{
        Text: "Initial project setup complete",
    })
    
    ack2, _ := client.AddEntry(ctx, user.ID, vault.VaultID, memory.ID, client.AddEntryRequest{
        Text: "Added authentication module",
    })
    
    // Wait for consistency before reading
    client.AwaitConsistency(ctx, memory.ID)
    
    // Search entries
    results, _ := client.Search(ctx, client.SearchRequest{
        UserID:  user.ID,
        VaultID: vault.VaultID,
        Query:   "authentication",
    })
}
```

### MCP Server Integration

```go
// MCP handlers use the SDK directly
func (h *Handler) CreateMemory(ctx context.Context, req CreateMemoryRequest) (*Memory, error) {
    return h.client.CreateMemory(ctx, req.UserID, req.VaultID, client.CreateMemoryRequest{
        Title:       req.Title,
        Description: req.Description,
        Type:        req.Type,
    })
}

func (h *Handler) AddEntry(ctx context.Context, req AddEntryRequest) (*EnqueueAck, error) {
    return h.client.AddEntry(ctx, req.UserID, req.VaultID, req.MemoryID, client.AddEntryRequest{
        Text:     req.Text,
        Metadata: req.Metadata,
    })
}
```

## Implementation Notes

- **Stateless client** - No local state beyond configuration and executor
- **Memory-safe** - Proper cleanup via `Close()` method
- **Context-aware** - All operations accept `context.Context` for cancellation
- **HTTP transport** - Built on Go's standard `net/http` package
- **Retry logic** - Automatic retry with exponential backoff for transient failures
- **Metrics ready** - ShardQueue exposes Prometheus metrics for observability

## Migration from Legacy Client

The SDK replaced `pkg/memoryclient` with these improvements:

| Legacy Pattern | New SDK Pattern | Benefit |
|----------------|-----------------|---------|
| `memoryclient.CreateMemory()` | `client.CreateMemory()` | Simpler imports |
| No retry logic | Automatic retries | Reliability |
| No ordering guarantees | ShardQueue + `AwaitConsistency()` | Consistency |
| Scattered types | Consolidated in `client` package | Type safety |
| Manual HTTP handling | Abstracted away | Developer experience |

## References

- **Concurrency Design**: `docs/designs/client-api-concurrency.md`
- **ShardQueue Specification**: `docs/specs/shardqueue.md`  
- **Memory Service API**: `docs/api-reference.md`
- **MCP Integration**: `docs/designs/mcp-server.md`
