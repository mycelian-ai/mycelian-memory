"""Factory function to build the agent with invoker."""

import asyncio
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

    # Load prompts from MCP server
    async def _get_prompts():
        # Get the get_default_prompts tool from MCP
        for tool in tools:
            if getattr(tool, "name", None) == "get_default_prompts":
                result = await tool.ainvoke({"memory_type": "chat"})
                if result and hasattr(result, "content"):
                    import json
                    prompts_data = json.loads(result.content[0].text)
                    return {
                        "entry_capture_prompt": prompts_data.get("entry_capture_prompt", ""),
                        "summary_prompt": prompts_data.get("summary_prompt", ""),
                        "context_prompt": prompts_data.get("context_prompt", "")
                    }
        return None

    # Try to get prompts from MCP
    prompts = run_async(_get_prompts())

    # Fallback to empty prompts if MCP fails
    if not prompts:
        prompts = {
            "entry_capture_prompt": "",
            "summary_prompt": "",
            "context_prompt": ""
        }

    # Initialize LLM with built-in retry (supports multiple providers)
    llm = get_chat_model(model_id)  # max_retries=6 is default

    # Create the agent (logging is always enabled)
    agent = MycelianMemoryAgent(
        llm=llm,
        tools=tools,  # Use MCP tools directly with async execution
        prompts=prompts,
        vault_id=vault_id,
        memory_id=memory_id,
        context_only=context_only
    )

    # Wrap with invoker
    invoker = MycelianAgentInvoker(agent)

    # Store some metadata for compatibility
    invoker._mcp = mcp_client  # For QA search later
    invoker._vault_id = vault_id
    invoker._memory_id = memory_id

    return invoker
