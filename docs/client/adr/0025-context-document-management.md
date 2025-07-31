# 0025 – Context Document Management (Filesystem + Shard Queue)

*Status: ✅ Accepted – 2025-07-29*

## Context
Agents need to read and update a **memory-level context document** separate from individual entry writes.  For the MVP we want a solution that is:

* Idiomatic Go, no external services.
* Fast round-trip for the agent.
* Consistent with the existing client concurrency model.
* Easy to migrate to a database table later.

## Decision
Store the context document (arbitrary **raw text**; agents will treat it as Markdown via prompt rules) on disk, one file per memory, and integrate the write path with the existing `ShardExecutor` to preserve FIFO ordering:

| Layer | Responsibility |
|-------|----------------|
| **MCP tool** (`put_context`, `get_context`) | Thin wrappers; delegate to Go SDK |
| **Go SDK** | Owns atomic write/read logic, queue submission, back-pressure |
| **ShardExecutor** | Guarantees per-memory FIFO and retries |
| **Filesystem** | `$CONTEXT_DATA_DIR/<memory_id>/context.md` |

### Write algorithm (executed inside shard worker)
1. `tmp := path + ".tmp"`
2. `WriteFile(tmp, content)`
3. `fsync(tmp)`
4. `rename(tmp → path)`  *(atomic switch)*
5. `fsync(parentDir)`

#### Interaction with `add_entry`
When the same worker later processes an `add_entry` job for the memory it:
1. Reads `context.json`. If the file is **missing** or **malformed** it first initialises it with `{ "activeContext": "" }` using the same atomic write sequence.
2. Embeds the raw JSON bytes into the outgoing `AddEntryRequest.Context` field so every entry carries the latest snapshot.
3. Sends the HTTP POST to the backend.

Because `put_context` and `add_entry` jobs share the per-memory queue, any context update enqueued **before** an entry is guaranteed to be written first, so the entry always contains an up-to-date snapshot.

### Read algorithm (`get_context`)
Plain `ReadFile(path)` returns the **raw text**; rename ensures readers never see partial data.

### Consistency Model
* Default: **eventual** – the agent gets an "enqueued" acknowledgement immediately.
* Strong read-after-write: call `AwaitConsistency(ctx, memoryID)` before `get_context`.

## Consequences
* Latency: sub-millisecond ack, disk I/O happens asynchronously.
* Ordering: context updates interleave correctly with entry writes.
* Durability: fsync on file + directory ensures crash-safe persistence.
* Migration path: backend can later move to `memories_context_log` table (see ADR-0024); SDK just swaps out the writer implementation.

## Alternatives Considered
* **Synchronous write** – blocks agent, hurts latency.
* **Store in entries table** – polluted query semantics, removed.
* **CRDT/patch ops** – overkill for single-writer model.
* **Compression** – deferred until we profile real workloads.

## Follow-up Work
1. Implement `PutContext` / `GetContext` in Go SDK.
2. Add `ContextHandler` & register tools in MCP server.
3. Integration tests + CLI helper (follow-up PR).
4. When backend stores context in its own table, add HTTP endpoints and deprecate on-disk store. 