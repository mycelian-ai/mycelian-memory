# System Patterns & Architecture

## Simplified SaaS Architecture
Focusing on single PostgreSQL dialect for simplicity and customer focus:

```
mycelian-memory/
├── go.work                    # Workspace coordinator
├── client/                   # Client SDK module
├── server/                   # Backend service module  
├── tools/mycelianCli/        # CLI tool module
├── deployments/aws/          # AWS deployment package (new)
└── tools/                    # Helper tools
```

**Benefits:**
- Simplified database layer (PostgreSQL only)
- AWS-optimized deployment
- Faster development cycle
- Clear path to paying customers

## MCP Server Architecture
```
Agent (Claude) → MCP Protocol → mycelian-mcp-server → Go Client SDK → Memory Service
```

**Key Decisions:**
- MCP server acts as thin protocol adapter
- All business logic lives in Go Client SDK
- ✅ **SDK provides simplified direct method API**: `client.Method()` pattern (2025-08-06 refactor)
- Live schema generation from Go struct definitions (eliminates static JSON maintenance)
- Async operations (entries, contexts) preserve FIFO ordering via internal executor

## Memory Model
```
Vault (tenant) → Memory (context) → Entry (data)
```

**Multi-Agent Support:**
- Shared memories for collaboration
- Private memories for agent workspaces  
- Session-based conflict detection
- Append-only audit trail

## Build System Patterns
- **Deterministic paths**: All binaries → `bin/` directory
- **Make targets**: `build-mycelian-cli`, `build-mcp-server`, `build-all`
- **Cross-module builds**: Use workspace-aware commands
- **CI/CD ready**: Containerized builds with proper dependency management

## Schema Management Pattern
**Evolution from Static → Live:**
- **Before**: Static JSON schema files (`tools.schema.json`) required manual sync
- **After**: Live schema generation from MCP server registration
- **Implementation**: `mycelianCli get-tools-schema` queries live MCP server
- **Benefits**: Single source of truth, eliminates maintenance debt

## Testing Patterns
- **Unit tests**: Per-module isolation with direct client method patterns
- **Integration tests**: 
  - **Many tests**: Organized in `integration_test/mock/` and `integration_test/real/` directories
  - **Few tests**: Single `_integration_test.go` file in package itself
- **Mock integration**: Testing with `httptest.Server` and mocked dependencies
- **Real integration**: End-to-end tests with actual services and databases
- **Benchmarking**: Python harness with MSC dataset evaluation
- **Schema validation**: Live MCP tools schema testing
- **Transport testing**: MCP server HTTP and stdio transport validation

## Error Handling Patterns
- **Go SDK**: Structured errors with context
- **MCP Server**: Standard JSON-RPC error responses
- **CLI Tools**: User-friendly error messages with action suggestions
- **Python Integration**: Graceful fallbacks with clear failure paths

## Security Patterns
- **No hardcoded credentials**: Environment-based configuration
- **Least privilege**: Service-specific access controls
- **Input validation**: Request/response schema enforcement
- **Audit logging**: Comprehensive operation tracking