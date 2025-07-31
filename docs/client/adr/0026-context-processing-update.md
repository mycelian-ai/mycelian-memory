# 0026 – Backend Context Processing Update

*Status: ✅ Accepted – 2025-07-30*

## Context
The initial MVP (ADR-0025) stored the **memory-level context document** exclusively on the local filesystem and propagated snapshots by embedding them in every `AddEntry` request.  This design ensured FIFO ordering via the `ShardExecutor`, but had several drawbacks:

* Context visibility required at least one entry write and a `ListEntries` read.  Multi-agent scenarios caused stale reads.
* The filesystem copy was the only system-of-record; backend queries could not retrieve the latest snapshot.
* `AddEntry` payloads were inflated by the context blob.

Meanwhile ADR-0024 introduced simple *put/get* HTTP endpoints but kept the filesystem path for ordering.  With the v 3 Search & Context APIs now available server-side, we can move the authoritative store to the backend while retaining the client-side executor ordering.

## Decision
1. **Context API (v 3)**
   * `PUT /api/users/{userId}/memories/{memoryId}/contexts` – *PutContext* writes a new snapshot row (append-only, last-write-wins).
   * `GET /api/users/{userId}/memories/{memoryId}/contexts` – returns the latest snapshot or 404.

2. **Go SDK changes**
   * New synchronous helper `putContextHTTP` (internal) and exported wrapper `Client.PutContext` that enqueues the HTTP write on the **same `ShardExecutor` shard** as `AddEntry`.  Ordering is therefore:
     `… AddEntry₁, … AddEntryₙ, PutContext → backend`.
   * `Client.GetLatestContext` replaces the old `GetContextFromBackend` ListEntries hack.
   * `AddEntryRequest` no longer includes a `context` field; backend will reject it.

3. **MCP Tool semantics remain identical** (`put_context`, `get_context` → SDK methods).  Prompts do **not** change.

4. **No local cache** – The job *only* performs the backend `PUT`.  The context snapshot exists:
   • in-memory while queued, and
   • in the backend table after the HTTP 201 response.
   Writing to a local file is deliberately omitted to avoid stale or divergent copies and to keep the design simple.

## Consequences
* All agents, regardless of host, observe the same latest snapshot with a single 1-RTT GET.
* Entry payloads shrink; backend storage of context moves to dedicated table, decoupling access patterns.
* The ShardExecutor continues to guarantee write-after-write consistency for context vs. entries **even though the full blob sits in memory while queued** (monitored as a future optimisation target).
* ADR-0025 (Filesystem-only Context Document Management) is ❌ **superseded** by this ADR.
* ADR-0024 (Context Read/Write APIs – No Versioning) remains valid; this ADR refines its endpoint paths and async ordering model.

## Alternatives Considered
* **Fully synchronous `PutContext`** – rejected (agent latency).  Async queue keeps UX snappy while preserving order.
* **Keep context in entries** – redundant after dedicated API; wastes bandwidth.
* **CRDT / merge patches** – unnecessary with single-writer assumption.

## Follow-up Work
1. Remove `Context` from `AddEntryRequest` and associated marshalling logic.
2. Update `internal/handlers/context_handler.go` to call SDK `PutContext` / `GetLatestContext`.
3. Delete `GetContextFromBackend` once all callers migrate.
4. Add negative tests: `AddEntry` with `context` → expect HTTP 400.
5. Track queue memory utilisation for large context blobs; revisit staging-file optimisation if needed. 