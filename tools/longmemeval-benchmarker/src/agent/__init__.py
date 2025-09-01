"""Agent package for LongMemEval benchmarker.

This package contains all agent-related components for the Mycelian Memory Agent.

New architecture (use these):
- AgentBuilder: Sets up infrastructure
- Agent: Defines graph behavior
- AgentMessageProcessor: Handles runtime
- build_agent (from factory): Creates the full pipeline

Legacy (being phased out):
- MycelianMemoryAgent: Old monolithic class
- GraphBuilder: Old graph builder
"""

# New architecture components
from .agent import Agent
from .agent_builder import AgentBuilder, build_agent
from .agent_message_processor import AgentMessageProcessor

# Helper classes (used by both old and new)
from .message_logger import MessageLogger
from .prompt_builder import PromptBuilder
from .state_debugger import StateDebugger
from .tool_logger import ToolLogger

# Legacy components (for backward compatibility)
from .graph_builder import GraphBuilder
from .mycelian_memory_agent import MycelianMemoryAgent
from .mycelian_memory_agent import build_agent as build_agent_legacy

__all__ = [
    # New architecture
    "AgentBuilder",
    "Agent",
    "AgentMessageProcessor",
    "build_agent",
    # Helper classes
    "PromptBuilder",
    "ToolLogger",
    "StateDebugger",
    "MessageLogger",
    # Legacy (deprecated)
    "MycelianMemoryAgent",
    "GraphBuilder",
    "build_agent_legacy",
]