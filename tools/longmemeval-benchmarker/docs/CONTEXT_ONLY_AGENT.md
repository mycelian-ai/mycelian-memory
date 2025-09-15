# Context-Only Agent Specification

## Executive Summary
A streamlined agent that manages memory through context documents alone, eliminating per-message entry storage for dramatically faster processing while maintaining sufficient accuracy for most use cases.

## Motivation

### Current Performance Bottleneck
The standard agent makes ~600 MCP calls for a 500-message dataset:
- 500 `add_entry` calls (one per message)
- 50 `await_consistency` calls
- 50 `put_context` calls
- Multiple `get_context`/`list_entries` calls

This creates significant latency during benchmark evaluation, slowing iteration cycles.

### Observation
Most LongMemEval questions (4 out of 5) can be answered from well-maintained context alone:
- Q1 (5K run time): Single fact in context
- Q2 (degree update): Knowledge update in context
- Q4 (museum dates): Timeline in context
- Q5 (abstention): Context sufficient

Only Q3 (counting items across sessions) benefits from entry-level granularity.

## Design Philosophy

### Core Principle: Context as Primary Storage
Instead of storing individual messages as entries and periodically synthesizing context, the context-only agent treats the context document as the primary (and only) storage mechanism.

### Key Insights
1. **Context is sufficient** for most retrieval needs when well-structured
2. **Recency bias** naturally occurs in human memory - recent information should override older information
3. **Session-level granularity** is adequate for most applications
4. **Append-only logs** (entries) are overhead for many use cases

## Architecture

### Data Flow
```
Session Start → Await Consistency → Get Previous Context → Process All Messages → Synthesize New Context → Put Context → Session End
```

### Per-Session Operations
- **Inputs**: Previous context + All messages from current session
- **Processing**: In-memory accumulation (no intermediate persistence)
- **Output**: Single updated context document
- **MCP Calls**: Exactly 3 (await_consistency + get_context + put_context)
- **No mid-session flushes**: Synthesis occurs only once at session end
- **No entry storage**: `add_entry` and `list_entries` are not used in this mode

### Configuration (Default On)
- Configured exclusively via TOML (no CLI switch)
- Default: context-only mode is enabled
- Example:
  ```toml
  [agent]
  context_only = true
  ```

### Clean Integration & Coexistence
- Introduce a separate, dedicated context-only agent + invoker pair
- Keep the existing agent unmodified and available
- Add a runner interface with two concrete implementations:
  - Current runner (entries + periodic flush)
  - Context-only runner (no entries, session-end synthesis only)
- Selection is driven by TOML config; do not break current behavior

## Context Synthesis Strategy

### Recency-Biased Merge Algorithm (handled by LLM via existing context_prompt)
When combining previous context with current session:

1. **Facts Section**
   - Current session facts take precedence
   - Previous facts retained only if not contradicted
   - Example: "User has MBA" replaces "User has bachelor's degree"

2. **Timeline Section**
   - Append new conversation dates
   - Preserve all historical dates
   - Maintain chronological order
   - Include conversation_time from current session

3. **Entities Section**
   - Update relationships based on current session
   - Recent relationships override previous ones
   - New entities added, obsolete ones removed

4. **Preferences Section**
   - Current session preferences override previous
   - Allows natural preference evolution

5. **Notes Section**
   - Prioritize current session observations
   - Retain important historical notes within size limits

### Context Size Management
- Target: 5000 words maximum (enforced by prompt/LLM)
- Pruning strategy: Remove oldest, least-referenced information first
- Protected elements: Recent facts, all Timeline entries, key entities

## Information Preservation

### What Gets Preserved
- All conversation dates (Timeline)
- Current facts and knowledge
- Active entity relationships
- Recent preferences and decisions
- Key themes and topics

### What Gets Pruned
- Outdated facts (superseded by updates)
- Old conversation details (kept as summary)
- Inactive entity relationships
- Historical preferences (if changed)
- Redundant information

## Advantages

### Performance
- **~75% reduction in MCP calls** (approx. 600 → 150 for a 50-session dataset)
- **10x faster evaluation** cycles
- **Reduced network latency**
- **Lower database load**

### Simplicity
- **Single source of truth** (context document)
- **No synchronization issues** between entries and context
- **Easier debugging** (inspect one document vs. hundreds of entries)
- **Cleaner mental model**

### Maintenance
- **No orphaned entries**
- **No consistency delays**
- **Simpler error recovery**
- **Predictable storage growth**

## Limitations and Mitigations

### Limitation 1: Granular Search
- **Issue**: Cannot search individual messages
- **Impact**: Q3-type questions (counting specific items)
- **Mitigation**: Ensure context includes comprehensive Facts section

### Limitation 2: Audit Trail
- **Issue**: No per-message history
- **Impact**: Cannot reconstruct exact conversation flow
- **Mitigation**: Timeline provides conversation-level granularity

### Limitation 3: Information Loss
- **Issue**: Details may be lost in synthesis
- **Impact**: Very specific queries might fail
- **Mitigation**: Careful synthesis prompts that preserve key details

### Clarifications
- Conversation-time handling may be added later; out of scope for v1
- Contradictory facts are resolved by the LLM and prompt; no extra logic

## Implementation Strategy

### Phase 1: Proof of Concept
1. Implement context-only agent/invoker pair with session orchestration:
   - Start-of-session: `await_consistency` → `get_context`
   - End-of-session: `put_context`
   - No mid-session flushes; no `add_entry`/`list_entries`
2. Add runner interface and a context-only runner; keep current runner intact
3. Add TOML `[agent].context_only = true` (default) to select this mode
4. Run on 5-question dataset; measure speed and ensure stability

### Phase 2: Optimization
1. Refine orchestration and logging (`context_only=true` in logs)
2. Keep using existing `context_prompt`; avoid changing prompt semantics
3. Prepare for future conversation_time support (out of scope for v1)

### Phase 3: Evaluation
1. Compare with standard agent on full benchmark
2. Identify question types that fail
3. Document performance/accuracy tradeoff
4. Determine appropriate use cases

## Expected Results

### Performance Metrics
- **Ingestion Speed**: 10x improvement
- **Total Runtime**: 5-8x improvement
- **MCP Calls**: ~75% reduction
- **Database Storage**: 90% reduction

### Accuracy Expectations
- **Q1, Q2, Q5**: 100% accuracy maintained
- **Q4**: 100% when conversation_time support is added later
- **Q3**: Expected degradation vs entry-level baseline (context-only by design)
- **Overall**: 80-100% of baseline accuracy

## Use Case Recommendations

### Ideal For
- Rapid prototyping and evaluation
- Chat applications with session-based interactions
- Systems where recent information matters most
- High-volume, low-granularity applications

### Not Recommended For
- Audit-required systems
- Fine-grained search requirements
- Historical analysis applications
- Multi-user collaborative memories

## Conclusion
The context-only agent represents a pragmatic tradeoff between performance and granularity. By eliminating per-message storage overhead and focusing on session-level context management, we can achieve order-of-magnitude performance improvements while maintaining sufficient accuracy for most practical applications. This approach is particularly valuable for rapid iteration during development and for production systems where speed is prioritized over perfect recall.

## Operational Notes
- Logging should include `context_only=true` in agent initialization and runner events
- QA retrieval remains unchanged (two-pass search supported as-is)
- LLM decides what to store; it may incorporate retrieved previous context into the new context
