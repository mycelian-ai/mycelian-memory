---
adr: 0007-delete-flow-sync
status: accepted
date: 2025-07-25
supersedes: 0006-deletion-retention
---
# Synchronous Delete Flow for Entries and Memories

## Context
ADR 0006 introduced an explicit delete endpoint with a 24-hour grace period and a `pending_delete` flag in SQLite. Implementation spikes showed the grace window added complexity (dual-state rows, sweeper jobs) without meaningful safety gains. Human users deleting data typically want **immediate** removal, and the CLI already asks for confirmation.

## Decision
1. **Immediate, synchronous delete**
   • Endpoint remains `POST /api/users/{userId}/memories/{memoryId}/entries/{entryId}:delete` (same shape).  
   • The MCP server deletes the row in Spanner *within the same request*.
2. **Local-first transactional strategy**
   • The CLI / SDK starts a SQL transaction on the local SQLite mirror.  
   • Step-1: `DELETE FROM entries WHERE id = ?` removes the row locally.  
   • Step-2: Call MCP → Spanner to delete the remote row.  
   • On MCP success → `COMMIT`; on error → `ROLLBACK` (row re-appears locally).
3. **No `pending_delete` column, no sweeper, no grace period.**
4. **Durability helper**
   • `await_commit` command/tool still exists but now only waits for *pending writes*; deletes finish in-line.

## Consequences
+ Simpler codebase: one code path, no dual states.  
+ Users see immediate disappearance of deleted entries both locally and remotely.  
+ Failure modes are well-defined via the surrounding transaction; no torn replicas.  
+ Removes the need for extra sweeper jobs and `pending_delete` flag.

_Status: Accepted – supersedes ADR 0006_ 