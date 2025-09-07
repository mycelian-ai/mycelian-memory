###  TOOL: context_maintenance (Markdown)

You MUST follow these instructions while materializing your context to be stored with Mycelian Memory.
Context is organized as context shards in the memory where shards belonging to earlier part of the conversation
should have more specificity about that part of the conversation. You are responsible for creating accurate context
shards. This allows the memory to ensure that we have high fidelity context available in at least some of the shards.

**Core Rules:**

Context Sharding and Pruning Rules: You MUST limit a materialized context to be under 5000 words or 1000 tokens, whichever is smaller. If it is larger than that this limit then you have to prune some information. Prioritize pruning information that is from an older part of the conversation that you didn't observe in this specific session. If an input message begins with the exact tag `[previous_context]`, treat everything after the tag as prior conversation context from earlier turns. Use this prior context to guide pruning when applying the sharding/size limits: prefer preserving facts, entities, topics, decisions, and timeline items that align with or are referenced by this prior context; deprioritize unrelated older details. Do not copy the whole prior context verbatim. Extract durable facts and integrate them into the appropriate sections of the materialized context shards per the Document Structure above. Strip the `[previous_context]` tag itself and do not persist the tag or its raw content verbatim in shards.

Data Extraction Rule:
- Record durable facts, user preferences, decisions, key events, key topics, and important entities.
  - For example: Preserve quantities and counts explicitly,
    determine details of a transaction happening between two entities, etc
- Be specific, enrich the information with NER where-ever possible.
- Facts: Put current actionable items first. Don't overwrite past facts. For e.g.: user may have adidas shoes at time t1 then they have nike at time t2. Both facts must be recorded with time (if available).
- Timeline: Keep detailed timeline of event in `YYYY‑MM‑DD – event` format.
- Include dates when helpful in ISO format (YYYY‑MM‑DD). Add pointers to which information is old vs new to help with pruning old information, when needed.

**Document Structure:**
Use these section headings. You can add additional sections if the conversation needs it but have a high bar for creating new sections. Omit empty sections entirely.

`# Description` - 1-3 concise sentences on purpose, scope, and success criteria
`# Facts` - One fact per line, bullet format
`# Preferences` - Stable user preferences (short, actionable)
`# Decisions` - Key decisions with brief rationale
`# Topics` - Key topics/themes to track
`# Entities` - Subject → Object (brief role/relationship)
`# Notes` - Free-form nuance that doesn't fit above (use sparingly)
`# Timeline` - YYYY-MM-DD – succinct event
`# Diagram` - Optional Mermaid diagram (≤10 nodes, ≤600 chars, only if clarifying)
