"""Factory function to build the agent with invoker."""

import asyncio
import os
from typing import Optional
from src.model_providers import get_chat_model
from .agent import MycelianMemoryAgent
from .agent_invoker import MycelianAgentInvoker
from .mcp_utils import create_mcp_client
from src.async_utils import run as run_async


def build_agent_with_invoker(
    model_id: str,
    vault_id: str,
    memory_id: str,
    mcp_client: Optional[object] = None,
    max_tool_calls_per_turn: int = 5,
    context_only: bool = True
) -> MycelianAgentInvoker:
    """Build an agent with invoker for the benchmarker.

    Args:
        model_id: LLM model identifier
        vault_id: Vault UUID for memory organization
        memory_id: Memory UUID for storage
        mcp_client: Optional pre-configured MCP client
        max_tool_calls_per_turn: Maximum tool calls (unused in new implementation)
        context_only: If True, skip add_entry and flush operations (default: True)

    Returns:
        MycelianAgentInvoker ready to handle messages
    """
    # Create MCP client if not provided
    if mcp_client is None:
        mcp_client = create_mcp_client()

    # Get tools from MCP client
    async def _get_tools():
        return await mcp_client.get_tools()

    tools = run_async(_get_tools())

    # Load prompts from the filesystem (fail fast if any are missing/empty)
    # Resolve repo root from this file path
    # From: longmemeval-benchmarker/src/mycelian_memory_agent/build.py
    # Go up 4 levels to get to mycelian-memory/
    root_dir = os.path.dirname(
        os.path.dirname(
            os.path.dirname(
                os.path.dirname(__file__)
            )
        )
    )
    prompts_dir = os.path.join(root_dir, "client", "prompts", "default", "chat")

    entry_path = os.path.join(prompts_dir, "entry_capture_prompt.md")
    summary_path = os.path.join(prompts_dir, "summary_prompt.md")
    context_path = os.path.join(prompts_dir, "context_prompt.md")

    def _read_required(path: str, name: str) -> str:
        if not os.path.exists(path):
            raise RuntimeError(f"Required prompt file not found: {name} at {path}")
        with open(path, "r", encoding="utf-8") as f:
            content = f.read().strip()
        if not content:
            raise RuntimeError(f"Required prompt file is empty: {name} at {path}")
        return content

    entry_capture_prompt = _read_required(entry_path, "entry_capture_prompt.md")
    summary_prompt = _read_required(summary_path, "summary_prompt.md")
    context_prompt = _read_required(context_path, "context_prompt.md")

    prompts = {
        "entry_capture_prompt": entry_capture_prompt,
        "summary_prompt": summary_prompt,
        "context_prompt": context_prompt,
    }

    # Initialize LLM with built-in retry (supports multiple providers)
    llm = get_chat_model(model_id)  # max_retries=6 is default

    # Create the agent (logging is always enabled)
    # Create per-instance loggers for agent and invoker
    import logging
    agent_logger = logging.getLogger(f"lme.agent.{memory_id}")

    agent = MycelianMemoryAgent(
        llm=llm,
        tools=tools,  # Use MCP tools directly with async execution
        prompts=prompts,
        vault_id=vault_id,
        memory_id=memory_id,
        context_only=context_only,
        logger=agent_logger
    )

    # Wrap with invoker, using a per-instance logger
    invoker_logger = logging.getLogger(f"lme.agent.invoker.{memory_id}")
    invoker = MycelianAgentInvoker(agent, logger=invoker_logger)

    # Store some metadata for compatibility
    invoker._mcp = mcp_client  # For QA search later
    invoker._vault_id = vault_id
    invoker._memory_id = memory_id

    return invoker
