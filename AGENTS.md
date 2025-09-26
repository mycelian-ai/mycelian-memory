# Repository Guidelines

## Project Structure & Module Organization
- `server/` provides the service; core logic in `internal/{core,api,storage,search}`.
- `client/` hosts the SDK (`docs/designs/client-sdk.md`).
- `cmd/` contains entry points for `memory-service`, `outbox-worker`, `mycelian-mcp-server`.
- `mcp/` implements the streamable MCP server (`docs/designs/mcp-server.md`).
- `tools/` ships CLIs; `deployments/docker/` and `data/` manage the local stack.
- See [docs/monorepo.md](docs/monorepo.md) for full layout and release flow.

## Architecture & Data Model
- Append-only pipeline: agents emit entries, the outbox worker queues embeddings, and search fans out to Postgres + vector store (`docs/designs/001_mycelian_memory_architecture.md`).
- Isolation runs user ➝ vault ➝ memory ➝ entry/context; vault titles guide routing and contexts auto-shard (`docs/designs/data-model.md`).
- MCP handlers rely on hybrid search; keep tags, summaries, and timestamps rich (`docs/designs/context-management.md`).

## Build, Test, and Development Commands
- `make build` → all binaries under `bin/`.
- `make start-dev-mycelian-server` → Postgres, Weaviate, service, worker via Docker Compose.
- `make build-check` → compiles every module and asserts the canonical ports.
- `make quality-check` → fmt, vet, race tests, golangci-lint, govulncheck, workspace sync.
- `make start-mcp-streamable-server` → rebuilds the HTTP MCP endpoint on `localhost:11546/mcp`.

## Coding Style & Naming Conventions
- Keep Go files `gofmt`-clean, tab-indented, ≲120 chars, with explicit package aliases.
- Exported identifiers use PascalCase; packages and files stay lowercase.
- Defaults live in `server/internal/config`; promote knobs to env vars.
- Follow the AI pairing guardrails in `docs/coding-stds/ai-coding-best-practices.md`.

## Testing Guidelines
- Unit tests colocate with code; integration suites sit in `server/dev_env_e2e_tests` and `client/integration_test/real`.
- Iterate with `make server-test`, `make client-test`, and `make server-e2e`; `make test-full-local-stack` runs the Docker-backed gate.
- Maintain ≥78% Go client coverage via `make client-coverage-check`; add regressions whenever persistence or search semantics shift.

## Commit & Pull Request Guidelines
- Follow the existing prefixes (`feat:`, `fix:`, `refactor:`, `revert:`) and keep messages imperative.
- Reference issues, call out contract or schema changes, and list validation commands in PRs.
- Link the relevant design note when touching `api/` or material in `docs/designs/`.

## Current Priorities & Philosophy
- Reliability first: keep abstractions clear, errors well handled, and observability wired (`CLAUDE.md`).
- Track memory quality with LongMemEval (`docs/designs/langgraph_longmemeval_benchmarker.md`) after search, embedding, or context changes.
- Multi-tenant planning is in flight (org ➝ project ➝ vault); note new isolation assumptions.

## Security & Configuration Tips
- Never commit secrets; copy `.env.example` patterns and rely on Compose overrides.
- Keep ports `11543-11546` aligned with Makefile checks and `docs/ports-verification.md`; use `make clean-local-postgres-data` for destructive resets.
