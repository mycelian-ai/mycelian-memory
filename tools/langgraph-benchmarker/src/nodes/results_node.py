"""Results Aggregation Node - Compile final benchmark results."""

import structlog
from typing import Dict, Any

from ..types import LongMemEvalState

logger = structlog.get_logger(__name__)


def determine_question_type(question_id: str) -> str:
    """
    Determine question type from question_id based on LongMemEval conventions.
    
    Args:
        question_id: The question identifier
        
    Returns:
        Question type string
    """
    # Handle abstention questions (end with _abs)
    if question_id.endswith('_abs'):
        return 'abstention'
    
    # Map based on common patterns in LongMemEval
    # These mappings come from the original evaluate_qa.py
    type_mapping = {
        'single_hop': 'single-session-user',
        'implicit_preference_v2': 'single-session-preference',
        'assistant_previnfo': 'single-session-assistant',
        'two_hop': 'multi-session',
        'multi_session_synthesis': 'multi-session',
        'temp_reasoning_implicit': 'temporal-reasoning',
        'temp_reasoning_explicit': 'temporal-reasoning',
        'knowledge_update': 'knowledge-update'
    }
    
    # Try to match based on question_id patterns
    for pattern, qtype in type_mapping.items():
        if pattern in question_id:
            return qtype
    
    # Default fallback
    return 'unknown'


async def results_aggregation_node(state: LongMemEvalState) -> LongMemEvalState:
    """
    Aggregate final benchmark results for a single conversation.
    
    This LangGraph node compiles the results from QA and evaluation
    into the final benchmark output format.
    
    Args:
        state: Workflow state with evaluation results
        
    Returns:
        Updated state with benchmark_results populated
    """
    conversation_id = state["conversation_id"]
    
    logger.info(
        "Aggregating results",
        conversation_id=conversation_id,
        evaluation_score=state["evaluation_score"]
    )

    try:
        # Determine question type
        question_type = determine_question_type(conversation_id)
        
        # Aggregate results
        benchmark_results = {
            # Core identification
            "conversation_id": conversation_id,
            "question_type": question_type,
            
            # Question and answer data
            "question": state["question"],
            "expected_answer": state["expected_answer"],
            "qa_response": state["qa_response"],
            
            # Evaluation results
            "evaluation_score": state["evaluation_score"],
            "correct": state["evaluation_score"] == 1.0,
            
            # Memory infrastructure used
            "vault_id": state.get("vault_id"),
            "memory_id": state.get("memory_id"),
            
            # Processing metadata
            "ingestion_complete": state["ingestion_complete"],
            "num_haystack_sessions": len(state["haystack_sessions"]),
            "total_haystack_turns": sum(
                len(session) for session in state["haystack_sessions"]
            ),
            
            # Timestamps would be added here if needed
            # "processed_at": datetime.utcnow().isoformat(),
        }

        state["benchmark_results"] = benchmark_results

        logger.info(
            "Results aggregation complete",
            conversation_id=conversation_id,
            question_type=question_type,
            correct=benchmark_results["correct"]
        )

    except Exception as e:
        logger.error(
            "Error aggregating results",
            conversation_id=conversation_id,
            error=str(e)
        )
        # Create minimal error result
        state["benchmark_results"] = {
            "conversation_id": conversation_id,
            "error": str(e),
            "evaluation_score": 0.0,
            "correct": False
        }

    return state


class ResultsAggregationNode:
    """
    Class-based results aggregation node for more complex scenarios.
    """

    def __init__(self, include_detailed_metrics: bool = True):
        """
        Initialize the results aggregation node.
        
        Args:
            include_detailed_metrics: Whether to include detailed timing/memory metrics
        """
        self.include_detailed_metrics = include_detailed_metrics

    async def aggregate_results(self, state: LongMemEvalState) -> LongMemEvalState:
        """
        Aggregate comprehensive results including optional detailed metrics.
        
        Args:
            state: Workflow state
            
        Returns:
            Updated state with detailed benchmark results
        """
        # Start with basic aggregation
        state = await results_aggregation_node(state)
        
        if self.include_detailed_metrics:
            # Add detailed metrics if enabled
            results = state["benchmark_results"]
            
            # Add session-level breakdown
            results["session_breakdown"] = [
                {
                    "session_idx": idx,
                    "turn_count": len(session),
                    "roles": [turn.get("role", "unknown") for turn in session]
                }
                for idx, session in enumerate(state["haystack_sessions"])
            ]
            
            # Add question complexity metrics
            results["question_length"] = len(state["question"])
            results["expected_answer_length"] = len(state["expected_answer"])
            results["response_length"] = len(state["qa_response"])
            
            state["benchmark_results"] = results

        return state


def create_results_node(include_detailed_metrics: bool = False) -> callable:
    """
    Create a results aggregation node function for LangGraph workflows.
    
    Args:
        include_detailed_metrics: Whether to include detailed metrics
        
    Returns:
        Async function for use as LangGraph node
    """
    if include_detailed_metrics:
        node_instance = ResultsAggregationNode(include_detailed_metrics=True)
        return node_instance.aggregate_results
    else:
        return results_aggregation_node