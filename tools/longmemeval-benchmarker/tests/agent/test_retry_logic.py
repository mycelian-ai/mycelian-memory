"""Tests for retry logic in agent LLM calls."""

import unittest
from unittest.mock import Mock, patch, call
import json

from src.mycelian_memory_agent.agent import MycelianMemoryAgent
from src.mycelian_memory_agent.control_state import ControlState
from langchain_core.messages import ChatMessage


class TestRetryLogic(unittest.TestCase):
    """Test the retry logic for LLM calls."""

    @patch('src.mycelian_memory_agent.agent.invoke_with_backoff')
    @patch('src.mycelian_memory_agent.agent.MemorySaver')
    @patch('src.mycelian_memory_agent.agent.ToolNode')
    def test_invoke_llm_with_retry_success(self, mock_tool_node, mock_memory_saver, mock_invoke_backoff):
        """Test that _invoke_llm_with_retry calls invoke_with_backoff correctly."""
        # Setup
        mock_llm = Mock()
        mock_llm_with_tools = Mock()
        mock_llm.bind_tools = Mock(return_value=mock_llm_with_tools)

        # Mock successful response
        mock_response = Mock(content="Test response")
        mock_invoke_backoff.return_value = mock_response

        agent = MycelianMemoryAgent(
            llm=mock_llm,
            tools=[],
            prompts={},
            vault_id="vault",
            memory_id="memory"
        )

        # Execute
        messages = [{"role": "user", "content": "test"}]
        result = agent._invoke_llm_with_retry(messages)

        # Verify
        mock_invoke_backoff.assert_called_once()
        call_args = mock_invoke_backoff.call_args

        # Check that a callable was passed
        self.assertTrue(callable(call_args[0][0]))

        # Check that log function was passed
        self.assertIn('log', call_args[1])
        self.assertTrue(callable(call_args[1]['log']))

        # Result should be the mocked response
        self.assertEqual(result, mock_response)

    @patch('src.mycelian_memory_agent.agent.invoke_with_backoff')
    @patch('src.mycelian_memory_agent.agent.MemorySaver')
    @patch('src.mycelian_memory_agent.agent.ToolNode')
    def test_retry_logging(self, mock_tool_node, mock_memory_saver, mock_invoke_backoff):
        """Test that retry attempts are logged correctly."""
        # Setup
        mock_llm = Mock()
        mock_llm_with_tools = Mock()
        mock_llm.bind_tools = Mock(return_value=mock_llm_with_tools)

        agent = MycelianMemoryAgent(
            llm=mock_llm,
            tools=[],
            prompts={},
            vault_id="vault",
            memory_id="memory"
        )

        # Capture the log function
        messages = [{"role": "user", "content": "test"}]
        agent._invoke_llm_with_retry(messages)

        # Get the log function that was passed
        log_fn = mock_invoke_backoff.call_args[1]['log']

        # Test the log function
        with patch('src.mycelian_memory_agent.agent.logger') as mock_logger:
            log_fn("Test retry message")

            # Verify logging was called
            mock_logger.info.assert_called_once()

            # Check the logged JSON structure
            logged_json = mock_logger.info.call_args[0][0]
            logged_data = json.loads(logged_json)

            self.assertEqual(logged_data['event'], 'llm_retry')
            self.assertEqual(logged_data['message'], 'Test retry message')
            self.assertIn('timestamp', logged_data)

    @patch('src.mycelian_memory_agent.agent.build_add_entry_prompt')
    @patch('src.mycelian_memory_agent.agent.invoke_with_backoff')
    @patch('src.mycelian_memory_agent.agent.MemorySaver')
    @patch('src.mycelian_memory_agent.agent.ToolNode')
    def test_observe_uses_retry(self, mock_tool_node, mock_memory_saver, mock_invoke_backoff, mock_build_prompt):
        """Test that observe method uses retry logic for LLM calls."""
        # Setup
        mock_llm = Mock()
        mock_llm_with_tools = Mock()
        mock_llm.bind_tools = Mock(return_value=mock_llm_with_tools)

        mock_build_prompt.return_value = "Test prompt"
        mock_response = Mock(content="", tool_calls=[{"name": "add_entry", "args": {}, "id": "1"}])
        mock_invoke_backoff.return_value = mock_response

        agent = MycelianMemoryAgent(
            llm=mock_llm,
            tools=[],
            prompts={},
            vault_id="vault",
            memory_id="memory"
        )

        # Execute observe with PROCESS_MESSAGE
        state = {
            "control": ControlState.PROCESS_MESSAGE,
            "tool_history": [],
            "conversation_history": [],
            "to_process": [ChatMessage(role="user", content="test")]
        }

        result = agent.observe(state)

        # Verify invoke_with_backoff was called (not direct llm_with_tools.invoke)
        mock_invoke_backoff.assert_called()
        self.assertEqual(result["tool_history"][0], mock_response)


if __name__ == '__main__':
    unittest.main()
