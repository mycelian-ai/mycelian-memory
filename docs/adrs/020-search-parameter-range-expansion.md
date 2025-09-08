# ADR-020: Search API Parameter Range Expansion

**Status:** Accepted
**Date:** 2025-01-08
**Supersedes:** Partially amends ADR-017
**Authors:** @sam33rch

## Context

ADR-017 established standardized search API parameters with conservative ranges:
- `top_ke`: 0-10 (entries)
- `top_kc`: 1-3 (context shards)

During implementation and testing with the LongMemEval benchmark, we discovered that these ranges were too restrictive for effective QA operations. When agents are uncertain during the QA phase, they need the ability to explore more context and entries thoroughly to find relevant information.

The benchmarking revealed that limiting context shards to 3 and entries to 10 was insufficient for complex queries where information might be scattered across multiple conversation sessions or when the agent needs to disambiguate between similar memories.

## Decision

We will expand the parameter ranges while maintaining the same defaults:

1. **`top_ke` (entries)**: Expand from 0-10 to **0-25**
   - Default remains 5
   - Allows agents to retrieve more entries when needed for comprehensive analysis

2. **`top_kc` (context shards)**: Expand from 1-3 to **1-10**
   - Default remains 2
   - Enables deeper context exploration during uncertainty

The expanded ranges provide flexibility for agents to:
- Perform more thorough searches when initial results are ambiguous
- Gather additional context for complex multi-turn conversations
- Better handle temporally distributed information

## Consequences

### Positive
- **Better QA Accuracy**: Agents can retrieve more context when uncertain, improving answer quality
- **Flexibility**: Applications can tune retrieval depth based on their specific needs
- **Reduced False Negatives**: More comprehensive searches reduce "I don't know" responses when information exists
- **Progressive Refinement**: Agents can start narrow and expand search scope as needed

### Negative
- **Increased Token Usage**: Larger result sets consume more tokens in LLM context
- **Higher Latency**: Retrieving and processing more results takes longer
- **Potential Noise**: More results may include less relevant information

### Neutral
- Defaults remain unchanged, so existing integrations are unaffected
- The change is backward compatible with ADR-017's original ranges

## Implementation Details

### Current Implementation (as deployed)
```go
// Handle top_ke parameter (default: 5, range: 0-25)
topKE := 5
if v, ok := req.GetArguments()["top_ke"].(float64); ok {
    topKE = int(v)
    if topKE < 0 || topKE > 25 {
        return mcp.NewToolResultError("top_ke must be between 0 and 25"), nil
    }
}

// Handle top_kc parameter (default: 2, range: 1-10)
topKC := 2
if v, ok := req.GetArguments()["top_kc"].(float64); ok {
    topKC = int(v)
    if topKC < 1 || topKC > 10 {
        return mcp.NewToolResultError("top_kc must be between 1 and 10"), nil
    }
}
```

### Usage Recommendations
- Start with defaults (5 entries, 2 context shards)
- Increase `top_ke` when dealing with list-type queries or comprehensive summaries
- Increase `top_kc` when temporal context evolution is important
- Consider token limits when using higher values

## Alternatives Considered

### Alternative 1: Keep Conservative Ranges
**Description**: Maintain ADR-017's original 0-10 and 1-3 ranges
**Pros**: Lower token usage, faster responses
**Cons**: Reduced QA accuracy, more false negatives
**Why rejected**: Benchmark testing showed insufficient retrieval for complex queries

### Alternative 2: Dynamic Range Adjustment
**Description**: Automatically adjust ranges based on query complexity
**Pros**: Optimal per-query performance
**Cons**: Complex implementation, unpredictable behavior
**Why rejected**: Added complexity without clear benefit over explicit control

### Alternative 3: Unlimited Ranges
**Description**: Remove upper bounds entirely
**Pros**: Maximum flexibility
**Cons**: Risk of excessive token usage, performance degradation
**Why rejected**: Upper bounds provide necessary guardrails

## Implementation Notes

- This change is already deployed in the MCP handler
- No migration required as ranges are expanded, not reduced
- Documentation and MCP tool descriptions should be updated to reflect new ranges
- Monitor token usage patterns with expanded ranges

## References
- [ADR-017: Search API Parameter Standardization](./017-search-api-improvements.md)
- [MCP Search Handler Implementation](../../mcp/internal/handlers/search_handler.go)
- LongMemEval benchmark results showing improved retrieval with expanded ranges
