---
adr: 0006-deletion-retention
status: accepted
date: 2025-07-25
---
# Deletion & Retention Policies for Entries and Memories

## Context
Early API designs allowed callers to specify an `expirationTime` when creating an entry. This posed an **accidental data-loss risk**: a typo or mis-calculated timestamp could permanently delete valuable history. We need a safer, predictable model while still giving users (humans) a way to remove data.

## Decision
1. **Remove `expirationTime` from the `CreateEntry` API.**  
   Entries are created without any embedded TTL.
2. **Introduce an explicit delete endpoint** (human/CLI scope only):  
   `POST /api/users/{userId}/memories/{memoryId}/entries/{entryId}:delete`  
   • No body or parameters.  
   • Backend immediately flags the entry as `pending_delete`, hiding it from all synchronous read/search APIs.  
   • Physical row removal happens asynchronously after a fixed **24 h grace window**.
3. **Grace-period guarantees**  
   • Users can build an (optional) "undo" admin tool during the window.  
   • Prevents race conditions with replication / caching layers.
4. **MCP / SDK surface**  
   • CLI command `synapse delete-entry --user-id … --memory-id … --entry-id …`.  
   • No MCP tool for agents – deletion is a human action.  
5. **Local SQLite mirror**  
   • Table `entries` gains `pending_delete INTEGER DEFAULT 0`.  
   • On delete ack MCP sets `pending_delete = 1`. Local sweeper purges the row once the backend confirms hard delete.
6. **Commit guarantee**  
   • New MCP tool & CLI command `await_commit` blocks until all queued writes/deletes are acknowledged.  
   • Capture-rules default instruct LLM agents to call `await_commit` before ending a turn.

## Consequences
+ Eliminates accidental premature deletion via bad timestamps.  
+ Simple, predictable user model: *create → read → explicit delete.*  
+ Backend keeps storage tidy via grace-period sweeper; no user-provided dates to validate.  
+ `await_commit` ensures durability semantics for agents.

_Status: Accepted – implementation scheduled for Milestone 2_ 