"""Type definitions for the LangGraph LongMemEval benchmarker."""

from typing import TypedDict, List, Dict, Any, Optional
from pydantic import BaseModel


class LongMemEvalState(TypedDict):
    """State model for the end-to-end LangGraph workflow."""
    
    # Input data
    conversation_id: str
    haystack_sessions: List[List[Dict[str, Any]]]
    question: str
    expected_answer: str
    memory_id: str
    vault_id: str
    run_id: str  # Unique identifier for this benchmark run
    
    # Intermediate results
    ingestion_complete: bool
    qa_response: str
    
    # Final results
    evaluation_score: float
    benchmark_results: Dict[str, Any]


class ConversationTurn(BaseModel):
    """Individual turn in a conversation session."""
    role: str
    content: str
    has_answer: Optional[bool] = None


class ConversationSession(BaseModel):
    """A session containing multiple conversation turns."""
    session_id: str
    turns: List[ConversationTurn]


class LongMemEvalQuestion(BaseModel):
    """Individual question from the LongMemEval dataset."""
    question_id: str
    question_type: str
    question: str
    answer: str
    question_date: str
    haystack_session_ids: List[str]
    haystack_dates: List[str]
    haystack_sessions: List[List[Dict[str, str]]]
    answer_session_ids: List[str]


class BenchmarkConfig(BaseModel):
    """Configuration for the benchmark run."""
    
    # Dataset configuration
    longmemeval_repo_path: str  # Path to local LongMemEval GitHub repository
    dataset_variant: str = "longmemeval_s.json"  # longmemeval_s.json, longmemeval_m.json, longmemeval_oracle.json
    
    # MCP configuration
    mcp_url: str = "http://localhost:11546/mcp"
    mcp_transport: str = "streamable_http"
    
    # Model configuration
    memory_agent_model: str = "gpt-4o-mini-2024-07-18"
    qa_agent_model: str = "gpt-4o-2024-08-06"
    evaluation_model: str = "anthropic.claude-3-haiku-20240307-v1:0"
    
    # AWS Bedrock configuration
    aws_region: str = "us-east-1"
    aws_access_key_id: Optional[str] = None
    aws_secret_access_key: Optional[str] = None
    
    # Benchmark configuration
    vault_name: str = "longmemeval-benchmark"
    output_dir: str = "./results"
    max_concurrent_conversations: int = 1
    enable_debug_logging: bool = False


class BenchmarkResults(BaseModel):
    """Results from a benchmark run."""
    
    # Overall metrics
    total_questions: int
    correct_answers: int
    overall_accuracy: float
    
    # Per-question-type metrics
    accuracy_by_type: Dict[str, float]
    counts_by_type: Dict[str, int]
    
    # Individual results
    question_results: List[Dict[str, Any]]
    
    # Timing information
    total_runtime_seconds: float
    average_time_per_question: float
    
    # Configuration used
    config: BenchmarkConfig


class MemoryAgentResponse(BaseModel):
    """Response from the Mycelian Memory Agent."""
    success: bool
    message: str
    tool_calls: List[Dict[str, Any]] = []
    error: Optional[str] = None


class QAAgentResponse(BaseModel):
    """Response from the QA Agent."""
    answer: str
    confidence: Optional[float] = None
    sources: List[str] = []
    reasoning: Optional[str] = None


class EvaluationResult(BaseModel):
    """Result from evaluating a QA response."""
    score: float  # 0.0 or 1.0
    reasoning: str
    model_used: str
    evaluation_prompt: str