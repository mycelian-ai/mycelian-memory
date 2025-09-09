.PHONY: help start-mcp-streamable-server mcp-streamable-down mcp-streamable-restart

# ==============================================================================
# Monorepo Convenience Makefile (top-level)
# Provides shortcuts for running the Mycelian MCP server via Docker Compose.
# ==============================================================================

MCP_COMPOSE_FILE := deployments/docker/docker-compose.streamable.yml
API_HEALTH_URL := http://localhost:11545/v0/health

# ------------------------------------------------------------------------------
# Backend (server) convenience wrappers
# ------------------------------------------------------------------------------
.PHONY: start-dev-mycelian-server backend-down backend-status backend-logs backend-clean-postgres



start-dev-mycelian-server:
	$(MAKE) -C server run-postgres

backend-down:
	$(MAKE) -C server docker-stop
# Data cleanup wrappers (explicit destructive actions)
backend-clean-postgres:
	$(MAKE) -C server clean-postgres-data


backend-status:
	$(MAKE) -C server docker-status

backend-logs:
	$(MAKE) -C server docker-logs

# ------------------------------------------------------------------------------
# Binary building
# ------------------------------------------------------------------------------
.PHONY: build build-mycelian-cli build-mcp-server build-all clean-bin build-mycelian-service-tools build-outbox-worker build-check quality-check

# Create bin directory
bin:
	mkdir -p bin

# Build mycelianCli tool to deterministic path
build-mycelian-cli: bin
	cd tools/mycelianCli && go build -o ../../bin/mycelianCli .

# Build MCP server to deterministic path
build-mcp-server: bin
	go build -o bin/mycelian-mcp-server ./cmd/mycelian-mcp-server

# Build Mycelian Service Tools CLI
build-mycelian-service-tools: bin
	cd tools/mycelian-service-tools && GOWORK=off go build -o ../../bin/mycelian-service-tools .

# Build outbox-worker app
build-outbox-worker: bin
	cd cmd/outbox-worker && GOWORK=off go build -o ../../bin/outbox-worker .

# Build memory-service app
build-memory-service: bin
	go build -o bin/memory-service ./cmd/memory-service

# Build all binaries
build-all: build-mycelian-cli build-mcp-server build-mycelian-service-tools build-outbox-worker build-memory-service

# Alias for build-all
build: build-all

# Build check - compile all modules to catch compilation errors
build-check:
	@echo "Building all workspace modules to check for compilation errors..."
	@echo "Building client module..."
	@cd client && go build ./... && go test -c ./... -o /dev/null
	@echo "Building server module..."
	@cd server && go build ./... && go test -c ./... -o /dev/null
	@echo "Building tools/mycelianCli module..."
	@cd tools/mycelianCli && go build .
	@echo "Building mcp module..."
	@cd mcp && go build ./... && go test -c ./... -o /dev/null
	@echo "Building cmd/mycelian-mcp-server module..."
	@cd cmd/mycelian-mcp-server && go build .
	@echo "Building cmd/memory-service module..."
	@cd cmd/memory-service && go build .
	@echo "Building cmd/outbox-worker module..."
	@cd cmd/outbox-worker && go build .
	@echo "Building tools/mycelian-service-tools module..."
	@cd tools/mycelian-service-tools && go build .
	@echo "Building tools/invariants-checker module..."
	@cd tools/invariants-checker && go build .
	@echo "All modules compiled successfully!"
	@$(MAKE) assert-ports

# Quality check - run full quality gate for release-ready code
quality-check:
	@echo "Running full quality gate..."
	@echo "1. Formatting code..."
	@cd client && go fmt ./...
	@cd server && go fmt ./...
	@cd mcp && go fmt ./...
	@cd tools/mycelianCli && go fmt ./...
	@cd tools/mycelian-service-tools && go fmt ./...
	@cd tools/invariants-checker && go fmt ./...
	@cd cmd/mycelian-mcp-server && go fmt ./...
	@cd cmd/memory-service && go fmt ./...
	@cd cmd/outbox-worker && go fmt ./...
	@echo "2. Running static analysis..."
	@cd client && go vet ./...
	@cd server && go vet ./...
	@cd mcp && go vet ./...
	@cd tools/mycelianCli && go vet ./...
	@cd tools/mycelian-service-tools && go vet ./...
	@cd tools/invariants-checker && go vet ./...
	@cd cmd/mycelian-mcp-server && go vet ./...
	@cd cmd/memory-service && go vet ./...
	@cd cmd/outbox-worker && go vet ./...
	@echo "3. Running tests with race detector..."
	@cd client && go test -race ./...
	@cd server && go test -race ./...
	@cd mcp && go test -race ./...
	@cd tools/mycelianCli && go test -race ./...
	@cd tools/mycelian-service-tools && go test -race ./...
	@cd tools/invariants-checker && go test -race ./...
	@echo "4. Cleaning up dependencies..."
	@go work sync
	@echo "5. Running comprehensive linter..."
	@cd client && golangci-lint run
	@cd server && golangci-lint run
	@cd mcp && golangci-lint run
	@cd tools/mycelianCli && golangci-lint run
	@cd tools/mycelian-service-tools && golangci-lint run
	@cd tools/invariants-checker && golangci-lint run
	@echo "6. Scanning for vulnerabilities..."
	@cd client && govulncheck ./...
	@cd server && govulncheck ./...
	@cd mcp && govulncheck ./...
	@cd tools/mycelianCli && govulncheck ./...
	@cd tools/mycelian-service-tools && govulncheck ./...
	@cd tools/invariants-checker && govulncheck ./...
	@$(MAKE) assert-ports
	@echo "All quality checks passed!"

.PHONY: assert-ports
assert-ports:
	@set -eu; \
	echo "Asserting authoritative port mappings..."; \
	f1=deployments/docker/docker-compose.postgres.yml; \
	f2=deployments/docker/docker-compose.streamable.yml; \
	if ! grep -q '11544:5432' $$f1; then echo "ERROR: Postgres mapping 11544:5432 not found in $$f1" >&2; exit 1; fi; \
	if ! grep -q '11543:8080' $$f1; then echo "ERROR: Weaviate mapping 11543:8080 not found in $$f1" >&2; exit 1; fi; \
	if ! grep -q '11545:11545' $$f1; then echo "ERROR: Memory service mapping 11545:11545 not found in $$f1" >&2; exit 1; fi; \
	if ! grep -q '11546:11546' $$f2; then echo "ERROR: MCP mapping 11546:11546 not found in $$f2" >&2; exit 1; fi; \
	if ! grep -q 'default:"11545"' server/internal/config/config.go; then echo "ERROR: Default HTTP port 11545 not found in server/internal/config/config.go" >&2; exit 1; fi; \
	if ! grep -q ':11546' mcp/server.go; then echo "ERROR: MCP server bind :11546 not found in mcp/server.go" >&2; exit 1; fi; \
	if ! (grep -q '### Ports' README.md && grep -q '11546' README.md && grep -q '11545' README.md && grep -q '11544' README.md && grep -q '11543' README.md); then echo "ERROR: README ports section missing expected entries" >&2; exit 1; fi; \
	echo "Port assertions passed."

# Clean built binaries
clean-bin:
	rm -rf bin/

# Update help output
help:
	@echo "Mycelian Monorepo Makefile — available commands:"
	@echo ""
	@echo "Build Commands:"
	@echo "  build                  Build all binaries to bin/ directory"
	@echo "  build-check            Compile all workspace modules (catches compilation errors)"
	@echo "  quality-check          Run full quality gate (fmt, vet, test, lint, vuln scan)"
	@echo "  build-mycelian-cli     Build mycelianCli to bin/mycelianCli"
	@echo "  build-mcp-server       Build MCP server to bin/mycelian-mcp-server"
	@echo "  build-memory-service   Build memory service to bin/memory-service"
	@echo "  clean-bin              Remove all built binaries"
	@echo ""
	@echo "Service Commands:"
	@echo "  start-mcp-streamable-server      Start (or rebuild) the Mycelian MCP server container (streamable HTTP for Cursor)"
	@echo "  mcp-streamable-down    Stop and remove the Mycelian MCP server container"
	@echo "  mcp-streamable-restart Shortcut for mcp-streamable-down then start-mcp-streamable-server"
	@echo "  start-dev-mycelian-server    Start backend stack (Postgres)"
	@echo "  backend-down           Stop backend stack containers"
	@echo "  backend-status         Show backend container status"
	@echo "  backend-logs           Tail backend container logs"
	@echo ""
	@echo "Test Commands:"
	@echo "  client-coverage-check  Run client tests and assert >= 78% coverage"
	@echo "  protogen               Generate gRPC code from api/proto via buf"
	@echo "  test-full-local-stack  Run server tests, start postgres backend, then client tests (unit+integration)"

start-mcp-streamable-server:
	docker compose -f $(MCP_COMPOSE_FILE) up -d --build --force-recreate

mcp-streamable-down:
	docker compose -f $(MCP_COMPOSE_FILE) down

mcp-streamable-restart: mcp-streamable-down start-mcp-streamable-server

.PHONY: client-coverage-check
client-coverage-check:
	cd client && bash scripts/coverage_check.sh 78.0

.PHONY: protogen
protogen:
	cd api && buf generate

# ------------------------------------------------------------------------------
# End-to-end developer test pipeline
# ------------------------------------------------------------------------------
.PHONY: server-test server-e2e client-test client-test-integration wait-backend-health test-full-local-stack

server-test:
	$(MAKE) -C server test

# Server dev-env E2E tests (tagged e2e) run against the running docker stack
server-e2e:
	cd server && go test -v ./dev_env_e2e_tests -tags=e2e


client-test:
	cd client && go test -v ./...

client-test-integration:
	cd client && TEST_BACKEND_URL=http://localhost:11545 go test -v -tags=integration ./integration_test/real

wait-backend-health:
	@echo "Waiting for memory-service to be healthy at $(API_HEALTH_URL) ..."
	@i=0; \
	until curl -sf $(API_HEALTH_URL) | grep -q '"status":"healthy"'; do \
	  if [ $$i -ge 60 ]; then echo "ERROR: backend health timeout"; exit 1; fi; \
	  i=$$((i+1)); sleep 2; \
	done; \
	echo "Backend is responding."

test-full-local-stack:
	@set -euo pipefail; \
	cleanup() { $(MAKE) backend-down; }; \
	trap 'cleanup' EXIT INT TERM; \
    $(MAKE) server-test; \
    $(MAKE) start-dev-mycelian-server; \
	$(MAKE) wait-backend-health; \
	$(MAKE) server-e2e; \
	$(MAKE) client-test; \
	$(MAKE) client-test-integration; \
	trap - EXIT INT TERM; \
	cleanup; \
	echo "ALL LOCAL STACK TESTS COMPLETED SUCCESSFULLY"
