"""Test structured conversation format for context synthesis."""

import unittest
import sys
from pathlib import Path

# Add src to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from langchain_core.messages import ChatMessage

# These functions will be implemented in agent.py
from src.mycelian_memory_agent.agent import (
    build_structured_conversation,
    format_structured_prompt,
    AGENT_PREFIX
)


class TestStructuredPrompt(unittest.TestCase):
    """Test structured conversation format for proper context synthesis."""

    def test_build_structured_conversation_basic(self):
        """Test building structured conversation sections from messages."""
        # Given messages with previous context and current session
        messages = [
            ChatMessage(role="system", content="[previous_context]\nUser has a dog named Max"),
            ChatMessage(role="system", content="Recent entries: []"),
            ChatMessage(role="user", content="I have a cat named Luna"),
            ChatMessage(role="assistant", content="Nice to meet Luna!")
        ]

        prompts = {
            "context_prompt": "Replace old content with new when topics differ"
        }

        # When we structure them
        sections = build_structured_conversation(messages, prompts)

        # Then we get typed sections
        self.assertEqual(len(sections), 3)
        self.assertEqual(sections[0]["type"], "system_prompt")
        self.assertEqual(sections[1]["type"], "previous_context")
        self.assertEqual(sections[2]["type"], "current_session_messages")

        # System prompt should contain agent prefix and context rules
        self.assertIn(AGENT_PREFIX, sections[0]["content"])
        self.assertIn("Replace old content", sections[0]["content"])

        # Previous context should contain the dog info
        self.assertIn("dog named Max", sections[1]["content"])

        # Current session should have the right messages
        current = sections[2]["content"]
        self.assertEqual(len(current), 2)
        self.assertEqual(current[0]["role"], "user")
        self.assertIn("cat named Luna", current[0]["content"])
        self.assertEqual(current[1]["role"], "assistant")
        self.assertIn("Nice to meet Luna", current[1]["content"])

    def test_build_structured_conversation_no_previous_context(self):
        """Test when there's no previous context, only current session."""
        messages = [
            ChatMessage(role="user", content="Hello world"),
            ChatMessage(role="assistant", content="Hi there!")
        ]

        prompts = {"context_prompt": "Some rules"}

        sections = build_structured_conversation(messages, prompts)

        # Should have system prompt and current session, but no previous context
        self.assertEqual(len(sections), 2)
        self.assertEqual(sections[0]["type"], "system_prompt")
        self.assertEqual(sections[1]["type"], "current_session_messages")

        # Verify no previous_context section
        types = [s["type"] for s in sections]
        self.assertNotIn("previous_context", types)

    def test_build_structured_conversation_only_previous(self):
        """Test when there's only previous context, no current session."""
        messages = [
            ChatMessage(role="system", content="[previous_context]\nOld information"),
            ChatMessage(role="system", content="Recent entries: empty")
        ]

        prompts = {}

        sections = build_structured_conversation(messages, prompts)

        # Should have system prompt and previous context
        self.assertEqual(len(sections), 2)
        self.assertEqual(sections[0]["type"], "system_prompt")
        self.assertEqual(sections[1]["type"], "previous_context")

    def test_format_structured_prompt(self):
        """Test formatting sections into final prompt with markers."""
        sections = [
            {
                "type": "system_prompt",
                "content": "You are an agent\nWith some rules"
            },
            {
                "type": "previous_context",
                "content": "Old information from before"
            },
            {
                "type": "current_session_messages",
                "content": [
                    {"role": "user", "content": "New question"},
                    {"role": "assistant", "content": "New answer"}
                ]
            }
        ]

        prompt = format_structured_prompt(sections)

        # Check all expected markers are present
        self.assertIn("=== SYSTEM INSTRUCTIONS ===", prompt)
        self.assertIn("=== PREVIOUS CONTEXT ===", prompt)
        self.assertIn("=== CURRENT SESSION ===", prompt)

        # Check content is present
        self.assertIn("You are an agent", prompt)
        self.assertIn("Old information from before", prompt)
        self.assertIn("Role: user\nContent: New question", prompt)
        self.assertIn("Role: assistant\nContent: New answer", prompt)

        # Verify ordering - instructions before context before session
        system_idx = prompt.index("=== SYSTEM INSTRUCTIONS ===")
        prev_idx = prompt.index("=== PREVIOUS CONTEXT ===")
        curr_idx = prompt.index("=== CURRENT SESSION ===")

        self.assertLess(system_idx, prev_idx)
        self.assertLess(prev_idx, curr_idx)

    def test_format_structured_prompt_no_previous_context(self):
        """Test formatting when there's no previous context section."""
        sections = [
            {
                "type": "system_prompt",
                "content": "Instructions"
            },
            {
                "type": "current_session_messages",
                "content": [
                    {"role": "user", "content": "Question"}
                ]
            }
        ]

        prompt = format_structured_prompt(sections)

        # Should have system and current markers, but not previous
        self.assertIn("=== SYSTEM INSTRUCTIONS ===", prompt)
        self.assertIn("=== CURRENT SESSION ===", prompt)
        self.assertNotIn("=== PREVIOUS CONTEXT ===", prompt)

    def test_previous_context_detection(self):
        """Test that [previous_context] tag is properly detected."""
        messages = [
            # These should be detected as previous context
            ChatMessage(role="system", content="[previous_context]\nOld stuff"),
            ChatMessage(role="system", content="[previous_context] More old"),
            # These should be current session
            ChatMessage(role="user", content="Not previous context"),
            ChatMessage(role="assistant", content="Also not [previous_context] in middle")
        ]

        prompts = {}
        sections = build_structured_conversation(messages, prompts)

        # Find previous context section
        prev_section = next(s for s in sections if s["type"] == "previous_context")
        self.assertIn("Old stuff", prev_section["content"])
        self.assertIn("More old", prev_section["content"])

        # Find current session section
        curr_section = next(s for s in sections if s["type"] == "current_session_messages")
        self.assertEqual(len(curr_section["content"]), 2)
        self.assertEqual(curr_section["content"][0]["role"], "user")

    def test_complex_conversation_structure(self):
        """Test a complex conversation like the 5K example."""
        messages = [
            ChatMessage(role="system", content="[previous_context]\n# Description\nUser organizing documents"),
            ChatMessage(role="system", content="Recent entries:\n{\"entries\": []}"),
            ChatMessage(role="user", content="I'm training for a 5K run. My personal best is 25:50"),
            ChatMessage(role="assistant", content="Great! Here are training tips..."),
            ChatMessage(role="user", content="Should I focus on intervals?"),
            ChatMessage(role="assistant", content="Yes, intervals will help...")
        ]

        prompts = {
            "context_prompt": "CRITICAL: Replace old unrelated content with new"
        }

        sections = build_structured_conversation(messages, prompts)

        # Verify structure
        self.assertEqual(sections[0]["type"], "system_prompt")
        self.assertEqual(sections[1]["type"], "previous_context")
        self.assertEqual(sections[2]["type"], "current_session_messages")

        # Previous context should have document organization
        self.assertIn("organizing documents", sections[1]["content"])

        # Current session should have 5K conversation
        current = sections[2]["content"]
        self.assertEqual(len(current), 4)  # 2 user + 2 assistant

        # Verify the 5K personal best is in current session
        user_msg = next(m for m in current if "25:50" in m["content"])
        self.assertEqual(user_msg["role"], "user")
        self.assertIn("personal best is 25:50", user_msg["content"])

    def test_empty_messages_list(self):
        """Test handling of empty messages list."""
        messages = []
        prompts = {"context_prompt": "Rules"}

        sections = build_structured_conversation(messages, prompts)

        # Should still have system prompt section
        self.assertEqual(len(sections), 1)
        self.assertEqual(sections[0]["type"], "system_prompt")
        self.assertIn(AGENT_PREFIX, sections[0]["content"])

    def test_multiline_content_preserved(self):
        """Test that multiline content is preserved in formatting."""
        sections = [
            {
                "type": "current_session_messages",
                "content": [
                    {"role": "user", "content": "Line 1\nLine 2\nLine 3"}
                ]
            }
        ]

        prompt = format_structured_prompt(sections)

        # Multiline content should be preserved
        self.assertIn("Line 1\nLine 2\nLine 3", prompt)


if __name__ == "__main__":
    unittest.main()
