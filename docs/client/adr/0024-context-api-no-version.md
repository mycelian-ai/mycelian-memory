# 0024 – Context Read/Write APIs (No Versioning)

*Status: ✅ Accepted – 2025-07-29*

## Context
The MVP requires agents to load and persist the **context document** independently of individual entry writes.  Previous proposals included version numbers and conflict handling, but we later simplified multi-agent strategy to **one writer per memory** (per-agent memories + Uber-Case).

Therefore we can eliminate versioning and rely on a simple *last-write-wins* model.

## Decision
Introduce two new MCP tools backed by HTTP endpoints:

| Tool | HTTP | Behaviour |
|------|------|-----------|
| `mcp_synapse-memory_get_context` | `GET /memories/{id}/context` | Returns the most-recent context markdown row. 404 if none. |
| `mcp_synapse-memory_update_context` | `PUT /memories/{id}/context` | Inserts a new context row with full markdown; always succeeds. |

Data model (append-only log)
```sql
CREATE TABLE memories_context_log (
  memory_id   TEXT,
  session_id  UUID,
  updated_at  TIMESTAMP DEFAULT now(),
  markdown    TEXT,
  PRIMARY KEY (memory_id, updated_at DESC)
);
CREATE INDEX idx_ctx_mem_time ON memories_context_log (memory_id, updated_at DESC);
```

Key points
* **No version column, no 409 conflicts.** The newest `updated_at` row is authoritative.
* Payload carries the **entire** context document each time (≤5 000 chars by rule).
* `session_id` is required for future analytics; backend stores it but does not enforce uniqueness.

## Consequences
* Implementation is trivial—single row insert / latest row select.
* Agents flush context only when it changes (enforced by prompt rule) or on shutdown, minimising churn.
* Multi-agent scenarios remain safe because each memory has one writer; Uber-Case summaries are append-only.
* If future requirements demand conflict detection, a `version` column can be added via a backward-compatible migration.

## Alternatives Considered
* **Versioned optimistic concurrency** – rejected as unnecessary after per-agent memory simplification.
* **Short leases / locks** – added complexity; discarded.
* **CRDT or patch ops** – overkill for one-writer model.

## Follow-up Work
* Implement server handlers and SDK methods.  
* Update prompts: startup `get_context` + conditional `update_context` rule.  
* Add integration tests (happy path + 404 when context missing). 