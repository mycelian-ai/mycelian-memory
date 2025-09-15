"""Test the observe method with mocks to verify control flow logic."""

import unittest
from unittest.mock import Mock, MagicMock, patch
from typing import Dict, Any

from langchain_core.messages import AIMessage, ToolMessage, ChatMessage

import sys
from pathlib import Path
# Add src to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from src.mycelian_memory_agent.agent import MycelianMemoryAgent, AgentState
from src.mycelian_memory_agent.control_state import ControlState


class TestObserveMethod(unittest.TestCase):
    """Test the observe method for each control state."""

    def setUp(self):
        """Set up mocks for testing."""
        # Mock LLM
        self.mock_llm = Mock()
        self.mock_llm_with_tools = Mock()
        self.mock_llm.bind_tools = Mock(return_value=self.mock_llm_with_tools)

        # Mock tools
        self.mock_get_context = Mock(name="get_context")
        self.mock_get_context.name = "get_context"
        self.mock_get_context.invoke = Mock(return_value="Previous context data")

        self.mock_list_entries = Mock(name="list_entries")
        self.mock_list_entries.name = "list_entries"
        self.mock_list_entries.invoke = Mock(return_value="Entry 1\nEntry 2")

        self.mock_add_entry = Mock(name="add_entry")
        self.mock_add_entry.name = "add_entry"

        self.mock_await_consistency = Mock(name="await_consistency")
        self.mock_await_consistency.name = "await_consistency"
        self.mock_await_consistency.invoke = Mock(return_value=None)

        self.mock_put_context = Mock(name="put_context")
        self.mock_put_context.name = "put_context"

        self.tools = [
            self.mock_get_context,
            self.mock_list_entries,
            self.mock_add_entry,
            self.mock_await_consistency,
            self.mock_put_context
        ]

        # Mock prompts
        self.prompts = {
            "entry_capture_prompt": "Capture entry prompt",
            "summary_prompt": "Summary prompt",
            "context_prompt": "Context prompt"
        }

        # Create agent with mocks
        with patch.object(MycelianMemoryAgent, '_build_graph', return_value=Mock()), \
             patch('src.mycelian_memory_agent.agent.ToolNode', return_value=Mock()):
            self.agent = MycelianMemoryAgent(
                llm=self.mock_llm,
                tools=self.tools,
                prompts=self.prompts,
                vault_id="test_vault",
                memory_id="test_memory"
            )

    def test_start_session_first_call(self):
        """Test START_SESSION control state - first call should invoke get_context."""
        state = {
            "control": ControlState.START_SESSION,
            "tool_history": [],
            "conversation_history": [],
            "to_process": []
        }

        result = self.agent.observe(state)

        # Should call get_context directly
        self.mock_get_context.invoke.assert_called_once_with({
            "vault_id": "test_vault",
            "memory_id": "test_memory"
        })

        # Should return tool message in history
        self.assertEqual(len(result["tool_history"]), 1)
        self.assertIsInstance(result["tool_history"][0], ToolMessage)
        self.assertEqual(result["tool_history"][0].name, "get_context")
        self.assertEqual(result["tool_history"][0].content, "Previous context data")

    def test_start_session_second_call(self):
        """Test START_SESSION control state - second call should invoke list_entries."""
        # Simulate we already called get_context
        state = {
            "control": ControlState.START_SESSION,
            "tool_history": [
                ToolMessage(
                    name="get_context",
                    content="Previous context data",
                    tool_call_id="get_context_call"
                )
            ],
            "conversation_history": [],
            "to_process": []
        }

        result = self.agent.observe(state)

        # Should call list_entries directly
        self.mock_list_entries.invoke.assert_called_once_with({
            "vault_id": "test_vault",
            "memory_id": "test_memory",
            "limit": 10
        })

        # Should update conversation_history with context
        self.assertIn("conversation_history", result)
        self.assertEqual(len(result["conversation_history"]), 2)

        # First message should be context
        context_msg = result["conversation_history"][0]
        self.assertIsInstance(context_msg, ChatMessage)
        self.assertEqual(context_msg.role, "system")
        self.assertIn("Previous context data", context_msg.content)

        # Second message should be entries
        entries_msg = result["conversation_history"][1]
        self.assertIsInstance(entries_msg, ChatMessage)
        self.assertEqual(entries_msg.role, "system")
        self.assertIn("Entry 1", entries_msg.content)

        # Should mark complete
        self.assertIsInstance(result["tool_history"][0], AIMessage)
        self.assertEqual(result["tool_history"][0].content, "Session started.")

    def test_process_message_no_message(self):
        """Test PROCESS_MESSAGE without a message should raise error."""
        state = {
            "control": ControlState.PROCESS_MESSAGE,
            "tool_history": [],
            "conversation_history": [],
            "to_process": []  # No message!
        }

        with self.assertRaises(ValueError) as ctx:
            self.agent.observe(state)

        self.assertIn("No message to process", str(ctx.exception))

    @patch('src.mycelian_memory_agent.agent.build_add_entry_prompt')
    def test_process_message_with_message(self, mock_build_prompt):
        """Test PROCESS_MESSAGE with a message should call LLM."""
        mock_build_prompt.return_value = "Test prompt for add_entry"

        # Mock LLM response
        mock_ai_response = AIMessage(
            content="",
            tool_calls=[{"name": "add_entry", "args": {"summary": "test"}, "id": "123"}]
        )
        self.mock_llm_with_tools.invoke.return_value = mock_ai_response

        test_message = ChatMessage(role="user", content="Hello")
        state = {
            "control": ControlState.PROCESS_MESSAGE,
            "tool_history": [],
            "conversation_history": [ChatMessage(role="system", content="Context")],
            "to_process": [test_message]
        }

        result = self.agent.observe(state)

        # Should build prompt
        mock_build_prompt.assert_called_once()

        # Should call LLM
        self.mock_llm_with_tools.invoke.assert_called_once()

        # Should return LLM response
        self.assertEqual(result["tool_history"][0], mock_ai_response)

    def test_process_message_after_add_entry(self):
        """Test PROCESS_MESSAGE after add_entry completes."""
        state = {
            "control": ControlState.PROCESS_MESSAGE,
            "tool_history": [
                AIMessage(content="", tool_calls=[{"name": "add_entry", "args": {}, "id": "123"}]),
                ToolMessage(name="add_entry", content="Entry added", tool_call_id="123")
            ],
            "conversation_history": [],
            "to_process": []
        }

        result = self.agent.observe(state)

        # Should mark complete
        self.assertIsInstance(result["tool_history"][0], AIMessage)
        self.assertEqual(result["tool_history"][0].content, "Message processed.")

    @patch('src.mycelian_memory_agent.agent.build_add_entry_prompt')
    @patch('src.mycelian_memory_agent.agent.build_put_context_prompt')
    def test_flush_sequence(self, mock_put_prompt, mock_add_prompt):
        """Test PROCESS_MESSAGE_AND_FLUSH full sequence."""
        mock_add_prompt.return_value = "Add entry prompt"
        mock_put_prompt.return_value = "Put context prompt"

        # Test first call - should do add_entry
        state = {
            "control": ControlState.PROCESS_MESSAGE_AND_FLUSH,
            "tool_history": [],
            "conversation_history": [],
            "to_process": [ChatMessage(role="user", content="Test")]
        }

        mock_ai_response = AIMessage(content="", tool_calls=[{"name": "add_entry", "args": {}, "id": "1"}])
        self.mock_llm_with_tools.invoke.return_value = mock_ai_response

        result = self.agent.observe(state)
        self.assertEqual(result["tool_history"][0], mock_ai_response)

        # Test second call - after add_entry, should do await_consistency
        state["tool_history"] = [
            AIMessage(content="", tool_calls=[{"name": "add_entry", "args": {}, "id": "1"}]),
            ToolMessage(name="add_entry", content="Added", tool_call_id="123")
        ]

        result = self.agent.observe(state)
        self.mock_await_consistency.invoke.assert_called_once()

        # Should return just the await_consistency message (not accumulate)
        self.assertEqual(len(result["tool_history"]), 1)
        self.assertIsInstance(result["tool_history"][0], ToolMessage)
        self.assertEqual(result["tool_history"][0].name, "await_consistency")

        # Test third call - after await_consistency, should do put_context
        state["tool_history"] = [
            AIMessage(content="", tool_calls=[{"name": "add_entry", "args": {}, "id": "1"}]),
            ToolMessage(name="add_entry", content="Added", tool_call_id="123"),
            ToolMessage(name="await_consistency", content="Done", tool_call_id="456")
        ]

        mock_ai_response2 = AIMessage(content="", tool_calls=[{"name": "put_context", "args": {}, "id": "2"}])
        self.mock_llm_with_tools.invoke.return_value = mock_ai_response2

        result = self.agent.observe(state)
        mock_put_prompt.assert_called_once()
        self.assertEqual(result["tool_history"][0], mock_ai_response2)

        # Test final call - after put_context, should complete
        state["tool_history"] = [
            AIMessage(content="", tool_calls=[{"name": "put_context", "args": {}, "id": "2"}]),
            ToolMessage(name="put_context", content="Saved", tool_call_id="789")
        ]

        result = self.agent.observe(state)
        self.assertEqual(result["tool_history"][0].content, "Flushed to context.")

    def test_end_session_sequence(self):
        """Test END_SESSION sequence."""
        # First call - should do await_consistency
        state = {
            "control": ControlState.END_SESSION,
            "tool_history": [],
            "conversation_history": [ChatMessage(role="user", content="Goodbye")],
            "to_process": []
        }

        result = self.agent.observe(state)
        self.mock_await_consistency.invoke.assert_called_once()
        self.assertIsInstance(result["tool_history"][0], ToolMessage)
        self.assertEqual(result["tool_history"][0].name, "await_consistency")

        # Second call - after await_consistency, should do put_context
        state["tool_history"] = [
            ToolMessage(name="await_consistency", content="Done", tool_call_id="123")
        ]

        mock_ai_response = AIMessage(content="", tool_calls=[{"name": "put_context", "args": {}, "id": "3"}])
        self.mock_llm_with_tools.invoke.return_value = mock_ai_response

        with patch('src.mycelian_memory_agent.agent.build_put_context_prompt') as mock_prompt:
            mock_prompt.return_value = "Put context prompt"
            result = self.agent.observe(state)
            mock_prompt.assert_called_once()

        self.assertEqual(result["tool_history"][0], mock_ai_response)

        # Final call - after put_context, should complete
        state["tool_history"] = [
            AIMessage(content="", tool_calls=[{"name": "put_context", "args": {}, "id": "3"}]),
            ToolMessage(name="put_context", content="Saved", tool_call_id="456")
        ]

        result = self.agent.observe(state)
        self.assertEqual(result["tool_history"][0].content, "Session ended.")


if __name__ == "__main__":
    unittest.main()
