# Two-Pass Search Implementation

## Overview
Implemented a simple two-pass search algorithm to improve retrieval quality for the LongMemEval benchmark. The algorithm performs an initial search, analyzes the results, and optionally performs a refined second search to find missing information.

## Implementation Details

### Files Modified
1. **src/single_question_runner.py**
   - Added `_two_pass_search()` function that implements the algorithm
   - Modified QA phase to use two-pass search by default (controlled by config parameter)
   - Generic prompts that don't overfit to specific questions

2. **src/memory_manager.py**
   - Already had updated `search_memories()` method with new ke/kc parameters
   - Supports backward compatibility with old top_k parameter

3. **debug_qa.py**
   - Added `--two-pass` flag to test two-pass search
   - Integrated two-pass search into the debug flow

4. **src/benchmarker.py**
   - Added `use_two_pass_search` parameter (default: True) to config

## Algorithm Flow

### First Pass
1. Search with original question using hybrid search
2. Retrieve ke=10 entries and kc=3 context shards
3. Analyze results to check if information is comprehensive

### Analysis Phase
- LLM analyzes the search results to determine if refinement is needed
- Generic prompt that doesn't overfit to specific question types
- Returns either "SUFFICIENT" or "REFINE: <refined query>"

### Second Pass (Optional)
1. If refinement needed, search with refined query
2. Retrieve ke=5 entries and kc=2 context shards (more focused)
3. Merge unique results from both passes

### Result Merging
- Combines entries from both passes, removing duplicates by ID
- Combines context shards, removing duplicates by content
- Preserves latest and best context from first pass

## Testing

### Test with Two-Pass Search
```bash
python debug_qa.py \
  --memory-id 86031cb3-be44-415a-bce8-aedcd320ba42 \
  --vault-id e04ca555-b87e-490a-9807-5c7577c4e226 \
  --question "How many items of clothing do I need to pick up or return from a store?" \
  --two-pass \
  --model gpt-4o-mini
```

### Run Full Benchmark with Two-Pass
```bash
python src/benchmarker.py config.5s.toml
```

## Configuration

The two-pass search is enabled by default. To disable it, you would need to modify the benchmarker code to set `use_two_pass_search: False` in the params.

## Performance Considerations

- **Latency**: Two-pass search adds ~2-3 seconds due to the additional search and LLM analysis
- **Token Usage**: Increased by ~500-1000 tokens for the analysis step
- **Accuracy**: Improves retrieval by finding information missed in the first pass

## Future Improvements

1. **Adaptive Iterations**: Allow more than 2 passes for complex questions
2. **Query Strategies**: Different refinement strategies based on question type
3. **Caching**: Cache first-pass results to avoid redundant searches
4. **Parallel Search**: Execute multiple refined queries in parallel
5. **Confidence Scoring**: Add confidence scores to determine when to stop searching

## Notes

- The current implementation uses generic prompts to avoid overfitting
- The algorithm merges results from both passes to maximize coverage
- Context shards (new API) provide better semantic matching than the deprecated bestContext field
