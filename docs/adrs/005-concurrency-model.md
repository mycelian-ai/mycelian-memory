# ADR-005: Concurrency & Replication Model

**Status**: Accepted
**Date**: 2025-07-25

## Context

Memory service processes mixed workloads from humans (CLI) and AI agents (MCP stdio).
Requirements:
- Per-memory ordering for writes (entries, deletes)
- High parallelism across memories
- Fast local-first durability
- Bounded resource usage & back-pressure

## Decision

### 1. Sharded Worker Pool
- `N = 256` shards, each a goroutine with bounded channel (`queueLen=128`)
- `shard = fnv32(memoryID) % N`
- All jobs for the same memory run FIFO on the same shard

### 2. Job Types
| Job | Path | Queue? | Notes |
| --- | --- | --- | --- |
| `add_entry` | Agent / CLI | YES | Local insert → mark `synced=0` |
| `list_entries` | Agent / CLI | YES | Ensures read sees prior writes |
| `get_entry` | Agent / CLI | YES | Single-row read with ordering |
| `put_context` | Agent / CLI | YES | Writes/updates context fragments |
| `get_context` | Agent / CLI | YES | Reads after previous writes |
| `delete_entry` | Agent / CLI | YES | Enqueued; use `await_consistency` barrier when needed |
| `create_user` | Human CLI | Priority queue |
| `create_memory` | Human CLI | Priority queue |
| `list_memories` | Human CLI | NO | Direct backend read |

### 3. Replication Loop
- Each shard owns background `drain()` that retries unsynced rows until backend confirms
- Marks `synced=1` after successful replication

### 4. Back-pressure
- When shard queue full, `Submit` blocks for timeout period
- Returns `ErrQueueFull` on timeout for 429 response

### 5. Await Commit Primitive
- Polls local store until `synced=0` count is zero for given memory

## Consequences

### Positive Consequences
- Guarantees per-memory write ordering without global locks
- Fast ACKs via local persist; network issues handled in background
- Reads execute on same shard ensuring consistency with prior writes
- Human CLI operations use priority queue avoiding agent congestion

### Negative Consequences
- Increased complexity in client implementation
- Memory usage grows with number of shards and queue depth

## Alternatives Considered

### Alternative 1: Global Mutex Per Memory
**Why rejected**: Hotspot risk, eliminates parallelism benefits

### Alternative 2: Database Serializable Transactions
**Why rejected**: Queue contention, higher latency impact

### Alternative 3: gRPC Stream Per Memory
**Why rejected**: Additional infrastructure complexity not needed for current scale
