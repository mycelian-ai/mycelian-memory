.PHONY: help mcp-streamable-up mcp-streamable-down mcp-streamable-restart

# ==============================================================================
# Monorepo Convenience Makefile (top-level)
# Provides shortcuts for running the Synapse MCP server via Docker Compose.
# ==============================================================================

MCP_COMPOSE_FILE := deployments/docker/docker-compose.streamable.yml

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
	cd clients/go && go build -o ../../bin/mycelian-mcp-server ./cmd/mycelian-mcp-server

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

mcp-streamable-up:
	docker compose -f $(MCP_COMPOSE_FILE) up -d --build --force-recreate

mcp-streamable-down:
	docker compose -f $(MCP_COMPOSE_FILE) down

mcp-streamable-restart: mcp-streamable-down mcp-streamable-up 