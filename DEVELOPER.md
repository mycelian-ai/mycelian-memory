# Developing Mycelian Memory Backend

- [Developing Mycelian Memory Backend](#developing-mycelian-memory-backend)
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

Thank you for your interest in contributing to Mycelian Memory Backend! This document will guide you through setting up your development environment and understanding our codebase.

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
        API[Memory Service API<br/>:11545]
    end
    
    subgraph "Storage Layer"
        Postgres[Postgres<br/>:5432]
    end
    
    subgraph "Search Layer"
        Weaviate[Weaviate<br/>:8082]
        Indexer[Indexer Service]
    end
    
    subgraph "CLI Tools"
        ServiceTools[mycelian-service-tools]
    end
    
    API --> Postgres
    Indexer --> Postgres
    Indexer --> Weaviate
    API --> Weaviate
    
    ServiceTools --> API
```

### Data Flow

1. **Write Path**: Client → API → Postgres → Event Bus → Indexer → Weaviate
2. **Read Path**: Client → API → Postgres (for metadata) + Weaviate (for search)
3. **Search Path**: Client → API → Weaviate → Ranked Results

### Storage Backend

The system uses Postgres as its primary storage backend:

| Backend | Use Case | Configuration |
|---------|----------|---------------|
| Postgres | All environments | `DB_ENGINE=postgres` |

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
   git remote add upstream https://github.com/mycelian/mycelian-memory.git
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
   
   # Database Configuration
   DB_ENGINE=postgres
   
   # Postgres Configuration
   POSTGRES_HOST=localhost
   POSTGRES_PORT=5432
   POSTGRES_USER=mycelian_user
   POSTGRES_PASSWORD=mycelian_password
   POSTGRES_DATABASE=mycelian_memory
   
   # Weaviate Configuration
   WEAVIATE_URL=http://localhost:8082
   ```

3. **Local Development User**: The system automatically creates a default local user for development:
   - **User ID**: `local_user`
   - **Email**: `dev@localhost`
   - **Display Name**: `Local Developer`
   - **Time Zone**: `UTC`
   - **Status**: `ACTIVE`

### Running the Stack

#### Quick Start (Postgres + Weaviate)

```bash
# Start all services with Postgres backend
make start-dev-mycelian-server

# Verify services are running
make backend-status

# View logs
make backend-logs
```

#### Production Setup (Postgres + Weaviate)

```bash
# Start all services
make start-dev-mycelian-server

# Verify services
make backend-status

# View logs
make backend-logs
```

## Development Workflow

### Code Organization

```
memory-backend/
├── cmd/                    # Application entry points
│   ├── memory-service/     # Main API server
│   ├── mycelian-service-tools/  # CLI tool (formerly memoryctl)
│   └── (removed) indexer-prototype/
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
   git commit -m "fix(storage): handle nil pointer in postgres client"
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
# Build service tools CLI
make build-mycelian-service-tools   # → bin/mycelian-service-tools
// indexer-prototype removed

# Build with specific tags
go build -tags production -o bin/memory-service ./cmd/memory-service

# Cross-compilation
GOOS=linux GOARCH=amd64 go build -o bin/memory-service-linux ./cmd/memory-service
```

### Running Services

#### Local Development (No Docker)

```bash
# Build and run locally
make run-local

# Or manually with custom settings
BUILD_TARGET=local \
  DB_ENGINE=postgres \
  POSTGRES_HOST=localhost \
  PORT=8080 \
  ./bin/memory-service
```

#### Docker Development

```bash
# Start specific services
docker-compose up -d memory-service
docker-compose up -d weaviate
// indexer-prototype removed

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

Postgres is the only supported backend; use `deployments/docker/docker-compose.postgres.yml` via server Makefile targets.

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
go test ./internal/v0/...

# Run specific test
go test -run TestMemoryService_Create ./internal/core/memory/

# Verbose output
go test -v ./...

# Run benchmarks
go test -bench=. ./...
go test -bench=BenchmarkMemoryCreate ./internal/core/memory/
```

### E2E Testing

#### Using Service Tools Cookbooks

```bash
# Quick validation cookbook
./tools/mycelian-service-tools/cookbook/memoryctl-simple-scenario.sh

# Comprehensive workflow cookbook
./tools/mycelian-service-tools/cookbook/memoryctl-multi-entities-scenario.sh
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
go test -bench=. -benchtime=10s ./internal/v0/...

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
curl -s http://localhost:11545/health | jq .

# Weaviate health
curl -s http://localhost:8082/v1/.well-known/ready

# Check all services
./scripts/docker-setup/docker-health-check.sh
```

#### Local User Creation

The system automatically creates a default local user for development:

**Local Development**: Uses `EnsureDefaultUser()` function
- Creates user if Users table is empty
- User ID: `local_user`
- Email: `dev@localhost`
- Display Name: `Local Developer`

**Docker Compose**: Uses `user-bootstrap` container
- Waits for memory-service to be healthy
- Uses `mycelian-service-tools` CLI to create user via API
- User ID: `local_user`
- Email: `dev@localhost`
- Display Name: `Local Developer`

You can verify the local user exists:
```bash
# Check if local user was created
curl -s http://localhost:11545/v0/users/local_user | jq .

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
export API="http://localhost:11545"
export EMAIL="debug-$(date +%s)@test.com"

# 1. Create user
USER_RESPONSE=$(curl -s -X POST "$API/v0/users" \
  -H 'Content-Type: application/json' \
  -d '{"email":"'"$EMAIL"'","name":"Debug User"}')
echo "$USER_RESPONSE" | jq .
export USER_ID=$(echo "$USER_RESPONSE" | jq -r '.userId')

# 2. Create vault
VAULT_RESPONSE=$(curl -s -X POST "$API/v0/users/$USER_ID/vaults" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Debug Vault","description":"Testing vault"}')
echo "$VAULT_RESPONSE" | jq .
export VAULT_ID=$(echo "$VAULT_RESPONSE" | jq -r '.vaultId')

# 3. Create memory
MEMORY_RESPONSE=$(curl -s -X POST "$API/v0/users/$USER_ID/vaults/$VAULT_ID/memories" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Debug Memory","description":"Testing memory"}')
echo "$MEMORY_RESPONSE" | jq .
export MEMORY_ID=$(echo "$MEMORY_RESPONSE" | jq -r '.memoryId')

# 4. Add entry
ENTRY_RESPONSE=$(curl -s -X POST "$API/v0/users/$USER_ID/vaults/$VAULT_ID/memories/$MEMORY_ID/entries" \
  -H 'Content-Type: application/json' \
  -d '{"rawText":"The quick brown fox jumps over the lazy dog","summary":"Test entry"}')
echo "$ENTRY_RESPONSE" | jq .
export ENTRY_ID=$(echo "$ENTRY_RESPONSE" | jq -r '.entryId')

# 5. Wait for indexing
echo "Waiting for indexer..."
sleep 3

# 6. Search
SEARCH_RESPONSE=$(curl -s -X POST "$API/v0/search" \
  -H 'Content-Type: application/json' \
  -d '{"userId":"'"$USER_ID"'","memoryId":"'"$MEMORY_ID"'","query":"fox","topK":5}')
echo "$SEARCH_RESPONSE" | jq .
```

### Database Debugging

#### Postgres Debugging

```bash
# Connect to database
psql postgresql://mycelian_user:mycelian_password@localhost:5432/mycelian_memory

# Useful queries
\dt                           # List tables
\d users                      # Describe users table
\d+ memories                  # Describe memories table with details

SELECT * FROM users ORDER BY created_at DESC LIMIT 5;
SELECT * FROM vaults WHERE user_id = 'YOUR_USER_ID';
SELECT * FROM memories WHERE vault_id = 'YOUR_VAULT_ID';
SELECT COUNT(*) as entry_count, memory_id 
  FROM entries 
  GROUP BY memory_id;

# Check database size and activity
SELECT pg_size_pretty(pg_database_size('mycelian_memory'));
SELECT * FROM pg_stat_activity WHERE datname = 'mycelian_memory';
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
lsof -i :11545
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

1. **Postgres**: Check connection and permissions
   ```bash
   # Test database connection
   psql postgresql://mycelian_user:mycelian_password@localhost:5432/mycelian_memory -c "SELECT version();"
   
   # Check if Postgres is running
   ps aux | grep postgres
   docker ps | grep postgres
   
   # Check port availability
   netstat -tulpn | grep 5432
   lsof -i :5432
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
   # indexer-prototype removed
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

3. Update factory in `internal/factory/`:
   ```go
   case "newbackend":
       return newbackend.New(cfg.NewBackend)
   ```

4. Add tests following existing patterns

### Extending the API

1. **Define the endpoint** in `internal/v0/router.go`
2. **Create handler** in `internal/v0/http/`
3. **Add validation** in `internal/v0/validate/`
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
go tool pprof http://localhost:11545/debug/pprof/profile

# Capture memory profile
go tool pprof http://localhost:11545/debug/pprof/heap
```

## Community

- **GitHub Issues**: Report bugs and request features
- **Discussions**: Ask questions and share ideas
- **Discord**: Join our community chat
- **Contributing**: See [CONTRIBUTING.md](CONTRIBUTING.md)

Remember: We're here to help! Don't hesitate to ask questions in discussions or on Discord.
