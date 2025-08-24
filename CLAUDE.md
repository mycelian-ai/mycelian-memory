# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

### Quick Setup
```bash
# Activate Python environment before any Python operations
source venv/bin/activate && export PATH="$(go env GOBIN 2>/dev/null || go env GOPATH)/bin:$PATH"
```

### Building
```bash
# Build all binaries to bin/ directory
make build                    

# Build individual components
make build-mycelian-cli       # → bin/mycelianCli
make build-mcp-server         # → bin/mycelian-mcp-server

# Build server components
cd server && make build       # Build all Go modules
```

### Testing
```bash
# Run tests (always with race detector)
go test -race ./...

# Code quality checks - run before committing
go fmt ./... && go vet ./... && go test -race ./... && go mod tidy && go build ./...
```

### Service Management
```bash
# Postgres backend (local development)
make start-dev-mycelian-server     # Start Postgres + Weaviate stack
make backend-down           # Stop all services
make backend-status         # Show service status

# MCP Server (for Claude integration)
make start-mcp-streamable-server      # Start MCP server on streamable HTTP
```

## Architecture Overview

### Multi-Module Monorepo
This is a Go workspace-coordinated monorepo with independent modules:

```
mycelian-memory/
├── go.work                    # Workspace coordinator
├── clients/go/               # Client SDK (independent module)
├── server/                   # Backend service (independent module)  
├── tools/mycelianCli/        # CLI tool (independent module)
└── tools/                    # Helper tools
```

**Benefits:** Independent versioning, clear dependency boundaries, isolated testing.

### Core Data Model
```
Vault (tenant) → Memory (context) → Entry (data)
```

### Service Architecture
```
Agent (Claude) → MCP Protocol → mycelian-mcp-server → Go Client SDK → Memory Service API
                                                                   ↓
                                                               Storage Layer
                                                            (Postgres + Weaviate)
```

### Key Components
1. **Memory Service API** (server/): RESTful backend on port 11545
2. **Go Client SDK** (clients/go/): Type-safe client library  
3. **MCP Server** (clients/go/cmd/mycelian-mcp-server/): Model Context Protocol server
4. **CLI Tools**: mycelianCli for management, mycelian-service-tools for operations
5. **Benchmarking** (tools/benchmarker/): Python-based performance testing

## Development Patterns

### Local Development User
The system auto-creates a default user for development:
- **User ID**: `local_user`
- **Email**: `dev@localhost` 
- **Display Name**: `Local Developer`

### Storage Backends
- **Postgres**: Primary relational storage (`DB_ENGINE=postgres`)
- **Weaviate**: Vector search (port 8082)

### Multi-Agent Memory Support
- **Shared memories**: For agent collaboration
- **Private memories**: For individual agent workspaces
- **Session-based conflict detection**
- **Append-only audit trail**

## Testing Strategy

### Test Types
- **Unit tests**: `go test ./...` (per module)
- **Integration tests**: End-to-end with real services
- **Cookbook**: `./tools/mycelian-service-tools/cookbook/memoryctl-simple-scenario.sh`
- **Benchmarking**: Python harness with MSC dataset
- **Schema validation**: Live MCP tools schema testing

### Critical Path Testing
Before any commit touching **Critical** code (HTTP handlers, transaction logic, cursor math):
- Run full quality gate: `go fmt ./... && go vet ./... && go test -race ./... && go mod tidy && go build ./...`
- Add property-based or fuzz tests for new Critical logic
- Include Risk Paragraph in PR: "What can go wrong? How is it prevented?"

## Code Quality Requirements

### Commit Standards
- **Format**: `type(scope): subject` (≤50 chars, imperative)
- **Types**: feat, fix, docs, style, refactor, test, chore
- **Examples**: 
  - `feat(memory): add context validation`
  - `fix(storage): handle nil pointer in postgres client`

### Pre-commit Checklist
1. `go fmt ./...` - Format code
2. `go vet ./...` - Linter checks  
3. `go test -race ./...` - Tests with race detector
4. `go mod tidy` - Clean dependencies
5. `go build ./...` - Verify builds

### Security Standards
- **No hardcoded credentials**: Environment-based configuration only
- **Input validation**: Request/response schema enforcement
- **Least privilege**: Service-specific access controls
- **Audit logging**: Comprehensive operation tracking

## Key Operational Commands

### Health Checks
```bash
# Service health
curl http://localhost:11545/health

# Weaviate health  
curl http://localhost:8082/v1/.well-known/ready

# All services
./scripts/docker-setup/docker-health-check.sh
```

### Database Management
```bash
# Start Postgres backend (includes schema initialization)
make start-dev-mycelian-server

# Postgres inspection
psql postgresql://user:password@localhost:5432/mycelian_memory
```

### Live Schema Management
- **MCP Tools Schema**: Generate live from server with `mycelianCli get-tools-schema`
- **No static JSON files**: Eliminates maintenance debt
- **Single source of truth**: MCP server registration drives schema

## Development Workflow

1. **Plan → Act**: Start with bullet-list plan, execute immediately
2. **One feature per branch**: Fix root causes, not symptoms  
3. **Pre-commit validation**: Run full quality gate
4. **Stage all changes**: `git add -A` unless explicitly excluded
5. **Error handling**: After 2 consecutive failures, pause and ask guidance

## Claude Code Integration

### External Memory Access
Claude Code has access to this repository's memory system through:
- **Vault**: `claude-code-memory` (d24a5f5b-40b4-452f-93c7-29dab0f13f24)
- **Memory**: `project-context` (e319d8e5-16ce-4628-90cb-02a47e97939a)
- **Storage**: Project architecture, development workflows, and session context

### MCP Server Configuration
Add the local MCP server to Claude Code:
```bash
claude mcp add mycelian -- ./bin/mycelian-mcp-server
```

Claude can store and retrieve context across sessions, maintaining continuity of project understanding and development patterns.

## Important Notes

- **Python Environment**: Always activate venv and set PATH before Python operations
- **Build Targets**: All binaries go to `bin/` directory for deterministic paths
- **Docker Compose**: Postgres backend with Weaviate vector search
- **MCP Integration**: Streamable HTTP for Cursor, standard JSON-RPC for other clients
- **Workspace Coordination**: Use `go work` commands for cross-module operations