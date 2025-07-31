---
adr: 0009-delete-queue-integration
status: accepted
date: 2025-07-25
supersedes: 0007-delete-flow-sync
---
# Queued, Synchronous Deletes

## Context
ADR-0007 switched to immediate, synchronous deletes (local TX → remote delete → commit).  It assumed deletes bypass the per-memory shard queue.  After formalising the concurrency model (ADR-0008) we realised ordering issues:

Example: `add1, add2, read, delete, add3`.
If `delete` skips the queue it could overtake inflight adds, producing inconsistent reads.

## Decision
1. **Delete jobs are enqueued on the same per-memory shard queue as writes.**
2. Inside the queued job we still perform the **synchronous delete**:  
   • BEGIN TX (local) → remote delete → COMMIT / ROLLBACK.
3. Caller receives success only after COMMIT, preserving the semantics of ADR-0007.
4. If a client needs global visibility it can call `await_commit` after the delete, same as for writes.

## Consequences
+ Maintains strict FIFO ordering for all mutating operations (`add`, `delete`).  
+ Removes race where delete could leapfrog preceding writes.  
+ Queue latency is negligible; deletes are rare and human-triggered.

## Migration
No schema change.  Implementation: remove the bypass path, route `delete_entry` through `pool.Submit(...)`.

_Status: Accepted – queues will handle deletes in upcoming worker-pool implementation._ 