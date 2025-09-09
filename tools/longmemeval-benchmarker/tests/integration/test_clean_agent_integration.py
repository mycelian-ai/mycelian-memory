"""Integration test for the clean agent implementation."""

import unittest
from unittest.mock import Mock, MagicMock, patch
import sys
from pathlib import Path

# Add src to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from langchain_core.messages import ChatMessage, AIMessage, ToolMessage

from src.mycelian_memory_agent.agent import MycelianMemoryAgent
from src.mycelian_memory_agent.agent_invoker import MycelianAgentInvoker
from src.mycelian_memory_agent.control_state import ControlState


class TestCleanAgentIntegration(unittest.TestCase):
    """Integration test for the clean agent implementation."""

    def setUp(self):
        """Set up mock environment for integration testing."""
        # Mock LLM
        self.mock_llm = Mock()
        self.mock_llm_with_tools = Mock()
        self.mock_llm.bind_tools = Mock(return_value=self.mock_llm_with_tools)

        # Mock tools with realistic behavior
        self.mock_get_context = Mock(name="get_context")
        self.mock_get_context.name = "get_context"
        self.mock_get_context.invoke = Mock(return_value="# Previous Context\n- User is Alice\n- Working on recommendation systems")

        self.mock_list_entries = Mock(name="list_entries")
        self.mock_list_entries.name = "list_entries"
        self.mock_list_entries.invoke = Mock(return_value="Entry 1: Alice discussed cold start problem\nEntry 2: Explored hybrid approaches")

        self.mock_add_entry = Mock(name="add_entry")
        self.mock_add_entry.name = "add_entry"
        self.mock_add_entry.invoke = Mock(return_value="Entry added successfully")

        self.mock_await_consistency = Mock(name="await_consistency")
        self.mock_await_consistency.name = "await_consistency"
        self.mock_await_consistency.invoke = Mock(return_value="consistent")

        self.mock_put_context = Mock(name="put_context")
        self.mock_put_context.name = "put_context"
        self.mock_put_context.invoke = Mock(return_value="Context saved")

        self.tools = [
            self.mock_get_context,
            self.mock_list_entries,
            self.mock_add_entry,
            self.mock_await_consistency,
            self.mock_put_context
        ]

        # Mock prompts
        self.prompts = {
            "entry_capture_prompt": "Capture entry following rules",
            "summary_prompt": "Generate summary in past tense",
            "context_prompt": "Maintain context document"
        }

    @patch('src.mycelian_memory_agent.agent.StateGraph')
    @patch('src.mycelian_memory_agent.agent.MemorySaver')
    @patch('src.mycelian_memory_agent.agent.ToolNode')
    def test_full_session_flow(self, mock_tool_node, mock_memory_saver, mock_state_graph):
        """Test a complete session flow with the invoker."""
        # Setup mock graph
        mock_graph = Mock()
        mock_graph.invoke = Mock(return_value={"status": "complete"})
        mock_workflow = Mock()
        mock_workflow.compile = Mock(return_value=mock_graph)
        mock_state_graph.return_value = mock_workflow

        # Create agent and invoker
        agent = MycelianMemoryAgent(
            llm=self.mock_llm,
            tools=self.tools,
            prompts=self.prompts,
            vault_id="test_vault",
            memory_id="test_memory"
        )
        invoker = MycelianAgentInvoker(agent)

        thread_id = "test_thread_001"

        # 1. Start session
        invoker.start_session(thread_id)

        # Verify START_SESSION was invoked
        self.assertEqual(mock_graph.invoke.call_count, 1)
        call_args = mock_graph.invoke.call_args_list[0]
        state = call_args[0][0]
        config = call_args[0][1]
        self.assertEqual(state["control"], ControlState.START_SESSION)
        self.assertEqual(config["configurable"]["thread_id"], thread_id)

        # 2. Process 7 messages to test flush
        messages = [
            ("user", "Hi, I'm Alice"),
            ("assistant", "Hello Alice!"),
            ("user", "I work on recommendation systems"),
            ("assistant", "That's interesting"),
            ("user", "The cold start problem is challenging"),
            ("assistant", "Indeed, it's a common issue"),  # 6th message - should flush
            ("user", "Any suggestions?")  # 7th message
        ]

        for i, (role, content) in enumerate(messages, 1):
            mock_graph.invoke.reset_mock()
            invoker.process_conversation_message(role, content, thread_id)

            # Check the control state
            call_args = mock_graph.invoke.call_args
            state = call_args[0][0]

            if i == 6:
                # 6th message should trigger flush
                self.assertEqual(state["control"], ControlState.PROCESS_MESSAGE_AND_FLUSH)
            else:
                self.assertEqual(state["control"], ControlState.PROCESS_MESSAGE)

            # Verify message was passed correctly
            self.assertEqual(len(state["to_process"]), 1)
            msg = state["to_process"][0]
            self.assertIsInstance(msg, ChatMessage)
            self.assertEqual(msg.role, role)
            self.assertEqual(msg.content, content)

        # 3. End session
        mock_graph.invoke.reset_mock()
        invoker.end_session(thread_id)

        call_args = mock_graph.invoke.call_args
        state = call_args[0][0]
        self.assertEqual(state["control"], ControlState.END_SESSION)
        self.assertEqual(state["to_process"], [])

    @patch('src.mycelian_memory_agent.agent.MemorySaver')
    @patch('src.mycelian_memory_agent.agent.ToolNode')
    def test_observe_logic_with_real_tools(self, mock_tool_node, mock_memory_saver):
        """Test the observe method with more realistic tool interactions."""
        # Create agent
        agent = MycelianMemoryAgent(
            llm=self.mock_llm,
            tools=self.tools,
            prompts=self.prompts,
            vault_id="vault_123",
            memory_id="memory_456"
        )

        # Test START_SESSION sequence
        state = {
            "control": ControlState.START_SESSION,
            "tool_history": [],
            "conversation_history": [],
            "to_process": []
        }

        # First call - should get_context
        result = agent.observe(state)
        self.mock_get_context.invoke.assert_called_once_with({
            "vault_id": "vault_123",
            "memory_id": "memory_456"
        })
        self.assertIn("tool_history", result)
        self.assertEqual(len(result["tool_history"]), 1)

        # Simulate tool execution completed
        state["tool_history"] = [
            ToolMessage(name="get_context", content="Previous context", tool_call_id="1")
        ]

        # Second call - should list_entries
        result = agent.observe(state)
        self.mock_list_entries.invoke.assert_called_once_with({
            "vault_id": "vault_123",
            "memory_id": "memory_456",
            "limit": 10
        })

        # Should update conversation_history
        self.assertIn("conversation_history", result)
        self.assertEqual(len(result["conversation_history"]), 2)

        # Verify context messages
        context_msg = result["conversation_history"][0]
        self.assertEqual(context_msg.role, "system")
        self.assertIn("Previous context", context_msg.content)

        entries_msg = result["conversation_history"][1]
        self.assertEqual(entries_msg.role, "system")
        self.assertIn("Entry 1", entries_msg.content)

    def test_message_counting_accuracy(self):
        """Test that message counting works correctly across sessions."""
        # Mock the agent instead of creating a real one
        mock_agent = Mock()
        mock_agent.invoke = Mock(return_value={"status": "ok"})

        invoker = MycelianAgentInvoker(mock_agent)

        # Start first session
        invoker.start_session("session1")
        self.assertEqual(invoker.msg_count, 0)

        # Process 3 messages
        for i in range(3):
            invoker.process_conversation_message("user", f"Msg {i+1}", "session1")
        self.assertEqual(invoker.msg_count, 3)

        # Start new session - should reset counter
        invoker.start_session("session2")
        self.assertEqual(invoker.msg_count, 0)

        # Process 6 messages in new session
        for i in range(6):
            invoker.process_conversation_message("user", f"Msg {i+1}", "session2")
        self.assertEqual(invoker.msg_count, 6)

        # End session doesn't reset counter
        invoker.end_session("session2")
        self.assertEqual(invoker.msg_count, 6)


if __name__ == "__main__":
    unittest.main()
