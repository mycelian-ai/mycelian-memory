# Context Management in SessionSimulator

This document describes how context is managed in the `SessionSimulator` class, including when and how context updates occur.

## Overview

The `SessionSimulator` maintains a context that is updated periodically during a conversation. This context is stored in the Mycelian memory system and can be used to maintain state across multiple turns of conversation.

## When Context is Updated

Context is updated in the following scenarios:

1. **Periodic Updates**: Every 5 messages, the context is automatically updated.
2. **Critical Updates**: After certain tool calls (like adding an entry), the context is updated immediately.
3. **Session End**: The context is always updated when the session ends.

## Context Structure

The context is a JSON object with the following structure:

```typescript
interface ContextData {
  // Timestamp of the last update in ISO format
  last_updated: string;
  
  // Total number of messages in the current session
  message_count: number;
  
  // Summary of the conversation history
  history_summary: string;
  
  // Which component last updated the context
  last_updated_by: string;
  
  // Any additional context data
  [key: string]: any;
}
```

## Implementation Details

### Message Counting

The `SessionSimulator` maintains a message counter that increments with each user message. When the counter reaches a multiple of 5, a context update is triggered.

### Error Handling

- If a context update fails, the error is logged but the session continues.
- If getting the current context fails, an empty context is used.

### Tool Integration

The following tools interact with the context system:

- `mcp_mycelian-memory-streamable_add_entry`: Adds an entry to the memory and triggers a context update.
- `put_context`: Directly updates the context with provided content.

## Best Practices

1. **Frequent Updates**: Don't rely on the automatic 5-message update for critical state changes. Use explicit `put_context` calls when needed.
2. **Error Handling**: Always check if context updates were successful in critical paths.
3. **Size Management**: Be mindful of context size. Large contexts may impact performance.

## Example Usage

```python
# Create a session
simulator = SessionSimulator(api_key, mycelian_client, system_prompt_builder)

# Send messages (context updates happen automatically)
await simulator.step("Hello, how are you?")

# Force a context update
await simulator._update_context(force=True)

# Close the session (triggers final context update)
await simulator.close_session()
```

## Testing

Test coverage includes:

- Message counter increments
- Context updates after 5 messages
- Critical updates after tool calls
- Session end behavior
- Error handling

Run tests with:

```bash
pytest benchmarks/python/tests/test_session_simulator_context.py
pytest benchmarks/python/tests/test_session_simulator_helpers.py
pytest benchmarks/python/tests/test_session_simulator.py
```
