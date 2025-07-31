---
adr: 0005-prompt-management
status: accepted
date: 2025-07-24
---
# Prompt Management & Override Mechanism for Summarisation Rules

## Context
LLM agents interacting with Synapse need two prompt templates per memory:
1. **when_to_write** – instructs the agent *when* to call `add_entry`.
2. **summarise** – explains *how* to generate the required `summary` field (NER-dense, vector-friendly).

Defaults must be version-controlled and immutable; power-users, however, need a safe way to customise prompts without editing source or touching the cloud backend.

## Decision
1. **Immutable defaults** are embedded in the MCP binary at build time (`prompts/default/<memory_type>/<kind>.txt`). Updating them requires a code change / new release.
2. **User overrides** live in SQLite (`policies` table) via two new nullable columns:
   * `summarise_prompt TEXT`
   * `when_to_write_prompt TEXT`
3. **Prompt resolution order** at runtime:
   1. SQLite override (if not NULL)
   2. Embedded default
4. **MCP Tool** `get_prompt` (read-only):
   * Args: `memory_id`, `kind="summarise"|"when_to_write"`
   * Returns `{prompt, source}` where `source ∈ {"override","default"}`.
5. **CLI commands** allow humans to set, show, or unset overrides; no tool exists to edit defaults.
6. **Limits**: overrides capped at 2 KB; memory types validated against enum.

## Consequences
+ Predictable default behaviour across upgrades.
+ Users/teams iterate on prompts by CLI only; backend remains dumb.
+ Raising limits or adding new memory types is server-side config only.

_Status: Accepted – implementation started in Milestone 2_ 