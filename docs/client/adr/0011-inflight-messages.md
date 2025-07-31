---
description: Provide MCP tools to list and retrieve unsynced (inflight) entries for local-only usage
---
# ADR-0011: Inflight Entry Retrieval Tools

| Status | Date | Deciders |
| ------ | ---- | -------- |
| ✅ Accepted | 2025-07-26 | @core-team |

## Context

Milestone 2 introduces a 256-shard worker pool and an **async replication** model (see [ADR-0008](0008-concurrency-model.md)).  Writes are ACKed as soon as the local SQLite transaction commits, while a background replicator flushes them to the cloud backend. Until replication completes the rows have `synced = 0` and therefore:

* They are **not** visible to other devices.
* They are **not** returned by the existing `get_entries`, `get_top_k`, or `search` APIs (which intentionally reflect the authoritative, committed state).

However, an on-device LLM / agent often needs *immediate* access to the text it just wrote in order to evolve the conversation without waiting for replication.

## Decision

Introduce two **read-only** MCP tools & CLI commands:

```
list_inflight_entries
get_inflight_entry
```

**Signature**
```
Arguments:
  user_id   (required, string, UUID) – owner of the memory
  memory_id (required, string, UUID) – target memory
  limit     (optional, int, 1-50, default 25) – max rows
```

### list_inflight_entries
* SELECT rows from the local mirror `entries` table where `memory_id = ? AND synced = 0`
* Order **chronologically** by `creation_time` (ASC)
* Return at most `limit` rows **without** pagination cursors (MVP)

**Payload**
```jsonc
{
  "entries": [
    {
      "localId": "pending-1721476912345", // NOT a canonical entryId
      "creationTime": "RFC3339Nano",
      "rawEntry": "string",
      "summary": "string",
      "tags": {"k":"v"}
    }
  ],
  "count": 3,
  "applied_limit": 25
}
```

`localId` is derived from the local SQLite `rowid` (prefixed with `pending-`) to make it explicit that the row is provisional and *will be replaced* by a canonical `entryId` once replicated.

### get_inflight_entry

When a client needs the full details of a single unsynced row:

**Signature**
```
Arguments:
  user_id   (required, string, UUID)
  memory_id (required, string, UUID)
  local_id  (required, string, "pending-<rowid>") – provisional ID returned by list_inflight_entries
```

**Behaviour**
* SELECT the row where `memory_id = ? AND rowid = ? AND synced = 0`.
* If found, return HTTP 200 with payload below.
* If the row is already synced (`synced = 1`) return HTTP 404 with error code `already_committed` – callers should switch to `get_entry` with the canonical `entryId`.
* If not found, return HTTP 404.

**Payload (200 OK)**
```jsonc
{
  "entry": {
    "localId": "pending-1721476912345",
    "creationTime": "RFC3339Nano",
    "rawEntry": "string",
    "summary": "string",
    "tags": {"k":"v"}
  }
}
```

## Consequences

* **LLMs** can stitch recent, unsynced context immediately via `list_inflight_entries` without sacrificing the correctness guarantees of `get_entries`.
* Existing APIs remain unchanged; downstream systems continue to operate only on committed data.
* The replicator, upon successful commit, will:
  1. Replace the provisional `localId` with the authoritative `entryId` (UUIDv4) in the local DB.
  2. Flip `synced = 1`.
* No additional ordering guarantees are needed—rows already respect per-memory FIFO via the shard queue.
* Future versions may add pagination cursors or per-message status (`replicating`, `failed`, etc.) based on user feedback.

## Alternatives considered
| Option | Reason Rejected |
| ------ | --------------- |
| Return unsynced rows in `get_entries` | Violates mental model that API reflects globally consistent state. |
| Wait for replication before ACK | Fails latency requirement (< 5 ms). |
| Use same UUID space as committed entries | Blurs provenance; complicates deduplication. |

## References
* [ADR-0008](0008-concurrency-model.md) – Worker & replication model
* [Client concurrency model design](../design/client_concurrency_model.md) 