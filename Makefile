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

# Update help output
help:
	@echo "Synapse Monorepo Makefile — available commands:"
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