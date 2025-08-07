from __future__ import annotations

from dataclasses import dataclass
from typing import List, Dict, Any, Optional


@dataclass
class ConversationMessage:
    speaker: str  # "user" or "assistant"
    text: str
    session_id: int
    timestamp: Optional[str] = None


@dataclass
class TestQuestion:
    question: str
    expected_answer: str
    question_type: str  # e.g. "cross_session"


@dataclass
class BenchmarkConversation:
    conversation_id: str
    messages: List[ConversationMessage]
    test_questions: List[TestQuestion]
    metadata: Dict[str, Any] 