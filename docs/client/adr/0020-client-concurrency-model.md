---
title: Client Concurrency Model – Sharded Queue & Direct Paths
status: accepted
ADR: 0020
date: 2025-07-28
supersedes: 0008-concurrency-model
---
# Context

ADR-0008 introduced a *SQLite-backed* concurrency model with a 256-shard worker pool.  Since then we have:

* Dropped the local SQLite mirror (ADR-0018).
* Renamed the await primitive to `await_consistency` (ADR-0019).
* Adopted Cloud-Run fan-out + Spanner for storage.

The original ADR therefore no longer reflects reality.

# Decision

Adopt a **three-class** client concurrency model:

| Class | API Verbs | Ordering Guarantee | Execution Path |
|-------|-----------|--------------------|----------------|
| Ordered (SQ) | `add_entry`, `delete_entry`, `await_consistency`, `Get*/List*` with `Consistent=true` | FIFO per memory | Sharded Queue → 4 worker goroutines |
| Eventual Reads (Direct) | `get_entry`, `list_entries`, `search_entries` without `Consistent=true` | None (eventual) | Direct gRPC/HTTP |
| Admin Strong (Direct) | `create_user`, `create_memory`, metadata updates, list APIs | Strong (backend enforced) | Direct gRPC/HTTP |

Key points:

1. **Per-client FIFO** – SQ guarantees the caller's own writes are ordered even when issued concurrently or offline.
2. **Offline resilience** – queued writes are retried when connectivity returns.
3. **Back-pressure** – queue length and worker count bound local resources; no unbounded goroutine storm.
4. **Stateless servers** – backend does not need a sequencer; Spanner provides global external consistency.

See `docs/design/client_concurrency_model.md` for detailed rationale and comparison with Mem0s.

# Consequences

* ADR-0008 is marked **superseded**.
* Implementation lives in `internal/shard` (SQ) and SDK option plumbing.
* Design doc updated; no other API changes. 