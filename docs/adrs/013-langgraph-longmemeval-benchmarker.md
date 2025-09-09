# ADR-013: LangGraph-Based LongMemEval Benchmarker

**Status**: Superseded
**Date**: 2025-08-26
**Superseded by**: ADR-015

## Context

The current LongMemEval benchmarker (`tools/benchmarker/`) has grown to over 2000 lines with significant architectural complexity:

- **SessionSimulator**: Complex state management with manual flags (`_boot_seen_get_context`, `_boot_seen_list_entries`)
- **Custom message routing**: Prefix-based routing system (`benchmark_conversation:speaker_1`, `control:test_harness`)
- **Manual tool orchestration**: Direct Anthropic API calls with custom tool dispatch logic
- **State tracking complexity**: Manual conversation state management and bootstrap sequences

This complexity makes the benchmarker difficult to maintain, debug, and extend. Additionally, we need to establish baseline memory quality metrics using the LongMemEval academic benchmark, which tests 5 core long-term memory abilities across 500 evaluation questions.

LangGraph offers native MCP (Model Context Protocol) support and built-in agent orchestration capabilities that could significantly simplify this architecture while providing a reusable pattern for customers implementing memory management workflows.

## Decision

We will replace the current Python-based SessionSimulator with a LangGraph-based benchmarker architecture that leverages:

1. **End-to-End LangGraph Workflow**: Single workflow orchestrating three sub-workflows (Ingestion, QA, Evaluation)
2. **Mycelian Memory Agent**: Stateful LangGraph Agent using observer pattern to build high-quality memories following `@client/prompts/system/context_summary_rules.md` protocol exactly
3. **Frugal Search Strategy**: Enhanced `context_summary_rules.md` with specific guidance for when to use `search_memories` tool to balance memory quality with efficiency

### Key Components:
- **Conversation Processor Node**: Stateless LangGraph node that reads haystack sessions and sends turns to Memory Agent
- **Mycelian Memory Agent**: Stateful agent maintaining session awareness, processing turns with format: `"Observe this turn: <role>: <content>"`
- **QA Agent**: Separate agent for answering questions using stored memories
- **Evaluation Node**: GPT-4o judge following LongMemEval's evaluation methodology

### Search Guidance Integration:
Added to `context_summary_rules.md` protocol - use `search_memories` only when:
- Contradictory information (updates/corrections to previous facts)
- References to specific past events ("as we discussed before", "like last time")
- Direct questions about memory content
- Information that may exist in older context shards (beyond current 5000-char limit)

## Consequences

### Positive Consequences
- **Architectural Simplification**: Eliminates 2000+ lines of custom state management code
- **Native MCP Integration**: Leverages LangGraph's built-in MCP support for tool orchestration
- **Reusable Pattern**: Provides customer-facing example of memory management with Mycelian
- **Protocol Compliance**: Memory agent strictly follows existing `context_summary_rules.md` without additional instructions
- **Benchmark Baseline**: Establishes memory quality metrics for future improvements
- **Maintainability**: Clear separation of concerns between stateless processing and stateful memory management

### Negative Consequences
- **New Dependency**: Adds LangGraph as a required dependency for benchmarking
- **Learning Curve**: Team needs to understand LangGraph concepts (Agents vs Nodes, State Management)
- **Search Strategy Risk**: Frugal search approach may initially miss some memory retrieval opportunities

### Neutral Consequences
- **Different Technology Stack**: Moves from pure Python to LangGraph-based approach
- **Observer Pattern**: Changes from role assumption to observer pattern for memory persistence

## Alternatives Considered

### Alternative 1: Refactor Current SessionSimulator
**Description**: Clean up existing Python-based SessionSimulator architecture
**Pros**: No new dependencies, familiar codebase
**Cons**: Maintains architectural complexity, missing native MCP support, limited reusability
**Why rejected**: Doesn't address core complexity issues and provides less customer value

### Alternative 2: Multi-Agent Cost-Optimization Architecture
**Description**: Complex multi-agent system optimizing for API call costs with batch processing
**Pros**: Lower API costs for large-scale benchmarking
**Cons**: Significantly more complex, compromises memory quality for cost optimization
**Why rejected**: User feedback prioritized memory quality over cost reduction: "lets not worry about cost reduction"

### Alternative 3: Role Assumption Pattern
**Description**: Memory agent assumes it is the original conversation participant
**Pros**: More immersive context building
**Cons**: Potential confusion in multi-participant scenarios, harder to maintain conversation boundaries
**Why rejected**: User feedback: "This won't work" followed by preference for observer pattern

## Implementation Notes

### Migration Steps:
1. Implement LangGraph-based architecture following design document
2. Add frugal search guidance to `context_summary_rules.md` (✅ completed)
3. Create Mycelian Memory Agent with observer pattern
4. Build end-to-end workflow with three sub-workflows
5. Test against LongMemEval dataset variants (S, M, Oracle)
6. Compare memory quality metrics with current benchmarker
7. Iterate search guidance based on benchmark results

### Success Criteria:
- Successful processing of all 500 LongMemEval questions
- Memory quality metrics (precision/recall) baseline established
- Significant reduction in codebase complexity vs current SessionSimulator
- Reusable pattern documentation for customer implementations

### Dependencies:
- LangGraph framework installation
- `langchain_mcp_adapters.client` for MCP integration
- OpenAI API access for GPT-4o evaluation judge
- Existing MCP server and Mycelian Memory backend

### Timeline Considerations:
- After benchmark completion, refine search guidance based on failure analysis
- Use results to inform memory protocol improvements
- Consider expanding search criteria if precision/recall metrics indicate gaps

## References

- Design Document: `/docs/designs/langgraph_longmemeval_benchmarker.md`
- LongMemEval Paper: https://arxiv.org/pdf/2410.10813.pdf
- LongMemEval Dataset: https://huggingface.co/datasets/xiaowu0162/longmemeval
- Current Benchmarker: `/tools/benchmarker/`
- Memory Protocol: `/client/prompts/system/context_summary_rules.md`
- ADR-008: Await Consistency Primitive (related memory consistency mechanisms)

---
