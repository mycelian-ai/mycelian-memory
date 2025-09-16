# ADR-016: LongMemEval Three-Class Agent Architecture

**Status**: Proposed
**Date**: 2025-01-01
**Superseded by**: N/A

Immutability Policy: Once an ADR is Accepted, it becomes an immutable record of that decision. Future changes to the decision must be captured in a new ADR that references and supersedes this one. Do not edit Accepted ADRs beyond correcting typos or adding the Superseded by link.

## Context

The LongMemEval benchmarker's agent implementation has evolved organically through rapid development, resulting in architectural issues that impede maintainability and clarity.

### Background and Current Situation

The current implementation consists of:
- `MycelianMemoryAgent`: A monolithic class handling infrastructure, graph building, and message processing
- `GraphBuilder`: An intermediate layer that defines graph topology but doesn't own it
- `PromptBuilder`: A utility for prompt construction
- Multiple helper classes for logging and debugging

This has led to:
- **Terminology misalignment**: LangGraph calls the compiled graph "the agent", but our `MycelianMemoryAgent` is actually a runner/orchestrator
- **Scattered responsibilities**: Setup logic spread across multiple classes without clear ownership
- **Circular dependencies**: MycelianMemoryAgent creates resources that GraphBuilder uses
- **Mixed concerns**: Infrastructure, behavior definition, and runtime processing intertwined

### Technical Drivers
- Need for better testability of individual components
- Desire to align with LangGraph's conceptual model
- Requirement for clearer separation of concerns
- Goal to reduce complexity for new contributors

### Constraints
- Must maintain backward compatibility with existing runner code
- Should preserve all current functionality
- Need to work within LangGraph's architecture patterns

### Stakeholders Affected
- Development team maintaining the benchmarker
- Users running LongMemEval benchmarks
- Future contributors to the codebase

## Decision

Refactor the agent implementation into three distinct classes with clear single responsibilities:

### What We Will Do

1. **AgentBuilder**: Infrastructure and dependency setup
   - Setup MCP client connection
   - Load tools from MCP server
   - Create LLM instance
   - Build system prompt
   - Initialize helper objects
   - Return configured Agent instance

2. **Agent**: Graph behavior definition
   - Accept dependencies from builder
   - Define graph topology (nodes, edges, conditions)
   - Compile the graph
   - Expose compiled_graph property
   - Focus solely on defining agent behavior

3. **AgentMessageProcessor**: Runtime message handling
   - Validate incoming messages
   - Create LangChain message objects
   - Manage logging context
   - Invoke the Agent's compiled graph
   - Handle results and state
   - Manage session/thread context

### Why This Approach

This three-class design provides optimal separation where:
- Each class has exactly one responsibility
- Dependencies flow in one direction (no cycles)
- The compiled graph is correctly identified as "the agent"
- Testing becomes straightforward with clear boundaries
- The architecture aligns with LangGraph's mental model

## Consequences

### Positive Consequences
- **Clarity**: Each class has an obvious, single purpose
- **Testability**: Components can be tested in isolation with mocked dependencies
- **Flexibility**: Different processors or builders could be swapped
- **Maintainability**: Changes to one concern don't ripple through others
- **Alignment**: Matches LangGraph's terminology and patterns
- **Modularity**: Clean interfaces between components

### Negative Consequences
- **File count**: Increases from 2 main classes to 3
- **Migration effort**: Existing code requires refactoring
- **Learning curve**: New contributors must understand three components instead of one
- **Initial complexity**: Setting up the pipeline requires understanding the flow

### Neutral Consequences
- Total lines of code remains roughly the same
- No runtime performance impact
- External API (factory function) remains unchanged
- Memory footprint essentially identical

## Alternatives Considered

### Alternative 1: Two-Class Design (Builder + Agent)
**Description**: Combine Agent and AgentMessageProcessor into a single class
**Pros**: Fewer classes, simpler file structure
**Cons**: Agent class becomes large, mixes graph definition with runtime logic
**Why rejected**: Violates single responsibility principle, makes testing harder

### Alternative 2: Keep Current Architecture
**Description**: Maintain the existing MycelianMemoryAgent + GraphBuilder structure
**Pros**: No migration needed, team familiar with current code
**Cons**: Continues terminology confusion, poor separation of concerns
**Why rejected**: Technical debt continues to accumulate, misalignment with LangGraph

### Alternative 3: Four+ Class Design
**Description**: Further separate concerns (e.g., separate SessionManager, MessageValidator)
**Pros**: Even more focused classes, maximum flexibility
**Cons**: Over-engineering for current needs, too many small classes
**Why rejected**: Adds complexity without proportional benefit

## Implementation Notes

### Migration Steps
1. Create new classes in parallel with existing code
2. Implement AgentBuilder with all setup logic
3. Implement Agent with graph definition
4. Implement AgentMessageProcessor with runtime logic
5. Update factory function to use new architecture
6. Remove old classes after verification
7. Update all imports and tests

### Timeline Considerations
- Phase 1 (New classes): 2-3 days
- Phase 2 (Migration): 1-2 days
- Phase 3 (Testing): 1-2 days
- Total estimated: 1 week

### Dependencies
- No new external dependencies required
- Must maintain compatibility with existing LangGraph version
- Helper classes (ToolLogger, StateDebugger, MessageLogger) remain unchanged

### Success Criteria
- All existing tests pass
- New unit tests for each class achieve >80% coverage
- Performance benchmarks show no regression
- Code review confirms improved clarity

## References

- [LangGraph Agent Documentation](https://langchain-ai.github.io/langgraph/tutorials/workflows/#agent)
- [ADR-005: Async Job Processing Architecture](./005-async-job-processing-architecture.md)
- Clean Code (Robert C. Martin) - Single Responsibility Principle
- SOLID Design Principles

---

**ADR Guidelines**:
- Use clear, factual language
- Focus on the decision and its rationale
- Include enough context for future readers
- Update status as the decision evolves
- Number ADRs sequentially (001, 002, etc.)
