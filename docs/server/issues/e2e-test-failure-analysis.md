# E2E Test Failure Root Cause Analysis

**Date**: 2025-07-26  
**Issue**: E2E tests failing - missing vector generation in test

## Summary

The failing e2e test is not generating vectors for its hybrid search query, while Weaviate is correctly configured with `Vectorizer: "none"` expecting pre-computed vectors.

## Root Cause

The `TestDevEnv_HybridRelevance_TagFilter` test is missing vector generation code. The architecture expects:

1. The indexer uses Ollama with mxbai-embed-large model to generate vectors
2. Weaviate is configured with `Vectorizer: "none"` to receive pre-computed vectors
3. The failing test attempts hybrid search without providing a vector
4. Weaviate throws error: `VectorFromInput was called without vectorizer on class MemoryEntry for input playful cat`
5. Test panics when trying to access nil results

## Technical Details

### Failing Test
- `TestDevEnv_HybridRelevance_TagFilter` in `dev_env_e2e_tests/search_relevance_test.go`
- Line 254: `items := resp2.Data["Get"].(map[string]interface{})["MemoryEntry"].([]interface{})`

### Working Test vs Failing Test

**Working Test** (`TestDevEnv_HybridRelevance_AlphaSweep`):
```go
embedder, err := indexer.NewProvider("ollama", env("EMBED_MODEL", "mxbai-embed-large"))
vec, _ := embedder.Embed(context.Background(), tc.query)
hy := (&gql.HybridArgumentBuilder{}).WithQuery(tc.query).WithVector(vec).WithAlpha(alpha)
```

**Failing Test** (`TestDevEnv_HybridRelevance_TagFilter`):
```go
// Missing embedder creation and vector generation!
hy := (&gql.HybridArgumentBuilder{}).WithQuery("playful cat").WithAlpha(0.6)
// No .WithVector() call
```

### Weaviate Configuration (Correct)
```go
// internal/indexer-prototype/uploader.go
model := &models.Class{
    Class:              u.className,
    Vectorizer:         "none",  // Expects pre-computed vectors
    MultiTenancyConfig: &models.MultiTenancyConfig{Enabled: true},
    // ...
}
```

## Impact

- The `TestDevEnv_HybridRelevance_TagFilter` test will fail
- Any other tests attempting hybrid search without providing vectors will fail
- The architecture is working correctly; it's a test implementation bug

## Solution

**Fix the test by adding vector generation** (This is the correct approach):

```go
// Add embedder creation
embedder, err := indexer.NewProvider("ollama", env("EMBED_MODEL", "mxbai-embed-large"))
if err != nil {
    t.Fatalf("embed provider: %v", err)
}

// Generate vector for the query
vec, _ := embedder.Embed(context.Background(), "playful cat")

// Update the hybrid query builder to include the vector
hy := (&gql.HybridArgumentBuilder{}).WithQuery("playful cat").WithVector(vec).WithAlpha(0.6)
```

This maintains the correct architecture where:
- The indexer handles all vectorization using Ollama
- Weaviate stores and searches pre-computed vectors
- Tests must follow the same pattern as production code

## Note on Title Validation

The title validation changes (lowercase + hyphens only) mentioned in the task description are NOT the root cause. The tests are creating valid titles that pass validation. The issue is purely a missing vector generation in the test code.
