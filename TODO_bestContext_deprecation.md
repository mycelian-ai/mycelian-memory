# TODO: Complete BestContext Deprecation

## Background
The `ke` (entries top-k) and `kc` (context shards top-k) parameters have been implemented to replace the legacy `bestContext` functionality. However, the deprecation is incomplete, leaving technical debt in the codebase.

## Current State
- ✅ `ke` and `kc` parameters fully implemented in API
- ✅ MCP handler properly exposes `ke` and `kc`
- ✅ API returns `contexts` array when `kc > 0`
- ❌ `BestContext` still in searchindex.Index interface
- ❌ Client response types still have BestContext fields
- ❌ API never actually calls or populates BestContext

## Tasks

### 1. Remove BestContext from Interface
**File**: `server/internal/searchindex/index.go`
- [ ] Remove `BestContext()` method from Index interface
- [ ] Update all implementations (weaviate_native.go, etc.)

### 2. Update Client Response Types
**File**: `client/internal/types/responses.go`
- [ ] Remove `BestContext` field from SearchResponse
- [ ] Remove `BestContextTimestamp` field from SearchResponse
- [ ] Remove `BestContextScore` field from SearchResponse
- [ ] Add migration note in comments

### 3. Fix Test Mocks
**Files**: 
- `server/internal/api/search_handler_test.go`
- `server/internal/outbox/worker_test.go`
- `server/internal/services/vault_test.go`
- `server/internal/searchindex/healthchecker_test.go`
- [ ] Remove BestContext from all mock implementations
- [ ] Update test assertions that might expect BestContext

### 4. Update Weaviate Implementation
**File**: `server/internal/searchindex/weaviate_native.go`
- [ ] Remove BestContext implementation
- [ ] Ensure SearchContexts fully replaces its functionality

### 5. Documentation Updates
- [ ] Update API documentation to show migration path
- [ ] Document that `contexts[0]` replaces `bestContext` when sorted by score
- [ ] Update any examples that reference bestContext

### 6. Backward Compatibility (Optional)
- [ ] Consider if we need a deprecation period
- [ ] If yes, mark fields as deprecated with warnings
- [ ] Set removal target version

## Migration Guide for API Consumers

### Before (using bestContext)
```json
{
  "bestContext": "...",
  "bestContextScore": 0.95,
  "bestContextTimestamp": "2024-01-01T00:00:00Z"
}
```

### After (using contexts array)
```json
{
  "contexts": [
    {
      "context": "...",
      "score": 0.95,
      "timestamp": "2024-01-01T00:00:00Z"
    }
  ]
}
```

To get the best context: `response.contexts[0]` (contexts are sorted by score descending)

## Testing Checklist
- [ ] All unit tests pass after removal
- [ ] Integration tests updated
- [ ] MCP server tests updated
- [ ] Client SDK tests updated
- [ ] Benchmarker still works with search

## Notes
- The API handler already doesn't use BestContext, so removing it won't affect current functionality
- Main impact will be on client libraries that might expect these fields
- Consider semantic versioning implications (this would be a breaking change)