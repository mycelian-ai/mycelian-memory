"""AgentBuilder - Orchestrates building an Agent with all dependencies."""

import asyncio
import os
from typing import Any, Dict, Optional

from langchain.chat_models import init_chat_model  # type: ignore
from langchain_mcp_adapters.client import MultiServerMCPClient  # type: ignore
from langgraph.checkpoint.memory import InMemorySaver  # type: ignore

from .message_logger import MessageLogger
from .prompt_builder import PromptBuilder
from .state_debugger import StateDebugger
from .tool_logger import ToolLogger


class AgentBuilder:
    """Orchestrates building an Agent with all dependencies.
    
    This class is responsible for:
    - Setting up infrastructure (MCP client, tools)
    - Creating the LLM instance
    - Building the system prompt using PromptBuilder
    - Initializing helper objects
    - Returning a configured Agent
    
    It reuses existing helper classes and extracts setup logic
    from the monolithic MycelianMemoryAgent.
    """
    
    # Constants (extracted from MycelianMemoryAgent)
    DEFAULT_MCP_URL = "http://localhost:11546/mcp"
    SERVER_NAME = "mycelian-memory-streamable"
    
    def __init__(
        self,
        model_id: str,
        vault_id: str,
        memory_id: str,
        max_tool_calls_per_turn: int = 5,
        server_url: Optional[str] = None,
        debug: bool = False,
        mcp_client: Optional[MultiServerMCPClient] = None
    ):
        """Initialize AgentBuilder with configuration.
        
        Args:
            model_id: LLM model identifier
            vault_id: Vault UUID for memory organization
            memory_id: Memory UUID for storage
            max_tool_calls_per_turn: Maximum tool calls allowed per turn
            server_url: Optional MCP server URL (ignored if mcp_client provided)
            debug: Enable debug logging
            mcp_client: Optional pre-configured MCP client to use (dependency injection)
        """
        # Store configuration immutably
        self.config = {
            'model_id': model_id,
            'vault_id': str(vault_id),
            'memory_id': str(memory_id),
            'max_tools': max_tool_calls_per_turn,
            'server_url': server_url or os.environ.get("MYCELIAN_MCP_URL", self.DEFAULT_MCP_URL),
            'debug': bool(debug)
        }
        
        # Store injected MCP client if provided
        self.mcp_client = mcp_client
    
    def build(self) -> 'AgentMessageProcessor':
        """Build the complete agent pipeline.
        
        This orchestrates the creation of all components:
        1. Sets up infrastructure (MCP, tools, LLM)
        2. Creates the Agent with graph topology
        3. Returns AgentMessageProcessor for runtime
        
        Returns:
            AgentMessageProcessor ready to handle messages
        """
        # 1. Setup MCP client and load tools
        # Use injected client or create new one
        if self.mcp_client:
            mcp_client = self.mcp_client
        else:
            mcp_client = self._create_mcp_client()
        
        tools = self._load_tools(mcp_client)
        
        # 2. Create LLM instance
        llm = init_chat_model(self.config['model_id'])
        
        # 3. Build system prompt using existing PromptBuilder
        prompt_builder = PromptBuilder(
            vault_id=self.config['vault_id'],
            memory_id=self.config['memory_id'],
            mcp_client=mcp_client,
            server_name=self.SERVER_NAME
        )
        system_prompt = prompt_builder.build_system_prompt()
        
        # 4. Create helper objects using existing classes
        helpers = {
            'tool_logger': ToolLogger(debug=self.config['debug']),
            'state_debugger': StateDebugger(debug=self.config['debug']),
            'message_logger': MessageLogger(debug=self.config['debug']),
            'mcp_client': mcp_client  # Agent might need for cleanup
        }
        
        # 5. Apply debug logging to tools if enabled
        if self.config['debug']:
            tool_logger = helpers['tool_logger']
            for tool in tools:
                try:
                    tool_logger.attach_logging_to_tool(tool)
                except Exception:
                    # Silent failure for tool logging attachment
                    pass
        
        # 6. Create checkpointer for conversation state
        checkpointer = InMemorySaver()
        
        # 7. Import classes here to avoid circular dependency
        from .agent import Agent
        from .agent_message_processor import AgentMessageProcessor
        
        # 8. Create Agent with all dependencies
        agent = Agent(
            llm=llm,
            tools=tools,
            system_prompt=system_prompt,
            checkpointer=checkpointer,
            config=self.config,
            helpers=helpers
        )
        
        # 9. Create and return AgentMessageProcessor
        return AgentMessageProcessor(
            agent=agent,
            config=self.config,
            helpers=helpers
        )
    
    def _create_mcp_client(self) -> MultiServerMCPClient:
        """Create and configure MCP client.
        
        Extracted from MycelianMemoryAgent._setup_mcp_client()
        
        Returns:
            Configured MCP client instance
        """
        return MultiServerMCPClient({
            self.SERVER_NAME: {
                "url": self.config['server_url'],
                "transport": "streamable_http"
            }
        })
    
    def _load_tools(self, mcp_client: MultiServerMCPClient) -> list:
        """Load tools from MCP server.
        
        Extracted from MycelianMemoryAgent._setup_tools()
        
        Args:
            mcp_client: MCP client instance
            
        Returns:
            List of available tools
        """
        async def _load():
            return await mcp_client.get_tools()  # type: ignore[attr-defined]
        
        return asyncio.run(_load())


def build_agent(
    model_id: str,
    vault_id: str,
    memory_id: str,
    max_tool_calls_per_turn: int = 5,
    provider_type: str = None,  # Kept for backward compatibility
    debug: bool = False,
    mcp_client: Optional[MultiServerMCPClient] = None
) -> 'AgentMessageProcessor':
    """Convenience function to build an agent.
    
    This is the main entry point for creating an agent.
    
    Args:
        model_id: LLM model identifier
        vault_id: Vault UUID for memory organization
        memory_id: Memory UUID for storage
        max_tool_calls_per_turn: Maximum tool calls allowed per turn
        provider_type: Optional provider type (unused, for compatibility)
        debug: Enable debug logging
        mcp_client: Optional pre-configured MCP client (for dependency injection)
        
    Returns:
        AgentMessageProcessor ready to handle messages
    """
    # OpenAI-only simplification (from original)
    resolved_model = model_id if str(model_id).startswith("openai:") else f"openai:{model_id}"
    
    # Create builder and build the complete pipeline
    builder = AgentBuilder(
        model_id=resolved_model,
        vault_id=vault_id,
        memory_id=memory_id,
        max_tool_calls_per_turn=max_tool_calls_per_turn,
        debug=debug,
        mcp_client=mcp_client
    )
    
    return builder.build()