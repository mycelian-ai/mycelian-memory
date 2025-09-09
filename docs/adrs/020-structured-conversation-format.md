# ADR-020: Structured Conversation Format for Context Synthesis

**Status**: Proposed
**Date**: 2025-01-09

## Context

The LongMemEval benchmarker agent needs to synthesize conversation context at the end of each session. Currently, the system has critical issues that prevent correct context synthesis:

1. **Ambiguous Message Structure**: The LLM cannot distinguish between:
   - System instructions and rules
   - Previous context from earlier sessions
   - Current session messages
   - This leads to incorrect application of context replacement rules

2. **Wrong Prompt Ordering**: Instructions come after the conversation data, causing the model to process data before understanding the rules

3. **State Management Bug**: conversation_history is reset instead of appended at START_SESSION (line 378), losing accumulated messages

These issues manifest as the "5K personal best bug" where new, relevant information (charity 5K personal best time of 25:50) is discarded in favor of old, unrelated content (document organization from previous sessions).

### Current Implementation Problems

The current format embeds everything as plain text in a single system message:
```
Role: system
Content: [previous_context]
User has a dog named Max.

Role: user
Content: I'm training for a 5K run...
```

The LLM cannot determine:
- Where previous context ends and current session begins
- Which content should be prioritized
- How to apply the "prefer current over old" rules

## Decision

Implement a structured conversation format using typed sections with clear delimiters:

### Data Structure
```python
conversation_sections = [
    {
        "type": "system_prompt",
        "content": "Agent instructions, rules, and prompts"
    },
    {
        "type": "previous_context",
        "content": "Context from earlier sessions (marked with [previous_context])"
    },
    {
        "type": "current_session_messages",
        "content": [
            {"role": "user", "content": "..."},
            {"role": "assistant", "content": "..."}
        ]
    }
]
```

### Formatted Output
When serialized for the LLM, use explicit section markers:

```
=== SYSTEM INSTRUCTIONS ===
[All rules, prompts, and instructions]

=== PREVIOUS CONTEXT ===
[Content from earlier sessions]

=== CURRENT SESSION ===
[Messages from current session]
```

### Implementation Details

1. **Fix line 378**: Change from replacing to appending conversation_history
2. **New functions**:
   - `build_structured_conversation()`: Separates messages into typed sections
   - `format_structured_prompt()`: Serializes sections with clear markers
3. **Message detection**: Use `[previous_context]` tag to identify old content
4. **Prompt ordering**: All instructions before conversation data

## Consequences

### Positive Consequences
- Clear distinction between old and new content enables correct prioritization
- LLM can properly apply context replacement rules
- Type-safe structure that's easy to test and validate
- Fixes the bug where new information is incorrectly discarded
- Improved debugging with clear section boundaries

### Negative Consequences
- Requires refactoring existing prompt building code
- All existing tests need updating for new format
- Slightly more complex code structure

### Neutral Consequences
- Prompt length remains approximately the same
- No changes to MCP tool interfaces
- No impact on other agent operations (add_entry, search, etc.)

## Alternatives Considered

### Alternative 1: Multiple LLM Messages
**Description**: Pass conversation as separate LLM messages instead of embedded text
**Pros**: Preserves semantic message structure
**Cons**: Would require major refactoring of agent architecture
**Why rejected**: Too invasive for fixing this specific bug

### Alternative 2: JSON Structure in System Message
**Description**: Use JSON formatting within the system message
**Pros**: Machine-readable structure
**Cons**: LLMs sometimes struggle with nested JSON in prompts
**Why rejected**: Plain text with markers is more reliable

### Alternative 3: XML Tags
**Description**: Use XML-style tags like `<previous_context>` and `<current_session>`
**Pros**: LLMs parse XML well
**Cons**: More verbose, harder to read in logs
**Why rejected**: Section markers with `===` are cleaner and equally effective

## Implementation Notes

### Migration Steps
1. Create unit tests for new format (test_structured_prompt.py)
2. Implement build_structured_conversation() function
3. Implement format_structured_prompt() function
4. Update build_put_context_prompt() to use new functions
5. Fix line 378 to append instead of replace
6. Update existing tests
7. Validate with 5K test dataset

### Success Criteria
- 5K personal best time (25:50) is correctly retained in context
- Previous unrelated content is replaced when topics differ
- All existing tests pass with new format
- Clear section boundaries visible in logs

### Code Changes Required
- `/src/mycelian_memory_agent/agent.py`: Add new functions, fix line 378
- `/tests/agent/test_structured_prompt.py`: New test file
- `/tests/agent/test_prompts.py`: Update existing tests

## References

- ADR-016: LongMemEval Agent Architecture (establishes agent structure)
- Issue: 5K personal best time incorrectly discarded during context synthesis
- Logs: run_1757375741 showing incorrect context synthesis
