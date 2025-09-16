# MCP Server

**Type**: Component Documentation
**Status**: Active

## Overview

The Mycelian MCP (Model Context Protocol) server provides JSON-RPC interfaces via stdio and streamable HTTP for AI agents to interact with the Memory Service. It acts as a bridge between AI assistants and the backend memory storage system.

The server supports dual transport modes thanks to the excellent [mcpgo](https://github.com/mark3labs/mcpgo) project:
- **JSON stdio** for Claude Desktop and compatible clients
- **Streamable HTTP** for Cursor and web-based integrations

## Architecture

```
AI Agent (Claude, Cursor, etc.)
    ↓ (JSON-RPC stdio or HTTP)
MCP Server
    ↓ (HTTP/REST)
Memory Service Backend
```

The MCP server runs as a local process that:
- Receives JSON-RPC requests via stdio or HTTP from AI agents
- Translates them to HTTP REST calls to the Memory Service
- Handles authentication, context management, and error translation
- Provides memory operations, search, and prompt management

## Core Components

### Server (`mcp/server.go`)
- **Configuration**: Environment-based config with sensible defaults
- **Graceful shutdown**: Handles signals and drains connections
- **HTTP client**: Configurable timeouts for backend communication
- **Logging**: Structured logging with configurable levels

### Handlers (`mcp/internal/handlers/`)
- **Memory operations**: Create, list, delete memories
- **Entry management**: Add entries, search, list with filters
- **Context handling**: Upload/download context snapshots
- **Vault operations**: Organize memories in vaults
- **User management**: Create and manage user accounts
- **Prompt management**: Load and customize prompts
- **Consistency control**: Explicit consistency guarantees

## Key Features

### Memory Operations
- **Create memories** with metadata (title, description, type)
- **Add entries** asynchronously with immediate acknowledgment
- **Search entries** using hybrid vector + keyword search
- **Context snapshots** for maintaining conversation state

### Consistency Model
- **Asynchronous writes** for performance
- **Explicit consistency** via `await_consistency` tool
- **Per-memory ordering** through sharded execution
- **Read-after-write** guarantees when needed

### Authentication & Security
- **User-scoped operations** with required user IDs
- **Vault isolation** for organizing memories
- **Input validation** on all parameters
- **Error sanitization** to avoid leaking internals

## Configuration

The server uses environment variables with command-line flag overrides:

| Variable | Default | Description |
|----------|---------|-------------|
| `MEMORY_SERVICE_URL` | `http://localhost:11545` | Backend service endpoint |
| `CONTEXT_DATA_DIR` | `./data/context` | Local context storage |
| `LOG_LEVEL` | `info` | Logging verbosity |
| `MCP_SERVER_NAME` | `mycelian-mcp-server` | Server identification |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout |
| `MCP_STDIO` | `false` | Enable stdio transport mode for Claude |

## Tool Categories

### Memory Management
- `create_memory` - Create new memory with metadata
- `list_memories` - List memories for a user/vault
- `delete_memory` - Remove memory and all entries

### Entry Operations
- `add_entry` - Add content to memory (async)
- `search_entries` - Search across memories
- `list_entries` - Get entries with optional filters
- `delete_entry` - Remove specific entry

### Context Management
- `put_context` - Upload context snapshot
- `get_context` - Download context data
- `delete_context` - Remove context snapshot

### Consistency Control
- `await_consistency` - Wait for all pending writes to complete

### Organizational
- `create_vault` - Create memory container
- `list_vaults` - List available vaults
- `delete_vault` - Remove vault and contents

## Error Handling

The server provides clean error translation:
- **Validation errors** → Clear parameter feedback
- **Backend errors** → Sanitized user messages
- **Network errors** → Retry guidance
- **Rate limiting** → Backoff suggestions

## Usage Examples

### Claude Desktop (Docker)

```json
{
  "mcpServers": {
    "mycelian-memory": {
      "command": "docker",
      "args": [
        "run",
        "--rm",
        "-i",
        "--network",
        "host",
        "-e",
        "MEMORY_SERVICE_URL=http://localhost:11545",
        "-e",
        "MCP_STDIO=true",
        "mycelian-mcp-server:dev"
      ]
    }
  }
}
```

### Claude Desktop (Binary)

```json
{
  "mcpServers": {
    "mycelian-memory": {
      "command": "/path/to/mycelian-mcp-server",
      "env": {
        "MEMORY_SERVICE_URL": "http://localhost:11545",
        "MCP_STDIO": "true"
      }
    }
  }
}
```

### Cursor (Streamable HTTP)

```bash
# Start server in HTTP mode
./mycelian-mcp-server

# Configure in Cursor settings
# The server automatically provides streamable HTTP on port 3001
```

## Development

### Building
```bash
cd cmd/mycelian-mcp-server
go build -o mycelian-mcp-server .
```

### Testing
```bash
cd mcp
go test ./...
```

### Integration Testing
```bash
# Start backend first
make start-dev-mycelian-server

# Run MCP integration tests
cd mcp
go test -tags=integration ./...
```

## Implementation Notes

- **Stateless design** - No local state beyond configuration
- **Client SDK integration** - Uses the Go client SDK for all backend operations
- **Dual transport support** - Built on mcpgo for stdio and streamable HTTP modes
- **Concurrent-safe** - Handles multiple simultaneous requests
- **Resource-bounded** - Configurable timeouts prevent resource leaks
- **Standards-compliant** - Follows MCP JSON-RPC specification

## References

- **Client SDK**: `docs/designs/client-sdk.md`
- **Memory Service API**: `docs/api-reference.md`
- **MCP Specification**: https://modelcontextprotocol.io/
