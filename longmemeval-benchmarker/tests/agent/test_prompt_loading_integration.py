"""Integration-ish test to verify prompt loading bug.

This test simulates the MCP get_default_prompts response shape
and asserts that the current parser returns empty prompt strings,
which then results in an empty CONTEXT MAINTENANCE RULES section
in build_put_context_prompt.
"""

import json
import sys
import unittest
from pathlib import Path
from unittest.mock import patch

# Ensure src/ is importable
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from langchain_core.messages import ChatMessage  # type: ignore

from src.mycelian_memory_agent.build import build_agent_with_invoker
from src.mycelian_memory_agent.agent import build_put_context_prompt


class FakeContent:
    def __init__(self, text: str) -> None:
        self.text = text


class FakeToolResult:
    def __init__(self, text: str) -> None:
        self.content = [FakeContent(text)]


class FakeGetDefaultPromptsTool:
    name = "get_default_prompts"

    async def ainvoke(self, args):  # noqa: D401 - simple async stub
        # Simulate MCP server JSON payload
        payload = {
            "version": "v1",
            "context_summary_rules": "RULES: Use current session over old context.",
            "templates": {
                "entry_capture_prompt": "ENTRY RULES",  # should be used but will be ignored by the bug
                "summary_prompt": "SUMMARY RULES",
                "context_prompt": "CONTEXT RULES (CRITICAL)"
            },
        }
        return FakeToolResult(json.dumps(payload))


class FakeMCPClient:
    async def get_tools(self):  # noqa: D401 - simple async stub
        return [FakeGetDefaultPromptsTool()]


class TestPromptLoadingIntegration(unittest.TestCase):
    def test_prompt_loading_bug_yields_empty_prompts(self):
        # Minimal fake LLM and Agent to avoid LangChain tool binding requirements
        class FakeLLM:
            def bind_tools(self, tools):
                return self

        class FakeAgent:
            def __init__(self, llm, tools, prompts, vault_id, memory_id, context_only=True):
                self.prompts = prompts

        # Patch MCP client factory, model provider, and Agent class in build module
        with patch("src.mycelian_memory_agent.build.create_mcp_client", return_value=FakeMCPClient()):
            with patch("src.mycelian_memory_agent.build.get_chat_model", return_value=FakeLLM()):
                with patch("src.mycelian_memory_agent.build.MycelianMemoryAgent", FakeAgent):
                    invoker = build_agent_with_invoker(
                        model_id="stub",
                        vault_id="v",
                        memory_id="m",
                        mcp_client=None,
                        context_only=True,
                    )

        # Verify prompts are correctly initialized from templates
        self.assertEqual(invoker.agent.prompts.get("context_prompt"), "CONTEXT RULES (CRITICAL)")
        self.assertEqual(invoker.agent.prompts.get("entry_capture_prompt"), "ENTRY RULES")
        self.assertEqual(invoker.agent.prompts.get("summary_prompt"), "SUMMARY RULES")

        # Verify the context rules appear in the synthesis prompt
        history = [ChatMessage(role="user", content="hello world")]
        prompt = build_put_context_prompt(history, invoker.agent.prompts, "v", "m")
        self.assertIn("CONTEXT MAINTENANCE RULES:", prompt)
        self.assertIn("CONTEXT RULES (CRITICAL)", prompt)
        self.assertIn("hello world", prompt)


if __name__ == "__main__":
    unittest.main()
