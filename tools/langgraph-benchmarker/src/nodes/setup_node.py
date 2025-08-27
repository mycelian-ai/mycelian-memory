"""Setup Memory Node - Initialize vault and memory for conversations."""

import json
import structlog
from typing import Optional
from langchain_mcp_adapters.client import MultiServerMCPClient

from ..types import LongMemEvalState

logger = structlog.get_logger(__name__)


async def setup_memory_node(
    state: LongMemEvalState,
    mcp_client: MultiServerMCPClient,
    vault_name: Optional[str] = None
) -> LongMemEvalState:
    """
    Initialize vault and memory for a conversation.
    
    This LangGraph node sets up the memory infrastructure needed for
    the conversation processing.
    
    Args:
        state: Current workflow state
        mcp_client: Initialized MCP client
        vault_name: Optional vault name (defaults to benchmark vault)
        
    Returns:
        Updated state with vault_id and memory_id set
    """
    conversation_id = state["conversation_id"]
    vault_name = vault_name or "longmemeval-benchmark"
    
    logger.info(
        "Setting up memory for conversation",
        conversation_id=conversation_id,
        vault_name=vault_name
    )

    try:
        # Get MCP tools
        tools = await mcp_client.get_tools()
        
        # Create or get the benchmark vault (shared across all conversations)
        vault_id = await _get_or_create_vault(
            tools=tools,
            vault_name=vault_name,
            description=f"Vault for LongMemEval benchmark runs"
        )
        
        # Create memory for this specific conversation within this benchmark run
        # Each conversation gets its own memory within the shared vault
        # Include run_id to track across benchmark runs
        run_id = state.get("run_id", "unknown")
        memory_title = f"run-{run_id}-conv-{conversation_id}"
        memory_id = await _create_memory(
            tools=tools,
            vault_id=vault_id,
            title=memory_title,
            memory_type="CHAT",
            description=f"Memory for LongMemEval run {run_id}, conversation {conversation_id}"
        )

        # Update state with IDs
        state["vault_id"] = vault_id
        state["memory_id"] = memory_id

        logger.info(
            "Memory setup complete",
            conversation_id=conversation_id,
            vault_id=vault_id,
            memory_id=memory_id
        )

    except Exception as e:
        logger.error(
            "Failed to setup memory",
            conversation_id=conversation_id,
            error=str(e)
        )
        raise

    return state


async def _get_or_create_vault(
    tools: list,
    vault_name: str,
    description: str
) -> str:
    """Get existing vault by name or create a new one using MCP tools."""
    
    # Find and invoke list_vaults tool
    list_vaults_tool = None
    create_vault_tool = None
    
    for tool in tools:
        if tool.name == "list_vaults":
            list_vaults_tool = tool
        elif tool.name == "create_vault":
            create_vault_tool = tool
    
    if not list_vaults_tool or not create_vault_tool:
        raise RuntimeError("Required MCP tools (list_vaults, create_vault) not found")
    
    # List existing vaults
    try:
        vaults_result = await list_vaults_tool.ainvoke({})
        # Parse JSON string response
        if isinstance(vaults_result, str):
            vaults = json.loads(vaults_result)
        else:
            vaults = vaults_result
        
        # Look for existing vault with matching title
        for vault in vaults:
            if vault.get("title") == vault_name:
                vault_id = vault.get("vaultId")
                logger.info("Found existing vault", name=vault_name, vault_id=vault_id)
                return vault_id
    except Exception as e:
        logger.warning("Error listing vaults", error=str(e))
        # Continue to create new vault
    
    # Create new vault if not found
    logger.info("Creating new vault", name=vault_name)
    try:
        result = await create_vault_tool.ainvoke({
            "title": vault_name,
            "description": description
        })
        
        # Parse JSON string response if needed
        if isinstance(result, str):
            result_dict = json.loads(result)
        else:
            result_dict = result
            
        vault_id = result_dict.get("vaultId")
        if not vault_id:
            raise ValueError("No vaultId returned from create_vault")
        
        logger.info("Created vault", title=vault_name, vault_id=vault_id)
        return vault_id
    except Exception as e:
        logger.error("Failed to create vault", title=vault_name, error=str(e))
        raise


async def _create_memory(
    tools: list,
    vault_id: str,
    title: str,
    memory_type: str,
    description: str
) -> str:
    """Create a memory within a vault using MCP tools."""
    
    # Find create_memory_in_vault tool
    create_memory_tool = None
    for tool in tools:
        if tool.name == "create_memory_in_vault":
            create_memory_tool = tool
            break
    
    if not create_memory_tool:
        raise RuntimeError("Required MCP tool (create_memory_in_vault) not found")
    
    try:
        result = await create_memory_tool.ainvoke({
            "vault_id": vault_id,
            "title": title,
            "memory_type": memory_type,
            "description": description
        })
        
        # Parse JSON string response if needed
        if isinstance(result, str):
            result_dict = json.loads(result)
        else:
            result_dict = result
            
        memory_id = result_dict.get("memoryId")
        if not memory_id:
            raise ValueError("No memoryId returned from create_memory_in_vault")
        
        logger.info(
            "Created memory",
            title=title,
            memory_id=memory_id,
            vault_id=vault_id
        )
        return memory_id
    except Exception as e:
        logger.error(
            "Failed to create memory",
            title=title,
            vault_id=vault_id,
            error=str(e)
        )
        raise


class SetupMemoryNode:
    """
    Class-based setup node for more complex initialization scenarios.
    """

    def __init__(self, mcp_client: MultiServerMCPClient, vault_name: Optional[str] = None):
        """
        Initialize the setup node.
        
        Args:
            mcp_client: Initialized MCP client
            vault_name: Optional default vault name
        """
        self.mcp_client = mcp_client
        self.vault_name = vault_name or "longmemeval-benchmark"

    async def setup_memory(self, state: LongMemEvalState) -> LongMemEvalState:
        """
        Set up memory infrastructure for the conversation.
        
        Args:
            state: Current workflow state
            
        Returns:
            Updated state with memory setup complete
        """
        return await setup_memory_node(
            state=state,
            mcp_client=self.mcp_client,
            vault_name=self.vault_name
        )

    async def cleanup_memory(
        self, 
        state: LongMemEvalState,
        delete_memory: bool = False
    ) -> None:
        """
        Optional cleanup after benchmark completion.
        
        Args:
            state: Workflow state containing memory IDs
            delete_memory: Whether to delete the created memory
        """
        if not delete_memory:
            return

        try:
            # Note: Mycelian might not have delete operations exposed via MCP
            # This is a placeholder for potential cleanup operations
            logger.info(
                "Cleanup requested for memory",
                conversation_id=state["conversation_id"],
                memory_id=state.get("memory_id")
            )
            
            # TODO: Implement memory deletion if/when available in MCP interface
            
        except Exception as e:
            logger.warning(
                "Error during memory cleanup",
                conversation_id=state["conversation_id"],
                error=str(e)
            )


def create_setup_node(
    mcp_client: MultiServerMCPClient,
    vault_name: Optional[str] = None
) -> callable:
    """
    Create a setup node function for LangGraph workflows.
    
    Args:
        mcp_client: Initialized MCP client
        vault_name: Optional vault name
        
    Returns:
        Async function for use as LangGraph node
    """
    setup_node_instance = SetupMemoryNode(mcp_client, vault_name)
    
    async def node_function(state: LongMemEvalState) -> LongMemEvalState:
        return await setup_node_instance.setup_memory(state)
    
    return node_function