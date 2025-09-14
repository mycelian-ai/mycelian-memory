###  TOOL: context_maintenance (Markdown)

You MUST follow these instructions while materializing your context to be stored with Mycelian Memory.
Context is organized as context shards in the memory where shards belonging to earlier part of the conversation
should have more specificity about that part of the conversation. You are responsible for creating accurate context
shards. This allows the memory to ensure that we have high fidelity context available in at least some of the shards.

**Core Rules:**

Context Sharding and Pruning Rules: You MUST limit a materialized context to be under 5000 tokens. If it exceeds this limit, only then prune information. When pruning, prioritize removing old topic-specific details from sections OTHER than Facts. The Facts section should remain durable and complete.


**CRITICAL CONTEXT REPLACEMENT RULES:**
- If an input message begins with the exact tag `[previous_context]`, treat everything after the tag as OLD context from previous sessions
- Messages WITHOUT the `[previous_context]` tag are the NEW conversation from the current session
- The Facts section is DURABLE - preserve ALL facts from previous context and add new facts from current session
- For other sections (Description, Topics, Entities, etc.), prefer information from CURRENT SESSION when topics differ
- Only prune if exceeding 5000 token limit, and when pruning:
  - NEVER remove items from Facts section
  - Remove old topic-specific details from other sections first
- Priority for preservation: Facts (always keep) > Current session info > Relevant old context > Unrelated old context
- Strip the `[previous_context]` tag itself and do not persist the tag or its raw content verbatim in shards

Data Extraction Rule:
- Record durable facts, user preferences, decisions, key events, key topics, and important entities.
  - For example: Preserve quantities and counts explicitly,
    determine details of a transaction happening between two entities, etc
- Be specific, enrich the information with NER where-ever possible.
- Timeline: Keep detailed timeline of event in `YYYY‑MM‑DD – event` format.
- Include dates when helpful in ISO format (YYYY‑MM‑DD). Add pointers to which information is old vs new to help with pruning old information, when needed.

Factual information extraction rules. For each fact:
1. Express it as a complete, standalone statement
2. Include temporal markers if relevant
3. Mark confidence as certain/probable/possible
4. Resolve all pronouns to specific entities
5. Separate facts from interpretations

Format: [Entity] [Relationship] [Value/Entity] [Confidence] [Timestamp if applicable]

**Fact Update Rules:**
- MERGE facts from previous context with facts from current session
- NEVER DELETE facts - instead, CORRECT contradictory facts to help LLM reason about changes
- When facts are contradicted or updated:
  - Show the correction with both old and new values
  - Format: `- [Entity] [attribute]: [old_value] → [new_value] [Confidence] [YYYY-MM-DD]`
  - Example: `- User location: New York → San Francisco [Certain] [2025-09-12]`
  - Example: `- User has cats: 2 → 3 (adopted one more) [Certain] [2025-09-12]`
- For conflicting facts without clear resolution:
  - Keep both with uncertainty markers
  - Example: `- User prefers coffee [Probable] [2025-09-10]`
  - Example: `- User mentioned preferring tea today [Certain] [2025-09-12]`
- When adding new facts, simply add them to the list:
  - Format: `- [Complete fact statement] [Confidence] [Timestamp if known]`
  - Example: `- User graduated with Business Administration degree [Certain] [date unknown]`
- Durable facts include but are not limited to: education, personal attributes, relationships, owned items, achievements, preferences, and any other persistent information about the user or their life

**Temporal Normalization:**
- Add dates to durable facts and updates when available; otherwise note “(date unknown)”.
- Add a Timeline entry for important updates (YYYY‑MM‑DD – updated <attribute>: <old> → <new>).

**Document Structure:**
Use these section headings. You can add additional sections if the conversation needs it but have a high bar for creating new sections. Omit empty sections entirely.

`# Description` - 1-3 concise sentences on purpose, scope, and success criteria
`# Facts` - One fact per line, bullet format (DURABLE SECTION - preserve ALL facts from previous context and add new ones)
`# Preferences` - Stable user preferences (short, actionable)
`# Decisions` - Key decisions with brief rationale
`# Topics` - Key topics/themes to track
`# Entities` - Subject → Object (brief role/relationship)
`# Notes` - Free-form nuance that doesn't fit above (use sparingly)
`# Timeline` - YYYY-MM-DD – succinct event
`# Diagram` - Optional Mermaid diagram (≤10 nodes, ≤600 chars, only if clarifying)
