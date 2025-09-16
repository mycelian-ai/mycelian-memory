"""Test the MycelianAgentInvoker class."""

import unittest
from unittest.mock import Mock, MagicMock
import sys
from pathlib import Path

# Add src to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from langchain_core.messages import ChatMessage

from src.mycelian_memory_agent.agent_invoker import MycelianAgentInvoker
from src.mycelian_memory_agent.control_state import ControlState


class TestAgentInvoker(unittest.TestCase):
    """Test the agent invoker's message counting and control logic."""

    def setUp(self):
        """Set up mock agent and invoker."""
        self.mock_agent = Mock()
        self.mock_agent.invoke = Mock(return_value={"status": "success"})
        self.invoker = MycelianAgentInvoker(self.mock_agent)

    def test_init(self):
        """Test invoker initialization."""
        self.assertEqual(self.invoker.agent, self.mock_agent)
        self.assertEqual(self.invoker.msg_count, 0)

    def test_start_session(self):
        """Test start_session method."""
        result = self.invoker.start_session("thread_123")

        # Should reset message count
        self.assertEqual(self.invoker.msg_count, 0)

        # Should call agent.invoke with START_SESSION
        self.mock_agent.invoke.assert_called_once_with(
            control=ControlState.START_SESSION,
            thread_id="thread_123"
        )

        self.assertEqual(result, {"status": "success"})

    def test_process_conversation_message_normal(self):
        """Test process_conversation_message for messages 1-5."""
        # Reset counter
        self.invoker.msg_count = 0

        # Process messages 1-5 (should use PROCESS_MESSAGE)
        for i in range(1, 6):
            self.mock_agent.invoke.reset_mock()

            result = self.invoker.process_conversation_message(
                role="user" if i % 2 else "assistant",
                content=f"Message {i}",
                thread_id="thread_test"
            )

            # Check message count incremented
            self.assertEqual(self.invoker.msg_count, i)

            # Should call with PROCESS_MESSAGE (not flush)
            call_args = self.mock_agent.invoke.call_args
            self.assertEqual(call_args[1]["control"], ControlState.PROCESS_MESSAGE)
            self.assertEqual(call_args[1]["thread_id"], "thread_test")

            # Check message was built correctly
            message = call_args[1]["to_process"]
            self.assertIsInstance(message, ChatMessage)
            self.assertEqual(message.role, "user" if i % 2 else "assistant")
            self.assertEqual(message.content, f"Message {i}")

    def test_process_conversation_message_flush(self):
        """Test that every 6th message triggers flush."""
        # Set count to 5 (next will be 6th)
        self.invoker.msg_count = 5

        result = self.invoker.process_conversation_message(
            role="assistant",
            content="Message 6",
            thread_id="thread_flush"
        )

        # Should be 6 now
        self.assertEqual(self.invoker.msg_count, 6)

        # Should call with PROCESS_MESSAGE_AND_FLUSH
        self.mock_agent.invoke.assert_called_once()
        call_args = self.mock_agent.invoke.call_args
        self.assertEqual(call_args[1]["control"], ControlState.PROCESS_MESSAGE_AND_FLUSH)
        self.assertEqual(call_args[1]["thread_id"], "thread_flush")

        message = call_args[1]["to_process"]
        self.assertEqual(message.content, "Message 6")

    def test_process_conversation_message_flush_pattern(self):
        """Test that flush happens every 6 messages (6, 12, 18, etc)."""
        test_cases = [
            (5, ControlState.PROCESS_MESSAGE_AND_FLUSH),   # 5 -> 6 (flush!)
            (6, ControlState.PROCESS_MESSAGE),             # 6 -> 7 (normal)
            (7, ControlState.PROCESS_MESSAGE),             # 7 -> 8 (normal)
            (11, ControlState.PROCESS_MESSAGE_AND_FLUSH),  # 11 -> 12 (flush!)
            (12, ControlState.PROCESS_MESSAGE),            # 12 -> 13 (normal)
            (17, ControlState.PROCESS_MESSAGE_AND_FLUSH),  # 17 -> 18 (flush!)
            (18, ControlState.PROCESS_MESSAGE),            # 18 -> 19 (normal)
        ]

        for initial_count, expected_control in test_cases:
            self.mock_agent.invoke.reset_mock()
            self.invoker.msg_count = initial_count

            self.invoker.process_conversation_message(
                role="user",
                content=f"Test at {initial_count + 1}",
                thread_id="thread_pattern"
            )

            call_args = self.mock_agent.invoke.call_args
            self.assertEqual(
                call_args[1]["control"],
                expected_control,
                f"Failed at count {initial_count + 1}: expected {expected_control}"
            )

    def test_end_session(self):
        """Test end_session method."""
        # Set some count to verify it doesn't affect end_session
        self.invoker.msg_count = 3

        result = self.invoker.end_session("thread_end")

        # msg_count should remain unchanged (not reset)
        self.assertEqual(self.invoker.msg_count, 3)

        # Should call agent.invoke with END_SESSION
        self.mock_agent.invoke.assert_called_once_with(
            control=ControlState.END_SESSION,
            thread_id="thread_end"
        )

        self.assertEqual(result, {"status": "success"})

    def test_full_session_flow(self):
        """Test a complete session flow with multiple messages."""
        thread_id = "full_session"

        # Start session
        self.invoker.start_session(thread_id)
        self.assertEqual(self.invoker.msg_count, 0)

        # Process 7 messages (6th should flush)
        controls_seen = []
        for i in range(1, 8):
            self.mock_agent.invoke.reset_mock()
            self.invoker.process_conversation_message(
                role="user" if i % 2 else "assistant",
                content=f"Msg {i}",
                thread_id=thread_id
            )
            call_args = self.mock_agent.invoke.call_args
            controls_seen.append(call_args[1]["control"])

        # Verify the pattern
        expected = [
            ControlState.PROCESS_MESSAGE,      # 1
            ControlState.PROCESS_MESSAGE,      # 2
            ControlState.PROCESS_MESSAGE,      # 3
            ControlState.PROCESS_MESSAGE,      # 4
            ControlState.PROCESS_MESSAGE,      # 5
            ControlState.PROCESS_MESSAGE_AND_FLUSH,  # 6 - flush!
            ControlState.PROCESS_MESSAGE,      # 7
        ]
        self.assertEqual(controls_seen, expected)

        # End session
        self.invoker.end_session(thread_id)
        call_args = self.mock_agent.invoke.call_args
        self.assertEqual(call_args[1]["control"], ControlState.END_SESSION)

    def test_chat_message_creation(self):
        """Test that ChatMessage objects are created correctly."""
        test_cases = [
            ("user", "Hello"),
            ("assistant", "Hi there"),
            ("system", "Context info"),
            ("function", "Tool output"),
        ]

        for role, content in test_cases:
            self.mock_agent.invoke.reset_mock()
            self.invoker.msg_count = 0  # Reset to avoid flush

            self.invoker.process_conversation_message(
                role=role,
                content=content,
                thread_id="msg_test"
            )

            call_args = self.mock_agent.invoke.call_args
            message = call_args[1]["to_process"]

            self.assertIsInstance(message, ChatMessage)
            self.assertEqual(message.role, role)
            self.assertEqual(message.content, content)


if __name__ == "__main__":
    unittest.main()
