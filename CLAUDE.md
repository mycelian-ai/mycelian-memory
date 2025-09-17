# Mycelian Memory - AI Assistant Context

YOU MUST NEVER praise me or support my ideas blindly.
YOU MUST remain unbiased in your feedback.

## Project Overview
**Mycelian Memory** is an open-source memory framework providing long-term memory and context to AI agents through a log-structured architecture. Licensed under Apache v2.

**GitHub**: https://github.com/mycelian-ai/mycelian-memory
**Status**: Early-stage, active development
**Author**: @sam33rch (built using AI development tools: Cursor, Claude Code, Kiro)

## Communication Preferences
When working with @sam33rch, use professional Staff Engineer communication style:

- **Professional tone**: Direct technical analysis without validation or praise language
- **Concise writing**: Apply "On Writing Well" principles - clear, specific, actionable
- **No flowery language**: Avoid phrases like "brilliant insight," "you're absolutely right," or similar validations
- **Focus on substance**: Lead with key findings, use concrete examples, eliminate hedging
- **Staff Engineer perspective**: Communicate as an experienced peer, not a subordinate seeking approval

## Architecture

### Core Concept
- **Log-structured memory**: Append-only entry logs + evolving context snapshots
- **No lossy summarization**: Direct agent context offloading during conversations
- **Vault-based organization**: Memories scoped by vaults containing purpose-specific collections
- **Hybrid search**: Combines vector embeddings with keyword search

### System Components
```
AI Agent <-> MCP Server <-> Memory Service <-> Postgres + Vector DB
                                             <-> Outbox Worker
```

### Key Tables
- `vaults`: Organization containers
- `memories`: Individual memory stores
- `entries`: Immutable log entries
- `context`: Evolving context snapshots
- `tx_outbox`: Async job processing

## Repository Structure

### Go Modules (Workspace)
- `/server` - HTTP API service, core backend logic
- `/client` - Go SDK for API interaction
- `/mcp` - Model Context Protocol server
- `/cmd/memory-service` - Main service entry point
- `/cmd/mycelian-mcp-server` - MCP server binary
- `/cmd/outbox-worker` - Async worker process
- `/tools/mycelianCli` - CLI for testing/debugging
- `/tools/mycelian-service-tools` - Service management tools
- `/tools/invariants-checker` - Data consistency checker
- `/tools/dep-update` - Dependency update utilities
- `/pkg/devauth` - Development authentication

### Other Components
- `/longmemeval-benchmarker` - LangGraph-based LongMemEval benchmark implementation
- `/tools/longmemeval-benchmarker` - Legacy Python benchmarker (deprecated)
- `/deployments/docker` - Docker compose configurations
- `/docs` - Architecture, ADRs, API documentation
- `/data` - Test data and datasets

## Development Setup

### Prerequisites
- Go 1.23+
- Docker Desktop
- Ollama (with nomic-embed-text model)
- Make, jq
- Python 3.11+ (for benchmarker)

### Quick Start
```bash
# Start Ollama
ollama serve &
ollama pull nomic-embed-text

# Start backend stack
make start-dev-mycelian-server

# Verify health
curl -s http://localhost:11545/v0/health | jq
```

### MCP Configuration
- **Cursor**: Streamable HTTP mode on port 11546
- **Claude Desktop**: stdio mode via binary

## Key Commands

### Build
- `make build` - Build all binaries to bin/
- `make build-check` - Verify compilation
- `make quality-check` - Full quality gate (fmt, vet, test, lint, vuln)

### Test
- `make test-full-local-stack` - Complete test suite with backend
- `make server-test` - Server unit tests
- `make client-test` - Client tests
- `make client-test-integration` - Integration tests

### Service Management
- `make start-dev-mycelian-server` - Start Postgres backend
- `make backend-down` - Stop services
- `make backend-status` - Check status
- `make backend-logs` - View logs

## API Overview

Base URL: `http://localhost:11545/v0`

### Core Operations
- Create vault: `POST /vaults`
- Create memory: `POST /vaults/{vaultId}/memories`
- Store context: `PUT /vaults/{vaultId}/memories/{memoryId}/contexts`
- Add entry: `POST /vaults/{vaultId}/memories/{memoryId}/entries`
- Search: `POST /search`

### Authentication
Dev mode: `Authorization: Bearer LOCAL_DEV_MODE_NOT_FOR_PRODUCTION`

## Configuration

Environment variables use `MEMORY_SERVER_` prefix:
- `MEMORY_SERVER_HTTP_PORT` (default: 11545)
- `MEMORY_SERVER_DEV_MODE` (true/false)
- `MEMORY_SERVER_POSTGRES_DSN`
- `MEMORY_SERVER_SEARCH_INDEX_URL` (Weaviate)
- `MEMORY_SERVER_EMBED_PROVIDER` (ollama)
- `MEMORY_SERVER_MAX_CONTEXT_CHARS` (65536)

## Key Technologies
- **Language**: Go (server), Python (benchmarker)
- **Database**: PostgreSQL (primary), Weaviate (vector search)
- **Embeddings**: Ollama with nomic-embed-text
- **Protocol**: MCP (Model Context Protocol)
- **Architecture**: Monorepo with Go workspaces

## Development Philosophy
- Simple over complex
- User success over benchmarks
- Production over demos
- Correctness over speed
- Transparency over magic
- Memory quality = accuracy/completeness (precision/recall); Performance = speed/throughput

## Testing Strategy
- Unit tests per module
- Integration tests with real backend
- E2E tests against Docker stack
- Performance benchmarking with MemGPT dataset
- Coverage target: 78%+ for client

## Current Focus Areas

### Primary Focus: Code Reliability Improvement
Performing thorough code review and refactoring using:

1. **Clean Code Principles**
   - Clear naming and intent
   - Small, focused functions
   - DRY (Don't Repeat Yourself)
   - Single Responsibility Principle
   - Proper error handling
   - Meaningful abstractions

2. **Go Idioms**
   - Idiomatic error handling patterns
   - Proper use of interfaces
   - Effective goroutine and channel patterns
   - Context propagation
   - Proper package organization
   - Following Go proverbs and best practices

3. **Monitoring & Observability**
   - Structured logging with appropriate levels
   - Metrics collection points
   - Tracing for distributed operations
   - Health checks and readiness probes
   - Error tracking and alerting hooks
   - Performance instrumentation

### Secondary Focus: LongMemEval Benchmark
Establishing baseline memory quality metrics:

1. **Benchmark Implementation**
   - Running LongMemEval benchmark suite
   - Getting initial score for baseline
   - Identifying memory quality bottlenecks
   - Comparing with other memory systems

2. **Memory Quality Goals**
   - Precision and recall metrics (core memory quality)
   - Answer accuracy and abstention quality
   - Degradation rate analysis
   - Note: Quality = accuracy/completeness, Performance = speed/throughput

### Third Focus: Organization & Project Scoping
Implementing multi-tenant memory organization:

1. **Hierarchical Structure**
   - Organization-level isolation
   - Project-based memory grouping
   - Vault organization within projects
   - Access control and permissions

2. **Implementation Goals**
   - Multi-tenant data isolation
   - Cross-project memory sharing capabilities
   - Scoped search and retrieval
   - Resource quotas and limits per org/project

### Supporting Areas
- Core memory CRUD operations stability
- Hybrid search implementation
- Context management and sharding
- MCP server integration

## Known Limitations
- Early-stage, APIs may change
- Not production-ready
- Active refactoring (synapse -> mycelian rename in progress)

## Contribution Guidelines
- Fork and create feature branches
- Follow existing code patterns
- Run `make build-check` before commits
- Run `make test-full-local-stack` for full validation
- AI-assisted development is encouraged

## Support
- Issues: https://github.com/mycelian-ai/mycelian-memory/issues
- Discord: https://discord.gg/mEqsYcDcAj

## Notes for AI Assistants
- Codebase uses AI-friendly patterns and documentation
- Prefer editing existing files over creating new ones
- Follow Go idioms and workspace structure
- Check existing patterns in similar files
- Test commands are in Makefile
- Development uses Docker for dependencies
- **Git commits**: Use user's git config only - no AI agent as committer or co-author
- Remember my conversation preference
