## Conversation Time – Implementation Plan (longmemeval-benchmarker)

### TL;DR
- **Goal**: Define, instrument, compute, and report conversation-time metrics per session/question and per run, without altering benchmark semantics.
- **Output**: Persisted metrics in `progress.db` (UTC), per-session metrics in `out/run_*/hypotheses.jsonl`, and a run-level summary artifact.
- **Approach**: Start with clear definitions and post-hoc validation from logs, then add targeted instrumentation at the runner level.

### Scope
- **In-scope**: Definitions, instrumentation points in the runner/orchestrator, storage schema shaping, reporting, validation, and rollout.
- **Out-of-scope**: Changing agent behavior, model parameters, dataset content, or server-side APIs.

### Non-Goals
- No change to how messages are authored or persisted beyond timestamps/metrics.
- No introduction of environment-variable switches; behavior is driven by runner configuration/flags only.

### Definitions (all timestamps UTC)
- **Conversation window**:
  - `conversation_start_at`: Timestamp of the first persisted conversation message for a session after the question begins (user or assistant), or the first agent invocation start—whichever occurs first and is observed by the runner.
  - `conversation_end_at`: Timestamp when the session reaches a terminal state (DONE/FAIL/CANCEL), defined as the timestamp of the last persisted conversation message leading to finalization if available; otherwise, the terminal event time recorded by the runner.
  - `conversation_elapsed_ms = conversation_end_at - conversation_start_at`.
- **Active time**:
  - Sum of assistant invocation active durations observed by the runner (model request/stream spans, tool resolution spans) across the session.
  - `conversation_active_ms = Σ(invocation_active_ms + tool_active_ms)`.
- **Idle time**:
  - Time inside the window not accounted as active.
  - `conversation_idle_ms = conversation_elapsed_ms - conversation_active_ms` (floored at 0).
- **Turns**:
  - Count of persisted conversation messages in the session (assistant or user) that are part of the solution attempt.

Edge cases to define precisely during validation:
- Sessions with zero assistant invocations (report `active_ms = 0`).
- Finalization without a last message (use terminal event timestamp).
- Retries/backoffs: count only time when a live request is in-flight as active; backoff delays are idle.
- Concurrent spans: do not double-count overlapping active intervals.

### Data Sources
- Runner/orchestrator logs (e.g., events like `invoker_start`, `invoker_flush`, tool timing).
- Runner in-memory timing spans around assistant/model calls and tool operations.
- `progress.db` (SQLite) session/message rows and timestamps (stored in UTC).

### Instrumentation Plan (runner-side only)
1. **Boundary markers**
   - Start: on the first persisted conversation message for the session or the first assistant invocation start—whichever occurs first.
   - End: on terminal session state (DONE/FAIL/CANCEL); prefer the timestamp of the final persisted message if present, else the terminal event time.
2. **Active spans**
   - Assistant invocation span: from request dispatch to final token/stream completion.
   - Tool span(s): from tool-call dispatch to completion per tool.
   - Merge spans to avoid double counting (interval union).
3. **Turn counting**
   - Count persisted conversation messages (assistant/user) within the window.
4. **Clock discipline**
   - Use monotonic timers for durations; attach wall-clock UTC timestamps for persistence.

### Storage & Schema (progress.db)
- Store per-session metrics:
  - `conversation_start_at` (UTC ISO-8601)
  - `conversation_end_at` (UTC ISO-8601)
  - `conversation_elapsed_ms` (INTEGER)
  - `conversation_active_ms` (INTEGER)
  - `conversation_idle_ms` (INTEGER)
  - `conversation_turns` (INTEGER)
- Optional: per-invocation spans kept ephemeral in logs; avoid schema bloat unless needed for audits.
- All timestamps stored in UTC; ensure consistency with `last_progress_at` and existing runner updates.

### Reporting
- Per-session: extend `out/run_<id>/hypotheses.jsonl` entries with the conversation metrics.
- Run summary: emit `out/run_<id>/summary.json` with aggregates: mean/median/p95 for elapsed/active/idle, distribution of turns, and counts by terminal state.
- Logs: one-line summary per session at completion, e.g., `CONV_TIME total=..., active=..., idle=..., turns=..., status=...`.

### Validation Plan (pre-instrumentation and post)
- Phase 0: Post-hoc compute from existing logs to validate definitions, compare against runner-observed windows, and tune boundary rules.
- Phase 1: With instrumentation, cross-check stored metrics against recomputation from logs on a sample of runs (tolerance ±1s).

### Rollout Plan
- Phase 0: Land definitions and doc-only plan (this file).
- Phase 1: Add runner instrumentation under a guarded code path; keep default behavior unchanged.
- Phase 2: Enable metrics persistence and JSONL reporting by default.
- Phase 3: Add run-level summary and dashboards (if desired).

### Risks and Mitigations
- **Boundary ambiguity** (start/end): Prefer persisted message timestamps; fall back to terminal event time; assert invariants in code.
- **Clock skew**: Use a single process clock; rely on monotonic deltas for durations.
- **Double counting active spans**: Interval-union merge of overlapping spans; test coverage on overlap cases.
- **Missing data**: If spans are missing, compute elapsed and turns; leave `active_ms` null/0 with a warning flag.

### Acceptance Criteria
- Metrics appear in `progress.db` and `out/run_*/hypotheses.jsonl` for every terminal session.
- Aggregates present in `summary.json`; values consistent with log-derived recomputation within tolerance.
- No >2% runtime overhead on the runner; no change in pass/fail outcomes.

### Open Questions
- Should backoff delays be counted as idle (proposed: yes)?
- Do we include tool network time as active (proposed: yes)?
- How to treat partial sessions on crash/restart (proposed: compute window on re-finalization only)?

# Conversation Time Implementation Plan

## Overview
Add `conversation_time` as a first-class timestamp to track when user/agent conversations actually occurred, distinct from `creation_time` which tracks when entries are stored in the database.

## Core Concept
- **`conversation_time`**: When the conversation between user and agent happened
- **`creation_time`**: When the entry was stored in Mycelian (DB-generated)
- Default: `conversation_time = CURRENT_TIMESTAMP` (for real-time conversations)
- Override: Can specify past `conversation_time` for historical imports

## Implementation Steps

### 1. Database Schema Update

**File: `server/internal/storage/postgres/schema.sql`**

Add `conversation_time` column after `creation_time` (line 33):
```sql
CREATE TABLE IF NOT EXISTS memory_entries (
  actor_id       TEXT NOT NULL,
  vault_id       TEXT NOT NULL,
  memory_id      TEXT NOT NULL,
  creation_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
  conversation_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,  -- NEW
  entry_id       TEXT NOT NULL,
  raw_entry      TEXT NOT NULL,
  summary        TEXT,
  metadata       JSONB,
  tags           JSONB,
  correction_time TIMESTAMPTZ,
  corrected_entry_memory_id TEXT,
  corrected_entry_creation_time TIMESTAMPTZ,
  correction_reason TEXT,
  last_update_time TIMESTAMPTZ,
  PRIMARY KEY (actor_id, vault_id, memory_id, creation_time, entry_id)
);
```

Add indexes for temporal queries (after line 47):
```sql
CREATE INDEX IF NOT EXISTS memory_entries_conversation_time_idx
  ON memory_entries(memory_id, conversation_time DESC);
CREATE INDEX IF NOT EXISTS memory_entries_temporal_range_idx
  ON memory_entries(vault_id, memory_id, conversation_time DESC);
```

### 2. Prompt File Updates

#### `client/prompts/default/chat/entry_capture_prompt.md`

Update line 21-24 to include conversation_time:
```markdown
3) Call `add_entry` once with:
   - raw_entry: full, unedited text
   - summary: generated per the summary prompt
   - conversation_time: when this conversation occurred (if provided, otherwise defaults to current time)
   - Optional tags: { "role": "user" | "assistant" }
```

Update examples (around line 38-43):
```markdown
Example add_entry call (user):
{
  "raw_entry": "Hi there, could you help me compare options for X?",
  "summary": "User asked for help comparing options for X.",
  "conversation_time": "2023-01-08T12:49:00Z",  // When conversation occurred
  "tags": { "role": "user" }
}
```

#### `client/prompts/default/chat/context_prompt.md`

**Critical**: This prompt instructs the agent how to BUILD context documents that include Timeline sections.

Update Timeline section (around line 20-21):
```markdown
- Timeline: Keep detailed timeline of events in `YYYY-MM-DD – event` format
- Use conversation_time from your conversation history (when each exchange with user occurred)
- Sort Timeline entries chronologically by conversation_time
- If user mentions dates/events within a conversation, include those as part of the description, not as the Timeline date
```

Update line 33:
```markdown
`# Timeline` - YYYY-MM-DD – succinct event description
```

Add new section after existing instructions:
```markdown
## Timeline Construction Rules

When building the Timeline section for context:

1. **Primary Date**: Use conversation_time (when you and user had the exchange)
   - This comes from system messages like "This conversation occurred at 2023-01-08T12:49:00Z"
   - Extract YYYY-MM-DD portion for Timeline

2. **Event Description**: Summarize what was discussed in that conversation
   - Include any dates/times the user mentioned as part of the description
   - Keep descriptions concise but include key entities and actions

3. **Handling Referenced Dates**:
   - If user says "I visited Paris on June 5th, 2022" during a conversation on 2024-07-05
   - Timeline entry: `2024-07-05 – User discussed Paris visit from 2022-06-05`
   - NOT: `2022-06-05 – User visited Paris` (this would incorrectly imply the conversation happened in 2022)

4. **Chronological Order**: Sort all Timeline entries by conversation_time (oldest first)

Example Timeline in well-formed context:
```
# Timeline
2023-01-08 – User visited MoMA and discussed modern art exhibition
2023-01-15 – User visited Met Ancient Civilizations exhibit
2024-07-05 – User recalled vacation to Paris from 2022-06-05
2024-07-06 – User planned upcoming trip to London for 2024-09-15
```

**Note**: The memory_contexts table does NOT have a conversation_time column. The Timeline is embedded in the context text content itself.

#### `client/prompts/default/chat/summary_prompt.md`

Add clarification after line 8:
```markdown
Note on timestamps:
- conversation_time: When the user/agent exchange occurred (use for temporal references)
- creation_time: When stored in database (system-generated)
Use conversation_time for all date references in summaries.
```

### 3. Go Code Updates

#### Server Storage Layer

**File: `server/internal/storage/postgres/adapter.go`**

Add to AddEntryRequest struct:
```go
type AddEntryRequest struct {
    ActorID          string
    VaultID          uuid.UUID
    MemoryID         string
    RawEntry         string
    Summary          string
    ConversationTime *time.Time  // NEW
    Metadata         map[string]interface{}
    Tags             map[string]string
}
```

Update INSERT statement (around line 383):
```sql
INSERT INTO memory_entries (
    actor_id, vault_id, memory_id, conversation_time,
    raw_entry, summary, metadata, tags, entry_id
)
VALUES ($1,$2,$3,COALESCE($4, CURRENT_TIMESTAMP),$5,$6,$7,$8,$9)
RETURNING creation_time, conversation_time
```

**File: `server/internal/store/postgres/postgres.go`**

Similar updates to struct and INSERT statement.

#### Client Library

**File: `client/client.go`**

Add to AddEntryRequest:
```go
type AddEntryRequest struct {
    RawEntry         string            `json:"raw_entry"`
    Summary          string            `json:"summary"`
    ConversationTime *time.Time        `json:"conversation_time,omitempty"`  // NEW
    Metadata         map[string]any    `json:"metadata,omitempty"`
    Tags             map[string]string `json:"tags,omitempty"`
}
```

#### MCP Server

**File: `mcp/internal/handlers/entry_handler.go`**

Update handleAddEntry (around line 64):
```go
func (eh *EntryHandler) handleAddEntry(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    vaultID, _ := req.RequireString("vault_id")
    memoryID, _ := req.RequireString("memory_id")
    rawEntry, _ := req.RequireString("raw_entry")
    summary, _ := req.RequireString("summary")

    // NEW: Parse conversation_time
    var conversationTime *time.Time
    if ct, ok := req.GetArguments()["conversation_time"].(string); ok && ct != "" {
        if parsed, err := time.Parse(time.RFC3339, ct); err == nil {
            conversationTime = &parsed
        }
    }

    var tags map[string]string
    if t, ok := req.GetArguments()["tags"]; ok {
        _ = mapstructureDecode(t, &tags)
    }

    // Pass to client
    ack, err := eh.client.AddEntry(jobCtx, vaultID, memoryID, clientpkg.AddEntryRequest{
        RawEntry:         rawEntry,
        Summary:          summary,
        ConversationTime: conversationTime,  // NEW
        Tags:             tags,
    })
    // ... rest of function
}
```

### 4. LongMemEval Benchmarker Updates

#### Dataset Loader

**File: `longmemeval-benchmarker/src/dataset_loader.py`**

Add date parsing and attachment:
```python
from datetime import datetime

def parse_date_to_iso(date_str):
    """Convert '2023/01/08 (Sun) 12:49' to '2023-01-08T12:49:00Z'"""
    # Remove day of week: "2023/01/08 (Sun) 12:49" -> "2023/01/08 12:49"
    date_part = date_str.split(' (')[0]
    # Parse the date
    dt = datetime.strptime(date_part, "%Y/%m/%d %H:%M")
    return dt.isoformat() + "Z"

def normalize_question(rec):
    # ... existing code ...

    # Get haystack_dates if available
    dates = rec.get("haystack_dates", [])

    # Parse sessions
    sessions_raw = rec.get("haystack_sessions", [])
    norm_sessions = []

    for idx, s in enumerate(sessions_raw):
        session = {
            "session_id": f"S{idx+1}",
            "messages": normalize_messages(s)
        }

        # Attach conversation_time if we have a date for this session
        if idx < len(dates):
            session["conversation_time"] = parse_date_to_iso(dates[idx])

        norm_sessions.append(session)

    return {
        "question_id": qid,
        "sessions": norm_sessions,
        # ... other fields
    }
```

#### Single Question Runner

**File: `longmemeval-benchmarker/src/single_question_runner.py`**

Pass conversation_time through pipeline (around line 345):
```python
# Process all sessions
for s_idx, s in enumerate(q.get("sessions", []), start=1):
    thread_id = f"{memory_id}:s{s_idx}"
    conversation_time = s.get("conversation_time")  # NEW

    runner_log.info("SESSION_START qid=%s s=%d memory_id=%s thread_id=%s conversation_time=%s",
                    qid, s_idx, memory_id, thread_id, conversation_time)

    invoker.start_session(thread_id)

    try:
        for msg_idx, m in enumerate(s.get("messages", []), start=1):
            role = (m.get("role") or "").strip().lower()
            content = m.get("content") or ""

            if role in ("user", "assistant") and isinstance(content, str) and content.strip():
                # Pass conversation_time to invoker
                invoker.process_conversation_message(
                    role=role,
                    content=content,
                    conversation_time=conversation_time,  # NEW
                    thread_id=thread_id
                )
                messages_processed += 1
```

#### Agent Invoker

**File: `longmemeval-benchmarker/src/mycelian_memory_agent/invoker.py`**

Update process_conversation_message:
```python
def process_conversation_message(self, role: str, content: str,
                                conversation_time: Optional[str] = None,
                                thread_id: Optional[str] = None) -> None:
    """Process a conversation message.

    Args:
        role: Message role (user/assistant)
        content: Message content
        conversation_time: When conversation occurred (ISO format)
        thread_id: Optional thread identifier
    """
    # Store conversation_time in state for agent to use
    if conversation_time:
        # Make it available to the agent via context
        self.current_conversation_time = conversation_time

    # ... rest of existing implementation
```

### 5. Critical: Ensure conversation_time Flows Through Entire Pipeline

**IMPORTANT**: For the QA model to answer temporal questions correctly, conversation_time must flow through:
1. **ADD** - Store conversation_time when adding entries
2. **RETRIEVE** - Return conversation_time in list_entries and search results
3. **CONTEXT** - Include dates in Timeline section for QA model

#### Update Entry Response Structures

**File: `server/internal/storage/postgres/adapter.go`**

Update GetEntry and ListEntries to include conversation_time in responses:
```go
type EntryResponse struct {
    EntryID          string    `json:"entry_id"`
    ActorID          string    `json:"actor_id"`
    MemoryID         string    `json:"memory_id"`
    VaultID          string    `json:"vault_id"`
    CreationTime     time.Time `json:"creation_time"`
    ConversationTime time.Time `json:"conversation_time"`  // NEW
    RawEntry         string    `json:"raw_entry"`
    Summary          string    `json:"summary"`
    Tags             map[string]string `json:"tags,omitempty"`
}
```

Update SELECT queries to include conversation_time:
```sql
SELECT entry_id, actor_id, memory_id, vault_id,
       creation_time, conversation_time, raw_entry, summary, tags
FROM memory_entries
WHERE ...
```

**File: `mcp/internal/handlers/entry_handler.go`**

Ensure list_entries and get_entry responses include conversation_time:
```go
// In handleListEntries response building
entry := map[string]interface{}{
    "entryId":          e.EntryID,
    "creationTime":     e.CreationTime,
    "conversationTime": e.ConversationTime,  // NEW
    "rawEntry":         e.RawEntry,
    "summary":          e.Summary,
    // ...
}
```

#### Update Search Integration

**File: `server/internal/search/weaviate_client.go` (or equivalent)**

Ensure search results include conversation_time so it's available for context building:
- Include conversation_time in indexed data
- Return conversation_time in search results

#### Update Context Building

**File: `longmemeval-benchmarker/src/single_question_runner.py`**

In `_build_qa_context`, ensure Timeline is built from conversation_time:
```python
def _build_qa_context(search_result, top_k):
    # ... existing code ...

    # Build timeline from entries with conversation_time
    timeline_entries = []
    for entry in search_result.get("entries", []):
        if "conversationTime" in entry or "conversation_time" in entry:
            conv_time = entry.get("conversationTime") or entry.get("conversation_time")
            summary = entry.get("summary", "")
            # Parse date and format for timeline
            date_str = conv_time.split("T")[0]  # Get YYYY-MM-DD part
            timeline_entries.append(f"{date_str} - {summary}")

    # Add timeline to context if we have dated entries
    if timeline_entries:
        timeline_section = "\n\n# Timeline\n" + "\n".join(sorted(timeline_entries))
        context += timeline_section

    return context
```

### 6. Database Recreation

Since we're updating the schema directly (no migration):

```bash
# Stop the backend
make backend-down

# Remove the existing database volume
docker volume rm longmemeval-benchmarker_mycelian_postgres_data

# Start fresh with new schema
make start-dev-mycelian-server
```

### 7. Testing Plan

1. **Unit Tests**: Verify conversation_time is stored and retrieved correctly
2. **Integration Test**: Run LongMemEval Q4 (museum dates question)
3. **Expected Result**: Should correctly identify 7-day gap between visits
4. **Query Verification**:
   ```sql
   SELECT entry_id, conversation_time, raw_entry
   FROM memory_entries
   WHERE memory_id = ?
   ORDER BY conversation_time;
   ```

## Benefits

1. **Temporal Reasoning**: Correctly answers time-based questions (Q4 museum visits)
2. **Historical Import**: Supports importing past conversations with correct dates
3. **Real-time Default**: Works naturally for live conversations
4. **Clean Separation**: Clear distinction between when things happened vs when stored
5. **Efficient Queries**: Indexed for temporal range queries
6. **Future-proof**: Foundation for any temporal analysis features

## Example Usage

**Real-time conversation:**
```python
add_entry("User asked about pricing")
# conversation_time = 2025-01-07T10:30:00Z (NOW)
# creation_time = 2025-01-07T10:30:00Z (NOW)
```

**Historical import (LongMemEval):**
```python
add_entry(
    "User visited MoMA",
    conversation_time="2023-01-08T12:49:00Z"
)
# conversation_time = 2023-01-08T12:49:00Z (provided)
# creation_time = 2025-01-07T10:30:00Z (NOW)
```

**Retrieval with dates:**
```python
entries = list_entries(memory_id)
# Returns:
# [{
#   "entryId": "...",
#   "summary": "User visited MoMA",
#   "conversationTime": "2023-01-08T12:49:00Z",  # Available for Timeline
#   "creationTime": "2025-01-07T10:30:00Z"
# }]
```

**QA Context with Timeline:**
```markdown
# Context
User visited MoMA and saw modern art exhibit...
User visited Met Ancient Civilizations exhibit...

# Timeline
2023-01-08 - User visited MoMA
2023-01-15 - User visited Met Ancient Civilizations exhibit
```

## Notes

- This is a clean, first-class implementation with no tech debt
- The name `conversation_time` clearly indicates when the user/agent exchange occurred
- Not a one-way door - can be refined later if needed
- Defaults to CURRENT_TIMESTAMP for backward compatibility
