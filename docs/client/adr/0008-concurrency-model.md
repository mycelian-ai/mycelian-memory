---
adr: 0008-concurrency-model
status: accepted
date: 2025-07-25
---
# Concurrency & Replication Model

## Context
Synapse Memory processes mixed workloads from humans (CLI) and LLM agents (MCP stdio).
We need:
• Per-memory ordering for writes (entries, deletes).  
• High parallelism across memories.  
• Fast local-first durability via SQLite.  
• Bounded resource usage & back-pressure.

## Decision
1. **Sharded worker pool**  
   • `N = 256` shards, each a goroutine with a bounded channel (`queueLen=128`).  
   • `shard = fnv32(memoryID) % N`.  
   • All jobs for the same memory run FIFO on the same shard.

2. **Job types**  
   | Job                | Path                     | Queue? | Notes |
   | ------------------ | ------------------------ | ------ | ----- |
   | `add_entry`        | Agent / CLI             | YES    | Local insert → mark `synced=0` |
   | `list_entries`     | Agent / CLI            | YES    | Ensures read sees prior writes; handler waits for job result |
   | `get_entry`        | Agent / CLI            | YES    | Single-row read; same ordering guarantee |
   | `put_fragment`     | Agent / CLI            | YES    | Writes/updates context fragments |
   | `get_fragment`     | Agent / CLI            | YES    | Reads fragment after previous writes |
   | `manifest`         | Agent / CLI            | YES    | Returns fragment list, needs ordering |
   | `delete_entry`     | Agent / CLI            | YES    | Enqueued; `await_commit` can block until replicated |
   | `create_user`      | Human CLI               | **Priority queue** (`priorityCh`) |
   | `create_memory`    | Human CLI               | **Priority queue** |
   | `list_memories`    | Human CLI               | NO     | Direct backend read; low traffic |

3. **Replication loop**  
   • Each shard owns a background `drain()` that retries unsynced rows until Spanner confirms, then marks `synced=1`.

4. **Back-pressure**  
   • When shard queue is full, `Submit` blocks for `X ms`; on timeout returns `ErrQueueFull` so the handler can respond 429.

5. **await_commit primitive**  
   • Polls SQLite until `synced=0` count is zero for the given memory.

## Consequences
+ Guarantees per-memory write ordering without global locks.  
+ Fast ACKs via local persist; network hic-ups handled in the background.  
+ Reads (`list_entries`) now execute on the same shard ensuring they observe all prior writes.  
+ Deletes follow the same per-memory ordering; callers can `await_commit` if they need global visibility.  
+ Human CLI ops (`create_user`, `create_memory`) still use the priority queue and avoid agent congestion.

## Alternatives Considered
| Alternative | Reason Rejected |
| --- | --- |
| Global mutex per memory | Hotspot risk, no parallelism. |
| Database serializable TX | SQLite queue contention, higher latency. |
| GRPC stream per memory | More infra, not needed for current scale. |

_Status: Accepted – implementation in progress (Milestone 2B)_ 