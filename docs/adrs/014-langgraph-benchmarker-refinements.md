# ADR-014: LangGraph Benchmarker Architecture Refinements

**Status**: Superseded
**Date**: 2025-08-27
**Superseded by**: ADR-015
**Amends**: ADR-013

## Context

During implementation of the LangGraph-based LongMemEval benchmarker (ADR-013), several critical architectural issues were discovered that required design refinements:

1. **Recursion Limit Issues**: Memory Agent was calling `get_default_prompts` on every turn, leading to ~26 tool calls per turn and hitting LangChain's default recursion limit of 25.

2. **Agent Lifecycle Confusion**: Initial design was ambiguous about whether Memory Agent should persist per conversation or per session.

3. **Node Communication Pattern**: Direct agent invocation from nodes vs proper LangGraph patterns needed clarification.

4. **Prompt Loading Strategy**: The design instructed agents to fetch prompts dynamically but this caused performance issues.

5. **QA Agent Complexity**: Original design showed multiple sequential tool calls when a single search would suffice.

These issues were discovered through debugging sessions where the Memory Agent appeared to complete successfully but stored 0 entries, leading to root cause analysis of the tool-calling patterns.

## Decision

We are refining the LangGraph benchmarker architecture with the following amendments:

### 1. Memory Agent Lifecycle - Per Session
- **NEW**: Create a fresh Memory Agent instance for **each session** within a conversation
- **Rationale**: Mirrors production reality where agents don't persist indefinitely
- **Continuity**: Cross-session continuity provided by Mycelian Memory through bootstrap protocol
- Each new session's agent bootstraps from stored memory (`get_context` → `list_entries`)

### 2. Prompt Pre-loading Strategy
- **NEW**: Pre-load prompts **once** during Memory Agent initialization
- **Implementation**: Fetch `get_default_prompts(memory_type='chat')` during agent creation
- **Embed**: Include full rules directly in system prompt
- **Result**: Reduces tool calls from ~26 per turn to ~3 per turn (only `add_entry`, `put_context`)

### 3. Node-Wrapped Agent Pattern
- **NEW**: Memory Agent wrapped in dedicated LangGraph Node
- **Benefits**:
  - Better observability in LangGraph traces
  - Error boundaries and retry logic
  - State management separation
  - Conditional routing based on outcomes
- **Implementation**: `Conversation Processor Node → Memory Agent Node → Memory Agent`

### 4. Vault Organization
- **NEW**: One vault per host/environment (e.g., "sam-macbook-longmemeval-vault")
- **Pattern**: Get-or-create for idempotency
- **Memory Naming**: `conv-{conversation_id}-run-{run_id}`
- **Rationale**: Persistent vault across benchmark runs on same host

### 5. Error Handling Strategy
- **NEW**: Retry with exponential backoff, then fail
- **Special Case**: Bedrock throttling requires longer backoff periods
- **Implementation**:
  - Max 3 retries with exponential backoff
  - 2x longer wait times for Bedrock-specific errors
  - Recursion limit errors retry once with higher limit

### 6. Simplified QA Strategy
- **NEW**: Single `search_memories()` call returns everything needed
- **Returns**: Relevant entries + best context shard + latest context
- **Removes**: Sequential `get_context()` → `list_entries()` → `search_memories()` pattern
- **Benefit**: More efficient, fewer tool calls

### 7. Run Management
- **NEW**: Generate run_id at workflow level (UUID or timestamp)
- **Scope**: Same run_id for all conversations in benchmark run
- **Purpose**: Track and compare benchmark runs over time

## Consequences

### Positive Consequences
- **Performance**: ~87% reduction in tool calls per turn (26 → 3)
- **Reliability**: Eliminates recursion limit errors
- **Clarity**: Clear agent lifecycle management
- **Production-Ready**: Patterns match real-world usage
- **Observability**: Better debugging through node wrapping
- **Efficiency**: Simplified QA reduces API calls

### Negative Consequences
- **Complexity**: Slightly more complex with node wrapping
- **Memory**: Pre-loaded prompts increase agent memory footprint

### Neutral Consequences
- **Different from Original**: Significant changes from ADR-013 design
- **Learning Curve**: Team needs to understand refined patterns

## Implementation Changes Required

### From ADR-013 Design to Implementation:

1. **Memory Agent System Prompt**:
   ```python
   # OLD: Instruct to fetch prompts
   "Use the get_default_prompts tool to fetch..."

   # NEW: Pre-load and embed
   f"Here are the rules you must follow:\n{prompts_data['context_summary_rules']}"
   ```

2. **Conversation Processor**:
   ```python
   # OLD: Single agent for conversation
   # NEW: Fresh agent per session
   for session in haystack_sessions:
       agent = create_memory_agent(...)  # New agent
       # Process session
   ```

3. **State Definition**:
   ```python
   # Added fields:
   run_id: str  # Workflow-level
   vault_id: str  # From setup node
   mcp_client: Any  # Shared client
   error: Optional[str]  # Error tracking
   ```

## Validation

The refined architecture was validated through:
1. Debug script confirming Memory Agent works with pre-loaded prompts
2. Recursion limit testing showing ~26 tool calls needed without optimization
3. Successful reduction to ~3 tool calls with pre-loading
4. Confirmation that session-level agents with Mycelian bootstrap provides continuity

## References

- Original Design: ADR-013
- Updated Design: `/docs/designs/langgraph_longmemeval_benchmarker.md`
- Debug Session: `/tools/langgraph-benchmarker/debug_single_turn.py`
- Memory Protocol: `/client/prompts/system/context_summary_rules.md`

---
