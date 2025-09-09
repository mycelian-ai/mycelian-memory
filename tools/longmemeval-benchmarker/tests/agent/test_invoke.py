"""Test the invoke method."""

import unittest
from unittest.mock import Mock, patch
import sys
from pathlib import Path

# Add src to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from langchain_core.messages import ChatMessage

from src.mycelian_memory_agent.agent import MycelianMemoryAgent
from src.mycelian_memory_agent.control_state import ControlState


class TestInvokeMethod(unittest.TestCase):
    """Test the invoke method."""

    def setUp(self):
        """Set up test agent with mocked graph."""
        self.mock_graph = Mock()
        self.mock_graph.invoke = Mock(return_value={"result": "success"})

        with patch('src.mycelian_memory_agent.agent.MemorySaver'), \
             patch('src.mycelian_memory_agent.agent.ToolNode'), \
             patch.object(MycelianMemoryAgent, '_build_graph', return_value=self.mock_graph):
            self.agent = MycelianMemoryAgent(
                llm=Mock(),
                tools=[],
                prompts={},
                vault_id="vault",
                memory_id="memory"
            )

    def test_invoke_start_session(self):
        """Test invoke with START_SESSION control."""
        result = self.agent.invoke(
            control=ControlState.START_SESSION,
            thread_id="thread_123"
        )

        # Check graph.invoke was called
        self.mock_graph.invoke.assert_called_once()

        # Verify the state passed
        call_args = self.mock_graph.invoke.call_args
        initial_state = call_args[0][0]
        config = call_args[0][1]

        # Verify state structure
        self.assertEqual(initial_state["control"], ControlState.START_SESSION)
        self.assertEqual(initial_state["to_process"], [])
        self.assertEqual(initial_state["tool_history"], [])
        self.assertEqual(initial_state["conversation_history"], [])

        # Verify config
        self.assertEqual(config["configurable"]["thread_id"], "thread_123")

        # Verify result
        self.assertEqual(result, {"result": "success"})

    def test_invoke_process_message(self):
        """Test invoke with PROCESS_MESSAGE control and a message."""
        test_message = ChatMessage(role="user", content="Hello")

        result = self.agent.invoke(
            control=ControlState.PROCESS_MESSAGE,
            thread_id="thread_456",
            to_process=test_message
        )

        # Check graph.invoke was called
        self.mock_graph.invoke.assert_called_once()

        # Verify the state passed
        call_args = self.mock_graph.invoke.call_args
        initial_state = call_args[0][0]
        config = call_args[0][1]

        # Verify state structure
        self.assertEqual(initial_state["control"], ControlState.PROCESS_MESSAGE)
        self.assertEqual(initial_state["to_process"], [test_message])
        self.assertEqual(initial_state["tool_history"], [])
        # When processing a message, it's also added to conversation_history
        self.assertEqual(initial_state["conversation_history"], [test_message])

        # Verify config
        self.assertEqual(config["configurable"]["thread_id"], "thread_456")

    def test_invoke_process_message_and_flush(self):
        """Test invoke with PROCESS_MESSAGE_AND_FLUSH."""
        test_message = ChatMessage(role="assistant", content="Response")

        result = self.agent.invoke(
            control=ControlState.PROCESS_MESSAGE_AND_FLUSH,
            thread_id="thread_789",
            to_process=test_message
        )

        # Verify the state passed
        call_args = self.mock_graph.invoke.call_args
        initial_state = call_args[0][0]

        self.assertEqual(initial_state["control"], ControlState.PROCESS_MESSAGE_AND_FLUSH)
        self.assertEqual(initial_state["to_process"], [test_message])
        self.assertEqual(initial_state["conversation_history"], [test_message])

    def test_invoke_end_session(self):
        """Test invoke with END_SESSION control."""
        result = self.agent.invoke(
            control=ControlState.END_SESSION,
            thread_id="thread_end"
        )

        # Verify the state passed
        call_args = self.mock_graph.invoke.call_args
        initial_state = call_args[0][0]
        config = call_args[0][1]

        self.assertEqual(initial_state["control"], ControlState.END_SESSION)
        self.assertEqual(initial_state["to_process"], [])
        self.assertEqual(initial_state["conversation_history"], [])
        self.assertEqual(config["configurable"]["thread_id"], "thread_end")

    def test_invoke_without_message_when_expected(self):
        """Test invoke handles None to_process correctly."""
        # PROCESS_MESSAGE with no message - should still work
        result = self.agent.invoke(
            control=ControlState.PROCESS_MESSAGE,
            thread_id="thread_none",
            to_process=None
        )

        # Verify empty lists for to_process and conversation_history
        call_args = self.mock_graph.invoke.call_args
        initial_state = call_args[0][0]

        self.assertEqual(initial_state["to_process"], [])
        self.assertEqual(initial_state["conversation_history"], [])

    def test_invoke_preserves_thread_id_across_calls(self):
        """Test that thread_id is properly passed to config each time."""
        thread_ids = ["thread_1", "thread_2", "thread_3"]

        for thread_id in thread_ids:
            self.agent.invoke(
                control=ControlState.START_SESSION,
                thread_id=thread_id
            )

        # Verify each call used the correct thread_id
        self.assertEqual(self.mock_graph.invoke.call_count, 3)

        for i, call in enumerate(self.mock_graph.invoke.call_args_list):
            config = call[0][1]
            self.assertEqual(config["configurable"]["thread_id"], thread_ids[i])


if __name__ == "__main__":
    unittest.main()
