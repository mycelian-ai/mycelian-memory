"""Test the prompt building functions."""

import unittest
import sys
from pathlib import Path

# Add src to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from langchain_core.messages import ChatMessage

from src.mycelian_memory_agent.agent import (
    format_messages,
    build_add_entry_prompt,
    build_put_context_prompt,
    AGENT_PREFIX
)


class TestPromptFunctions(unittest.TestCase):
    """Test prompt building helper functions."""

    def test_format_messages_single(self):
        """Test format_messages with a single message."""
        messages = [ChatMessage(role="user", content="Hello world")]
        result = format_messages(messages)
        expected = "Role: user\nContent: Hello world"
        self.assertEqual(result, expected)

    def test_format_messages_multiple(self):
        """Test format_messages with multiple messages."""
        messages = [
            ChatMessage(role="user", content="Question"),
            ChatMessage(role="assistant", content="Answer"),
            ChatMessage(role="system", content="Context")
        ]
        result = format_messages(messages)
        expected = "Role: user\nContent: Question\n\nRole: assistant\nContent: Answer\n\nRole: system\nContent: Context"
        self.assertEqual(result, expected)

    def test_format_messages_empty(self):
        """Test format_messages with empty list."""
        messages = []
        result = format_messages(messages)
        self.assertEqual(result, "")

    def test_build_add_entry_prompt_with_context(self):
        """Test build_add_entry_prompt with conversation history."""
        history = [
            ChatMessage(role="system", content="Previous context"),
            ChatMessage(role="user", content="Earlier question")
        ]
        current = ChatMessage(role="user", content="Current message")
        prompts = {
            "entry_capture_prompt": "Capture rules here",
            "summary_prompt": "Summary rules here"
        }

        result = build_add_entry_prompt(history, current, prompts)

        # Check key components are present
        self.assertIn(AGENT_PREFIX, result)
        self.assertIn("PROCESS_MESSAGE", result)
        self.assertIn("Previous context", result)
        self.assertIn("Earlier question", result)
        self.assertIn("Current message", result)
        self.assertIn("Capture rules here", result)
        self.assertIn("Summary rules here", result)
        self.assertIn("Role: user", result)
        self.assertIn("Content: Current message", result)

    def test_build_add_entry_prompt_no_history(self):
        """Test build_add_entry_prompt with no conversation history."""
        history = []
        current = ChatMessage(role="assistant", content="First response")
        prompts = {}

        result = build_add_entry_prompt(history, current, prompts)

        # Check it handles empty history gracefully
        self.assertIn("No previous context available", result)
        self.assertIn("First response", result)
        self.assertIn("Role: assistant", result)

    def test_build_add_entry_prompt_no_message_raises(self):
        """Test build_add_entry_prompt raises when no message to process."""
        history = [ChatMessage(role="user", content="Something")]
        prompts = {}

        with self.assertRaises(ValueError) as ctx:
            build_add_entry_prompt(history, None, prompts)

        self.assertIn("No message to process", str(ctx.exception))

    def test_build_add_entry_prompt_missing_prompts(self):
        """Test build_add_entry_prompt handles missing prompt keys."""
        history = []
        current = ChatMessage(role="user", content="Test")
        prompts = {}  # Empty prompts dict

        result = build_add_entry_prompt(history, current, prompts)

        # Should still work with empty prompt values
        self.assertIn("ENTRY CAPTURE RULES:", result)
        self.assertIn("SUMMARY GENERATION RULES:", result)

    def test_build_put_context_prompt_with_history(self):
        """Test build_put_context_prompt with conversation history."""
        history = [
            ChatMessage(role="user", content="Question 1"),
            ChatMessage(role="assistant", content="Answer 1"),
            ChatMessage(role="user", content="Question 2"),
            ChatMessage(role="assistant", content="Answer 2")
        ]
        prompts = {
            "context_prompt": "Context maintenance rules"
        }

        result = build_put_context_prompt(history, prompts)

        # Check key components
        self.assertIn(AGENT_PREFIX, result)
        self.assertIn("CONTEXT_SYNTHESIS", result)
        self.assertIn("Question 1", result)
        self.assertIn("Answer 1", result)
        self.assertIn("Question 2", result)
        self.assertIn("Answer 2", result)
        self.assertIn("Context maintenance rules", result)

    def test_build_put_context_prompt_no_history_raises(self):
        """Test build_put_context_prompt raises when no history."""
        history = []
        prompts = {}

        with self.assertRaises(ValueError) as ctx:
            build_put_context_prompt(history, prompts)

        self.assertIn("No conversation history to synthesize", str(ctx.exception))

    def test_build_put_context_prompt_missing_prompt(self):
        """Test build_put_context_prompt handles missing context_prompt."""
        history = [ChatMessage(role="user", content="Something")]
        prompts = {}  # No context_prompt key

        result = build_put_context_prompt(history, prompts)

        # Should still work
        self.assertIn("CONTEXT MAINTENANCE RULES:", result)
        self.assertIn("Something", result)

    def test_prompts_preserve_formatting(self):
        """Test that prompts preserve multiline formatting."""
        history = [
            ChatMessage(role="user", content="Line 1\nLine 2\nLine 3")
        ]
        current = ChatMessage(role="assistant", content="Multi\nLine\nResponse")
        prompts = {
            "entry_capture_prompt": "Rule 1\nRule 2\nRule 3"
        }

        result = build_add_entry_prompt(history, current, prompts)

        # Check multiline content is preserved
        self.assertIn("Line 1\nLine 2\nLine 3", result)
        self.assertIn("Multi\nLine\nResponse", result)
        self.assertIn("Rule 1\nRule 2\nRule 3", result)


if __name__ == "__main__":
    unittest.main()
