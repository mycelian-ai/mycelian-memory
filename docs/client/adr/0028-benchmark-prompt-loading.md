# ADR 0028 – Benchmark Harness Prompt Loading Strategy

Status: ⏳ Accepted  
Date: 2025-07-30  
Authors: Benchmark Team

---

## Context

The benchmark ingestion harness replays historical multi-turn conversations through Claude-3 and persists every turn in Synapse. Claude must follow several static rule/prompt assets:

* `ctx_rules` – global Context & Summary rules  
* `ctx_prompt_chat` – context-maintenance instructions  
* `entry_prompt_chat` – entry-capture protocol  
* `summary_prompt_chat` – summary-generation rules

These assets live server-side and are fetched with the MCP tool `get_asset(id)`. We need to expose each asset to the model exactly **once** per session—enough for compliance, but without bloating the context window.

## Decision

1. **Single fetch, single exposure**  
   • Claude calls `get_asset(id)` during bootstrap.  
   • The harness returns the raw asset text inside the associated `tool_result` block of that same turn.

   Example history fragment:
   ```json
   { "role": "assistant",
     "content": [{ "type": "tool_use", "id": "42", "name": "get_asset", "input": { "id": "entry_prompt_chat" }}] },
   { "role": "user",
     "content": [{ "type": "tool_result", "tool_use_id": "42", "content": "<full prompt text here>" }] }
   ```

2. **In-memory cache**  
   Repeat `get_asset` calls are served from a session-local cache (`_asset_cache`) and do **not** re-insert the text into message history.

3. **History never duplicates assets**  
   Asset content lives only in the single `tool_result` message; normal conversation turns do not contain prompt text.

4. **System prompt only names asset IDs**  
   The system prompt lists the required IDs so Claude knows what to fetch, keeping the system prompt itself small (<200 tokens).

## Consequences

* Each asset (~1–2 kB, 500–800 tokens) is paid once per session—negligible versus the ≥100-turn benchmark context window.
* Token growth is linear with conversation length, not quadratic; no “token explosion”.
* New assets can be added by updating the ID list; the fetch mechanism is generic.

## Rejected Alternatives

* **Embedding assets in the system prompt** – would breach Anthropic’s 4 k token limit and cannot be updated per session.
* **Fetching assets every turn** – guarantees freshness but inflates token usage ~3× per turn.

## Follow-ups / Future Work

* Auto-prune the initial `tool_result` asset messages if the session ever exceeds the 4 k context-token sweet-spot (after rules are internalised). 