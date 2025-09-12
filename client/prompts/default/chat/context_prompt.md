###  TOOL: context_maintenance (Markdown)

You MUST follow these instructions while materializing your context to be stored with Mycelian Memory.
Context is organized as context shards in the memory where shards belonging to earlier part of the conversation
should have more specificity about that part of the conversation. You are responsible for creating accurate context
shards. This allows the memory to ensure that we have high fidelity context available in at least some of the shards.

**Core Rules:**

Context Sharding and Pruning Rules: You MUST limit a materialized context to be under 5000 words or 1000 tokens, whichever is smaller. If it is larger than that this limit then you have to prune some information. Prioritize pruning information that is from an older part of the conversation that you didn't observe in this specific session.

**Durability Criteria (domain‑agnostic):**
- Treat a fact as durable if ANY of the following hold:
  - The user explicitly asks to remember it
  - It is low‑volatility (likely stable over months)
  - It is repeated across sessions
  - It is used by the assistant across different topics/tasks
- Use these criteria to infer durability; examples (if mentioned) are illustrative, not exhaustive.

**Pinned Durable Facts (Latest Context):**
- Keep a small, compressed set (≤12 one‑line items) of the most durable facts in the latest context.
- Under token pressure, compress or demote the least‑used durable facts to older snapshots/entries; do not drop high‑value durable facts that meet the criteria above.

**CRITICAL CONTEXT REPLACEMENT RULES:**
- If an input message begins with the exact tag `[previous_context]`, treat everything after the tag as OLD context from previous sessions
- Messages WITHOUT the `[previous_context]` tag are the NEW conversation from the current session
- When the new conversation discusses DIFFERENT topics, replace old topic‑specific details, but NEVER remove durable facts (as defined by the Durability Criteria)
- Only preserve old context if it's directly relevant to or referenced by the new conversation
- Priority order: Current session messages > Relevant old context > Unrelated old context (prune this)
- Do not copy the whole prior context verbatim. Extract only durable facts that remain relevant given the new conversation
- Strip the `[previous_context]` tag itself and do not persist the tag or its raw content verbatim in shards

Data Extraction Rule:
- Record durable facts, user preferences, decisions, key events, key topics, and important entities.
  - For example: Preserve quantities and counts explicitly,
    determine details of a transaction happening between two entities, etc
- Be specific, enrich the information with NER where-ever possible.
- Facts: Put current actionable items first. Don't overwrite past facts. For e.g.: user may have adidas shoes at time t1 then they have nike at time t2. Both facts must be recorded with time (if available).
- Timeline: Keep detailed timeline of event in `YYYY‑MM‑DD – event` format.
- Include dates when helpful in ISO format (YYYY‑MM‑DD). Add pointers to which information is old vs new to help with pruning old information, when needed.

**Knowledge Updates:**
- When a durable fact changes, record an update instead of deleting the prior:
  - previous_value → new_value (YYYY‑MM‑DD), optional brief rationale

**Temporal Normalization:**
- Add dates to durable facts and updates when available; otherwise note “(date unknown)”.
- Add a Timeline entry for important updates (YYYY‑MM‑DD – updated <attribute>: <old> → <new>).

**Document Structure:**
Use these section headings. You can add additional sections if the conversation needs it but have a high bar for creating new sections. Omit empty sections entirely.

`# Description` - 1-3 concise sentences on purpose, scope, and success criteria
`# Facts` - One fact per line, bullet format (place durable facts first)
`# Preferences` - Stable user preferences (short, actionable)
`# Decisions` - Key decisions with brief rationale
`# Topics` - Key topics/themes to track
`# Entities` - Subject → Object (brief role/relationship)
`# Notes` - Free-form nuance that doesn't fit above (use sparingly)
`# Timeline` - YYYY-MM-DD – succinct event
`# Diagram` - Optional Mermaid diagram (≤10 nodes, ≤600 chars, only if clarifying)

**Output Constraints:**
- If headings are forbidden by the caller, emit the same content as labeled bullet blocks in this order (omit empty): durable:, facts:, preferences:, updates:, topics:, entities:, notes:, timeline:
