---
description: Defines the read-side consistency guarantees, queue strategy, and inflight entry optimisation.
---
# ADR-0012: Read Consistency Model & Inflight Entry Optimisation

| Status | Date | Deciders |
| ------ | ---- | -------- |
| ✅ Accepted | 2025-07-26 | @core-team |

## Context

ADR-0008 set the **write-side** concurrency foundation (256 shard FIFO queues) and declared that *reads* are generally enqueued to observe prior writes.  Since then we have:

* Added async replication to a cloud backend (SQLite ➜ Spanner).
* Introduced new APIs to surface **unsynced** rows: `list_inflight_entries` and `get_inflight_entry` (ADR-0011).

These changes require a **holistic read-consistency story** so SDK users understand what each API guarantees and which operations bypass the queue.

## Consistency Levels Offered

| Level | API Surface | Visibility | Ordering Guarantees | Typical Use |
| ----- | ----------- | ---------- | ------------------- | ------------ |
| **Committed** (global) | `list_entries`, `get_entry`, `search`, `get_top_k` | Only rows with `synced = 1` (replicated to backend) | FIFO per memory (via shard queue) | Cross-device UX, analytics, AI context requiring durable history |
| **Inflight** (local) | `list_inflight_entries`, `get_inflight_entry` | Local rows with `synced = 0` | None beyond local commit | Agent self-reflection, UI progress bars |

> Note: callers wanting *both* views must merge the two result sets client-side.

## Decision

1. **Queue Strategy**
   • Committed reads (`list_entries`, `get_entry`, etc.) stay *inside* the shard queue so they see all prior writes.
   • Inflight reads bypass the queue and execute a **direct SQLite SELECT** – latency <0.2 ms.

2. **API Contract**
   • No single API surfaces both inflight and committed rows; this keeps mental models clean.
   • Inflight IDs follow the `pending-<rowid>` scheme and are **not stable** after replication.

3. **Consistency Caveats**
   • An inflight read may miss a write that is still queued but not yet committed (<1 ms window). This is acceptable because inflight APIs are explicitly *best-effort/local*.
   • Committed reads + `await_commit` give the strongest guarantee: once the call returns, all prior writes are visible to every device.

### Updated Job Matrix
| Job | Queue? | Consistency Target |
| --- | ------ | ------------------ |
| `add_entry`, `delete_entry` | YES | Ensure ordering before ACK |
| `list_entries`, `get_entry`, `search`, `get_top_k` | YES | Observe prior writes (committed) |
| `list_inflight_entries`, `get_inflight_entry` | **NO** | Fast local view of unsynced rows |

## Consequences

+ **Transparency**: Users choose the level of consistency vs. latency they need.
+ **Performance**: Critical UI/agent paths can access unsynced data almost immediately without locking up the queue.
+ **Scope Control**: We explicitly *disallow* cross-device read-your-write via inflight APIs; users must call `await_commit` + committed reads instead.

## Alternatives Considered
| Option | Reason Rejected |
| --- | --- |
| Expose unsynced rows in committed APIs | Breaks mental model and complicates pagination/cursors |
| Queue all reads | Unnecessary latency for purely local inflight access |

## References
* [ADR-0008](0008-concurrency-model.md) – Sharded worker pool
* [ADR-0011](0011-inflight-messages.md) – Inflight entry API surface 