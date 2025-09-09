# Client SDK

**Type**: Component Documentation
**Status**: Stable

## Overview

The Mycelian Go Client SDK provides a type-safe, idiomatic Go interface for interacting with the Memory Service backend. It serves as the primary library for applications, tools, and the MCP server to perform memory operations.

This SDK provides an idiomatic Go interface to the Memory Service with a focus on simplicity, ordering guarantees, and clear error handling.

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
// Clean, idiomatic Go API (no user ID at call sites)
client.CreateMemory(ctx, vaultID, req)
client.AddEntry(ctx, vaultID, memID, entry)
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
    Barrier(context.Context, string) error
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
User management is now external to the service and not provided by this SDK.

### Vault Operations
```go
CreateVault(ctx, req) (*Vault, error)
ListVaults(ctx) ([]Vault, error)
GetVault(ctx, vaultID) (*Vault, error)
GetVaultByTitle(ctx, title) (*Vault, error)
DeleteVault(ctx, vaultID) error
```

### Memory Operations
```go
CreateMemory(ctx, vaultID, req) (*Memory, error)
ListMemories(ctx, vaultID) ([]Memory, error)
GetMemory(ctx, vaultID, memoryID) (*Memory, error)
DeleteMemory(ctx, vaultID, memoryID) error
```

### Entry Operations
```go
AddEntry(ctx, vaultID, memID, req) (*EnqueueAck, error) // Async
ListEntries(ctx, vaultID, memID, params) (*ListEntriesResponse, error)
GetEntry(ctx, vaultID, memID, entryID) (*Entry, error)
DeleteEntry(ctx, vaultID, memID, entryID) error         // Sync; awaits prior writes before HTTP delete
```

### Context Management
```go
PutContext(ctx, vaultID, memID, req) (*EnqueueAck, error)   // Async
GetContext(ctx, vaultID, memID) (*GetContextResponse, error)
DeleteContext(ctx, vaultID, memID, contextID) error         // Sync; awaits prior writes before HTTP delete
```

### Search & Consistency
```go
Search(ctx, req) (*SearchResponse, error)
AwaitConsistency(ctx, memoryID) error                               // Wait for async ops
```

### Prompt Management
```go
// Reads embedded defaults locally; no network call
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
// Basic client (error-returning constructor)
c, err := client.New("http://localhost:11545", "<api-key>")
if err != nil { /* handle */ }

// With options
c, err := client.New(
    "http://localhost:11545",
    "<api-key>",
    client.WithHTTPTimeout(10*time.Second),
    client.WithDebugLogging(true),
)
if err != nil { /* handle */ }
```

### Environment Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `MYCELIAN_DEBUG` | `false` | Enable debug HTTP logging |

### Functional Options

```go
WithHTTPTimeout(time.Duration)  // Set HTTP timeout
WithDebugLogging(bool)          // Enable request/response logging
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
    c, _ := client.New("http://localhost:11545", "<api-key>")
    defer c.Close()

    // Create vault
    vault, _ := c.CreateVault(ctx, client.CreateVaultRequest{Title: "My Vault"})

    // Create memory and add entries
    memory, _ := c.CreateMemory(ctx, vault.VaultID, client.CreateMemoryRequest{
        Title:      "Project Notes",
        MemoryType: "code",
    })

    // Add entries asynchronously
    _, _ = c.AddEntry(ctx, vault.VaultID, memory.ID, client.AddEntryRequest{
        RawEntry: "Initial project setup complete",
    })
    _, _ = c.AddEntry(ctx, vault.VaultID, memory.ID, client.AddEntryRequest{
        RawEntry: "Added authentication module",
    })

    // Wait for consistency before reading
    _ = c.AwaitConsistency(ctx, memory.ID)

    // Search entries
    results, _ := c.Search(ctx, client.SearchRequest{
        MemoryID: memory.ID,
        Query:    "authentication",
    })
    _ = results
}
```

### MCP Server Integration

```go
// MCP handlers use the SDK directly
func (h *Handler) CreateMemory(ctx context.Context, req CreateMemoryRequest) (*Memory, error) {
    return h.client.CreateMemory(ctx, req.VaultID, client.CreateMemoryRequest{
        Title:       req.Title,
        Description: req.Description,
        MemoryType:  req.Type,
    })
}

func (h *Handler) AddEntry(ctx context.Context, req AddEntryRequest) (*EnqueueAck, error) {
    return h.client.AddEntry(ctx, req.VaultID, req.MemoryID, client.AddEntryRequest{
        RawEntry: req.Text,
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

## Notes

- The client is stateless aside from its executor and HTTP configuration.
- Async write APIs return quickly and preserve FIFO per memory; call `AwaitConsistency` for read-after-write semantics.

## References

- **Concurrency Design**: `docs/designs/client-api-concurrency.md`
- **ShardQueue Specification**: `docs/specs/shardqueue.md`
- **Memory Service API**: `docs/api-reference.md`
- **MCP Integration**: `docs/designs/mcp-server.md`
