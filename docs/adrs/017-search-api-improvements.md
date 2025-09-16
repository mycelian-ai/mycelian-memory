# ADR-017: Search API Parameter Standardization

**Status:** Accepted
**Date:** 2025-01-04
**Authors:** @sam33rch

## Context

The search API had evolved with multiple parameter naming conventions:
- `topK` - Legacy combined top-k parameter
- `ke` - Entries top-k (added later)
- `kc` - Context shards top-k (added later)

This created confusion for API consumers and made it difficult for LLMs to understand the API specification. Additionally, the response structure was inconsistent, with conditional fields based on parameter values.

## Decision

We will standardize the search API with:

1. **Clear parameter names**: Replace `topK`/`ke`/`kc` with `top_ke` and `top_kc`
2. **Explicit defaults**: `top_ke=5` (range: 0-10), `top_kc=2` (range: 1-3)
3. **Consistent response structure**: Always include all fields regardless of parameters
4. **Temporal context**: Add timestamps to all results for temporal understanding

This is a breaking change with no backward compatibility.

## Consequences

### Positive
- **API Clarity**: Parameters clearly indicate their purpose (`top_ke` for entries, `top_kc` for contexts)
- **LLM-Friendly**: Comprehensive MCP tool descriptions with defaults help LLMs use the API correctly
- **Temporal Understanding**: Timestamps on all results enable understanding of memory evolution
- **Predictable Response**: Consistent response structure simplifies client implementations

### Negative
- **Breaking Change**: All clients must update their implementations
- **Migration Effort**: Existing integrations need code changes
- **No Compatibility Period**: Hard cutover without deprecation period

## Implementation Details

### Request Structure
```json
{
  "memoryId": "string",    // Required
  "query": "string",        // Required
  "top_ke": 5,             // Optional (default: 5, range: 0-10)
  "top_kc": 2              // Optional (default: 2, range: 1-3)
}
```

### Response Structure (Always Same)
```json
{
  "entries": [...],                              // Array of 0 to top_ke entries
  "count": 5,                                    // Number of entries returned
  "latestContext": "string",                     // Always present
  "latestContextTimestamp": "2025-01-04T12:00:00Z",  // Always present
  "contexts": [...]                              // Array of 0 to top_kc contexts
}
```

### Key Changes
- `contextTimestamp` → `latestContextTimestamp` for clarity
- `contexts` array always present (can be empty)
- Entries include `creationTime` field
- `top_ke=0` is valid (returns no entries, useful for context-only searches)
- `top_kc` must be at least 1 (0 returns validation error)

### MCP Tool Description
The MCP tool description acts as a specification for LLMs, explicitly stating:
- Parameter defaults and ranges
- Response structure with field descriptions
- Temporal context importance
- Sort order (by relevance score descending)

## References
- [Search API Implementation](../../server/internal/api/search_handler.go)
- [MCP Handler](../../mcp/internal/handlers/search_handler.go)
- [API Documentation](../api-reference.md#search-memories)
- Related to ADR-011 (Memory Scoping and Isolation) for memory access patterns
