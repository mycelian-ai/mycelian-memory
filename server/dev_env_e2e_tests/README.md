# Developer-Environment E2E Tests

These tests run **against the running developer stack** (started via `docker compose up`).
They are tagged with `//go:build e2e` so they execute only when you explicitly ask for them:

```bash
# Run only these tests
go test ./dev_env_e2e_tests -tags=e2e -v

# Run the entire repository's e2e-tagged tests
go test ./... -tags=e2e
```

## Files

| File                       | Purpose                                                |
|----------------------------|--------------------------------------------------------|
| `helpers.go`               | Shared utility functions used by all tests            |
| `smoke_test.go` *(todo)*   | Fast health-check tests (ingestion + /api/search)      |
| `search_relevance_test.go` *(todo)* | Hybrid relevance scenarios (alpha, tag, metadata) |

## Environment variables

| Variable        | Default                 | Meaning                                  |
|-----------------|-------------------------|------------------------------------------|
| `MEMORY_API`    | `http://localhost:8080` | Base URL of memory-service               |
| `WAVIATE_URL`   | `http://localhost:8082` | Base URL of Weaviate instance            |
| `OLLAMA_URL`    | `http://localhost:11434`| Base URL of Ollama embedding service     |
| `EMBED_MODEL`   | `mxbai-embed-large`     | Model name used for hybrid search vectors|

## Quick start

1. `docker compose up -d` (from project root) – starts Spanner emulator, memory-service, indexer, Weaviate & Ollama.
2. Verify all services healthy: `curl localhost:8080/api/health` etc.
3. Run smoke tests: `go test ./dev_env_e2e_tests -tags=e2e -run TestDevEnv_.* -v` *(after Phase 2)*.

> **Note**
> Time-outs inside these tests are intentionally short (1–5 s) so failures surface fast. 