# Technical Context

## Technology Stack

### Backend Services
- **Language**: Go 1.24.5
- **HTTP Framework**: Gorilla Mux
- **Database**: PostgreSQL (local dev), Aurora Serverless V2 (AWS prod)
- **Vector Search**: Weaviate vs OpenSearch (evaluating for AWS)
- **Logging**: Zerolog
- **Testing**: Go's built-in testing + Testcontainers

### Client SDKs
- **Go**: Primary SDK with full feature support
- **Python**: Benchmarking and testing tools
- **Future**: TypeScript, Rust planned

### Development Tools
- **Build System**: Make + Go workspace
- **CLI Tools**: mycelianCli (management), mycelian-service-tools (operations)
- **Schema Management**: Live generation from Go structs
- **Containerization**: Docker + Docker Compose

### External Integrations
- **MCP Protocol**: Model Context Protocol for AI tool integration
- **AWS Aurora Serverless V2**: Production-scale managed PostgreSQL
- **Vector Search**: Weaviate or OpenSearch (evaluating for AWS compatibility)

## Development Setup

### Local Environment
```bash
# Core requirements
Go 1.24.6+
Docker Desktop
Make
jq (JSON processing)

# Build all components
make build-all

# Run local stack (PostgreSQL)
docker run -d --name postgres -e POSTGRES_PASSWORD=password -p 5432:5432 postgres:15
```

### Project Structure
```
mycelian-memory/
├── go.work                    # Go workspace coordination
├── clients/go/               # Go Client SDK
├── server/                   # Backend services
├── tools/                    # CLI tools and utilities
├── deployments/              # Docker configurations
└── docs/                     # Documentation and ADRs
```

### Build Targets
- `build-mycelian-cli`: CLI management tool → `bin/mycelianCli`
- `build-mcp-server`: MCP protocol server → `bin/mycelian-mcp-server`
- `build-all`: Build all binaries with deterministic paths

### Configuration Management
- **Environment-based**: No hardcoded values
- **Service URLs**: Configurable endpoints
- **Database**: PostgreSQL-focused (local dev and AWS prod)
- **Debug modes**: `MYCELIAN_DEBUG=true` for verbose logging

## Dependencies Management

### Go Modules Strategy
- **Independent modules**: Each component has its own `go.mod`
- **Workspace coordination**: Root `go.work` file manages all modules
- **Replace directives**: Local module development support
- **Dependency isolation**: Tools don't pull unnecessary backend deps

### Key Dependencies
- **MCP Go**: `github.com/mark3labs/mcp-go` - MCP protocol implementation
- **UUID**: `github.com/google/uuid` - ID generation
- **Cobra**: `github.com/spf13/cobra` - CLI framework
- **Zerolog**: `github.com/rs/zerolog` - Structured logging
- **Testcontainers**: Integration testing with real services

### Python Integration
- **Benchmarker**: Performance and accuracy testing
- **Schema Loading**: Dynamic tool schema from live MCP server
- **Binary Discovery**: Smart path resolution for CLI tools

## Deployment Patterns

### Local Development
- **PostgreSQL**: Vanilla Docker postgres:15 image
- **Docker Compose**: Multi-service orchestration
- **Hot reload**: File watching for development

### AWS Production (Beta Stack)
- **Aurora Serverless V2**: Managed PostgreSQL with auto-scaling
- **Manual Deployment**: Beta phase only, CI automation before public launch
- **Vector Search**: Weaviate or OpenSearch (to be evaluated)
- **Monitoring**: CloudWatch integration

### CI/CD Approach
- **Multi-module builds**: Independent component versioning
- **Containerized testing**: Consistent environment across platforms
- **Deterministic outputs**: Reproducible builds with fixed paths