# 0023 – Multi-Agent Memory Architecture

*Status: ✅ Accepted – 2025-07-29*

## Context
Today the Memory service is optimised for **one writer per memory**.  
Upcoming use-cases (support swarming, co-pilots, human + bot pairs) require **concurrent agents** to operate on the *same* customer issue while preserving:

* zero write-contention during the live session, and
* a single authoritative history for retrieval and audit after the session.

## Decision
We will implement a **two-tier model**:

1. **Uber-Case memory**  
   • Shared, append-only log that stores concise decision / milestone summaries.  
   • Agents append only; no edits or deletes.
2. **Per-agent memory**  
   • Private workspace for each agent (`memory-<agent_id>`).  
   • Agents write raw entries and maintain their own context here.
3. **Session ID column**  
   • `entries.session_id UUID NULL` stored on every write for audit analytics.
4. **Checkpoint protocol**  
   • Each agent records the `last_entry_id` it has read from the Uber-Case.  
   • On resume it calls `GET /entries?after=<checkpoint>` to pull recent summaries.
5. **Batch AddEntries**  
   • Up to 20 entries per call, optional embedded context.  
   • Shared between all memories; remains synchronous JSON.

This model ships in two phases:

* **MVP** – single-agent support, session_id column nullable.  
* **Post-MVP** – multi-agent best-practices docs; no backend merge logic needed.

## Consequences
* Zero live contention – agents never edit the same context file.  
* Retrieval spans *{own memory ∪ Uber-Case}* by default; additional memories optional.  
* Simple mental model for customers: private scratch pads + one canonical log.  
* Minor schema migration (add session_id).  
* Future push notifications (SSE/WebSocket) can stream only new Uber-Case entry stubs.

## Alternatives Considered (and rejected)
* **Leases / locks** – risk of poor UX under latency; discarded.  
* **409 conflict + retry** – would spam users with errors in concurrent chats.  
* **CRDT / patch-based context** – deterministic but too heavy for LLM agents and MVP scope.  
* **Branch-per-session** – good isolation but requires server-side merge; postponed until a real need surfaces.

## Follow-up Work
* Client SDK helper `CheckpointAndSummarise()` to enforce Uber-Case updates.  
* Docs: multi-agent usage guide.  
* Monitor Uber-Case summary length; archive closed cases. 