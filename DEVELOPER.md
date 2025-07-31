# Developing Synapse Memory Backend

- [Developing Synapse Memory Backend](#developing-synapse-memory-backend)
  - [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [System Requirements](#system-requirements)
  - [Architecture Overview](#architecture-overview)
    - [Core Components](#core-components)
    - [Data Flow](#data-flow)
    - [Storage Backends](#storage-backends)
  - [Local Development](#local-development)
    - [Fork and Clone](#fork-and-clone)
    - [Install Dependencies](#install-dependencies)
    - [Environment Setup](#environment-setup)
    - [Running the Stack](#running-the-stack)
  - [Development Workflow](#development-workflow)
    - [Code Organization](#code-organization)
    - [Making Changes](#making-changes)
    - [Testing Your Changes](#testing-your-changes)
    - [Code Quality Standards](#code-quality-standards)
  - [Building and Running](#building-and-running)
    - [Build Commands](#build-commands)
    - [Running Services](#running-services)
    - [Docker Development](#docker-development)
  - [Testing](#testing)
    - [Test Categories](#test-categories)
    - [Running Tests](#running-tests)
    - [E2E Testing](#e2e-testing)
    - [Performance Testing](#performance-testing)
  - [Debugging](#debugging)
    - [API Debugging](#api-debugging)
    - [Database Debugging](#database-debugging)
    - [Search Debugging](#search-debugging)
    - [Common Issues](#common-issues)
  - [Contributing](#contributing)
    - [Creating Pull Requests](#creating-pull-requests)
    - [Code Review Process](#code-review-process)
    - [Documentation](#documentation)
  - [Advanced Topics](#advanced-topics)
    - [Adding New Storage Backends](#adding-new-storage-backends)
    - [Extending the API](#extending-the-api)
    - [Performance Optimization](#performance-optimization)
  - [Community](#community)

## Getting Started

Thank you for your interest in contributing to Synapse Memory Backend! This document will guide you through setting up your development environment and understanding our codebase.

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.24.3 or higher** - [Download Go](https://golang.org/dl/)
- **Docker Desktop** - [Download Docker](https://www.docker.com/products/docker-desktop)
- **Make** - Usually pre-installed on Unix systems
- **Git** - [Download Git](https://git-scm.com/downloads)
- **jq** - JSON processor for testing scripts
  ```bash
  # macOS
  brew install jq
  
  # Ubuntu/Debian
  sudo apt-get install jq
  ```

### System Requirements

- **OS**: macOS, Linux, or Windows with WSL2
- **RAM**: Minimum 8GB (16GB recommended for running full stack)
- **Disk**: 10GB free space for Docker images and data

## Architecture Overview

### Core Components

```mermaid
graph TB
    subgraph "API Layer"
        API[Memory Service API<br/>:8080]
    end
    
    subgraph "Storage Layer"
        SQLite[SQLite<br/>Local Dev]
        Spanner[Spanner<br/>Production]
    end
    
    subgraph "Search Layer"
        Weaviate[Weaviate<br/>:8082]
        Indexer[Indexer Service]
    end
    
    subgraph "CLI Tools"
        MemCtl[memoryctl]
        Schema[schema-manager]
    end
    
    API --> SQLite
    API --> Spanner
    Indexer --> SQLite
    Indexer --> Spanner
    Indexer --> Weaviate
    API --> Weaviate
    
    MemCtl --> API
    Schema --> Spanner
```

### Data Flow

1. **Write Path**: Client → API → Storage (SQLite/Spanner) → Event Bus → Indexer → Weaviate
2. **Read Path**: Client → API → Storage (for metadata) + Weaviate (for search)
3. **Search Path**: Client → API → Weaviate → Ranked Results

### Storage Backends

The system supports multiple storage backends:

| Backend | Use Case | Configuration |
|---------|----------|---------------|
| SQLite | Local development, testing | `DB_ENGINE=sqlite` |
| Spanner | Production, high scale | `DB_ENGINE=spanner` |

## Local Development

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/<your-userId>/memory-backend.git
   cd memory-backend
   ```
3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/synapse/memory-backend.git
   ```

### Install Dependencies

```bash
# Download Go modules
go mod download

# Build development tools
make build-tools

# Verify installation
go version
docker --version
make --version
```

### Environment Setup

1. Copy example environment files:
   ```bash
   cp .env.example .env
   ```

2. Key environment variables:
   ```bash
   # API Configuration
   PORT=8080
   LOG_LEVEL=debug
   
   # Database Selection
   DB_ENGINE=sqlite  # or 'spanner'
   
   # SQLite Configuration
   SQLITE_PATH=~/.synapse-memory/memory.db
   
   # Spanner Configuration (if using)
   SPANNER_PROJECT=test-project
   SPANNER_INSTANCE=test-instance
   SPANNER_DATABASE=memory-backend
   
   # Weaviate Configuration
   WEAVIATE_URL=http://localhost:8082
   ```

3. **Local Development User**: The system automatically creates a default local user for development:
   - **User ID**: `local_user`
   - **Email**: `dev@localhost`
   - **Display Name**: `Local Developer` (SQLite local) / `Local User` (Docker)
   - **Time Zone**: `UTC`
   - **Status**: `ACTIVE`

### Running the Stack

#### Quick Start (SQLite + Weaviate)

```bash
# Start all services with SQLite backend
make docker-run-sqlite

# Verify services are running
make docker-status

# View logs
docker-compose -f docker-compose.sqlite.yml logs -f
```

#### Production-like Setup (Spanner + Weaviate)

```bash
# Start Spanner emulator first
gcloud emulators spanner start

# Create schema
make schema-create-emulator

# Start all services
make docker-run-spanner

# Verify services
make docker-status
```

## Development Workflow

### Code Organization

```
memory-backend/
├── cmd/                    # Application entry points
│   ├── memory-service/     # Main API server
│   ├── memoryctl/          # CLI tool
│   └── indexer-prototype/  # Background indexer
├── internal/               # Private application code
│   ├── api/                # HTTP handlers and routing
│   ├── core/               # Business logic
│   │   ├── memory/         # Memory domain
│   │   └── vault/          # Vault domain
│   ├── storage/            # Database interfaces
│   └── search/             # Search interfaces
├── pkg/                    # Public libraries
├── scripts/                # Development scripts
├── docs/                   # Documentation
└── tools/                  # Development tools
```

### Making Changes

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Follow our coding standards**:
   - Use meaningful variable and function names
   - Add comments for complex logic
   - Keep functions small and focused
   - Write tests for new functionality

3. **Commit guidelines**:
   ```bash
   # Format: type(scope): subject
   git commit -m "feat(memory): add context validation"
   git commit -m "fix(storage): handle nil pointer in spanner client"
   git commit -m "docs(api): update search endpoint documentation"
   ```

   Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

### Testing Your Changes

Before submitting a PR, ensure:

```bash
# 1. Code is formatted
go fmt ./...

# 2. Linter passes
go vet ./...
golangci-lint run

# 3. Tests pass
go test -race ./...

# 4. Build succeeds
make build
```

### Code Quality Standards

We maintain high code quality through:

1. **Automated Checks**: CI runs on every PR
2. **Code Review**: All changes require review
3. **Test Coverage**: Aim for >80% coverage
4. **Documentation**: Update docs with code changes

## Building and Running

### Build Commands

```bash
# Build everything
make build

# Build specific components
go build -o bin/memory-service ./cmd/memory-service
go build -o bin/memoryctl ./cmd/memoryctl
go build -o bin/indexer-prototype ./cmd/indexer-prototype

# Build with specific tags
go build -tags production -o bin/memory-service ./cmd/memory-service

# Cross-compilation
GOOS=linux GOARCH=amd64 go build -o bin/memory-service-linux ./cmd/memory-service
```

### Running Services

#### Local Development (No Docker)

```bash
# Build and run with SQLite
make run-local

# Or manually with custom settings
BUILD_TARGET=local \
  DB_ENGINE=sqlite \
  SQLITE_PATH=./test.db \
  PORT=8080 \
  ./bin/memory-service
```

#### Docker Development

```bash
# Start specific services
docker-compose up -d memory-service
docker-compose up -d weaviate
docker-compose up -d indexer-prototype

# Rebuild and restart a service
docker-compose up -d --build memory-service

# View real-time logs
docker-compose logs -f memory-service

# Execute commands in container
docker-compose exec memory-service /bin/sh
```

### Docker Development

#### Building Images

```bash
# Build all images
docker-compose build

# Build specific service
docker-compose build memory-service

# Build with no cache
docker-compose build --no-cache
```

#### Docker Compose Profiles

We use different compose files for different backends:

```bash
# SQLite stack (default for development)
docker-compose -f docker-compose.sqlite.yml up -d

# Spanner stack (production-like)
docker-compose -f docker-compose.spanner.yml up -d
```

## Testing

### Test Categories

1. **Unit Tests**: Test individual functions and methods
2. **Integration Tests**: Test component interactions
3. **E2E Tests**: Test complete user workflows
4. **Invariant Tests**: Verify system constraints
5. **Performance Tests**: Measure and optimize performance

### Running Tests

```bash
# Run all tests
go test ./...

# Run with race detector (recommended)
go test -race ./...

# Run with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/core/memory/...
go test ./internal/api/...

# Run specific test
go test -run TestMemoryService_Create ./internal/core/memory/

# Verbose output
go test -v ./...

# Run benchmarks
go test -bench=. ./...
go test -bench=BenchmarkMemoryCreate ./internal/core/memory/
```

### E2E Testing

#### Using memoryctl Scripts

```bash
# Quick validation test
./scripts/memoryctl/memoryctl-simple-test.sh

# Comprehensive workflow test
./scripts/memoryctl/memoryctl-multi-entities.sh
```

#### Using Go E2E Tests

```bash
cd dev_env_e2e_tests

# Run all E2E tests
go test -v

# Run specific test suites
go test -v -run TestSmoke
go test -v -run TestSearchRelevance
go test -v -run TestConcurrency
```

### Performance Testing

```bash
# Run load tests
go test -bench=. -benchtime=10s ./internal/api/...

# Memory profiling
go test -memprofile=mem.prof ./internal/core/memory/
go tool pprof mem.prof

# CPU profiling
go test -cpuprofile=cpu.prof ./internal/core/memory/
go tool pprof cpu.prof
```

## Debugging

### API Debugging

#### Health Checks

```bash
# Service health
curl -s http://localhost:8080/health | jq .

# Weaviate health
curl -s http://localhost:8082/v1/.well-known/ready

# Check all services
./scripts/docker-setup/docker-health-check.sh
```

#### Local User Creation

The system automatically creates a default local user for development:

**SQLite Local Development**: Uses `EnsureDefaultUser()` function
- Creates user if Users table is empty
- User ID: `local_user`
- Email: `dev@localhost`
- Display Name: `Local Developer`

**Docker Compose**: Uses `user-bootstrap` container
- Waits for memory-service to be healthy
- Uses `memoryctl` CLI to create user via API
- User ID: `local_user`
- Email: `dev@localhost`
- Display Name: `Local User`

You can verify the local user exists:
```bash
# Check if local user was created
curl -s http://localhost:8080/api/users/local_user | jq .

# Expected response:
{
  "userId": "local_user",
  "email": "dev@localhost",
  "displayName": "Local Developer",
  "timeZone": "UTC",
  "status": "ACTIVE",
  "creationTime": "2025-01-27T10:30:00Z"
}
```

#### Complete Workflow Example

```bash
# Set up environment
export API="http://localhost:8080"
export EMAIL="debug-$(date +%s)@test.com"

# 1. Create user
USER_RESPONSE=$(curl -s -X POST "$API/api/users" \
  -H 'Content-Type: application/json' \
  -d '{"email":"'"$EMAIL"'","name":"Debug User"}')
echo "$USER_RESPONSE" | jq .
export USER_ID=$(echo "$USER_RESPONSE" | jq -r '.userId')

# 2. Create vault
VAULT_RESPONSE=$(curl -s -X POST "$API/api/users/$USER_ID/vaults" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Debug Vault","description":"Testing vault"}')
echo "$VAULT_RESPONSE" | jq .
export VAULT_ID=$(echo "$VAULT_RESPONSE" | jq -r '.vaultId')

# 3. Create memory
MEMORY_RESPONSE=$(curl -s -X POST "$API/api/users/$USER_ID/vaults/$VAULT_ID/memories" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Debug Memory","description":"Testing memory"}')
echo "$MEMORY_RESPONSE" | jq .
export MEMORY_ID=$(echo "$MEMORY_RESPONSE" | jq -r '.memoryId')

# 4. Add entry
ENTRY_RESPONSE=$(curl -s -X POST "$API/api/users/$USER_ID/vaults/$VAULT_ID/memories/$MEMORY_ID/entries" \
  -H 'Content-Type: application/json' \
  -d '{"rawText":"The quick brown fox jumps over the lazy dog","summary":"Test entry"}')
echo "$ENTRY_RESPONSE" | jq .
export ENTRY_ID=$(echo "$ENTRY_RESPONSE" | jq -r '.entryId')

# 5. Wait for indexing
echo "Waiting for indexer..."
sleep 3

# 6. Search
SEARCH_RESPONSE=$(curl -s -X POST "$API/api/search" \
  -H 'Content-Type: application/json' \
  -d '{"userId":"'"$USER_ID"'","memoryId":"'"$MEMORY_ID"'","query":"fox","topK":5}')
echo "$SEARCH_RESPONSE" | jq .
```

### Database Debugging

#### SQLite Debugging

```bash
# Connect to database
sqlite3 ~/.synapse-memory/memory.db

# Useful queries
.tables
.schema users
.mode column
.headers on

SELECT * FROM users ORDER BY created_at DESC LIMIT 5;
SELECT * FROM vaults WHERE user_id = 'YOUR_USER_ID';
SELECT * FROM memories WHERE vault_id = 'YOUR_VAULT_ID';
SELECT COUNT(*) as entry_count, memory_id 
  FROM entries 
  GROUP BY memory_id;
```

#### Spanner Debugging

```bash
# Using gcloud CLI
gcloud spanner databases execute-sql memory-backend \
  --instance=test-instance \
  --sql="SELECT * FROM users LIMIT 10"

# Check schema
gcloud spanner databases ddl describe memory-backend \
  --instance=test-instance
```

### Search Debugging

#### Direct Weaviate Queries

```bash
# Get schema
curl -s http://localhost:8082/v1/schema | jq .

# Search for specific entry
curl -s -X POST http://localhost:8082/v1/graphql \
  -H 'Content-Type: application/json' \
  -d '{
    "query": "{
      Get {
        MemoryEntry(
          tenant: \"'"$USER_ID"'\",
          where: {
            path: [\"entryId\"],
            operator: Equal,
            valueText: \"'"$ENTRY_ID"'\"
          }
        ) {
          entryId
          memoryId
          summary
          _additional {
            id
            vector
          }
        }
      }
    }"
  }' | jq .

# Vector search
curl -s -X POST http://localhost:8082/v1/graphql \
  -H 'Content-Type: application/json' \
  -d '{
    "query": "{
      Get {
        MemoryEntry(
          tenant: \"'"$USER_ID"'\",
          nearText: {
            concepts: [\"fox\"]
          },
          limit: 5
        ) {
          entryId
          summary
          _additional {
            distance
            certainty
          }
        }
      }
    }"
  }' | jq .
```

### Common Issues

#### Port Already in Use

```bash
# Find process using port
lsof -i :8080
# or
netstat -tulpn | grep 8080

# Kill process
kill -9 <PID>
```

#### Docker Issues

```bash
# Clean restart
docker-compose down -v
docker system prune -a --volumes

# Check container logs
docker logs memory-service
docker logs weaviate

# Inspect container
docker inspect memory-service | jq .

# Check resource usage
docker stats
```

#### Database Connection Issues

1. **SQLite**: Check file permissions
   ```bash
   ls -la ~/.synapse-memory/
   chmod 755 ~/.synapse-memory
   chmod 644 ~/.synapse-memory/memory.db
   ```

2. **Spanner**: Verify emulator is running
   ```bash
   # Check if emulator is running
   ps aux | grep spanner
   
   # Test connection
   gcloud spanner instances list --project=test-project
   ```

#### Search Not Working

1. Check Weaviate is running:
   ```bash
   curl http://localhost:8082/v1/.well-known/ready
   ```

2. Verify schema exists:
   ```bash
   curl http://localhost:8082/v1/schema | jq '.classes[].class'
   ```

3. Check indexer logs:
   ```bash
   docker-compose logs -f indexer-prototype
   ```

## Contributing

### Creating Pull Requests

1. **Update your fork**:
   ```bash
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```

2. **Create feature branch**:
   ```bash
   git checkout -b feature/your-feature
   ```

3. **Make changes and test**:
   ```bash
   # Make your changes
   # Run tests
   go test ./...
   # Check formatting
   go fmt ./...
   ```

4. **Push and create PR**:
   ```bash
   git push origin feature/your-feature
   ```

### Code Review Process

1. **Automated Checks**: CI runs tests, linting, and security scans
2. **Review Requirements**: At least one maintainer approval
3. **Response Time**: We aim to review PRs within 48 hours
4. **Feedback**: Address review comments promptly

### Documentation

When contributing, please update relevant documentation:

- **API Changes**: Update `docs/api-documentation.md`
- **New Features**: Add to user guides
- **Architecture Changes**: Update ADRs in `docs/adr/`
- **Configuration**: Update environment examples

## Advanced Topics

### Adding New Storage Backends

To add a new storage backend:

1. Implement the `storage.Interface`:
   ```go
   type Interface interface {
       CreateUser(ctx context.Context, user *User) error
       GetUser(ctx context.Context, userID string) (*User, error)
       // ... other methods
   }
   ```

2. Add configuration in `internal/config/`:
   ```go
   type NewBackendConfig struct {
       ConnectionString string
       MaxConnections   int
   }
   ```

3. Update factory in `internal/platform/factory/`:
   ```go
   case "newbackend":
       return newbackend.New(cfg.NewBackend)
   ```

4. Add tests following existing patterns

### Extending the API

1. **Define the endpoint** in `internal/api/router.go`
2. **Create handler** in `internal/api/http/`
3. **Add validation** in `internal/api/validate/`
4. **Implement business logic** in `internal/core/`
5. **Add tests** at each layer
6. **Update API documentation**

### Performance Optimization

1. **Profile First**: Use pprof to identify bottlenecks
2. **Optimize Queries**: Add appropriate indexes
3. **Batch Operations**: Reduce round trips
4. **Caching**: Implement where appropriate
5. **Concurrent Processing**: Use goroutines wisely

Example profiling:
```bash
# Enable profiling endpoint
ENABLE_PROFILING=true ./bin/memory-service

# Capture CPU profile
go tool pprof http://localhost:8080/debug/pprof/profile

# Capture memory profile
go tool pprof http://localhost:8080/debug/pprof/heap
```

## Community

- **GitHub Issues**: Report bugs and request features
- **Discussions**: Ask questions and share ideas
- **Discord**: Join our community chat
- **Contributing**: See [CONTRIBUTING.md](CONTRIBUTING.md)

Remember: We're here to help! Don't hesitate to ask questions in discussions or on Discord.
