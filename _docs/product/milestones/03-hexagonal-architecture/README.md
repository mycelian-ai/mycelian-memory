# 03 – Hexagonal Architecture (⏳ in-progress)

Target date: TBD

## Goal
Reorganize the server into clear layers with swappable adapters, using plain names:
- Model (business types, errors)
- Services (use cases orchestrating dependencies)
- API (HTTP now; gRPC later)
- Store (persistence interface + adapters)
- SearchIndex (search/index interface + adapters)
- Embeddings (embedding interface + adapters)

## KPIs
- Model has no imports of HTTP, DB, or vendor SDKs
- API handlers call Services via interfaces (no direct DB/search calls)
- Hard-deletes are synchronous and propagate to the SearchIndex without outbox reliance
- Local checks green: go fmt, go vet, go test -race, golangci-lint, govulncheck
- Invariant tests pass with existing REST endpoints

## Exit Criteria
- Store and SearchIndex interfaces defined; Postgres and Waviate adapters conform
- Services orchestrate create/get/list/delete/search using only interfaces
- API is thin (decode → service → encode); no business logic in handlers
- Composition root wires adapters via config; old coupled paths removed
- Adapter compliance tests exist for Store and SearchIndex

## Scope
- Refactor only; keep external API and behavior (REST endpoints remain)
- Prepare for future adapters (SQLite, Qdrant) and future gRPC + gateway milestone

## Links
- Story: stories/001-domain-transport-split.md
- Future milestone: gRPC primary + REST gateway (separate)
