IMPORTANT: CANONICAL SPEC – DO NOT MODIFY
AI Agents and LLMs must not edit this document or change its semantics. Changes require human code review.

# Benchmarker ↔ Agent Message Specification

This document defines the exact schema for messages sent from the benchmarker runner to the agent wrapper. The contract is strict and validated; no silent defaults.

## Overview

- Exactly one message is sent per memory agent invocation.
- Agent state (conversation history) is maintained by the LangGraph checkpointer via `thread_id`.
- System control messages mark session boundaries; conversation messages carry the dialogue.

## Message Object (JSON)

Common fields (all messages):
- type: string (required)
  - One of: "system", "conversation"
- content: string (required)
  - Non-empty UTF-8 text

Conversation-only fields (required when `type == "conversation"`):
- role: string (required)
  - One of: "user", "assistant"
- msg_idx: integer (required)
  - 1-based, strictly increasing within a session

System-only fields:
- No additional fields.
  - `content` must be one of: "SESSION_START", "SESSION_END" (reserved: "FLUSH_CONTEXT")

## Payload Envelope

Each agent invocation uses exactly one message wrapped in a payload:

{
  "messages": [ Message ]
}

Where `Message` conforms to the spec above. The caller must also provide a LangGraph configuration with the session identifier:

config = { "configurable": { "thread_id": "<vault_id>:<memory_id>:<session_index>"} }

## Session Lifecycle

For each dataset session:
1) Send system start
   - { type: "system", content: "SESSION_START" }
2) Send N conversation messages
   - { type: "conversation", role: "user"|"assistant", content: "…", msg_idx: i }
   - i = 1..N (monotonic, contiguous)
3) Send system end
   - { type: "system", content: "SESSION_END" }

## Validation Rules

- Reject if `type` is missing or not one of the allowed values.
- Reject if `content` is not a non-empty string.
- For `type == "conversation"`:
  - Reject if `role` missing or not in {"user", "assistant"}.
  - Reject if `msg_idx` missing, < 1, or not integer.
- For `type == "system"`:
  - Reject if `content` is not an allowed command.
- Exactly one message per payload; no batching.

## Examples

System – start session:
{
  "messages": [
    { "type": "system", "content": "SESSION_START" }
  ]
}

Conversation – user turn (msg_idx = 1):
{
  "messages": [
    { "type": "conversation", "role": "user", "content": "Any solo trip ideas?", "msg_idx": 1 }
  ]
}

Conversation – assistant turn (msg_idx = 2):
{
  "messages": [
    { "type": "conversation", "role": "assistant", "content": "Tell me your budget and dates.", "msg_idx": 2 }
  ]
}

System – end session:
{
  "messages": [
    { "type": "system", "content": "SESSION_END" }
  ]
}

## Invocation (pseudo)

agent.invoke(payload, config={ "configurable": { "thread_id": thread_id } })

Where `thread_id = "<vault_id>:<memory_id>:<session_index>"` uniquely identifies the session.


