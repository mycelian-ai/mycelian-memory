# ==============================================================================
# Mycelian Memory Backend - Makefile
# ==============================================================================
#
# This Makefile provides build automation for the Mycelian Memory Backend project.
# It handles both application binaries and development tools.
#
# QUICK START:
#   make build                    - Build all application code
#   make test                     - Run all tests
#   make run-postgres             - Start Postgres backend stack
#   make help                     - Show all available targets
#
# PROJECT STRUCTURE:
#   cmd/memory-service/           - Main API service binary
#   internal/                     - Shared internal packages
#
# DEPENDENCIES:
#   - Go 1.24.6 or later
#   - Docker Desktop (for Postgres stack)
#   - Make sure GOPATH and GOROOT are properly configured
#
# DEVELOPMENT WORKFLOW:
#   1. Build and test: make build test
#   2. Start backend: make run-postgres
#   3. Run service: ./bin/memory-service (after building)
#
# PRODUCTION DEPLOYMENT:
#   - Build binaries with appropriate GOOS/GOARCH for target platform
#   - Use Aurora Serverless V2 for production Postgres
#
# ==============================================================================

# Go version
GO := go

# Directories
TOOLS_DIR := tools

# Build targets
.PHONY: help

# Default target - show help
help:
	@echo "🚀 Mycelian Memory Backend - Available Commands"
	@echo "=============================================="
	@echo ""
	@echo "📦 APPLICATION BUILDS:"
	@echo "  make build                    - Build all application code"
	@echo "  make test                     - Run all tests"
	@echo ""
	@echo "🐳 DOCKER TESTING:"
	@echo "  make docker-start             - Start services in headless mode"
	@echo "  make docker-stop              - Stop all Docker services"
	@echo "  make docker-status            - Show Docker service status"
	@echo "  make docker-help              - Show detailed Docker commands"
	@echo ""
	@echo "🔧 POSTGRES BACKEND:"
	@echo "  make run-postgres             - Start Postgres + Weaviate stack"
	@echo "  make clean-postgres-data      - Reset Postgres data (DANGEROUS)"
	@echo ""
	@echo "💡 QUICK START:"
	@echo "  1. make build test"
	@echo "  2. make run-postgres"
	@echo "  3. ./bin/memory-service"
	@echo ""
	@echo "📚 For more details, see the file header comments in this Makefile"

# Default target
.DEFAULT_GOAL := help

# ============================================================================== 
# Local macOS development targets
# ==============================================================================

.PHONY: build-local run-local

# Build backend binary for macOS
build-local:
	@echo "🔨 Building memory-service for macOS (local target)..."
	cd .. && GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o server/bin/memory-service ./cmd/memory-service
	@echo "✅ Built server/bin/memory-service"

# Run backend in local mode
run-local: build-local
	@echo "🛑 Stopping any existing memory-service ..."
	- pkill -f "server/bin/memory-service" || true
	@echo "🚀 Starting memory-service (local build target)"
	MEMORY_SERVER_BUILD_TARGET=local ./bin/memory-service

 

 

# Development targets
.PHONY: test build

test:
	$(GO) test ./...

build:
	$(GO) build ./...

# Docker targets
.PHONY: docker-help docker-build docker-start docker-stop docker-status docker-logs docker-clean

docker-help:
	@echo "🐳 Docker Commands:"
	@echo ""
	@echo "  make docker-build        - Build Docker images"
	@echo "  make docker-start        - Start services in headless mode"
	@echo "  make docker-stop         - Stop all Docker services"
	@echo "  make docker-status       - Show status of Docker services"
	@echo "  make docker-logs         - Show logs from all services"
	@echo "  make docker-clean        - Clean up containers and images"
	@echo ""
	@echo "💡 For more advanced options, use: ./scripts/start-headless.sh --help"

docker-build:
	./scripts/start-headless.sh --build

docker-start:
	docker-compose up -d --build --force-recreate

# Run stack
.PHONY: docker-run
docker-run:
	docker-compose up -d --build --force-recreate

docker-stop:
	# Stop postgres stack if running (best-effort, non-destructive)
	docker compose -f ../deployments/docker/docker-compose.postgres.yml down --remove-orphans || true

docker-status:
	./scripts/start-headless.sh --status

docker-logs:
	docker-compose logs -f

docker-clean:
	./scripts/start-headless.sh --clean 

# Postgres backend targets


.PHONY: run-postgres
run-postgres:
	docker compose -f ../deployments/docker/docker-compose.postgres.yml up -d --build --force-recreate

# -----------------------------------------------------------------------------
# Explicit data cleanup targets (DANGEROUS)
# -----------------------------------------------------------------------------
.PHONY: clean-postgres-data

clean-postgres-data:
	@echo "This will delete all local Postgres data under ../data/postgres";
	rm -rf ../data/postgres && mkdir -p ../data/postgres && echo "Postgres data directory reset."


# Backwards compatibility aliases
docker-run-postgres: run-postgres
