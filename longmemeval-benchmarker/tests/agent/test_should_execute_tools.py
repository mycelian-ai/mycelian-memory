"""Test the should_execute_tools conditional edge function."""

import unittest
from unittest.mock import Mock, patch
import sys
from pathlib import Path

# Add src to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from langchain_core.messages import AIMessage, ToolMessage

from src.mycelian_memory_agent.agent import MycelianMemoryAgent, AgentState


class TestShouldExecuteTools(unittest.TestCase):
    """Test the should_execute_tools conditional edge function."""

    def setUp(self):
        """Set up test agent."""
        with patch('src.mycelian_memory_agent.agent.MemorySaver'), \
             patch('src.mycelian_memory_agent.agent.ToolNode'):
            self.agent = MycelianMemoryAgent(
                llm=Mock(),
                tools=[],
                prompts={},
                vault_id="vault",
                memory_id="memory"
            )

    def test_empty_tool_history_returns_execute(self):
        """Test that empty tool_history returns 'execute'."""
        state = {"tool_history": []}
        result = self.agent.should_execute_tools(state)
        self.assertEqual(result, "execute")

    def test_no_tool_history_key_returns_execute(self):
        """Test that missing tool_history key returns 'execute'."""
        state = {}
        result = self.agent.should_execute_tools(state)
        self.assertEqual(result, "execute")

    def test_ai_message_with_tool_calls_returns_execute(self):
        """Test AIMessage with tool_calls returns 'execute'."""
        ai_msg = AIMessage(
            content="",
            tool_calls=[{"name": "test_tool", "args": {}, "id": "123"}]
        )
        state = {"tool_history": [ai_msg]}
        result = self.agent.should_execute_tools(state)
        self.assertEqual(result, "execute")

    def test_ai_message_with_content_returns_end(self):
        """Test AIMessage with content (completion) returns 'end'."""
        ai_msg = AIMessage(content="Task completed.")
        state = {"tool_history": [ai_msg]}
        result = self.agent.should_execute_tools(state)
        self.assertEqual(result, "end")

    def test_tool_message_returns_execute(self):
        """Test ToolMessage returns 'execute' to continue processing."""
        tool_msg = ToolMessage(
            name="test_tool",
            content="Tool result",
            tool_call_id="123"
        )
        state = {"tool_history": [tool_msg]}
        result = self.agent.should_execute_tools(state)
        self.assertEqual(result, "execute")

    def test_mixed_history_checks_last_message(self):
        """Test that only the last message in history is checked."""
        # History with AIMessage with tool calls, then ToolMessage
        ai_msg = AIMessage(
            content="",
            tool_calls=[{"name": "test_tool", "args": {}, "id": "123"}]
        )
        tool_msg = ToolMessage(
            name="test_tool",
            content="Tool result",
            tool_call_id="123"
        )
        state = {"tool_history": [ai_msg, tool_msg]}
        result = self.agent.should_execute_tools(state)
        self.assertEqual(result, "execute")

        # History ending with completion message
        completion_msg = AIMessage(content="Done.")
        state = {"tool_history": [ai_msg, tool_msg, completion_msg]}
        result = self.agent.should_execute_tools(state)
        self.assertEqual(result, "end")

    def test_ai_message_with_both_content_and_tools(self):
        """Test AIMessage with both content and tool_calls prioritizes tool_calls."""
        # This is an edge case - if the message has tool_calls, execute them
        ai_msg = AIMessage(
            content="Some content",
            tool_calls=[{"name": "test_tool", "args": {}, "id": "123"}]
        )
        state = {"tool_history": [ai_msg]}
        result = self.agent.should_execute_tools(state)
        self.assertEqual(result, "execute")

    def test_empty_ai_message_returns_end(self):
        """Test AIMessage with empty content and no tool_calls returns 'end'."""
        ai_msg = AIMessage(content="")
        # Ensure no tool_calls attribute
        if hasattr(ai_msg, 'tool_calls'):
            ai_msg.tool_calls = []
        state = {"tool_history": [ai_msg]}
        result = self.agent.should_execute_tools(state)
        self.assertEqual(result, "end")


if __name__ == "__main__":
    unittest.main()
