# Multi-Agent Memory Architecture (Draft)

_Status: ⏳ In-progress • Target: post-MVP_

## 1  Goals
• Enable multiple autonomous agents or humans to work concurrently on the same customer issue without write contention.  
• Preserve a single authoritative case history for audit and retrieval.  
• Keep MVP complexity low; allow incremental evolution.

## 2  Key Concepts
| Concept | Purpose |
|---------|---------|
| **Uber-Case Memory** | Shared, append-only log that stores high-level decisions / summaries. Acts as the authoritative record. |
| **Per-Agent Memory** | Private workspace for each agent; free-form context updates and raw entries. |
| **Session (first-class)** | Represents one logical connection (browser tab, agent process). Entries now carry `session_id` for analytics and future conflict detection. |
| **Checkpoint** | An entry ID in Uber-Case marking the last summary an agent consumed. Agents fetch `entries?after=checkpoint` on resume. |

## 3  Write Path
1. Agent writes raw+summary to its **Per-Agent Memory** (batch endpoint, ≤20).  
2. When a milestone/decision occurs, it appends a concise summary to **Uber-Case** via the same batch endpoint (blocking).  
3. Server returns `context_version` and new `entry_id`; agent stores them as its checkpoint.

## 4  Read Path
• Hybrid search queries span _{own memory ∪ Uber-Case}_ by default.  
• For multi-agent reasoning, an agent may also include other agents' memories.

## 5  Schema Extensions
```sql
ALTER TABLE entries ADD COLUMN session_id UUID NULL;
```
Session IDs are minted by the server on `GET /context` and echoed by the client in subsequent writes.

_No additional schema needed for Uber-Case; it's just another memory record._

## 6  Operational Guidelines
• Keep Uber-Case summaries ≤ 512 chars.  
• Agents may **append only** to Uber-Case—no edits/deletes.  
• Reconcile conflicting summaries by writing a new entry that references superseded `entry_id`s.

## 7  Future Enhancements
1. **Event Stream** – SSE/WebSocket to push new Uber-Case entry stubs.  
2. **Automated Merge Bot** – aggregates key facts from per-agent memories into Uber-Case.  
3. **Branch-Per-Session** – optional path if shared context edits become necessary.

## 8  Alternatives Considered (and defered)
• **Short Leases** – one-minute context lock; rejected as fragile under network latency.  
• **Full-document Conflict (409)** – would cause poor UX during live support.  
• **CRDT/Patch-Based Context** – deterministic but too heavy for LLM agents in MVP.  
• **SSE in MCP Protocol** – protocol currently synchronous; adding streams postponed.

## 9  Risks & Mitigations
| Risk | Mitigation |
|------|-----------|
| Agents forget to append to Uber-Case | Add client SDK helper `CheckpointAndSummarise()`; monitor missing-summary alerts. |
| Uber-Case bloat | Enforce 512-char summaries; archive closed cases. |
| Read latency across many memories | Cache recent Uber-Case summaries in vector DB.

---
_Created: 2025-06-23 • Author: AI assistant • Reviewers: TBD_ 