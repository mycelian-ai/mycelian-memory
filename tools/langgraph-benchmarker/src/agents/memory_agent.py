"""Mycelian Memory Agent - Observer pattern agent for memory persistence."""

from typing import Dict, Any, List
from langchain_mcp_adapters.client import MultiServerMCPClient
from langgraph.prebuilt import create_react_agent
from langchain_core.messages import BaseMessage
import structlog

from ..utils.model_factory import create_chat_model

logger = structlog.get_logger(__name__)


class MycelianMemoryAgent:
    """
    Observer pattern agent that receives conversation turns and builds high-quality
    memories following the exact protocol in @client/prompts/system/context_summary_rules.md.
    
    This agent maintains session awareness and conversation history, observing
    conversations like an independent note-taker.
    """

    def __init__(
        self,
        mcp_url: str = "http://localhost:11546/mcp",
        model: str = "gpt-4o-mini-2024-07-18",
        transport: str = "streamable_http",
        aws_region: str = "us-west-2"
    ):
        """
        Initialize the Mycelian Memory Agent.
        
        Args:
            mcp_url: URL of the Mycelian MCP server
            model: LLM model for the agent
            transport: MCP transport type
            aws_region: AWS region for Bedrock models
        """
        self.mcp_url = mcp_url
        self.model = model
        self.transport = transport
        self.aws_region = aws_region
        self._agent = None
        self._mcp_client = None

    async def initialize(self) -> None:
        """Initialize MCP client and create the LangGraph agent."""
        # Configure MCP client for memory operations
        self._mcp_client = MultiServerMCPClient({
            "mycelian": {
                "url": self.mcp_url,
                "transport": self.transport
            }
        })

        # Get available MCP tools
        memory_tools = await self._mcp_client.get_tools()

        # Create the agent with clean system prompt
        system_prompt = self._build_system_prompt()
        
        # Create model instance using factory (supports both OpenAI and Bedrock)
        model_instance = create_chat_model(
            model_id=self.model,
            provider="auto",  # Auto-detect based on model_id
            aws_region=self.aws_region
        )
        
        self._agent = create_react_agent(
            model=model_instance,
            tools=memory_tools,
            prompt=system_prompt
        )

    def _build_system_prompt(self) -> str:
        """
        Build the system prompt for the Mycelian Memory Agent.
        
        The prompt focuses on the observer pattern and protocol compliance
        with vault management for consistent benchmark runs.
        """
        return """You are the Mycelian Memory Agent. Your sole responsibility is to produce high-quality memory and context following the exact protocol in context_summary_rules.md.

You will receive conversation turns in the format:
"Observe this turn: <role>: <content>"

Along with Vault ID and Memory ID context for where to store the memories.

Use the get_default_prompts tool to fetch the context_summary_rules.md protocol, then process each turn according to those rules.

Always work within the specified vault_id and memory_id provided in the messages."""

    async def observe_turn(self, role: str, content: str, memory_id: str, vault_id: str = "longmemeval-benchmark") -> Dict[str, Any]:
        """
        Process a single conversation turn using the observer pattern.
        
        Args:
            role: The role of the speaker (user, assistant, speaker_1, etc.)
            content: The message content
            memory_id: ID of the memory to store the turn in
            vault_id: ID of the vault (defaults to longmemeval-benchmark)
            
        Returns:
            Response from the agent including any tool calls made
        """
        if not self._agent:
            raise RuntimeError("Agent not initialized. Call initialize() first.")
            
        logger.info(
            "Memory Agent observe_turn - STARTING",
            role=role,
            content_length=len(content),
            memory_id=memory_id,
            vault_id=vault_id,
            content_preview=content[:100]
        )

        # Format the observation message
        observation_message = f"Observe this turn: {role}: {content}"
        
        # Include vault and memory context for the agent
        full_message = f"Vault ID: {vault_id}\nMemory ID: {memory_id}\n\n{observation_message}"

        logger.info(
            "Memory Agent - INVOKING LangChain agent",
            full_message_length=len(full_message),
            message_preview=full_message[:200]
        )

        # Invoke the agent with the observation
        try:
            response = await self._agent.ainvoke({
                "messages": [{"role": "user", "content": full_message}]
            }, config={"recursion_limit": 50})  # Increase limit to see full pattern
            
            logger.info(
                "Memory Agent observe_turn - COMPLETED",
                response_type=type(response).__name__,
                message_count=len(response.get("messages", [])) if isinstance(response, dict) else "unknown"
            )
            
        except Exception as e:
            logger.error(
                "Memory Agent LangChain execution failed",
                error=str(e),
                error_type=type(e).__name__
            )
            
            # If recursion limit hit, let's see the tool call pattern
            if "recursion" in str(e).lower():
                logger.error("RECURSION ISSUE: LangChain agent stuck in tool-calling loop")
                
            raise

        return response

    async def initialize_session(
        self, 
        conversation_id: str, 
        memory_id: str,
        vault_id: str = "longmemeval-benchmark"
    ) -> Dict[str, Any]:
        """
        Initialize a memory session for a conversation.
        
        Args:
            conversation_id: Unique identifier for the conversation
            memory_id: ID of the memory to initialize
            vault_id: ID of the vault (defaults to longmemeval-benchmark)
            
        Returns:
            Response from the agent after initialization
        """
        if not self._agent:
            raise RuntimeError("Agent not initialized. Call initialize() first.")

        init_message = f"""Initialize memory session for conversation {conversation_id}.
Vault ID: {vault_id}
Memory ID: {memory_id}

Follow bootstrap sequence from context_summary_rules.md."""

        response = await self._agent.ainvoke({
            "messages": [{"role": "user", "content": init_message}]
        })

        return response

    async def start_session(
        self, 
        session_idx: int, 
        conversation_id: str, 
        memory_id: str,
        vault_id: str = "longmemeval-benchmark"
    ) -> Dict[str, Any]:
        """
        Signal the start of a new session within a conversation.
        
        Args:
            session_idx: Index of the session (1-based)
            conversation_id: Unique identifier for the conversation
            memory_id: ID of the memory
            vault_id: ID of the vault (defaults to longmemeval-benchmark)
            
        Returns:
            Response from the agent
        """
        if not self._agent:
            raise RuntimeError("Agent not initialized. Call initialize() first.")

        session_message = f"""Vault ID: {vault_id}
Memory ID: {memory_id}

Starting session {session_idx} of conversation {conversation_id}"""

        response = await self._agent.ainvoke({
            "messages": [{"role": "user", "content": session_message}]
        })

        return response

    async def finalize_session(
        self, 
        conversation_id: str, 
        memory_id: str,
        vault_id: str = "longmemeval-benchmark"
    ) -> Dict[str, Any]:
        """
        Finalize the memory session for a conversation.
        
        Args:
            conversation_id: Unique identifier for the conversation
            memory_id: ID of the memory
            vault_id: ID of the vault (defaults to longmemeval-benchmark)
            
        Returns:
            Response from the agent after finalization
        """
        if not self._agent:
            raise RuntimeError("Agent not initialized. Call initialize() first.")

        end_message = f"""Vault ID: {vault_id}
Memory ID: {memory_id}

End of conversation {conversation_id}. Finalize memory session."""

        response = await self._agent.ainvoke({
            "messages": [{"role": "user", "content": end_message}]
        })

        return response

    async def close(self) -> None:
        """Close the MCP client connection."""
        # Note: MultiServerMCPClient doesn't have a close() method
        logger.info("Memory Agent cleanup complete")


# Convenience function for creating and initializing the agent
async def create_mycelian_memory_agent(
    mcp_url: str = "http://localhost:11546/mcp",
    model: str = "gpt-4o-mini-2024-07-18",
    transport: str = "streamable_http"
) -> MycelianMemoryAgent:
    """
    Create and initialize a Mycelian Memory Agent.
    
    Args:
        mcp_url: URL of the Mycelian MCP server
        model: LLM model for the agent
        transport: MCP transport type
        
    Returns:
        Initialized MycelianMemoryAgent instance
    """
    agent = MycelianMemoryAgent(mcp_url=mcp_url, model=model, transport=transport)
    await agent.initialize()
    return agent