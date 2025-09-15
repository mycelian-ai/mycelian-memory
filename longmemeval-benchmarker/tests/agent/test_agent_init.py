"""Test MycelianMemoryAgent initialization and graph building."""

import unittest
from unittest.mock import Mock, MagicMock, patch
import sys
from pathlib import Path

# Add src to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from langchain_core.messages import AIMessage, ToolMessage, ChatMessage

from src.mycelian_memory_agent.agent import MycelianMemoryAgent, AgentState
from src.mycelian_memory_agent.control_state import ControlState


class TestAgentInitialization(unittest.TestCase):
    """Test agent initialization and graph building."""

    @patch('src.mycelian_memory_agent.agent.MemorySaver')
    @patch('src.mycelian_memory_agent.agent.ToolNode')
    def test_init_creates_components(self, mock_tool_node, mock_memory_saver):
        """Test that __init__ creates all necessary components."""
        # Mock dependencies
        mock_llm = Mock()
        mock_llm_with_tools = Mock()
        mock_llm.bind_tools = Mock(return_value=mock_llm_with_tools)

        mock_tools = [Mock(name="tool1"), Mock(name="tool2")]
        mock_prompts = {"prompt1": "value1"}

        # Create instance
        agent = MycelianMemoryAgent(
            llm=mock_llm,
            tools=mock_tools,
            prompts=mock_prompts,
            vault_id="test_vault",
            memory_id="test_memory"
        )

        # Verify attributes are set
        self.assertEqual(agent.llm, mock_llm)
        self.assertEqual(agent.tools, mock_tools)
        self.assertEqual(agent.prompts, mock_prompts)
        self.assertEqual(agent.vault_id, "test_vault")
        self.assertEqual(agent.memory_id, "test_memory")

        # Verify bind_tools was called
        mock_llm.bind_tools.assert_called_once_with(mock_tools)
        self.assertEqual(agent.llm_with_tools, mock_llm_with_tools)

        # Verify ToolNode was created
        mock_tool_node.assert_called_once_with(mock_tools)

        # Verify MemorySaver was created
        mock_memory_saver.assert_called_once()

    @patch('src.mycelian_memory_agent.agent.StateGraph')
    @patch('src.mycelian_memory_agent.agent.MemorySaver')
    @patch('src.mycelian_memory_agent.agent.ToolNode')
    def test_build_graph_structure(self, mock_tool_node, mock_memory_saver, mock_state_graph):
        """Test that _build_graph creates the correct graph structure."""
        # Setup mocks
        mock_llm = Mock()
        mock_workflow = Mock()
        mock_compiled_graph = Mock()

        mock_state_graph.return_value = mock_workflow
        mock_workflow.compile = Mock(return_value=mock_compiled_graph)

        # Create agent
        agent = MycelianMemoryAgent(
            llm=mock_llm,
            tools=[],
            prompts={},
            vault_id="vault",
            memory_id="memory"
        )

        # Verify StateGraph was called with AgentState
        mock_state_graph.assert_called_once_with(AgentState)

        # Verify nodes were added
        mock_workflow.add_node.assert_any_call("observe", agent.observe)
        mock_workflow.add_node.assert_any_call("tools", agent.tool_node)

        # Verify entry point
        mock_workflow.set_entry_point.assert_called_once_with("observe")

        # Verify conditional edge
        mock_workflow.add_conditional_edges.assert_called_once()
        call_args = mock_workflow.add_conditional_edges.call_args
        self.assertEqual(call_args[0][0], "observe")
        self.assertEqual(call_args[0][1], agent.should_execute_tools)

        # Verify edge from tools to observe
        mock_workflow.add_edge.assert_called_once_with("tools", "observe")

        # Verify compile was called with checkpointer
        mock_workflow.compile.assert_called_once_with(checkpointer=agent.checkpointer)

        # Verify graph is set
        self.assertEqual(agent.graph, mock_compiled_graph)

    def test_get_tool_by_name(self):
        """Test _get_tool_by_name helper method."""
        # Create mock tools with name attributes
        tool1 = Mock()
        tool1.name = "tool_one"
        tool2 = Mock()
        tool2.name = "tool_two"

        with patch('src.mycelian_memory_agent.agent.MemorySaver'), \
             patch('src.mycelian_memory_agent.agent.ToolNode'):
            agent = MycelianMemoryAgent(
                llm=Mock(),
                tools=[tool1, tool2],
                prompts={},
                vault_id="vault",
                memory_id="memory"
            )

        # Test finding existing tool
        found = agent._get_tool_by_name("tool_one")
        self.assertEqual(found, tool1)

        found = agent._get_tool_by_name("tool_two")
        self.assertEqual(found, tool2)

        # Test non-existent tool raises error
        with self.assertRaises(ValueError) as ctx:
            agent._get_tool_by_name("non_existent")
        self.assertIn("Tool 'non_existent' not found", str(ctx.exception))


if __name__ == "__main__":
    unittest.main()
