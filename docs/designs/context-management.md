# Context Management (Single-Document Strategy)

## Executive Summary

We simplify conversational memory by keeping a single, text-based context document per memory. The agent maintains one concise document with clearly labeled sections (e.g., description, patterns, preferences, facts, key topics, timeline, decisions). The service stores and returns the whole document as text. No facet keys are required.

## Rationale

- Simpler authoring: one thing to read/write; no routing rules.
- Cleaner prompts: teach “what to capture” (extraction rules), not storage schema.
- Lower coordination cost: fewer moving parts than facet indices.
- Extensible: if selective reads become necessary, add section markers or re-introduce 1–2 keys later.

Trade-offs
- Selective reads and type-aware ranking are weaker with a single blob.
- Every update rewrites the whole document.

## Agent Guidance (Prompt Shape)

Keep the prompt short (strategy-style). Instruct the model to:
- Extract only durable, useful info from user messages (assistant messages are context).
- Maintain one concise context document with these sections (short headings, terse content):
  - Description of memory (why it exists / scope)
  - Active context (current focus / working state)
  - Progress (milestones, blockers, next actions)
  - Patterns (approaches that work; mini playbooks)
  - Preferences (stable user preferences)
  - Facts (stable facts, dates, decisions, references)
  - Timeline (important events; terse one-liners)
  - Decisions (key decisions and rationale)
- Keep it brief and updated; remove sections only if permanently irrelevant.
- Output: the full document as plain text. If nothing changes, return the current text.

Example output (shape only)
```
# Description
Short paragraph…

# Active Context
What we’re doing now…

# Progress
- 2025-08-15: deadline moved to 2025-09-01
- Next: ship auth hotfix

# Patterns
- Retry flaky tests once before failing

# Preferences
- Prefers brief answers

# Facts
- CEO = Bob (2025-07-05)

# Timeline
2025-07-05 – CEO changed from Alice to Bob

# Decisions
- Use JWT for service-to-service auth (2025-08-10)
```

## API Semantics (Target)

- PutContext
  - Accepts plain text (preferred). Content-Type: text/plain (or application/json string).
  - Stores the entire context document as-is.
  - Validations: content size cap (configurable), non-empty string.
- GetLatestContext
  - Returns the stored text.
- No facet discovery or filtering endpoints.

Note: Existing clients that send JSON objects can be supported by wrapping or migration, but the target is plain text.

## Storage Model (Target)

PostgreSQL schema (text-based):
```
CREATE TABLE IF NOT EXISTS memory_contexts (
  vault_id      TEXT NOT NULL,
  memory_id     TEXT NOT NULL,
  context_id    TEXT NOT NULL,
  actor_id      TEXT NOT NULL,
  context       TEXT NOT NULL,
  creation_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (vault_id, memory_id, context_id)
);
CREATE INDEX IF NOT EXISTS memory_contexts_recent_idx
  ON memory_contexts (vault_id, memory_id, creation_time DESC);
```

## Search Integration

- Index the full context document as a single field (latest snapshot) for retrieval.
- If needed later, add light heuristics (e.g., headings as fields) without changing the on-wire API.

## Migration Plan

Phase 1 (API + Prompt)
- Keep current behavior but allow text PUTs (text/plain or JSON string). Maintain size cap.
- Update prompt: single-document strategy; full-text output; no keys.

Phase 2 (Storage)
- Switch `context` column to TEXT (from JSONB) or store JSON stringified text for backward compatibility.
- Remove facet-specific validations (keep only size cap).

Phase 3 (Cleanup)
- Remove facet-specific docs and code paths.
- Optional: add section-marker utilities (e.g., helpers to jump to “Progress”).

## Risks & Mitigations

- Loss of selective reads → Mitigate by concise sections and good headings; add light parsing later if needed.
- Large documents → Enforce size cap; encourage brevity in prompt; chunking later if required.

## Status

- Design approved to move toward single text-based context per memory.
- Follow-up PRs will: (1) enable text PUT/GET, (2) update prompt, (3) adjust validations, (4) storage update if needed.
