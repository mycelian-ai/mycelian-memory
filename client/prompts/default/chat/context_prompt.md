#  TOOL: context_maintenance

You MUST follow these instructions while materializing your context to be stored with Mycelian Memory. Context is organized as context shards in the memory where shards belonging to earlier part of the conversation should have more specificity about that part of the conversation. You are responsible for creating accurate context shards. This allows the memory to ensure that we have high fidelity context available in at least some of the shards.

## Context Structure
Context is organized in sections. Add additional section IF AND ONLY IF they cover important context that is not covered by the following sections.

`# Description` - 1-3 sentences summarizing the overall context and conversation focus
`# Facts` - definition covered later
`# Preferences` - definition covered later
`# Decisions` - Specific decisions made with brief rationale (e.g., "Will try Todoist for one week") and time
`# Recommendations` - Advice and suggestions PROVIDED TO the user by the assistant
`# Topics` - Main themes/subjects discussed (not facts, just areas of conversation)
`# Entities` - Specific named things (people, products, organizations, tools) that could match search queries
`# Notes` - Important contextual details that don't fit other sections
`# Timeline` - Events in chronological order. Use minimum granularity needed:
  YYYY-MM-DD for single daily events, add HH:MM for multiple same-day events,
  add :SS for rapid sequences. Goal: clear sequence without excess precision.

## Core Rules

### CRITICAL CONTEXT GENERATION AND PRUNING RULES
- If an input message begins with the exact tag `[previous_context]`, treat everything after the tag as OLD context from previous sessions
- Messages WITHOUT the `[previous_context]` tag are the NEW conversation from the current session
- Strip the `[previous_context]` tag itself and do not persist the tag or its raw content verbatim in shards
- You MUST limit a materialized context to be under 5000 characters. If it exceeds this limit, only then prune information. When pruning, prioritize removing old topic-specific details from sections OTHER than Facts, Preferences, Decisions, and Recommendations.
- Facts, Preferences, Decisions, and Recommendations sections are DURABLE and must be preserved across sessions.
- For other sections (Topics, Entities, Timeline, Notes), prefer information from CURRENT SESSION when topics differ and you meet the pruning condition.

### CRITICAL: Answer-Oriented Information Extraction
Extract information that could answer "who, what, when, where, why, how" questions:
- Identity attributes (credentials, relationships, affiliations)
- States and possessions (what user has, owns, is)
- Temporal events (when things happened)
- Quantifiable information (numbers, amounts, counts)
- Decisions and their outcomes
- Recommendations and advice given

### Data Extraction Rule

- Extract and categorize information into the appropriate sections.
- Do not duplicate information across sections. Each piece of information should appear in exactly ONE section based on its type.
- Be specific, enrich the information with NER where-ever possible.
- Include events with dates in Timeline, derive current state for Facts:
  * Events go in Timeline with dates
  * Current state derived from events goes in Facts
  * Implicit "today" should use current date
- Omit dates for atemporal attributes (has MBA, has 2 siblings) but include date for temporal facts (e.g., 'got MBA on 2019-05-15')
- Timeline section tracks when topics were discussed and when you learned things

#### Factual extraction rules

STRICT definition of a Fact:
An item can ONLY be in Facts if it passes ALL THREE tests:
  1. Is this objectively verifiable? (not opinion/preference/feeling)
  2. Does this describe what IS currently true? (not plans/intentions/possibilities)
  3. Is this a state or attribute, not an action? (states, not activities)

Note: Facts can change over time - that's what Timeline tracks changes for.

For each fact:
- Express it as a complete, standalone statement
- Include temporal markers if relevant
- Resolve all pronouns to specific entities
- Separate facts from interpretations

Fact Update Rules:
- MERGE valid facts from previous context with new facts from current session
- When facts are contradicted or updated:
  - Show the correction with both old and new values
  - Format: `- [Entity] [attribute]: [old_value] → [new_value] [YYYY-MM-DD]`
  - Example: `- User location: New York → San Francisco [2025-09-12]`
  - Example: `- User has cats: 2 → 3 (adopted one more) [2025-09-12]`
- When adding new facts, simply add them to the list:
  - Format: `- [Complete fact statement] [Timestamp if known]`
  - Example: `- User graduated with Business Administration degree [date unknown]`
- Derive current state from past events:
  - Past actions → Current state
  - Acquisitions → Possessions
  - Achievements → Attributes
  - Changes → New status
  Apply this pattern broadly to all events, not just specific categories

  Examples that PASS all tests: "has MBA degree", "owns an Instant Pot", "is employed", "lives in NYC"
  Examples that FAIL: "is evaluating apps" (activity not state), "plans to try Todoist" (intention),
  "prefers simple tools" (preference), "should implement scanning" (recommendation),
  "wants to save money" (intention), "is considering options" (activity)

  If it fails ANY test → it MUST go in another section.
  Common mistakes to AVOID putting in Facts:
  - Current activities ("is evaluating", "is trying", "is considering") - these are actions not states
  - Goals, aims, or objectives ("goal is to", "aims to", "wants to")
  - Plans or intentions ("plans to", "will try", "intends to")
  - Preferences or openness ("prefers", "likes", "open to", "comfortable with")

#### Preference  Definition
  A preference is a subjective choice, inclination, or favored approach that:
  1. Reflects personal taste or style (not objective truth)
  2. Can change over time without contradiction
  3. Describes "how I like things" rather than "what is"
  4. Is about approach/method rather than goals/outcomes

  Examples that belong in Preferences:
  - "Prefers simple, low-effort meal prep"
  - "Likes using Todoist for task management"
  - "Prefers morning workouts"
  - "Favors minimalist design"
  - "Prefers working from home"

### Section-Specific Merge Rules:
- **Facts**: MERGE all valid facts from previous context with new facts
- **Preferences**: ACCUMULATE all preferences unless explicitly contradicted
- **Decisions**: PRESERVE all decisions with timestamps
- **Recommendations**: ACCUMULATE all recommendations given to user
- **Timeline**: Keep all events when possible

### Strategic Information Preservation
When approaching token limits, preserve information by searchability:
- Keep specific names, numbers, and credentials
- Keep decisions and recommendations
- Compress verbose text while preserving searchable terms
- Remove filler words but keep entities and key concepts
Focus: Maintain information that could match future search queries

## Temporal Normalization:
- For temporal facts (events, achievements with dates), include the date when known; otherwise note "(date unknown)".
- When facts change, update the fact and add a Timeline entry showing the change.

### CRITICAL: Validate Before Outputting
Before returning your context, pause and verify:
1. Did I follow ALL the instructions in this prompt?
2. Did I extract ALL relevant information into appropriate sections?
2. Did I extract ALL Facts?
3. Did I preserve ALL durable sections from previous context?
4. Is EVERY section properly formatted in markdown?
5. For each event mentioned, did I derive the resulting state?
6. For each recommendation given, is it captured?

If any answer is "no", revise before outputting.
