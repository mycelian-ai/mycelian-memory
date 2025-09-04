"""Factory function to build the agent with invoker."""

import asyncio
from typing import Optional
from langchain.chat_models import init_chat_model
from .agent import MycelianMemoryAgent
from .agent_invoker import MycelianAgentInvoker
from .mcp_utils import create_mcp_client


def build_agent_with_invoker(
    model_id: str,
    vault_id: str,
    memory_id: str,
    mcp_client: Optional[object] = None,
    max_tool_calls_per_turn: int = 5
) -> MycelianAgentInvoker:
    """Build an agent with invoker for the benchmarker.
    
    Args:
        model_id: LLM model identifier
        vault_id: Vault UUID for memory organization  
        memory_id: Memory UUID for storage
        mcp_client: Optional pre-configured MCP client
        max_tool_calls_per_turn: Maximum tool calls (unused in new implementation)
        
    Returns:
        MycelianAgentInvoker ready to handle messages
    """
    # Create MCP client if not provided
    if mcp_client is None:
        mcp_client = create_mcp_client()
    
    # Get tools from MCP client
    async def _get_tools():
        return await mcp_client.get_tools()
    
    tools = asyncio.run(_get_tools())
    
    # Get prompts from MCP
    # For now, using dummy prompts - in production these would come from MCP
    prompts = {
        "entry_capture_prompt": "",  # Will be loaded from MCP
        "summary_prompt": "",        # Will be loaded from MCP
        "context_prompt": ""         # Will be loaded from MCP
    }
    
    # Try to get actual prompts from MCP if available
    try:
        # This would be the actual MCP call to get prompts
        # prompts = mcp_client.get_prompts("mycelian-memory-streamable")
        pass
    except:
        pass
    
    # Initialize LLM
    resolved_model = model_id if str(model_id).startswith("openai:") else f"openai:{model_id}"
    llm = init_chat_model(resolved_model)
    
    # Create the agent (logging is always enabled)
    agent = MycelianMemoryAgent(
        llm=llm,
        tools=tools,  # Use MCP tools directly with async execution
        prompts=prompts,
        vault_id=vault_id,
        memory_id=memory_id
    )
    
    # Wrap with invoker
    invoker = MycelianAgentInvoker(agent)
    
    # Store some metadata for compatibility
    invoker._mcp = mcp_client  # For QA search later
    invoker._vault_id = vault_id
    invoker._memory_id = memory_id
    
    return invoker