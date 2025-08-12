# Story 001 — Model/Services/API Split

## Problem
Server code mixes HTTP transport with storage/search concerns. We need a clean Model (business types) and thin API (HTTP) with Services orchestrating work through Store/SearchIndex/Embeddings interfaces and swappable adapters.

## Approach
- Add `internal/model` with types and errors.
- Add `internal/services` with use-case methods (no HTTP/DB/vendor imports).
- Define interfaces in `internal/store` (persistence), `internal/searchindex` (search/index), `internal/embeddings` (embedding).
- Add `internal/api/http` with thin handlers and routing.
- Move Postgres and Waviate into adapters conforming to the Store and SearchIndex interfaces.
- Wire everything in `memoryservice/run.go`.

## Definition of Done (DoD)
- Model has no imports of HTTP, DB, or vendor SDKs
- API handlers call only Services; no business logic in handlers
- Hard-deletes propagate synchronously to SearchIndex; outbox is fallback only
- All local checks pass; invariant tests pass
- Old coupled code paths deleted; imports updated

## Tasks
| Order | Title | Brief | Status |
|---|---|---|---|
| 001 | Create model types and errors | `internal/model/{types.go,errors.go}` | done |
| 004 | Define Store/SearchIndex/Embeddings interfaces | `internal/{store,searchindex,embeddings}/*.go` | done |
| 007 | Implement Services | `internal/services/*.go` CRUD/search orchestration | in-progress |
| 010 | Adapt Postgres to Store | Move to `internal/store/postgres` and conform | done (bridge over legacy `internal/storage`) |
| 013 | Adapt Waviate to SearchIndex | Move to `internal/searchindex/waviate` and conform | done |
| 016 | Extract Embeddings adapter | `internal/embeddings/ollama` (provider split) | todo |
| 019 | Wire composition root | `memoryservice/run.go` builds adapters → services → router | done |
| 022 | Thin HTTP API | `internal/api/http/{handlers.go,router.go}` using services | done (Users, Vaults, Memories CRUD + entries/contexts wired; title routes migrated; legacy routes commented) |
| 025 | Enforce synchronous deletes | Services call SearchIndex delete on memory/entry/context/vault | done |
| 037 | Legacy adapter bridge | `internal/store/legacy` wraps `internal/storage` during migration | removed (replaced by `internal/store/postgres` adapter) |
| 028 | Add adapter compliance tests | `storetest` and `searchindextest` suites | todo |
| 031 | Verify invariants | Blackbox tests still pass with existing endpoints | todo |
| 034 | Remove legacy paths | Delete superseded code; update imports | done (legacy HTTP handlers removed; legacy router removed; `internal/search/*` deleted) |
| 038 | Implement `internal/store/postgres` and swap | Move code from `internal/storage/postgres` to `internal/store/postgres`; implement `store.Store`; switch composition root to new store | planned |
| 041 | Native Waviate `searchindex` | Implement `internal/searchindex/waviate` with full DeleteEntry/Context/Memory/Vault; remove bridge | done |
| 044 | Extract Embeddings | Create `internal/embeddings` with `ollama` and `openai` adapters; update services wiring | planned |
| 047 | Factory refactor | Add `factory.NewStore(cfg)` returning `store.Store`; update server wiring; deprecate/remove `NewStorage` | done (NewStore in use; NewStorage kept temporarily for health) |
| 050 | Remove legacy router | Ensure parity, then delete legacy router mount and files | done |
| 053 | Search index compliance tests | Add `searchindextest` suite incl. delete ops; skip when not configured | planned |
| 056 | Expand store compliance | Broaden `storetest` coverage incl. paging/filters; run against new Postgres store | planned |
| 059 | Final cleanup | Delete `internal/storage/*` and unused `internal/core/*` after confirming no references | planned |

## Notes
- Future work (new milestone): gRPC primary + REST via grpc-gateway.
- Planned adapters to unblock portability: SQLite (Store), Qdrant (SearchIndex).

## Progress Summary
- Introduced Model/Services/API layering to separate business logic from HTTP and vendors.
- Defined Store/SearchIndex/Embeddings interfaces for swappable adapters.
- Added legacy Store adapter bridging existing `internal/storage` to new Store interfaces.
- Added `WaviateIndex` adapter bridging existing searcher to SearchIndex.
- Implemented v2 HTTP handlers (Users, Vaults, Memories, Entries, Contexts); migrated title-based routes.
- Removed legacy router; v2 routes wired directly (health, search, CRUD).
- Wired native SearchIndex and Embeddings in `memoryservice/run.go` (vector optional if no Waviate URL).
- Enforced sync delete propagation (memory/entry/context) in services with nil-guarded index.

Why
- Improves testability and maintainability (no transport/vendor coupling in services).
- Enables easy swaps (Postgres⇄SQLite, Waviate⇄Qdrant) and future gRPC gateway.
- Aligns with hard-delete policy by ensuring index deletes occur synchronously.

## Next Tasks
- Store/postgres swap
  - Implement `internal/store/postgres` and run `storetest.Run` against it.
  - Switch composition root to the new store; remove `internal/store/legacy`.
- Embeddings providers
  - Add `openai` provider and tests; keep `ollama` placeholder until native client ready.
- Tests and CI
  - Expand `storetest` coverage; keep `searchindextest` delete coverage; add CI gates for server on PRs touching `server/`.
- Final cleanup
  - Delete `internal/storage/*` and unused `internal/core/*` after confirming no references.

## Legacy Completion Plan → Target Architecture
- Storage abstraction
  - Build `internal/store/postgres` implementing `store.Store`. Migrate code from `internal/storage/postgres` and convert types to `internal/model`.
  - Add `factory.NewStore(cfg)` returning `store.Store`. Update `memoryservice/run.go` to use it. Remove `internal/store/legacy` and then `internal/storage/*`.
- SearchIndex abstraction
  - Implement `internal/searchindex/waviate` natively with coarse deletes for Memory and Vault. Remove `searchindex.WaviateIndex` bridge and delete `internal/search`.
- Embeddings
  - Create `internal/embeddings` interface and providers (`ollama`, `openai`). Wire into services. Avoid coupling embeddings to `searchindex`.
- HTTP/API
  - Keep only `internal/api/http` v2 handlers and router. Remove legacy router mount after parity is verified.
- Tests & CI
  - `storetest` runs against `store.Store` implementations (postgres). `searchindextest` validates delete ops. Invariants stay green. CI gates run fmt/vet/test/lint/govulncheck on server changes.
- Definition of Done for Legacy Removal
  - No references to `internal/storage/*`, `internal/search/*`, or legacy router remain. Composition root uses `store.Store`, `searchindex.Index`, and `embeddings.Provider` exclusively.
