.PHONY: help mcp-streamable-up mcp-streamable-down mcp-streamable-restart

# ==============================================================================
# Monorepo Convenience Makefile (top-level)
# Provides shortcuts for running the Synapse MCP server via Docker Compose.
# ==============================================================================

MCP_COMPOSE_FILE := deployments/docker/docker-compose.streamable.yml
API_HEALTH_URL := http://localhost:8080/api/health

# ------------------------------------------------------------------------------
# Backend (server) convenience wrappers
# ------------------------------------------------------------------------------
.PHONY: backend-spanner-up backend-sqlite-up backend-down backend-status backend-logs

backend-spanner-up:
	$(MAKE) -C server run-spanner

backend-sqlite-up:
	$(MAKE) -C server run-sqlite

backend-down:
	$(MAKE) -C server docker-stop

backend-status:
	$(MAKE) -C server docker-status

backend-logs:
	$(MAKE) -C server docker-logs

# ------------------------------------------------------------------------------
# Binary building
# ------------------------------------------------------------------------------
.PHONY: build build-mycelian-cli build-mcp-server build-all clean-bin

# Create bin directory
bin:
	mkdir -p bin

# Build mycelianCli tool to deterministic path
build-mycelian-cli: bin
	cd tools/mycelianCli && go build -o ../../bin/mycelianCli .

# Build MCP server to deterministic path  
build-mcp-server: bin
	go build -o bin/mycelian-mcp-server ./cmd/mycelian-mcp-server

# Build all binaries
build-all: build-mycelian-cli build-mcp-server

# Alias for build-all
build: build-all

# Clean built binaries
clean-bin:
	rm -rf bin/

# Update help output
help:
	@echo "Synapse Monorepo Makefile — available commands:"
	@echo ""
	@echo "Build Commands:"
	@echo "  build                  Build all binaries to bin/ directory"
	@echo "  build-mycelian-cli     Build mycelianCli to bin/mycelianCli"
	@echo "  build-mcp-server       Build MCP server to bin/mycelian-mcp-server"
	@echo "  clean-bin              Remove all built binaries"
	@echo ""
	@echo "Service Commands:"
	@echo "  mcp-streamable-up      Start (or rebuild) the Synapse MCP server container (streamable HTTP for Cursor)"
	@echo "  mcp-streamable-down    Stop and remove the Synapse MCP server container"
	@echo "  mcp-streamable-restart Shortcut for mcp-streamable-down then mcp-streamable-up"
	@echo "  backend-spanner-up     Start backend stack (Spanner emulator)"
	@echo "  backend-sqlite-up      Start backend stack (SQLite)"
	@echo "  backend-down           Stop backend stack containers"
	@echo "  backend-status         Show backend container status"
	@echo "  backend-logs           Tail backend container logs"
	@echo ""
	@echo "Test Commands:"
	@echo "  client-coverage-check  Run client tests and assert >= 78% coverage"
	@echo "  protogen               Generate gRPC code from api/proto via buf"
	@echo "  test-all               Run server tests, start sqlite backend, then client tests (unit+integration)"

mcp-streamable-up:
	docker compose -f $(MCP_COMPOSE_FILE) up -d --build --force-recreate

mcp-streamable-down:
	docker compose -f $(MCP_COMPOSE_FILE) down

mcp-streamable-restart: mcp-streamable-down mcp-streamable-up 

.PHONY: client-coverage-check
client-coverage-check:
	cd client && bash scripts/coverage_check.sh 78.0

.PHONY: protogen
protogen:
	cd api && buf generate

# ------------------------------------------------------------------------------
# End-to-end developer test pipeline
# ------------------------------------------------------------------------------
.PHONY: server-test server-e2e client-test client-test-integration wait-backend-health test-all

server-test:
	$(MAKE) -C server test

# Server dev-env E2E tests (tagged e2e) run against the running docker stack
server-e2e:
	cd server && go test -v ./dev_env_e2e_tests -tags=e2e || true

client-test:
	cd client && go test -v ./...

client-test-integration:
	cd client && TEST_BACKEND_URL=http://localhost:8080 go test -v -tags=integration ./integration_test/real

wait-backend-health:
	@echo "Waiting for memory-service to be healthy at $(API_HEALTH_URL) ..."
	@i=0; \
	until curl -sf $(API_HEALTH_URL) | grep -q 'UP'; do \
	  if [ $$i -ge 60 ]; then echo "ERROR: backend health timeout"; exit 1; fi; \
	  i=$$((i+1)); sleep 2; \
	done; \
	echo "Backend is healthy."

test-all:
	@set -e; \
	cleanup() { $(MAKE) backend-down; }; \
	trap 'cleanup' EXIT INT TERM; \
	$(MAKE) server-test; \
	$(MAKE) backend-sqlite-up; \
	$(MAKE) wait-backend-health; \
	$(MAKE) server-e2e; \
	$(MAKE) client-test; \
	$(MAKE) client-test-integration; \
	trap - EXIT INT TERM; \
	cleanup; \
	echo "ALL TESTS COMPLETED SUCCESSFULLY"