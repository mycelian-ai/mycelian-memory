# TODO: Add include_raw_entries Parameter and Increase Search Limits

## Goals
1. Add a parameter to control whether raw entries are RETURNED in search results to reduce token usage (default: false)
   - **Important**: Raw entries will STILL be searched/evaluated for better search quality
   - Only the response will exclude rawEntry field content to save tokens
2. Increase max allowed search results: top_ke from 10 to 25, top_kc from 3 to 10

## Implementation Tasks

### 1. API Request Types
- [ ] Add `IncludeRawEntries bool` field to `/client/internal/types/requests.go` SearchRequest
- [ ] Add `IncludeRawEntries bool` field to `/server/internal/api/search.go` SearchRequest
- [ ] Update SearchRequest.Validate() to handle new field with default false
- [ ] Update top_ke validation range from 0-10 to 0-25 in `/server/internal/api/search.go`
- [ ] Update top_kc validation range from 1-3 to 1-10 in `/server/internal/api/search.go`

### 2. Search Handler
- [ ] Update `/server/internal/api/search_handler.go` HandleSearch to pass includeRawEntries flag to index.Search()

### 3. Search Index Interface
- [ ] Update searchindex.Index interface Search method signature to accept includeRawEntries parameter
- [ ] Update Weaviate implementation in `/server/internal/searchindex/weaviate_native.go`
- [ ] Update any mock implementations

### 4. Model Layer
- [ ] Modify search index implementations to conditionally populate RawEntry field in SearchHit
- [ ] When includeRawEntries is false, set RawEntry to empty string
- [ ] **Note**: Weaviate will still search against rawEntry for quality, but we won't return it in the response

### 5. MCP Server
- [ ] Add `include_raw_entries` parameter to tool definition in `/mcp/internal/handlers/search_handler.go`
- [ ] Default to false
- [ ] Update top_ke range from 0-10 to 0-25 in tool validation
- [ ] Update top_kc range from 1-3 to 1-10 in tool validation
- [ ] Update tool description to reflect new ranges
- [ ] Pass parameter through to client.Search()

### 6. Client SDK
- [ ] Update client.Search() method to accept IncludeRawEntries parameter
- [ ] Ensure backward compatibility

### 7. Testing
- [ ] Write unit tests for parameter validation
- [ ] Write integration tests with include_raw_entries=true
- [ ] Write integration tests with include_raw_entries=false
- [ ] Test new top_ke max limit (25 results)
- [ ] Test new top_kc max limit (10 results)
- [ ] Test MCP tool with new parameter
- [ ] Verify backward compatibility (omitted parameter defaults to false)

## Files to Modify
1. `/client/internal/types/requests.go`
2. `/server/internal/api/search.go`
3. `/server/internal/api/search_handler.go`
4. `/server/internal/searchindex/interface.go` (if exists, or wherever Index interface is defined)
5. `/server/internal/searchindex/weaviate_native.go`
6. `/mcp/internal/handlers/search_handler.go`
7. `/client/client.go` (or wherever Search method is implemented)
8. Test files for all modified components

## Notes
- Default behavior: exclude raw entries from response to reduce token usage
- Search quality preserved: Weaviate still searches against rawEntry content
- Maintain backward compatibility
- No breaking changes to response structure (field present but empty when excluded)

## Implementation Decision
**Approach**: Keep searching against rawEntry for quality, but conditionally exclude from response
- Weaviate hybrid search will continue to use both `summary` and `rawEntry` in `WithProperties()`
- The `rawEntry` field will always be fetched from Weaviate in `WithFields()`
- When `include_raw_entries=false`, we'll clear the rawEntry field before returning the response
- This preserves search quality while reducing token usage in responses