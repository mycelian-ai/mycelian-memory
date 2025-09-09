# ADR-019: QA Agent with Iterative Search Refinement

## Status
Proposed

## Context

The current LongMemEval benchmark implementation uses a simple single-query search approach that struggles with complex questions, particularly those requiring information from multiple scattered contexts. Analysis of the benchmark results reveals several challenges:

1. **Semantic Pollution**: Common terms like "return" appear in multiple unrelated contexts (investment returns, function returns, clothing returns), causing retrieval noise
2. **Information Fragmentation**: Answers often span multiple conversation sessions with different terminology
3. **Query Ambiguity**: Single queries cannot capture the nuanced intent of complex questions
4. **Fixed Search Strategy**: No ability to refine based on initial results

For example, the question "How many items of clothing do I need to pick up or return from a store?" requires finding three separate items mentioned across different conversations using varied terminology (exchange, dry cleaning, lent).

## Decision

Implement an **Agentic Search System** that iteratively refines queries based on search results and identified noise patterns. The agent will:

1. Analyze question intent before searching
2. Execute phased search strategies with refinement
3. Identify and filter noise patterns
4. Adaptively determine when sufficient information is found

## Proposed Architecture

### Core Components

```python
class SearchAgent:
    """Agentic search with iterative refinement based on observations."""

    def __init__(self, memory_id: str, memory_manager: MemoryManager, model_id: str):
        self.memory_id = memory_id
        self.mm = memory_manager
        self.model_id = model_id
        self.search_history = []
        self.accumulated_results = {
            'entries': [],
            'contexts': [],
            'noise_patterns': [],
            'signal_patterns': []
        }
```

### Search Phases

#### Phase 1: Intent Analysis
Understand the semantic meaning and requirements of the question:
- Core intent and expected answer type
- Key entities and temporal scope
- Action verbs and potential ambiguities

#### Phase 2: Discovery Search
Broad initial search to understand the information landscape:
- Use original question for semantic matching
- Gather diverse results to identify patterns
- Analyze results to distinguish signal from noise

#### Phase 3: Iterative Refinement
Adaptive search based on observations:
- **FILTER_REFINE**: Add filters to exclude identified noise
- **EXPAND_SEARCH**: Broaden search for missing information
- **TARGETED_SEARCH**: Focus on specific identified gaps

### Search Actions

```python
class SearchAction(Enum):
    SEMANTIC_SEARCH = "semantic_search"    # Initial broad search
    FILTER_REFINE = "filter_refine"         # Exclude noise patterns
    EXPAND_SEARCH = "expand_search"         # Find missing pieces
    TARGETED_SEARCH = "targeted_search"     # Specific item search
    ANALYZE_RESULTS = "analyze"             # Deep dive into results
    COMPLETE = "complete"                   # Sufficient information found
```

### Decision Logic

The agent decides next actions based on:
- Search history and accumulated results
- Identified noise vs signal patterns
- Coverage of the original question's requirements
- Diminishing returns from additional searches

### Example Flow

For "How many items of clothing do I need to pick up or return from a store?":

**Iteration 1: SEMANTIC_SEARCH**
- Query: "items clothing pick up return store"
- Found: Mix of contexts including investment returns, e-commerce setup
- Identified noise: "investment returns", "returns management"
- Identified signal: "Zara", "dry cleaning", clothing items

**Iteration 2: FILTER_REFINE**
- Query: "Zara boots exchange OR dry cleaning blazer -investment -ecommerce"
- Found: Specific mentions of Zara boots and dry cleaning
- Still missing: Possible third item

**Iteration 3: TARGETED_SEARCH**
- Query: "sweater lent sister borrow"
- Found: Green sweater lent to sister
- Complete: All three items identified

## Implementation Details

### Integration with LongMemEval

```python
# In single_question_runner.py
def _agentic_search(memory_id: str, question: str, mm: MemoryManager, model_id: str) -> Dict:
    """Use search agent for complex questions."""

    agent = SearchAgent(memory_id, mm, model_id)
    results = agent.search(question, max_iterations=3)

    # Log search trajectory for debugging
    logger.info(f"Search completed in {len(agent.search_history)} iterations")
    for step in agent.search_history:
        logger.debug(f"  {step['action']}: {step['query']} -> {step['results_count']} results")

    return results
```

### Noise Pattern Recognition

```python
def _identify_noise_patterns(self, results: Dict, question_intent: Dict):
    """Identify irrelevant contexts that match keywords."""

    prompt = """Analyze these search results to identify noise vs signal:

    Question Intent: {intent}
    Results: {results_summary}

    Identify:
    1. NOISE: Contexts matching keywords but irrelevant to the question
    2. SIGNAL: Contexts relevant to answering the question

    Return patterns to filter and patterns to pursue."""
```

### Completion Detection

```python
def _is_sufficient(self, question: str, intent: Dict) -> bool:
    """Determine if accumulated results are sufficient."""

    if intent['answer_type'] == 'count':
        # Check if we have all items for counting
        return self._verify_count_completeness()
    elif intent['answer_type'] == 'list':
        # Check if list seems complete
        return self._verify_list_completeness()
    else:
        # Check general sufficiency
        return self._verify_general_sufficiency()
```

## Benefits

1. **Improved Recall**: Iterative refinement finds scattered information
2. **Better Precision**: Active noise filtering reduces irrelevant results
3. **Adaptive Strategy**: Adjusts approach based on what's found
4. **Explainable Process**: Search history provides debugging insight
5. **Graceful Degradation**: Works with partial information

## Drawbacks

1. **Increased Latency**: Multiple search rounds take more time
2. **Higher Token Usage**: LLM calls for decision-making and analysis
3. **Complexity**: More complex than single-query search
4. **Potential Over-searching**: May continue searching when answer is already found

## Alternatives Considered

1. **Multi-query Parallel Search**: Execute multiple queries simultaneously
   - Pros: Lower latency
   - Cons: No adaptive refinement based on results

2. **Query Expansion Only**: Generate multiple query variants upfront
   - Pros: Simpler implementation
   - Cons: Cannot adapt to discovered noise patterns

3. **Reranking Pipeline**: Single search with ML reranking
   - Pros: Fast, single pass
   - Cons: Cannot find information missed in initial search

## Migration Path

1. Implement `SearchAgent` as optional component
2. Add feature flag to enable agentic search
3. Run A/B comparison on LongMemEval dataset
4. Graduate to default if metrics improve

## Success Metrics

- Improvement in LongMemEval accuracy, particularly for multi-item questions
- Reduction in false positives from semantic pollution
- Average iterations to completion (target: ≤3)
- Search latency (target: <5s for 95th percentile)

## Related ADRs

- ADR-017: Search API Improvements (ke/kc parameters)
- ADR-018: LongMemEval Resumable Benchmarker

## References

- LongMemEval benchmark paper
- Analysis of "3 items" question retrieval failure
- Hybrid search with alpha parameter analysis
