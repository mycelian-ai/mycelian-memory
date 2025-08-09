# ==============================================================================
# Synapse Memory Backend - Makefile
# ==============================================================================
#
# This Makefile provides build automation for the Synapse Memory Backend project.
# It handles both application binaries and development tools.
#
# QUICK START:
#   make build                    - Build all application code
#   make test                     - Run all tests
#   make build-tools              - Build development tools
#   make schema-create-emulator   - Set up local Spanner schema
#   make help                     - Show all available targets
#
# PROJECT STRUCTURE:
#   cmd/memory-service/           - Main API service binary
#   tools/schema-manager/         - Database schema management tool
#   internal/                     - Shared internal packages
#
# DEPENDENCIES:
#   - Go 1.24.3 or later
#   - Google Cloud Spanner emulator (for local development)
#   - Make sure GOPATH and GOROOT are properly configured
#
# DEVELOPMENT WORKFLOW:
#   1. Start Spanner emulator: gcloud emulators spanner start
#   2. Create schema: make schema-create-emulator
#   3. Build and test: make build test
#   4. Run service: ./bin/memory-service (after building)
#
# PRODUCTION DEPLOYMENT:
#   - Use schema-manager tool directly with production credentials
#   - Build binaries with appropriate GOOS/GOARCH for target platform
#   - See tools/README.md for detailed schema management instructions
#
# ==============================================================================

# Go version
GO := go

# Directories
TOOLS_DIR := tools
SCHEMA_MANAGER_DIR := $(TOOLS_DIR)/schema-manager

# Build targets
.PHONY: help build-tools build-schema-manager clean-tools

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
	@echo "🔧 DEVELOPMENT TOOLS:"
	@echo "  make build-tools              - Build all development tools"
	@echo "  make build-schema-manager     - Build schema manager tool"
	@echo "  make clean-tools              - Clean tool binaries"
	@echo ""
	@echo "🗄️  DATABASE SCHEMA (Local Development):"
	@echo "  make schema-create-emulator   - Create tables in Spanner emulator"
	@echo "  make schema-validate-emulator - Validate schema against emulator"
	@echo "  make schema-drop-emulator     - Drop all tables in emulator (DANGEROUS)"
	@echo "  make schema-help              - Show detailed schema management help"
	@echo ""
	@echo "💡 QUICK START:"
	@echo "  1. gcloud emulators spanner start"
	@echo "  2. make schema-create-emulator"
	@echo "  3. make build test"
	@echo ""
	@echo "📚 For more details, see the file header comments in this Makefile"

# Default target
.DEFAULT_GOAL := help

# ============================================================================== 
# Local macOS development targets (SQLite + Waviate)
# ==============================================================================

.PHONY: build-local run-local

# Build backend binary for macOS (CGO disabled for pure-Go SQLite)
build-local:
	@echo "🔨 Building memory-service for macOS (local target)..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o bin/memory-service ./cmd/memory-service
	@echo "✅ Built bin/memory-service"

# Run backend in local mode (uses ~/.synapse-memory)
run-local: build-local
	@echo "🛑 Stopping any existing memory-service ..."
	- pkill -f "bin/memory-service" || true
	@echo "🚀 Starting memory-service (local build target)"
	BUILD_TARGET=local ./bin/memory-service

build-tools: build-schema-manager

build-schema-manager:
	@echo "🔨 Building schema manager..."
	cd $(SCHEMA_MANAGER_DIR) && $(GO) build -o schema-manager .
	@echo "✅ Schema manager built successfully"

clean-tools:
	@echo "🧹 Cleaning tools..."
	rm -f $(SCHEMA_MANAGER_DIR)/schema-manager
	@echo "✅ Tools cleaned"

# Schema management targets
.PHONY: schema-help schema-create-emulator schema-validate-emulator schema-drop-emulator

schema-help:
	@echo "📚 Schema Management Commands:"
	@echo ""
	@echo "  make build-schema-manager     - Build the schema manager tool"
	@echo "  make schema-create-emulator   - Create tables in Spanner emulator"
	@echo "  make schema-validate-emulator - Validate schema against emulator"
	@echo "  make schema-drop-emulator     - Drop all tables in emulator (DANGEROUS)"
	@echo ""
	@echo "💡 For production use, run the tool manually with appropriate credentials:"
	@echo "   ./tools/schema-manager/schema-manager -project YOUR_PROJECT -instance YOUR_INSTANCE -database YOUR_DB -credentials PATH_TO_CREDS -operation create-tables"

schema-create-emulator: build-schema-manager
	@echo "🏗️  Creating tables in Spanner emulator..."
	./$(SCHEMA_MANAGER_DIR)/schema-manager \
		-project test-project \
		-instance test-instance \
		-database memory-backend \
		-emulator localhost:9010 \
		-operation create-tables

schema-validate-emulator: build-schema-manager
	@echo "🔍 Validating schema in Spanner emulator..."
	./$(SCHEMA_MANAGER_DIR)/schema-manager \
		-project test-project \
		-instance test-instance \
		-database memory-backend \
		-emulator localhost:9010 \
		-operation validate-schema

schema-drop-emulator: build-schema-manager
	@echo "⚠️  WARNING: This will drop all tables in the emulator!"
	./$(SCHEMA_MANAGER_DIR)/schema-manager \
		-project test-project \
		-instance test-instance \
		-database memory-backend \
		-emulator localhost:9010 \
		-operation drop-tables

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

# Run stack with selectable DB_ENGINE (e.g., make docker-run DB_ENGINE=sqlite)
.PHONY: docker-run
docker-run:
	docker-compose up -d --build --force-recreate

docker-stop:
	# Stop sqlite-based local stack (memory-service + weaviate + indexer + bootstrap)
	docker compose -f ../deployments/docker/docker-compose.sqlite.yml down --remove-orphans -v || true
	# Also stop legacy spanner stack if running (best-effort)
	docker compose -f ../deployments/docker/docker-compose.spanner.yml down --remove-orphans -v || true

docker-status:
	./scripts/start-headless.sh --status

docker-logs:
	docker-compose logs -f

docker-clean:
	./scripts/start-headless.sh --clean 

# Short concise local run targets
.PHONY: run-spanner run-sqlite

run-spanner:
	docker compose -f ../deployments/docker/docker-compose.spanner.yml up -d --build --force-recreate

run-sqlite:
	docker compose -f ../deployments/docker/docker-compose.sqlite.yml up -d --build --force-recreate

# Map previous long form targets to new concise names (for backwards compatibility)
docker-run-local-mycelian-service-spanner-emulator: run-spanner
docker-run-local-mycelian-service-sqlite: run-sqlite

docker-run-local-spanner: run-spanner
docker-run-local-sqllite: run-sqlite

docker-run-spanner: run-spanner
docker-run-sqlite: run-sqlite 
