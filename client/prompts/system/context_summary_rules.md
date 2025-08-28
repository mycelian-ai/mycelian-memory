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

1. **Source of truth** – The *context document* grows directly from NEW RAW ENTRIES only; never from summaries.
2. **Context purpose** – Store lasting knowledge: participants, stable facts, timeline, decisions, open tasks. Size cap: **5000 chars**. Use Mermaid diagrams (≤10 nodes) when helpful.
3. **Summary purpose** – Optimize vector search. Each stored entry has ONE context-aware summary capped at **512 characters** (≈80 tokens). Summaries must:
   • resolve pronouns using current context
   • follow Subject-Verb-Object past-tense
   • keep names, dates, IDs; drop filler
4. **No feedback loop** – Summaries must NOT be used to update the context.
5. **Flow per incoming message** (await_consistency barrier)
   1. Read current context → understand message.
   2. Decide to store? If yes → generate summary & persist raw+summary.
   3. Update context from raw entry (merge/trim rules).
   4. Flush cadence – put_context is expensive: after you've stored ≈ 6 messages (user + assistant) with add_entry, **issue `await_consistency()` to ensure writes are durable, then call `put_context`**, and continue. Always repeat the sequence (`await_consistency` → `put_context`) once more just before exit.

6. **Search guidance** – Only use `search_memories` when the incoming message contains:
   • Contradictory information (updates/corrections to previous facts)
   • References to specific past events ("as we discussed before", "like last time")
   • Direct questions about memory content
   • Information that may exist in older context shards (beyond current 5000-char limit)
   For routine conversation turns, rely on current context and recent entries from bootstrap.

7. **Overflow Handling**
   1. Context ≤ 5 000 chars: Before writing: if new text would exceed the cap, delete the oldest low-value lines until the length is ≈ 4 800 chars. Keep core facts (participants, active tasks, decisions).
   2. Summary ≤ 512 chars: Trim sentences with little factual content (greetings, filler) first. Keep the lines that name entities, dates, numbers, or other data-rich details that boost vector search. Continue pruning until the text fits within 512 characters, then append "…" if any content was removed.

8. **Session bootstrap**
   1. You **MUST** call `get_context()` first. If the result is **exactly** the default placeholder string
      `This is default context that's created with the memory. Instructions for AI Agent: Provide relevant context as soon as it's available.`
      (inserted automatically when a memory is created), treat it as empty and immediately call `put_context`. Otherwise, keep the returned string as your working context.
   2. Immediately afterwards you **MUST** call `list_entries(limit = 10)` and merge any facts that are missing from the working context **before** replying to the user.

### State machines

**Entry Persistence (per message)**
```mermaid
flowchart TD
    NewMsg[New user / assistant message] --> Store?{Store this?}
    Store? -- No --> Continue
    Store? -- Yes --> Summ[Generate ≤512-char summary]
    Summ --> Add[add_entry(raw_entry, summary)] --> Continue[Continue conversation]
```

**Session Lifecycle**
```mermaid
flowchart TD
    Boot[get_context()]
    Boot -->|plain-text placeholder| Base[put_context({})]
    Boot -->|JSON context| Load[list_entries(10) → merge]
    Base --> Load
    Load --> Loop[Message loop]

    %% per-message handling
    Loop --> Store?[store this entry?]
    Store? -- Yes --> Add[add_entry & track ≈6] --> Check
    Store? -- No  --> Check
    Check{≈6 stored?}
    Check -- Yes / put_context; reset --> Loop
    Check -- No  --> Loop

    %% graceful exit
    Loop --> Bye{<END_SESSION>?}
    Bye -- Yes --> Finish[add_entry (any remaining)] --> Barrier[await_consistency()] --> Final[put_context] --> Exit[Session ends]
```

### Prompt loading

Load memory-type-specific prompts from `prompts/default/{memory_type}/` directory. 

For memory_type="chat", use prompts from `prompts/default/chat/` including:
  1. `context_prompt.md` - for context maintenance
  2. `entry_capture_prompt.md` - for entry persistence 
  3. `summary_prompt.md` - for summary generation

## Appendix A – Worked Examples (ILLUSTRATIVE ONLY)

**IMPORTANT**: These examples demonstrate the workflow and format only. NEVER use the example content (like "Q3 launch date" or "customer-support agent"). Always use ACTUAL content from the conversation you are observing.

#### Example A-1: New memory bootstrap

```text
// BEFORE: get_context() returns the default placeholder
// AGENT ACTION: put_context() with actual conversation content
// For example, if conversation is about Paris trip, context would include Paris facts
```

---

#### Example A-2: Raw entry → summary → context update

```text
// This shows the PATTERN only. Use YOUR conversation's actual content:
// When you see a NEW RAW MESSAGE like "Hi, I'm planning a trip to Paris"
// You would generate a SUMMARY like "User is planning a trip to Paris"
// And update CONTEXT with actual facts from that conversation
```

