# Context & Summary Rules

### MCP tools available (use these names exactly)

* **create_memory** – create a new memory for the user; used only during benchmark setup.
* **get_memory** – read metadata for an existing memory (title, type, stats).
* **add_entry** – persist one raw message plus its summary in a memory.
* **list_entries** – retrieve recent entries; supports `limit`, `before`, `after` cursors.
* **get_context** – fetch the current context document for a memory.
* **put_context** – write/overwrite the context document.
* **await_consistency** – wait until previous writes are durably visible.
* **get_user** – fetch the user profile object (name, email, quotas).
* **search_memories** – search within a specific memory, returning ranked entries, best and latest context.

**Tool Scoping** – If several tool-specific instruction blocks exist in the prompt window, obey **only** the block whose `### TOOL:` label matches the function you are currently executing; ignore all other blocks.

1. **Workflow scope** – This file defines WHEN to call tools, not WHAT to write. For content and formatting of context and summaries, see the prompt files referenced below.
2. **Flow per incoming message** (await_consistency barrier)
   1. Process the message.
   2. You MUST call `add_entry` exactly once for every user and assistant message.
   3. **Flush cadence (STRICT)** – `put_context` is expensive and MUST be batched:
      • Do NOT call `put_context` after every message.
      • Call `put_context` only after ≈ 6 messages (user + assistant) have been stored via `add_entry`.
      • Before calling `put_context`, issue `await_consistency()` to ensure previous writes are durable.
      • At session end, issue `await_consistency()` → `put_context` once to flush any remaining updates.

6. **TOOL: search_memories (STRICT)**

   • YOU MUST NOT call `search_memories` on assistant turns.
   • YOU MUST call at most once per user turn.
   • YOU MUST NOT search if the information is already in your current context, in the latest `list_entries(10)`, or was stated in THIS session.
   • If your query equals or paraphrases the current user message and the answer is derivable from your working context, YOU MUST NOT call `search_memories`.
   • If your working context already contains the answer, YOU MUST answer directly and MUST NOT call `search_memories`.
   • If you already confirmed the fact in THIS session, YOU MUST answer directly and MUST NOT call `search_memories`.
   • YOU MUST use `top_k=5` by default; increase ONLY if nothing relevant is found.
   • YOU MUST NOT repeat a semantically similar query within the last 3 turns.
   • YOU MAY search ONLY when:
     – The user references prior discussion or specific past events, or
     – The user asks about remembered facts, or
     – There is a contradiction/update to a prior fact, or
     – Needed facts likely sit beyond the 5,000‑char context window.
   • QUERY STYLE: ≤8 tokens; include key entities/IDs/dates; avoid generic terms (e.g., NOT "sky color" if discussed this session).

   For routine conversation turns, rely on current context and the recent entries.

7. **TOOL: get_context (RESTRICTED)**

   • DO NOT call `get_context` on every turn or automatically before processing messages.
   • Call `get_context` only when:
     – immediately after `put_context` followed by `await_consistency`, to verify the write; or
     – resuming a previously paused session; or
     – explicitly instructed by the user to reload the context.
   • When adding retrieved context to the conversation history for synthesis, prefix it with `[previous_context]` to mark it as prior context for pruning. This tag MUST NOT be persisted in stored context.

8. **References**
   • Context content/format: `client/prompts/default/chat/context_prompt.md`
   • Entry capture guidance: `client/prompts/default/chat/entry_capture_prompt.md`
   • Summary content/format: `client/prompts/default/chat/summary_prompt.md`

9. **Prompt usage**
   • For `add_entry`: Generate the summary using `client/prompts/default/chat/summary_prompt.md` and construct tool inputs per `client/prompts/default/chat/entry_capture_prompt.md`. Do not include any `[SYSTEM_MSG]`/`[CONVERSATION_MSG]` markers or tag markers like `[previous_context]` in raw_entry or summary.
   • For `put_context`: Produce the full context string using `client/prompts/default/chat/context_prompt.md`, then write it via `put_context`.
     – If any input message begins with `[previous_context]`, treat everything after the tag as prior conversation context from earlier turns; use it to guide pruning per the context prompt; strip the tag and do not copy the raw prior context verbatim.

10. **Creation procedures**
   **Entry (per message):**
   1) Take the incoming conversation turn (user/assistant). Use its text as `raw_entry` (strip any prompt markers; exclude system/control content).
   2) Generate `summary` by invoking the summary template in `client/prompts/default/chat/summary_prompt.md` over the same text.
   3) Call `add_entry` with `{ raw_entry, summary }` and, if supported, `tags` (e.g., `{ "role": "user" | "assistant" }`).
   4) On success, increment the local counter for flush cadence; on failure, retry per tool policy.

   **Context (on flush or session end):**
   1) When cadence triggers (≈6 stored entries) or at session end, construct the full context by invoking `client/prompts/default/chat/context_prompt.md`.
      - Previous context handling: If any input message begins with `[previous_context]`, treat the content after the tag as prior context. Use it to guide pruning according to the context prompt. Strip the `[previous_context]` tag and do not copy the raw prior context verbatim into the stored context.
   2) Do NOT include any system/control text or prompt markers in the context.
   3) Issue `await_consistency()`; then call `put_context` with the generated context.

### State machines

**Entry Persistence (per message)**
```mermaid
flowchart TD
    NewMsg[New user / assistant message] --> Summ[Generate ≤512-char summary]
    Summ --> Add[add_entry(raw_entry, summary)]
    Add --> Continue[Continue conversation]
```
<!-- See References above for canonical prompt files -->
